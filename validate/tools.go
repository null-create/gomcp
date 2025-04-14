package validate // Or your orchestrator package

import (
	"context"
	"errors"
	"fmt"
	"strings"

	msg "github.com/gomcp/types"

	"github.com/xeipuuv/gojsonschema"
)

// FindToolDescription retrieves the trusted tool description by name.
// In a real system, this might involve looking up in a secure registry
// and potentially verifying signatures/sources stored in SecurityMetadata.
func FindToolDescription(name string, availableTools []msg.ToolDescription) (*msg.ToolDescription, error) {
	for _, tool := range availableTools {
		if tool.Name == name {
			// TODO: Add verification of tool description source/integrity here
			// based on tool.SecurityMetadata if available.
			return &tool, nil // Return pointer to avoid copying large schemas
		}
	}
	return nil, fmt.Errorf("tool '%s' not found or not permitted", name)
}

// ValidateToolSchema is called by the orchestrator when an LLM requests a tool call.
func ValidateToolSchema(
	ctx context.Context, // Pass context for cancellation during execution
	toolCall msg.ToolCall,
	availableTools []msg.ToolDescription,
) (executionStatus msg.ExecutionStatus, execErr error) {

	// 1. Find the trusted Tool Description
	toolDesc, err := FindToolDescription(toolCall.FunctionName, availableTools)
	if err != nil {
		return msg.StatusError, fmt.Errorf("tool description lookup failed: %w", err)
	}

	// 2. --- Input Schema Validation ---
	if len(toolDesc.InputSchema) > 0 { // Only validate if schema is provided
		schemaLoader := gojsonschema.NewBytesLoader(toolDesc.InputSchema)
		documentLoader := gojsonschema.NewBytesLoader(toolCall.Arguments)

		// It's often good to compile schemas once and cache them, but for simplicity:
		schema, err := gojsonschema.NewSchema(schemaLoader)
		if err != nil {
			// Schema itself is invalid! Log this serious config error.
			return msg.StatusError, fmt.Errorf("internal schema error for tool '%s'", toolDesc.Name)
		}

		result, err := schema.Validate(documentLoader)
		if err != nil {
			// Error during validation process itself
			return msg.StatusError, fmt.Errorf("internal validation error for tool '%s'", toolDesc.Name)
		}

		if !result.Valid() {
			// Validation FAILED! Do not execute the tool.
			var validationErrors []string
			for _, desc := range result.Errors() {
				validationErrors = append(validationErrors, fmt.Sprintf("- %s", desc))
			}
			errorMsg := fmt.Sprintf("Input validation failed for tool '%s':\n%s",
				toolDesc.Name, strings.Join(validationErrors, "\n"))
			fmt.Println("SECURITY ALERT:", errorMsg) // Log prominently
			return msg.StatusFailed, errors.New(errorMsg)
		}
		// --- Validation Passed ---
		fmt.Printf("Input arguments for tool '%s' validated successfully.\n", toolDesc.Name)
	} else {
		fmt.Printf("WARNING: No InputSchema defined for tool '%s'. Skipping input validation.\n", toolDesc.Name)
	}

	return msg.StatusSucceeded, nil
}

func ValidateToolCallOutput(rawResult string, toolCall msg.ToolCall,
	availableTools []msg.ToolDescription) (msg.ExecutionStatus, error) {
	toolDesc, err := FindToolDescription(toolCall.FunctionName, availableTools)
	if err != nil {
		return msg.StatusError, fmt.Errorf("tool description lookup failed: %w", err)
	}

	if len(toolDesc.OutputSchema) > 0 {
		outputSchemaLoader := gojsonschema.NewBytesLoader(toolDesc.OutputSchema)
		outputDocumentLoader := gojsonschema.NewStringLoader(rawResult) // Assume result is JSON string

		outputSchema, err := gojsonschema.NewSchema(outputSchemaLoader)
		if err != nil {
			// Schema itself is invalid! Log this serious config error.
			fmt.Printf("ERROR: Invalid OutputSchema for tool '%s': %v\n", toolDesc.Name, err)
			// Decide how to handle: return error, return raw result anyway?
			// For security, maybe return an error.
			return msg.StatusError, fmt.Errorf("internal output schema error for tool '%s'", toolDesc.Name)
		}

		outputResult, err := outputSchema.Validate(outputDocumentLoader)
		if err != nil {
			fmt.Printf("ERROR: Output validation process error for tool '%s': %v\n", toolDesc.Name, err)
			return msg.StatusError, fmt.Errorf("internal output validation error for tool '%s'", toolDesc.Name)
		}

		if !outputResult.Valid() {
			// Output validation FAILED! Potential poisoned result from tool.
			var validationErrors []string
			for _, desc := range outputResult.Errors() {
				validationErrors = append(validationErrors, fmt.Sprintf("- %s", desc))
			}
			errorMsg := fmt.Sprintf("Tool '%s' output failed validation:\n%s\nRaw Output: %s",
				toolDesc.Name, strings.Join(validationErrors, "\n"), rawResult)
			fmt.Println("SECURITY ALERT:", errorMsg) // Log prominently

			// Decide action: Don't send back to LLM? Send error message instead?
			// Sending an error message is safer than sending malformed/malicious data.
			return msg.StatusFailed, errors.New(errorMsg)
		}
		fmt.Printf("Output content for tool '%s' validated successfully.\n", toolDesc.Name)
	}
	return msg.StatusSucceeded, nil
}

// --- Example Usage in a hypothetical orchestrator loop ---
/*
func handleAssistantMessageWithToolCalls(ctx context.Context, assistantMsg mcp.Message, currentModelContext *mcp.ModelContext, toolExec func(context.Context, string, json.RawMessage) (string, error)) error {
    if assistantMsg.Role != mcp.RoleAssistant || len(assistantMsg.ToolCalls) == 0 {
        return nil // Not relevant
    }

    // Process each requested tool call
    for _, toolCall := range assistantMsg.ToolCalls {
        toolResultContent, status, execErr := ValidateAndExecuteTool(
            ctx,
            toolCall,
            currentModelContext.AvailableTools, // Pass the available tools
            toolExec,                           // Pass the actual executor function
        )

        // Create a ToolResult message to send back to the LLM
        // Need a helper like AddToolResultMessage from previous examples
        // This helper would create a new MCP Message with RoleTool, ToolCallID,
        // the toolResultContent (or error message), and the status/error metadata.
        addToolResultMessageToContext(currentModelContext, toolCall.ID, toolResultContent, status, execErr, "", "") // Pass hash/env if calculated


		// If a handler error occurred during execution or validation, maybe stop processing further calls?
		if status != mcp.StatusSucceeded {
			fmt.Printf("Tool call %s for %s failed with status %s\n", toolCall.ID, toolCall.FunctionName, status)
			// Decide if one failure stops all subsequent calls in this turn
			// return execErr // Optionally propagate the error up
		}
    }
	// After processing all calls, the updated context would be sent back to the LLM
	return nil
}
*/
