package main

import (
	"os"
	"time"

	"github.com/j32u4ukh/glog/v2"
	"github.com/j32u4ukh/gos"
	"github.com/j32u4ukh/gos/ans"
	"github.com/j32u4ukh/gos/ask"
	"github.com/j32u4ukh/gos/base/ghttp"
	"github.com/j32u4ukh/gos/define"
	"github.com/j32u4ukh/gos/utils"
)

var logger *glog.Logger

func init() {
	gosLgger := glog.SetLogger(0, "gos", glog.DebugLevel)
	gosLgger.SetOptions(glog.DefaultOption(true, true), glog.UtcOption(8))
	gosLgger.SetFolder("log")
	utils.SetLogger(gosLgger)

	logger = glog.SetLogger(1, "DemoEndpoint", glog.DebugLevel)
	logger.SetFolder("log")
	logger.SetOptions(glog.DefaultOption(true, true), glog.UtcOption(8))
}

func main() {
	service_type := os.Args[1]
	var port int = 1023

	if service_type == "ans" {
		RunAns(port)
	} else if service_type == "ask" {
		RunAsk("127.0.0.1", port)
	}

	logger.Debug("End of gos example.")
}

func RunAns(port int) {
	anser, err := gos.Listen(define.Http, int32(port))
	logger.Debug("Listen to port %d", port)

	if err != nil {
		logger.Error("ListenError: %+v", err)
		return
	}

	httpAnswer := anser.(*ans.HttpAnser)
	mgr := &Mgr{}
	mgr.HttpAnswer = httpAnswer
	mgr.Handler(httpAnswer.Router)
	logger.Debug("伺服器初始化完成")

	gos.StartListen()
	logger.Debug("開始監聽")

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
	asker, err := gos.Bind(0, ip, port, define.Http, nil)

	if err != nil {
		logger.Error("BindError: %+v", err)
		return
	}

	http := asker.(*ask.HttpAsker)
	logger.Debug("http: %+v", http)

	req, err := ghttp.NewRequest(ghttp.MethodGet, "127.0.0.1:1023/abc/get", nil)

	if err != nil {
		logger.Error("NewRequestError: %+v", err)
		return
	}

	logger.Debug("req: %+v", req)
	var site int32
	site, err = gos.SendRequest(req, func(c *ghttp.Context) {
		logger.Info("I'm Context, Query: %s", c.Query)
	})

	logger.Debug("site: %d", site)

	if err != nil {
		logger.Error("SendRequestError: %+v", err)
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
