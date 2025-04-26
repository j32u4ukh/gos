package main

import (
	"fmt"
	"os"

	"github.com/j32u4ukh/gos/cmd/demo"
	"github.com/j32u4ukh/gos/cmd/httpserver"
	"github.com/j32u4ukh/gos/cmd/tcp_server"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{}
	demo.RegisterCommand(rootCmd)
	tcp_server.RegisterCommand(rootCmd)
	httpserver.RegisterCommand(rootCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
