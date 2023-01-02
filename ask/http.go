package ask

import (
	"fmt"
	"gos/base"
	"gos/base/ghttp"
	"gos/define"
	"net"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type CallbackHandler struct {
	// 對應工作結構的 id
	WorkId int32
	Data   []byte
	Length int32
	// 工作完成時的 Callback 函式
	Callback func(*ghttp.Response)
}

func newCallbackHandler(workId int32, callback func(*ghttp.Response)) *CallbackHandler {
	handler := &CallbackHandler{
		WorkId:   workId,
		Data:     make([]byte, 64*1024),
		Length:   0,
		Callback: callback,
	}
	return handler
}

func (h *CallbackHandler) SetData(data []byte, length int32) {
	h.Length = length
	copy(h.Data[:length], data[:length])
}

func (h *CallbackHandler) String() string {
	return fmt.Sprintf("CallbackHandler(WorkId: %d, Length: %d)\nData: %+v", h.WorkId, h.Length, h.Data[:h.Length])
}

type HttpAsker struct {
	*Asker

	// ==================================================
	// Request & Response
	// 個數與 nConnect 相同，利用 Conn 中的 id 作為索引值，來存取 R2
	// 可以不用從第一個開始使用，結束使用後也不需要對順序進行調整
	// 和工作結構不同，一個 Conn 和一個 R2 一一對應，但可以有 0 到多個工作結構
	// ==================================================
	r2s    []*ghttp.R2
	currR2 *ghttp.R2

	// 依序處理請求
	Handlers []*CallbackHandler
}

func NewHttpAsker(site int32, laddr *net.TCPAddr, nConnect int32, nWork int32) (IAsker, error) {
	var err error
	a := &HttpAsker{
		r2s:      make([]*ghttp.R2, nConnect),
		currR2:   nil,
		Handlers: []*CallbackHandler{},
	}

	// ===== Anser =====
	a.Asker, err = newAsker(site, laddr, nConnect, nWork, false)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to new HttpAsker.")
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
		a.r2s[i] = ghttp.NewR2(i)
	}

	//////////////////////////////////////////////////
	// HttpAsker 自定義函式
	//////////////////////////////////////////////////
	a.Asker.readFunc = a.Read
	a.Asker.writeFunc = a.writeFunc
	return a, nil
}

func (a *HttpAsker) Connect() error {
	return a.Asker.Connect(-1)
}

