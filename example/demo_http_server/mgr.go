package main

import (
	"gos/ans"
	"gos/base/ghttp"
)

type Mgr struct {
}

func (m *Mgr) Handler(router *ans.Router) {
	router.GET("/", m.Get)
	router.POST("/", m.Post)

	r1 := router.NewRouter("/abc")

	r1.GET("/get", m.GetAbc)
	r1.POST("/post", m.PostAbc)
}

func (m *Mgr) Handler2(router *ans.Router2) {
	router.GET("/", func(c *ghttp.Context) {
		c.Response2.Json(200, ghttp.H{
			"index": 1,
			"msg":   "GET | /",
		})
	})
	router.POST("/", func(c *ghttp.Context) {
		c.Response2.Json(200, ghttp.H{
			"index": 2,
			"msg":   "POST | /",
		})
	})

	r1 := router.NewRouter("/abc")

	r1.GET("/get", func(c *ghttp.Context) {
		c.Response2.Json(200, ghttp.H{
			"index": 3,
			"msg":   "GET | /abc/get",
		})
	})
	r1.POST("/post", func(c *ghttp.Context) {
		c.Response2.Json(200, ghttp.H{
			"index": 4,
			"msg":   "POST | /abc/post",
		})
	})
}

func (m *Mgr) Get(req ghttp.Request, res *ghttp.Response) {
	res.Json(200, ghttp.H{
		"index": 1,
		"msg":   "GET | /",
	})
}

func (m *Mgr) Post(req ghttp.Request, res *ghttp.Response) {
	res.Json(200, ghttp.H{
		"index": 2,
		"msg":   "POST | /",
	})
}

func (m *Mgr) GetAbc(req ghttp.Request, res *ghttp.Response) {
	res.Json(200, ghttp.H{
		"index": 3,
		"msg":   "GET | /abc/get",
	})
}

func (m *Mgr) PostAbc(req ghttp.Request, res *ghttp.Response) {
	res.Json(200, ghttp.H{
		"index": 4,
		"msg":   "POST | /abc/post",
	})
}
