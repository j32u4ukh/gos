package tcp_server

import (
	"github.com/j32u4ukh/gos/async/gos"
	"github.com/j32u4ukh/gos/async/gos/ans"
	"github.com/j32u4ukh/gos/async/gos/ask"
	"github.com/pkg/errors"
)

type Manager struct {
	anser *ans.TcpAnser
	asker *ask.TcpAsker
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) InitAnser(port int32, nConnect int32) error {
	m.anser = ans.NewTcpAnser(port, nConnect)
	err := m.anser.Init()
	if err != nil {
		return errors.Wrap(err, "初始化 TcpAnser 伺服器時發生錯誤")
	}
	m.anser.SetWorkHandler(m.AnserHandler)
	gos.RegisterAnser(port, m.anser)
	return nil
}

func (m *Manager) InitAsker(port int32, nConnect int32) error {
	m.asker = ask.NewTcpAsker("127.0.0.1", port, nConnect)
	err := m.asker.Init()
	if err != nil {
		return errors.Wrap(err, "初始化 TcpAsker 伺服器時發生錯誤")
	}
	m.asker.SetWorkHandler(m.AskerHandler)
	return nil
}

func (m *Manager) Run() {
	gos.Run(nil)
}
