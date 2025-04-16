package mcp

const (
	// Initiates connection and negotiates protocol capabilities.
	// https://spec.modelcontextprotocol.io/specification/2024-11-05/basic/lifecycle/#initialization
	MethodInitialize string = "initialize"

	// Verifies connection liveness between client and server.
	// https://spec.modelcontextprotocol.io/specification/2024-11-05/basic/utilities/ping/
	MethodPing string = "ping"

	// Lists all available server resources.
	// https://spec.modelcontextprotocol.io/specification/2024-11-05/server/resources/
	MethodResourcesList string = "resources/list"

	// Provides URI templates for constructing resource URIs.
	// https://spec.modelcontextprotocol.io/specification/2024-11-05/server/resources/
	MethodResourcesTemplatesList string = "resources/templates/list"

	// Retrieves content of a specific resource by URI.
	// https://spec.modelcontextprotocol.io/specification/2024-11-05/server/resources/
	MethodResourcesRead string = "resources/read"

	// Lists all available prompt templates.
	// https://spec.modelcontextprotocol.io/specification/2024-11-05/server/prompts/
	MethodPromptsList string = "prompts/list"

	// Retrieves a specific prompt template with filled parameters.
	// https://spec.modelcontextprotocol.io/specification/2024-11-05/server/prompts/
	MethodPromptsGet string = "prompts/get"

	// Lists all available executable tools.
	// https://spec.modelcontextprotocol.io/specification/2024-11-05/server/tools/
	MethodToolsList string = "tools/list"

	// Invokes a specific tool with provided parameters.
	// https://spec.modelcontextprotocol.io/specification/2024-11-05/server/tools/
	MethodToolsCall string = "tools/call"
)
