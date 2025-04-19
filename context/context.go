package context

import (
	"encoding/json"
	"log"
	"maps"
	"time"

	"github.com/gomcp/types"

	"github.com/google/uuid"
)

// Context represents a conversational context with memory, metadata, etc.
type Context struct {
	ID             string            `json:"id"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	Memory         []*MemoryBlock    `json:"memory"`
	Messages       []types.Message   `json:"messages"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	IsArchived     bool              `json:"is_archived"`
	AvailableTools []types.ToolDescription
}

// NewContext creates a new Context with the given ID and optional metadata.
func NewContext(metadata map[string]string) *Context {
	return &Context{
		ID:        uuid.NewString(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Memory:    make([]*MemoryBlock, 0),
		Metadata:  metadata,
	}
}

// Convert to json formatted bytes
func (c *Context) ToJSON() []byte {
	b, err := json.Marshal(c)
	if err != nil {
		log.Fatal(err)
	}
	return b
}

// ContextUpdate represents an update request to an existing context.
type ContextUpdate struct {
	ID       string            `json:"id"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Append   []*MemoryBlock    `json:"append,omitempty"`
	Archive  *bool             `json:"archive,omitempty"`
}

func NewContextUpdate() ContextUpdate {
	return ContextUpdate{
		Metadata: make(map[string]string),
		Append:   make([]*MemoryBlock, 0),
	}
}

// MemoryBlock represents a single unit of contextual memory within a conversation.
type MemoryBlock struct {
	ID      string    `json:"id"`
	Role    string    `json:"role"` // e.g., "user", "assistant", etc.
	Content string    `json:"content"`
	Time    time.Time `json:"time"`
}

// Convert to JSON-formatted bytes
func (m *MemoryBlock) ToJSON() []byte {
	b, err := json.Marshal(m)
	if err != nil {
		log.Fatal(err)
	}
	return b
}

func (m *MemoryBlock) UpdateContent(newContent string) {
	m.Content = newContent
}

// ApplyUpdate modifies the context based on the update request.
func (ctx *Context) ApplyUpdate(update ContextUpdate) {
	if update.Metadata != nil {
		maps.Copy(ctx.Metadata, update.Metadata)
	}
	if update.Append != nil {
		ctx.Memory = append(ctx.Memory, update.Append...)
	}
	if update.Archive != nil {
		ctx.IsArchived = *update.Archive
	}
	ctx.UpdatedAt = time.Now()
}
