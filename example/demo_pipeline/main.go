package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/j32u4ukh/glog/v2"
	"github.com/j32u4ukh/gos"
	"github.com/j32u4ukh/gos/ans"
	"github.com/j32u4ukh/gos/ask"
	"github.com/j32u4ukh/gos/base"
	"github.com/j32u4ukh/gos/define"
	"github.com/j32u4ukh/gos/utils"
)

const ERandomReturnServer int32 = 0

var logger *glog.Logger

func init() {
	utils.SetLogger(glog.SetLogger(0, "gos", glog.DebugLevel))
	glog.GetLogger(0).SetFolder("log")
	logger = glog.SetLogger(1, "DemoPipeline", glog.DebugLevel)
	logger.SetFolder("log")
	logger.SetOptions(glog.DefaultOption(true, true), glog.UtcOption(8))
	logger.Warn("Test")
	logger.Error("Test")
}

// TODO: 獲取關閉訊號，將 log 數據寫完再關閉程式
func main() {
	service_type := os.Args[1]

	if service_type == "ms" {
		RunMainServer(1023)
	} else if service_type == "ask" {
		RunAsk(1023)
	} else if service_type == "rrs" {
		RunRandomReturnServer(1022)
	}

	// fmt.Println("[Example] Run | End of gos example.")
	logger.Info("End of gos example.")
	glog.Flush()
}

// MainServer 接受客戶端 http 請求，再將請求發送到 RandomReturnServer 做處理，RandomReturnServer 將結果返還 MainServer，再由 MainServer 回覆客戶端
func RunMainServer(port int) {
	anser, err := gos.Listen(define.Http, int32(port))
	// fmt.Printf("RunMainServer | Listen to port %d\n", port)
	logger.Debug("Listen to port %d", port)

	if err != nil {
		// fmt.Printf("RunMainServer | Listen error: %+v\n", err)
		logger.Error("Listen error: %+v", err)
		return
	}

	mgr := &Mgr{}
	httpAnswer := anser.(*ans.HttpAnser)
	mgr.HttpAnswer = anser.(*ans.HttpAnser)
	mgr.HttpHandler(httpAnswer.Router)
	// fmt.Printf("RunMainServer | Http Anser 伺服器初始化完成\n")
	logger.Debug("Http Anser 伺服器初始化完成")

	asker, err := gos.Bind(ERandomReturnServer, "127.0.0.1", 1022, define.Tcp0, base.OnEventsFunc{
		define.OnConnected: func(any) {
			fmt.Printf("(s *Service) RunAsk | onConnect to %s:%d\n", "127.0.0.1", port)
		},
	})

	if err != nil {
		// fmt.Printf("RunMainServer | Bind error: %+v\n", err)
		logger.Error("Bind error: %+v", err)
		return
	}

	tcp0Asker := asker.(*ask.Tcp0Asker)
	tcp0Asker.SetWorkHandler(mgr.RandomReturnServerHandler)
	// fmt.Printf("RunMainServer | RandomReturnServer Asker 伺服器初始化完成\n")
	// fmt.Printf("RunMainServer | 伺服器初始化完成\n")
	logger.Debug("RandomReturnServer Asker 伺服器初始化完成")
	logger.Debug("伺服器初始化完成")

	// =============================================
	// 開始所有已註冊的監聽
	// =============================================
	gos.StartListen()
	// fmt.Printf("RunMainServer | 開始監聽\n")
	logger.Debug("開始監聽")

	err = gos.StartConnect()

	if err != nil {
		// fmt.Printf("RunMainServer | 與 RandomReturnServer 連線時發生錯誤, error: %+v\n", err)
		logger.Error("與 RandomReturnServer 連線時發生錯誤, error: %+v", err)
		return
	}

	// fmt.Printf("RunMainServer | 成功與 RandomReturnServer 連線\n")
	logger.Debug("成功與 RandomReturnServer 連線")
	var start time.Time
	var during, frameTime time.Duration = 0, 20 * time.Millisecond

	for {
		start = time.Now()

		gos.RunAns()
		gos.RunAsk()
		mgr.Run()

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
		// fmt.Println(err)
		logger.Error("error: %+v", err)
		return
	}

	req.Header.Add("Content-Type", "application/json")
	res, err := client.Do(req)

	if err != nil {
		// fmt.Println(err)
		logger.Error("error: %+v", err)
		return
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		// fmt.Println(err)
		logger.Error("error: %+v", err)
		return
	}

	// fmt.Println(string(body))
	logger.Info("Response: %s", string(body))
}

func RunRandomReturnServer(port int) {
	anser, err := gos.Listen(define.Tcp0, int32(port))
	// fmt.Printf("RunRandomReturnServer | Listen to port %d\n", port)
	logger.Debug("Listen to port %d", port)

	if err != nil {
		// fmt.Printf("RunRandomReturnServer | Error: %+v\n", err)
		logger.Error("Listen error: %+v", err)
		return
	}

	rrs := &RandomReturnServer{}
	tcpAnser := anser.(*ans.Tcp0Anser)
	tcpAnser.SetWorkHandler(rrs.Handler)
	// fmt.Printf("RunRandomReturnServer | 伺服器初始化完成\n")
	logger.Debug("伺服器初始化完成")

	// =============================================
	// 開始所有已註冊的監聽
	// =============================================
	gos.StartListen()
	// fmt.Printf("RunRandomReturnServer | 開始監聽\n")
	logger.Debug("開始監聽")
	var start time.Time
	var during, frameTime time.Duration = 0, 20 * time.Millisecond

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
