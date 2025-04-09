package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/gomcp/codec"
	"github.com/gomcp/logger"
	"github.com/gomcp/mcp"
	"github.com/gomcp/types"

	"github.com/google/uuid"
)

// SSEMCPClient implements the MCPClient using Server-Sent Events (SSE).
type SSEMCPClient struct {
	mu         sync.Mutex
	log        *logger.Logger
	serverURL  string
	initURL    string
	clientID   string
	httpClient *http.Client
	handlers   map[string]chan json.RawMessage
	contexts   map[string]*types.Context
	state      ClientState
}

func NewSSEMCPClient(serverURL, clientID string) *SSEMCPClient {
	return &SSEMCPClient{
		log:       logger.NewLogger("SSEMCPClient", uuid.NewString()),
		serverURL: serverURL,
		initURL:   "CHANGEME",
		clientID:  clientID,
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
		handlers: make(map[string]chan json.RawMessage),
		contexts: make(map[string]*types.Context),
		state:    NewClientState("CHANGEME"),
	}
}

func (c *SSEMCPClient) Send(request any) error {
	body, err := json.Marshal(request)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", c.serverURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-ID", c.clientID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}
	return nil
}

// MCP method integrations
func (c *SSEMCPClient) HandleMCPNotification(method string, raw json.RawMessage) error {
	switch method {
	case "context/update":
		return c.handleContextUpdate(raw)
	default:
		return fmt.Errorf("unsupported notification method: %s", method)
	}
}

func (c *SSEMCPClient) handleContextUpdate(raw json.RawMessage) error {
	var update types.ContextUpdate
	if err := json.Unmarshal(raw, &update); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	ctx, ok := c.contexts[c.clientID]
	if !ok {
		ctx = types.NewContext(c.clientID, nil)
		c.contexts[c.clientID] = ctx
	}

	ctx.ApplyUpdate(update)
	return nil
}

func (c *SSEMCPClient) GetClientContext() *types.Context {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.contexts[c.clientID]
}

func (c *SSEMCPClient) AppendAssistantResponse(content string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ctx, ok := c.contexts[c.clientID]; ok {
		ctx.ApplyUpdate(types.ContextUpdate{
			ID: ctx.ID,
			Append: []types.MemoryBlock{{
				Role:    "assistant",
				Content: content,
				Time:    time.Now(),
			}},
		})
	}
}

// ============= Client State ================

type ClientState struct {
	initURL           string
	SupportedVersions []string
	Info              mcp.ClientInfo
	Capabilities      mcp.ClientCapabilities
	NegotiatedVersion string
	ServerCaps        *mcp.ServerCapabilities
	ServerInfo        *mcp.ServerInfo
	Initialized       bool
	log               *logger.Logger
	httpClient        http.Client
}

func NewClientState(initUrl string) ClientState {
	return ClientState{
		initURL:           initUrl,
		SupportedVersions: make([]string, 0),
		Info:              mcp.NewClientInfo(),
		Capabilities:      mcp.NewClientCapabilities(),
		NegotiatedVersion: codec.DefaultProtocolVersion,
		Initialized:       false,
		log:               logger.NewLogger("CLIENT STATE", uuid.NewString()),
	}
}

// Handshake methods

func (cs *ClientState) CreateInitializeRequest(requestID any) ([]byte, error) {
	if len(cs.SupportedVersions) == 0 {
		return nil, errors.New("client must support at least one protocol version")
	}
	// Client SHOULD offer the latest version it supports
	offeredVersion := cs.SupportedVersions[len(cs.SupportedVersions)-1]

	params := mcp.InitializeParams{
		ProtocolVersion: offeredVersion,
		Capabilities:    cs.Capabilities,
		ClientInfo:      cs.Info,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal initialize params: %w", err)
	}

	req := codec.JSONRPCRequest{
		JSONRPC: codec.JsonRPCVersion,
		ID:      requestID,
		Method:  string(mcp.MethodInitialize),
		Params:  paramsJSON,
	}

	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal initialize request: %w", err)
	}
	cs.log.Info(fmt.Sprintf("CLIENT: Sending Initialize Request (ID: %v): %s\n", requestID, string(reqJSON)))
	return reqJSON, nil
}

func (cs *ClientState) SendInitRequest(request any) ([]byte, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", cs.initURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := cs.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var data bytes.Buffer
	_, err = io.Copy(&data, resp.Body)
	if err != nil {
		return nil, err
	}

	return data.Bytes(), nil
}

func (cs *ClientState) ProcessInitializeResponse(respJSON []byte) error {
	var resp codec.JSONRPCResponse
	if err := json.Unmarshal(respJSON, &resp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}
	cs.log.Info(fmt.Sprintf("CLIENT: Received Response (ID: %v): %s\n", resp.ID, string(respJSON)))

	// Basic validation (ID matching should be done by the caller comparing with sent request ID)
	if resp.Error != nil {
		return fmt.Errorf("server returned error: code=%d, message=%s", resp.Error.Code, resp.Error.Message)
	}

	if resp.Result == nil {
		return errors.New("server response missing 'result' field")
	}

	var result mcp.InitializeResult
	if err := json.Unmarshal(resp.Bytes(), &result); err != nil {
		return fmt.Errorf("failed to unmarshal initialize result: %w", err)
	}

	// --- Version Negotiation ---
	serverVersion := result.ProtocolVersion
	versionSupported := slices.Contains(cs.SupportedVersions, serverVersion)

	// Client does not support the version server responded with.
	if !versionSupported {
		cs.Initialized = false
		cs.log.Error(fmt.Sprintf("CLIENT: Server responded with unsupported version '%s'. Supported: %v. Disconnecting.\n", serverVersion, cs.SupportedVersions))
		return fmt.Errorf("unsupported protocol version '%s' from server", serverVersion)
	}

	// Version is supported!
	cs.NegotiatedVersion = serverVersion
	cs.ServerCaps = &result.Capabilities
	cs.ServerInfo = &result.ServerInfo
	cs.log.Info(fmt.Sprintf("Handshake successful! Negotiated Version: %s\n", cs.NegotiatedVersion))
	cs.log.Info(fmt.Sprintf("Server Info: %+v\n", *cs.ServerInfo))
	cs.log.Info(fmt.Sprintf("Server Capabilities: %+v\n", *cs.ServerCaps)) // Might need better printing for nested structs

	return nil
}

func (cs *ClientState) CreateInitializedNotification() ([]byte, error) {
	if cs.NegotiatedVersion == "" || cs.ServerInfo == nil {
		return nil, errors.New("cannot send initialized notification before successful handshake")
	}

	noti := codec.Notification{
		JSONRPC: codec.JsonRPCVersion,
		Method:  "notifications/initialized",
		Params:  nil, // No params specified for this notification
	}

	notiJSON, err := json.Marshal(noti)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal initialized notification: %w", err)
	}

	cs.log.Info(fmt.Sprintf("Sending Initialized Notification: %s\n", string(notiJSON)))
	cs.Initialized = true
	return notiJSON, nil
}

func (cs *ClientState) SendInitNotification(notification []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, cs.initURL, bytes.NewReader((notification)))
	if err != nil {
		return nil, err
	}

	resp, err := cs.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send init notification: %s", err)
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response body: %s", err)
	}

	return buf.Bytes(), nil
}
