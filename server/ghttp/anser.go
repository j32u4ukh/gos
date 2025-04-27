package ghttp

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/j32u4ukh/gos/base"
	"github.com/j32u4ukh/gos/core"
	"github.com/j32u4ukh/gos/log"
	"github.com/pkg/errors"
)

type HttpAnser struct {
	*core.Anser
	*Server
	router *Router

	// key1: Method(Get/Post); key2: node number of EndPoint; value: []*EndPoint
	endpoints *EndPointHandlers

	// ==================================================
	// Context
	// 個數與 Anser 的 nConnect 相同，因此可利用 Conn 中的 id 作為索引值，來存取,
	// 由於 Context 是使用 Conn 的 id 作為索引值，因此可以不用從第一個開始使用，結束使用後也不需要對順序進行調整
	// ==================================================
	contextPool sync.Pool
	contextMap  map[int32]*HttpContext
}

func NewHttpAnser(port int32, nConnect int32) *HttpAnser {
	a := &HttpAnser{
		Anser: core.NewAnser(port, nConnect),
		router: &Router{
			// 最開始的 '/' 會形成空字串的 node
			nodes:    []*node{newNode("")},
			Handlers: HandlerChain{},
		},
		endpoints: &EndPointHandlers{},
		Server:    NewServer(HTTPMODE_REQUEST),
	}
	a.router.endpoints = a.endpoints
	return a
}

func (a *HttpAnser) Init() error {
	err := a.Anser.Init()
	if err != nil {
		return errors.Wrap(err, "Failed to init HttpAnser")
	}
	return nil
}

// 監聽連線並註冊
func (a *HttpAnser) Listen() {
	a.InitHandlerFunc()
	a.Anser.Listen()
}

func (a *HttpAnser) GetRouter() *Router {
	return a.router
}

// 由外部定義 HandlerFunc，定義如何處理工作
func (a *HttpAnser) InitHandlerFunc() {
	a.SetHandlerFunc(func(baseConn *base.Conn) error {
		cid := baseConn.GetId()
		context := a.GetContext(cid, baseConn)
		defer func(ctx *HttpContext) {
			if err := recover(); err != nil {
				log.Error("Recover err: %+v", err)
				a.errorHandler(context, StatusInternalServerError, "Internal Server Error")
			}
		}(context)
		// 釋放上一輪的 request 狀態，以便接收新的請求資料
		if context.state == WRITE_RESPONSE || context.state == READ_RESPONSE_FINISH {
			context.Release()
		}
		// HttpContext 讀取數據
		err := context.Read()
		if err != nil {
			return errors.Wrap(err, "Failed to read data from socket")
		}
		if context.state != WRITE_RESPONSE && context.state != READ_RESPONSE_FINISH {
			return nil
		}
		log.Debug("request:\n%s\n", context.req.String())
		var splits []string
		var nSplit int32
		if context.req.query == "" || context.req.query == "/" {
			nSplit = 1
			splits = []string{""}
		} else {
			context.req.query = strings.TrimSuffix(context.req.query, "/")
			splits = strings.Split(context.req.query, "/")
			nSplit = int32(len(splits))
		}
		var ok, unmatched bool = true, true
		var handlers HandlerChain
		var function HandlerFunc
		var key string
		var value any
		for _, endpoint := range *a.endpoints {
			if endpoint.nNode != nSplit {
				continue
			}
			if _, ok = endpoint.Handlers[context.req.method]; !ok {
				continue
			}
			if !endpoint.Macth(splits) {
				continue
			}
			log.Debug("endpoint path: %s", endpoint.path)
			unmatched = false
			if context.req.method == METHOD_OPTIONS {
				a.optionsRequestHandler(context, endpoint.options)
				break
			}
			for key, value = range endpoint.params {
				if _, ok = context.req.params[key]; !ok {
					context.req.params[key] = fmt.Sprintf("%v", value)
				}
				if _, ok = context.req.values[key]; !ok {
					context.req.values[key] = value
				}
			}
			handlers = endpoint.Handlers[context.req.method]
			for _, function = range handlers {
				function(context)
			}
			a.Send(context)
			break
		}
		if unmatched {
			a.errorHandler(context, StatusBadRequest, "Unmatched endpoint.")
		}
		return nil
	})
}

func (a *HttpAnser) optionsRequestHandler(context *HttpContext, options []string) {
	context.SetHeader("Allow", strings.Join(options, ", "))
	context.SetHeader("Connection", "close")
	context.Status(StatusOK)
	context.res.bodyLength = 0
	a.Send(context)
}

func (a *HttpAnser) errorHandler(c *HttpContext, code int32, msg string) {
	log.Error("method: %s, query: %s", c.req.Method(), c.req.params)
	c.Json(code, H{
		"error": msg,
	})
	a.Send(c)
}

func (a *HttpAnser) Send(c *HttpContext) {
	connections, ok := c.req.GetHeader(HEADER_CONNECTION)
	var connection string = "close"
	if ok && len(connections) > 0 {
		connection = connections[0]
	}
	// TODO: 若是 Connection: keep-alive，應檢查距離前一次請求的間隔時間，若在 timeout 時間內沒有收到請求，就關閉連線
	// 根據 RFC 7230, section 6.1:
	// A server that receives a Connection: close request header field must initiate a close of the connection after completing the response.
	// The server SHOULD also send Connection: close in the response to indicate that it will close the connection.
	// 接收到 Connection: close 請求標頭欄位的伺服器必須在完成回應後啟動連線關閉。
	// 伺服器也應該在回應中發送 Connection: close 來表示它將關閉連線。
	c.res.setDefaultHeader(HEADER_CONNECTION, connection)
	// 構成返回數據並寫出
	c.Write(c.res.Bytes())
	// 若標頭為 Connection: close, 則數秒後關閉連線
	c.SetShouldCloseAfterWrite(3*time.Second, func() bool {
		return connection == "close"
	})
}
