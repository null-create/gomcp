package sse

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

	"github.com/gomcp/logger"

	"github.com/google/uuid"
)

type MessageHandler func(message json.RawMessage)

// SSEMCPClient implements the MCPClient interface using Server-Sent Events (SSE).
type SSEMCPClient struct {
	log        *logger.Logger
	serverURL  string
	clientID   string
	httpClient *http.Client
	handlers   map[string]chan json.RawMessage
	mu         sync.Mutex
}

func NewSSEMCPClient(serverURL, clientID string) *SSEMCPClient {
	return &SSEMCPClient{
		log:        logger.NewLogger("SSEMCPClient", uuid.NewString()),
		serverURL:  serverURL,
		clientID:   clientID,
		httpClient: &http.Client{},
		handlers:   make(map[string]chan json.RawMessage),
	}
}

func (c *SSEMCPClient) Start(ctx context.Context, handler MessageHandler) error {
	retries, maxRextries := 0, 3
	url := fmt.Sprintf("%s?id=%s", c.serverURL, c.clientID)
	for {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Accept", "text/event-stream")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			c.log.Error(fmt.Sprintf("SSE connection error: %v", err))
			retries += 1
			if retries == maxRextries {
				break
			}
			time.Sleep(2 * time.Second)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			c.log.Error(fmt.Sprintf("received non-200 response: %d", resp.StatusCode))
			return fmt.Errorf("received non-200 response code: %d", resp.StatusCode)
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				msg := strings.TrimPrefix(line, "data: ")
				handler(json.RawMessage(msg))
			}
			select {
			case <-ctx.Done():
				resp.Body.Close()
				return nil
			default:
			}
		}

		if err := scanner.Err(); err != nil {
			c.log.Error(fmt.Sprintf("SSE scanner error: %v", err))
		}
		resp.Body.Close()
		time.Sleep(2 * time.Second) // Reconnect delay
	}
	return nil
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
