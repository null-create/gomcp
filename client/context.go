package client

import (
	"encoding/json"
	"time"

	mcpctx "github.com/gomcp/context"

	"github.com/google/uuid"
)

func (c *MCPClient) handleContextUpdate(raw json.RawMessage) error {
	var update mcpctx.ContextUpdate
	if err := json.Unmarshal(raw, &update); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	ctx, ok := c.contexts[c.clientID]
	if !ok {
		ctx = mcpctx.NewContext(make(map[string]string))
		c.contexts[c.clientID] = ctx
	}

	ctx.ApplyUpdate(update)
	return nil
}

func (c *MCPClient) handleContextClear(raw json.RawMessage) error {
	var update mcpctx.ContextUpdate
	if err := json.Unmarshal(raw, &update); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	metadata := c.contexts[c.clientID].Metadata
	c.contexts[c.clientID] = mcpctx.NewContext(metadata)
	return nil
}

func (c *MCPClient) GetClientContext() *mcpctx.Context {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.contexts[c.clientID]
}

func (c *MCPClient) AppendAssistantResponse(content string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ctx, ok := c.contexts[c.clientID]; ok {
		ctx.ApplyUpdate(mcpctx.ContextUpdate{
			ID:       ctx.ID,
			Metadata: c.contexts[c.clientID].Metadata,
			Append: []*mcpctx.MemoryBlock{{
				ID:      uuid.NewString(),
				Role:    "assistant",
				Content: content,
				Time:    time.Now(),
			}},
		})
	}
}
