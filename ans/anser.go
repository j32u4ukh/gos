package ans

import (
	"encoding/binary"
	"fmt"
	"gos/base"
	"gos/define"
	"net"
	"time"

	"github.com/pkg/errors"
)

type IAnswer interface {
	// 開始監聽
	Listen()
	// 執行一次主迴圈
	Handler()
	// 數據寫出(寫到寫出緩存中)
	Write(int32, *[]byte, int32) error
}

func NewAnser(socketType define.SocketType, laddr *net.TCPAddr, nConnect int32, nWork int32) (IAnswer, error) {
	switch socketType {
	case define.Tcp0:
		return NewTcp0Anser(laddr, nConnect, nWork)
	case define.Http2:
		return NewHttpAnser2(laddr, nConnect, nWork)
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
	// 連線 ID
	index int32
	// 當前連線數
	nConn int32
	// 最大連線數
	maxConn int32
	// 指向第一個連線物件
	conns *base.Conn
	// 指向最後一個連線物件
	lastConn *base.Conn
	// 指向空的連線物件
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
	connBuffer chan net.Conn

	// ==================================================
	// 工作緩存
	// ==================================================
	works    *base.Work
	currWork *base.Work
	lastWork *base.Work

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

func newAnser(laddr *net.TCPAddr, nConnect int32, nWork int32) (*Anser, error) {
	listener, err := net.ListenTCP("tcp", laddr)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to listen at port %d.", laddr.Port)
	}

	a := &Anser{
		laddr:      laddr,
		listener:   listener,
		index:      0,
		nConn:      0,
		maxConn:    nConnect,
		conns:      base.NewConn(0, define.BUFFER_SIZE),
		readBuffer: make([]byte, 64*1024),
		order:      binary.LittleEndian,
		connBuffer: make(chan net.Conn, nWork),
		works:      base.NewWork(0),
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

// 監聽連線並註冊
func (a *Anser) Listen() {
	for {
		conn, err := a.listener.AcceptTCP()

		if err != nil {
			fmt.Printf("(a *Anser) | 接受客戶端連接異常: %+v\n", err.Error())
			continue
		}

		fmt.Printf("(a *Anser) | 客戶端連接來自: %s\n", conn.RemoteAddr())

		// 註冊連線通道
		a.connBuffer <- conn
	}
}

// 持續檢查是否有未完成的工作，若有，則呼叫外部定義的 workHandler 函式
func (a *Anser) Handler() {

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

		// Connecting, Disconnected, Timeout
		default:
			a.currConn = a.currConn.Next
		}
	}

	// 斷線處理: 釋放標註為 define.Disconnect 的連線物件，並確保有被使用的連線物件排在前面，而非有無使用的連線物件交錯排列
	a.disconnectHandler()

	// 根據 work.state 分別做不同處理，並重新整理工作結構的鏈結關係
	a.dealWork()
}

// 檢查是否有新的連線
func (a *Anser) checkConnection() {
	var netConn net.Conn
	for {
		select {
		case netConn = <-a.connBuffer:
			fmt.Printf("(a *Anser) checkConnection | Conn(%d)\n", a.index)
			a.emptyConn.Index = a.index
			a.emptyConn.NetConn = netConn
			a.emptyConn.State = define.Connected
			go a.emptyConn.Handler()

			// 更新空連線指標位置
			a.emptyConn = a.emptyConn.Next

			// 更新連線數與連線物件的索引值
			// TODO: a.nConnect == a.maxConnect, 檢查有沒有可以踢掉的連線
			a.nConn += 1
			a.index += 1
		default:
			return
		}
	}
}

// 連線處理
func (a *Anser) connectedHandler() {
	var packet *base.Packet
	var err error

	// TODO: 處理主動斷線
	select {
	// 封包事件
	case packet = <-a.currConn.ReadCh:

		// 封包讀取發生異常
		if packet.Error != nil {
			switch eType := packet.Error.(type) {
			case net.Error:
				if eType.Timeout() {
					fmt.Printf("(a *Anser) Handler | Conn %d 發生 timeout error.\n", a.currConn.Index)
				} else {
					fmt.Printf("(a *Anser) Handler | Conn %d 發生 net.Error.\n", a.currConn.Index)
				}
			default:
				fmt.Printf("(a *Anser) Handler | Conn %d 讀取 socket 時發生錯誤\nError: %+v\n", a.currConn.Index, packet.Error)
			}

			// 連線狀態設為結束
			a.currConn.State = define.Disconnect

			// 設定 3 秒後斷線
			a.currConn.SetDisconnectTime(3)

			// 指標指向下一個連線物件
			a.preConn = a.currConn
			a.currConn = a.currConn.Next
			return
		}

		// 將封包數據寫入 readBuffer
		a.currConn.SetReadBuffer(packet)

		// 更新斷線時間(NOTE: 若斷線時間與客戶端睡眠時間相同，會變成讀取錯誤，而非 timeout 錯誤，造成誤判)
		err = a.currConn.NetConn.SetReadDeadline(time.Now().Add(a.ReadTimeout))

		if err != nil {
			fmt.Printf("(a *Anser) handler | DeadlineError: %+v\n", err)

			// 連線狀態設為結束
			a.currConn.State = define.Disconnect

			// 設定 3 秒後斷線
			a.currConn.SetDisconnectTime(3)

			// 指標指向下一個連線物件
			a.preConn = a.currConn
			a.currConn = a.currConn.Next
			return
		}

	default:
		// 從緩存中讀取數據
		// a.read 根據補不同 SocketType，有不同的讀取數據函式實作
		a.readFunc()

		// 實際數據寫出，未因 SocketType 不同而有不同
		err = a.currConn.Write()

		if a.shouldCloseFunc(err) {
			// 連線狀態設為結束
			a.currConn.State = define.Disconnect

			// 設定 3 秒後斷線
			a.currConn.SetDisconnectTime(3)
		}

		// 指標指向下一個連線物件
		a.preConn = a.currConn
		a.currConn = a.currConn.Next
	}
}

// 斷線處理
func (a *Anser) disconnectHandler() {
	a.preConn = nil
	a.currConn = a.conns
	now := time.Now()

	for a.currConn != nil {
		// 標註為斷線的連線物件，數秒後才切斷連線，預留時間給對方讀取數據
		if a.currConn.State == define.Disconnect && a.currConn.DisconnectTime.Before(now) {
			fmt.Printf("(a *Anser) disconnectHandler | cid: %d\n", a.currConn.GetId())
			// hasDisconnect = true
			a.nConn -= 1

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
				a.currConn = a.preConn.Next
			}

		} else {
			a.preConn = a.currConn
			a.currConn = a.currConn.Next
		}
	}
}

