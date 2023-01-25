package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/j32u4ukh/gos"
	"github.com/j32u4ukh/gos/ans"
	"github.com/j32u4ukh/gos/ask"
	"github.com/j32u4ukh/gos/define"
)

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

	fmt.Println("[Example] Run | End of gos example.")
}

func (s *Service) Stop() {
	s.StopCh <- true
}

func (s *Service) RunAns(port int) {
	anser, err := gos.Listen(define.Tcp0, int32(port))
	fmt.Printf("(s *Service) RunAns | Listen to port %d\n", port)

	if err != nil {
		fmt.Printf("Error: %+v\n", err)
		return
	}

	mgr := &Mgr{}
	tcp0Answer := anser.(*ans.Tcp0Anser)
	tcp0Answer.SetWorkHandler(mgr.Handler)
	// anser.SetWorkHandler(mgr.Handler)
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

func (s *Service) RunAsk(ip string, port int) {
	asker, err := gos.Bind(0, ip, port, define.Tcp0)

	if err != nil {
		fmt.Printf("Error: %+v\n", err)
		return
	}

	mgr := NewMgr()
	tcp0Asker := asker.(*ask.Tcp0Asker)
	tcp0Asker.SetWorkHandler(mgr.Handler)
	fmt.Printf("(s *Service) RunAsk | 伺服器初始化完成\n")
	err = gos.StartConnect()

	if err != nil {
		fmt.Printf("Error: %+v\n", err)
		return
	}

	fmt.Printf("(s *Service) RunAsk | 開始連線\n")
	var start time.Time
	mgr.FrameTime = 200 * time.Millisecond
	var during time.Duration = 0
	fmt.Printf("(s *Service) RunAsk | 開始 gos.RunAsk()\n")

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
