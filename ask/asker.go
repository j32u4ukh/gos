package ask

import (
	"encoding/binary"
	"fmt"
	"gos/base"
	"gos/define"
	"net"
	"time"

	"github.com/pkg/errors"
)

type IAsker interface {
	Connect() error
	Handler()
	GetAddress() (string, int32)
	// 定義如何讀取(一次讀取多少；多少數據算一個完整的封包)
	Read()
	// 數據寫出(寫到寫出緩存中)
	Write(*[]byte, int32) error
	getConn(id int32) *base.Conn
}

func NewAsker(socketType define.SocketType, site int32, laddr *net.TCPAddr, nWork int32) (IAsker, error) {
	switch socketType {
	case define.Tcp0:
		return NewTcp0Asker(site, laddr, 1, nWork)
	// TODO: Chrome 一次最多可同時送出 6 個請求, HttpAsker nConnect = 6
	default:
		return nil, fmt.Errorf("invalid socket type: %v", socketType)
	}
}

type Asker struct {
	// 連線位置
	addr *net.TCPAddr
	// 是否需要心跳
	needHeartbeat bool
	// 心跳包數據
	heartData []byte
	// 心跳事件時間戳
	heartbeatTime time.Time
	// ==================================================
	// 連線列表
	// ==================================================
	// 連線 ID
	index int32
	// 最大連線數
	maxConn int32
	// 指向第一個連線物件
	conns *base.Conn
	// 指向最後一個連線物件
	lastConn *base.Conn
	// 指向空的連線物件(TODO: 移除)
	emptyConn *base.Conn
	// 指向當前連線物件
	currConn *base.Conn
	// 指向前一個連線物件
	preConn *base.Conn
	// 數據讀取緩存
	readBuffer []byte
	// 位元組順序 (Byte Order)，即 位元組 的排列順序
	order binary.ByteOrder

	// ==================================================
	// 連線緩存
	// ==================================================
	connBuffer chan base.ConnBuffer

	// ==================================================
	// 工作緩存
	// ==================================================
	works    *base.Work
	currWork *base.Work
	lastWork *base.Work

	// ==================================================
	// 外部定義函式
	// ==================================================
	// 工作處理函式
	workHandler func(*base.Work)
	readFunc    func()
	writeFunc   func(int32, *[]byte, int32) error
}

func newAsker(site int32, laddr *net.TCPAddr, nConnect int32, nWork int32, needHeartbeat bool) (*Asker, error) {
	a := &Asker{
		addr:          laddr,
		needHeartbeat: needHeartbeat,
		order:         binary.LittleEndian,
		index:         site,
		maxConn:       nConnect,
		conns:         base.NewConn(0, define.BUFFER_SIZE),
		readBuffer:    make([]byte, 64*1024),
		connBuffer:    make(chan base.ConnBuffer, nWork),
		works:         base.NewWork(0),
	}

	// TODO: 個別伺服器應自定義自己的心跳包數據
	if a.needHeartbeat {
		// 重複使用心跳包數據
		a.works.Body.AddByte(0)
		a.works.Body.AddUInt16(0)
		a.heartData = append(a.heartData, a.works.Body.FormData()...)
		a.works.Body.Clear()
	}

	var i int32
	var nextConn *base.Conn
	var nextWork *base.Work
	a.emptyConn = a.conns
	a.lastConn = a.conns

	for i = 1; i < nConnect; i++ {
		nextConn = base.NewConn(i, define.BUFFER_SIZE)
		a.lastConn.Next = nextConn
		a.lastConn = nextConn
	}

	a.lastWork = a.works

	for i = 1; i < nWork; i++ {
		nextWork = base.NewWork(i)
		a.lastWork.Next = nextWork
		a.lastWork = nextWork
	}

	return a, nil
}

func (a *Asker) GetAddress() (string, int32) {
	return a.addr.IP.String(), int32(a.addr.Port)
}

