package ans

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/j32u4ukh/gos/define"
	"github.com/j32u4ukh/gos/utils/log"

	"github.com/j32u4ukh/gos/async/gos/base"

	"github.com/pkg/errors"
)

type IAnser interface {
	// 開始監聽
	Listen()
	// 執行一次主迴圈
	Handler(baseConn *base.Conn)
	// 主動中斷指定連線
	Disconnect(cid int32, duration time.Duration) error
}

type Anser struct {
	port int32
	// 連線位置
	addr *net.TCPAddr
	// 監聽連線物件
	listener *net.TCPListener
	// 讀取超時
	readTimeout time.Duration
	// ==================================================
	// 連線列表
	// ==================================================
	contextMap map[int32]base.IContext
	connPool   *sync.Pool
	connId     int32
	// 當前連線數
	nConn int32
	// 最大連線數
	maxConn           int32
	connMu            sync.Mutex
	contextBufferSize uint32
	connBufferSize    int32
	// ==================================================
	// 外部定義函式(由各 SocketType 實作)
	// ==================================================
	handlerFunc func(baseConn *base.Conn)
}

func NewAnser(port int32, nConnect int32) *Anser {
	a := &Anser{
		port:              port,
		contextMap:        make(map[int32]base.IContext),
		connId:            -1,
		nConn:             0,
		maxConn:           nConnect,
		readTimeout:       5 * time.Second,
		contextBufferSize: 64 * 1024,
		connBufferSize:    10,
	}
	return a
}

func (a *Anser) SetConnBufferSize(bufferSize int32) {
	a.connBufferSize = bufferSize
}

func (a *Anser) SetContextBufferSize(bufferSize uint32) {
	a.contextBufferSize = bufferSize
}

func (a *Anser) SetReadTimeout(readTimeout time.Duration) {
	a.readTimeout = readTimeout
}

func (a *Anser) Init() error {
	var err error
	a.addr, err = net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return errors.Wrapf(err, "Failed to resolve tcp addr :%d", a.port)
	}
	log.Info("addr: %+v", a.addr)
	a.listener, err = net.ListenTCP("tcp", a.addr)
	if err != nil {
		return errors.Wrapf(err, "Failed to listen at port %d", a.port)
	}
	log.Info("listener: %+v", a.listener)
	a.connPool = &sync.Pool{
		New: func() any {
			a.connId++
			baseConn := base.NewConn(a.connId, a.connBufferSize)
			baseConn.SetReadTimeout(a.readTimeout)
			return baseConn
		},
	}
	return nil
}

// 監聽連線並註冊
func (a *Anser) Listen() {
	var netConn net.Conn
	var baseConn *base.Conn
	var err error
	for {
		netConn, err = a.listener.AcceptTCP()
		if err != nil {
			log.Error("接受客戶端連接異常: %+v", err.Error())
			return
		}
		// if err != nil {
		// 	// 判斷是否因 Listener 關閉而中斷
		// 	if opErr, ok := err.(*net.OpError); ok && opErr.Op == "accept" {
		// 		log.Info("監聽已停止")
		// 		return
		// 	}
		// 	log.Error("接受客戶端連接異常: %+v", err)
		// 	continue
		// }
		log.Info("客戶端連接來自: %s", netConn.RemoteAddr())
		if a.nConn < a.maxConn {
			a.connMu.Lock()
			a.nConn++
			a.connMu.Unlock()
			baseConn = a.GetConn(netConn)
			baseConn.State = define.Connected
			go a.handlerFunc(baseConn)
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
	defer a.connMu.Unlock()
	a.nConn--
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
