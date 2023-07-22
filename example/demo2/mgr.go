package main

import (
	"fmt"
	"time"

	"github.com/j32u4ukh/gos"
	"github.com/j32u4ukh/gos/base"
)

type Mgr struct {
	Body       *base.TransData
	nextSecond time.Duration
	cumTime    time.Duration
	FrameTime  time.Duration
	task       map[int32]bool
	data       []byte
	temp       []byte
}

func NewMgr() *Mgr {
	mgr := &Mgr{
		Body:       base.NewTransData(),
		nextSecond: 0,
		cumTime:    0,
		task:       map[int32]bool{},
		temp:       []byte{},
	}

	var i int32

	for i = 50; i < 60; i++ {
		mgr.task[i] = false
	}

	return mgr
}

func (m *Mgr) Handler(work *base.Work) {
	kind := work.Body.PopInt32()

	switch kind {
	case SystemCmd:
		m.HandlerKind0(work)
	case NormalCmd:
		m.HandlerKind1(work)
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
	case HeartbeatService:
		logger.Debug("Heartbeat")
		work.Body.Clear()
		work.Body.AddByte(0)
		work.Body.AddUInt16(1)
		work.Body.AddString("OK")
		work.SendTransData()
	case IntroductionService:
		tag := work.Body.PopString()
		if tag != "GOS" {
			fmt.Printf("Kind0, IntroductionService | 無效連線請求, TODO: 將當前連線中斷\n")
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
	case RequestService:
		data := work.Body.GetData()
		logger.Debug("data from asker: %+v", data)
		work.Body.Clear()

		work.Body.AddInt32(NormalCmd)
		work.Body.AddInt32(ResponseService)
		work.Body.AddString(fmt.Sprintf("Message from (m *Mgr) Handler(work *gos.Work), #data: %d", len(data)))
		work.SendTransData()
		logger.Debug("SendTransData back")
	case ResponseService:
		response := work.Body.PopString()
		logger.Debug("response: %s", response)
		work.Finish()
	default:
		data := work.Body.GetData()
		fmt.Printf("Kind1, undefined serivce %d, data: %+v\n", serivce, data)

		// 標註當前工作已完成，將該工作結構回收
		work.Finish()
	}
}

func (m *Mgr) Run() {
	m.cumTime += m.FrameTime

	if m.cumTime > m.nextSecond {
		var i int32

		for i = 50; i < 60; i++ {
			// 若任務 i 尚未完成
			if !m.task[i] {
				m.temp = append(m.temp, byte(i))
				logger.Debug("i: %d, temp: %+v", i, m.temp)

				m.Body.AddInt32(NormalCmd)
				m.Body.AddInt32(RequestService)
				m.Body.AddByteArray(m.temp)
				m.data = m.Body.FormData()
				// fmt.Printf("(s *Service) RunAsk | i: %d, length: %d, data: %+v\n", i, len(data), data)
				gos.SendToServer(0, &m.data, int32(len(m.data)))
				time.Sleep(1 * time.Second)
				m.Body.Clear()

				// 更新下次任務執行時間
				if i == 54 {
					m.nextSecond += 5 * time.Second
				} else {
					m.nextSecond += 1 * time.Second
				}

				logger.Debug("m.cumTime: %d, m.nextSecond: %d", m.cumTime, m.nextSecond)
				m.task[i] = true
				break
			}
		}
	}
}
