package client

import (
	"encoding/json"

	mcpctx "github.com/gomcp/context"
)

func (c *MCPClient) handleMemoryAppend(raw json.RawMessage) error {
	var update mcpctx.ContextUpdate
	if err := json.Unmarshal(raw, &update); err != nil {
		return err
	}

	for _, ctx := range c.contexts {
		if ctx.ID == update.ID {
			c.mu.Lock()
			c.contexts[c.clientID].Memory = append(c.contexts[c.clientID].Memory, update.Append...)
			c.mu.Unlock()
		}
	}

	return nil
}

func (c *MCPClient) handleMemoryReplace(raw json.RawMessage) error {
	var update mcpctx.ContextUpdate
	if err := json.Unmarshal(raw, &update); err != nil {
		return err
	}

	if ctx, ok := c.contexts[c.clientID]; ok {
		for _, memory := range ctx.Memory {
			for _, updatedmemory := range update.Append {
				if memory.ID == updatedmemory.ID {
					c.mu.Lock()
					memory.UpdateContent(updatedmemory.Content)
					c.mu.Unlock()
				}
			}
		}
	}

	return nil
}