func (a *Asker) Connect(index int32) error {
	// 註冊連線通道
	netConn, err := net.DialTCP("tcp", nil, a.addr)
	if err != nil {
		return errors.Wrapf(err, "Failed to connect to %s:%d.", a.addr.IP, a.addr.Port)
	}
	// 註冊連線通道
	a.connBuffer <- base.ConnBuffer{Conn: netConn, Index: index}
	return nil
}

// TODO: 區分 1. 使用心跳機制維持連線的版本() 2.
// 根據 RFC 2616 (page 46) 的標準定義，單個客戶端不允許開啟 2 個以上的長連接，這個標準的目的是減少 HTTP 響應的時候，減少網絡堵塞。
func (a *Asker) Handler() {
	// 檢查是否有新的連線
	a.checkConnection()

	a.preConn = nil
	a.currConn = a.conns
	a.currWork = a.getEmptyWork()

	// 依序檢查有被使用的連線物件(State 不是 Unused)
	// 未使用 Unused, 嘗試連線中 Connecting, 連線中 Connected, 超時斷線 Timeout, 斷線 Disconnected, 重新連線中 Reconnect
	for a.currConn != nil && a.currConn.State != define.Unused {
		switch a.currConn.State {

		// 連線中
		case define.Connected:
			a.connectedHandler()

		// 超時斷線
		case define.Timeout:
			a.timeoutHandler()

		// 重新連線
		case define.Reconnect:
			a.reconnectHandler()

		// Connecting, Disconnected
		default:
			a.currConn = a.currConn.Next
		}
	}

	// 斷線處理: 釋放標註為 define.Disconnect 的連線物件，並確保有被使用的連線物件排在前面，而非有無使用的連線物件交錯排列
	a.disconnectHandler()

	// 工作處理
	a.dealWork()
}

// 檢查是否有新的連線
func (a *Asker) checkConnection() {
	var connBuffer base.ConnBuffer
	for {
		select {
		case connBuffer = <-a.connBuffer:
			// TODO: 檢查是否有空閒的連線物件可以使用
			a.emptyConn = a.getConn(connBuffer.Index)
			if a.emptyConn == nil {
				fmt.Printf("(a *Asker) checkConnection | Conn is nil\n")
				return
			}
			fmt.Printf("(a *Asker) checkConnection | Conn(%d)\n", a.emptyConn.GetId())
			if a.needHeartbeat {
				a.heartbeatTime = time.Now().Add(1 * time.Second)
			}
			a.emptyConn.NetConn = connBuffer.Conn
			a.emptyConn.State = define.Connected
			go a.emptyConn.Handler()
		default:
			return
		}
	}
}

