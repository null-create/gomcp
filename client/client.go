package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	done       chan struct{}
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

func (c *MCPClient) Start(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.serverURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to SSE stream: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// start listening for server-side events
	go c.listen(ctx, resp.Body, nil)

	select {
	// case <-c.endpointChan:
	// 	// Endpoint received, proceed
	case <-ctx.Done():
		return fmt.Errorf("context cancelled while waiting for endpoint")
	case <-time.After(30 * time.Second): // Add a timeout
		return fmt.Errorf("timeout waiting for endpoint")
	}

	// return nil
}

// Continually listens for server-side events using the given reader.
// Processes events with the given handler.
func (c *MCPClient) listen(ctx context.Context, reader io.ReadCloser, handler types.MessageHandler) error {
	defer reader.Close()

	br := bufio.NewReader(reader)
	var event, data string

	for {
		select {
		case <-c.done:
			return nil
		case <-ctx.Done():
			return nil
		default:
			line, err := br.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					// Process any pending event before exit
					if event != "" && data != "" {
						if err := handler(json.RawMessage(data)); err != nil {
							return err
						}
					}
					break
				}
				select {
				case <-c.done:
					return nil
				default:
					fmt.Printf("SSE stream error: %v\n", err)
					return nil
				}
			}

			// Remove only newline markers
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				// Empty line means end of event
				if event != "" && data != "" {
					if err := handler(json.RawMessage(data)); err != nil {
						return err
					}
					event = ""
					data = ""
				}
				continue
			}

			if strings.HasPrefix(line, "event:") {
				event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			} else if strings.HasPrefix(line, "data:") {
				data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			}
		}
	}
}
