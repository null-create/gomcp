package client

import (
	"fmt"
)

// file for client-side sse handlers

func (c *MCPClient) handleSSE(event, data string) error {
	switch event {
	case "endpoint":
		endpoint, err := c.serverURL.Parse(data)
		if err != nil {
			c.log.Error(fmt.Sprintf("error parsing endpoint URL: %v\n", err))
			return err
		}
		if endpoint.Host != c.serverURL.Host {
			c.log.Error("endpoint origin does not match connection origin")
			return nil
		}
		c.serverURL = endpoint
		close(c.endpointChan)
	case "message":

	default:
		c.log.Warn(fmt.Sprintf("unknown server event: %s", event))
	}
	return nil
}
