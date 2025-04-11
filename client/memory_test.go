package client

import (
	"encoding/json"
	"sync"
	"testing"

	mcpctx "github.com/gomcp/context"
)

func createTestClient(clientID string, contextID string) *MCPClient {
	return &MCPClient{
		contexts: map[string]*mcpctx.Context{
			clientID: {
				ID:     contextID,
				Memory: []*mcpctx.MemoryBlock{},
			},
		},
		clientID: clientID,
		mu:       sync.Mutex{},
	}
}

func TestHandleMemoryAppend(t *testing.T) {
	clientID := "client1"
	contextID := "ctx1"
	client := createTestClient(clientID, contextID)

	update := mcpctx.ContextUpdate{
		ID: contextID,
		Append: []*mcpctx.MemoryBlock{
			{ID: "m1", Content: "test content"},
		},
	}

	raw, err := json.Marshal(update)
	if err != nil {
		t.Fatalf("failed to marshal update: %v", err)
	}

	err = client.handleMemoryAppend(raw)
	if err != nil {
		t.Fatalf("handleMemoryAppend failed: %v", err)
	}

	if len(client.contexts[clientID].Memory) != 1 {
		t.Errorf("expected 1 memory item, got %d", len(client.contexts[clientID].Memory))
	}

	if client.contexts[clientID].Memory[0].Content != "test content" {
		t.Errorf("memory content mismatch, got: %s", client.contexts[clientID].Memory[0].Content)
	}
}

func TestHandleMemoryReplace(t *testing.T) {
	clientID := "client1"
	contextID := "ctx1"
	initialMemory := &mcpctx.MemoryBlock{ID: "m1", Content: "old content"}

	client := &MCPClient{
		contexts: map[string]*mcpctx.Context{
			clientID: {
				ID:     contextID,
				Memory: []*mcpctx.MemoryBlock{initialMemory},
			},
		},
		clientID: clientID,
		mu:       sync.Mutex{},
	}

	update := &mcpctx.ContextUpdate{
		ID: contextID,
		Append: []*mcpctx.MemoryBlock{
			{ID: "m1", Content: "new content"},
		},
	}

	raw, err := json.Marshal(update)
	if err != nil {
		t.Fatalf("failed to marshal update: %v", err)
	}

	err = client.handleMemoryReplace(raw)
	if err != nil {
		t.Fatalf("handleMemoryReplace failed: %v", err)
	}

	if initialMemory.Content != "new content" {
		t.Errorf("expected content to be 'new content', got: %s", initialMemory.Content)
	}
}
