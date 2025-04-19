package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gomcp/codec"
	mcpctx "github.com/gomcp/context"
	"github.com/gomcp/logger"
	"github.com/gomcp/mcp"
	"github.com/gomcp/types"
)

// MCPClient implements the MCPClient using Server-Sent Events (SSE).
type MCPClient struct {
	mu           sync.Mutex
	log          *logger.Logger
	serverURL    *url.URL
	initURL      *url.URL
	clientID     string
	requestID    atomic.Int64
	responses    map[int64]chan codec.JSONRPCResponse
	done         chan struct{}
	endpointChan chan struct{}
	initialized  bool
	httpClient   *http.Client
	headers      map[string]string
	handlers     map[string]chan json.RawMessage
	contexts     map[string]*mcpctx.Context
	state        types.ClientState
}

// Initializes a new Client. Must be followed by a call to client.Handshake()
// to establish client state.
func NewMCPClient(serverURL *url.URL, initURL *url.URL, clientID string) *MCPClient {
	return &MCPClient{
		log:          logger.NewLogger("MCPClient", clientID),
		serverURL:    serverURL,
		initURL:      initURL,
		clientID:     clientID,
		requestID:    atomic.Int64{},
		responses:    make(map[int64]chan codec.JSONRPCResponse),
		done:         make(chan struct{}),
		endpointChan: make(chan struct{}),
		initialized:  false,
		httpClient:   &http.Client{Timeout: time.Second * 30},
		headers:      make(map[string]string),
		handlers:     make(map[string]chan json.RawMessage),
		contexts:     make(map[string]*mcpctx.Context),
		state:        NewClientState(initURL.String()),
	}
}

func (c *MCPClient) Close() error {
	select {
	case <-c.done:
		return nil // Already closed
	default:
		close(c.done)
	}

	// Clean up any pending responses
	c.mu.Lock()
	for _, ch := range c.responses {
		close(ch)
	}
	c.responses = make(map[int64]chan codec.JSONRPCResponse)
	c.mu.Unlock()

	return nil
}

func (c *MCPClient) AddHeaders(customHeaders map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.headers = customHeaders
}

// Ping the MCP server
func (c *MCPClient) Ping() error {
	_, err := c.SendRequest(context.Background(), mcp.MethodPing, nil)
	return err
}

// Send JSONRPC requests to the server.
// Does not wait for a response, only checks for the return code.
func (c *MCPClient) Send(data codec.JSONRPCRequest) error {
	if !c.initialized {
		return errors.New("client not initialized")
	}

	body, err := json.Marshal(data)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, c.serverURL.String(), bytes.NewReader(body))
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

// SendRequest sends a JSON-RPC request to the server and waits for a response.
// Returns the raw JSON response message or an error if the request fails. Creates
// a dedicated response channel to receive responses with.
//
// https://modelcontextprotocol.io/specification/2025-03-26/basic/transports#sending-messages-to-the-server
func (c *MCPClient) SendRequest(ctx context.Context, method string, params json.RawMessage) (codec.JSONRPCResponse, error) {
	if !c.initialized {
		return codec.NewJSONRPCResponse(), errors.New("client not initialized")
	}

	if c.serverURL.String() == "" {
		return codec.NewJSONRPCResponse(), errors.New("endpoint not received")
	}

	id := c.requestID.Add(1)

	request := codec.JSONRPCRequest{
		JSONRPC: codec.JsonRPCVersion,
		ID:      id,
		Method:  method,
		Params:  params,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return codec.NewJSONRPCResponse(), fmt.Errorf("failed to marshal request: %w", err)
	}

	responseChan := make(chan codec.JSONRPCResponse, 1)
	c.mu.Lock()
	c.responses[id] = responseChan
	c.mu.Unlock()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.serverURL.String(),
		bytes.NewReader(requestBytes),
	)
	if err != nil {
		return codec.NewJSONRPCResponse(), fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "text/event-stream,application/json")
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return codec.NewJSONRPCResponse(), fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return codec.NewJSONRPCResponse(), fmt.Errorf(
			"request failed with status %d: %s",
			resp.StatusCode,
			body,
		)
	}

	select {
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.responses, id)
		c.mu.Unlock()
		return codec.NewJSONRPCResponse(), ctx.Err()
	case response := <-responseChan:
		if response.Error != nil {
			return codec.NewJSONRPCResponse(), errors.New(response.Error.Msg())
		}
		return response, nil
	}
}

// Starts the MCP client by initiating the MCP handshake, then requesting
// a keep-alive connection with the server to receive events from.
func (c *MCPClient) Start(ctx context.Context) error {
	// execute initial MCP handshake
	if err := c.Handshake(); err != nil {
		return fmt.Errorf("mcp handshake failed: %s", err)
	}

	// create a keep-alive connection to receive events from the server.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.serverURL.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream,application/json")
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

	// TODO: validate server response headers before initiating listening
	// https://modelcontextprotocol.io/docs/concepts/transports#security-warning%3A-dns-rebinding-attacks

	// start listening for server-side events
	go func() {
		if err := c.listen(ctx, resp.Body, c.handleSSE); err != nil {
			c.log.Error(fmt.Sprintf("SSE event listener failed: %v", err))
		}
	}()

	select {
	case <-c.endpointChan:
		// Endpoint received, proceed
	case <-ctx.Done():
		return fmt.Errorf("context cancelled while waiting for endpoint")
	case <-time.After(30 * time.Second): // Add a timeout
		return fmt.Errorf("timeout waiting for endpoint")
	}

	return nil
}

// Continually listens for server-side events using the given reader.
// Processes events with the given handler.
func (c *MCPClient) listen(ctx context.Context, reader io.ReadCloser, handler types.Handler) error {
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
						if err := handler(event, data); err != nil {
							return fmt.Errorf("listener handler failed: %v", err)
						}
					}
					break
				}
				select {
				case <-c.done:
					return nil
				default:
					return err
				}
			}

			// Remove only newline markers
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				// Empty line means end of event
				if event != "" && data != "" {
					if err := handler(event, data); err != nil {
						return fmt.Errorf("listener handler failed: %v", err)
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
