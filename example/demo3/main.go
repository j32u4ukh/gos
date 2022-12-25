package main

import (
	"fmt"
	"gos"
	"gos/ans"
	"gos/base"
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

	httpAnswer.GET("/", func(req base.Request, res *base.Response) {
		fmt.Println("GET | /")
	})

	httpAnswer.POST("/", func(req base.Request, res *base.Response) {
		fmt.Println("POST | /")
	})

	r1 := httpAnswer.NewRouter("/abc")

	r1.GET("/get", func(req base.Request, res *base.Response) {
		fmt.Println("GET | /abc/get")
	})

	r1.POST("/post", func(req base.Request, res *base.Response) {
		fmt.Println("POST | /abc/post")
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
