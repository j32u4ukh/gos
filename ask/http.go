package ask

import (
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/j32u4ukh/gos/base"
	"github.com/j32u4ukh/gos/base/ghttp"
	"github.com/j32u4ukh/gos/define"
	"github.com/j32u4ukh/gos/utils"

	"github.com/pkg/errors"
)

type HttpAsker struct {
	*Asker

	// ==================================================
	// Request & Response
	// 個數與 nConnect 相同，利用 Conn 中的 id 作為索引值，來存取 R2
	// 可以不用從第一個開始使用，結束使用後也不需要對順序進行調整
	// 和工作結構不同，一個 Conn 和一個 httpConn 一一對應，但可以有 0 到多個工作結構
	// ==================================================
	contextPool sync.Pool
	contexts    []*ghttp.Context
	context     *ghttp.Context

	// 依序處理請求
	Handlers map[int32]ghttp.HandlerFunc
}

func NewHttpAsker(site int32, laddr *net.TCPAddr, nConnect int32, nWork int32) (IAsker, error) {
	var err error
	a := &HttpAsker{
		contexts:    make([]*ghttp.Context, nConnect),
		context:     nil,
		contextPool: sync.Pool{New: func() any { return &ghttp.Context{} }},
		Handlers:    map[int32]ghttp.HandlerFunc{},
	}

	// ===== Anser =====
	a.Asker, err = newAsker(site, laddr, nConnect, nWork, nil, nil)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to new HttpAsker.")
	}
	a.currConn = a.conns

	// 設置連線的模式
	for a.currConn != nil {
		a.currConn.Mode = base.CLOSE
		a.currConn = a.currConn.Next
	}

	// ===== Context =====
	var i int32
	for i = 0; i < nConnect; i++ {
		a.contexts[i] = ghttp.NewContext(i)
	}

	//////////////////////////////////////////////////
	// HttpAsker 自定義函式
	//////////////////////////////////////////////////
	a.readFunc = a.read
	a.writeFunc = a.write

	a.SetWorkHandler()
	return a, nil
}

func (a *HttpAsker) Connect() error {
	return a.Asker.Connect(-1)
}

func (a *HttpAsker) read() {
	// 根據 Conn 的 Id，存取對應的 httpConn
	a.context = a.contexts[a.currConn.GetId()]
	utils.Debug("Conn(%d), State: %s", a.currConn.GetId(), a.context.State)

	// 讀取 第一行
	if a.context.State == ghttp.READ_FIRST_LINE {
		if a.currConn.CheckReadable(a.context.HasLineData) {
			a.currConn.Read(&a.readBuffer, a.context.Response.ReadLength)

			// 拆分第一行數據 HTTP/1.1 200 OK\r\n
			firstLine := strings.TrimRight(string(a.readBuffer[:a.context.Response.ReadLength]), "\r\n")
			utils.Debug("firstLine: %s", firstLine)

			a.context.ParseFirstResLine(firstLine)
			a.context.State = ghttp.READ_HEADER
			utils.Debug("State: READ_FIRST_LINE -> READ_HEADER")
		}
	}

	// 讀取 Header 數據
	if a.context.State == ghttp.READ_HEADER {
		var headerLine, key, value string
		var ok bool

		for a.currConn.CheckReadable(a.context.HasLineData) && a.context.State == ghttp.READ_HEADER {
			// 讀取一行數據
			a.currConn.Read(&a.readBuffer, a.context.Response.ReadLength)

			// mustHaveFieldNameColon ensures that, per RFC 7230, the field-name is on a single line,
			// so the first line must contain a colon.
			// 將讀到的數據從冒號拆分成 key, value
			headerLine = strings.TrimRight(string(a.readBuffer[:a.context.Response.ReadLength]), "\r\n")
			key, value, ok = strings.Cut(headerLine, ghttp.COLON)

			if ok {
				// 持續讀取 Header
				if _, ok := a.context.Response.Header[key]; !ok {
					a.context.Response.Header[key] = []string{}
				}
				value = strings.TrimLeft(value, " \t")
				a.context.Response.Header[key] = append(a.context.Response.Header[key], value)
				utils.Debug("Header, key: %s, value: %s", key, value)

			} else {
				// 當前這行數據不包含":"，結束 Header 的讀取
				utils.Debug("Empty line")

				// Header 中包含 Content-Length，狀態值設為 2，等待讀取後續數據
				if contentLength, ok := a.context.Response.Header["Content-Length"]; ok {
					length, err := strconv.Atoi(contentLength[0])
					utils.Debug("Content-Length: %d", length)

					if err != nil {
						utils.Error("Content-Length err: %+v", err)
						return
					}

					a.context.Response.ReadLength = int32(length)
					utils.Debug("a.httpConn.ReadLength: %d", a.context.Response.ReadLength)

					a.context.State = ghttp.READ_BODY
					utils.Debug("State: READ_HEADER -> READ_BODY")

				} else {
					// Header 中不包含 Content-Length，狀態值恢復為 0
					a.context.State = ghttp.READ_FIRST_LINE

					// 數據已讀入 currR2 當中，此處工作結構僅負責觸發 WorkHandler，進一步觸發 Callback 函式
					a.currWork.Index = a.currConn.GetId()
					a.currWork.RequestTime = time.Now().UTC()
					a.currWork.State = base.WORK_NEED_PROCESS
				}
				return
			}
		}
	}

	// 讀取 Body 數據
	if a.context.State == ghttp.READ_BODY {
		utils.Debug("State READ_BODY, a.httpConn.ReadLength: %d", a.context.Response.ReadLength)

		if a.currConn.CheckReadable(a.context.HasEnoughData) {
			// 將傳入的數據，加入工作緩存中
			a.currConn.Read(&a.readBuffer, a.context.Response.ReadLength)
			utils.Debug("State READ_BODY, data: %s", string(a.readBuffer[:a.context.Response.ReadLength]))

			a.context.Response.SetBody(a.readBuffer, a.context.Response.ReadLength)

			// 重置狀態值
			a.context.State = ghttp.READ_FIRST_LINE

			// 數據已讀入 httpConn 當中，此處工作結構僅負責觸發 WorkHandler，進一步觸發 Callback 函式
			a.currWork.Index = a.currConn.GetId()
			a.currWork.RequestTime = time.Now().UTC()
			a.currWork.State = base.WORK_NEED_PROCESS

			// 指向下一個工作結構
			a.currWork = a.currWork.Next
		}
	}
}

