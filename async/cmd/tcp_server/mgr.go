package tcp_server

import (
	"github.com/j32u4ukh/gos/async/gos/ans"
	"github.com/j32u4ukh/gos/async/gos/gtcp"
	"github.com/j32u4ukh/gos/utils/log"
	"github.com/pkg/errors"
)

type Manager struct {
	anser *ans.TcpAnser
}

func NewManager() *Manager {
	return &Manager{}
}

func (m *Manager) InitAnser(port int32, nConnect int32) error {
	var err error
	m.anser, err = ans.NewTcpAnser(port, nConnect)
	if err != nil {
		return errors.Wrap(err, "建立 TcpAnser 伺服器時發生錯誤")
	}
	m.anser.SetWorkHandler(m.AnserHandler)
	return nil
}

func (m *Manager) AnserHandler(tcp *gtcp.TcpContext) error {
	cmd, err := tcp.Data.PopInt32()
	if err != nil {
		return errors.Wrap(err, "Failed to get cmd code from tcp")
	}
	log.Info("Cmd code: %d", cmd)
	switch cmd {
	case 0:
		err = m.handleSystemCommand(tcp)
		if err != nil {
			return errors.Wrap(err, "Failed to handle system command")
		}
	default:
	}
	return nil
}

func (m *Manager) handleSystemCommand(tcp *gtcp.TcpContext) error {
	service, err := tcp.Data.PopInt32()
	if err != nil {
		return errors.Wrap(err, "Failed to get service code from tcp")
	}
	log.Info("Service code: %d", service)
	return nil
}
