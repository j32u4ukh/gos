package httpserver

import (
	"fmt"

	"github.com/j32u4ukh/gos/log"
	"github.com/spf13/cobra"
)

func RegisterCommand(rootCmd *cobra.Command) {
	taskCmd := &cobra.Command{
		Use: "http",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("HTTP server")
			kind, err := cmd.Flags().GetString("kind")
			if err != nil {
				return
			}
			fmt.Printf("kind: %s\n", kind)
			switch kind {
			case "ans":
				AnserDemo(args)
			case "ask":
				AskerDemo(args)
			default:
				fmt.Printf("Undefined kind: %s\n", kind)
			}
		},
	}
	taskCmd.Flags().StringP("kind", "k", "ans", "Anser or Asker")
	rootCmd.AddCommand(taskCmd)
}

func AnserDemo(args []string) {
	err := log.SetLogger("tcp-ans", "log", log.DEBUG)
	if err != nil {
		fmt.Printf("設置日誌失敗, err: %+v\n", err)
		return
	}
	defer func() {
		err = log.Close()
		if err != nil {
			fmt.Printf("關閉日誌時發生錯誤, err: %+v\n", err)
			return
		}
	}()
	var mgr *AnserManager = NewAnserManager()
	var port int32 = 5000
	err = mgr.Init(port, 10)
	if err != nil {
		fmt.Printf("監聽 port %d 失敗\n", port)
		return
	}
	mgr.Run()
}

func AskerDemo(args []string) {
	// err := log.SetLogger("tcp-ask", "log", log.DEBUG)
	// if err != nil {
	// 	fmt.Printf("設置日誌失敗, err: %+v\n", err)
	// 	return
	// }
	// defer func() {
	// 	err = log.Close()
	// 	if err != nil {
	// 		fmt.Printf("關閉日誌時發生錯誤, err: %+v\n", err)
	// 		return
	// 	}
	// }()
	// var mgr *Manager = NewManager()
	// var port int32 = 1023
	// err = mgr.InitAsker(port, 10)
	// if err != nil {
	// 	fmt.Printf("127.0.0.1:%d 連線失敗\n", port)
	// 	return
	// }
	// mgr.Run("ask")
}
