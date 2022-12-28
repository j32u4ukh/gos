package main

import (
	"fmt"
	"gos"
	"gos/ans"
	"gos/base/ghttp"
	"gos/define"
	"time"
)

func main() {
	// service_type := os.Args[1]
	var port int = 1023
	RunAns(port)
	// if len(os.Args) >= 3 {
	// 	port, _ = strconv.Atoi(os.Args[2])
	// }

	// if service_type == "ans" {

	// }

	fmt.Println("[Example] Run | End of gos example.")
}

func RunAns(port int) {
	anser, err := gos.Listen(define.Http, int32(port))
	fmt.Printf("RunAns | Listen to port %d\n", port)

	if err != nil {
		fmt.Printf("Error: %+v\n", err)
		return
	}

	httpAnswer := anser.(*ans.HttpAnser)

	httpAnswer.GET("/", func(req ghttp.Request, res *ghttp.Response) {
		res.Json(200, ghttp.H{
			"index": 1,
			"msg":   "GET | /",
		})
	})

	httpAnswer.POST("/", func(req ghttp.Request, res *ghttp.Response) {
		res.Json(200, ghttp.H{
			"index": 2,
			"msg":   "POST | /",
		})
	})

	r1 := httpAnswer.NewRouter("/abc")

	r1.GET("/get", func(req ghttp.Request, res *ghttp.Response) {
		res.Json(200, ghttp.H{
			"index": 3,
			"msg":   "GET | /abc/get",
		})
	})

	r1.POST("/post", func(req ghttp.Request, res *ghttp.Response) {
		res.Json(200, ghttp.H{
			"index": 4,
			"msg":   "POST | /abc/post",
		})
	})

	fmt.Printf("(s *Service) RunAns | 伺服器初始化完成\n")
	gos.StartListen()
	fmt.Printf("(s *Service) RunAns | 開始監聽\n")
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
