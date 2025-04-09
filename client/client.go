package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gomcp/logger"
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
