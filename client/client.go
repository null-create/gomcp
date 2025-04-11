package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gomcp/codec"
	mcpctx "github.com/gomcp/context"
	"github.com/gomcp/logger"
	"github.com/gomcp/types"
)

// MCPClient implements the MCPClient using Server-Sent Events (SSE).
type MCPClient struct {
	mu         sync.Mutex
	log        *logger.Logger
	serverURL  string
	initURL    string
	clientID   string
	httpClient *http.Client
	handlers   map[string]chan json.RawMessage
	contexts   map[string]*mcpctx.Context
	state      types.Initializer
}

// Initializes a new Client. Must be followed by a call to client.Handshake()
// to establish client state.
func NewMCPClient(serverURL, initURL, clientID string) *MCPClient {
	return &MCPClient{
		log:       logger.NewLogger("MCPClient", clientID),
		serverURL: serverURL,
		initURL:   initURL,
		clientID:  clientID,
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
		handlers: make(map[string]chan json.RawMessage),
		contexts: make(map[string]*mcpctx.Context),
	}
}

// Send JSONRPC requests to the server
func (c *MCPClient) Send(data codec.JSONRPCRequest) error {
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
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("received non-204 response: %d", resp.StatusCode)
	}
	return nil
}

// Listen for and handle server-sent events using a provided handler
func (c *MCPClient) Listen(ctx context.Context, handler types.MessageHandler) error {
	url := fmt.Sprintf("%s?id=%s", c.serverURL, c.clientID)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Accept", "text/event-stream")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("client connection error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("received non-200 return code: %d", resp.StatusCode)
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				resp.Body.Close()
				return nil
			default:
			}

			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				msg := strings.TrimPrefix(line, "data: ")
				if err := handler(json.RawMessage(msg)); err != nil {
					return err
				}
			}
		}

		if err := scanner.Err(); err != nil {
			return err
		}
		scanner = nil
		resp.Body.Close()
		time.Sleep(2 * time.Second)
	}
}
