package ask

import (
	"gos/base"
	"net"
	"time"

	"github.com/pkg/errors"
)

type Tcp0Asker struct {
	*Asker
	tcp0 *base.Tcp0
}

func NewTcp0Asker(site int32, laddr *net.TCPAddr, nWork int32) (IAsker, error) {
	var err error
	a := &Tcp0Asker{
		tcp0: base.NewTcp0(),
	}
	a.Asker, err = newAsker(site, laddr, nWork)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to new Tcp0Asker.")
	}

	a.Asker.read = a.Read

	return a, nil
}

func (a *Tcp0Asker) Connect() error {
	return a.Asker.Connect()
}

func (a *Tcp0Asker) Handler() {
	a.Asker.Handler()
}

func (a *Tcp0Asker) GetAddress() (string, int32) {
	return a.Asker.GetAddress()
}

func (a *Tcp0Asker) Read() bool {
	if a.conn.CheckReadable(a.tcp0.ReadableChecker) {
		// 此時的 a.conn.readLength 會是 4
		if a.tcp0.State == 0 {
			// 從 readBuffer 當中讀取數據
			a.conn.Read(&a.readBuffer, 4)

			// 下次欲讀取長度為封包長度
			a.tcp0.ReadLength = base.BytesToInt32(a.readBuffer[:4], a.order)

			// 更新 currTcp0 狀態值
			a.tcp0.State = 1

		} else {
			// 將傳入的數據，加入工作緩存中
			a.conn.Read(&a.readBuffer, a.tcp0.ReadLength)

			// 考慮分包問題，收到完整一包數據傳完才傳到應用層
			a.currWork.Index = a.conn.Index
			a.currWork.RequestTime = time.Now().UTC()
			a.currWork.State = 1
			// fmt.Printf("(a *Asker) handler | 將傳入的數據，加入工作緩存中, Index: %d, state: %d\n", work.Index, work.state)
			a.currWork.Body.AddRawData(a.readBuffer[:a.tcp0.ReadLength])
			a.currWork.Body.ResetIndex()

			// 指向下一個工作結構
			a.currWork = a.currWork.Next

			// 重置 封包長度
			a.conn.PacketLength = -1

			// 重置 欲讀取長度 以及 狀態值
			a.tcp0.ResetReadLength()
		}

		return true
	}

	// // 可讀長度 大於 欲讀取長度
	// if a.conn.ReadableLength >= a.conn.ReadLength {
	// 	// 此時的 a.conn.readLength 會是 4
	// 	if a.conn.PacketLength == -1 {
	// 		// 從 readBuffer 當中讀取數據
	// 		a.conn.Read(&a.readBuffer, 4)
	// 		a.conn.PacketLength = base.BytesToInt32(a.readBuffer[:4], a.order)

	// 		// 下次欲讀取長度為封包長度
	// 		a.conn.ReadLength = a.conn.PacketLength
	// 		// fmt.Printf("readLength: %d, packetLength: %d\n", a.conn.readLength, a.conn.packetLength)
	// 	} else {
	// 		// 將傳入的數據，加入工作緩存中
	// 		a.conn.Read(&a.readBuffer, a.conn.ReadLength)

	// 		// 考慮分包問題，收到完整一包數據傳完才傳到應用層
	// 		a.currWork.Index = a.conn.Index
	// 		a.currWork.RequestTime = time.Now().UTC()
	// 		a.currWork.State = 1
	// 		// fmt.Printf("(a *Asker) handler | 將傳入的數據，加入工作緩存中, Index: %d, state: %d\n", work.Index, work.state)
	// 		a.currWork.Body.AddRawData(a.readBuffer[:a.conn.ReadLength])
	// 		a.currWork.Body.ResetIndex()

	// 		// 指向下一個工作結構
	// 		a.currWork = a.currWork.Next

	// 		// 重置 封包長度
	// 		a.conn.PacketLength = -1

	// 		// 重置 欲讀取長度
	// 		a.conn.ReadLength = define.DATALENGTH
	// 	}

	// 	return true
	// }

	return false
}

func (a *Tcp0Asker) Write(data *[]byte, length int32) error {
	return a.Asker.Write(data, length)
}

// 由外部定義 workHandler，定義如何處理工作
func (a *Tcp0Asker) SetWorkHandler(handler func(*base.Work)) {
	a.Asker.workHandler = handler
}
