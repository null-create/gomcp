package mcp

import (
	"encoding/json"
	"log"
)

// --- MCP Handshake Specific Structures ---

// Capabilities Structures
type RootCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type SamplingCapabilities struct {
	// Empty object {} indicates support
}

type LoggingCapabilities struct {
	// Empty object {} indicates support
}

type PromptCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type ResourceCapabilities struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

type ToolCapabilities struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

// Use map for flexibility with experimental features
type ExperimentalCapabilities map[string]any

type ClientCapabilities struct {
	Roots        *RootCapabilities        `json:"roots,omitempty"`
	Sampling     *SamplingCapabilities    `json:"sampling,omitempty"`
	Experimental ExperimentalCapabilities `json:"experimental,omitempty"`
}

func NewClientCapabilities() ClientCapabilities {
	return ClientCapabilities{
		Roots:        &RootCapabilities{ListChanged: true},
		Sampling:     &SamplingCapabilities{},
		Experimental: ExperimentalCapabilities{},
	}
}

type ServerCapabilities struct {
	Logging      *LoggingCapabilities     `json:"logging,omitempty"`
	Prompts      *PromptCapabilities      `json:"prompts,omitempty"`
	Resources    *ResourceCapabilities    `json:"resources,omitempty"`
	Tools        *ToolCapabilities        `json:"tools,omitempty"`
	Experimental ExperimentalCapabilities `json:"experimental,omitempty"`
}

// Info Structures
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func NewClientInfo(name, version string) ClientInfo {
	return ClientInfo{
		Name:    name,
		Version: version,
	}
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func NewServerInfo(name, version string) ServerInfo {
	return ServerInfo{
		Name:    name,
		Version: version,
	}
}

// Initialize Request/Response Payloads
type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      ClientInfo         `json:"clientInfo"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      ServerInfo         `json:"serverInfo"`
}

func (i *InitializeResult) Bytes() []byte {
	b, err := json.Marshal(i)
	if err != nil {
		log.Fatal(err)
	}
	return b
}
