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

type Asker struct {
	// 連線編號
	site int32
	// 連線物件
	conn *base.Conn
	// 連線位置
	laddr *net.TCPAddr
	// 通訊類型
	socketType define.SocketType
	// 每幀時間
	frameTime time.Duration
	// 數據讀取緩存
	readBuffer []byte
	// 心跳包數據
	heartData []byte
	// 心跳事件通道
	heartbeat    *time.Ticker
	beatInterval time.Duration
	// 位元組順序 (Byte Order)，即 位元組 的排列順序
	order binary.ByteOrder

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

func NewAsker(site int32, laddr *net.TCPAddr, socketType define.SocketType, nWork int32) (*Asker, error) {
	a := &Asker{
		site:         site,
		conn:         base.NewConn(0, define.BUFFER_SIZE),
		laddr:        laddr,
		socketType:   socketType,
		frameTime:    200 * time.Millisecond,
		readBuffer:   make([]byte, 64*1024),
		beatInterval: 1000 * time.Millisecond,
		order:        binary.LittleEndian,
		works:        base.NewWork(0),
	}

	a.conn.Index = 0

	// 重複使用心跳包數據
	a.works.Body.AddByte(0)
	a.works.Body.AddUInt16(0)
	a.heartData = append(a.heartData, a.works.Body.FormData()...)
	a.works.Body.Clear()

	var i int32
	var nextWork *base.Work
	a.lastWork = a.works

	for i = 1; i < nWork; i++ {
		nextWork = base.NewWork(i)
		a.lastWork.Next = nextWork
		a.lastWork = nextWork
	}

	return a, nil
}

func (a *Asker) Connect() error {
	netConn, err := net.DialTCP("tcp", nil, a.laddr)

	if err != nil {
		return errors.Wrapf(err, "Failed to connect to %s:%d.", a.laddr.IP, a.laddr.Port)
	}

	// 連線前準備
	a.conn.PrepareBeforeConnect()

	// 更新連線物件與狀態
	a.conn.Index = 0
	a.conn.NetConn = netConn
	a.conn.State = define.Connected
	a.heartbeat = time.NewTicker(time.Second * 1)
	a.beatInterval = 1000 * time.Millisecond
	go a.conn.Handler()
	return nil
}

func (a *Asker) GetAddress() (string, int32) {
	return a.laddr.IP.String(), int32(a.laddr.Port)
}

// 由外部定義 workHandler，定義如何處理工作
func (a *Asker) SetWorkHandler(handler func(*base.Work)) {
	a.workHandler = handler
}

func (a *Asker) Handler() {
	var packet *base.Packet
	var err error
	var nWrite int32
	keepHandling := true
	work := a.getEmptyWork()

	for a.conn.State == define.Connected && keepHandling {
		select {
		// 發送心跳包
		case <-a.heartbeat.C:
			a.beatInterval -= 1000 * time.Millisecond
			// fmt.Printf("(a *Asker) handler | Heartbeat, a.beatInterval: %v\n", a.beatInterval)

			if a.beatInterval <= 0 {
				a.beatInterval = 1000 * time.Millisecond

				// TODO: 每隔數分鐘再印一次資訊即可
				work.Index = 999
				work.RequestTime = time.Now().UTC()
				work.Body.AddRawData(a.heartData)
				work.Send()

				// 指向下一個工作結構
				work = work.Next

				// 結束當前迴圈
				keepHandling = false
				continue
			}

		// 封包事件
		case packet = <-a.conn.ReadCh:

			if packet.Error != nil {
				switch eType := packet.Error.(type) {
				case net.Error:
					if eType.Timeout() {
						fmt.Printf("(a *Asker) handler | Asker %d 發生 timeout error.\n", a.site)
					} else {
						fmt.Printf("(a *Asker) handler | Asker %d 發生 net.Error.\n", a.site)
					}
				default:
					fmt.Printf("(a *Asker) handler | Asker %d 讀取 socket 時發生錯誤\nError: %+v\n", a.site, packet.Error)
				}

				// 修改連線狀態
				a.conn.State = define.Reconnecting

				// 結束當前迴圈
				keepHandling = false
				continue
			}

			// 將封包數據寫入 readBuffer
			a.conn.SetReadBuffer(packet)

			err = a.conn.NetConn.SetReadDeadline(time.Now().Add(5000 * time.Millisecond))
			a.beatInterval = 5000 * time.Millisecond

			if err != nil {
				fmt.Printf("(a *Asker) handler | 更新斷線時間 err: %+v\n", err)

				// 修改連線狀態
				a.conn.State = define.Reconnecting

				// 結束當前迴圈
				keepHandling = false
			}

		// 從緩存中讀取數據
		default:
			// 結束當前迴圈(若未進入下方兩個區塊)
			keepHandling = false

			// 可讀長度 大於 欲讀取長度
			if a.conn.ReadableLength >= a.conn.ReadLength {
				keepHandling = true

				// 此時的 a.conn.readLength 會是 4
				if a.conn.PacketLength == -1 {
					// 從 readBuffer 當中讀取數據
					a.conn.Read(&a.readBuffer, 4)
					a.conn.PacketLength = base.BytesToInt32(a.readBuffer[:4], a.order)

					// 下次欲讀取長度為封包長度
					a.conn.ReadLength = a.conn.PacketLength
					// fmt.Printf("readLength: %d, packetLength: %d\n", a.conn.readLength, a.conn.packetLength)
				} else {
					// 將傳入的數據，加入工作緩存中
					a.conn.Read(&a.readBuffer, a.conn.ReadLength)

					// 考慮分包問題，收到完整一包數據傳完才傳到應用層
					work.Index = a.conn.Index
					work.RequestTime = time.Now().UTC()
					work.State = 1
					// fmt.Printf("(a *Asker) handler | 將傳入的數據，加入工作緩存中, Index: %d, state: %d\n", work.Index, work.state)
					work.Body.AddRawData(a.readBuffer[:a.conn.ReadLength])
					work.Body.ResetIndex()

					// 指向下一個工作結構
					work = work.Next

					// 重置 封包長度
					a.conn.PacketLength = -1

					// 重置 欲讀取長度
					a.conn.ReadLength = define.DATALENGTH
				}
			}

			nWrite, err = a.conn.Write()

			if err != nil {
				// 修改連線狀態
				a.conn.State = define.Reconnecting

				// 結束當前迴圈
				keepHandling = false
			}

			if nWrite > 0 {
				keepHandling = true
			}
		}
	}

	switch a.conn.State {
	// 已連線
	case define.Connected:
		a.dealWork()

	// 重新連線
	case define.Reconnecting:
		fmt.Printf("(a *Asker) handler | Try to reconnect to server.\n")

		// 釋放當前連線
		a.conn.Release()

		time.Sleep(1 * time.Second)

		// 重新連線
		a.Connect()

	// 結束連線
	case define.Disconnected:
		if a.conn.NetConn != nil {
			// 釋放當前連線
			a.conn.Release()
		}
	}
}

func (a *Asker) checkWorks() {
	work := a.works
	for work != nil {
		fmt.Printf("(a *Asker) checkWorks | %s\n", work)
		work = work.Next
	}
	fmt.Println()
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
	work := a.works
	var finished, yet *base.Work = nil, nil

	for work.State != -1 {
		// fmt.Printf("(a *Asker) dealWork | work.Index: %d, state: %d\n", work.Index, work.state)

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
				a.Write(&work.Data, work.Length)

				// 將完成的工作加入 finished，並更新 work 所指向的工作結構
				work, finished = a.relinkWork(work, finished, true)
			}
		case 2:
			// 將向客戶端傳輸數據，寫入 writeBuffer
			a.Write(&work.Data, work.Length)

			// 將完成的工作加入 finished，並更新 work 所指向的工作結構
			work, finished = a.relinkWork(work, finished, true)
		default:
			fmt.Printf("(a *Asker) dealWork | 連線 %d 發生異常工作 state(%d)，直接將工作結束\n", work.Index, work.State)
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

// 將寫出數據加入緩存
func (a *Asker) Write(data *[]byte, length int32) error {
	// fmt.Printf("(a *Asker) write | length: %d\n", length)
	a.conn.SetWriteBuffer(data, length)

	// if err != nil {
	// 	// 修改連線狀態
	// 	a.conn.state = define.Reconnecting
	// 	return errors.Wrapf(err, "Failed to write to site(%d)", a.site)
	// }
	return nil
}

// 將處理後的 work 移到所屬分類的鏈式結構 destination 之下
func (a *Asker) relinkWork(work *base.Work, destination *base.Work, done bool) (*base.Work, *base.Work) {
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
