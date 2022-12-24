package ans

import (
	"gos/base"
	"gos/define"
	"net"
	"time"

	"github.com/pkg/errors"
)

type Tcp0Anser struct {
	*Anser
}

func NewTcp0Anser(laddr *net.TCPAddr, nConnect int32, nWork int32) (IAnswer, error) {
	var err error
	a := &Tcp0Anser{}
	a.Anser, err = newAnser(laddr, nConnect, nWork)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to new Tcp0Anser.")
	}

	// 設置數據讀取函式
	a.Anser.read = a.Read

	return a, nil
}

// 監聽連線並註冊
func (a *Tcp0Anser) Listen() {
	a.Anser.Listen()
}

func (a *Tcp0Anser) Handler() {
	a.Anser.Handler()
}

func (a *Tcp0Anser) Read(work *base.Work) (bool, *base.Work) {
	// 可讀長度 大於 欲讀取長度
	// fmt.Printf("readableLength: %d, readLength: %d\n", a.currConn.readableLength, a.currConn.readLength)
	if a.currConn.ReadableLength >= a.currConn.ReadLength {
		// 此時的 a.currConn.readLength 會是 4
		if a.currConn.PacketLength == -1 {
			// 從 readBuffer 當中讀取數據
			a.currConn.Read(&a.readBuffer, 4)

			// fmt.Printf("(a *Anser) handler | packetLength: %+v\n", a.readBuffer[:4])
			a.currConn.PacketLength = base.BytesToInt32(a.readBuffer[:4], a.order)

			// 下次欲讀取長度為封包長度
			a.currConn.ReadLength = a.currConn.PacketLength
			// fmt.Printf("readLength: %d, packetLength: %d\n", a.currConn.readLength, a.currConn.packetLength)
		} else {
			// 將傳入的數據，加入工作緩存中
			a.currConn.Read(&a.readBuffer, a.currConn.ReadLength)

			// 考慮分包問題，收到完整一包數據傳完才傳到應用層
			work.Index = a.currConn.Index
			work.RequestTime = time.Now().UTC()
			work.State = 1
			work.Body.AddRawData(a.readBuffer[:a.currConn.ReadLength])
			work.Body.ResetIndex()

			// 指向下一個工作結構
			work = work.Next

			// 重置 封包長度
			a.currConn.PacketLength = -1

			// 重置 欲讀取長度
			a.currConn.ReadLength = define.DATALENGTH
		}
	}
	return true, work
}

func (a *Tcp0Anser) Write(cid int32, data *[]byte, length int32) error {
	return a.Anser.Write(cid, data, length)
}

// 由外部定義 workHandler，定義如何處理工作
func (a *Tcp0Anser) SetWorkHandler(handler func(*base.Work)) {
	a.Anser.workHandler = handler
}
