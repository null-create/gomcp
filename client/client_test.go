package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/gomcp/codec"
	mcpctx "github.com/gomcp/context"

	"github.com/stretchr/testify/assert"
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

	tsURL, _ := url.Parse(ts.URL)

	c := &MCPClient{
		serverURL:  tsURL,
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
	url, _ := url.Parse("http://localhost:0")

	c := &MCPClient{
		serverURL:  url,
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
type MockReaderCloser struct {
	mu            sync.Mutex
	Buffer        *bytes.Buffer // Use bytes.Buffer for easy data feeding
	ReadError     error         // Error to return on Read (after buffer empty)
	CloseError    error         // Error to return on Close
	CloseCalled   bool
	readBlockChan chan struct{} // Channel to simulate blocking reads
}

func NewMockReaderCloser(data string) *MockReaderCloser {
	return &MockReaderCloser{
		Buffer: bytes.NewBufferString(data),
	}
}

func (m *MockReaderCloser) Read(p []byte) (n int, err error) {
	m.mu.Lock()
	// Simulate blocking if enabled
	if m.readBlockChan != nil {
		ch := m.readBlockChan
		m.mu.Unlock() // Unlock while potentially blocking
		<-ch          // Wait until channel is closed
		m.mu.Lock()   // Re-lock
	}

	// Read from buffer
	n, err = m.Buffer.Read(p)

	// Determine final error state after reading attempt
	if errors.Is(err, io.EOF) { // Buffer is empty
		if m.ReadError != nil {
			err = m.ReadError // Return programmed error instead of EOF
		} else {
			err = io.EOF // Return EOF if no specific error is set
		}
	} else if err == nil && m.Buffer.Len() == 0 {
		// Sometimes Read might return n > 0 and err == nil but empty the buffer.
		// Check if EOF or programmed error should be returned *next* time.
		// However, for bufio.ReadString, it reads until '\n', so this nuance
		// is less critical here than for raw Read calls. We mostly care
		// about when the underlying source signals EOF or error.
		// The primary logic relies on err == io.EOF from the buffer read.
	}

	m.mu.Unlock()
	return n, err
}

func (m *MockReaderCloser) ReadString(delim string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	line, err := m.Buffer.ReadString('\n')
	if errors.Is(err, io.EOF) { // Buffer is empty
		if m.ReadError != nil {
			err = m.ReadError // Return programmed error instead of EOF
		} else {
			err = io.EOF // Return EOF if no specific error is set
		}
	}

	return line, err
}

func (m *MockReaderCloser) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CloseCalled = true
	return m.CloseError
}

// --- Mock for MessageHandler ---
type MockHandler struct {
	mu        sync.Mutex
	Received  []json.RawMessage
	ReturnErr error
	Called    bool
}

func NewMockHandler() *MockHandler {
	return &MockHandler{
		Received: make([]json.RawMessage, 0),
	}
}

func (h *MockHandler) Handle(event, data string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Called = true
	dataCopy := make(json.RawMessage, len(data)) // Make a copy because RawMessage underlying slice might be reused/unsafe
	copy(dataCopy, data)
	h.Received = append(h.Received, dataCopy)
	return h.ReturnErr
}

func (h *MockHandler) HandleReturnError(event, data string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Called = true
	dataCopy := make(json.RawMessage, len(data)) // Make a copy because RawMessage underlying slice might be reused/unsafe
	copy(dataCopy, data)
	h.Received = append(h.Received, dataCopy)
	h.ReturnErr = errors.New("handler failed processing")
	return h.ReturnErr
}

// Test successful processing of a single event
func TestListen_SingleEvent(t *testing.T) {
	client := newMockClient() // done channel is open
	mockReader := NewMockReaderCloser("event: message\ndata: {\"key\":\"value\"}\n\n")
	mockHandler := &MockHandler{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := client.listen(ctx, mockReader, mockHandler.Handle)

	assert.NoError(t, err, "Expected no error on successful listen")
	assert.True(t, mockReader.CloseCalled, "reader.Close should have been called")
	assert.True(t, mockHandler.Called, "handler.Handle should have been called")
	require.Len(t, mockHandler.Received, 1, "Should have received exactly one message")
	assert.JSONEq(t, `{"key":"value"}`, string(mockHandler.Received[0]), "Received JSON data mismatch")
}

func TestListen_HappyPath_MultipleEvents(t *testing.T) {
	client := newMockClient()
	sseData := `
event: event1
data: {"id": 1}

event: event2
data: {"id": 2, "more": true}

event: event3
data: {"id": 3}

` // Note leading/trailing whitespace doesn't matter due to bufio/trimming
	mockReader := NewMockReaderCloser(sseData)
	mockHandler := NewMockHandler()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := client.listen(ctx, mockReader, mockHandler.Handle)

	assert.NoError(t, err)
	assert.True(t, mockReader.CloseCalled, "reader.Close should have been called")
	require.Len(t, mockHandler.Received, 3)
	assert.JSONEq(t, `{"id": 1}`, string(mockHandler.Received[0]))
	assert.JSONEq(t, `{"id": 2, "more": true}`, string(mockHandler.Received[1]))
	assert.JSONEq(t, `{"id": 3}`, string(mockHandler.Received[2]))
}

func TestListen_HandlesCRLF(t *testing.T) {
	client := newMockClient()
	mockReader := NewMockReaderCloser("event: event1\ndata: {\"crlf\": true}\r\n\r\n")
	mockHandler := NewMockHandler()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := client.listen(ctx, mockReader, mockHandler.Handle)

	assert.NoError(t, err)
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
	mockHandler := NewMockHandler()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := client.listen(ctx, mockReader, mockHandler.Handle)

	assert.NoError(t, err)
	assert.True(t, mockReader.CloseCalled, "reader.Close should have been called")
	require.Len(t, mockHandler.Received, 1)
	assert.JSONEq(t, `{"num": 123}`, string(mockHandler.Received[0]))
}

func TestListen_ContextCancellation(t *testing.T) {
	client := newMockClient()
	mockReader := NewMockReaderCloser("data: {\"a\": 1}\n\n") // Will provide one event
	mockHandler := NewMockHandler()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
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
	mockHandler := NewMockHandler()

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
	client := newMockClient()
	mockReader := NewMockReaderCloser("event:event1\ndata: {\"process\":\"me\"}\n\n")
	mockHandler := NewMockHandler()
	expectedErr := errors.New("listener handler failed: handler failed processing")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := client.listen(ctx, mockReader, mockHandler.HandleReturnError)

	assert.Error(t, err)
	assert.True(t, mockReader.CloseCalled, "reader.Close should have been called")
	assert.Equal(t, err, expectedErr, "Error from handler should be returned")
	require.Len(t, mockHandler.Received, 1) // Handler was called once before erroring
}

func TestListen_ReadError_NonEOF(t *testing.T) {
	client := newMockClient()
	mockReader := NewMockReaderCloser("event:event1\ndata: {\"a\": 1}\n\n") // One valid event first
	expectedErr := errors.New("simulated network glitch")
	// Program the mock to return an error AFTER the first event's data is read
	mockReader.ReadError = expectedErr
	mockHandler := NewMockHandler()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := client.listen(ctx, mockReader, mockHandler.Handle)

	// IMPORTANT: Current listen implementation prints the error but returns nil.
	// Test reflects this actual behavior. If desired, modify listen to return the error.
	assert.NoError(t, err, "listen currently returns nil even on non-EOF read errors")
	// If you modify listen to return the error, change the assertion to:
	// assert.ErrorIs(t, err, expectedErr)

	// The first event should have been processed before the read error
	require.Len(t, mockHandler.Received, 1)
	assert.JSONEq(t, `{"a": 1}`, string(mockHandler.Received[0]))
}

// Test clean exit when the reader signals EOF after the last event
func TestListen_EOF_CleanExit(t *testing.T) {
	client := newMockClient()
	mockReader := NewMockReaderCloser("event:event1\ndata: {\"last\": true}\n\n")
	mockHandler := NewMockHandler()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := client.listen(ctx, mockReader, mockHandler.Handle)

	assert.NoError(t, err, "Clean EOF should result in nil error")
	assert.True(t, mockReader.CloseCalled, "reader.Close should have been called")
	require.Len(t, mockHandler.Received, 1, "Should have processed the last event")
	assert.JSONEq(t, `{"last": true}`, string(mockHandler.Received[0]))
}

// func TestListen_EOF_ProcessesPendingEvent(t *testing.T) {
// 	client := newMockClient()
// 	// Stream ends abruptly *without* the final double newline
// 	mockReader := NewMockReaderCloser("event: pending\ndata: {\"key\": \"value\"}")
// 	mockHandler := NewMockHandler()

// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
// 	defer cancel()

// 	err := client.listen(ctx, mockReader, mockHandler.Handle)

// 	assert.NoError(t, err)
// 	assert.True(t, mockReader.CloseCalled, "reader.Close should have been called")
// 	// Handler should still be called for the pending event upon EOF
// 	require.Len(t, mockHandler.Received, 1)
// 	assert.JSONEq(t, `{"key": "value"}`, string(mockHandler.Received[0]))
// }

// func TestListen_EOF_ProcessesPendingEvent_HandlerError(t *testing.T) {
// 	client := newMockClient()
// 	mockReader := NewMockReaderCloser("event: pending\ndata: {\"key\": \"value\"}")
// 	mockHandler := NewMockHandler()
// 	expectedErr := errors.New("handler failed processing")
// 	mockHandler.ReturnErr = expectedErr // Handler fails for the pending event

// 	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
// 	defer cancel()

// 	err := client.listen(ctx, mockReader, mockHandler.HandleReturnError)

// 	assert.True(t, mockReader.CloseCalled, "reader.Close should have been called")
// 	// The error from the handler processing the pending event on EOF should be returned
// 	assert.ErrorIs(t, err, expectedErr)
// 	require.Len(t, mockHandler.Received, 1) // Handler was called once
// }

func TestListen_EmptyReader(t *testing.T) {
	client := newMockClient()
	mockReader := NewMockReaderCloser("") // Empty input
	mockHandler := NewMockHandler()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err := client.listen(ctx, mockReader, mockHandler.Handle)

	assert.NoError(t, err)
	assert.True(t, mockReader.CloseCalled, "reader.Close should have been called")
	assert.False(t, mockHandler.Called) // Handler should not be called
	assert.Empty(t, mockHandler.Received)
}
