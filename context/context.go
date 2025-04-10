package context

import (
	"time"

	"github.com/gomcp/types"

	"github.com/google/uuid"
)

// Context represents a conversational context with memory, metadata, etc.
type Context struct {
	ID         string            `json:"id"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	Memory     []MemoryBlock     `json:"memory"`
	Messages   []types.Message   `json:"messages"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	IsArchived bool              `json:"is_archived"`
}

// ContextUpdate represents an update request to an existing context.
type ContextUpdate struct {
	ID       string            `json:"id"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Append   []MemoryBlock     `json:"append,omitempty"`
	Archive  *bool             `json:"archive,omitempty"`
}

func NewContextUpdate() ContextUpdate {
	return ContextUpdate{}
}

// MemoryBlock represents a single unit of contextual memory within a conversation.
type MemoryBlock struct {
	ID      string    `json:"id"`
	Role    string    `json:"role"` // e.g., "user", "assistant", etc.
	Content string    `json:"content"`
	Time    time.Time `json:"time"`
}

func (m *MemoryBlock) UpdateContent(newContent string) {
	m.Content = newContent
}

// NewContext creates a new Context with the given ID and optional metadata.
func NewContext(metadata map[string]string) *Context {
	return &Context{
		ID:        uuid.NewString(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Memory:    make([]MemoryBlock, 0),
		Metadata:  metadata,
	}
}

// ApplyUpdate modifies the context based on the update request.
func (ctx *Context) ApplyUpdate(update ContextUpdate) {
	if update.Metadata != nil {
		for k, v := range update.Metadata {
			ctx.Metadata[k] = v
		}
	}
	if update.Append != nil {
		ctx.Memory = append(ctx.Memory, update.Append...)
	}
	if update.Archive != nil {
		ctx.IsArchived = *update.Archive
	}
	ctx.UpdatedAt = time.Now()
}
