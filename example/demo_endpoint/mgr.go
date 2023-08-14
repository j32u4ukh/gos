package main

import (
	"fmt"
	"time"

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
	})
	router.POST("/", func(c *ghttp.Context) {
		c.Response.Json(200, ghttp.H{
			"index": 2,
			"msg":   "POST | /",
		})
	})

	r1 := router.NewRouter("/abc")

	r1.GET("/get", func(c *ghttp.Context) {
		c.Response.Json(200, ghttp.H{
			"index": 3,
			"msg":   "GET | /abc/get",
		})
	})
	r1.POST("/post", func(c *ghttp.Context) {
		protocol := &Protocol{}
		c.ReadJson(protocol)
		c.Response.Json(200, ghttp.H{
			"index": 4,
			"msg":   fmt.Sprintf("POST | /abc/post | protocol: %v", protocol),
		})
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
	})
	rName.POST("/<id int>", func(c *ghttp.Context) {
		protocol := &Protocol{}
		c.ReadJson(protocol)
		ok, name := c.GetParam("name")
		if ok {
			protocol.Name = name
		}
		value := c.GetValue("id")
		var id int = 0
		if value != nil {
			id = value.(int)
		}
		c.Response.Json(200, ghttp.H{
			"index": 7,
			"msg":   fmt.Sprintf("POST | /abc/<name>/<id int> | protocol: %v, id: %d", protocol, id),
		})
	})
	rName.POST("/<pi float>", func(c *ghttp.Context) {
		protocol := &Protocol{}
		c.ReadJson(protocol)
		ok, name := c.GetParam("name")
		if ok {
			protocol.Name = name
		}
		value := c.GetValue("pi")
		var pi float64 = 0
		if value != nil {
			pi = value.(float64)
		}
		c.Response.Json(200, ghttp.H{
			"index": 8,
			"msg":   fmt.Sprintf("POST | /abc/<name>/<pi float> | protocol: %v, pi: %v", protocol, pi),
		})
	})
	rName.GET("/get/<user_id int>", func(c *ghttp.Context) {
		_, name := c.GetParam("name")
		value := c.GetValue("user_id")
		var id int = 0
		if value != nil {
			id = value.(int)
		}
		dt := time.Now().UnixMilli()
		c.Response.Json(200, ghttp.H{
			"index": 9,
			"msg":   fmt.Sprintf("GET | /abc/<name>/get/<user_id int> | name: %s, id: %d", name, id),
			"dt":    dt,
		})
	})
	rName.GET("/uint/<value uint>", func(c *ghttp.Context) {
		_, name := c.GetParam("name")
		value := c.GetValue("value")
		var id uint64 = 0
		if value != nil {
			id = value.(uint64)
		}
		dt := time.Now().UnixMilli()
		c.Response.Json(200, ghttp.H{
			"index": 10,
			"msg":   fmt.Sprintf("GET | /abc/<name>/uint/<value uint> | name: %s, id(#id = %d): %d", name, len(fmt.Sprintf("%d", id)), id),
			"dt":    dt,
		})
	})
	rName.GET("/uint/<value int>", func(c *ghttp.Context) {
		name := c.GetValue("name").(string)
		value := c.GetValue("value")
		var id int64 = 0
		if value != nil {
			id = value.(int64)
		}
		dt := time.Now().UnixMilli()
		c.Response.Json(200, ghttp.H{
			"index": 11,
			"msg":   fmt.Sprintf("GET | /abc/<name>/uint/<value int> | name: %s, id(#id = %d): %d", name, len(fmt.Sprintf("%d", id)), id),
			"dt":    dt,
		})
	})
}
