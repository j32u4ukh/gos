package ans

import (
	"fmt"
	"gos/base"
	"gos/base/ghttp"
	"net"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type HandlerFunc func(req ghttp.Request, res *ghttp.Response)
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
	// Request & Response
	// 個數與 Anser 的 nConnect 相同，因此可利用 Conn 中的 id 作為索引值，來存取 rrStates, Request 與 Response
	// 由於是使用 Conn 的 id 作為索引值，因此可以不用從第一個開始使用，結束使用後也不需要對順序進行調整
	// ==================================================
	r2s    []*ghttp.R2
	currR2 *ghttp.R2

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
		r2s:    make([]*ghttp.R2, nConnect),
		currR2: nil,
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

	// ===== R2 =====
	var i int32
	for i = 0; i < nConnect; i++ {
		a.r2s[i] = ghttp.NewR2(i)
	}

	//////////////////////////////////////////////////
	// 自定義函式
	//////////////////////////////////////////////////
	// 設置數據讀取函式
	a.Anser.readFunc = a.Read

	return a, nil
}

// 監聽連線並註冊
func (a *HttpAnser) Listen() {
	a.SetWorkHandler()
	a.Anser.Listen()
}

func (a *HttpAnser) Read() bool {
	// 根據 Conn 的 Id，存取對應的 R2
	a.currR2 = a.r2s[a.currConn.GetId()]

	// 讀取 第一行
	if a.currR2.State == 0 {
		if a.currConn.CheckReadable(a.currR2.HasLineData) {
			a.currConn.Read(&a.readBuffer, a.currR2.ReadLength)

			// 拆分第一行數據
			a.lineString = strings.TrimRight(string(a.readBuffer[:a.currR2.ReadLength]), "\r\n")
			fmt.Printf("(a *HttpAnser) Read | firstLine: %s\n", a.lineString)

			if a.currR2.Request.ParseFirstLine(a.lineString) {
				if a.currR2.Request.Method == ghttp.MethodGet {
					// 解析第一行數據中的請求路徑
					a.currR2.Request.ParseQuery()
				}
				a.currR2.State = 1
				fmt.Printf("(a *HttpAnser) Read | State: 0 -> 1\n")
			}
		}
	}

	// 讀取 Header 數據
	if a.currR2.State == 1 {
		var key, value string
		var ok bool

		for a.currConn.CheckReadable(a.currR2.HasLineData) && a.currR2.State == 1 {
			// 讀取一行數據
			a.currConn.Read(&a.readBuffer, a.currR2.ReadLength)

			// mustHaveFieldNameColon ensures that, per RFC 7230, the field-name is on a single line,
			// so the first line must contain a colon.
			// 將讀到的數據從冒號拆分成 key, value
			// k, v, ok := bytes.Cut(a.readBuffer[:a.currR2.ReadLength], COLON)
			a.lineString = strings.TrimRight(string(a.readBuffer[:a.currR2.ReadLength]), "\r\n")
			key, value, ok = strings.Cut(a.lineString, ghttp.COLON)

			if ok {
				// 持續讀取 Header
				// key := string(k)

				if _, ok := a.currR2.Header[key]; !ok {
					a.currR2.Header[key] = []string{}
				}

				value = strings.TrimLeft(value, " \t")
				// value = strings.TrimRight(value, "\r\n")
				a.currR2.Header[key] = append(a.currR2.Header[key], value)
				fmt.Printf("(a *HttpAnser) Read | Header, key: %s, value: %s\n", key, value)

			} else {
				// 當前這行數據不包含":"，結束 Header 的讀取

				// Header 中包含 Content-Length，狀態值設為 2，等待讀取後續數據
				if contentLength, ok := a.currR2.Header["Content-Length"]; ok {
					length, err := strconv.Atoi(contentLength[0])
					fmt.Printf("(a *HttpAnser) Read | Content-Length: %d\n", length)

					if err != nil {
						fmt.Printf("(a *HttpAnser) Read | Content-Length err: %+v\n", err)
						return false
					}

					a.currR2.ReadLength = int32(length)
					a.currR2.State = 2
					fmt.Printf("(a *HttpAnser) Read | State: 1 -> 2\n")

				} else {
					// 考慮分包問題，收到完整一包數據傳完才傳到應用層
					a.currWork.Index = a.currConn.GetId()
					a.currWork.RequestTime = time.Now().UTC()
					a.currWork.State = 1
					a.currWork.Body.ResetIndex()

					// 指向下一個工作結構
					a.currWork = a.currWork.Next

					// Header 中不包含 Content-Length，狀態值恢復為 0
					a.currR2.State = 3
					return true
				}
			}
		}
	}

	// 讀取 Body 數據
	if a.currR2.State == 2 {
		if a.currConn.CheckReadable(a.currR2.HasEnoughData) {
			// ==========
			// 讀取 data
			// ==========
			// 將傳入的數據，加入工作緩存中
			a.currConn.Read(&a.readBuffer, a.currR2.ReadLength)
			fmt.Printf("(a *HttpAnser) Read | %s\n", string(a.readBuffer[:a.currR2.ReadLength]))

			// 考慮分包問題，收到完整一包數據傳完才傳到應用層
			a.currWork.Index = a.currConn.GetId()
			a.currWork.RequestTime = time.Now().UTC()
			a.currWork.State = 1
			a.currWork.Body.AddRawData(a.readBuffer[:a.currR2.ReadLength])
			a.currWork.Body.ResetIndex()

			// 指向下一個工作結構
			a.currWork = a.currWork.Next

			// 重置狀態值
			a.currR2.State = 3

			return false
		}
	}

	return true
}

func (a *HttpAnser) Write(cid int32, data *[]byte, length int32) error {
	return a.Anser.Write(cid, data, length)
}

// 由外部定義 workHandler，定義如何處理工作
func (a *HttpAnser) SetWorkHandler() {
	a.workHandler = func(w *base.Work) {
		r2 := a.r2s[w.Index]

		if handler, ok := a.Handlers[r2.Request.Method]; ok {
			if functions, ok := handler[r2.Request.Query]; ok {
				for _, f := range functions {

					f(*r2.Request, r2.Response)

					r2.SetHeader("Connection", "close")

					// 將 Response 回傳數據轉換成 Work 傳遞的格式
					bs := r2.FormResponse()
					fmt.Printf("Response: %s\n", string(bs))
					w.Body.AddRawData(bs)
					w.Send()
				}
			} else {
				a.errorRequestHandler(w, r2, "Unregistered http query.")
			}
		} else {
			a.errorRequestHandler(w, r2, "Unregistered http method.")
		}
	}
}

func (a *HttpAnser) errorRequestHandler(w *base.Work, r2 *ghttp.R2, msg string) {
	fmt.Printf("(s *Server) errorRequestHandler | method: %s, query: %s\n", r2.Request.Method, r2.Request.Query)
	r2.Response.Json(400, ghttp.H{
		"code": 400,
		"msg":  msg,
	})
	r2.SetHeader("Connection", "close")

	// 將 Response 回傳數據轉換成 Work 傳遞的格式
	bs := r2.FormResponse()
	fmt.Printf("Response: %s\n", string(bs))
	w.Body.AddRawData(bs)
	w.Send()
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
			fmt.Printf("(r *Router) handle | Duplicate handler, method: %v, path: %s\n", method, path)
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
