package client

import (
	"errors"
	"fmt"
)

func (c *MCPClient) handleSSE(event, data string) error {
	switch event {
	case "endpoint":
		endpoint, err := c.serverURL.Parse(data)
		if err != nil {
			c.log.Error(fmt.Sprintf("error parsing endpoint URL: %v", err))
			return fmt.Errorf("error parsing endpoint URL: %v", err)
		}
		if endpoint.Host != c.serverURL.Host {
			c.log.Error("endpoint origin does not match connection origin")
			return errors.New("endpoint origin does not match connection origin")
		}
		c.serverURL = endpoint
		close(c.endpointChan)
	case "message":

	default:
		c.log.Warn(fmt.Sprintf("unknown server event: %s", event))
	}
	return nil
}