func (a *HttpAsker) Read() {
	// 根據 Conn 的 Id，存取對應的 R2
	a.currR2 = a.r2s[a.currConn.GetId()]
	fmt.Printf("(a *HttpAsker) Read | Conn(%d), State: %d\n", a.currConn.GetId(), a.currR2.State)

	// 讀取 第一行
	if a.currR2.State == 0 {
		if a.currConn.CheckReadable(a.currR2.HasLineData) {
			a.currConn.Read(&a.readBuffer, a.currR2.ReadLength)

			// 拆分第一行數據 HTTP/1.1 200 OK\r\n
			firstLine := strings.TrimRight(string(a.readBuffer[:a.currR2.ReadLength]), "\r\n")
			fmt.Printf("(a *HttpAsker) Read | firstLine: %s\n", firstLine)
			a.currR2.Response.ParseFirstLine(firstLine)
			a.currR2.State = 1
			fmt.Printf("(a *HttpAsker) Read | State: 0 -> 1\n")
		}
	}

	// 讀取 Header 數據
	if a.currR2.State == 1 {
		var headerLine, key, value string
		var ok bool

		for a.currConn.CheckReadable(a.currR2.HasLineData) && a.currR2.State == 1 {
			// 讀取一行數據
			a.currConn.Read(&a.readBuffer, a.currR2.ReadLength)

			// mustHaveFieldNameColon ensures that, per RFC 7230, the field-name is on a single line,
			// so the first line must contain a colon.
			// 將讀到的數據從冒號拆分成 key, value
			// k, v, ok := bytes.Cut(a.readBuffer[:a.currR2.ReadLength], COLON)
			headerLine = strings.TrimRight(string(a.readBuffer[:a.currR2.ReadLength]), "\r\n")
			key, value, ok = strings.Cut(headerLine, ghttp.COLON)

			if ok {
				// 持續讀取 Header
				// key := string(k)

				if _, ok := a.currR2.Header[key]; !ok {
					a.currR2.Header[key] = []string{}
				}

				value = strings.TrimLeft(value, " \t")
				// value = strings.TrimRight(value, "\r\n")
				a.currR2.Header[key] = append(a.currR2.Header[key], value)
				fmt.Printf("(a *HttpAsker) Read | Header, key: %s, value: %s\n", key, value)

			} else {
				// 當前這行數據不包含":"，結束 Header 的讀取

				// Header 中包含 Content-Length，狀態值設為 2，等待讀取後續數據
				if contentLength, ok := a.currR2.Header["Content-Length"]; ok {
					length, err := strconv.Atoi(contentLength[0])
					fmt.Printf("(a *HttpAsker) Read | Content-Length: %d\n", length)

					if err != nil {
						fmt.Printf("(a *HttpAsker) Read | Content-Length err: %+v\n", err)
						return
					}

					a.currR2.ReadLength = int32(length)
					a.currR2.State = 2
					fmt.Printf("(a *HttpAsker) Read | State: 1 -> 2\n")

				} else {
					// Header 中不包含 Content-Length，狀態值恢復為 0
					a.currR2.State = 0
					return
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
			fmt.Printf("(a *HttpAsker) Read | %s\n", string(a.readBuffer[:a.currR2.ReadLength]))
			a.currR2.BodyLength = a.currR2.ReadLength
			copy(a.currR2.Body[:a.currR2.ReadLength], a.readBuffer[:a.currR2.ReadLength])

			// 重置狀態值
			a.currR2.State = 0
		}
	}
}

func (a *HttpAsker) writeFunc(id int32, data *[]byte, length int32) error {
	fmt.Printf("(a *HttpAsker) writeFunc | work id: %d\n", id)

	// 取得連線物件(若 id 為 -1，表示尋找空閒的連線物件)
	a.currConn = a.getConn(id)

	// 目前沒有空閒的連線物件，等待下次迴圈再處理
	if a.currConn == nil {
		fmt.Printf("(a *HttpAsker) writeFunc | currConn is nil\n")
		return nil
	}

	if a.currConn.State == define.Unused {
		fmt.Printf("(a *HttpAsker) writeFunc | currConn.State is Unused\n")
		a.Asker.Connect(a.currConn.GetId())
	}

	// 設置當前工作結構對應的連線物件
	a.currWork.Index = a.currConn.GetId()

	// 將數據寫入連線物件的緩存
	a.currConn.SetWriteBuffer(data, length)
	return nil
}

func (a *HttpAsker) Write(data *[]byte, length int32) error {
	// 取得空的工作結構
	w := a.getEmptyWork()
	// 標註此工作之後須寫出
	w.State = 2
	// 標註此工作未指定寫出的連線物件，由空閒的連線物件來寫出
	w.Index = -1
	handler := newCallbackHandler(w.GetId(), func(r *ghttp.Response) {
		fmt.Printf("Response: %v\n", r)
	})
	handler.SetData(*data, length)
	a.Handlers = append(a.Handlers, handler)
	return nil
}

// [Work State: 1] 由外部定義 workHandler，定義如何處理工作
func (a *HttpAsker) SetWorkHandler() {
	a.Asker.workHandler = func(w *base.Work) {
		if w.Index == -1 {
			return
		}

		fmt.Printf("(a *HttpAsker) workHandler | work: %+v\n", w)

		// 取得連線物件
		a.currConn = a.getConn(w.Index)

		// 根據 Conn 的 Id，存取對應的 R2
		a.currR2 = a.r2s[a.currConn.GetId()]

		var handler *CallbackHandler
		var ok bool = false

		for _, handler = range a.Handlers {
			if handler.WorkId == w.GetId() {
				ok = true
				break
			}
		}

		if ok {
			// 將取得的 Response，透過註冊的 Callback 函釋回傳回去
			fmt.Printf("(a *HttpAsker) Callback | Response: %+v\n", a.currR2.Response)
			handler.Callback(a.currR2.Response)
		}
	}
}

func (a *HttpAsker) Send(req *ghttp.Request, callback func(*ghttp.Response)) error {
	fmt.Printf("(a *HttpAsker) Send | req: %+v\n", req)

	if callback == nil {
		return errors.New("callback 函式不可為 nil")
	}

	// 取得空的工作結構
	w := a.getEmptyWork()
	// 標註此工作之後須寫出
	w.State = 2
	// 標註此工作未指定寫出的連線物件，由空閒的連線物件來寫出
	w.Index = -1
	fmt.Printf("(a *HttpAsker) Send | work: %+v\n", w)

	handler := newCallbackHandler(w.GetId(), callback)
	data := req.FormRequest()
	handler.SetData(data, int32(len(data)))
	fmt.Printf("(a *HttpAsker) Send | handler: %+v\n", handler)
	a.Handlers = append(a.Handlers, handler)
	return nil
}
