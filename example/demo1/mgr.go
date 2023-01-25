package main

import (
	"fmt"

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
		fmt.Printf("(m *Mgr) Handler | SendTransData back, work: %+v\n", work)

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