// 連線處理
func (a *Asker) connectedHandler() {
	var packet *base.Packet
	var err error

	// TODO: 處理主動斷線
	select {
	// 封包事件
	case packet = <-a.currConn.ReadCh:
		fmt.Printf("(a *Asker) connectedHandler | Conn %d 封包事件\n", a.currConn.GetId())

		// 封包讀取發生異常
		if packet.Error != nil {
			switch eType := packet.Error.(type) {
			case net.Error:
				if eType.Timeout() {
					fmt.Printf("(a *Asker) connectedHandler | Conn %d 發生 timeout error.\n", a.currConn.GetId())
				} else {
					fmt.Printf("(a *Asker) connectedHandler | Conn %d 發生 net.Error.\n", a.currConn.GetId())
				}
			default:
				fmt.Printf("(a *Asker) connectedHandler | Conn %d 讀取 socket 時發生錯誤\nError: %+v\n", a.currConn.GetId(), packet.Error)
			}
			// 若需要維持連線
			if a.currConn.Mode == base.KEEPALIVE {
				// 重新連線
				a.currConn.State = define.Reconnect
			} else {
				// 連線狀態設為結束
				a.currConn.State = define.Disconnect
			}
			// 指標指向下一個連線物件
			a.preConn = a.currConn
			a.currConn = a.currConn.Next
			return
		}

		// 將封包數據寫入 readBuffer
		a.currConn.SetReadBuffer(packet)
		// 延後下次發送心跳包的時間
		a.heartbeatTime = time.Now().Add(5000 * time.Millisecond)
		// 更新連線維持時間
		err = a.currConn.NetConn.SetReadDeadline(a.heartbeatTime)
		if err != nil {
			fmt.Printf("(a *Asker) connectedHandler | 更新斷線時間 err: %+v\n", err)
			// 若需要維持連線
			if a.currConn.Mode == base.KEEPALIVE {
				// 重新連線
				a.currConn.State = define.Reconnect
			} else {
				// 連線狀態設為結束
				a.currConn.State = define.Disconnect
			}
			// 指標指向下一個連線物件
			a.preConn = a.currConn
			a.currConn = a.currConn.Next
			return
		}

	// 從緩存中讀取數據
	default:
		// 結束當前迴圈(若未進入下方兩個區塊)
		a.readFunc()

		_, err = a.currConn.Write()

		if err != nil {
			fmt.Printf("(a *Asker) connectedHandler | 更新斷線時間 err: %+v\n", err)
			// 若需要維持連線
			if a.currConn.Mode == base.KEEPALIVE {
				// 重新連線
				a.currConn.State = define.Reconnect
			} else {
				// 連線狀態設為結束
				a.currConn.State = define.Disconnect
			}
			// 指標指向下一個連線物件
			a.preConn = a.currConn
			a.currConn = a.currConn.Next
			return
		}

		if a.needHeartbeat {
			// 若當前時間已晚於發送心跳的時間戳
			if time.Now().After(a.heartbeatTime) {
				// 發送心跳包
				fmt.Printf("(a *Asker) connectedHandler | Heartbeat: %+v\n", a.heartbeatTime)

				// TODO: 每隔數分鐘再印一次資訊即可
				a.currWork.Index = 0
				a.currWork.RequestTime = time.Now().UTC()
				a.currWork.Body.AddRawData(a.heartData)
				a.currWork.Send()
				a.heartbeatTime = time.Now().Add(1000 * time.Millisecond)
			}
		}

		// 指標指向下一個連線物件
		a.preConn = a.currConn
		a.currConn = a.currConn.Next
	}
}

// 超時連線處理
func (a *Asker) timeoutHandler() {
	fmt.Printf("(a *Asker) timeoutHandler | Conn %d\n", a.currConn.GetId())
	if a.currConn.Mode == base.KEEPALIVE {
		a.currConn.State = define.Reconnect
	} else {
		a.currConn.State = define.Disconnect
	}
}

// 重新連線處理
func (a *Asker) reconnectHandler() {
	fmt.Printf("(a *Asker) reconnectHandler | Conn %d\n", a.currConn.GetId())

	// 嘗試連線中
	a.currConn.State = define.Connecting

	// 重新連線準備
	a.currConn.Reconnect()

	// 重新連線
	a.Connect(a.currConn.GetId())
}

// 尋找空閒的工作結構
func (a *Asker) getEmptyWork() *base.Work {
	work := a.works
	for work != nil {
		if work.State == -1 {
			return work
		}

		work = work.Next
	}
	return nil
}

