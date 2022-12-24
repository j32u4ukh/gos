package ans

import (
	"gos/base"
	"net"
	"time"

	"github.com/pkg/errors"
)

// ====================================================================================================
// Tcp0Anser
// ====================================================================================================

type Tcp0Anser struct {
	*Anser
	tcp0s    []*Tcp0
	currTcp0 *Tcp0
}

func NewTcp0Anser(laddr *net.TCPAddr, nConnect int32, nWork int32) (IAnswer, error) {
	var err error
	a := &Tcp0Anser{
		tcp0s:    make([]*Tcp0, nConnect),
		currTcp0: nil,
	}
	a.Anser, err = newAnser(laddr, nConnect, nWork)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to new Tcp0Anser.")
	}

	// 設置數據讀取函式
	a.Anser.read = a.Read

	var i int32

	for i = 0; i < nConnect; i++ {
		a.tcp0s[i] = NewTcp0()
	}

	return a, nil
}

// 監聽連線並註冊
func (a *Tcp0Anser) Listen() {
	a.Anser.Listen()
}

func (a *Tcp0Anser) Handler() {
	a.Anser.Handler()
}

func (a *Tcp0Anser) Read() bool {
	a.currTcp0 = a.tcp0s[a.currConn.GetId()]

	if a.currConn.CheckReadable(a.ReadableChecker) {
		if a.currTcp0.state == 0 {
			// 從 readBuffer 當中讀取封包長度
			a.currConn.Read(&a.readBuffer, 4)

			// 下次欲讀取長度為封包長度
			a.currTcp0.ReadLength = base.BytesToInt32(a.readBuffer[:4], a.order)

			// 更新 currTcp0 狀態值
			a.currTcp0.state = 1

		} else {
			// 將傳入的數據，加入工作緩存中
			a.currConn.Read(&a.readBuffer, a.currTcp0.ReadLength)

			// 考慮分包問題，收到完整一包數據傳完才傳到應用層
			a.currWork.Index = a.currConn.Index
			a.currWork.RequestTime = time.Now().UTC()
			a.currWork.State = 1
			a.currWork.Body.AddRawData(a.readBuffer[:a.currTcp0.ReadLength])
			a.currWork.Body.ResetIndex()

			// 指向下一個工作結構
			a.currWork = a.currWork.Next

			// 重置 封包長度
			a.currConn.PacketLength = -1

			// 重置 欲讀取長度 以及 狀態值
			a.currTcp0.resetReadLength()
		}

		// // 此時的 a.currConn.readLength 會是 4
		// if a.currConn.PacketLength == -1 {
		// 	// 從 readBuffer 當中讀取數據
		// 	a.currConn.Read(&a.readBuffer, 4)

		// 	// fmt.Printf("(a *Anser) handler | packetLength: %+v\n", a.readBuffer[:4])
		// 	a.currConn.PacketLength = base.BytesToInt32(a.readBuffer[:4], a.order)

		// 	// 下次欲讀取長度為封包長度
		// 	a.currConn.ReadLength = a.currConn.PacketLength
		// 	// fmt.Printf("readLength: %d, packetLength: %d\n", a.currConn.readLength, a.currConn.packetLength)
		// } else {
		// 	// 將傳入的數據，加入工作緩存中
		// 	a.currConn.Read(&a.readBuffer, a.currConn.ReadLength)

		// 	// 考慮分包問題，收到完整一包數據傳完才傳到應用層
		// 	a.currWork.Index = a.currConn.Index
		// 	a.currWork.RequestTime = time.Now().UTC()
		// 	a.currWork.State = 1
		// 	a.currWork.Body.AddRawData(a.readBuffer[:a.currConn.ReadLength])
		// 	a.currWork.Body.ResetIndex()

		// 	// 指向下一個工作結構
		// 	a.currWork = a.currWork.Next

		// 	// 重置 封包長度
		// 	a.currConn.PacketLength = -1

		// 	// 重置 欲讀取長度
		// 	a.currConn.ReadLength = define.DATALENGTH
		// }
	}
	return true
}

// 檢查是否滿足：可讀長度 大於 欲讀取長度
func (a *Tcp0Anser) ReadableChecker(buffer *[]byte, i int32, o int32, length int32) bool {
	return length >= a.currTcp0.ReadLength
}

func (a *Tcp0Anser) Write(cid int32, data *[]byte, length int32) error {
	return a.Anser.Write(cid, data, length)
}

// 由外部定義 workHandler，定義如何處理工作
func (a *Tcp0Anser) SetWorkHandler(handler func(*base.Work)) {
	a.Anser.workHandler = handler
}

// ====================================================================================================
// Tcp0
// ====================================================================================================
type Tcp0 struct {
	state      int8
	headerSize int32
	ReadLength int32
}

func NewTcp0() *Tcp0 {
	t := &Tcp0{
		state:      0,
		headerSize: 4,
	}
	t.ReadLength = t.headerSize
	return t
}

func (t *Tcp0) resetReadLength() {
	t.state = 0
	t.ReadLength = t.headerSize
}
