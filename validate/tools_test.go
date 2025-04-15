package validate

import (
	"context"
	"encoding/json"
	"testing"

	msg "github.com/gomcp/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test Data Setup ---

var testSchemaBasic = json.RawMessage(`{
	"type": "object",
	"properties": {
		"location": {"type": "string", "description": "City name"},
		"unit": {"type": "string", "enum": ["celsius", "fahrenheit"]}
	},
	"required": ["location"]
}`)

var testSchemaOutputBasic = json.RawMessage(`{
	"type": "object",
	"properties": {
		"temperature": {"type": "number"},
		"conditions": {"type": "string"}
	},
	"required": ["temperature", "conditions"]
}`)

var testSchemaInvalidSyntax = json.RawMessage(`{"type": "object", "properties": {"location": }}`) // Invalid JSON

var availableToolsFixture = []msg.ToolDescription{
	{
		Name:         "get_weather",
		Description:  "Fetches weather",
		InputSchema:  testSchemaBasic,
		OutputSchema: testSchemaOutputBasic,
	},
	{
		Name:         "get_stock",
		Description:  "Fetches stock price",
		InputSchema:  json.RawMessage(`{"type": "object", "properties": {"symbol": {"type": "string"}}, "required": ["symbol"]}`),
		OutputSchema: json.RawMessage(`{"type": "object", "properties": {"price": {"type": "number"}}, "required": ["price"]}`),
	},
	{
		Name:        "no_schema_tool",
		Description: "A tool without any schemas defined",
	},
	{
		Name:        "bad_input_schema_tool",
		Description: "Tool with invalid input schema",
		InputSchema: testSchemaInvalidSyntax,
	},
	{
		Name:         "bad_output_schema_tool",
		Description:  "Tool with invalid output schema",
		InputSchema:  testSchemaBasic, // Valid input
		OutputSchema: testSchemaInvalidSyntax,
	},
}

// --- Tests for FindToolDescription ---

func TestFindToolDescription(t *testing.T) {
	t.Run("Tool Found", func(t *testing.T) {
		tool, err := FindToolDescription("get_weather", availableToolsFixture)
		require.NoError(t, err)
		require.NotNil(t, tool)
		assert.Equal(t, "get_weather", tool.Name)
		assert.Equal(t, testSchemaBasic, tool.InputSchema)
	})

	t.Run("Tool Not Found", func(t *testing.T) {
		tool, err := FindToolDescription("non_existent_tool", availableToolsFixture)
		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "'non_existent_tool' not found")
	})

	t.Run("Empty Tool List", func(t *testing.T) {
		tool, err := FindToolDescription("get_weather", []msg.ToolDescription{})
		require.Error(t, err)
		assert.Nil(t, tool)
		assert.Contains(t, err.Error(), "'get_weather' not found")
	})
}

// --- Tests for ValidateToolSchema ---

func TestValidateToolSchema(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		toolCall       msg.ToolCall
		availableTools []msg.ToolDescription
		expectedStatus msg.ExecutionStatus
		expectError    bool
		errorContains  string // Substring to check in error message if expectError is true
	}{
		{
			name: "Valid Arguments",
			toolCall: msg.ToolCall{
				FunctionName: "get_weather",
				Arguments:    json.RawMessage(`{"location": "London", "unit": "celsius"}`),
			},
			availableTools: availableToolsFixture,
			expectedStatus: msg.StatusSucceeded,
			expectError:    false,
		},
		{
			name: "Valid Arguments (Optional Field Missing)",
			toolCall: msg.ToolCall{
				FunctionName: "get_weather",
				Arguments:    json.RawMessage(`{"location": "Paris"}`), // unit is optional
			},
			availableTools: availableToolsFixture,
			expectedStatus: msg.StatusSucceeded,
			expectError:    false,
		},
		{
			name: "Invalid Arguments (Type Mismatch)",
			toolCall: msg.ToolCall{
				FunctionName: "get_weather",
				Arguments:    json.RawMessage(`{"location": 123}`), // location expects string
			},
			availableTools: availableToolsFixture,
			expectedStatus: msg.StatusFailed,
			expectError:    true,
			errorContains:  "location: Invalid type. Expected: string, given: integer",
		},
		{
			name: "Invalid Arguments (Missing Required Field)",
			toolCall: msg.ToolCall{
				FunctionName: "get_weather",
				Arguments:    json.RawMessage(`{"unit": "fahrenheit"}`), // location is required
			},
			availableTools: availableToolsFixture,
			expectedStatus: msg.StatusFailed,
			expectError:    true,
			errorContains:  "location is required",
		},
		{
			name: "Invalid Arguments (Enum Mismatch)",
			toolCall: msg.ToolCall{
				FunctionName: "get_weather",
				Arguments:    json.RawMessage(`{"location": "Tokyo", "unit": "kelvin"}`), // kelvin not in enum
			},
			availableTools: availableToolsFixture,
			expectedStatus: msg.StatusFailed,
			expectError:    true,
			errorContains:  "unit: unit must be one of the following: \"celsius\", \"fahrenheit\"",
		},
		{
			name: "Invalid Arguments (Not JSON)",
			toolCall: msg.ToolCall{
				FunctionName: "get_weather",
				Arguments:    json.RawMessage(`{location: "Berlin"}`), // Invalid JSON syntax
			},
			availableTools: availableToolsFixture,
			expectedStatus: msg.StatusError, // This fails during schema.Validate loading the document
			expectError:    true,
			errorContains:  "internal validation error", // gojsonschema validation process error
		},
		{
			name: "Tool Not Found",
			toolCall: msg.ToolCall{
				FunctionName: "unknown_tool",
				Arguments:    json.RawMessage(`{}`),
			},
			availableTools: availableToolsFixture,
			expectedStatus: msg.StatusError,
			expectError:    true,
			errorContains:  "tool description lookup failed",
		},
		{
			name: "No Input Schema Defined",
			toolCall: msg.ToolCall{
				FunctionName: "no_schema_tool",
				Arguments:    json.RawMessage(`{"any": "data"}`),
			},
			availableTools: availableToolsFixture,
			expectedStatus: msg.StatusFailed,
			expectError:    true,
		},
		{
			name: "Invalid Input Schema Syntax",
			toolCall: msg.ToolCall{
				FunctionName: "bad_input_schema_tool",
				Arguments:    json.RawMessage(`{"location": "Rome"}`),
			},
			availableTools: availableToolsFixture,
			expectedStatus: msg.StatusError, // Error occurs when loading the schema
			expectError:    true,
			errorContains:  "internal schema error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, err := ValidateToolSchema(ctx, tc.toolCall, tc.availableTools)

			assert.Equal(t, tc.expectedStatus, status, "Status mismatch")
			if tc.expectError {
				require.Error(t, err, "Expected an error but got nil")
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains, "Error message mismatch")
				}
			} else {
				assert.NoError(t, err, "Expected no error but got one")
			}
		})
	}
}

