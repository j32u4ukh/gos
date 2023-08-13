package main

import (
	"time"

	"github.com/j32u4ukh/gos/base"
)

type RandomReturnServer struct {
}

func (rrs *RandomReturnServer) Handler(work *base.Work) {
	cmd := work.Body.PopInt32()

	switch cmd {
	case 0:
		rrs.handleSystemCommand(work)
	case 1:
		rrs.handleCommission(work)
	default:
		logger.Error("Unsupport command: %d", cmd)
		work.Finish()
	}
}

func (rrs *RandomReturnServer) handleSystemCommand(work *base.Work) {
	service := work.Body.PopInt32()

	switch service {
	case 0:
		logger.Debug("Heart beat! Now: %+v", time.Now())
		work.Body.Clear()
		work.Body.AddInt32(0)
		work.Body.AddInt32(0)
		work.Body.AddString("OK")
		work.SendTransData()
	default:
		logger.Error("Unsupport service: %d", service)
		work.Finish()
	}
}

func (rrs *RandomReturnServer) handleCommission(work *base.Work) {
	commission := work.Body.PopInt32()

	switch commission {
	case 1023:
		cid := work.Body.PopInt32()
		work.Body.Clear()

		work.Body.AddInt32(1)
		work.Body.AddInt32(1023)
		work.Body.AddInt32(cid)
		work.Body.AddString("Commission completed.")
		work.SendTransData()

	default:
		logger.Error("Unsupport commission: %d", commission)
		work.Finish()
	}
}
