package main

import (
	"github.com/j32u4ukh/gos"
	"github.com/j32u4ukh/gos/base"
)

const (
	SystemCmd     int32 = 0
	NormalCmd     int32 = 1
	CommissionCmd int32 = 2
)

// SystemCmd
const (
	ServerHeartbeatService int32 = 0
	ClientHeartbeatService int32 = 1
	IntroductionService    int32 = 2
)

// NormalCmd
const (
	RequestService  int32 = 0
	ResponseService int32 = 1
)

type Mgr struct{}

func NewMgr() *Mgr {
	mgr := &Mgr{}
	return mgr
}

func (m *Mgr) Handler(work *base.Work) {
	kind := work.Body.PopInt32()

	switch kind {
	case SystemCmd:
		m.HandlerKind0(work)
	default:
		data := work.Body.GetData()
		logger.Debug("data: %+v", data)

		// 標註當前工作已完成，將該工作結構回收
		work.Finish()
	}
}

func (m *Mgr) HandlerKind0(work *base.Work) {
	serivce := work.Body.PopInt32()
	switch serivce {
	case ServerHeartbeatService:
		logger.Debug("Heartbeat from client")
		work.Body.Clear()
		work.Body.AddInt32(SystemCmd)
		work.Body.AddInt32(ClientHeartbeatService)
		work.Body.AddString("OK")
		work.SendTransData()
	case ClientHeartbeatService:
		response := work.Body.PopString()
		work.Body.Clear()
		logger.Debug("Heartbeat from server, response: %s", response)
		work.Finish()
	case IntroductionService:
		tag := work.Body.PopString()
		if tag != "GOS" {
			err := gos.Disconnect(1023, work.Index)
			if err != nil {
				logger.Error("無效連線請求, 中斷當前連線失敗\nerr: %+v", err)
			} else {
				logger.Error("無效連線請求, 成功中斷當前連線")
			}
		} else {
			identity := work.Body.PopInt32()
			logger.Debug("identity: %d", identity)
		}
		work.Finish()
	default:
		data := work.Body.GetData()
		logger.Error("Undefined serivce %d, data: %+v", serivce, data)

		// 標註當前工作已完成，將該工作結構回收
		work.Finish()
	}
}
