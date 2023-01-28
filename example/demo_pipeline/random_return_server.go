package main

import (
	"fmt"
	"time"

	"github.com/j32u4ukh/gos/base"
)

type RandomReturnServer struct {
}

func (rrs *RandomReturnServer) Handler(work *base.Work) {
	cmd := work.Body.PopByte()

	switch cmd {
	case 0:
		rrs.handleSystemCommand(work)
	case 1:
		rrs.handleCommission(work)
	default:
		fmt.Printf("Unsupport command: %d\n", cmd)
		work.Finish()
	}
}

func (rrs *RandomReturnServer) Run() {

}

func (rrs *RandomReturnServer) handleSystemCommand(work *base.Work) {
	service := work.Body.PopUInt16()

	switch service {
	case 0:
		fmt.Printf("Heart beat! Now: %+v\n", time.Now())
		work.Body.Clear()
		work.Body.AddByte(0)
		work.Body.AddUInt16(0)
		work.Body.AddString("OK")
		work.SendTransData()
	default:
		fmt.Printf("Unsupport service: %d\n", service)
		work.Finish()
	}
}

func (rrs *RandomReturnServer) handleCommission(work *base.Work) {
	commission := work.Body.PopUInt16()

	switch commission {
	case 1023:
		cid := work.Body.PopInt32()
		work.Body.Clear()

		work.Body.AddByte(1)
		work.Body.AddUInt16(1023)
		work.Body.AddInt32(cid)
		work.Body.AddString("Commission completed.")
		work.SendTransData()

	default:
		fmt.Printf("Unsupport commission: %d\n", commission)
		work.Finish()
	}
}
