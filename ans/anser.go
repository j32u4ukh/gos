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

type Anser struct {
	// 連線位置
	laddr *net.TCPAddr
	// 監聽連線物件
	listener *net.TCPListener
	// 通訊類型
	socketType define.SocketType
	// ==================================================
	// 連線列表
	// ==================================================
	// 連線 ID
	index int32
	// 當前連線數
	nConnect int32
	// 最大連線數
	maxConnect int32
	// 指向第一個連線物件
	connectors *base.Conn
	// 指向最後一個連線物件
	lastConnector *base.Conn
	// 指向空的連線物件
	emptyConnector *base.Conn
	// 指向當前連線物件
	currConnector *base.Conn
	// 指向前一個連線物件
	preConnector *base.Conn
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
	lastWork *base.Work

	// ==================================================
	// 外部定義函式
	// ==================================================
	// 工作處理函式
	workHandler func(*base.Work)
}

func NewAnser(laddr *net.TCPAddr, socketType define.SocketType, nConnect int32, nWork int32) (*Anser, error) {
	listener, err := net.ListenTCP("tcp", laddr)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to listen at port %d.", laddr.Port)
	}

	a := &Anser{
		laddr:      laddr,
		listener:   listener,
		socketType: socketType,
		index:      0,
		nConnect:   0,
		maxConnect: nConnect,
		connectors: base.NewConn(define.BUFFER_SIZE),
		readBuffer: make([]byte, 64*1024),
		order:      binary.LittleEndian,
		connBuffer: make(chan net.Conn, nWork),
		works:      base.NewWork(0),
	}

	var i int32
	var nextConnector *base.Conn
	var nextWork *base.Work
	a.emptyConnector = a.connectors
	a.lastConnector = a.connectors

	for i = 1; i < nConnect; i++ {
		nextConnector = base.NewConn(define.BUFFER_SIZE)
		a.lastConnector.Next = nextConnector
		a.lastConnector = nextConnector
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

// 註冊連線
// TODO: 以 MAP 管理，相同客戶端斷線重連可以使用同一 Conn 物件，將未讀完或未寫完的數據繼續寫出
func (a *Anser) register(netConn net.Conn) {
	fmt.Printf("(a *Anser) register | Connector(%d)\n", a.index)
	a.emptyConnector.Index = a.index
	a.emptyConnector.NetConn = netConn
	a.emptyConnector.State = define.Connected
	go a.emptyConnector.Handler()

	// 更新空連線指標位置
	a.emptyConnector = a.emptyConnector.Next

	// 更新連線數與連線物件的索引值
	// TODO: a.nConnect == a.maxConnect, 檢查有沒有可以踢掉的連線
	a.nConnect += 1
	a.index += 1
}

// 由外部定義 workHandler，定義如何處理工作
func (a *Anser) SetWorkHandler(handler func(*base.Work)) {
	a.workHandler = handler
}

// 持續檢查是否有未完成的工作，若有，則呼叫外部定義的 workHandler 函式
func (a *Anser) Handler() {
	var packet *base.Packet
	var err error
	var netConn net.Conn
	registerConn := true

	for registerConn {
		select {
		case netConn = <-a.connBuffer:
			a.register(netConn)
		default:
			registerConn = false
		}
	}

	a.preConnector = nil
	a.currConnector = a.connectors
	work := a.getEmptyWork()

	for a.currConnector != nil && a.currConnector.State != define.Unused {
		// TODO: 處理主動斷線
		select {
		// 封包事件
		case packet = <-a.currConnector.ReadCh:

			if packet.Error != nil {
				switch eType := packet.Error.(type) {
				case net.Error:
					if eType.Timeout() {
						fmt.Printf("(a *Anser) Handler | Connector %d 發生 timeout error.\n", a.currConnector.Index)
					} else {
						fmt.Printf("(a *Anser) Handler | Connector %d 發生 net.Error.\n", a.currConnector.Index)
					}
				default:
					fmt.Printf("(a *Anser) Handler | Connector %d 讀取 socket 時發生錯誤\nError: %+v\n", a.currConnector.Index, packet.Error)
				}

				// 結束連線
				a.releaseConnector()
				continue
			}

			// 將封包數據寫入 readBuffer
			a.currConnector.SetReadBuffer(packet)

			// 更新斷線時間(NOTE: 若斷線時間與客戶端睡眠時間相同，會變成讀取錯誤，而非 timeout 錯誤，造成誤判)
			err = a.currConnector.NetConn.SetReadDeadline(time.Now().Add(5000 * time.Millisecond))

			if err != nil {
				fmt.Printf("(a *Anser) handler | DeadlineError: %+v\n", err)

				// 結束連線
				a.releaseConnector()
				continue
			}

		// 從緩存中讀取數據
		default:
			// 可讀長度 大於 欲讀取長度
			// fmt.Printf("readableLength: %d, readLength: %d\n", a.currConnector.readableLength, a.currConnector.readLength)
			if a.currConnector.ReadableLength >= a.currConnector.ReadLength {
				// 此時的 a.currConnector.readLength 會是 4
				if a.currConnector.PacketLength == -1 {
					// 從 readBuffer 當中讀取數據
					a.currConnector.Read(&a.readBuffer, 4)

					// fmt.Printf("(a *Anser) handler | packetLength: %+v\n", a.readBuffer[:4])
					a.currConnector.PacketLength = base.BytesToInt32(a.readBuffer[:4], a.order)

					// 下次欲讀取長度為封包長度
					a.currConnector.ReadLength = a.currConnector.PacketLength
					// fmt.Printf("readLength: %d, packetLength: %d\n", a.currConnector.readLength, a.currConnector.packetLength)
				} else {
					// 將傳入的數據，加入工作緩存中
					a.currConnector.Read(&a.readBuffer, a.currConnector.ReadLength)

					// 考慮分包問題，收到完整一包數據傳完才傳到應用層
					work.Index = a.currConnector.Index
					work.RequestTime = time.Now().UTC()
					work.State = 1
					work.Body.AddRawData(a.readBuffer[:a.currConnector.ReadLength])
					work.Body.ResetIndex()

					// 指向下一個工作結構
					work = work.Next

					// 重置 封包長度
					a.currConnector.PacketLength = -1

					// 重置 欲讀取長度
					a.currConnector.ReadLength = define.DATALENGTH
				}
			}

			_, err = a.currConnector.Write()

			if err != nil {
				fmt.Printf("(a *Anser) handler | Failed to write: %+v\n", err)

				// 結束連線
				a.releaseConnector()
				continue
			}

			// 指標指向下一個連線物件
			a.preConnector = a.currConnector
			a.currConnector = a.currConnector.Next
		}
	}

	// 根據 work.state 分別做不同處理，並重新整理工作結構的鏈結關係
	a.dealWork()
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
	work := a.works
	var finished, yet *base.Work = nil, nil

	for work.State != -1 {
		// fmt.Printf("(a *Anser) dealWork | work.Index: %d\n", work.Index)

		switch work.State {
		// 工作已完成
		case 0:
			work, finished = a.relinkWork(work, finished, true)
		case 1:
			// 對工作進行處理
			a.workHandler(work)

			switch work.State {
			case 0:
				// 將完成的工作加入 finished，並更新 work 所指向的工作結構
				work, finished = a.relinkWork(work, finished, true)
			case 1:
				// 將工作接入待處理的區塊，下次回圈再行處理
				work, yet = a.relinkWork(work, yet, false)
			case 2:
				// 將向客戶端傳輸數據，寫入 writeBuffer
				a.Write(work.Index, &work.Data, work.Length)

				// 將完成的工作加入 finished，並更新 work 所指向的工作結構
				work, finished = a.relinkWork(work, finished, true)
			}
		case 2:
			// 將向客戶端傳輸數據，寫入 writeBuffer
			a.Write(work.Index, &work.Data, work.Length)

			// 將完成的工作加入 finished，並更新 work 所指向的工作結構
			work, finished = a.relinkWork(work, finished, true)
		default:
			fmt.Printf("(a *Anser) dealWork | 連線 %d 發生異常工作 state(%d)，直接將工作結束\n", work.Index, work.State)

			// 將完成的工作加入 finished，並更新 work 所指向的工作結構
			work, finished = a.relinkWork(work, finished, true)
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
	c := a.getConnector(cid)

	if c == nil {
		return errors.New(fmt.Sprintf("There is no cid equals to %d.", cid))
	}

	c.SetWriteBuffer(data, length)

	// if err != nil {
	// 	a.currConnector = c
	// 	a.releaseConnector()
	// 	return errors.Wrapf(err, "Failed to write to port: %d, conn(%d)", a.laddr.Port, cid)
	// }

	return nil
}

func (a *Anser) getConnector(cid int32) *base.Conn {
	c := a.connectors

	for c != nil {
		if c.Index == cid {
			return c
		}
		c = c.Next
	}

	return nil
}

// 將處理後的 work 移到所屬分類的鏈式結構 destination 之下
func (a *Anser) relinkWork(work *base.Work, destination *base.Work, done bool) (*base.Work, *base.Work) {
	// 更新 works 指標位置
	a.works = work.Next

	// 空做是否已完成
	if done {
		// 清空當前工作結構
		work.Release()
	} else {
		// 從原本的鏈式結構中移除
		work.Next = nil
	}

	if destination == nil {
		destination = work
	} else {
		destination.Add(work)
	}

	work = a.works
	return work, destination
}

func (a *Anser) releaseConnector() {
	fmt.Printf("(a *Anser) releaseConnector | 釋放連線資源 Connector(%d)", a.currConnector.Index)
	a.nConnect -= 1

	if a.preConnector == nil {
		// 更新連線物件起始位置
		a.connectors = a.currConnector.Next

		// 釋放連線物件
		a.currConnector.Release()

		// 將釋放後的 Connector 移到最後
		a.lastConnector.Next = a.currConnector

		// 更新指向最後一個連線物件的位置
		a.lastConnector = a.currConnector

		// 更新下次檢查的指標位置
		a.currConnector = a.connectors
	} else {
		// 更新鏈式指標所指向的對象
		a.preConnector.Next = a.currConnector.Next

		// 釋放連線物件
		a.currConnector.Release()

		// 將釋放後的 Connector 移到最後
		a.lastConnector.Next = a.currConnector

		// 更新指向最後一個連線物件的位置
		a.lastConnector = a.currConnector

		// 更新下次檢查的指標位置
		a.currConnector = a.currConnector.Next
	}
}
