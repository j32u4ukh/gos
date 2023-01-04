package main

import (
	"fmt"
	"gos"
	"gos/ans"
	"gos/ask"
	"gos/base/ghttp"
	"gos/define"
	"io/ioutil"
	"net/http"
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
	// demoNativeHttpRequest(ip, port)
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

func demoNativeHttpRequest(ip string, port int) {
	// client := &http.Client{}

	// //這邊可以任意變換 http method  GET、POST、PUT、DELETE
	// req, err := http.NewRequest("GET", fmt.Sprintf("%s:%d/abc/get", ip, port), nil)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// // req.Header.Add("If-None-Match", `W/"wyzzy"`)
	// res, err := client.Do(req)
	// requestURL := fmt.Sprintf("http://localhost:%d/abc/get", port)

	requestURL := fmt.Sprintf("http://127.0.0.1:%d/abc/get", port)

	res, err := http.Get(requestURL)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()
	sitemap, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("err: %+v\n", err)
		return
	}

	fmt.Printf("%s\n", sitemap)
}
