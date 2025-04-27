package ghttp

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/j32u4ukh/gos/base"
	"github.com/j32u4ukh/gos/core"
	"github.com/j32u4ukh/gos/log"
	"github.com/pkg/errors"
)

type HttpAsker struct {
}

func NewHttpAsker() *HttpAsker {
	return &HttpAsker{}
}

func (a *HttpAsker) Get(uri string, params map[string]string) (*Response, error) {
	ask, err := newAsker(uri)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to build http asker")
	}
	response, err := ask.send(func(req *Request) error {
		err := req.SetRequest(METHOD_GET, uri, params)
		if err != nil {
			return errors.Wrap(err, "Faild to make a http connection")
		}
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to send get request")
	}
	return response, nil
}

type asker struct {
	*core.Asker
	*Server
	future *core.Future[*Response]
}

func newAsker(uri string) (*asker, error) {
	u := uri
	if !strings.Contains(uri, "://") {
		u = "http://" + uri
	}
	fmt.Printf("newAsker | uri: %s\n", u)
	// 解析 URL
	parsedUrl, err := url.Parse(u)
	if err != nil {
		return nil, errors.New("Invalid URL: " + err.Error())
	}
	fmt.Printf("newAsker | parsedUrl: %+v\n", parsedUrl)
	host := parsedUrl.Hostname()
	fmt.Printf("newAsker | host: %s, port: %s\n", host, parsedUrl.Port())
	port, err := strconv.ParseInt(parsedUrl.Port(), 10, 64)
	if err != nil {
		return nil, errors.New("Invalid port number: " + err.Error())
	}
	return &asker{
		Asker:  core.NewAsker(host, int32(port), 1),
		Server: NewServer(HTTPMODE_RESPONSE),
	}, nil
}

// 送出請求
func (a *asker) send(setRequest func(req *Request) error) (*Response, error) {
	a.initHandlerFunc()
	go a.Bind()
	defer a.Stop()
	fmt.Println("Bind asker")
	cid, err := a.Connect(-1)
	if err != nil {
		return nil, errors.Wrap(err, "Faild to make a http connection")
	}
	// baseConn := a.GetConn(cid, nil)
	c := a.GetContext(cid, a.GetConn(cid, nil))
	err = setRequest(c.req)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read http response")
	}
	a.future = core.NewFuture[*Response]()
	// 構成請求數據並寫出
	c.Write(c.req.Bytes())
	response, err := a.future.Get()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read http response")
	}
	return response, nil
}

// 由外部定義 HandlerFunc，定義如何處理工作
func (a *asker) initHandlerFunc() {
	a.SetHandlerFunc(func(baseConn *base.Conn) error {
		cid := baseConn.GetId()
		context := a.GetContext(cid, baseConn)
		defer func(ctx *HttpContext) {
			if err := recover(); err != nil {
				log.Error("Recover err: %+v", err)
				// a.errorHandler(context, StatusInternalServerError, "Internal Server Error")
				a.PutContext(context)
			}
		}(context)
		// 釋放上一輪的 response 狀態，以便接收新的請求資料
		if context.state == READ_RESPONSE_FINISH {
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
		a.future.Set(context.res, nil)
		return nil
	})
}
