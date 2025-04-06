package types

// Request is a message that expects a response
// It corresponds to a method call with optional parameters.
type Request struct {
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

// Result is a successful response to a Request.
type Result map[string]any

// Error represents a failure in handling a Request.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Notification is a one-way message that does not expect a response.
type Notification struct {
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

type ToolCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type ToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ChatCompletionRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role      string     `json:"role"`
		Content   string     `json:"content"`
		ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	} `json:"messages"`
	Tools  []ToolDefinition `json:"tools,omitempty"`
	Stream bool             `json:"stream,omitempty"`
}
