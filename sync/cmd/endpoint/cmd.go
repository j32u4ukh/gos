package endpoint

import (
	"fmt"
	"time"

	"github.com/j32u4ukh/gos/define"
	"github.com/j32u4ukh/gos/sync/gos"
	"github.com/j32u4ukh/gos/sync/gos/ans"
	"github.com/j32u4ukh/gos/utils/log"
	"github.com/spf13/cobra"
)

// go run . endpoint -p 5000
func RegisterCommand(rootCmd *cobra.Command) {
	taskCmd := &cobra.Command{
		Use: "endpoint",
		Run: func(cmd *cobra.Command, args []string) {
			err := log.SetLogger("endpoint", "log", log.DEBUG)
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
			fmt.Printf("port: %d\n", port)
			anser, err := gos.Listen(define.Http, port)
			log.Debug("Listen to port %d", port)

			if err != nil {
				log.Error("ListenError: %+v", err)
				return
			}

			httpAnswer := anser.(*ans.HttpAnser)
			mgr := &Mgr{}
			mgr.HttpAnswer = httpAnswer
			mgr.Handler(httpAnswer.Router)
			log.Debug("伺服器初始化完成")

			gos.StartListen()
			log.Debug("開始監聽")

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
		},
	}
	rootCmd.AddCommand(taskCmd)
}
