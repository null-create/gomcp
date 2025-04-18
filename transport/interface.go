package transport

import (
	"context"

	"github.com/gomcp/codec"
)

// Interface for the transport layer.
type Interface interface {
	// Start the connection. Start should only be called once.
	Start(ctx context.Context) error

	// SendRequest sends a json RPC request and returns the response synchronously.
	SendRequest(ctx context.Context, request codec.JSONRPCRequest) (*codec.JSONRPCResponse, error)

	// SendNotification sends a json RPC Notification to the server.
	SendNotification(ctx context.Context, notification codec.Notification) error

	// SetNotificationHandler sets the handler for notifications.
	// Any notification before the handler is set will be discarded.
	SetNotificationHandler(handler func(notification codec.Notification))

	// Close the connection.
	Close() error
}
