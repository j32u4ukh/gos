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
		m.HttpAnswer.Send(c)
	})
	router.POST("/", func(c *ghttp.Context) {
		c.Json(200, ghttp.H{
			"index": 2,
			"msg":   "POST | /",
		})
		m.HttpAnswer.Send(c)
	})

	r1 := router.NewRouter("/abc")

	r1.GET("/get", func(c *ghttp.Context) {
		c.Json(200, ghttp.H{
			"index": 3,
			"msg":   "GET | /abc/get",
		})
		m.HttpAnswer.Send(c)
	})
	r1.POST("/post", func(c *ghttp.Context) {
		c.Json(200, ghttp.H{
			"index": 4,
			"msg":   "POST | /abc/post",
		})
		m.HttpAnswer.Send(c)
	})

	r1.POST("/delay_response", func(c *ghttp.Context) {
		m.HttpAnswer.Finish(c)
		// start := time.Now()
		// wait := 2 * time.Second

		// for time.Since(start) < wait {
		// 	time.Sleep(500 * time.Millisecond)
		// }

		m.CommissionHandler(1023, c.GetId())
	})
}

func (m *Mgr) CommissionHandler(site int32, cid int32) {
	if site == 1023 {
		// c := m.HttpAnswer.GetContext(-1)
		// c.Cid = cid
		// c.Json(200, ghttp.H{
		// 	"index": 5,
		// 	"msg":   "POST | /abc/delay_response",
		// })
		// m.HttpAnswer.Send(c)

		td := base.NewTransData()
		td.AddByte(1)
		td.AddUInt16(1023)
		td.AddInt32(cid)
		data := td.FormData()
		err := gos.SendToServer(ERandomReturnServer, &data, td.GetLength())

		if err != nil {
			fmt.Printf("(m *Mgr) CommissionHandler | Failed to send to server %d: %v\nError: %+v\n", ERandomReturnServer, data, err)
			return
		}
	}
}

func (m *Mgr) RandomReturnServerHandler(work *base.Work) {
	cmd := work.Body.PopByte()

	switch cmd {
	case 0:
		m.handleSystemCommand(work)
	case 1:
		m.handleCommission(work)
	default:
		fmt.Printf("Unsupport command: %d\n", cmd)
		work.Finish()
	}
}

func (m *Mgr) handleSystemCommand(work *base.Work) {
	service := work.Body.PopUInt16()

	switch service {
	case 0:
		// data := work.Body.GetData()
		// fmt.Printf("response data: %s\n%v", data, data)
		response := work.Body.PopString()
		fmt.Printf("Heart beat response: %s\n", response)
		work.Finish()
	default:
		fmt.Printf("Unsupport service: %d\n", service)
		work.Finish()
	}
}

func (m *Mgr) handleCommission(work *base.Work) {
	commission := work.Body.PopUInt16()

	switch commission {
	case 1023:
		c := m.HttpAnswer.GetContext(-1)
		c.Cid = work.Body.PopInt32()
		response := work.Body.PopString()
		fmt.Printf("(m *Mgr) handleCommission | response: %s\n", response)
		work.Finish()

		c.Json(200, ghttp.H{
			"index": 5,
			"msg":   fmt.Sprintf("POST | /abc/delay_response: %s", response),
		})
		m.HttpAnswer.Send(c)

	default:
		fmt.Printf("Unsupport commission: %d\n", commission)
		work.Finish()
	}
}

func (m *Mgr) Run() {

}
