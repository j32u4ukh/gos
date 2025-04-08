package pipeline

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/j32u4ukh/gos/utils/log"

	"github.com/j32u4ukh/glog/v2"
	"github.com/j32u4ukh/gos/define"
	"github.com/j32u4ukh/gos/sync/gos"
	"github.com/j32u4ukh/gos/sync/gos/ans"
	"github.com/j32u4ukh/gos/sync/gos/ask"
	"github.com/j32u4ukh/gos/sync/gos/base"
	"github.com/j32u4ukh/gos/sync/gos/base/ghttp"
	"github.com/spf13/cobra"
)

const ERandomReturnServer int32 = 0

var logger *glog.Logger

// go run . pipeline -p 5000
func RegisterCommand(rootCmd *cobra.Command) {
	taskCmd := &cobra.Command{
		Use: "pipeline",
		Run: func(cmd *cobra.Command, args []string) {
			err := log.SetLogger("pipeline", "log", log.DEBUG)
			if err != nil {
				fmt.Printf("取得 Logger 時發生錯誤, err: %+v\n", err)
				return
			}
			defer func() {
				if r := recover(); r != nil {
					log.Error("發生非預期錯誤, err: %+v\n", r)
				}
				err = log.Close()
				if err != nil {
					fmt.Printf("關閉 Logger 時發生錯誤, err: %+v\n", err)
				}
			}()
			log.SetSkip(3)
			port, err := cmd.Flags().GetInt32("port")
			if err != nil {
				fmt.Printf("Failed to get string kind, err: %+v", err)
				return
			}
			kind, err := cmd.Flags().GetString("kind")
			if err != nil {
				fmt.Printf("Failed to get string kind, err: %+v", err)
				return
			}
			fmt.Printf("port: %d, kind: %s\n", port, kind)
			switch kind {
			case "ms":
				RunMainServer(port)
			case "ask":
				RunAsk(port)
			case "rrs":
				RunRandomReturnServer(port)
			default:
			}
		},
	}
	rootCmd.AddCommand(taskCmd)
}

// MainServer 接受客戶端 http 請求，再將請求發送到 RandomReturnServer 做處理，RandomReturnServer 將結果返還 MainServer，再由 MainServer 回覆客戶端
func RunMainServer(port int32) {
	anser, err := gos.Listen(define.Http, port)
	logger.Debug("Listen to port %d", port)
	if err != nil {
		logger.Error("Listen error: %+v", err)
		return
	}
	mgr := &Mgr{}
	httpAnswer := anser.(*ans.HttpAnser)
	mgr.HttpAnswer = anser.(*ans.HttpAnser)
	mgr.HttpHandler(httpAnswer.Router)
	logger.Debug("Http Anser 伺服器初始化完成")

	td := base.NewTransData()
	td.AddInt32(0)
	td.AddInt32(0)
	heartbeat := td.FormData()

	asker, err := gos.Bind(ERandomReturnServer, "127.0.0.1", 1022, define.Tcp0, base.OnEventsFunc{
		define.OnConnected: func(any) {
			fmt.Printf("(s *Service) RunAsk | onConnect to %s:%d\n", "127.0.0.1", port)
		},
	}, nil, &heartbeat)

	if err != nil {
		logger.Error("Bind error: %+v", err)
		return
	}

	tcp0Asker := asker.(*ask.Tcp0Asker)
	tcp0Asker.SetWorkHandler(mgr.RandomReturnServerHandler)
	logger.Debug("RandomReturnServer Asker 伺服器初始化完成")
	logger.Debug("伺服器初始化完成")

	// =============================================
	// 開始所有已註冊的監聽
	// =============================================
	gos.StartListen()
	logger.Debug("開始監聽")

	err = gos.StartConnect()

	if err != nil {
		logger.Error("與 RandomReturnServer 連線時發生錯誤, error: %+v", err)
		return
	}

	logger.Debug("成功與 RandomReturnServer 連線")
	gos.SetFrameTime(20 * time.Millisecond)
	gos.Run(nil)
}

func RunAsk(port int32) {
	method := "GET"
	url := fmt.Sprintf("http://192.168.0.198:%d", port)
	payload := strings.NewReader(`{"client_message": "hello, server!"}`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		logger.Error("error: %+v", err)
		return
	}

	req.Header.Add(ghttp.HeaderContentType, "application/json")
	res, err := client.Do(req)

	if err != nil {
		logger.Error("error: %+v", err)
		return
	}

	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		logger.Error("error: %+v", err)
		return
	}
	logger.Info("Response: %s", string(body))
}

func RunRandomReturnServer(port int32) {
	anser, err := gos.Listen(define.Tcp0, port)
	logger.Debug("Listen to port %d", port)

	if err != nil {
		logger.Error("Listen error: %+v", err)
		return
	}

	rrs := &RandomReturnServer{}
	tcpAnser := anser.(*ans.Tcp0Anser)
	tcpAnser.SetWorkHandler(rrs.Handler)
	logger.Debug("伺服器初始化完成")

	// =============================================
	// 開始所有已註冊的監聽
	// =============================================
	gos.StartListen()
	logger.Debug("開始監聽")
	gos.SetFrameTime(20 * time.Millisecond)
	gos.Run(nil)
}
