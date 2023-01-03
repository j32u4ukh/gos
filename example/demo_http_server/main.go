package main

import (
	"fmt"
	"gos"
	"gos/ans"
	"gos/ask"
	"gos/base/ghttp"
	"gos/define"
	"os"
	"time"
)

func main() {
	service_type := os.Args[1]
	var port int = 1023
	// if len(os.Args) >= 3 {
	// 	port, _ = strconv.Atoi(os.Args[2])
	// }

	if service_type == "ans" {
		RunAns(port)
	} else if service_type == "ask" {
		RunAsk("127.0.0.1", port)
	}

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
	mrg := &Mgr{}
	mrg.Handler(httpAnswer.Router)

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

func RunAsk(ip string, port int) {
	asker, err := gos.Bind(0, ip, port, define.Http)

	if err != nil {
		fmt.Printf("BindError: %+v\n", err)
		return
	}
	http := asker.(*ask.HttpAsker)
	fmt.Printf("http: %+v\n", http)

	req, err := ghttp.NewRequest(ghttp.MethodGet, "127.0.0.1:1023/abc/get", nil)

	if err != nil {
		fmt.Printf("NewRequestError: %+v\n", err)
		return
	}

	fmt.Printf("req: %+v\n", req)
	err = gos.SendRequest(req, func(res *ghttp.Response) {
		fmt.Printf("Response: %+v\n", res)
	})

	if err != nil {
		fmt.Printf("SendRequestError: %+v\n", err)
		return
	}

	var start time.Time
	var during, frameTime time.Duration = 0, 200 * time.Millisecond

	for {
		start = time.Now()

		gos.RunAsk()

		during = time.Since(start)
		if during < frameTime {
			time.Sleep(frameTime - during)
		}
	}
}
