package main

import (
	"time"

	"github.com/j32u4ukh/gos/ans"
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
		start := time.Now()
		wait := 2 * time.Second

		for time.Since(start) < wait {
			time.Sleep(500 * time.Millisecond)
		}

		m.CommissionHandler(1023, c.GetId())
	})
}

func (m *Mgr) CommissionHandler(site int32, cid int32) {
	if site == 1023 {
		c := m.HttpAnswer.GetContext(-1)
		c.Cid = cid
		c.Json(200, ghttp.H{
			"index": 5,
			"msg":   "POST | /abc/delay_response",
		})
		m.HttpAnswer.Send(c)
	}
}
