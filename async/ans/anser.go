package ans

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/j32u4ukh/gos/base"
	"github.com/j32u4ukh/gos/define"
	"github.com/j32u4ukh/gos/utils"

	"github.com/pkg/errors"
)

type IAnswer interface {
	// 開始監聽
	Listen()
	// 執行一次主迴圈
	Handler(conn net.Conn)
	// 數據寫出(寫到寫出緩存中)
	Write(int32, *[]byte, int32) error
	//
	Disconnect(cid int32) error
}

func NewAnser(socketType define.SocketType, laddr *net.TCPAddr, nConnect int32) (IAnswer, error) {
	switch socketType {
	case define.Tcp0:
		return newAnser(laddr, nConnect)
	// case define.Http:
	// 	return NewHttpAnser(laddr, nConnect, nWork)
	default:
		return nil, fmt.Errorf("invalid socket type: %v", socketType)
	}
}

type Anser struct {
	// 連線位置
	laddr *net.TCPAddr
	// 監聽連線物件
	listener *net.TCPListener
	// 讀取超時
	ReadTimeout time.Duration
	// ==================================================
	// 連線列表
	// ==================================================
	// 當前連線數
	nConn int32
	// 最大連線數
	maxConn int32
	// 指向當前連線物件
	currConn *base.Conn
	// 數據讀取緩存
	readBuffer []byte
	// 位元組順序 (Byte Order)，即 位元組 的排列順序
	order binary.ByteOrder

	// ==================================================
	// 工作緩存
	// ==================================================
	currWork *base.Work

	// ==================================================
	// 外部定義函式(由各 SocketType 實作)
	// ==================================================
	// 工作處理函式
	workHandler func(*base.Work)

	// 數據讀取函式
	readFunc func() bool

	// 數據寫出
	writeFunc func(int32, *[]byte, int32) error

	// 當前連線是否應斷線
	shouldCloseFunc func(error) bool
}

func newAnser(laddr *net.TCPAddr, nConnect int32) (*Anser, error) {
	listener, err := net.ListenTCP("tcp", laddr)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to listen at port %d.", laddr.Port)
	}

	a := &Anser{
		laddr:      laddr,
		listener:   listener,
		nConn:      0,
		maxConn:    nConnect,
		readBuffer: make([]byte, utils.GosConfig.AnswerReadBuffer),
		order:      binary.LittleEndian,
		//
		ReadTimeout: 10 * time.Second,
	}
	return a, nil
}

// 監聽連線並註冊
func (a *Anser) Listen() {
	var conn *net.TCPConn
	var err error
	for {
		conn, err = a.listener.AcceptTCP()
		if err != nil {
			utils.Error("接受客戶端連接異常: %+v", err.Error())
			continue
		}
		utils.Info("客戶端連接來自: %s", conn.RemoteAddr())

		if a.nConn < a.maxConn {
			a.nConn++
			go a.Handler(conn)
		}
	}
}

// 持續檢查是否有未完成的工作，若有，則呼叫外部定義的 workHandler 函式
func (a *Anser) Handler(conn net.Conn) {
	var packet *base.Packet
	var err error
	// work := base.NewWork(0)
	a.currConn = base.NewConn(0, 10)
	a.currConn.NetConn = conn
	fmt.Printf("a.currConn.NetConn: %v\n", a.currConn.NetConn != nil)
	go a.currConn.Handler()

	for {
		// TODO: 處理主動斷線
		select {
		// 封包事件
		case packet = <-a.currConn.ReadCh:

			// 封包讀取發生異常
			if packet.Error != nil {
				switch eType := packet.Error.(type) {
				case net.Error:
					if eType.Timeout() {
						utils.Error("Conn %d 發生 timeout error.", a.currConn.GetId())
					} else {
						utils.Error("Conn %d 發生 net.Error.", a.currConn.GetId())
					}
				default:
					switch packet.Error {
					// 沒有數據可讀取，對方已關閉連線
					case io.EOF:
						utils.Warn("Conn %d 沒有數據可讀取，對方已關閉連線\nError(%v): %+v", a.currConn.GetId(), eType, packet.Error)
					default:
						utils.Error("Conn %d 讀取 socket 時發生錯誤, Error(%v): %+v", a.currConn.GetId(), eType, packet.Error)
					}
				}

				// 連線狀態設為結束
				a.currConn.State = define.Disconnect

				// 設定 N 秒後斷線
				a.currConn.SetDisconnectTime(utils.GosConfig.DisconnectTime)

				a.currConn = a.currConn.Next
				return
			}

			// 將封包數據寫入 readBuffer
			a.currConn.SetReadBuffer(packet)

			// 更新斷線時間(NOTE: 若斷線時間與客戶端睡眠時間相同，會變成讀取錯誤，而非 timeout 錯誤，造成誤判)
			err = a.currConn.NetConn.SetReadDeadline(time.Now().Add(a.ReadTimeout))

			if err != nil {
				utils.Error("DeadlineError: %+v", err)

				// 連線狀態設為結束
				a.currConn.State = define.Disconnect

				// 設定 3 秒後斷線
				a.currConn.SetDisconnectTime(utils.GosConfig.DisconnectTime)
				return
			}

		default:
			// 從緩存中讀取數據
			// a.read 根據不同 SocketType，有不同的讀取數據函式實作
			// a.readFunc()
			a.read()

			// 實際數據寫出，未因 SocketType 不同而有不同
			err = a.currConn.Write()
			if err != nil{
				fmt.Printf("Write error: %v\n", err)
				return
			}
			a.currConn.Release()
			return

			// if a.shouldCloseFunc(err) {
			// 	// 連線狀態設為結束
			// 	a.currConn.State = define.Disconnect

			// 	// 設定 3 秒後斷線
			// 	a.currConn.SetDisconnectTime(utils.GosConfig.DisconnectTime)
			// }
		}
	}

}

