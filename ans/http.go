package ans

import (
	"fmt"
	"net"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/j32u4ukh/gos/base"
	"github.com/j32u4ukh/gos/base/ghttp"
	"github.com/j32u4ukh/gos/utils"

	"github.com/pkg/errors"
)

type HandlerFunc func(c *ghttp.Context)
type HandlerChain []HandlerFunc

// ====================================================================================================
// HttpAnser
// ====================================================================================================

type HttpAnser struct {
	*Anser
	*Router

	// key1: Method(Get/Post); key2: path; value: []HandlerFunc
	Handlers map[string]map[string]HandlerChain

	// ==================================================
	// Context
	// 個數與 Anser 的 nConnect 相同，因此可利用 Conn 中的 id 作為索引值，來存取,
	// 由於 Context 是使用 Conn 的 id 作為索引值，因此可以不用從第一個開始使用，結束使用後也不需要對順序進行調整
	// ==================================================
	httpConns []*ghttp.Context
	httpConn  *ghttp.Context

	contextPool sync.Pool
	context     *ghttp.Context

	// Temp variables
	lineString string
}

func NewHttpAnser(laddr *net.TCPAddr, nConnect int32, nWork int32) (IAnswer, error) {
	var err error
	a := &HttpAnser{
		Handlers: map[string]map[string]HandlerChain{
			ghttp.MethodGet:  {},
			ghttp.MethodPost: {},
		},
		httpConns:   make([]*ghttp.Context, nConnect),
		httpConn:    nil,
		contextPool: sync.Pool{New: func() any { return ghttp.NewContext(-1) }},
		context:     nil,
	}

	// ===== Anser =====
	a.Anser, err = newAnser(laddr, nConnect, nWork)
	a.Anser.ReadTimeout = 5000 * time.Millisecond

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to new HttpAnser.")
	}

	// ===== Router =====
	a.Router = &Router{
		HttpAnser: a,
		path:      "/",
		Handlers:  HandlerChain{},
	}

	// ===== Context =====
	var i int32
	for i = 0; i < nConnect; i++ {
		a.httpConns[i] = ghttp.NewContext(i)
	}

	//////////////////////////////////////////////////
	// 自定義函式
	//////////////////////////////////////////////////
	// 設置數據讀取函式
	a.readFunc = a.read
	a.writeFunc = a.write
	a.shouldCloseFunc = a.shouldClose
	return a, nil
}

// 監聽連線並註冊
func (a *HttpAnser) Listen() {
	a.SetWorkHandler()
	a.Anser.Listen()
}

