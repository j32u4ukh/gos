package ask

import (
	"net"
	"time"

	"github.com/j32u4ukh/gos/base"
	"github.com/j32u4ukh/gos/define"

	"github.com/pkg/errors"
)

type Tcp0Asker struct {
	*Asker
	tcp0s    []*base.Tcp0
	currTcp0 *base.Tcp0
}

func NewTcp0Asker(site int32, laddr *net.TCPAddr, nConnect int32, nWork int32, onEvents map[define.EventType]func()) (IAsker, error) {
	var err error
	a := &Tcp0Asker{
		tcp0s: make([]*base.Tcp0, nConnect),
	}
	a.Asker, err = newAsker(site, laddr, nConnect, nWork, true)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to new Tcp0Asker.")
	}

	// 設置成功連線時的 callback
	a.Asker.onEvents = onEvents

	//
	a.currConn = a.conns

	// 設置連線的模式
	for a.currConn != nil {
		a.currConn.Mode = base.KEEPALIVE
		a.currConn = a.currConn.Next
	}

	var i int32

	for i = 0; i < nConnect; i++ {
		a.tcp0s[i] = base.NewTcp0()
	}

	//////////////////////////////////////////////////
	// Tcp0Asker 自定義函式
	//////////////////////////////////////////////////
	a.readFunc = a.read
	a.writeFunc = a.write
	return a, nil
}

func (a *Tcp0Asker) Connect() error {
	return a.Asker.Connect(-1)
}

func (a *Tcp0Asker) read() {
	a.currTcp0 = a.tcp0s[a.currConn.GetId()]

	if a.currConn.CheckReadable(a.currTcp0.ReadableChecker) {
		// 此時的 a.currConn.readLength 會是 a.currTcp0.HeaderSize(預設為 4)
		if a.currTcp0.State == 0 {
			// 從 readBuffer 當中讀取數據
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

			// fmt.Printf("(a *Asker) handler | 將傳入的數據，加入工作緩存中, Index: %d, state: %d\n", work.Index, work.state)
			a.currWork.Body.AddRawData(a.readBuffer[:a.currTcp0.ReadLength])
			a.currWork.Body.ResetIndex()

			// 指向下一個工作結構
			a.currWork = a.currWork.Next

			// 重置 欲讀取長度 以及 狀態值
			a.currTcp0.ResetReadLength()
		}
	}
}

// 內部寫出數據
func (a *Tcp0Asker) write(id int32, data *[]byte, length int32) error {
	a.Write(data, length)
	a.currWork.State = 0
	return nil
}

// 供外部寫出數據
func (a *Tcp0Asker) Write(data *[]byte, length int32) error {
	a.conns.SetWriteBuffer(data, length)
	return nil
}

// 由外部定義 workHandler，定義如何處理工作
func (a *Tcp0Asker) SetWorkHandler(handler func(*base.Work)) {
	a.Asker.workHandler = handler
}
