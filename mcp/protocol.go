package mcp

import (
	"context"
	"errors"
	"sync"
)

type RequestHandlerExtra struct {
	// Add contextual info if needed (e.g., trace IDs, client metadata)
	Context context.Context
}

type RequestHandler func(request any, extra RequestHandlerExtra) (any, error)
type NotificationHandler func(notification any) error

type requestHandlerEntry struct {
	schema  any
	handler RequestHandler
}

type notificationHandlerEntry struct {
	schema  any
	handler NotificationHandler
}

type Protocol struct {
	mu sync.RWMutex

	reqHandlers          map[string]requestHandlerEntry
	notificationHandlers map[string]notificationHandlerEntry
}

func NewProtocol() *Protocol {
	return &Protocol{
		reqHandlers:          make(map[string]requestHandlerEntry),
		notificationHandlers: make(map[string]notificationHandlerEntry),
	}
}

func (p *Protocol) SetRequestHandler(method string, schema any, handler RequestHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.reqHandlers[method] = requestHandlerEntry{
		schema:  schema,
		handler: handler,
	}
}

func (p *Protocol) SetNotificationHandler(method string, schema any, handler NotificationHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.notificationHandlers[method] = notificationHandlerEntry{
		schema:  schema,
		handler: handler,
	}
}

func (p *Protocol) HandleRequest(method string, request any, extra RequestHandlerExtra) (any, error) {
	p.mu.RLock()
	handlerEntry, ok := p.reqHandlers[method]
	p.mu.RUnlock()
	if !ok {
		return nil, errors.New("method not found")
	}
	return handlerEntry.handler(request, extra)
}

func (p *Protocol) HandleNotification(method string, notification any) error {
	p.mu.RLock()
	handlerEntry, ok := p.notificationHandlers[method]
	p.mu.RUnlock()
	if !ok {
		return errors.New("notification method not found")
	}
	return handlerEntry.handler(notification)
}
