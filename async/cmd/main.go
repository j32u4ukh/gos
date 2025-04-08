package main

import (
	"fmt"
	"os"

	"github.com/j32u4ukh/gos/async/cmd/tcp_server"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{}
	tcp_server.RegisterCommand(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
