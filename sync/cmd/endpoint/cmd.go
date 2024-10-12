package endpoint

import (
	"fmt"
	"time"

	"github.com/j32u4ukh/glog/v2"
	"github.com/j32u4ukh/gos/define"
	"github.com/j32u4ukh/gos/sync/gos"
	"github.com/j32u4ukh/gos/sync/gos/ans"
	"github.com/j32u4ukh/gos/utils"
	"github.com/spf13/cobra"
)

var logger *glog.Logger

func init() {
	gosLgger := glog.SetLogger(0, "gos", glog.DebugLevel)
	gosLgger.SetOptions(glog.DefaultOption(true, true), glog.UtcOption(8))
	gosLgger.SetFolder("log")
	gosLgger.SetSkip(3)
	utils.SetLogger(gosLgger)

	logger = glog.SetLogger(1, "DemoEndpoint", glog.DebugLevel)
	logger.SetFolder("log")
	logger.SetOptions(glog.DefaultOption(true, true), glog.UtcOption(8))
}

// go run . endpoint -p 5000
func RegisterCommand(rootCmd *cobra.Command) {
	taskCmd := &cobra.Command{
		Use: "endpoint",
		Run: func(cmd *cobra.Command, args []string) {
			port, err := cmd.Flags().GetInt32("port")
			if err != nil {
				fmt.Printf("Failed to get string kind, err: %+v", err)
				return
			}
			fmt.Printf("port: %d\n", port)
		},
	}
	rootCmd.AddCommand(taskCmd)
}

func RunAns(port int32) {
	anser, err := gos.Listen(define.Http, port)
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
