package mcp

type MCPMethod string

const (
	// Initiates connection and negotiates protocol capabilities.
	// https://spec.modelcontextprotocol.io/specification/2024-11-05/basic/lifecycle/#initialization
	MethodInitialize MCPMethod = "initialize"

	// Verifies connection liveness between client and server.
	// https://spec.modelcontextprotocol.io/specification/2024-11-05/basic/utilities/ping/
	MethodPing MCPMethod = "ping"

	// Lists all available server resources.
	// https://spec.modelcontextprotocol.io/specification/2024-11-05/server/resources/
	MethodResourcesList MCPMethod = "resources/list"

	// Provides URI templates for constructing resource URIs.
	// https://spec.modelcontextprotocol.io/specification/2024-11-05/server/resources/
	MethodResourcesTemplatesList MCPMethod = "resources/templates/list"

	// Retrieves content of a specific resource by URI.
	// https://spec.modelcontextprotocol.io/specification/2024-11-05/server/resources/
	MethodResourcesRead MCPMethod = "resources/read"

	// Lists all available prompt templates.
	// https://spec.modelcontextprotocol.io/specification/2024-11-05/server/prompts/
	MethodPromptsList MCPMethod = "prompts/list"

	// Retrieves a specific prompt template with filled parameters.
	// https://spec.modelcontextprotocol.io/specification/2024-11-05/server/prompts/
	MethodPromptsGet MCPMethod = "prompts/get"

	// Lists all available executable tools.
	// https://spec.modelcontextprotocol.io/specification/2024-11-05/server/tools/
	MethodToolsList MCPMethod = "tools/list"

	// Invokes a specific tool with provided parameters.
	// https://spec.modelcontextprotocol.io/specification/2024-11-05/server/tools/
	MethodToolsCall MCPMethod = "tools/call"
)
