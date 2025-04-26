package httpserver

import (
	"time"

	"github.com/j32u4ukh/gos/server/ghttp"
	"github.com/pkg/errors"
)

type AnserManager struct {
	anser      *ghttp.HttpAnser
	contextMap map[int32]*ghttp.HttpContext
}

func NewAnserManager() *AnserManager {
	return &AnserManager{
		contextMap: make(map[int32]*ghttp.HttpContext),
	}
}

func (m *AnserManager) Init(port int32, nConnect int32) error {
	m.anser = ghttp.NewHttpAnser(port, nConnect)
	err := m.anser.Init()
	if err != nil {
		return errors.Wrap(err, "Failed to init http server")
	}
	router := m.anser.GetRouter()
	router.GET("/", func(c *ghttp.HttpContext) {
		c.Json(ghttp.StatusOK, ghttp.H{
			"msg": "ok",
		})
	})
	router.POST("/", func(c *ghttp.HttpContext) {
		c.Json(ghttp.StatusOK, ghttp.H{
			"msg": 123.45,
		})
	})
	abc := router.NewRouter("/abc")
	abc.GET("/get", func(c *ghttp.HttpContext) {
		c.Json(ghttp.StatusOK, ghttp.H{
			"msg": "/abc/get",
		})
	})
	abc.POST("/post", func(c *ghttp.HttpContext) {
		c.Json(ghttp.StatusOK, ghttp.H{
			"msg": "/abc/post",
		})
	})
	return nil
}

func (m *AnserManager) Run() {
	var start time.Time
	var during time.Duration
	const frameTime time.Duration = 20 * time.Millisecond
	go m.anser.Listen()
	for {
		start = time.Now()
		// TODO: Do something
		during = time.Since(start)
		if during < frameTime {
			time.Sleep(frameTime - during)
		}
	}
}
