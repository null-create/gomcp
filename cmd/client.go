package cmd

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/gomcp/client"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

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

func getURLS() (*url.URL, *url.URL, error) {
	sURL := os.Getenv("GOMCP_SERVER_URL")
	iURL := os.Getenv("GOMCP_INIT_URL")
	if sURL == "" || iURL == "" {
		return nil, nil, fmt.Errorf("both GOMCP_SERVER_URL and GOMCP_INIT_URL env vars must be set")
	}

	serverURL, err := url.Parse(sURL)
	if err != nil {
		return nil, nil, err
	}
	initURL, err := url.Parse(iURL)
	if err != nil {
		return nil, nil, err
	}

	return serverURL, initURL, nil
}

func runClientCmd(cmd *cobra.Command, args []string) {
	serverURL, initURL, err := getURLS()
	if err != nil {
		log.Fatalf("failed to get server urls: %v", err)
	}

	client := client.NewMCPClient(serverURL, initURL, uuid.NewString())
	if err := client.Start(context.Background()); err != nil {
		log.Fatalf("failed to start mcp client: %v", err)
	}
}
