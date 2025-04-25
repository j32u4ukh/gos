package tcp_server

import (
	"fmt"
	"time"

	"github.com/j32u4ukh/gos/base"
	"github.com/j32u4ukh/gos/cntr"
	"github.com/j32u4ukh/gos/core"
	"github.com/j32u4ukh/gos/define"
	"github.com/j32u4ukh/gos/log"
	"github.com/j32u4ukh/gos/server/gtcp"

	"github.com/pkg/errors"
)

type Manager struct {
	anser             *core.Anser
	asker             *core.Asker
	contextMap        map[int32]*gtcp.TcpContext
	heartbeatResponse []byte
}

func NewManager() *Manager {
	return &Manager{
		contextMap: make(map[int32]*gtcp.TcpContext),
	}
}

func (m *Manager) InitAnser(port int32, nConnect int32) error {
	m.anser = core.NewAnser(port, nConnect)
	err := m.anser.Init()
	if err != nil {
		return errors.Wrap(err, "初始化 TcpAnser 伺服器時發生錯誤")
	}
	m.anser.SetHandlerFunc(m.AnserHandlerFunc)
	bd := cntr.NewBinaryData()
	err = bd.AddInt32(1)
	if err != nil {
		return errors.Wrap(err, "Failed to add cmd")
	}
	err = FormData(bd)
	if err != nil {
		return errors.Wrap(err, "Failed to form data")
	}
	m.heartbeatResponse = bd.GetData()
	return nil
}

func (m *Manager) AnserHandlerFunc(conn *base.Conn) error {
	var tcp *gtcp.TcpContext = m.getTcpContext(conn)
	// 將數據讀取到 TcpContext
	var err error = tcp.Read()
	if err != nil {
		return errors.Wrap(err, "TcpContext 讀取數據發生錯誤")
	}
	if !tcp.IsCompleted() {
		return nil
	}
	cmd, err := tcp.PopInt32()
	switch cmd {
	case 0:
		log.Info("Ping")
		conn.Write(m.heartbeatResponse)
	default:
		conn.Write([]byte("OK"))
	}
	return nil
}

func (m *Manager) InitAsker(port int32, nConnect int32) error {
	m.asker = core.NewAsker("127.0.0.1", port, nConnect)
	m.asker.SetHandlerFunc(m.AskerHandlerFunc)
	heartbeat := cntr.NewBinaryData()
	err := heartbeat.AddInt32(0)
	if err != nil {
		return errors.Wrap(err, "Failed to init asker server")
	}
	err = FormData(heartbeat)
	if err != nil {
		return errors.Wrap(err, "Failed to insert length of heartbeat")
	}
	err = m.asker.Init(nil, heartbeat.GetData())
	if err != nil {
		return errors.Wrap(err, "Failed to init asker server")
	}
	m.asker.SetOnEventFunc(define.OnConnected, func(data any) {
		cid := data.(int32)
		fmt.Printf("cid: %d connetced\n", cid)
	})
	m.asker.Connect(-1)
	return nil
}

func (m *Manager) AskerHandlerFunc(conn *base.Conn) error {
	var tcp *gtcp.TcpContext = m.getTcpContext(conn)
	// 將數據讀取到 TcpContext
	var err error = tcp.Read()
	if err != nil {
		return errors.Wrap(err, "TcpContext 讀取數據發生錯誤")
	}
	if !tcp.IsCompleted() {
		return nil
	}
	cmd, err := tcp.PopInt32()
	switch cmd {
	case 1:
		log.Info("(m *Manager) AskerHandlerFunc | Pong")
	default:
		conn.Write([]byte("OK"))
	}
	return nil
}

func (m *Manager) getTcpContext(conn *base.Conn) *gtcp.TcpContext {
	cid := conn.GetId()
	if _, ok := m.contextMap[cid]; !ok {
		m.contextMap[cid] = gtcp.NewTcpContext(conn)
	}
	return m.contextMap[cid]
}

func (m *Manager) Run(kind string) {
	var start time.Time
	var during time.Duration
	const frameTime time.Duration = 20 * time.Millisecond
	if kind == "ans" {
		go m.anser.Listen()
	} else {
		go m.asker.Bind()
	}
	for {
		start = time.Now()
		// TODO: Do something
		during = time.Since(start)
		if during < frameTime {
			time.Sleep(frameTime - during)
		}
	}
}
