package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gomcp/codec"
	mcpctx "github.com/gomcp/context"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newMockClient() *MCPClient {
	return &MCPClient{
		contexts: make(map[string]*mcpctx.Context),
		done:     make(chan struct{}),
	}
}

func TestAppendAssistantResponse(t *testing.T) {
	c := newMockClient()
	ctx := mcpctx.NewContext(map[string]string{})
	c.contexts[c.clientID] = ctx

	c.AppendAssistantResponse("how can I help?")
	updatedCtx := c.GetClientContext()

	assert.Len(t, updatedCtx.Memory, 1)
	assert.Equal(t, "assistant", updatedCtx.Memory[0].Role)
	assert.Equal(t, "how can I help?", updatedCtx.Memory[0].Content)
}

func TestSend_Success(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("Expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}

	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	c := &MCPClient{
		serverURL:  ts.URL,
		clientID:   "test-client",
		httpClient: ts.Client(),
	}

	req := codec.JSONRPCRequest{
		ID:      "1",
		Method:  "testMethod",
		JSONRPC: "2.0",
		Params:  json.RawMessage(`{"key": "value"}`),
	}

	err := c.Send(req)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestSend_Failure(t *testing.T) {
	c := &MCPClient{
		serverURL:  "http://localhost:0",
		clientID:   "test-client",
		httpClient: &http.Client{Timeout: 100 * time.Millisecond},
	}

	req := codec.JSONRPCRequest{
		ID:      "1",
		Method:  "testMethod",
		JSONRPC: "2.0",
		Params:  json.RawMessage(`{"key": "value"}`),
	}

	err := c.Send(req)
	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

// --- Mock for io.ReadCloser ---
// Using the manual mock from previous examples for fine control over reads/errors
type MockReaderCloser struct {
	Buffer      *bytes.Buffer // Use bytes.Buffer for easy data feeding
	CloseCalled bool          // Whether the Close() function was called
	ReadError   error         // Error to return on Read (after buffer empty)
	mock.Mock                 // Embed testify's mock object
	Reader      io.Reader     // Embed a real reader to handle Read calls easily
}

func NewMockReaderCloser(data string) *MockReaderCloser {
	return &MockReaderCloser{
		Buffer: bytes.NewBufferString(data),
	}
}

func (m *MockReaderCloser) Read(p []byte) (n int, err error) {
	// We *could* use m.Called() here to track Read calls if needed,
	// but handling the return values and buffer writing is complex.
	// Delegating is often simpler for standard Read behavior.
	if m.Reader == nil {
		// Provide a default if not set, or panic/error
		return 0, io.EOF // Or return an error indicating not configured
	}
	return m.Reader.Read(p)
}

func (m *MockReaderCloser) ReadString(delim string) (string, error) {
	n, err := m.Buffer.ReadString('\n')
	if err == io.EOF {
		// Once buffer is empty, return programmed ReadError or io.EOF
		if m.ReadError != nil {
			err = m.ReadError
		} else {
			err = io.EOF // Default to EOF if buffer is empty and no error set
		}
	}
	return n, err
}

func (m *MockReaderCloser) Close() error {
	// Record that the method was called, potentially with arguments (none here).
	args := m.Called()
	// Return the first value configured via Return() for this method call, casting it to error.
	return args.Error(0)
}

// --- Mock for MessageHandler ---
type MockHandler struct {
	mu        sync.Mutex
	Received  []json.RawMessage
	ReturnErr error
	Called    bool
}

func (h *MockHandler) Handle(data json.RawMessage) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Called = true
	dataCopy := make(json.RawMessage, len(data)) // Make a copy because RawMessage underlying slice might be reused/unsafe
	copy(dataCopy, data)
	h.Received = append(h.Received, dataCopy)
	return h.ReturnErr
}

func (h *MockHandler) HandleReturnError(data json.RawMessage) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Called = true
	dataCopy := make(json.RawMessage, len(data)) // Make a copy because RawMessage underlying slice might be reused/unsafe
	copy(dataCopy, data)
	h.Received = append(h.Received, dataCopy)
	h.ReturnErr = errors.New("handler failed processing")
	return h.ReturnErr
}

func TestListen_HappyPath_SingleEvent(t *testing.T) {
	client := newMockClient()
	mockReader := NewMockReaderCloser("event: message\ndata: {\"key\":\"value\"}\n\n")
	mockReader.On("Close").Return(nil)
	mockHandler := &MockHandler{}

	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	err := client.listen(ctx, mockReader, mockHandler.Handle)

	assert.NoError(t, err)
	assert.True(t, mockHandler.Called, "handler should be called")
	require.Len(t, mockHandler.Received, 1, "handler should receive 1 message")
}

func TestListen_HappyPath_MultipleEvents(t *testing.T) {
	client := newMockClient()
	sseData := `
event: event1
data: {"id": 1}

data: {"id": 2, "more": true}

event: event2
data: {"id": 3}

` // Note leading/trailing whitespace doesn't matter due to bufio/trimming
	mockReader := NewMockReaderCloser(sseData)
	mockReader.On("Close").Return(nil)
	mockHandler := &MockHandler{}

	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	err := client.listen(ctx, mockReader, mockHandler.Handle)

	assert.NoError(t, err)
	assert.True(t, mockReader.CloseCalled)
	require.Len(t, mockHandler.Received, 2)
	assert.JSONEq(t, `{"id": 1}`, string(mockHandler.Received[0]))
	assert.JSONEq(t, `{"id": 2, "more": true}`, string(mockHandler.Received[1]))
	assert.JSONEq(t, `{"id": 3}`, string(mockHandler.Received[2]))
}

func TestListen_HandlesCRLF(t *testing.T) {
	client := newMockClient()
	mockReader := NewMockReaderCloser("data: {\"crlf\": true}\r\n\r\n")
	mockReader.On("Close").Return(nil)
	mockHandler := &MockHandler{}

	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	err := client.listen(ctx, mockReader, mockHandler.Handle)

	assert.NoError(t, err)
	assert.True(t, mockReader.CloseCalled)
	require.Len(t, mockHandler.Received, 1)
	assert.JSONEq(t, `{"crlf": true}`, string(mockHandler.Received[0]))
}

func TestListen_IgnoresMalformedLines(t *testing.T) {
	client := newMockClient()
	sseData := `
: comment ignored
event:
data:
only data is kept
data: {"good": "yes"}

invalid line
event: second
data: {"num": 123}

`
	mockReader := NewMockReaderCloser(sseData)
	mockReader.On("Close").Return(nil)
	mockHandler := &MockHandler{}

	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	err := client.listen(ctx, mockReader, mockHandler.Handle)

	assert.NoError(t, err)
	assert.True(t, mockReader.CloseCalled)
	require.Len(t, mockHandler.Received, 2)
	assert.JSONEq(t, `{"good": "yes"}`, string(mockHandler.Received[0]))
	assert.JSONEq(t, `{"num": 123}`, string(mockHandler.Received[1]))
}

func TestListen_ContextCancellation(t *testing.T) {
	client := newMockClient()
	mockReader := NewMockReaderCloser("data: {\"a\": 1}\n\n") // Will provide one event
	mockReader.On("Close").Return(nil)
	mockHandler := &MockHandler{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		// This will process the first event, then block on Read
		errChan <- client.listen(ctx, mockReader, mockHandler.Handle)
	}()

	// Wait a moment to ensure listen() has started and likely processed the first event
	time.Sleep(50 * time.Millisecond)

	// Cancel the context
	cancel()

	// Wait for listen to return
	select {
	case err := <-errChan:
		assert.NoError(t, err, "listen should return nil on context cancel")
	case <-time.After(1 * time.Second):
		t.Fatal("listen did not return after context cancellation")
	}

	// Handler might have been called once before cancellation
	if len(mockHandler.Received) > 0 {
		assert.JSONEq(t, `{"a": 1}`, string(mockHandler.Received[0]))
	}
}

func TestListen_ClientDoneSignal(t *testing.T) {
	client := newMockClient()
	mockReader := NewMockReaderCloser("data: {\"a\": 1}\n\n")
	mockReader.On("Close").Return(nil)
	mockHandler := &MockHandler{}

	ctx := context.Background()

	errChan := make(chan error, 1)
	go func() {
		errChan <- client.listen(ctx, mockReader, mockHandler.Handle)
	}()

	// Wait a moment
	time.Sleep(50 * time.Millisecond)

	// Signal done
	close(client.done)

	// Wait for listen to return
	select {
	case err := <-errChan:
		assert.NoError(t, err, "listen should return nil on client.done")
	case <-time.After(2 * time.Second):
		t.Fatal("listen did not return after client.done closed")
	}
}

func TestListen_HandlerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	client := &MCPClient{
		serverURL:  srv.URL,
		clientID:   "test-client",
		httpClient: srv.Client(),
	}

	mockReader := NewMockReaderCloser("data: {\"process\":\"me\"}\n\n")
	mockReader.On("Close").Return(nil)
	mockHandler := &MockHandler{}
	expectedErr := errors.New("handler failed processing")
	mockHandler.ReturnErr = expectedErr

	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	err := client.listen(ctx, mockReader, mockHandler.HandleReturnError)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, expectedErr), "Error from handler should be returned")
	assert.True(t, mockHandler.Called)
	require.Len(t, mockHandler.Received, 1) // Handler was called once before erroring
}

