package ask

import (
	"fmt"
	"gos/base"
	"gos/base/ghttp"
	"gos/define"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type HttpAsker2 struct {
	*Asker

	// ==================================================
	// Request & Response
	// 個數與 nConnect 相同，利用 Conn 中的 id 作為索引值，來存取 R2
	// 可以不用從第一個開始使用，結束使用後也不需要對順序進行調整
	// 和工作結構不同，一個 Conn 和一個 R2 一一對應，但可以有 0 到多個工作結構
	// ==================================================
	httpConns []*ghttp.Context
	httpConn  *ghttp.Context

	contextPool sync.Pool
	context     *ghttp.Context

	// 依序處理請求
	Handlers map[int32]ghttp.HandlerFunc
}

func NewHttpAsker2(site int32, laddr *net.TCPAddr, nConnect int32, nWork int32) (IAsker, error) {
	var err error
	a := &HttpAsker2{
		httpConns:   make([]*ghttp.Context, nConnect),
		httpConn:    nil,
		contextPool: sync.Pool{New: func() any { return &ghttp.Context{} }},
		context:     nil,
		Handlers:    map[int32]ghttp.HandlerFunc{},
	}

	// ===== Anser =====
	a.Asker, err = newAsker(site, laddr, nConnect, nWork, false)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to new HttpAsker2.")
	}
	a.currConn = a.conns

	// 設置連線的模式
	for a.currConn != nil {
		a.currConn.Mode = base.CLOSE
		a.currConn = a.currConn.Next
	}

	// ===== R2 =====
	var i int32
	for i = 0; i < nConnect; i++ {
		a.httpConns[i] = ghttp.NewContext(i)
	}

	//////////////////////////////////////////////////
	// HttpAsker2 自定義函式
	//////////////////////////////////////////////////
	a.readFunc = a.read
	a.writeFunc = a.write

	a.SetWorkHandler()
	return a, nil
}

func (a *HttpAsker2) Connect() error {
	return a.Asker.Connect(-1)
}

func (a *HttpAsker2) read() {
	// 根據 Conn 的 Id，存取對應的 R2
	a.httpConn = a.httpConns[a.currConn.GetId()]
	fmt.Printf("(a *HttpAsker2) Read | Conn(%d), State: %d\n", a.currConn.GetId(), a.httpConn.State)

	// 讀取 第一行
	if a.httpConn.State == 0 {
		if a.currConn.CheckReadable(a.httpConn.HasLineData) {
			a.currConn.Read(&a.readBuffer, a.httpConn.ReadLength)

			// 拆分第一行數據 HTTP/1.1 200 OK\r\n
			firstLine := strings.TrimRight(string(a.readBuffer[:a.httpConn.ReadLength]), "\r\n")
			fmt.Printf("(a *HttpAsker2) Read | firstLine: %s\n", firstLine)
			a.httpConn.ParseFirstResLine(firstLine)
			a.httpConn.State = 1
			fmt.Printf("(a *HttpAsker2) Read | State: 0 -> 1\n")
		}
	}

	// 讀取 Header 數據
	if a.httpConn.State == 1 {
		var headerLine, key, value string
		var ok bool

		for a.currConn.CheckReadable(a.httpConn.HasLineData) && a.httpConn.State == 1 {
			// 讀取一行數據
			a.currConn.Read(&a.readBuffer, a.httpConn.ReadLength)

			// mustHaveFieldNameColon ensures that, per RFC 7230, the field-name is on a single line,
			// so the first line must contain a colon.
			// 將讀到的數據從冒號拆分成 key, value
			// k, v, ok := bytes.Cut(a.readBuffer[:a.currR2.ReadLength], COLON)
			headerLine = strings.TrimRight(string(a.readBuffer[:a.httpConn.ReadLength]), "\r\n")
			key, value, ok = strings.Cut(headerLine, ghttp.COLON)

			if ok {
				// 持續讀取 Header
				// key := string(k)

				if _, ok := a.httpConn.Header[key]; !ok {
					a.httpConn.Header[key] = []string{}
				}

				value = strings.TrimLeft(value, " \t")
				// value = strings.TrimRight(value, "\r\n")
				a.httpConn.Header[key] = append(a.httpConn.Header[key], value)
				fmt.Printf("(a *HttpAsker2) Read | Header, key: %s, value: %s\n", key, value)

			} else {
				// 當前這行數據不包含":"，結束 Header 的讀取
				fmt.Printf("(a *HttpAsker2) Read | Empty line\n")

				// Header 中包含 Content-Length，狀態值設為 2，等待讀取後續數據
				if contentLength, ok := a.httpConn.Header["Content-Length"]; ok {
					length, err := strconv.Atoi(contentLength[0])
					fmt.Printf("(a *HttpAsker2) Read | Content-Length: %d\n", length)

					if err != nil {
						fmt.Printf("(a *HttpAsker2) Read | Content-Length err: %+v\n", err)
						return
					}

					a.httpConn.ReadLength = int32(length)
					fmt.Printf("(a *HttpAsker2) Read | a.currR2.ReadLength: %d\n", a.httpConn.ReadLength)
					a.httpConn.State = 2
					fmt.Printf("(a *HttpAsker2) Read | State: 1 -> 2\n")

				} else {
					// Header 中不包含 Content-Length，狀態值恢復為 0
					a.httpConn.State = 0

					// 數據已讀入 currR2 當中，此處工作結構僅負責觸發 WorkHandler，進一步觸發 Callback 函式
					a.currWork.Index = a.currConn.GetId()
					a.currWork.RequestTime = time.Now().UTC()
					a.currWork.State = 1
				}
				return
			}
		}
	}

	// 讀取 Body 數據
	if a.httpConn.State == 2 {
		fmt.Printf("(a *HttpAsker2) Read | State 2, a.currR2.ReadLength: %d\n", a.httpConn.ReadLength)
		if a.currConn.CheckReadable(a.httpConn.HasEnoughData) {
			// ==========
			// 讀取 data
			// ==========
			// 將傳入的數據，加入工作緩存中
			a.currConn.Read(&a.readBuffer, a.httpConn.ReadLength)
			fmt.Printf("(a *HttpAsker2) Read | State 2, data: %s\n", string(a.readBuffer[:a.httpConn.ReadLength]))
			a.httpConn.BodyLength = a.httpConn.ReadLength
			copy(a.httpConn.Body[:a.httpConn.ReadLength], a.readBuffer[:a.httpConn.ReadLength])

			// 重置狀態值
			a.httpConn.State = 0

			// 數據已讀入 currR2 當中，此處工作結構僅負責觸發 WorkHandler，進一步觸發 Callback 函式
			a.currWork.Index = a.currConn.GetId()
			a.currWork.RequestTime = time.Now().UTC()
			a.currWork.State = 1

			// 指向下一個工作結構
			a.currWork = a.currWork.Next
		}
	}
}

