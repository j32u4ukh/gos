package tcp_server

import (
	"fmt"

	"github.com/spf13/cobra"
)

func RegisterCommand(rootCmd *cobra.Command) {
	taskCmd := &cobra.Command{
		Use: "tcp",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("TCP server")
			kind, err := cmd.Flags().GetString("kind")
			if err != nil {
				return
			}
			switch kind {
			case "ans":
				AnserDemo(args)
			default:
				fmt.Printf("Undefined kind: %s\n", kind)
			}
		},
	}
	taskCmd.Flags().StringP("kind", "k", "ans", "Anser or Asker")
	rootCmd.AddCommand(taskCmd)
}

func AnserDemo(args []string) {
	// var port int32 = 1024
	// server, err := ans.NewTcpAnser(port, 10)
	// if err != nil {
	// 	fmt.Printf("監聽 port %d 失敗\n", port)
	// 	return
	// }
	// gos.Run(nil)
}
