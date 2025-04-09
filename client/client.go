package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gomcp/codec"
	"github.com/gomcp/context"
	"github.com/gomcp/logger"

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
	contexts   map[string]*context.Context
	state      ClientState
}

func NewSSEMCPClient(serverURL, initURL, clientID string) *SSEMCPClient {
	return &SSEMCPClient{
		log:       logger.NewLogger("SSEMCPClient", uuid.NewString()),
		serverURL: serverURL,
		initURL:   initURL,
		clientID:  clientID,
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
		handlers: make(map[string]chan json.RawMessage),
		contexts: make(map[string]*context.Context),
	}
}

// Send RPC request to the server
func (c *SSEMCPClient) Request(data codec.JSONRPCRequest) error {
	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.serverURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Client-ID", c.clientID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Error("failed to send request: " + err.Error())
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		c.log.Warn(fmt.Sprintf("received non-204 response: %d", resp.StatusCode))
		return fmt.Errorf("received non-204 response: %s", resp.Status)
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
	var update context.ContextUpdate
	if err := json.Unmarshal(raw, &update); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	ctx, ok := c.contexts[c.clientID]
	if !ok {
		ctx = context.NewContext(c.clientID, nil)
		c.contexts[c.clientID] = ctx
	}

	ctx.ApplyUpdate(update)
	return nil
}

func (c *SSEMCPClient) GetClientContext() *context.Context {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.contexts[c.clientID]
}

func (c *SSEMCPClient) AppendAssistantResponse(content string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ctx, ok := c.contexts[c.clientID]; ok {
		ctx.ApplyUpdate(context.ContextUpdate{
			ID: ctx.ID,
			Append: []context.MemoryBlock{{
				Role:    "assistant",
				Content: content,
				Time:    time.Now(),
			}},
		})
	}
}
