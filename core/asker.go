package core

import (
	"fmt"
	"net"
	"sync/atomic"

	"github.com/j32u4ukh/gos/base"
	"github.com/j32u4ukh/gos/cntr"
	"github.com/j32u4ukh/gos/define"
	"github.com/j32u4ukh/gos/log"
	"github.com/pkg/errors"
)

type IAsker interface {
	// 開始連線
	Bind()
	Connect() (int32, error)
}

type Asker struct {
	*Core
	stopCh chan cntr.Void
	connCh chan *base.Conn
	// 管理各種連線事件觸發函式(例如: 連線、斷線、、、)
	onEvents base.OnEventsFunc
}

func NewAsker(ip string, port int32, nConnect int32) *Asker {
	a := &Asker{
		Core:     NewCore(nConnect),
		stopCh:   make(chan cntr.Void, 1),
		connCh:   make(chan *base.Conn, nConnect),
		onEvents: make(base.OnEventsFunc),
	}
	a.addr = &net.TCPAddr{IP: net.ParseIP(ip), Port: int(port), Zone: ""}
	return a
}

func (a *Asker) Init(introduction []byte, heartbeat []byte) error {
	// ===== 自我介紹數據 =====
	if introduction != nil {
		a.introductionData = make([]byte, len(introduction))
		copy(a.introductionData, introduction)
		log.Debug("introductionData: %+v", a.introductionData)
	}
	// ===== 心跳包數據 =====
	if heartbeat != nil {
		a.heartbeatLength = int32(len(heartbeat))
		a.heartbeatData = make([]byte, a.heartbeatLength)
		copy(a.heartbeatData, heartbeat)
		log.Debug("heartbeatData: %+v", a.heartbeatData)
	}
	return nil
}

func (a *Asker) Connect(cid int32) (int32, error) {
	if cid == -1 && atomic.LoadInt32(&a.nConn) >= a.MAX_CONN {
		return -1, fmt.Errorf("連線數量已達上限(%d)", a.MAX_CONN)
	}
	// 註冊連線通道
	netConn, err := net.DialTCP("tcp", nil, a.addr)
	if err != nil {
		log.Error("Failed to connect, err: %+v", err)
		return -1, errors.Wrapf(err, "Failed to connect to %s:%d.", a.addr.IP, a.addr.Port)
	}
	baseConn := a.GetConn(cid, netConn)
	a.connCh <- baseConn
	return baseConn.GetId(), nil
}

func (a *Asker) Bind() {
	var baseConn *base.Conn
	for {
		select {
		case baseConn = <-a.connCh:
			atomic.AddInt32(&a.nConn, 1)
			baseConn.SetState(define.Connected)
			cid := baseConn.GetId()
			go a.Handler(baseConn)
			if a.introductionData != nil {
				baseConn.Write(a.introductionData)
			}
			// 連線成功之 callback
			a.callEvent(define.OnConnected, cid)
		case <-a.stopCh:
			return
		}
	}
}

func (a *Asker) SetOnEventFunc(eventType define.EventType, onEventFunc func(data any)) {
	a.onEvents[eventType] = onEventFunc
}

func (a *Asker) callEvent(eventType define.EventType, data any) {
	if a.onEvents != nil {
		if event, ok := a.onEvents[eventType]; ok {
			event(data)
		}
	}
}

func (a *Asker) Write(cid int32, data []byte) error {
	if conn, ok := a.connMap[cid]; ok {
		conn.Write(data)
		return nil
	}
	return fmt.Errorf("Not found cid(%d)", cid)
}

func (a *Asker) Stop() {
	a.stopCh <- cntr.NULL
}
