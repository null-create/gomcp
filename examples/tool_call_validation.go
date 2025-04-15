package examples

import (
	"context"
	"fmt"

	mcpctx "github.com/gomcp/context"
	msg "github.com/gomcp/types"
	"github.com/gomcp/validate"
)

// --- Example Usage in a hypothetical orchestrator loop ---

func ValidateToolCalls(
	ctx mcpctx.Context,
	assistantMsg msg.Message,
	currentModelContext *mcpctx.Context,
) error {
	if len(assistantMsg.ToolCalls) == 0 {
		return nil
	}

	// Process each requested tool call
	for _, toolCall := range assistantMsg.ToolCalls {
		status, err := validate.ValidateToolSchema(
			context.Background(),
			toolCall,
			currentModelContext.AvailableTools, // Pass the available tools
		)
		if err != nil {
			return fmt.Errorf("tool validation failed: %v", err)
		}

		// If a handler error occurred during execution or validation, maybe stop processing further calls?
		if status != msg.StatusSucceeded {
			fmt.Printf("Tool call %s for %s failed with status %s\n", toolCall.ID, toolCall.FunctionName, status)
			// Decide if one failure stops all subsequent calls in this turn
			// return execErr // Optionally propagate the error up
		}
	}
	return nil
}
