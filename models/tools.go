package models

import "fmt"

type ToolCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

func HandleToolCall(tools []ToolDefinition, userMessage string) (interface{}, error) {
	for _, tool := range tools {
		if tool.Name == "get_time" && containsIgnoreCase(userMessage, "tool:get_time") {
			return map[string]interface{}{
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
	return "2025-04-05T12:00:00Z" // Replace with time.Now().Format(time.RFC3339) if dynamic
}