func (a *HttpAnser) read() bool {
	// 根據 Conn 的 Id，存取對應的 httpConn
	a.httpConn = a.httpConns[a.currConn.GetId()]

	// 讀取 第一行
	if a.httpConn.State == 0 {
		if a.currConn.CheckReadable(a.httpConn.HasLineData) {
			a.currConn.Read(&a.readBuffer, a.httpConn.ReadLength)

			// 拆分第一行數據
			a.lineString = strings.TrimRight(string(a.readBuffer[:a.httpConn.ReadLength]), "\r\n")
			// fmt.Printf("(a *HttpAnser) Read | firstLine: %s\n", a.lineString)
			a.logger.Info("firstLine: %s", a.lineString)

			if a.httpConn.ParseFirstReqLine(a.lineString) {
				if a.httpConn.Method == ghttp.MethodGet {
					// 解析第一行數據中的請求路徑
					a.httpConn.ParseQuery()
				}
				a.httpConn.State = 1
				// fmt.Printf("(a *HttpAnser) Read | State: 0 -> 1\n")
				a.logger.Debug("State: 0 -> 1")
			}
		}
	}

	// 讀取 Header 數據
	if a.httpConn.State == 1 {
		var key, value string
		var ok bool

		for a.httpConn.State == 1 && a.currConn.CheckReadable(a.httpConn.HasLineData) {
			// 讀取一行數據
			a.currConn.Read(&a.readBuffer, a.httpConn.ReadLength)

			// mustHaveFieldNameColon ensures that, per RFC 7230, the field-name is on a single line,
			// so the first line must contain a colon.
			// 將讀到的數據從冒號拆分成 key, value
			// k, v, ok := bytes.Cut(a.readBuffer[:a.currContext.ReadLength], COLON)
			a.lineString = strings.TrimRight(string(a.readBuffer[:a.httpConn.ReadLength]), "\r\n")
			key, value, ok = strings.Cut(a.lineString, ghttp.COLON)

			if ok {
				// 持續讀取 Header
				if _, ok := a.httpConn.Header[key]; !ok {
					a.httpConn.Header[key] = []string{}
				}

				value = strings.TrimLeft(value, " \t")
				// value = strings.TrimRight(value, "\r\n")
				a.httpConn.Header[key] = append(a.httpConn.Header[key], value)
				// fmt.Printf("(a *HttpAnser) Read | Header, key: %s, value: %s\n", key, value)
				a.logger.Debug("Header, key: %s, value: %s", key, value)

			} else {
				// 當前這行數據不包含":"，結束 Header 的讀取

				// Header 中包含 Content-Length，狀態值設為 2，等待讀取後續數據
				if contentLength, ok := a.httpConn.Header["Content-Length"]; ok {
					length, err := strconv.Atoi(contentLength[0])
					// fmt.Printf("(a *HttpAnser) Read | Content-Length: %d\n", length)
					a.logger.Debug("Content-Length: %d", length)

					if err != nil {
						// fmt.Printf("(a *HttpAnser) Read | Content-Length err: %+v\n", err)
						a.logger.Error("Content-Length err: %+v", err)
						return false
					}

					a.httpConn.ReadLength = int32(length)
					a.httpConn.State = 2
					// fmt.Printf("(a *HttpAnser) Read | State: 1 -> 2\n")
					a.logger.Debug("State: 1 -> 2")

				} else {
					// 考慮分包問題，收到完整一包數據傳完才傳到應用層
					a.currWork.Index = a.currConn.GetId()
					a.currWork.RequestTime = time.Now().UTC()
					a.currWork.State = 1
					a.currWork.Body.ResetIndex()

					// 指向下一個工作結構
					a.currWork = a.currWork.Next

					// 等待數據寫出
					a.httpConn.State = 3
					// fmt.Printf("(a *HttpAnser) Read | State: 1 -> 3\n")
					a.logger.Debug("State: 1 -> 3")
					return true
				}
			}
		}
	}

	// 讀取 Body 數據
	if a.httpConn.State == 2 {
		if a.currConn.CheckReadable(a.httpConn.HasEnoughData) {
			// ==========
			// 讀取 data
			// ==========
			// 將傳入的數據，加入工作緩存中
			a.currConn.Read(&a.readBuffer, a.httpConn.ReadLength)
			// fmt.Printf("(a *HttpAnser) Read | %s\n", string(a.readBuffer[:a.httpConn.ReadLength]))
			a.logger.Debug("Body 數據: %s", string(a.readBuffer[:a.httpConn.ReadLength]))

			// 考慮分包問題，收到完整一包數據傳完才傳到應用層
			a.currWork.Index = a.currConn.GetId()
			a.currWork.RequestTime = time.Now().UTC()
			a.currWork.State = 1
			a.currWork.Body.AddRawData(a.readBuffer[:a.httpConn.ReadLength])
			a.currWork.Body.ResetIndex()

			// 指向下一個工作結構
			a.currWork = a.currWork.Next

			// 等待數據寫出
			a.httpConn.State = 3
			// fmt.Printf("(a *HttpAnser) Read | State: 2 -> 3\n")
			a.logger.Debug("State: 2 -> 3")
			return false
		}
	}

	return true
}

func (a *HttpAnser) write(cid int32, data *[]byte, length int32) error {
	// 取得對應的連線結構
	a.currConn = a.getConn(cid)

	if a.currConn == nil {
		return errors.New(fmt.Sprintf("There is no cid equals to %d.", cid))
	}

	a.currConn.SetWriteBuffer(data, length)

	// 完成數據複製到寫出緩存
	a.httpConn.State = 4
	return nil
}

// 由外部定義 workHandler，定義如何處理工作
func (a *HttpAnser) SetWorkHandler() {
	a.workHandler = func(w *base.Work) {
		a.httpConn = a.httpConns[w.Index]
		a.httpConn.Cid = w.Index
		a.httpConn.Wid = w.GetId()
		// fmt.Printf("(a *HttpAnser) SetWorkHandler | Cid: %d, Wid: %d\n", a.httpConn.Cid, a.httpConn.Wid)
		a.logger.Debug("Cid: %d, Wid: %d", a.httpConn.Cid, a.httpConn.Wid)

		if handler, ok := a.Handlers[a.httpConn.Method]; ok {
			if functions, ok := handler[a.httpConn.Query]; ok {
				for _, handlerFunc := range functions {
					handlerFunc(a.httpConn)
				}
			} else {
				a.errorRequestHandler(w, a.httpConn, "Unregistered http query.")
			}
		} else {
			a.errorRequestHandler(w, a.httpConn, "Unregistered http method.")
		}
	}
}

func (a *HttpAnser) errorRequestHandler(w *base.Work, c *ghttp.Context, msg string) {
	// fmt.Printf("(s *Server) errorRequestHandler | method: %s, query: %s\n", c.Method, c.Query)
	a.logger.Debug("method: %s, query: %s", c.Method, c.Query)

	c.Json(400, ghttp.H{
		"code": 400,
		"msg":  msg,
	})
	c.SetHeader("Connection", "close")

	// 將 Response 回傳數據轉換成 Work 傳遞的格式
	bs := c.ToResponseData()
	// fmt.Printf("Response: %s\n", string(bs))
	a.logger.Debug("Response: %s", string(bs))

	w.Body.AddRawData(bs)
	w.Send()
}

