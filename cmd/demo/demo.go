package demo

import (
	"fmt"

	"github.com/j32u4ukh/gos/log"
	"github.com/j32u4ukh/gos/server/ghttp"
	"github.com/spf13/cobra"
)

func RegisterCommand(rootCmd *cobra.Command) {
	taskCmd := &cobra.Command{
		Use: "demo",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("demo")
			err := log.SetLogger("demo", "log", log.DEBUG)
			if err != nil {
				fmt.Printf("設置日誌失敗, err: %+v\n", err)
				return
			}
			request := ghttp.NewRequest()
			err = request.SetRequest(ghttp.METHOD_POST, "http://example.com/user/get", nil)
			if err != nil {
				fmt.Printf("err: %+v\n", err)
				return
			}
			request.SetJson(map[string]any{
				"id":    25,
				"value": "json body",
			})
			fmt.Printf("request:\n%+v\n", request)
		},
	}
	rootCmd.AddCommand(taskCmd)
}

func ModifyIntPtr(iptr *int, value int) {
	(*iptr) += value
}
