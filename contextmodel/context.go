package contextmodel

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gomcp/types"

	"github.com/google/uuid"
)

// Context represents a collection of memory blocks for a single client.
type Context struct {
	Mutex    sync.Mutex      `json:"-"`
	ID       string          `json:"id"`
	Memory   []MemoryBlock   `json:"memory"`
	Messages []types.Message `json:"messages"`
}

// MemoryBlock represents an individual memory block.
type MemoryBlock struct {
	Role    string    `json:"role"`
	Content string    `json:"content"`
	Time    time.Time `json:"time"`
}

// ContextUpdate represents an update to a context.
type ContextUpdate struct {
	ID      string        `json:"id"`
	Append  []MemoryBlock `json:"append,omitempty"`
	Replace []MemoryBlock `json:"replace,omitempty"`
}

func NewContext(id string, memory []MemoryBlock) *Context {
	return &Context{
		ID:     id,
		Memory: memory,
	}
}

func (ctx *Context) ApplyUpdate(update ContextUpdate) {
	ctx.Mutex.Lock()
	defer ctx.Mutex.Unlock()

	if update.Replace != nil {
		ctx.Memory = update.Replace
	}
	if update.Append != nil {
		ctx.Memory = append(ctx.Memory, update.Append...)
	}
}

func (ctx *Context) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		ID     string        `json:"id"`
		Memory []MemoryBlock `json:"memory"`
	}{
		ID:     ctx.ID,
		Memory: ctx.Memory,
	})
}

func (ctx *Context) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID     string        `json:"id"`
		Memory []MemoryBlock `json:"memory"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	ctx.ID = aux.ID
	ctx.Memory = aux.Memory
	return nil
}

func (ctx *Context) GetMemory(role string) []MemoryBlock {
	ctx.Mutex.Lock()
	defer ctx.Mutex.Unlock()

	var filteredMemory []MemoryBlock
	for _, block := range ctx.Memory {
		if block.Role == role {
			filteredMemory = append(filteredMemory, block)
		}
	}
	return filteredMemory
}

func (ctx *Context) ClearMemory() {
	ctx.Mutex.Lock()
	defer ctx.Mutex.Unlock()
	ctx.Memory = []MemoryBlock{}
}

func (ctx *Context) String() string {
	return fmt.Sprintf("Context{id: %s, memory: %v}", ctx.ID, ctx.Memory)
}

// AddToolResultMessage adds a tool result message to the context.
func (ctx *Context) AddToolResultMessage(toolCallID, content string, status types.ExecutionStatus, execError error, outputHash, execEnv string) *types.Message {
	resultMetadata := &types.ToolResultMetadata{
		ExecutionStatus: status,
		ExecutedAt:      time.Now().UTC(),
		OutputHash:      outputHash, // Should be calculated by the tool executor
		ExecutionEnv:    execEnv,
	}
	if execError != nil {
		resultMetadata.ErrorMessage = execError.Error()
	}

	msg := types.Message{
		ID:         generateUniqueID("msg"),
		Role:       types.RoleTool,
		Content:    content,
		Timestamp:  time.Now().UTC(),
		ToolCallID: toolCallID,
		ToolResult: resultMetadata,
	}
	ctx.Messages = append(ctx.Messages, msg)
	return &ctx.Messages[len(ctx.Messages)-1]
}

func generateUniqueID(prefix string) string {
	return prefix + "_" + uuid.NewString()
}
