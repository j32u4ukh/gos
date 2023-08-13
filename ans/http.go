package ans

import (
	"fmt"
	"net"
	"path"
	"sort"
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

	// key1: Method(Get/Post); key2: node number of EndPoint; value: []*EndPoint
	// Handlers         map[string]map[int32][]*EndPoint
	EndPointHandlers []*EndPoint

	// ==================================================
	// Context
	// 個數與 Anser 的 nConnect 相同，因此可利用 Conn 中的 id 作為索引值，來存取,
	// 由於 Context 是使用 Conn 的 id 作為索引值，因此可以不用從第一個開始使用，結束使用後也不需要對順序進行調整
	// ==================================================
	contextPool sync.Pool
	contexts    []*ghttp.Context
	context     *ghttp.Context

	// Temp variables
	lineString string
}

func NewHttpAnser(laddr *net.TCPAddr, nConnect int32, nWork int32) (IAnswer, error) {
	var err error
	a := &HttpAnser{
		// Handlers: map[string]map[int32][]*EndPoint{
		// 	ghttp.MethodHead:   {},
		// 	ghttp.MethodGet:    {},
		// 	ghttp.MethodPost:   {},
		// 	ghttp.MethodPut:    {},
		// 	ghttp.MethodPatch:  {},
		// 	ghttp.MethodDelete: {},
		// },
		EndPointHandlers: []*EndPoint{},
		contexts:         make([]*ghttp.Context, nConnect),
		context:          nil,
		contextPool:      sync.Pool{New: func() any { return ghttp.NewContext(-1) }},
	}

	// ===== Anser =====
	a.Anser, err = newAnser(laddr, nConnect, nWork)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to new HttpAnser.")
	}
	a.Anser.ReadTimeout = 5000 * time.Millisecond

	// ===== Router =====
	a.Router = &Router{
		HttpAnser: a,
		// 最開始的 '/' 會形成空字串的 node
		nodes:    []*node{newNode("")},
		Handlers: HandlerChain{},
	}

	// ===== Context =====
	var i int32
	for i = 0; i < nConnect; i++ {
		a.contexts[i] = ghttp.NewContext(i)
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
	a.context = a.contexts[a.currConn.GetId()]

	// 讀取 第一行(ex: GET /foo/bar HTTP/1.1)
	if a.context.State == ghttp.READ_FIRST_LINE {
		if a.currConn.CheckReadable(a.context.HasLineData) {
			a.currConn.Read(&a.readBuffer, a.context.Request.ReadLength)

			// 拆分第一行數據
			a.lineString = strings.TrimRight(string(a.readBuffer[:a.context.Request.ReadLength]), "\r\n")
			utils.Info("firstLine: %s", a.lineString)

			if a.context.ParseFirstReqLine(a.lineString) {
				if a.context.Method == ghttp.MethodGet {
					// 解析第一行數據中的請求路徑
					a.context.ParseQuery()
				}
				a.context.State = ghttp.READ_HEADER
				utils.Debug("State: READ_FIRST_LINE -> READ_HEADER")
			}
		}
	}

	// 讀取 Header 數據
	if a.context.State == ghttp.READ_HEADER {
		var key, value string
		var ok bool

		for a.context.State == ghttp.READ_HEADER && a.currConn.CheckReadable(a.context.HasLineData) {
			// 讀取一行數據
			a.currConn.Read(&a.readBuffer, a.context.Request.ReadLength)

			// mustHaveFieldNameColon ensures that, per RFC 7230, the field-name is on a single line,
			// so the first line must contain a colon.
			// 將讀到的數據從冒號拆分成 key, value
			// k, v, ok := bytes.Cut(a.readBuffer[:a.currContext.ReadLength], COLON)
			a.lineString = strings.TrimRight(string(a.readBuffer[:a.context.Request.ReadLength]), "\r\n")
			key, value, ok = strings.Cut(a.lineString, ghttp.COLON)

			if ok {
				// 持續讀取 Header
				if _, ok := a.context.Request.Header[key]; !ok {
					a.context.Request.Header[key] = []string{}
				}

				value = strings.TrimLeft(value, " \t")
				// value = strings.TrimRight(value, "\r\n")
				a.context.Request.Header[key] = append(a.context.Request.Header[key], value)
				// fmt.Printf("(a *HttpAnser) Read | Header, key: %s, value: %s\n", key, value)
				utils.Debug("Header, key: %s, value: %s", key, value)

			} else {
				// 當前這行數據不包含":"，結束 Header 的讀取

				// Header 中包含 Content-Length，狀態值設為 2，等待讀取後續數據
				if contentLength, ok := a.context.Request.Header["Content-Length"]; ok {
					length, err := strconv.Atoi(contentLength[0])
					// fmt.Printf("(a *HttpAnser) Read | Content-Length: %d\n", length)
					utils.Debug("Content-Length: %d", length)

					if err != nil {
						// fmt.Printf("(a *HttpAnser) Read | Content-Length err: %+v\n", err)
						utils.Error("Content-Length err: %+v", err)
						return false
					}

					a.context.Request.ReadLength = int32(length)
					a.context.State = ghttp.READ_BODY
					utils.Debug("State: READ_HEADER -> READ_BODY")

				} else {
					// 考慮分包問題，收到完整一包數據傳完才傳到應用層
					a.currWork.Index = a.currConn.GetId()
					a.currWork.RequestTime = time.Now().UTC()
					a.currWork.State = base.WORK_NEED_PROCESS
					a.currWork.Body.ResetIndex()

					// 指向下一個工作結構
					a.currWork = a.currWork.Next

					// 等待數據寫出
					a.context.State = ghttp.WRITE_RESPONSE
					utils.Debug("State: READ_HEADER -> WRITE_RESPONSE")
					return true
				}
			}
		}
	}

	// 讀取 Body 數據
	if a.context.State == ghttp.READ_BODY {
		if a.currConn.CheckReadable(a.context.HasEnoughData) {
			// 將傳入的數據，加入工作緩存中
			a.currConn.Read(&a.readBuffer, a.context.Request.ReadLength)
			utils.Debug("Body 數據: %s", string(a.readBuffer[:a.context.Request.ReadLength]))

			// 考慮分包問題，收到完整一包數據傳完才傳到應用層
			a.currWork.Index = a.currConn.GetId()
			a.currWork.RequestTime = time.Now().UTC()
			a.currWork.State = base.WORK_NEED_PROCESS
			a.context.Request.SetBody(a.readBuffer, a.context.Request.ReadLength)

			// 指向下一個工作結構
			a.currWork = a.currWork.Next

			// 等待數據寫出
			a.context.State = ghttp.WRITE_RESPONSE
			utils.Debug("State: READ_BODY -> WRITE_RESPONSE")
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
	a.context.State = ghttp.FINISH_RESPONSE
	return nil
}

// 由外部定義 workHandler，定義如何處理工作
func (a *HttpAnser) SetWorkHandler() {
	// 在此將通用的 Work 轉換成 Http 專用的 Context
	a.workHandler = func(w *base.Work) {
		defer func() {
			if err := recover(); err != nil {
				utils.Error("Recover err: %+v", err)
				a.serverErrorHandler(w, a.context, "Internal Server Error")
			}
		}()
		a.context = a.contexts[w.Index]
		a.context.Cid = w.Index
		a.context.Wid = w.GetId()
		utils.Debug("Cid: %d, Wid: %d", a.context.Cid, a.context.Wid)
		var key string
		var value any
		var unmatched bool = true
		var nSplit int32
		var splits []string

		if a.context.Query == "" || a.context.Query == "/" {
			nSplit = 1
			splits = []string{""}
		} else {
			a.context.Query = strings.TrimSuffix(a.context.Query, "/")
			splits = strings.Split(a.context.Query, "/")
			nSplit = int32(len(splits))
		}

		for _, endpoint := range a.EndPointHandlers {
			if endpoint.nNode == nSplit {
				if handlers, ok := endpoint.Handlers[a.context.Method]; ok {
					if endpoint.Macth(splits) {
						utils.Debug("endpoint path: %s", endpoint.path)
						unmatched = false
						if a.context.Method == ghttp.MethodOptions {
							a.optionsRequestHandler(w, a.context, endpoint.options)
						} else {
							for key, value = range endpoint.params {
								if _, ok = a.context.Params[key]; !ok {
									a.context.Params[key] = fmt.Sprintf("%v", value)
								}
								if _, ok = a.context.Values[key]; !ok {
									a.context.Values[key] = value
								}
							}
							for _, function := range handlers {
								function(a.context)
							}
							// TODO: Unit test 檢查 Response
							// 檢查 Response 是否需要寫出
							if a.context.Code != -1 {
								a.Send(a.context)
							} else {
								a.Finish(a.context)
							}
						}
						break
					}
				}
			}
		}
		if unmatched {
			a.errorRequestHandler(w, a.context, "Unmatched endpoint.")
		}
	}
}

func (a *HttpAnser) optionsRequestHandler(w *base.Work, c *ghttp.Context, options []string) {
	a.context.Response.SetHeader("Allow", strings.Join(options, ", "))
	a.context.Response.SetHeader("Connection", "close")
	a.context.Status(ghttp.StatusOK)
	a.context.Response.BodyLength = 0
	a.context.Response.SetContentLength()
	// 將 Response 回傳數據轉換成 Work 傳遞的格式
	bs := a.context.ToResponseData()
	w.Body.Clear()
	w.Body.AddRawData(bs)
	w.Send()
}

func (a *HttpAnser) errorRequestHandler(w *base.Work, c *ghttp.Context, msg string) {
	utils.Debug("method: %s, query: %s", c.Method, c.Query)

	c.Json(400, ghttp.H{
		"code": 400,
		"msg":  msg,
	})
	c.Response.SetHeader("Connection", "close")

	// 將 Response 回傳數據轉換成 Work 傳遞的格式
	bs := c.ToResponseData()
	// fmt.Printf("Response: %s\n", string(bs))
	utils.Debug("Response: %s", string(bs))

	w.Body.AddRawData(bs)
	w.Send()
}

func (a *HttpAnser) serverErrorHandler(w *base.Work, c *ghttp.Context, msg string) {
	utils.Debug("method: %s, query: %s", c.Method, c.Query)

	c.Json(ghttp.StatusInternalServerError, ghttp.H{
		"code": ghttp.StatusInternalServerError,
		"msg":  msg,
	})
	c.Response.SetHeader("Connection", "close")

	// 將 Response 回傳數據轉換成 Work 傳遞的格式
	bs := c.ToResponseData()
	utils.Debug("Response: %s", string(bs))
	w.Body.Clear()
	w.Body.AddRawData(bs)
	w.Send()
}

// 當前連線是否應斷線
func (a *HttpAnser) shouldClose(err error) bool {
	a.context = a.contexts[a.currConn.GetId()]
	if a.Anser.shouldClose(err) {
		a.context.Release()
		return true
	}
	if a.context.State == ghttp.FINISH_RESPONSE && a.currConn.WritableLength == 0 {
		utils.Info("Conn(%d) 完成數據寫出，準備關閉連線", a.currConn.GetId())
		a.context.Release()
		return true
	}
	return false
}

func (a *HttpAnser) GetContext(cid int32) *ghttp.Context {
	if cid == -1 {
		a.context = a.contextPool.Get().(*ghttp.Context)
	} else {
		a.context = a.contexts[cid]
	}
	return a.context
}

func (a *HttpAnser) Send(c *ghttp.Context) {
	c.Response.SetHeader("Connection", "close")

	// 將 Response 回傳數據轉換成 Work 傳遞的格式
	bs := c.ToResponseData()
	utils.Debug("Response: %s", string(bs))

	w := a.getWork(c.Wid)
	utils.Debug("Wid: %d, w: %+v", c.Wid, w)

	w.Index = c.Cid
	utils.Debug("c.Cid: %d, w.Index: %d", c.Cid, w.Index)

	w.Body.AddRawData(bs)
	w.Send()
	utils.Debug("Wid: %d, w: %+v", c.Wid, w)

	// 若 Context 是從 contextPool 中取得，id 會是 -1，因此需要回收
	if c.GetId() == -1 {
		a.contextPool.Put(c)
	}
}

func (a *HttpAnser) Finish(c *ghttp.Context) {
	// fmt.Printf("(a *HttpAnser) Finish | Context %d, c.Wid: %d\n", c.GetId(), c.Wid)
	utils.Info("Context %d, c.Wid: %d", c.GetId(), c.Wid)
	w := a.getWork(c.Wid)
	w.Finish()
}

// ====================================================================================================
// Router
// ====================================================================================================
type Router struct {
	*HttpAnser
	path     string
	nodes    []*node
	Handlers HandlerChain
}

// 每個 EndPoint 對應一個 Router，但每個 Router 不一定對應著一個 EndPoint
func (r *Router) NewRouter(relativePath string, handlers ...HandlerFunc) *Router {
	nr := &Router{
		HttpAnser: r.HttpAnser,
		path:      r.combinePath(relativePath),
		nodes:     r.combineNodes(relativePath),
		Handlers:  r.combineHandlers(handlers),
	}
	return nr
}

func (r *Router) HEAD(path string, handlers ...HandlerFunc) {
	r.handle(ghttp.MethodHead, path, handlers...)
}

func (r *Router) GET(path string, handlers ...HandlerFunc) {
	r.handle(ghttp.MethodGet, path, handlers...)
}

func (r *Router) POST(path string, handlers ...HandlerFunc) {
	r.handle(ghttp.MethodPost, path, handlers...)
}

func (r *Router) PUT(path string, handlers ...HandlerFunc) {
	r.handle(ghttp.MethodPut, path, handlers...)
}

func (r *Router) PATCH(path string, handlers ...HandlerFunc) {
	r.handle(ghttp.MethodPatch, path, handlers...)
}

func (r *Router) DELETE(path string, handlers ...HandlerFunc) {
	r.handle(ghttp.MethodDelete, path, handlers...)
}

func (r *Router) handle(method string, path string, handlers ...HandlerFunc) {
	var endpoint *EndPoint
	fullPath := r.combinePath(path)
	isExists := false

	for _, endpoint = range r.HttpAnser.EndPointHandlers {
		if endpoint.path == fullPath {
			isExists = true
			break
		}
	}

	if !isExists {
		endpoint = NewEndPoint()
		endpoint.path = fullPath
		nodes := r.combineNodes(path)
		endpoint.InitNodes(nodes)
		r.HttpAnser.EndPointHandlers = append(r.HttpAnser.EndPointHandlers, endpoint)
	}

	if _, ok := endpoint.Handlers[method]; !ok {
		endpoint.options = append(endpoint.options, method)
	}

	endpoint.Handlers[method] = r.combineHandlers(handlers)
	sort.SliceStable(r.HttpAnser.EndPointHandlers, func(i, j int) bool {
		// True 的話，會被排到前面
		return r.HttpAnser.EndPointHandlers[i].priority > r.HttpAnser.EndPointHandlers[j].priority
	})
}

func (r *Router) combinePath(relativePath string) string {
	return strings.TrimRight(path.Join(r.path, relativePath), "/")
}

func (r *Router) combineNodes(relativePath string) []*node {
	nodes := []*node{}
	nodes = append(nodes, r.nodes...)
	splits := strings.Split(relativePath, "/")
	var n *node
	for _, s := range splits {
		if s == "" {
			continue
		}
		n = newNode(s)
		nodes = append(nodes, n)
	}
	return nodes
}

func (r *Router) combineHandlers(handlers HandlerChain) HandlerChain {
	size := len(r.Handlers) + len(handlers)
	mergedHandlers := make(HandlerChain, size)
	copy(mergedHandlers, r.Handlers)
	copy(mergedHandlers[len(r.Handlers):], handlers)
	return mergedHandlers
}

// ====================================================================================================
// EndPoint
// ====================================================================================================
type EndPoint struct {
	path     string
	nodes    []*node
	nNode    int32
	priority float32
	params   map[string]any
	// key: HttpMethod(GET/POST/...), value: handler functions
	Handlers map[string]HandlerChain
	options  []string
}

func NewEndPoint() *EndPoint {
	ep := &EndPoint{
		nodes:    []*node{},
		nNode:    0,
		priority: 0,
		params:   make(map[string]any),
		Handlers: map[string]HandlerChain{
			ghttp.MethodOptions: {},
		},
		options: []string{ghttp.MethodOptions},
	}
	return ep
}

func (ep *EndPoint) InitNodes(nodes []*node) {
	var n *node
	for _, n = range nodes {
		if n.isParam {
			if n.routeType == "int" || n.routeType == "uint" || n.routeType == "float" {
				ep.priority += 0.5
			}
		} else {
			ep.priority += 1.0
		}
		ep.nodes = append(ep.nodes, n)
	}
	ep.nNode = int32(len(ep.nodes))
}

func (ep *EndPoint) Macth(routes []string) bool {
	if ep.nNode != int32(len(routes)) {
		return false
	}
	var n *node
	for i, route := range routes {
		n = ep.nodes[i]
		if !n.match(route) {
			return false
		}
	}
	for _, n := range ep.nodes {
		if n.isParam {
			ep.SetParam(n.route, n.value)
		}
	}
	return true
}

func (ep *EndPoint) SetParam(key string, value any) {
	ep.params[key] = value
}

// ====================================================================================================
// node
// ====================================================================================================
type node struct {
	route     string
	routeType string
	isParam   bool
	value     any
}

func newNode(route string) *node {
	n := new(node)
	if strings.HasPrefix(route, "<") && strings.HasSuffix(route, ">") {
		n.isParam = true
		route = route[1 : len(route)-1]
	}
	routes := strings.Split(route, " ")
	n.route = routes[0]
	if len(routes) > 1 {
		n.routeType = routes[1]
	}
	return n
}

func (n *node) match(route string) bool {
	if n.isParam {
		switch n.routeType {
		case "int":
			i, err := strconv.ParseInt(route, 10, 64)
			if err != nil {
				return false
			}
			n.value = i
		case "uint":
			i, err := strconv.ParseUint(route, 10, 64)
			if err != nil {
				return false
			}
			n.value = i
		case "float":
			f, err := strconv.ParseFloat(route, 64)
			if err != nil {
				return false
			}
			n.value = f
		case "string", "":
			n.value = route
		default:
			return false
		}
		return true
	} else {
		return route == n.route
	}
}
