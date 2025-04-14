package client

import (
	"encoding/json"
	"testing"
	"time"

	mcpctx "github.com/gomcp/context"
	"github.com/gomcp/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

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
