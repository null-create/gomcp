package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "display version",
		Run:   showVersion,
	}
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

func showVersion(cmd *cobra.Command, args []string) {
	b, err := os.ReadFile("VERSION")
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatal("VERSION file not found")
		} else {
			log.Fatal(err)
		}
	}
	fmt.Print("Version: " + string(b))
}
