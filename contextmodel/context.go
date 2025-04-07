package contextmodel

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Context represents a collection of memory blocks for a single client.
type Context struct {
	ID     string        `json:"id"`
	Memory []MemoryBlock `json:"memory"`
	Mutex  sync.Mutex    `json:"-"`
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
