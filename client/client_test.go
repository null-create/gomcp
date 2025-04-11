package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gomcp/codec"
	mcpctx "github.com/gomcp/context"
	"github.com/gomcp/types"

	"github.com/alecthomas/assert"
	"github.com/google/uuid"
)

func newMockClient() *MCPClient {
	return &MCPClient{
		contexts: make(map[string]*mcpctx.Context),
	}
}

func TestHandleContextUpdate_NewContext(t *testing.T) {
	c := newMockClient()
	update := mcpctx.ContextUpdate{
		ID:       uuid.NewString(),
		Metadata: map[string]string{"foo": "bar"},
	}
	b, _ := json.Marshal(update)
	err := c.handleContextUpdate(b)
	assert.NoError(t, err)

	ctx := c.GetClientContext()
	assert.Equal(t, "bar", ctx.Metadata["foo"])
}

func TestHandleContextUpdate_AppendMemory(t *testing.T) {
	c := newMockClient()
	c.contexts[c.clientID] = mcpctx.NewContext(map[string]string{})

	mem := &mcpctx.MemoryBlock{
		ID:      uuid.NewString(),
		Role:    "user",
		Content: "hello",
		Time:    time.Now(),
	}
	update := mcpctx.ContextUpdate{
		ID:     c.GetClientContext().ID,
		Append: []*mcpctx.MemoryBlock{mem},
	}
	b, _ := json.Marshal(update)
	err := c.handleContextUpdate(b)
	assert.NoError(t, err)
	assert.Len(t, c.GetClientContext().Memory, 1)
	assert.Equal(t, "hello", c.GetClientContext().Memory[0].Content)
}

func TestHandleContextClear(t *testing.T) {
	c := newMockClient()
	ctx := mcpctx.NewContext(map[string]string{"foo": "bar"})
	ctx.Memory = append(ctx.Memory, &mcpctx.MemoryBlock{Content: "keep this"})
	c.contexts[c.clientID] = ctx

	update := mcpctx.ContextUpdate{ID: ctx.ID}
	b, _ := json.Marshal(update)
	err := c.handleContextClear(b)
	assert.NoError(t, err)

	newCtx := c.GetClientContext()
	assert.Equal(t, "bar", newCtx.Metadata["foo"])
	assert.Empty(t, newCtx.Memory)
	assert.NotEqual(t, ctx.CreatedAt, newCtx.CreatedAt)
}

func TestHandleContextClear_PreservesMetadata(t *testing.T) {
	metadata := map[string]string{"key": "value"}
	client := &MCPClient{
		clientID: "client1",
		contexts: map[string]*mcpctx.Context{
			"client1": mcpctx.NewContext(metadata),
		},
	}

	client.contexts["client1"].Messages = append(client.contexts["client1"].Messages, types.Message{
		ID:        uuid.NewString(),
		Role:      "user",
		Content:   "hello",
		Timestamp: time.Now(),
	})

	update := mcpctx.ContextUpdate{ID: "ctx-id"}
	raw, _ := json.Marshal(update)

	err := client.handleContextClear(raw)
	assert.NoError(t, err)

	ctx := client.GetClientContext()
	assert.Equal(t, "value", ctx.Metadata["key"])
	assert.Empty(t, ctx.Messages)
}

func TestAppendAssistantResponse(t *testing.T) {
	c := newMockClient()
	ctx := mcpctx.NewContext(map[string]string{})
	c.contexts[c.clientID] = ctx

	c.AppendAssistantResponse("how can I help?")
	updatedCtx := c.GetClientContext()

	assert.Len(t, updatedCtx.Memory, 1)
	assert.Equal(t, "assistant", updatedCtx.Memory[0].Role)
	assert.Equal(t, "how can I help?", updatedCtx.Memory[0].Content)
}

func TestSend_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("Expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}

	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	c := &MCPClient{
		serverURL:  ts.URL,
		clientID:   "test-client",
		httpClient: ts.Client(),
	}

	req := codec.JSONRPCRequest{
		ID:      "1",
		Method:  "testMethod",
		JSONRPC: "2.0",
		Params:  json.RawMessage(`{"key": "value"}`),
	}

	err := c.Send(req)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSend_Failure(t *testing.T) {
	c := &MCPClient{
		serverURL:  "http://localhost:0",
		clientID:   "test-client",
		httpClient: &http.Client{Timeout: 100 * time.Millisecond},
	}

	req := codec.JSONRPCRequest{
		ID:      "1",
		Method:  "testMethod",
		JSONRPC: "2.0",
		Params:  json.RawMessage(`{"key": "value"}`),
	}

	err := c.Send(req)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestListen_CancelContext(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected http.Flusher")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: {\"message\":\"hello\"}")
		flusher.Flush()
		time.Sleep(2 * time.Second)
	}))
	defer ts.Close()

	c := &MCPClient{
		serverURL:  ts.URL,
		clientID:   "test-client",
		httpClient: ts.Client(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	handled := false
	handler := func(msg json.RawMessage) error {
		handled = true
		return nil
	}

	err := c.Listen(ctx, handler)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !handled {
		t.Error("expected message to be handled")
	}
}

func TestListen_Non200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	c := &MCPClient{
		serverURL:  ts.URL,
		clientID:   "test-client",
		httpClient: ts.Client(),
	}

	err := c.Listen(context.Background(), func(msg json.RawMessage) error { return nil })
	if err == nil {
		t.Error("expected error for non-200")
	}
}
