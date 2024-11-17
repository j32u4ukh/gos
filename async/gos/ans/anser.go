package ans

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/j32u4ukh/gos/utils/log"

	"github.com/j32u4ukh/gos/async/gos/base"
	"github.com/j32u4ukh/gos/utils"

	"github.com/pkg/errors"
)

type IAnser interface {
	// 開始監聽
	Listen()
	// 執行一次主迴圈
	Handler(conn *base.Conn)
	// 數據寫出(寫到寫出緩存中)
	Write(cid int32, data []byte, length int32) error
	// 主動中斷指定連線
	Disconnect(cid int32, duration time.Duration) error
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
	contextMap map[int32]base.IContext
	connPool   *sync.Pool
	connId     int32
	// 當前連線數
	nConn int32
	// 最大連線數
	maxConn int32
	connMu  sync.Mutex
	// ==================================================
	// 外部定義函式(由各 SocketType 實作)
	// ==================================================
	handlerFunc func(netConn net.Conn)
}

func NewAnser(port int32, nConnect int32) (*Anser, error) {
	laddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to resolve tcp addr :%d", port)
	}
	listener, err := net.ListenTCP("tcp", laddr)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to listen at port %d", port)
	}
	a := &Anser{
		laddr:       laddr,
		listener:    listener,
		contextMap:  make(map[int32]base.IContext),
		connId:      -1,
		nConn:       0,
		maxConn:     nConnect,
		ReadTimeout: 10 * time.Second,
	}
	a.connPool = &sync.Pool{
		New: func() any {
			a.connId++
			return base.NewConn(a.connId, utils.GosConfig.AnswerReadBuffer)
		},
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
			log.Error("接受客戶端連接異常: %+v", err.Error())
			continue
		}
		log.Info("客戶端連接來自: %s", conn.RemoteAddr())
		if a.nConn < a.maxConn {
			a.connMu.Lock()
			a.nConn++
			a.connMu.Unlock()
			go a.handlerFunc(conn)
		}
	}
}

func (a *Anser) GetConn(netConn net.Conn) *base.Conn {
	baseConn := a.connPool.Get().(*base.Conn)
	baseConn.NetConn = netConn
	return baseConn
}

func (a *Anser) PutConn(baseConn *base.Conn) {
	if baseConn == nil {
		return
	}
	baseConn.Release()
	a.connPool.Put(baseConn)
}

func (a *Anser) EndHandler(cid int32) {
	a.connMu.Lock()
	a.nConn--
	a.connMu.Unlock()
	if r := recover(); r != nil {
		log.Error("Error occurred while handling conn-%d, err: %+v\n", cid, r)
	}
}

func (a *Anser) Disconnect(cid int32, duration time.Duration) error {
	if ic, ok := a.contextMap[cid]; ok {
		if duration > 0 {
			time.Sleep(duration)
		}
		ic.GetConn().Stop()
		return nil
	}
	return fmt.Errorf("不存在 cid 為 %d 的連線", cid)
}
