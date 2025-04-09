package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gomcp/codec"
	"github.com/gomcp/logger"
	"github.com/gomcp/mcp"

	"github.com/google/uuid"
)

// Initial MCP handshake with server
func (c *MCPClient) Handshake() error {
	cs := ClientState{
		initURL:           c.initURL,
		SupportedVersions: []string{"2024-10-01", "2024-11-05"}, // Client supports two versions, latest is 2024-11-05
		Info: mcp.ClientInfo{
			Name:    "Client",
			Version: "1.0.0",
		},
		Capabilities: mcp.ClientCapabilities{
			Roots:    &mcp.RootCapabilities{ListChanged: true},
			Sampling: &mcp.SamplingCapabilities{}, // Indicate support with empty struct pointer
		},
		ServerCaps: &mcp.ServerCapabilities{},
		ServerInfo: &mcp.ServerInfo{},
		httpClient: http.Client{
			Timeout: time.Second * 30,
		},
		log: logger.NewLogger("CLIENT STATE", uuid.NewString()),
	}

	initReqJSON, err := cs.CreateInitializeRequest(uuid.NewString())
	if err != nil {
		return fmt.Errorf("client failed to create initialize request: %v", err)
	}

	initRespJSON, err := cs.SendInitRequest(initReqJSON)
	if err != nil {
		return fmt.Errorf("client init request failed: %s", err)
	}

	// Check if initRespJSON contains a JSON-RPC error response
	var potentialErrorResp codec.JSONRPCResponse
	if json.Unmarshal(initRespJSON, &potentialErrorResp) == nil && potentialErrorResp.Error != nil {
		return fmt.Errorf("server returned JSON-RPC error: %+v", potentialErrorResp.Error)
	}
	if err := cs.ProcessInitializeResponse(initRespJSON); err != nil {
		return fmt.Errorf("client failed to process initialize response: %v", err)
	}

	if cs.NegotiatedVersion != "" && cs.ServerInfo != nil { // Check if handshake was successful
		initializedNotiJSON, err := cs.CreateInitializedNotification()
		if err != nil {
			return fmt.Errorf("client failed to create initialized notification: %v", err)
		}

		_, err = cs.SendInitNotification(initializedNotiJSON)
		if err != nil {
			return fmt.Errorf("client failed to send init notification: %s", err)
		}
		c.state = cs

		// --- Handshake Complete ---
		log.Println("-------------------------------------")
		log.Printf("HANDSHAKE COMPLETE: Client Initialized: %v\n", cs.Initialized)
		log.Printf("Negotiated Protocol Version: %s\n", cs.NegotiatedVersion)
		log.Println("-------------------------------------")

	} else {
		return fmt.Errorf("client handshake failed. no negotiated version or server info retrieved")
	}

	return nil
}
