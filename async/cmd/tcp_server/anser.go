package tcp_server

import (
	"github.com/j32u4ukh/gos/async/gos/gtcp"
	"github.com/j32u4ukh/gos/utils/log"
	"github.com/pkg/errors"
)

func (m *Manager) AnserHandler(tcp *gtcp.TcpContext) error {
	cmd, err := tcp.Data.PopInt32()
	if err != nil {
		return errors.Wrap(err, "Failed to get cmd code from tcp")
	}
	log.Info("Cmd code: %d", cmd)
	switch cmd {
	case 0:
		err = m.handleAnserSystemCommand(tcp)
		if err != nil {
			return errors.Wrap(err, "Failed to handle system command")
		}
	default:
	}
	return nil
}

func (m *Manager) handleAnserSystemCommand(tcp *gtcp.TcpContext) error {
	service, err := tcp.Data.PopInt32()
	if err != nil {
		return errors.Wrap(err, "Failed to get service code from tcp")
	}
	log.Info("Service code: %d", service)
	switch service {
	case 0:
		if err = tcp.Data.AddInt32(0); err != nil {
			return errors.Wrap(err, "Failed to get service code from tcp")
		}
		if err = tcp.Data.AddInt32(1); err != nil {
			return errors.Wrap(err, "Failed to get service code from tcp")
		}
	}
	return nil
}
