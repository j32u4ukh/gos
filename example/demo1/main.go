package main

import (
	"fmt"
	"gos"
	"gos/define"
	"os"
	"strconv"
	"time"
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
	anser, err := gos.Listen(int32(port), define.Tcp0)
	fmt.Printf("(s *Service) RunAns | Listen to port %d\n", port)

	if err != nil {
		fmt.Printf("Error: %+v\n", err)
		return
	}

	mgr := &Mgr{}
	anser.SetWorkHandler(mgr.Handler)
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
	asker.SetWorkHandler(mgr.Handler)
	fmt.Printf("(s *Service) RunAsk | 伺服器初始化完成\n")
	err = gos.StartConnect()

	if err != nil {
		fmt.Printf("Error: %+v\n", err)
		return
	}

	fmt.Printf("(s *Service) RunAsk | 開始連線\n")
	var start time.Time
	var during, frameTime time.Duration = 0, 200 * time.Millisecond

	// TODO: 暫緩一般數據傳送，先實作心跳包機制
	go func() {
		time.Sleep(2 * time.Second)
		fmt.Printf("(s *Service) RunAsk | After 2 Second.\n")
		var data []byte
		temp := []byte{}

		for i := 50; i < 55; i++ {
			temp = append(temp, byte(i))
			// fmt.Printf("(s *Service) RunAsk | i: %d, temp: %+v\n", i, temp)
			mgr.Body.AddByte(1)
			mgr.Body.AddUInt16(0)
			mgr.Body.AddByteArray(temp)
			data = mgr.Body.FormData()
			// fmt.Printf("(s *Service) RunAsk | i: %d, length: %d, data: %+v\n", i, len(data), data)
			gos.SendToServer(0, &data, int32(len(data)))
			time.Sleep(1 * time.Second)
			mgr.Body.Clear()
		}

		time.Sleep(5 * time.Second)
		fmt.Printf("(s *Service) RunAsk | After 5 Second.\n")

		for i := 55; i < 60; i++ {
			temp = append(temp, byte(i))
			// fmt.Printf("(s *Service) RunAsk | i: %d, temp: %+v\n", i, temp)
			mgr.Body.AddByte(1)
			mgr.Body.AddUInt16(0)
			mgr.Body.AddByteArray(temp)
			data = mgr.Body.FormData()
			// fmt.Printf("(s *Service) RunAsk | i: %d, length: %d, data: %+v\n", i, len(data), data)
			gos.SendToServer(0, &data, int32(len(data)))
			time.Sleep(1 * time.Second)
			mgr.Body.Clear()
		}
	}()

	fmt.Printf("(s *Service) RunAsk | 開始 gos.RunAsk()\n")

	for {
		start = time.Now()

		gos.RunAsk()

		during = time.Since(start)
		if during < frameTime {
			time.Sleep(frameTime - during)
		}
	}
}
