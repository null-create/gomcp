package mcp

const (
	// Initiates connection and negotiates protocol capabilities.
	// https://modelcontextprotocol.io/specification/2025-03-26/basic/lifecycle#initialization
	MethodInitialize string = "initialize"

	// Verifies connection liveness between client and server.
	// https://modelcontextprotocol.io/specification/2025-03-26/basic/utilities/ping
	MethodPing string = "ping"

	// Lists all available server resources.
	// https://modelcontextprotocol.io/specification/2025-03-26/server/resources
	MethodResourcesList string = "resources/list"

	// Provides URI templates for constructing resource URIs.
	// https://modelcontextprotocol.io/specification/2025-03-26/server/resources
	MethodResourcesTemplatesList string = "resources/templates/list"

	// Retrieves content of a specific resource by URI.
	// https://modelcontextprotocol.io/specification/2025-03-26/server/resources
	MethodResourcesRead string = "resources/read"

	// Lists all available prompt templates.
	// https://modelcontextprotocol.io/specification/2025-03-26/server/prompts
	MethodPromptsList string = "prompts/list"

	// Retrieves a specific prompt template with filled parameters.
	// https://modelcontextprotocol.io/specification/2025-03-26/server/prompts
	MethodPromptsGet string = "prompts/get"

	// Lists all available executable tools.
	// https://modelcontextprotocol.io/specification/2025-03-26/server/tools
	MethodToolsList string = "tools/list"

	// Invokes a specific tool with provided parameters.
	// https://modelcontextprotocol.io/specification/2025-03-26/server/tools
	MethodToolsCall string = "tools/call"
)
