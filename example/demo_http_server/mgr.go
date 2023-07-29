package main

import (
	"fmt"

	"github.com/j32u4ukh/gos/ans"
	"github.com/j32u4ukh/gos/base/ghttp"
)

type Protocol struct {
	Id       int32
	Password string
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

	rAbc := router.NewRouter("/abc")
	m.AbcHander(rAbc)

	rMethod := router.NewRouter("/method")
	m.MethodHander(rMethod)
}

func (m *Mgr) AbcHander(router *ans.Router) {
	router.GET("/get/", func(c *ghttp.Context) {
		c.Response.Json(200, ghttp.H{
			"index": 3,
			"msg":   "GET | /abc/get",
		})
		m.HttpAnswer.Send(c)
	})
	router.POST("/post/", func(c *ghttp.Context) {
		type Protocol struct {
			Name   string
			Age    int
			Height float32
		}
		protocol := &Protocol{}
		c.ReadJson(protocol)
		c.Response.Json(200, ghttp.H{
			"index": 4,
			"msg":   fmt.Sprintf("POST | /abc/post | protocol: %v", protocol),
		})
		m.HttpAnswer.Send(c)
	})
}

func (m *Mgr) MethodHander(router *ans.Router) {
	router.HEAD("/", func(c *ghttp.Context) {
		c.Status(ghttp.StatusOK)
		c.SetHeader("HeadMessage", "Message from head router.")
		c.BodyLength = 0
		c.SetContentLength()
		m.HttpAnswer.Send(c)
	})
	router.GET("/", func(c *ghttp.Context) {
		c.Response.Json(200, ghttp.H{
			"msg": "GET | /",
		})
		m.HttpAnswer.Send(c)
	})
	router.POST("/", func(c *ghttp.Context) {
		p := &Protocol{}
		c.ReadJson(p)
		c.Response.Json(200, ghttp.H{
			"msg": fmt.Sprintf("POST | /, %+v", p),
		})
		m.HttpAnswer.Send(c)
	})
	router.PUT("/", func(c *ghttp.Context) {
		p := &Protocol{}
		c.ReadJson(p)
		c.Response.Json(200, ghttp.H{
			"msg": fmt.Sprintf("PUT | /, %+v", p),
		})
		m.HttpAnswer.Send(c)
	})
	router.PATCH("/", func(c *ghttp.Context) {
		p := &Protocol{}
		c.ReadJson(p)
		c.Response.Json(200, ghttp.H{
			"msg": fmt.Sprintf("PATCH | /, %+v", p),
		})
		m.HttpAnswer.Send(c)
	})
	router.DELETE("/<id int>", func(c *ghttp.Context) {
		value := c.GetValue("id")
		if value != nil {
			c.Response.Json(200, ghttp.H{
				"msg": fmt.Sprintf("DELETE | /%d", value.(int64)),
			})
		} else {
			c.Response.Json(ghttp.StatusBadRequest, ghttp.H{
				"msg": "DELETE | /<id int>",
			})
		}
		m.HttpAnswer.Send(c)
	})
}
