package models

import (
	"fmt"
	"time"

	"github.com/gomcp/types"
)

func HandleToolCall(tools []types.ToolDefinition, userMessage string) (any, error) {
	for _, tool := range tools {
		if tool.Name == "get_time" && containsIgnoreCase(userMessage, "tool:get_time") {
			return map[string]any{
				"tool":   "get_time",
				"result": fmt.Sprintf("The current time is %s", Now()),
			}, nil
		}
		// Add more tool handlers here
	}
	return nil, fmt.Errorf("no matching tool call found")
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || s == fmt.Sprintf("tool:%s", substr))
}

func Now() string {
	return time.Now().Format(time.RFC3339)
}