func (a *Anser) read() {
	a.currConn.Read(&a.readBuffer, 100)
	fmt.Printf("request: %s\n", string(a.readBuffer))

	response := []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nConnection: close\r\n%s: text/html\r\nContent-Length: 19\r\n\r\n<h1>Hola Mundo</h1>",
		"Content-Type"))
	a.currConn.SetWriteBuffer(&response, int32(len(response)))
}

func (a *Anser) Disconnect(cid int32) error {
	// c := a.getConn(cid)
	// if c == nil {
	// 	return errors.Errorf("Not found cid %d", cid)
	// }
	// c.Release()
	return nil
}

// // 尋找工作結構(若 widx 為 -1，返回空閒的工作結構)
// func (a *Anser) getWork(wid int32) *base.Work {
// 	work := a.works
// 	if wid == -1 {
// 		for work != nil {
// 			if work.State == base.WORK_FREE {
// 				return work
// 			}
// 			work = work.Next
// 		}
// 	} else {
// 		for work != nil {
// 			if work.GetId() == wid {
// 				return work
// 			}
// 			work = work.Next
// 		}
// 	}
// 	return nil
// }

// // 根據 work.state 對工作進行處理，並確保工作鏈式結構的最前端為須處理的工作，後面再接上空的工作結構
// func (a *Anser) dealWork() {
// 	var finished, yet *base.Work = nil, nil

// 	for a.currWork.State != base.WORK_FREE {
// 		switch a.currWork.State {
// 		// 工作已完成
// 		case base.WORK_DONE:
// 			finished = a.relinkWork(finished, true)
// 		case base.WORK_NEED_PROCESS:
// 			// 對工作進行處理
// 			a.workHandler(a.currWork)

// 			switch a.currWork.State {
// 			case base.WORK_DONE:
// 				// 將完成的工作加入 finished，並更新 work 所指向的工作結構
// 				finished = a.relinkWork(finished, true)
// 			case base.WORK_NEED_PROCESS:
// 				// 將工作接入待處理的區塊，下次回圈再行處理
// 				yet = a.relinkWork(yet, false)
// 			case base.WORK_OUTPUT:
// 				// 將向客戶端傳輸數據，寫入 writeBuffer
// 				a.writeFunc(a.currWork.Index, &a.currWork.Data, a.currWork.Length)

// 				// 將完成的工作加入 finished，並更新 work 所指向的工作結構
// 				finished = a.relinkWork(finished, true)
// 			}
// 		case base.WORK_OUTPUT:
// 			// 將向客戶端傳輸數據，寫入 writeBuffer
// 			a.writeFunc(a.currWork.Index, &a.currWork.Data, a.currWork.Length)

// 			// 將完成的工作加入 finished，並更新 work 所指向的工作結構
// 			finished = a.relinkWork(finished, true)
// 		default:
// 			utils.Error("連線 %d 發生異常工作 state(%s)，直接將工作結束", a.currWork.Index, a.currWork.State)

// 			// 將完成的工作加入 finished，並更新 work 所指向的工作結構
// 			finished = a.relinkWork(finished, true)
// 		}
// 	}
// }

func (a *Anser) Write(cid int32, data *[]byte, length int32) error {
	// c := a.getConn(cid)
	// if c == nil {
	// 	return errors.New(fmt.Sprintf("There is no cid equals to %d.", cid))
	// }
	// c.SetWriteBuffer(data, length)
	return nil
}

// func (a *Anser) getConn(cid int32) *base.Conn {
// 	c := a.conns
// 	if cid == -1 {
// 		for c != nil {
// 			if c.State == define.Unused {
// 				return c
// 			}
// 			c = c.Next
// 		}
// 	} else {
// 		for c != nil {
// 			if c.GetId() == cid {
// 				return c
// 			}
// 			c = c.Next
// 		}
// 	}
// 	return nil
// }

// // 更新空連線指標位置
// func (a *Anser) updateEmptyConn() {

// }

// // 將處理後的 work 移到所屬分類的鏈式結構 destination 之下
// func (a *Anser) relinkWork(destination *base.Work, done bool) *base.Work {
// 	return nil
// }

// // 當前連線是否應斷線
// func (a *Anser) shouldClose(err error) bool {
// 	if err != nil {
// 		utils.Error("Conn(%d) failed to write: %+v", a.currConn.GetId(), err)
// 		return true
// 	}
// 	return false
// }
