package main

import (
	"fmt"
	"time"

	"github.com/j32u4ukh/gos"
	"github.com/j32u4ukh/gos/ans"
	"github.com/j32u4ukh/gos/base/ghttp"
	"github.com/j32u4ukh/gos/define"
)

func main() {
	data := make([]int, 10)
	data[0] = 1
	data[1] = 3
	data[2] = 5
	data[3] = 7
	data[4] = 11
	fmt.Printf("#data: %d, data: %+v\n", len(data), data)
	data = data[:0]
	data[0] = 13
	fmt.Printf("#data: %d, data: %+v\n", len(data), data)
}

func RunAns(port int) {
	anser, err := gos.Listen(define.Http, int32(port))
	fmt.Printf("Listen to port %d", port)

	if err != nil {
		fmt.Printf("ListenError: %+v", err)
		return
	}

	httpAnswer := anser.(*ans.HttpAnser)
	mgr := &Mgr{}
	mgr.HttpAnswer = httpAnswer
	mgr.Handler(httpAnswer.Router)
	fmt.Printf("伺服器初始化完成")

	gos.StartListen()
	fmt.Printf("開始監聽")

	var start time.Time
	var during, frameTime time.Duration = 0, 200 * time.Millisecond

	for {
		start = time.Now()

		gos.RunAns()

		during = time.Since(start)
		if during < frameTime {
			time.Sleep(frameTime - during)
		}
	}
}

type Mgr struct {
	HttpAnswer *ans.HttpAnser
}

type Protocol struct {
	Name string
	Age  int
}

func (m *Mgr) Handler(router *ans.Router) {
	router.GET("/", func(c *ghttp.Context) {
		defer func() {
			if err := recover(); err != nil {
				c.Json(ghttp.StatusInternalServerError, ghttp.H{
					"msg": fmt.Sprintf("err: %+v", err),
				})
			}
		}()
		p := &Protocol{}
		data, err := c.ReadJson(p)

		if err != nil {
			fmt.Printf("ReadJson err: %v\n", err)
			c.Json(ghttp.StatusBadRequest, ghttp.H{
				"err": "ReadJson",
				"msg": data,
			})
			return
		}
		c.Json(ghttp.StatusOK, ghttp.H{
			"index": 1,
			"msg":   fmt.Sprintf("Protocol: %+v", p),
		})
	})
}
