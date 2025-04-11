package types

import (
	"github.com/gomcp/codec"
	"github.com/gomcp/mcp"
)

type Initializer interface {
	CreateInitializeRequest(id string) ([]byte, error)
	SendInitRequest([]byte) ([]byte, error)
	ProcessInitializeResponse(codec.JSONRPCResponse) error
	CreateInitializedNotification() ([]byte, error)
	IsInitialized() bool
	SendInitNotification([]byte) ([]byte, error)
	GetNegotiatedVersion() string
	GetServerInfo() *mcp.ServerInfo
	HasServerInfo() bool
	SetNegotiatedVersion(v string)
	SetServerInfo(info *mcp.ServerInfo)
	SetInitialized(bool)
}
