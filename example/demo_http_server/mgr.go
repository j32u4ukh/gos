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
