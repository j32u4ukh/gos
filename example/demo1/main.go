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
	"github.com/j32u4ukh/gos/base"
	"github.com/j32u4ukh/gos/define"
	"github.com/j32u4ukh/gos/utils"
)

var logger *glog.Logger

func init() {
	gosLogger := glog.SetLogger(0, "gos", glog.DebugLevel)
	gosLogger.SetFolder("log")
	gosLogger.SetOptions(glog.DefaultOption(true, true), glog.UtcOption(8))
	gosLogger.SetSkip(3)
	utils.SetLogger(gosLogger)
	logger = glog.SetLogger(1, "Demo1", glog.DebugLevel)
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

	logger.Info("End of gos example.")
}

func (s *Service) Stop() {
	s.StopCh <- true
}

func (s *Service) RunAns(port int) {
	anser, err := gos.Listen(define.Tcp0, int32(port))
	logger.Debug("Listen to port %d", port)

	if err != nil {
		logger.Error("ListenError: %+v", err)
		return
	}

	mgr := &Mgr{}
	tcp0Answer := anser.(*ans.Tcp0Anser)
	tcp0Answer.SetWorkHandler(mgr.Handler)
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

func (s *Service) RunAsk(ip string, port int) {
	td := base.NewTransData()
	td.AddInt32(SystemCmd)
	td.AddInt32(IntroductionService)
	td.AddString("GOSS")
	td.AddInt32(37)
	introduction := td.FormData()
	asker, err := gos.Bind(0, ip, port, define.Tcp0, base.OnEventsFunc{
		define.OnConnected: func(any) {
			fmt.Printf("(s *Service) RunAsk | onConnect to %s:%d\n", ip, port)
		},
	}, &introduction)

	if err != nil {
		logger.Error("BindError: %+v", err)
		return
	}

	mgr := NewMgr()
	tcp0Asker := asker.(*ask.Tcp0Asker)
	tcp0Asker.SetWorkHandler(mgr.Handler)
	logger.Debug("伺服器初始化完成")

	err = gos.StartConnect()

	if err != nil {
		logger.Error("ConnectError: %+v", err)
		return
	}

	logger.Debug("開始連線")
	var start time.Time
	var during, frameTime time.Duration = 0, 200 * time.Millisecond

	go func() {
		time.Sleep(2 * time.Second)
		logger.Info("After 2 Second.")

		var data []byte
		temp := []byte{}

		for i := 50; i < 55; i++ {
			temp = append(temp, byte(i))
			// fmt.Printf("(s *Service) RunAsk | i: %d, temp: %+v\n", i, temp)
			mgr.Body.AddInt32(NormalCmd)
			mgr.Body.AddInt32(TimerService)
			mgr.Body.AddByteArray(temp)
			data = mgr.Body.FormData()
			// fmt.Printf("(s *Service) RunAsk | i: %d, length: %d, data: %+v\n", i, len(data), data)
			gos.SendToServer(0, &data, int32(len(data)))
			time.Sleep(1 * time.Second)
			mgr.Body.Clear()
		}

		time.Sleep(5 * time.Second)
		// fmt.Printf("(s *Service) RunAsk | After 5 Second.\n")
		logger.Info("After 5 Second.")

		for i := 55; i < 60; i++ {
			temp = append(temp, byte(i))
			// fmt.Printf("(s *Service) RunAsk | i: %d, temp: %+v\n", i, temp)
			mgr.Body.AddInt32(NormalCmd)
			mgr.Body.AddInt32(TimerService)
			mgr.Body.AddByteArray(temp)
			data = mgr.Body.FormData()
			// fmt.Printf("(s *Service) RunAsk | i: %d, length: %d, data: %+v\n", i, len(data), data)
			gos.SendToServer(0, &data, int32(len(data)))
			time.Sleep(1 * time.Second)
			mgr.Body.Clear()
		}
	}()

	logger.Info("開始 gos.RunAsk()")

	for {
		start = time.Now()

		gos.RunAsk()

		during = time.Since(start)
		if during < frameTime {
			time.Sleep(frameTime - during)
		}
	}
}
