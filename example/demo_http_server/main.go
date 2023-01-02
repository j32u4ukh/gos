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
	// mgr := NewMgr()
	// tcp0Asker := asker.(*ask.Tcp0Asker)
	// tcp0Asker.SetWorkHandler(mgr.Handler)
	// fmt.Printf("(s *Service) RunAsk | 伺服器初始化完成\n")
	// err = gos.StartConnect()

	// if err != nil {
	// 	fmt.Printf("Error: %+v\n", err)
	// 	return
	// }

	// fmt.Printf("(s *Service) RunAsk | 開始連線\n")
	var start time.Time
	var during, frameTime time.Duration = 0, 200 * time.Millisecond

	// // TODO: 暫緩一般數據傳送，先實作心跳包機制
	// go func() {
	// 	time.Sleep(2 * time.Second)
	// 	fmt.Printf("(s *Service) RunAsk | After 2 Second.\n")
	// 	var data []byte
	// 	temp := []byte{}

	// 	for i := 50; i < 55; i++ {
	// 		temp = append(temp, byte(i))
	// 		// fmt.Printf("(s *Service) RunAsk | i: %d, temp: %+v\n", i, temp)
	// 		mgr.Body.AddByte(1)
	// 		mgr.Body.AddUInt16(0)
	// 		mgr.Body.AddByteArray(temp)
	// 		data = mgr.Body.FormData()
	// 		// fmt.Printf("(s *Service) RunAsk | i: %d, length: %d, data: %+v\n", i, len(data), data)
	// 		gos.SendToServer(0, &data, int32(len(data)))
	// 		time.Sleep(1 * time.Second)
	// 		mgr.Body.Clear()
	// 	}

	// 	time.Sleep(5 * time.Second)
	// 	fmt.Printf("(s *Service) RunAsk | After 5 Second.\n")

	// 	for i := 55; i < 60; i++ {
	// 		temp = append(temp, byte(i))
	// 		// fmt.Printf("(s *Service) RunAsk | i: %d, temp: %+v\n", i, temp)
	// 		mgr.Body.AddByte(1)
	// 		mgr.Body.AddUInt16(0)
	// 		mgr.Body.AddByteArray(temp)
	// 		data = mgr.Body.FormData()
	// 		// fmt.Printf("(s *Service) RunAsk | i: %d, length: %d, data: %+v\n", i, len(data), data)
	// 		gos.SendToServer(0, &data, int32(len(data)))
	// 		time.Sleep(1 * time.Second)
	// 		mgr.Body.Clear()
	// 	}
	// }()

	// fmt.Printf("(s *Service) RunAsk | 開始 gos.RunAsk()\n")

	for {
		start = time.Now()

		gos.RunAsk()

		during = time.Since(start)
		if during < frameTime {
			time.Sleep(frameTime - during)
		}
	}
}
