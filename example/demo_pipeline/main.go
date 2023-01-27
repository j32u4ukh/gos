package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/j32u4ukh/gos"
	"github.com/j32u4ukh/gos/ans"
	"github.com/j32u4ukh/gos/ask"
	"github.com/j32u4ukh/gos/define"
)

func main() {
	service_type := os.Args[1]

	if service_type == "ms" {
		RunMainServer(1023)
	} else if service_type == "ask" {
		RunAsk(1023)
	} else if service_type == "rrs" {
		RunRandomReturnServer(1022)
	}

	fmt.Println("[Example] Run | End of gos example.")
}

func RunMainServer(port int) {
	anser, err := gos.Listen(define.Http, int32(port))
	fmt.Printf("RunMainServer | Listen to port %d\n", port)

	if err != nil {
		fmt.Printf("RunMainServer | Listen error: %+v\n", err)
		return
	}

	mgr := &Mgr{}
	httpAnswer := anser.(*ans.HttpAnser)
	mgr.HttpAnswer = anser.(*ans.HttpAnser)
	mgr.HttpHandler(httpAnswer.Router)
	fmt.Printf("RunMainServer | Http Anser 伺服器初始化完成\n")

	asker, err := gos.Bind(0, "127.0.0.1", 1022, define.Tcp0)

	if err != nil {
		fmt.Printf("RunMainServer | Bind error: %+v\n", err)
		return
	}

	tcp0Asker := asker.(*ask.Tcp0Asker)
	tcp0Asker.SetWorkHandler(mgr.RandomReturnServerHandler)
	fmt.Printf("RunMainServer | RandomReturnServer Asker 伺服器初始化完成\n")

	fmt.Printf("RunMainServer | 伺服器初始化完成\n")

	// =============================================
	// 開始所有已註冊的監聽
	// =============================================
	gos.StartListen()
	fmt.Printf("RunMainServer | 開始監聽\n")

	err = gos.StartConnect()

	if err != nil {
		fmt.Printf("RunMainServer | 與 RandomReturnServer 連線時發生錯誤, error: %+v\n", err)
		return
	}

	fmt.Printf("RunMainServer | 成功與 RandomReturnServer 連線\n")
	var start time.Time
	var during, frameTime time.Duration = 0, 200 * time.Millisecond

	for {
		start = time.Now()

		gos.RunAns()
		gos.RunAsk()

		during = time.Since(start)
		if during < frameTime {
			time.Sleep(frameTime - during)
		}
	}
}

func RunAsk(port int) {
	method := "GET"
	url := fmt.Sprintf("http://192.168.0.198:%d", port)
	payload := strings.NewReader(`{"client_message": "hello, server!"}`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return
	}

	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)

	if err != nil {
		fmt.Println(err)
		return
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(body))
}

func RunRandomReturnServer(port int) {
	anser, err := gos.Listen(define.Tcp0, int32(port))
	fmt.Printf("RunRandomReturnServer | Listen to port %d\n", port)

	if err != nil {
		fmt.Printf("RunRandomReturnServer | Error: %+v\n", err)
		return
	}

	rrs := &RandomReturnServer{}
	tcpAnser := anser.(*ans.Tcp0Anser)
	tcpAnser.SetWorkHandler(rrs.Handler)
	fmt.Printf("RunRandomReturnServer | 伺服器初始化完成\n")

	// =============================================
	// 開始所有已註冊的監聽
	// =============================================
	gos.StartListen()
	fmt.Printf("RunRandomReturnServer | 開始監聽\n")
	var start time.Time
	var during, frameTime time.Duration = 0, 200 * time.Millisecond

	for {
		start = time.Now()

		gos.RunAns()
		rrs.Run()

		during = time.Since(start)
		if during < frameTime {
			time.Sleep(frameTime - during)
		}
	}
}
