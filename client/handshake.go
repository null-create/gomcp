package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/gomcp/codec"
)

// Initial MCP handshake with server.
//
// https://spec.modelcontextprotocol.io/specification/2024-11-05/basic/lifecycle/#initialization
func (c *MCPClient) Handshake() error {
	if c.state == nil {
		c.state = NewClientState(c.initURL.String())
	}

	initReqJSON, err := c.state.CreateInitializeRequest()
	if err != nil {
		return fmt.Errorf("client failed to create initialize request: %v", err)
	}

	initRespJSON, err := c.state.SendInitRequest(initReqJSON)
	if err != nil {
		return fmt.Errorf("client init request failed: %s", err)
	}

	// Check if initRespJSON contains a JSON-RPC error response
	var jsonRpcResponse codec.JSONRPCResponse
	if json.Unmarshal(initRespJSON, &jsonRpcResponse) == nil && jsonRpcResponse.Error != nil {
		return fmt.Errorf("server returned JSON-RPC error: %+v", jsonRpcResponse.Error)
	}
	if err := c.state.ProcessInitializeResponse(jsonRpcResponse); err != nil {
		return fmt.Errorf("client failed to process initialize response: %v", err)
	}

	if c.state.GetNegotiatedVersion() != "" && c.state.HasServerInfo() { // Check if handshake was successful
		initializedNotiJSON, err := c.state.CreateInitializedNotification()
		if err != nil {
			return fmt.Errorf("client failed to create initialized notification: %v", err)
		}

		_, err = c.state.SendInitNotification(initializedNotiJSON)
		if err != nil {
			return fmt.Errorf("client failed to send init notification: %s", err)
		}

		//  Handshake Complete
		log.Println("-------------------------------------")
		log.Printf("HANDSHAKE COMPLETE: Client Initialized: %v\n", c.state.IsInitialized())
		log.Printf("Negotiated Protocol Version: %s\n", c.state.GetNegotiatedVersion())
		log.Println("-------------------------------------")
		return nil
	} else {
		return errors.New("client handshake failed. no negotiated version or server info retrieved")
	}
}
