package client

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gomcp/codec"
	"github.com/gomcp/logger"
	"github.com/gomcp/mcp"

	"github.com/google/uuid"
)

// Initial MCP handshake with server
func (s *SSEMCPClient) Handshake() error {
	cs := ClientState{
		initURL:           s.initURL,
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
		log.Fatalf("Client failed to create initialize request: %v", err)
	}

	initRespJSON, err := cs.SendInitRequest(initReqJSON)
	if err != nil {
		log.Fatalf("Client init request failed: %s", err)
	}

	// Check if initRespJSON contains a JSON-RPC error response
	var potentialErrorResp codec.JSONRPCResponse
	if json.Unmarshal(initRespJSON, &potentialErrorResp) == nil && potentialErrorResp.Error != nil {
		log.Fatalf("Server returned JSON-RPC error: %+v", potentialErrorResp.Error)
	}
	if err := cs.ProcessInitializeResponse(initRespJSON); err != nil {
		log.Fatalf("Client failed to process initialize response: %v", err)
	}

	if cs.NegotiatedVersion != "" && cs.ServerInfo != nil { // Check if handshake was successful
		initializedNotiJSON, err := cs.CreateInitializedNotification()
		if err != nil {
			log.Fatalf("Client failed to create initialized notification: %v", err)
		}

		_, err = cs.SendInitNotification(initializedNotiJSON)
		if err != nil {
			log.Fatalf("Client failed to send init notification: %s", err)
		}
		s.state = cs

		// --- Handshake Complete ---
		log.Println("-------------------------------------")
		log.Printf("HANDSHAKE COMPLETE: Client Initialized: %v\n", cs.Initialized)
		log.Printf("Negotiated Protocol Version: %s\n", cs.NegotiatedVersion)
		log.Println("-------------------------------------")

	} else {
		log.Fatalf("Client handshake failed. Cannot send initialized notification.")
	}

	return nil
}
