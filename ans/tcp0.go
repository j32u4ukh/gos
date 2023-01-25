package ans

import (
	"net"
	"time"

	"github.com/j32u4ukh/gos/base"

	"github.com/pkg/errors"
)

// ====================================================================================================
// Tcp0Anser
// ====================================================================================================

type Tcp0Anser struct {
	*Anser
	tcp0s    []*base.Tcp0
	currTcp0 *base.Tcp0
}

func NewTcp0Anser(laddr *net.TCPAddr, nConnect int32, nWork int32) (IAnswer, error) {
	var err error
	a := &Tcp0Anser{
		tcp0s:    make([]*base.Tcp0, nConnect),
		currTcp0: nil,
	}

	// ===== Anser =====
	a.Anser, err = newAnser(laddr, nConnect, nWork)
	a.Anser.ReadTimeout = 3000 * time.Millisecond

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to new Tcp0Anser.")
	}

	// ===== Tcp0 =====
	var i int32

	for i = 0; i < nConnect; i++ {
		a.tcp0s[i] = base.NewTcp0()
	}

	//////////////////////////////////////////////////
	// 自定義函式
	//////////////////////////////////////////////////
	// 設置數據讀取函式
	a.readFunc = a.read
	a.writeFunc = a.write
	a.shouldCloseFunc = a.shouldClose
	return a, nil
}

// 監聽連線並註冊
func (a *Tcp0Anser) Listen() {
	a.Anser.Listen()
}

func (a *Tcp0Anser) read() bool {
	a.currTcp0 = a.tcp0s[a.currConn.GetId()]

	if a.currConn.CheckReadable(a.currTcp0.ReadableChecker) {
		if a.currTcp0.State == 0 {
			// 從 readBuffer 當中讀取封包長度
			a.currConn.Read(&a.readBuffer, a.currTcp0.HeaderSize)

			// 下次欲讀取長度為封包長度
			a.currTcp0.ReadLength = base.BytesToInt32(a.readBuffer[:a.currTcp0.HeaderSize], a.order)

			// 更新 currTcp0 狀態值
			a.currTcp0.State = 1

		} else {
			// 將傳入的數據，加入工作緩存中
			a.currConn.Read(&a.readBuffer, a.currTcp0.ReadLength)

			// 考慮分包問題，收到完整一包數據傳完才傳到應用層
			a.currWork.Index = a.currConn.GetId()
			a.currWork.RequestTime = time.Now().UTC()
			a.currWork.State = 1
			a.currWork.Body.AddRawData(a.readBuffer[:a.currTcp0.ReadLength])
			a.currWork.Body.ResetIndex()

			// 指向下一個工作結構
			a.currWork = a.currWork.Next

			// // 重置 封包長度
			// a.currConn.PacketLength = -1

			// 重置 欲讀取長度 以及 狀態值
			a.currTcp0.ResetReadLength()
		}
	}
	return true
}

func (a *Tcp0Anser) write(cid int32, data *[]byte, length int32) error {
	return a.Write(cid, data, length)
}

// 由外部定義 workHandler，定義如何處理工作
func (a *Tcp0Anser) SetWorkHandler(handler func(*base.Work)) {
	a.Anser.workHandler = handler
}

// 當前連線是否應斷線
func (a *Tcp0Anser) shouldClose(err error) bool {
	// if a.Anser.shouldClose(err) {
	// 	return true
	// }
	return a.Anser.shouldClose(err)
}
