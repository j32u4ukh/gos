package ans

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gos/base"
	"net"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	MethodGet  = "GET"
	MethodPost = "POST"
)

var (
	COLON []byte = []byte(":")
)

type HandlerFunc func(req base.Request, res *base.Response)
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
	// rrStates & Request & Response
	// 個數與 Anser 的 nConnect 相同，因此可利用 Conn 中的 id 作為索引值，來存取 rrStates, Request 與 Response
	// 由於是使用 Conn 的 id 作為索引值，因此可以不用從第一個開始使用，結束使用後也不需要對順序進行調整
	// ==================================================
	r2s    []*base.R2
	currR2 *base.R2
}

func NewHttpAnser(laddr *net.TCPAddr, nConnect int32, nWork int32) (IAnswer, error) {
	var err error
	a := &HttpAnser{
		Handlers: map[string]map[string]HandlerChain{
			MethodGet:  {},
			MethodPost: {},
		},
		r2s:    make([]*base.R2, nConnect),
		currR2: nil,
	}

	// ===== Anser =====
	a.Anser, err = newAnser(laddr, nConnect, nWork)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to new HttpAnser.")
	}

	// 設置數據讀取函式
	a.Anser.read = a.Read

	// ===== Router =====
	a.Router = &Router{
		HttpAnser: a,
		path:      "/",
		Handlers:  HandlerChain{},
	}

	// ===== R2 =====
	var i int32
	for i = 0; i < nConnect; i++ {
		a.r2s[i] = base.NewR2()
	}

	return a, nil
}

// 監聽連線並註冊
func (a *HttpAnser) Listen() {
	fmt.Printf("(a *HttpAnser) Listen\n")
	a.SetWorkHandler()
	a.Anser.Listen()
}

func (a *HttpAnser) Handler() {
	a.Anser.Handler()
}

func (a *HttpAnser) Read() bool {
	// 根據 Conn 的 Id，存取對應的 R2
	a.currR2 = a.r2s[a.currConn.GetId()]

	// 讀取 第一行
	if a.currR2.State == 0 {
		if a.currConn.CheckReadable(a.currR2.HasLineData) {
			a.currConn.Read(&a.readBuffer, a.currR2.ReadLength)

			// 拆分第一行數據
			firstLine := string(a.readBuffer[:a.currR2.ReadLength])
			fmt.Printf("(a *HttpAnser) Read | firstLine: %s\n", firstLine)
			ok := a.currR2.Request.ParseFirstLine(firstLine)

			if ok {
				// 解析第一行數據中的請求路徑
				a.currR2.Request.ParseQuery()
				a.currR2.State = 1
				fmt.Printf("(a *HttpAnser) Read | State: 0 -> 1\n")
			}
		}
	}

	// 讀取 Header 數據
	if a.currR2.State == 1 {
		for a.currConn.CheckReadable(a.currR2.HasLineData) {
			// 讀取一行數據
			a.currConn.Read(&a.readBuffer, a.currR2.ReadLength)

			// mustHaveFieldNameColon ensures that, per RFC 7230, the field-name is on a single line,
			// so the first line must contain a colon.
			// 將讀到的數據從冒號拆分成 key, value
			k, v, ok := bytes.Cut(a.readBuffer[:a.currR2.ReadLength], COLON)

			// 當前這行數據不包含":"，結束 Header 的讀取
			if !ok {
				if _, ok := a.currR2.Header["Content-Length"]; !ok {
					// 考慮分包問題，收到完整一包數據傳完才傳到應用層
					a.currWork.Index = a.currConn.Index
					a.currWork.RequestTime = time.Now().UTC()
					a.currWork.State = 1
					a.currWork.Body.ResetIndex()

					// 指向下一個工作結構
					a.currWork = a.currWork.Next

					// Header 中不包含 Content-Length，狀態值恢復為 0
					a.currR2.State = 0
					return true

				} else {
					// Header 中包含 Content-Length，狀態值設為 2，等待讀取後續數據
					length, err := strconv.Atoi(a.currR2.Header["Content-Length"][0])
					fmt.Printf("(a *HttpAnser) Read | Content-Length: %d\n", length)

					if err != nil {
						fmt.Printf("(a *HttpAnser) Read | Content-Length err: %+v\n", err)
						return false
					}

					a.currR2.ReadLength = int32(length)
					a.currR2.State = 2
					fmt.Printf("(a *HttpAnser) Read | State: 1 -> 2\n")
				}
			}

			key := string(k)

			if _, ok := a.currR2.Header[key]; !ok {
				a.currR2.Header[key] = []string{}
			}

			a.currR2.Header[key] = append(a.currR2.Header[key], strings.TrimLeft(string(v), " \t"))
			fmt.Printf("(a *HttpAnser) Read | Header, key: %s, value: %s\n", key, a.currR2.Header[key])
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

			// 考慮分包問題，收到完整一包數據傳完才傳到應用層
			a.currWork.Index = a.currConn.GetId()
			a.currWork.RequestTime = time.Now().UTC()
			a.currWork.State = 1
			a.currWork.Body.AddRawData(a.readBuffer[:a.currR2.ReadLength])
			a.currWork.Body.ResetIndex()

			// 指向下一個工作結構
			a.currWork = a.currWork.Next

			// 重置狀態值
			a.currR2.State = 0
		}
	}

	return true
}

func (a *HttpAnser) Write(cid int32, data *[]byte, length int32) error {
	return a.Anser.Write(cid, data, length)
}

// 由外部定義 workHandler，定義如何處理工作
func (a *HttpAnser) SetWorkHandler() {

	for method, functions := range a.Handlers {
		for p := range functions {
			fmt.Printf("(a *HttpAnser) SetWorkHandler | handlers method: %s, path: %s\n", method, p)
		}
	}

	a.workHandler = func(w *base.Work) {
		r2 := a.r2s[w.Index]

		if handler, ok := a.Handlers[r2.Request.Method]; ok {
			if functions, ok := handler[r2.Request.Query]; ok {
				for _, f := range functions {
					f(*r2.Request, r2.Response)

					for k := range r2.Header {
						delete(r2.Header, k)
					}

					r2.Code = 200
					r2.Message = "OK"
					r2.SetHeader("Connection", "close")
					r2.SetHeader("Content-Type", "application/json")
					j, _ := json.Marshal(map[string]any{
						"id":  w.Index,
						"msg": "json message",
					})
					r2.SetBody(j)
					// TODO: 將 Response 回傳數據轉換成 Work 傳遞的格式
					bs := r2.FormResponse()
					fmt.Printf("Response: %s\n", string(bs))
					// r := []byte("HTTP/1.1 200 OK\r\nConnection: close\r\nContent-Type: text/html\r\nContent-Length: 19\r\n\r\n<h1>Hola Mundo</h1>")
					w.Body.AddRawData(bs)
					w.Send()
				}
			}
		}
	}
}

func (a *HttpAnser) ServeHttp(method string, path string) {
	fmt.Printf("(s *Server) ServeHttp | method: %s, path: %s\n", method, path)

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
	r.handle(MethodGet, path, handlers...)
}

func (r *Router) POST(path string, handlers ...HandlerFunc) {
	r.handle(MethodPost, path, handlers...)
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
	// assert1(finalSize < int(abortIndex), "too many handlers")
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
