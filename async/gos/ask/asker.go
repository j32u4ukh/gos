package ask

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/j32u4ukh/cntr"
	"github.com/j32u4ukh/gos/async/gos/base"
	"github.com/j32u4ukh/gos/define"
	"github.com/j32u4ukh/gos/utils/log"
	"github.com/pkg/errors"
)

type IAsker interface {
	Bind()
	// 開始連線
	Connect() (int32, error)
	//
	Handler(baseConn *base.Conn)
	// 供外部寫出數據(寫到寫出緩存中)
	Write(ic base.IContext) error
}

type Asker struct {
	// 連線位置
	addr *net.TCPAddr
	// 讀取超時
	readTimeout time.Duration
	stopCh      chan cntr.Void
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
	connCh            chan *base.Conn
	contextBufferSize uint32
	connBufferSize    int32
	// ==================================================
	// 外部定義函式(由各 SocketType 實作)
	// ==================================================
	handlerFunc func(baseConn *base.Conn)
	// 管理各種連線事件觸發函式(例如: 連線、斷線、、、)
	onEvents base.OnEventsFunc
}

func NewAsker(ip string, port int, nConnect int32) *Asker {
	addr := &net.TCPAddr{IP: net.ParseIP(ip), Port: port, Zone: ""}
	a := &Asker{
		addr:              addr,
		readTimeout:       10 * time.Second,
		stopCh:            make(chan cntr.Void, 1),
		contextMap:        make(map[int32]base.IContext),
		connId:            -1,
		nConn:             0,
		maxConn:           nConnect,
		connCh:            make(chan *base.Conn, 1),
		contextBufferSize: 64 * 1024,
		connBufferSize:    10,
	}
	return a
}

func (a *Asker) SetConnBufferSize(bufferSize int32) {
	a.connBufferSize = bufferSize
}

func (a *Asker) SetContextBufferSize(bufferSize uint32) {
	a.contextBufferSize = bufferSize
}

func (a *Asker) SetReadTimeout(readTimeout time.Duration) {
	a.readTimeout = readTimeout
}

func (a *Asker) Init() error {
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

func (a *Asker) Connect() (int32, error) {
	// 註冊連線通道
	netConn, err := net.DialTCP("tcp", nil, a.addr)
	if err != nil {
		log.Error("Failed to connect, err: %+v", err)
		return -1, errors.Wrapf(err, "Failed to connect to %s:%d.", a.addr.IP, a.addr.Port)
	}
	if a.nConn >= a.maxConn {
		return -1, fmt.Errorf("連線數量已達上限(%d)", a.maxConn)
	}
	baseConn := a.GetConn(netConn)
	a.connCh <- baseConn
	return baseConn.GetId(), nil
}

func (a *Asker) Bind() {
	var baseConn *base.Conn
	for {
		select {
		case baseConn = <-a.connCh:
			a.connMu.Lock()
			a.nConn++
			a.connMu.Unlock()
			baseConn.State = define.Connected
			go a.handlerFunc(baseConn)
			// 連線成功之 callback
			a.callEvent(define.OnConnected, nil)
		case <-a.stopCh:
			return
		}
	}
}

func (a *Asker) GetConn(netConn net.Conn) *base.Conn {
	baseConn := a.connPool.Get().(*base.Conn)
	baseConn.NetConn = netConn
	return baseConn
}

func (a *Asker) PutConn(baseConn *base.Conn) {
	if baseConn == nil {
		return
	}
	baseConn.Release()
	a.connPool.Put(baseConn)
}

func (a *Asker) EndHandler(cid int32) {
	a.connMu.Lock()
	defer a.connMu.Unlock()
	a.nConn--
	if r := recover(); r != nil {
		log.Error("Error occurred while handling conn-%d, err: %+v\n", cid, r)
	}
}

func (a *Asker) callEvent(eventType define.EventType, data any) {
	if a.onEvents != nil {
		if event, ok := a.onEvents[eventType]; ok {
			event(data)
		}
	}
}

func (a *Asker) Stop() {
	a.stopCh <- cntr.NULL
}
