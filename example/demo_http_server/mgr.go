package main

import (
	"fmt"
	"time"

	"github.com/j32u4ukh/gos/ans"
	"github.com/j32u4ukh/gos/base/ghttp"
	"github.com/j32u4ukh/gos/utils"
)

type Protocol struct {
	Id       int32
	Password string
	Name     string
}

type Mgr struct {
}

func (m *Mgr) Handler(router *ans.Router) {
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

	rAbc := router.NewRouter("/abc")
	m.AbcHander(rAbc)

	rMethod := router.NewRouter("/method")
	m.MethodHander(rMethod)
}

func (m *Mgr) AbcHander(router *ans.Router) {
	router.GET("/get/", func(c *ghttp.Context) {
		c.Json(200, ghttp.H{
			"index": 3,
			"msg":   "GET | /abc/get",
		})
	})
	router.POST("/post/", func(c *ghttp.Context) {
		type Protocol struct {
			Name   string
			Age    int
			Height float32
		}
		protocol := &Protocol{}
		c.ReadJson(protocol)
		c.Json(200, ghttp.H{
			"index": 4,
			"msg":   fmt.Sprintf("POST | /abc/post | protocol: %v", protocol),
		})
	})

	rName := router.NewRouter("<name>")
	m.AbcNameHander(rName)
}

func (m *Mgr) AbcNameHander(router *ans.Router) {
	router.POST("/<tag>", func(c *ghttp.Context) {
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
		c.Json(200, ghttp.H{
			"index": 5,
			"msg":   fmt.Sprintf("POST | /abc/<name>/<tag> | protocol: %v, tag: %s", protocol, tag),
		})
	})
	router.POST("/def", func(c *ghttp.Context) {
		protocol := &Protocol{}
		c.ReadJson(protocol)
		ok, name := c.GetParam("name")
		if ok {
			protocol.Name = name
		}
		c.Json(200, ghttp.H{
			"index": 6,
			"msg":   fmt.Sprintf("POST | /abc/<name>/def | protocol: %v", protocol),
		})
	})
	router.POST("/<id int>", func(c *ghttp.Context) {
		protocol := &Protocol{}
		c.ReadJson(protocol)
		ok, name := c.GetParam("name")
		if ok {
			protocol.Name = name
		}
		value := c.GetValue("id")
		var id int64 = 0
		if value != nil {
			id = value.(int64)
		}
		c.Json(200, ghttp.H{
			"index": 7,
			"msg":   fmt.Sprintf("POST | /abc/<name>/<id int> | protocol: %v, id: %d", protocol, id),
		})
	})
	router.POST("/<pi float>", func(c *ghttp.Context) {
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
		c.Json(200, ghttp.H{
			"index": 8,
			"msg":   fmt.Sprintf("POST | /abc/<name>/<pi float> | protocol: %v, pi: %v", protocol, pi),
		})
	})
	router.GET("/get/<user_id int>", func(c *ghttp.Context) {
		_, name := c.GetParam("name")
		value := c.GetValue("user_id")
		var id int64 = 0
		if value != nil {
			id = value.(int64)
		}
		dt := time.Now().UnixMilli()
		c.Json(200, ghttp.H{
			"index": 9,
			"msg":   fmt.Sprintf("GET | /abc/<name>/get/<user_id int> | name: %s, id: %d", name, id),
			"dt":    dt,
		})
	})
	router.GET("/uint/<value uint>", func(c *ghttp.Context) {
		_, name := c.GetParam("name")
		value := c.GetValue("value")
		var id uint64 = 0
		if value != nil {
			id = value.(uint64)
		}
		dt := time.Now().UnixMilli()
		c.Json(200, ghttp.H{
			"index": 10,
			"msg":   fmt.Sprintf("GET | /abc/<name>/uint/<value uint> | name: %s, id(#id = %d): %d", name, len(fmt.Sprintf("%d", id)), id),
			"dt":    dt,
		})
	})
	router.GET("/uint/<value int>", func(c *ghttp.Context) {
		name := c.GetValue("name").(string)
		value := c.GetValue("value")
		var id int64 = 0
		if value != nil {
			id = value.(int64)
		}
		dt := time.Now().UnixMilli()
		c.Json(200, ghttp.H{
			"index": 11,
			"msg":   fmt.Sprintf("GET | /abc/<name>/uint/<value int> | name: %s, id(#id = %d): %d", name, len(fmt.Sprintf("%d", id)), id),
			"dt":    dt,
		})
	})
}

func (m *Mgr) MethodHander(router *ans.Router) {
	router.HEAD("/", func(c *ghttp.Context) {
		c.Status(ghttp.StatusOK)
		c.Response.SetHeader("HeadMessage", "Message from head router.")
		c.Response.BodyLength = 0
		c.Response.SetContentLength()

	})
	router.GET("/", func(c *ghttp.Context) {
		c.Json(200, ghttp.H{
			"msg": "GET | /",
		})
	})
	router.POST("/", func(c *ghttp.Context) {
		p := &Protocol{}
		c.ReadJson(p)
		c.Json(200, ghttp.H{
			"msg": fmt.Sprintf("POST | /, %+v", p),
		})
	})
	router.PUT("/", func(c *ghttp.Context) {
		p := &Protocol{}
		c.ReadJson(p)
		c.Json(200, ghttp.H{
			"msg": fmt.Sprintf("PUT | /, %+v", p),
		})
	})
	router.PATCH("/", func(c *ghttp.Context) {
		p := &Protocol{}
		c.ReadJson(p)
		c.Json(200, ghttp.H{
			"msg": fmt.Sprintf("PATCH | /, %+v", p),
		})
	})
	router.DELETE("/<id int>", func(c *ghttp.Context) {
		value := c.GetValue("id")
		if value != nil {
			c.Json(200, ghttp.H{
				"msg": fmt.Sprintf("DELETE | /%d", value.(int64)),
			})
		} else {
			c.Json(ghttp.StatusBadRequest, ghttp.H{
				"msg": "DELETE | /<id int>",
			})
		}
	})
}
