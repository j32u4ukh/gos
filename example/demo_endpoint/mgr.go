package main

import (
	"fmt"

	"github.com/j32u4ukh/gos/ans"
	"github.com/j32u4ukh/gos/base/ghttp"
	"github.com/j32u4ukh/gos/utils"
)

type Protocol struct {
	Name   string
	Age    int
	Height float32
}

type Mgr struct {
	HttpAnswer *ans.HttpAnser
}

func (m *Mgr) Handler(router *ans.Router) {
	router.GET("/", func(c *ghttp.Context) {
		c.Response.Json(200, ghttp.H{
			"index": 1,
			"msg":   "GET | /",
		})
		m.HttpAnswer.Send(c)
	})
	router.POST("/", func(c *ghttp.Context) {
		c.Response.Json(200, ghttp.H{
			"index": 2,
			"msg":   "POST | /",
		})
		m.HttpAnswer.Send(c)
	})

	r1 := router.NewRouter("/abc")

	r1.GET("/get", func(c *ghttp.Context) {
		c.Response.Json(200, ghttp.H{
			"index": 3,
			"msg":   "GET | /abc/get",
		})
		m.HttpAnswer.Send(c)
	})
	r1.POST("/post", func(c *ghttp.Context) {
		protocol := &Protocol{}
		c.ReadJson(protocol)
		c.Response.Json(200, ghttp.H{
			"index": 4,
			"msg":   fmt.Sprintf("POST | /abc/post | protocol: %v", protocol),
		})
		m.HttpAnswer.Send(c)
	})

	rName := r1.NewRouter("<name>")
	rName.POST("/<tag>", func(c *ghttp.Context) {
		protocol := &Protocol{}
		c.ReadJson(protocol)
		ok, name := c.GetParam("name")
		if ok {
			protocol.Name = name
		}
		var tag string
		ok, tag = c.GetParam("tag")
		if ok {
			utils.Debug("tag: %s", tag)
		}
		c.Response.Json(200, ghttp.H{
			"index": 5,
			"msg":   fmt.Sprintf("POST | /abc/<name>/<tag> | protocol: %v, tag: %s", protocol, tag),
		})
		m.HttpAnswer.Send(c)
	})
	rName.POST("/def", func(c *ghttp.Context) {
		protocol := &Protocol{}
		c.ReadJson(protocol)
		ok, name := c.GetParam("name")
		if ok {
			protocol.Name = name
		}
		c.Response.Json(200, ghttp.H{
			"index": 6,
			"msg":   fmt.Sprintf("POST | /abc/<name>/def | protocol: %v", protocol),
		})
		m.HttpAnswer.Send(c)
	})
}