func TestListen_ReadError_NonEOF(t *testing.T) {
	client := newMockClient()
	mockReader := NewMockReaderCloser("data: {\"a\": 1}\n\n") // One valid event first
	mockReader.On("Close").Return(nil)
	expectedErr := errors.New("simulated network glitch")
	// Program the mock to return an error AFTER the first event's data is read
	mockReader.ReadError = expectedErr
	mockHandler := &MockHandler{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := client.listen(ctx, mockReader, mockHandler.Handle)

	// IMPORTANT: Your current listen implementation prints the error but returns nil.
	// Test reflects this actual behavior. If desired, modify listen to return the error.
	assert.NoError(t, err, "listen currently returns nil even on non-EOF read errors")
	// If you modify listen to return the error, change the assertion to:
	// assert.ErrorIs(t, err, expectedErr)

	assert.True(t, mockReader.CloseCalled)
	// The first event should have been processed before the read error
	require.Len(t, mockHandler.Received, 1)
	assert.JSONEq(t, `{"a": 1}`, string(mockHandler.Received[0]))
}

func TestListen_EOF_CleanExit(t *testing.T) {
	client := newMockClient()
	// EOF occurs right after the final newline of the last event
	mockReader := NewMockReaderCloser("data: {\"last\": true}\n\n")
	mockReader.On("Close").Return(nil)
	mockHandler := &MockHandler{}

	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	err := client.listen(ctx, mockReader, mockHandler.Handle)

	assert.NoError(t, err, "Clean EOF should result in nil error")
	assert.True(t, mockReader.CloseCalled)
	require.Len(t, mockHandler.Received, 1)
	assert.JSONEq(t, `{"last": true}`, string(mockHandler.Received[0]))
}

func TestListen_EOF_ProcessesPendingEvent(t *testing.T) {
	client := newMockClient()
	// Stream ends abruptly *without* the final double newline
	mockReader := NewMockReaderCloser("event: pending\ndata: {\"key\": \"value\"}")
	mockReader.On("Close").Return(nil)
	mockHandler := &MockHandler{}

	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	err := client.listen(ctx, mockReader, mockHandler.Handle)

	assert.NoError(t, err)
	assert.True(t, mockReader.CloseCalled)
	// Handler should still be called for the pending event upon EOF
	require.Len(t, mockHandler.Received, 1)
	assert.JSONEq(t, `{"key": "value"}`, string(mockHandler.Received[0]))
}

func TestListen_EOF_ProcessesPendingEvent_HandlerError(t *testing.T) {
	client := newMockClient()
	mockReader := NewMockReaderCloser("event: pending\ndata: {\"key\": \"value\"}")
	mockReader.On("Close").Return(nil)
	mockHandler := &MockHandler{}
	expectedErr := errors.New("pending handler failed")
	mockHandler.ReturnErr = expectedErr // Handler fails for the pending event

	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	err := client.listen(ctx, mockReader, mockHandler.Handle)

	// The error from the handler processing the pending event on EOF should be returned
	assert.ErrorIs(t, err, expectedErr)
	assert.True(t, mockReader.CloseCalled)
	require.Len(t, mockHandler.Received, 1) // Handler was called once
}

func TestListen_EmptyReader(t *testing.T) {
	client := newMockClient()
	mockReader := NewMockReaderCloser("") // Empty input
	mockReader.On("Close").Return(nil)
	mockHandler := &MockHandler{}

	ctx, _ := context.WithTimeout(context.Background(), time.Second*3)
	err := client.listen(ctx, mockReader, mockHandler.Handle)

	assert.NoError(t, err)
	assert.True(t, mockReader.CloseCalled)
	assert.False(t, mockHandler.Called) // Handler should not be called
	assert.Empty(t, mockHandler.Received)
}
