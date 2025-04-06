package cmd

import "github.com/spf13/cobra"

var (
	serverCmd = &cobra.Command{
		Use:   "server",
		Short: "start the mcp server",
		Run:   runServerCmd,
	}
)

func init() {
	rootCmd.AddCommand(serverCmd)
}

func runServerCmd(cmd *cobra.Command, args []string) {

}