// 尋找空閒的工作結構
func (a *Anser) getEmptyWork() *base.Work {
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
func (a *Anser) dealWork() {
	a.currWork = a.works
	var finished, yet *base.Work = nil, nil

	for a.currWork.State != -1 {
		// fmt.Printf("(a *Anser) dealWork | work: %+v\n", a.currWork)

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
			fmt.Printf("(a *Anser) dealWork | 連線 %d 發生異常工作 state(%d)，直接將工作結束\n", a.currWork.Index, a.currWork.State)

			// 將完成的工作加入 finished，並更新 work 所指向的工作結構
			finished = a.relinkWork(finished, true)
		}
	}

	// a.works = yet -> finished -> a.works
	if finished != nil {
		finished.Add(a.works)

		if yet != nil {
			yet.Add(finished)
			a.works = yet

		} else {
			a.works = finished

		}

	} else if yet != nil {
		yet.Add(a.works)
		a.works = yet

	}
}

func (a *Anser) Write(cid int32, data *[]byte, length int32) error {
	// fmt.Printf("(a *Anser) write | cid: %d\n", cid)
	c := a.getConn(cid)

	if c == nil {
		return errors.New(fmt.Sprintf("There is no cid equals to %d.", cid))
	}

	c.SetWriteBuffer(data, length)
	// a.currWork.State = 0
	return nil
}

func (a *Anser) getConn(cid int32) *base.Conn {
	c := a.conns

	for c != nil {
		if c.Index == cid {
			return c
		}
		c = c.Next
	}

	return nil
}

// 將處理後的 work 移到所屬分類的鏈式結構 destination 之下
func (a *Anser) relinkWork(destination *base.Work, done bool) *base.Work {
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

// 當前連線是否應斷線
func (a *Anser) shouldClose(err error) bool {
	if err != nil {
		fmt.Printf("(a *Anser) shouldClose | Conn(%d) failed to write: %+v\n", a.currConn.GetId(), err)
		return true
	}
	return false
}
