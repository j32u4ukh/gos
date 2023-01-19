package ans

import (
	"fmt"
	"gos/base"
	"gos/base/ghttp"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type HandlerFunc2 func(c *ghttp.Context)
type HandlerChain2 []HandlerFunc2

// ====================================================================================================
// HttpAnser
// ====================================================================================================

type HttpAnser2 struct {
	*Anser
	*Router2

	// key1: Method(Get/Post); key2: path; value: []HandlerFunc
	Handlers map[string]map[string]HandlerChain2

	// ==================================================
	// Request & Response
	// 個數與 Anser 的 nConnect 相同，因此可利用 Conn 中的 id 作為索引值，來存取 , Request 與 Response
	// 由於是使用 Conn 的 id 作為索引值，因此可以不用從第一個開始使用，結束使用後也不需要對順序進行調整
	// ==================================================
	httpConns []*ghttp.Context
	httpConn  *ghttp.Context

	// Temp variables
	lineString string
}

func NewHttpAnser2(laddr *net.TCPAddr, nConnect int32, nWork int32) (IAnswer, error) {
	var err error
	a := &HttpAnser2{
		Handlers: map[string]map[string]HandlerChain2{
			ghttp.MethodGet:  {},
			ghttp.MethodPost: {},
		},
		httpConns: make([]*ghttp.Context, nConnect),
		httpConn:  nil,
	}

	// ===== Anser =====
	a.Anser, err = newAnser(laddr, nConnect, nWork)
	a.Anser.ReadTimeout = 5000 * time.Millisecond

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to new HttpAnser.")
	}

	// ===== Router =====
	a.Router2 = &Router2{
		HttpAnser2: a,
		path:       "/",
		Handlers:   HandlerChain2{},
	}

	// ===== R2 =====
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
func (a *HttpAnser2) Listen() {
	a.SetWorkHandler()
	a.Anser.Listen()
}

func (a *HttpAnser2) read() bool {
	// 根據 Conn 的 Id，存取對應的 R2
	a.httpConn = a.httpConns[a.currConn.GetId()]

	// 讀取 第一行
	if a.httpConn.State == 0 {
		if a.currConn.CheckReadable(a.httpConn.HasLineData) {
			a.currConn.Read(&a.readBuffer, a.httpConn.ReadLength)

			// 拆分第一行數據
			a.lineString = strings.TrimRight(string(a.readBuffer[:a.httpConn.ReadLength]), "\r\n")
			fmt.Printf("(a *HttpAnser) Read | firstLine: %s\n", a.lineString)

			if a.httpConn.ParseFirstReqLine(a.lineString) {
				if a.httpConn.Method == ghttp.MethodGet {
					// 解析第一行數據中的請求路徑
					a.httpConn.ParseQuery()
				}
				a.httpConn.State = 1
				fmt.Printf("(a *HttpAnser) Read | State: 0 -> 1\n")
			}
		}
	}

	// 讀取 Header 數據
	if a.httpConn.State == 1 {
		var key, value string
		var ok bool

		for a.currConn.CheckReadable(a.httpConn.HasLineData) && a.httpConn.State == 1 {
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
				// key := string(k)

				if _, ok := a.httpConn.Header[key]; !ok {
					a.httpConn.Header[key] = []string{}
				}

				value = strings.TrimLeft(value, " \t")
				// value = strings.TrimRight(value, "\r\n")
				a.httpConn.Header[key] = append(a.httpConn.Header[key], value)
				fmt.Printf("(a *HttpAnser) Read | Header, key: %s, value: %s\n", key, value)

			} else {
				// 當前這行數據不包含":"，結束 Header 的讀取

				// Header 中包含 Content-Length，狀態值設為 2，等待讀取後續數據
				if contentLength, ok := a.httpConn.Header["Content-Length"]; ok {
					length, err := strconv.Atoi(contentLength[0])
					fmt.Printf("(a *HttpAnser) Read | Content-Length: %d\n", length)

					if err != nil {
						fmt.Printf("(a *HttpAnser) Read | Content-Length err: %+v\n", err)
						return false
					}

					a.httpConn.ReadLength = int32(length)
					a.httpConn.State = 2
					fmt.Printf("(a *HttpAnser) Read | State: 1 -> 2\n")

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
			fmt.Printf("(a *HttpAnser) Read | %s\n", string(a.readBuffer[:a.httpConn.ReadLength]))

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

			return false
		}
	}

	return true
}

func (a *HttpAnser2) write(cid int32, data *[]byte, length int32) error {
	// 取得對應的連線物件
	a.currConn = a.getConn(cid)

	if a.currConn == nil {
		return errors.New(fmt.Sprintf("There is no cid equals to %d.", cid))
	}

	a.currConn.SetWriteBuffer(data, length)

	// 等待數據寫出
	a.httpConn.State = 4
	return nil
}

// 由外部定義 workHandler，定義如何處理工作
func (a *HttpAnser2) SetWorkHandler() {
	a.workHandler = func(w *base.Work) {
		a.httpConn = a.httpConns[w.Index]

		if handler, ok := a.Handlers[a.httpConn.Method]; ok {
			if functions, ok := handler[a.httpConn.Query]; ok {
				for _, f := range functions {

					f(a.httpConn)

					a.httpConn.SetHeader("Connection", "close")

					// 將 Response 回傳數據轉換成 Work 傳遞的格式
					bs := a.httpConn.ToResponseData()
					fmt.Printf("Response: %s\n", string(bs))
					w.Body.AddRawData(bs)
					w.Send()
				}
			} else {
				a.errorRequestHandler(w, a.httpConn, "Unregistered http query.")
			}
		} else {
			a.errorRequestHandler(w, a.httpConn, "Unregistered http method.")
		}
	}
}

func (a *HttpAnser2) errorRequestHandler(w *base.Work, c *ghttp.Context, msg string) {
	fmt.Printf("(s *Server) errorRequestHandler | method: %s, query: %s\n", c.Method, c.Query)
	c.Json(400, ghttp.H{
		"code": 400,
		"msg":  msg,
	})
	c.SetHeader("Connection", "close")

	// 將 Response 回傳數據轉換成 Work 傳遞的格式
	bs := c.ToResponseData()
	fmt.Printf("Response: %s\n", string(bs))
	w.Body.AddRawData(bs)
	w.Send()
}

// 當前連線是否應斷線
func (a *HttpAnser2) shouldClose(err error) bool {
	if a.Anser.shouldClose(err) {
		return true
	}
	a.httpConn = a.httpConns[a.currConn.GetId()]
	if a.httpConn.State == 4 && a.currConn.WritableLength == 0 {
		fmt.Printf("(a *HttpAnser) shouldClose | Conn(%d) 完成數據寫出，準備關閉連線\n", a.currConn.GetId())
		a.httpConn.State = 0
		return true
	}
	return false
}

// ====================================================================================================
// Router
// ====================================================================================================
type Router2 struct {
	*HttpAnser2
	path     string
	Handlers HandlerChain2
}

func (r *Router2) NewRouter(relativePath string, handlers ...HandlerFunc2) *Router2 {
	nr := &Router2{
		HttpAnser2: r.HttpAnser2,
		path:       r.combinePath(relativePath),
		Handlers:   r.combineHandlers(handlers),
	}
	return nr
}

func (r *Router2) GET(path string, handlers ...HandlerFunc2) {
	r.handle(ghttp.MethodGet, path, handlers...)
}

func (r *Router2) POST(path string, handlers ...HandlerFunc2) {
	r.handle(ghttp.MethodPost, path, handlers...)
}

func (r *Router2) handle(method string, path string, handlers ...HandlerFunc2) {
	if routers, ok := r.HttpAnser2.Handlers[method]; ok {
		path = r.combinePath(path)

		if _, ok := routers[path]; ok {
			fmt.Printf("(r *Router) handle | Duplicate handler, method: %v, path: %s\n", method, path)
			return
		}

		// 添加路徑對應的處理函式
		routers[path] = r.combineHandlers(handlers)
	}
}

func (r *Router2) combinePath(relativePath string) string {
	return joinPaths(r.path, relativePath)
}

func (r *Router2) combineHandlers(handlers HandlerChain2) HandlerChain2 {
	size := len(r.Handlers) + len(handlers)
	mergedHandlers := make(HandlerChain2, size)
	copy(mergedHandlers, r.Handlers)
	copy(mergedHandlers[len(r.Handlers):], handlers)
	return mergedHandlers
}
