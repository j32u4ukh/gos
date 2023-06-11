package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/j32u4ukh/glog/v2"
	"github.com/j32u4ukh/gos"
	"github.com/j32u4ukh/gos/ans"
	"github.com/j32u4ukh/gos/ask"
	"github.com/j32u4ukh/gos/define"
	"github.com/j32u4ukh/gos/utils"
)

var logger *glog.Logger

func init() {
	gosLogger := glog.SetLogger(0, "gos", glog.DebugLevel)
	gosLogger.SetFolder("log")
	gosLogger.SetOptions(glog.DefaultOption(true, true), glog.UtcOption(8))
	utils.SetLogger(gosLogger)
	logger = glog.SetLogger(1, "Demo2", glog.DebugLevel)
	logger.SetFolder("log")
	logger.SetOptions(glog.DefaultOption(true, true), glog.UtcOption(8))
}

type Service struct {
	// 總管整個服務的關閉流程(可能有不同原因會觸發關閉流程)
	StopCh chan bool
}

func main() {
	service := Service{StopCh: make(chan bool)}
	service.Run(os.Args)
}

func (s *Service) Run(args []string) {
	service_type := args[1]
	var port int = 1023

	if len(args) >= 3 {
		port, _ = strconv.Atoi(args[2])
	}

	if service_type == "ans" {
		s.RunAns(port)

	} else if service_type == "ask" {
		s.RunAsk("127.0.0.1", port)

	}

	// fmt.Println("[Example] Run | End of gos example.")
	logger.Info("End of gos example.")
}

func (s *Service) Stop() {
	s.StopCh <- true
}

func (s *Service) RunAns(port int) {
	anser, err := gos.Listen(define.Tcp0, int32(port))
	// fmt.Printf("(s *Service) RunAns | Listen to port %d\n", port)
	logger.Info("Listen to port %d", port)

	if err != nil {
		// fmt.Printf("Error: %+v\n", err)
		logger.Error("ListenError: %+v", err)
		return
	}

	mgr := &Mgr{}
	tcp0Answer := anser.(*ans.Tcp0Anser)
	tcp0Answer.SetWorkHandler(mgr.Handler)
	// anser.SetWorkHandler(mgr.Handler)
	// fmt.Printf("(s *Service) RunAns | 伺服器初始化完成\n")
	logger.Debug("伺服器初始化完成")

	gos.StartListen()
	// fmt.Printf("(s *Service) RunAns | 開始監聽\n")
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

func (s *Service) RunAsk(ip string, port int) {
	asker, err := gos.Bind(0, ip, port, define.Tcp0, map[define.EventType]func(){
		define.OnConnected: func() {
			fmt.Printf("(s *Service) RunAsk | onConnect to %s:%d\n", ip, port)
		},
	})

	if err != nil {
		// fmt.Printf("Error: %+v\n", err)
		logger.Error("BindError: %+v", err)
		return
	}

	mgr := NewMgr()
	tcp0Asker := asker.(*ask.Tcp0Asker)
	tcp0Asker.SetWorkHandler(mgr.Handler)

	// fmt.Printf("(s *Service) RunAsk | 伺服器初始化完成\n")
	logger.Debug("伺服器初始化完成")

	err = gos.StartConnect()

	if err != nil {
		// fmt.Printf("Error: %+v\n", err)
		logger.Error("ConnectError: %+v", err)
		return
	}

	// fmt.Printf("(s *Service) RunAsk | 開始連線\n")
	logger.Debug("開始連線")

	var start time.Time
	mgr.FrameTime = 200 * time.Millisecond
	var during time.Duration = 0
	// fmt.Printf("(s *Service) RunAsk | 開始 gos.RunAsk()\n")
	logger.Debug("開始 gos.RunAsk()")

	for {
		start = time.Now()

		gos.RunAsk()
		mgr.Run()

		during = time.Since(start)
		if during < mgr.FrameTime {
			time.Sleep(mgr.FrameTime - during)
		}
	}
}
