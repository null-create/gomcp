package cmd

import "github.com/spf13/cobra"

var (
	clientCmd = &cobra.Command{
		Use:   "server",
		Short: "start the mcp server",
		Run:   runClientCmd,
	}
)

func init() {
	rootCmd.AddCommand(clientCmd)
}

func runClientCmd(cmd *cobra.Command, args []string) {

}
