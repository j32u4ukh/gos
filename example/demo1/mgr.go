package main

import (
	"fmt"

	"github.com/j32u4ukh/gos"
	"github.com/j32u4ukh/gos/base"
)

type Mgr struct {
	Body *base.TransData
}

func NewMgr() *Mgr {
	mgr := &Mgr{
		Body: base.NewTransData(),
	}
	return mgr
}

func (m *Mgr) Handler(work *base.Work) {
	kind := work.Body.PopInt32()

	switch kind {
	case SystemCmd:
		m.HandlerKind0(work)
	case 1:
		m.HandlerKind1(work)
	case 2:
		m.HandlerKind2(work)
	default:
		data := work.Body.GetData()
		fmt.Printf("Undefined kind %d, data: %+v\n", kind, data)

		// 標註當前工作已完成，將該工作結構回收
		work.Finish()
	}
}

func (m *Mgr) HandlerKind0(work *base.Work) {
	serivce := work.Body.PopInt32()
	switch serivce {
	case HeartbeatService:
		logger.Debug("Heartbeat")
		work.Body.Clear()
		work.Body.AddInt32(SystemCmd)
		work.Body.AddInt32(HeartbeatService)
		work.Body.AddString("OK")
		work.SendTransData()
	case IntroductionService:
		tag := work.Body.PopString()
		if tag != "GOS" {
			err := gos.Disconnect(1023, work.Index)
			if err != nil {
				fmt.Printf("Kind0, IntroductionService | 無效連線請求, 中斷當前連線失敗\nerr: %+v\n", err)
			} else {
				fmt.Printf("Kind0, IntroductionService | 無效連線請求, 成功中斷當前連線\n")
			}
		} else {
			identity := work.Body.PopInt32()
			fmt.Printf("Kind0, IntroductionService | identity: %d\n", identity)
		}
		work.Finish()
	default:
		data := work.Body.GetData()
		fmt.Printf("Kind0, undefined serivce %d, data: %+v\n", serivce, data)

		// 標註當前工作已完成，將該工作結構回收
		work.Finish()
	}
}

func (m *Mgr) HandlerKind1(work *base.Work) {
	serivce := work.Body.PopInt32()
	switch serivce {
	case TimerService:
		data := work.Body.GetData()
		logger.Debug("data from asker: %+v", data)
		work.Body.Clear()
		work.Body.AddInt32(CommissionCmd)
		work.Body.AddUInt16(32)
		work.Body.AddString(fmt.Sprintf("Message from (m *Mgr) Handler(work *gos.Work), #data: %d", len(data)))
		work.SendTransData()
		logger.Debug("SendTransData back, work: %+v", work)
	default:
		data := work.Body.GetData()
		fmt.Printf("Kind1, undefined serivce %d, data: %+v\n", serivce, data)

		// 標註當前工作已完成，將該工作結構回收
		work.Finish()
	}
}

func (m *Mgr) HandlerKind2(work *base.Work) {
	serivce := work.Body.PopUInt16()
	switch serivce {
	case 32:
		response := work.Body.PopString()
		logger.Debug("response: %s", response)

		// 標註當前工作已完成，將該工作結構回收
		work.Finish()
	default:
		data := work.Body.GetData()
		fmt.Printf("Kind2, undefined serivce %d, data: %+v\n", serivce, data)

		// 標註當前工作已完成，將該工作結構回收
		work.Finish()
	}
}