// 根據 work.state 對工作進行處理，並確保工作鏈式結構的最前端為須處理的工作，後面再接上空的工作結構
func (a *Asker) dealWork() {
	a.currWork = a.works
	var finished, yet *base.Work = nil, nil

	for a.currWork.State != -1 {
		// fmt.Printf("(a *Asker) dealWork | work.Index: %d, state: %d\n", work.Index, work.state)

		switch a.currWork.State {
		// 工作已完成
		case 0:
			finished = a.relinkWork(finished, true)
		case 1:
			// 對工作進行處理
			a.workHandler(a.currWork)

			switch a.currWork.State {
			case 0:
				// 將完成的工作加入 finished，並更新 work 所指向的工作結構
				finished = a.relinkWork(finished, true)
			case 1:
				// 將工作接入待處理的區塊，下次回圈再行處理
				yet = a.relinkWork(yet, false)
			case 2:
				// 將向客戶端傳輸數據，寫入 writeBuffer
				a.writeFunc(a.currWork.Index, &a.currWork.Data, a.currWork.Length)

				// 將完成的工作加入 finished，並更新 work 所指向的工作結構
				finished = a.relinkWork(finished, true)
			}
		case 2:
			// 將向客戶端傳輸數據，寫入 writeBuffer
			a.writeFunc(a.currWork.Index, &a.currWork.Data, a.currWork.Length)

			// 將完成的工作加入 finished，並更新 work 所指向的工作結構
			finished = a.relinkWork(finished, true)
		default:
			fmt.Printf("(a *Asker) dealWork | 連線 %d 發生異常工作 state(%d)，直接將工作結束\n", a.currWork.Index, a.currWork.State)
			// 將完成的工作加入 finished，並更新 work 所指向的工作結構
			finished = a.relinkWork(finished, true)
		}
	}

	// a.works = yet -> finished -> a.works
	if finished != nil {
		finished.Add(a.works)
		// 若有尚未完成的工作
		if yet != nil {
			yet.Add(finished)
			a.works = yet
			// fmt.Printf("(a *Asker) dealWork | a.works = yet -> finished -> a.works\n")
		} else {
			a.works = finished
			// fmt.Printf("(a *Asker) dealWork | a.works = finished -> a.works\n")
		}
	} else if yet != nil {
		yet.Add(a.works)
		a.works = yet
		// fmt.Printf("(a *Asker) dealWork | a.works = yet -> a.works\n")
	}
}

// 將處理後的 work 移到所屬分類的鏈式結構 destination 之下
func (a *Asker) relinkWork(destination *base.Work, done bool) *base.Work {
	// 更新 works 指標位置
	a.works = a.currWork.Next

	// 空做是否已完成
	if done {
		// 清空當前工作結構
		a.currWork.Release()
	} else {
		// 從原本的鏈式結構中移除
		a.currWork.Next = nil
	}

	if destination == nil {
		destination = a.currWork
	} else {
		destination.Add(a.currWork)
	}

	a.currWork = a.works
	return destination
}

// // 將寫出數據加入緩存
// func (a *Asker) Write(data *[]byte, length int32) error {
// 	// fmt.Printf("(a *Anser) write | cid: %d\n", cid)
// 	c := a.getConn(cid)
// 	if c == nil {
// 		return errors.New(fmt.Sprintf("There is no cid equals to %d.", cid))
// 	}
// 	c.SetWriteBuffer(data, length)
// 	return nil
// }

// 取得連線物件編號為 id 的連線物件
// id 若為 -1，尋找當前空閒的連線物件
func (a *Asker) getConn(id int32) *base.Conn {
	fmt.Printf("(a *Asker) getConn | id: %d\n", id)
	c := a.conns
	if id == -1 {
		for c != nil {
			if c.State == define.Unused {
				return c
			}
			c = c.Next
		}
	} else {
		for c != nil {
			fmt.Printf("(a *Asker) getConn | GetId: %d\n", c.GetId())
			if c.GetId() == id {
				return c
			}
			c = c.Next
		}
	}
	return nil
}

// 斷線處理
func (a *Asker) disconnectHandler() {
	a.preConn = nil
	a.currConn = a.conns

	for a.currConn != nil {
		if a.currConn.State == define.Disconnect {
			fmt.Printf("(a *Asker) disconnectHandler | cid: %d\n", a.currConn.GetId())

			if a.preConn == nil {
				// 更新連線物件起始位置
				a.conns = a.currConn.Next

				// 釋放連線物件
				a.currConn.Release()

				// 將釋放後的 Conn 移到最後
				a.lastConn.Next = a.currConn

				// 更新指向最後一個連線物件的位置
				a.lastConn = a.currConn

				// 更新下次檢查的指標位置
				a.currConn = a.conns
			} else {
				// 更新鏈式指標所指向的對象
				a.preConn.Next = a.currConn.Next

				// 釋放連線物件
				a.currConn.Release()

				// 將釋放後的 Conn 移到最後
				a.lastConn.Next = a.currConn

				// 更新指向最後一個連線物件的位置
				a.lastConn = a.currConn

				// 更新下次檢查的指標位置
				a.currConn = a.currConn.Next
			}
		} else {
			a.preConn = a.currConn
			a.currConn = a.currConn.Next
		}
	}
}