func (a *HttpAsker) write(id int32, data *[]byte, length int32) error {
	utils.Debug("work id: %d", id)

	// 取得連線物件(若 id 為 -1，表示尋找空閒的連線物件)
	a.currConn = a.getConn(id)

	// 目前沒有空閒的連線物件，等待下次迴圈再處理
	if a.currConn == nil {
		utils.Error("currConn is nil")
		return nil
	}

	if a.currConn.State == define.Unused {
		utils.Debug("currConn.State is Unused")
		a.currConn.State = define.Connecting

		// 設置當前工作結構對應的連線物件
		a.currWork.Index = a.currConn.GetId()
		utils.Debug("a.currWork.Index <- %d", a.currConn.GetId())

		a.Asker.Connect(a.currConn.GetId())
		return nil
	} else if a.currConn.State == define.Connecting {
		utils.Debug("currConn.State is Connecting")
		return nil
	}

	// 將數據寫入連線物件的緩存
	utils.Debug("WriteBuffer, length: %d, data: %+v", length, (*data)[:length])

	a.currConn.SetWriteBuffer(data, length)
	a.currWork.State = base.WORK_DONE
	return nil
}

// 原始數據寫出函式，缺乏定義 Callback 函式的能力，應使用 Send 來傳送請求
func (a *HttpAsker) Write(data *[]byte, length int32) error {
	// 取得空的工作結構
	w := a.getEmptyWork()
	// 標註此工作未指定寫出的連線物件，由空閒的連線物件來寫出
	w.Index = -1
	w.Body.AddRawData((*data)[:length])
	a.Handlers[w.GetId()] = func(c *ghttp.Context) {
		utils.Info("Response: %+v", c)
	}
	w.Send()
	return nil
}

// [Work State: 1] 由外部定義 workHandler，定義如何處理工作
func (a *HttpAsker) SetWorkHandler() {
	a.Asker.workHandler = func(w *base.Work) {
		if w.Index == -2 {
			return
		}
		utils.Debug("work: %+v", w)

		// 取得連線物件
		a.currConn = a.getConn(w.Index)

		// 根據 Conn 的 Id，存取對應的 httpConn
		a.context = a.contexts[a.currConn.GetId()]

		if handler, ok := a.Handlers[w.GetId()]; ok {
			// 將取得的 Response，透過註冊的 Callback 函釋回傳回去
			handler(a.context)
		}
		w.Finish()
	}
}

func (a *HttpAsker) NewRequest(method string, uri string, params map[string]string) *ghttp.Request {
	a.context = a.contextPool.Get().(*ghttp.Context)
	a.context.Request.FormRequest(method, uri, params)
	return a.context.Request
}

// 供外部傳送 Http 請求
func (a *HttpAsker) Send(req *ghttp.Request, callback func(*ghttp.Context)) error {
	utils.Debug("req: %+v", req)

	if callback == nil {
		return errors.New("callback 函式不可為 nil")
	}

	// 取得空的工作結構
	w := a.getEmptyWork()
	// 標註此工作未指定寫出的連線物件，由空閒的連線物件(Context id = -1)來寫出
	w.Index = -1
	w.Body.AddRawData(req.ToRequestData())
	a.Handlers[w.GetId()] = callback
	w.Send()
	utils.Debug("work: %+v", w)
	// 釋放 req *ghttp.Request
	req.Release()
	// 將 Request 放回物件池
	a.contextPool.Put(req)
	return nil
}
