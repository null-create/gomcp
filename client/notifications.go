package client

import (
	"encoding/json"
	"fmt"

	"github.com/gomcp/mcp"
)

func (c *MCPClient) HandleMCPNotification(method mcp.MCPNotification, raw json.RawMessage) error {
	switch method {
	case mcp.ContextUpdate:
		return c.handleContextUpdate(raw)
	case mcp.ContextClear:
		return c.handleContextClear(raw)
	case mcp.MemoryAppend:
		return c.handleMemoryAppend(raw)
	case mcp.MemoryReplace:
		return c.handleMemoryReplace(raw)
	// case mcp.ToolResponse:
	// 	return c.handleToolResponse(raw)
	// case mcp.LogEvent:
	// 	return c.handleLogEvent(raw)
	default:
		return fmt.Errorf("unsupported notification method: %s", method)
	}
}
