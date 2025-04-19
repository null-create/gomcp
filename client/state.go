package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sync/atomic"
	"time"

	"github.com/gomcp/codec"
	"github.com/gomcp/logger"
	"github.com/gomcp/mcp"
	"github.com/gomcp/types"

	"github.com/google/uuid"
)

type ClientState struct {
	initURL           string
	SupportedVersions []string
	ClientInfo        mcp.ClientInfo
	Capabilities      mcp.ClientCapabilities
	NegotiatedVersion string
	ServerInfo        *mcp.ServerInfo
	ServerCaps        *mcp.ServerCapabilities
	Initialized       bool
	log               *logger.Logger
	httpClient        http.Client
	reqID             atomic.Int64
}

func NewClientState(initUrl string) *ClientState {
	return &ClientState{
		initURL:           initUrl,
		SupportedVersions: []string{"2024-11-05", "2025-03-26"}, // Client supports two versions, latest is 2025-03-26
		ClientInfo:        mcp.NewClientInfo("Client", "1.0.0"),
		Capabilities:      mcp.NewClientCapabilities(),
		ServerInfo:        new(mcp.ServerInfo),
		ServerCaps:        new(mcp.ServerCapabilities),
		httpClient: http.Client{
			Timeout: time.Second * 30,
		},
		log:   logger.NewLogger("CLIENT STATE", uuid.NewString()),
		reqID: atomic.Int64{},
	}
}

// Implements ClientState interface.

func (cs *ClientState) GetNegotiatedVersion() string       { return cs.NegotiatedVersion }
func (cs *ClientState) IsInitialized() bool                { return cs.Initialized }
func (cs *ClientState) HasServerInfo() bool                { return cs.ServerInfo != nil }
func (cs *ClientState) GetServerInfo() *mcp.ServerInfo     { return cs.ServerInfo }
func (cs *ClientState) SetNegotiatedVersion(v string)      { cs.NegotiatedVersion = v }
func (cs *ClientState) SetServerInfo(info *mcp.ServerInfo) { cs.ServerInfo = info }
func (cs *ClientState) SetInitialized(init bool)           { cs.Initialized = init }

// Set the MCP client state. Mainly used for testing.
func (c *MCPClient) SetClientState(state types.ClientState) { c.state = state }

// ======= Client State Handshake methods ====== //

func (cs *ClientState) CreateInitializeRequest() ([]byte, error) {
	if len(cs.SupportedVersions) == 0 {
		return nil, errors.New("client must support at least one protocol version")
	}
	// Client SHOULD offer the latest version it supports
	offeredVersion := cs.SupportedVersions[len(cs.SupportedVersions)-1]

	params := mcp.InitializeParams{
		ProtocolVersion: offeredVersion,
		Capabilities:    cs.Capabilities,
		ClientInfo:      cs.ClientInfo,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal initialize params: %w", err)
	}

	req := codec.JSONRPCRequest{
		JSONRPC: codec.JsonRPCVersion,
		ID:      cs.reqID.Add(1),
		Method:  string(mcp.MethodInitialize),
		Params:  paramsJSON,
	}

	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal initialize request: %w", err)
	}
	return reqJSON, nil
}

func (cs *ClientState) SendInitRequest(initRequest []byte) ([]byte, error) {
	return cs.send(initRequest)
}

func (cs *ClientState) ProcessInitializeResponse(resp codec.JSONRPCResponse) error {
	if resp.Error != nil {
		return fmt.Errorf("server returned error: code=%d, message=%s", resp.Error.Code, resp.Error.Message)
	}

	if resp.Result == nil {
		return errors.New("server response missing 'result' field")
	}

	var result mcp.InitializeResult
	if err := json.Unmarshal(resp.Bytes(), &result); err != nil {
		return fmt.Errorf("failed to unmarshal initialize result: %v", err)
	}

	// --- Version Negotiation ---
	serverVersion := result.ProtocolVersion
	versionSupported := slices.Contains(cs.SupportedVersions, serverVersion)

	// Client does not support the version server responded with.
	if !versionSupported {
		cs.Initialized = false
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
		Params:  codec.NotificationParams{}, // No params specified for this notification
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
	return cs.send(notification)
}

func (cs *ClientState) send(msg []byte) ([]byte, error) {
	body, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, cs.initURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := cs.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}
	defer resp.Body.Close()

	var data bytes.Buffer
	_, err = io.Copy(&data, resp.Body)
	if err != nil {
		return nil, err
	}
	return data.Bytes(), nil
}
