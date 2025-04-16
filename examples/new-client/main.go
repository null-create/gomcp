package examples

import (
	"context"
	"log"
	"net/url"
	"os"

	gomcp "github.com/gomcp/client"

	"github.com/google/uuid"
)

// create and start a new MCP client
func CreateMCPClient() {
	serverURL, _ := url.Parse(os.Getenv("MCP_SERVER_URL"))
	initURL, _ := url.Parse(os.Getenv("MCP_INIT_URL"))

	c := gomcp.NewMCPClient(serverURL, initURL, uuid.NewString())

	if err := c.Start(context.Background()); err != nil {
		log.Fatal(err)
	}
}