// --- Tests for ValidateToolCallOutput ---

func TestValidateToolCallOutput(t *testing.T) {
	toolCallWeather := msg.ToolCall{FunctionName: "get_weather"}
	toolCallNoSchema := msg.ToolCall{FunctionName: "no_schema_tool"}
	toolCallBadOutputSchema := msg.ToolCall{FunctionName: "bad_output_schema_tool"}
	toolCallUnknown := msg.ToolCall{FunctionName: "unknown_tool"}

	tests := []struct {
		name           string
		rawResult      string
		toolCall       msg.ToolCall
		availableTools []msg.ToolDescription
		expectedStatus msg.ExecutionStatus
		expectError    bool
		errorContains  string
	}{
		{
			name:           "Valid Output",
			rawResult:      `{"temperature": 25.5, "conditions": "Sunny"}`,
			toolCall:       toolCallWeather,
			availableTools: availableToolsFixture,
			expectedStatus: msg.StatusSucceeded,
			expectError:    false,
		},
		{
			name:           "Invalid Output (Type Mismatch)",
			rawResult:      `{"temperature": "hot", "conditions": "Cloudy"}`, // temp should be number
			toolCall:       toolCallWeather,
			availableTools: availableToolsFixture,
			expectedStatus: msg.StatusFailed,
			expectError:    true,
			errorContains:  "temperature: Invalid type. Expected: number, given: string",
		},
		{
			name:           "Invalid Output (Missing Required Field)",
			rawResult:      `{"temperature": 10}`, // conditions missing
			toolCall:       toolCallWeather,
			availableTools: availableToolsFixture,
			expectedStatus: msg.StatusFailed,
			expectError:    true,
			errorContains:  "conditions is required",
		},
		{
			name:           "Invalid Output (Not JSON)",
			rawResult:      `Temperature is 15`, // Not JSON
			toolCall:       toolCallWeather,
			availableTools: availableToolsFixture,
			expectedStatus: msg.StatusError, // Fails during document loading
			expectError:    true,
			errorContains:  "internal output validation error",
		},
		{
			name:           "Tool Not Found",
			rawResult:      `{}`,
			toolCall:       toolCallUnknown,
			availableTools: availableToolsFixture,
			expectedStatus: msg.StatusError,
			expectError:    true,
			errorContains:  "tool description lookup failed",
		},
		{
			name:           "No Output Schema Defined",
			rawResult:      `{"any": "data", "is": "fine"}`,
			toolCall:       toolCallNoSchema, // This tool has no output schema
			availableTools: availableToolsFixture,
			expectedStatus: msg.StatusSucceeded, // Should succeed, skipping validation
			expectError:    false,
		},
		{
			name:           "Invalid Output Schema Syntax",
			rawResult:      `{"temperature": 20, "conditions": "Rainy"}`, // Result is valid
			toolCall:       toolCallBadOutputSchema,                      // But schema is bad
			availableTools: availableToolsFixture,
			expectedStatus: msg.StatusError, // Error occurs when loading the schema
			expectError:    true,
			errorContains:  "internal output schema error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, err := ValidateToolCallOutput(tc.rawResult, tc.toolCall, tc.availableTools)

			assert.Equal(t, tc.expectedStatus, status, "Status mismatch")
			if tc.expectError {
				require.Error(t, err, "Expected an error but got nil")
				if tc.errorContains != "" {
					assert.Contains(t, err.Error(), tc.errorContains, "Error message mismatch")
				}
			} else {
				assert.NoError(t, err, "Expected no error but got one")
			}
		})
	}
}
