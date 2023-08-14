package main

import (
	"fmt"

	"github.com/j32u4ukh/gos"
	"github.com/j32u4ukh/gos/ans"
	"github.com/j32u4ukh/gos/base"
	"github.com/j32u4ukh/gos/base/ghttp"
)

type Mgr struct {
	HttpAnswer *ans.HttpAnser
}

func (m *Mgr) HttpHandler(router *ans.Router) {
	router.GET("/", func(c *ghttp.Context) {
		c.Json(200, ghttp.H{
			"index": 1,
			"msg":   "GET | /",
		})
	})
	router.POST("/", func(c *ghttp.Context) {
		c.Json(200, ghttp.H{
			"index": 2,
			"msg":   "POST | /",
		})
	})

	r1 := router.NewRouter("/abc")

	r1.GET("/get", func(c *ghttp.Context) {
		c.Json(200, ghttp.H{
			"index": 3,
			"msg":   "GET | /abc/get",
		})
	})
	r1.POST("/post", func(c *ghttp.Context) {
		c.Json(200, ghttp.H{
			"index": 4,
			"msg":   "POST | /abc/post",
		})
	})

	r1.POST("/delay_response", func(c *ghttp.Context) {
		m.CommissionHandler(1023, c.GetId())
	})
}

func (m *Mgr) CommissionHandler(site int32, cid int32) {
	if site == 1023 {
		td := base.NewTransData()
		td.AddInt32(1)
		td.AddInt32(1023)
		td.AddInt32(cid)
		data := td.FormData()
		err := gos.SendToServer(ERandomReturnServer, &data, int32(len(data)))

		if err != nil {
			// fmt.Printf("(m *Mgr) CommissionHandler | Failed to send to server %d: %v\nError: %+v\n", ERandomReturnServer, data, err)
			logger.Error("Failed to send to server %d: %v", ERandomReturnServer, data)
			logger.Error("Error: %+v", err)
			return
		}
	}
}

func (m *Mgr) RandomReturnServerHandler(work *base.Work) {
	cmd := work.Body.PopInt32()

	switch cmd {
	case 0:
		m.handleSystemCommand(work)
	case 1:
		m.handleCommission(work)
	default:
		// fmt.Printf("Unsupport command: %d\n", cmd)
		logger.Error("Unsupport command: %d", cmd)
		work.Finish()
	}
}

func (m *Mgr) handleSystemCommand(work *base.Work) {
	service := work.Body.PopInt32()

	switch service {
	case 0:
		response := work.Body.PopString()
		// fmt.Printf("Heart beat response: %s\n", response)
		logger.Debug("Heart beat response: %s", response)
		work.Finish()
	default:
		// fmt.Printf("Unsupport service: %d\n", service)
		logger.Error("Unsupport service: %d", service)
		work.Finish()
	}
}

func (m *Mgr) handleCommission(work *base.Work) {
	commission := work.Body.PopInt32()

	switch commission {
	case 1023:
		c := m.HttpAnswer.GetContext(-1)
		c.Cid = work.Body.PopInt32()
		response := work.Body.PopString()
		logger.Debug("response: %s", response)
		work.Finish()

		c.Json(200, ghttp.H{
			"index": 5,
			"msg":   fmt.Sprintf("POST | /abc/delay_response: %s", response),
		})
		m.HttpAnswer.Send(c)

	default:
		logger.Error("Unsupport commission: %d", commission)
		work.Finish()
	}
}
