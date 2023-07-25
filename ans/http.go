package ans

import (
	"fmt"
	"net"
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
	Handlers map[string]map[int32][]*EndPoint

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
		Handlers: map[string]map[int32][]*EndPoint{
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
		// 最開始的 '/' 會形成空字串的 node
		nodes:    []*node{newNode("")},
		Handlers: HandlerChain{},
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
			utils.Info("firstLine: %s", a.lineString)

			if a.httpConn.ParseFirstReqLine(a.lineString) {
				if a.httpConn.Method == ghttp.MethodGet {
					// 解析第一行數據中的請求路徑
					a.httpConn.ParseQuery()
				}
				a.httpConn.State = 1
				utils.Debug("State: 0 -> 1")
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
				utils.Debug("Header, key: %s, value: %s", key, value)

			} else {
				// 當前這行數據不包含":"，結束 Header 的讀取

				// Header 中包含 Content-Length，狀態值設為 2，等待讀取後續數據
				if contentLength, ok := a.httpConn.Header["Content-Length"]; ok {
					length, err := strconv.Atoi(contentLength[0])
					// fmt.Printf("(a *HttpAnser) Read | Content-Length: %d\n", length)
					utils.Debug("Content-Length: %d", length)

					if err != nil {
						// fmt.Printf("(a *HttpAnser) Read | Content-Length err: %+v\n", err)
						utils.Error("Content-Length err: %+v", err)
						return false
					}

					a.httpConn.ReadLength = int32(length)
					a.httpConn.State = 2
					// fmt.Printf("(a *HttpAnser) Read | State: 1 -> 2\n")
					utils.Debug("State: 1 -> 2")

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
					utils.Debug("State: 1 -> 3")
					return true
				}
			}
		}
	}

	// 讀取 Body 數據
	if a.httpConn.State == 2 {
		if a.currConn.CheckReadable(a.httpConn.HasEnoughData) {
			// 將傳入的數據，加入工作緩存中
			a.currConn.Read(&a.readBuffer, a.httpConn.ReadLength)
			utils.Debug("Body 數據: %s", string(a.readBuffer[:a.httpConn.ReadLength]))

			// 考慮分包問題，收到完整一包數據傳完才傳到應用層
			a.currWork.Index = a.currConn.GetId()
			a.currWork.RequestTime = time.Now().UTC()
			a.currWork.State = 1
			a.httpConn.SetBody(a.readBuffer, a.httpConn.ReadLength)

			// 指向下一個工作結構
			a.currWork = a.currWork.Next

			// 等待數據寫出
			a.httpConn.State = 3
			// fmt.Printf("(a *HttpAnser) Read | State: 2 -> 3\n")
			utils.Debug("State: 2 -> 3")
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
		utils.Debug("Cid: %d, Wid: %d", a.httpConn.Cid, a.httpConn.Wid)
		var endpoints []*EndPoint
		var key string
		var value any
		var unmatched bool = true

		if handler, ok := a.Handlers[a.httpConn.Method]; ok {
			var nSplit int32
			var splits []string

			if a.httpConn.Query == "" || a.httpConn.Query == "/" {
				nSplit = 1
				splits = []string{""}
			} else {
				a.httpConn.Query = strings.TrimSuffix(a.httpConn.Query, "/")
				splits = strings.Split(a.httpConn.Query, "/")
				nSplit = int32(len(splits))
			}

			if endpoints, ok = handler[nSplit]; ok {
				for _, endpoint := range endpoints {
					// handlerFunc(a.httpConn)
					if endpoint.Macth(splits) {
						unmatched = false
						for key, value = range endpoint.params {
							if _, ok = a.httpConn.Params[key]; !ok {
								a.httpConn.Params[key] = fmt.Sprintf("%v", value)
							}
							if _, ok = a.httpConn.Values[key]; !ok {
								a.httpConn.Values[key] = value
							}
						}
						for _, function := range endpoint.Handlers {
							function(a.httpConn)
						}
						break
					}
				}
				if unmatched {
					a.errorRequestHandler(w, a.httpConn, "Unmatched endpoint.")
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
	utils.Debug("method: %s, query: %s", c.Method, c.Query)

	c.Json(400, ghttp.H{
		"code": 400,
		"msg":  msg,
	})
	c.SetHeader("Connection", "close")

	// 將 Response 回傳數據轉換成 Work 傳遞的格式
	bs := c.ToResponseData()
	// fmt.Printf("Response: %s\n", string(bs))
	utils.Debug("Response: %s", string(bs))

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
		utils.Info("Conn(%d) 完成數據寫出，準備關閉連線", a.currConn.GetId())
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
	utils.Debug("Response: %s", string(bs))
	utils.Debug("Raw Response: %s", utils.SliceToString(bs))

	w := a.getWork(c.Wid)
	// fmt.Printf("Wid: %d, w: %+v\n", c.Wid, w)
	utils.Debug("Wid: %d, w: %+v", c.Wid, w)

	w.Index = c.Cid
	// fmt.Printf("c.Cid: %d, w.Index: %d\n", c.Cid, w.Index)
	utils.Debug("c.Cid: %d, w.Index: %d", c.Cid, w.Index)

	w.Body.AddRawData(bs)
	w.Send()
	// fmt.Printf("Wid: %d, w: %+v\n", c.Wid, w)
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
	nodes    []*node
	Handlers HandlerChain
}

// 每個 EndPoint 對應一個 Router，但每個 Router 不一定對應著一個 EndPoint
func (r *Router) NewRouter(relativePath string, handlers ...HandlerFunc) *Router {
	nr := &Router{
		HttpAnser: r.HttpAnser,
		Handlers:  r.combineHandlers(handlers),
		nodes:     []*node{},
	}
	nr.nodes = r.combineNodes(relativePath)
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
		path = strings.TrimSuffix(path, "/")
		// 結合前段路由的節點們，以及當前路由的節點們
		nodes := r.combineNodes(path)
		nNode := int32(len(nodes))

		if _, ok := routers[nNode]; !ok {
			routers[nNode] = []*EndPoint{}
		}

		// 添加路徑對應的處理函式
		endpoint := NewEndPoint()
		endpoint.InitNodes(nodes)
		endpoint.Handlers = r.combineHandlers(handlers)
		routers[nNode] = append(routers[nNode], endpoint)
		sort.SliceStable(routers[nNode], func(i, j int) bool {
			// True 的話，會被排到前面
			return routers[nNode][i].priority > routers[nNode][j].priority
		})
	}
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
	nodes    []*node
	nNode    int32
	priority float32
	params   map[string]any
	Handlers HandlerChain
}

func NewEndPoint() *EndPoint {
	ep := &EndPoint{
		nodes:    []*node{},
		nNode:    0,
		priority: 0,
		params:   make(map[string]any),
		Handlers: HandlerChain{},
	}
	return ep
}

func (ep *EndPoint) InitNodes(nodes []*node) {
	var n *node
	for _, n = range nodes {
		if n.isParam {
			if n.routeType == "int" || n.routeType == "float" {
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
