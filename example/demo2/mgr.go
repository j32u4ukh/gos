package main

import (
	"fmt"
	"gos"
	"gos/base"
	"time"
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
	kind := work.Body.PopByte()
	serivce := work.Body.PopUInt16()
	// fmt.Printf("(m *Mgr) Handler | index: %d, kind: %d, serivce: %d\n", work.Index, kind, serivce)

	if kind == 0 && serivce == 0 {
		fmt.Println("(m *Mgr) Handler | Heartbeat")

		// 標註當前工作已完成，將該工作結構回收
		work.Finish()

	} else if kind == 1 && serivce == 0 {
		data := work.Body.GetData()
		fmt.Printf("(m *Mgr) Handler | data from asker: %+v\n", data)
		work.Body.Clear()

		work.Body.AddByte(2)
		work.Body.AddUInt16(32)
		work.Body.AddString(fmt.Sprintf("Message from (m *Mgr) Handler(work *gos.Work), #data: %d", len(data)))
		work.SendTransData()
		fmt.Printf("(m *Mgr) Handler | SendTransData back\n")

	} else if kind == 2 && serivce == 32 {
		response := work.Body.PopString()
		fmt.Printf("(m *Mgr) Handler | response: %s\n", response)

		// 標註當前工作已完成，將該工作結構回收
		work.Finish()
	} else {
		data := work.Body.GetData()
		fmt.Printf("(m *Mgr) Handler | data: %+v\n", data)

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
				fmt.Printf("(m *Mgr) Run | i: %d, temp: %+v\n", i, m.temp)
				m.Body.AddByte(1)
				m.Body.AddUInt16(0)
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

				fmt.Printf("(m *Mgr) Run | m.cumTime: %d, m.nextSecond: %d\n", m.cumTime, m.nextSecond)
				m.task[i] = true
				break
			}
		}
	}
}