func (a *HttpAsker2) write(id int32, data *[]byte, length int32) error {
	fmt.Printf("(a *HttpAsker2) writeFunc | work id: %d\n", id)

	// 取得連線物件(若 id 為 -1，表示尋找空閒的連線物件)
	a.currConn = a.getConn(id)

	// 目前沒有空閒的連線物件，等待下次迴圈再處理
	if a.currConn == nil {
		fmt.Printf("(a *HttpAsker2) writeFunc | currConn is nil\n")
		return nil
	}

	if a.currConn.State == define.Unused {
		fmt.Printf("(a *HttpAsker2) writeFunc | currConn.State is Unused\n")
		a.currConn.State = define.Connecting

		// 設置當前工作結構對應的連線物件
		a.currWork.Index = a.currConn.GetId()
		fmt.Printf("(a *HttpAsker2) writeFunc | a.currWork.Index <- %d\n", a.currConn.GetId())
		a.Asker.Connect(a.currConn.GetId())
		return nil
	} else if a.currConn.State == define.Connecting {
		fmt.Printf("(a *HttpAsker2) writeFunc | currConn.State is Connecting\n")
		return nil
	}

	// 將數據寫入連線物件的緩存
	fmt.Printf("(a *HttpAsker2) writeFunc | WriteBuffer, length: %d, data: %+v\n", length, (*data)[:length])
	a.currConn.SetWriteBuffer(data, length)
	a.currWork.State = 0
	return nil
}

// 原始數據寫出函式，缺乏定義 Callback 函式的能力，應使用 Send 來傳送請求
func (a *HttpAsker2) Write(data *[]byte, length int32) error {
	// 取得空的工作結構
	w := a.getEmptyWork()
	// 標註此工作未指定寫出的連線物件，由空閒的連線物件來寫出
	w.Index = -1
	w.Body.AddRawData((*data)[:length])

	a.Handlers[w.GetId()] = func(c *ghttp.Context) {
		fmt.Printf("Response: %+v\n", c)
	}

	w.Send()

	return nil
}

// [Work State: 1] 由外部定義 workHandler，定義如何處理工作
func (a *HttpAsker2) SetWorkHandler() {
	fmt.Printf("(a *HttpAsker2) SetWorkHandler\n")

	a.Asker.workHandler = func(w *base.Work) {
		if w.Index == -1 {
			return
		}

		fmt.Printf("(a *HttpAsker2) SetWorkHandler | work: %+v\n", w)

		// 取得連線物件
		a.currConn = a.getConn(w.Index)

		// 根據 Conn 的 Id，存取對應的 R2
		a.httpConn = a.httpConns[a.currConn.GetId()]

		if handler, ok := a.Handlers[w.GetId()]; ok {
			// 將取得的 Response，透過註冊的 Callback 函釋回傳回去
			handler(a.httpConn)
		}

		w.Finish()
	}
}

func (a *HttpAsker2) NewRequest(method string, uri string, params map[string]string) *ghttp.Request2 {
	a.context = a.contextPool.Get().(*ghttp.Context)
	a.context.Request2.FormRequest(method, uri, params)
	return a.context.Request2
}

// 供外部傳送 Http 請求
func (a *HttpAsker2) Send(req *ghttp.Request2, callback func(*ghttp.Context)) error {
	fmt.Printf("(a *HttpAsker2) Send | req: %+v\n", req)

	if callback == nil {
		return errors.New("callback 函式不可為 nil")
	}

	// 取得空的工作結構
	w := a.getEmptyWork()
	// 標註此工作未指定寫出的連線物件，由空閒的連線物件來寫出
	w.Index = -1
	w.Body.AddRawData(req.ToRequestData())
	a.Handlers[w.GetId()] = callback
	w.Send()
	fmt.Printf("(a *HttpAsker2) Send | work: %+v\n", w)
	// 釋放 req *ghttp.Request2
	req.Release()
	// 將 Request2 放回物件池
	a.contextPool.Put(req)
	return nil
}
