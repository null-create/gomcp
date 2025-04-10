package mcp

type MCPNotification string

const (
	ContextUpdate MCPNotification = "context/update"
	ContextClear  MCPNotification = "context/clear"
	MemoryAppend  MCPNotification = "memory/append"
	MemoryReplace MCPNotification = "memory/replace"
	ToolResponse  MCPNotification = "tool/response"
	LogEvent      MCPNotification = "log/event"
)