// 當前連線是否應斷線
func (a *HttpAnser) shouldClose(err error) bool {
	if a.Anser.shouldClose(err) {
		return true
	}
	a.httpConn = a.httpConns[a.currConn.GetId()]
	if a.httpConn.State == 4 && a.currConn.WritableLength == 0 {
		// fmt.Printf("(a *HttpAnser) shouldClose | Conn(%d) 完成數據寫出，準備關閉連線\n", a.currConn.GetId())
		a.logger.Info("Conn(%d) 完成數據寫出，準備關閉連線", a.currConn.GetId())
		a.httpConn.State = 0
		return true
	}
	return false
}

func (a *HttpAnser) GetContext(cid int32) *ghttp.Context {
	if cid == -1 {
		a.context = a.contextPool.Get().(*ghttp.Context)
		return a.context
	} else {
		return a.httpConns[cid]
	}
}

func (a *HttpAnser) Send(c *ghttp.Context) {
	c.SetHeader("Connection", "close")

	// 將 Response 回傳數據轉換成 Work 傳遞的格式
	bs := c.ToResponseData()

	// fmt.Printf("Response: %s\n", string(bs))
	// fmt.Printf("Raw Response: %+v\n", utils.SliceToString(bs))
	a.logger.Debug("Response: %s", string(bs))
	a.logger.Debug("Raw Response: %s", utils.SliceToString(bs))

	w := a.getWork(c.Wid)
	// fmt.Printf("Wid: %d, w: %+v\n", c.Wid, w)
	a.logger.Debug("Wid: %d, w: %+v", c.Wid, w)

	w.Index = c.Cid
	// fmt.Printf("c.Cid: %d, w.Index: %d\n", c.Cid, w.Index)
	a.logger.Debug("c.Cid: %d, w.Index: %d", c.Cid, w.Index)

	w.Body.AddRawData(bs)
	w.Send()
	// fmt.Printf("Wid: %d, w: %+v\n", c.Wid, w)
	a.logger.Debug("Wid: %d, w: %+v", c.Wid, w)

	// 若 Context 是從 contextPool 中取得，id 會是 -1，因此需要回收
	if c.GetId() == -1 {
		a.contextPool.Put(c)
	}
}

func (a *HttpAnser) Finish(c *ghttp.Context) {
	// fmt.Printf("(a *HttpAnser) Finish | Context %d, c.Wid: %d\n", c.GetId(), c.Wid)
	a.logger.Info("Context %d, c.Wid: %d", c.GetId(), c.Wid)
	w := a.getWork(c.Wid)
	w.Finish()
}

// ====================================================================================================
// Router
// ====================================================================================================
type Router struct {
	*HttpAnser
	path     string
	Handlers HandlerChain
}

func (r *Router) NewRouter(relativePath string, handlers ...HandlerFunc) *Router {
	nr := &Router{
		HttpAnser: r.HttpAnser,
		path:      r.combinePath(relativePath),
		Handlers:  r.combineHandlers(handlers),
	}
	return nr
}

func (r *Router) GET(path string, handlers ...HandlerFunc) {
	r.handle(ghttp.MethodGet, path, handlers...)
}

func (r *Router) POST(path string, handlers ...HandlerFunc) {
	r.handle(ghttp.MethodPost, path, handlers...)
}

func (r *Router) handle(method string, path string, handlers ...HandlerFunc) {
	if routers, ok := r.HttpAnser.Handlers[method]; ok {
		path = r.combinePath(path)

		if _, ok := routers[path]; ok {
			// fmt.Printf("(r *Router) handle | Duplicate handler, method: %v, path: %s\n", method, path)
			r.logger.Warn("Duplicate handler, method: %v, path: %s", method, path)
			return
		}

		// 添加路徑對應的處理函式
		routers[path] = r.combineHandlers(handlers)
	}
}

func (r *Router) combinePath(relativePath string) string {
	return joinPaths(r.path, relativePath)
}

func (r *Router) combineHandlers(handlers HandlerChain) HandlerChain {
	size := len(r.Handlers) + len(handlers)
	mergedHandlers := make(HandlerChain, size)
	copy(mergedHandlers, r.Handlers)
	copy(mergedHandlers[len(r.Handlers):], handlers)
	return mergedHandlers
}

func joinPaths(absolutePath, relativePath string) string {
	if relativePath == "" {
		return absolutePath
	}

	finalPath := path.Join(absolutePath, relativePath)

	if lastChar(relativePath) == '/' && lastChar(finalPath) != '/' {
		return finalPath + "/"
	}

	return finalPath
}

func lastChar(str string) uint8 {
	if str == "" {
		panic("The length of the string can't be 0")
	}
	return str[len(str)-1]
}
