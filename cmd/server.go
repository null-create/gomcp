package cmd

import (
	"github.com/gomcp/server"

	"github.com/spf13/cobra"
)

var (
	serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Start the mcp server. Stop with CTRL-C.",
		Run:   runServerCmd,
	}
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

func runServerCmd(cmd *cobra.Command, args []string) {
	svr := server.NewServer()
	svr.Run()
}
