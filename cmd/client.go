package cmd

import "github.com/spf13/cobra"

var (
	clientCmd = &cobra.Command{
		Use:   "client",
		Short: "start the mcp client",
		Run:   runClientCmd,
	}
)

func init() {
	rootCmd.AddCommand(clientCmd)
}

func runClientCmd(cmd *cobra.Command, args []string) {

}
