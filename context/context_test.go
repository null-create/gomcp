package context

import (
	"testing"
	"time"

	"github.com/alecthomas/assert"
	"github.com/google/uuid"
)

func TestNewContext(t *testing.T) {
	metadata := map[string]string{"model": "claude"}
	ctx := NewContext(metadata)

	assert.NotEmpty(t, ctx.ID)
	assert.WithinDuration(t, time.Now(), ctx.CreatedAt, time.Second)
	assert.WithinDuration(t, ctx.CreatedAt, ctx.UpdatedAt, time.Second)
	assert.Equal(t, metadata, ctx.Metadata)
	assert.Empty(t, ctx.Memory)
	assert.False(t, ctx.IsArchived)
}

func TestApplyUpdate_MetadataAppend(t *testing.T) {
	ctx := NewContext(map[string]string{"foo": "bar"})
	update := ContextUpdate{
		ID:       ctx.ID,
		Metadata: map[string]string{"baz": "qux"},
	}

	ctx.ApplyUpdate(update)

	assert.Equal(t, "bar", ctx.Metadata["foo"])
	assert.Equal(t, "qux", ctx.Metadata["baz"])
}

func TestApplyUpdate_AppendMemory(t *testing.T) {
	ctx := NewContext(nil)
	mem := MemoryBlock{
		ID:      uuid.NewString(),
		Role:    "user",
		Content: "Hello!",
		Time:    time.Now(),
	}
	update := ContextUpdate{
		ID:     ctx.ID,
		Append: []MemoryBlock{mem},
	}

	ctx.ApplyUpdate(update)

	assert.Len(t, ctx.Memory, 1)
	assert.Equal(t, mem.Content, ctx.Memory[0].Content)
}

func TestApplyUpdate_Archive(t *testing.T) {
	ctx := NewContext(nil)
	archive := true
	update := ContextUpdate{
		ID:      ctx.ID,
		Archive: &archive,
	}

	ctx.ApplyUpdate(update)

	assert.True(t, ctx.IsArchived)
}

func TestMemoryBlock_UpdateContent(t *testing.T) {
	block := MemoryBlock{
		ID:      uuid.NewString(),
		Role:    "assistant",
		Content: "Hi there!",
		Time:    time.Now(),
	}

	block.UpdateContent("Updated content")

	assert.Equal(t, "Updated content", block.Content)
}
