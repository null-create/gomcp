package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"testing"

	"github.com/gomcp/codec"
	"github.com/gomcp/mcp"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- MOCKS ---

type MockClientState struct {
	mock.Mock
	ClientState // embed for fallback if needed
}

func (m *MockClientState) CreateInitializeRequest() ([]byte, error) {
	args := m.Called()

	switch v := args.Get(0).(type) {
	case nil:
		return nil, args.Error(1)
	case []byte:
		return v, args.Error(1)
	default:
		return nil, fmt.Errorf("SendInitRequest: unexpected return type %T", v)
	}
}

func (m *MockClientState) SendInitRequest(req []byte) ([]byte, error) {
	args := m.Called(req)

	switch v := args.Get(0).(type) {
	case nil:
		return nil, args.Error(1)
	case []byte:
		return v, args.Error(1)
	default:
		return nil, fmt.Errorf("SendInitRequest: unexpected return type %T", v)
	}
}

func (m *MockClientState) ProcessInitializeResponse(resp codec.JSONRPCResponse) error {
	args := m.Called(resp)
	return args.Error(0)
}

func (m *MockClientState) CreateInitializedNotification() ([]byte, error) {
	args := m.Called()
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockClientState) SendInitNotification(req []byte) ([]byte, error) {
	args := m.Called(req)

	switch v := args.Get(0).(type) {
	case nil:
		return nil, args.Error(1)
	case []byte:
		return v, args.Error(1)
	default:
		return nil, fmt.Errorf("SendInitRequest: unexpected return type %T", v)
	}
}

// --- HANDSHAKE TESTS ---

func TestHandshake_Success(t *testing.T) {
	mockState := new(MockClientState)
	mockReq := []byte(`{"jsonrpc":"2.0","method":"initialize"}`)
	mockResp := []byte(`{"jsonrpc":"2.0","result":{"ok":true}}`)

	mockState.On("CreateInitializeRequest", mock.Anything).Return(mockReq, nil)
	mockState.On("SendInitRequest", mockReq).Return(mockResp, nil)
	mockState.On("ProcessInitializeResponse", mock.Anything).Return(nil)
	mockState.On("CreateInitializedNotification").Return([]byte(`{"jsonrpc":"2.0","method":"initialized"}`), nil)
	mockState.On("SendInitNotification", mock.Anything).Return([]byte(`{}`), nil)

	url, _ := url.Parse("http://test-server/init")
	client := &MCPClient{initURL: url}
	client.SetClientState(mockState) // assume this sets `cs` in test mode

	// simulate success by setting negotiated version + server info
	mockState.NegotiatedVersion = "2024-11-05"
	mockState.ServerInfo = &mcp.ServerInfo{}

	err := client.Handshake()
	assert.NoError(t, err)
	mockState.AssertExpectations(t)
}

func TestHandshake_CreateRequestFails(t *testing.T) {
	mockState := new(MockClientState)
	mockState.On("CreateInitializeRequest", mock.Anything).Return(nil, errors.New("fail to build"))

	url, _ := url.Parse("bad")

	client := &MCPClient{initURL: url}
	client.SetClientState(mockState)

	err := client.Handshake()
	assert.ErrorContains(t, err, "client failed to create initialize request")
}

func TestHandshake_InitRequestFails(t *testing.T) {
	mockState := new(MockClientState)
	mockState.On("CreateInitializeRequest", mock.Anything).Return([]byte(`{}`), nil)
	mockState.On("SendInitRequest", []byte(`{}`)).Return(nil, errors.New("timeout"))

	client := &MCPClient{}
	client.SetClientState(mockState)

	err := client.Handshake()
	assert.ErrorContains(t, err, "client init request failed")
}

func TestHandshake_JSONRPCErrorFromServer(t *testing.T) {
	rpcErr := &codec.JSONRPCError{Code: -32600, Message: "Invalid request"}
	jsonResp, _ := json.Marshal(codec.JSONRPCResponse{Error: rpcErr})

	mockState := new(MockClientState)
	mockState.On("CreateInitializeRequest", mock.Anything).Return([]byte(`{}`), nil)
	mockState.On("SendInitRequest", []byte(`{}`)).Return(jsonResp, nil)

	client := &MCPClient{}
	client.SetClientState(mockState)

	err := client.Handshake()
	assert.ErrorContains(t, err, "server returned JSON-RPC error")
}

func TestHandshake_ProcessInitResponseFails(t *testing.T) {
	mockResp := []byte(`{"jsonrpc":"2.0","result":{}}`)
	mockState := new(MockClientState)
	mockState.On("CreateInitializeRequest", mock.Anything).Return([]byte(`{}`), nil)
	mockState.On("SendInitRequest", mock.Anything).Return(mockResp, nil)
	mockState.On("ProcessInitializeResponse", mock.Anything).Return(errors.New("processing failed"))

	client := &MCPClient{}
	client.SetClientState(mockState)

	err := client.Handshake()
	assert.ErrorContains(t, err, "client failed to process initialize response")
}

func TestHandshake_InitializedNotificationFails(t *testing.T) {
	mockResp := []byte(`{"jsonrpc":"2.0","result":{}}`)
	mockState := new(MockClientState)
	mockState.NegotiatedVersion = "2024-11-05"
	mockState.ServerInfo = &mcp.ServerInfo{}

	mockState.On("CreateInitializeRequest", mock.Anything).Return([]byte(`{}`), nil)
	mockState.On("SendInitRequest", mock.Anything).Return(mockResp, nil)
	mockState.On("ProcessInitializeResponse", mock.Anything).Return(nil)
	mockState.On("CreateInitializedNotification").Return([]byte(`init`), nil)
	mockState.On("SendInitNotification", []byte(`init`)).Return(nil, errors.New("send fail"))

	client := &MCPClient{}
	client.SetClientState(mockState)

	err := client.Handshake()
	assert.ErrorContains(t, err, "client failed to send init notification")
}

func TestHandshake_MissingNegotiatedVersionOrInfo(t *testing.T) {
	mockResp := []byte(`{"jsonrpc":"2.0","result":{}}`)
	mockState := new(MockClientState)

	mockState.On("CreateInitializeRequest", mock.Anything).Return([]byte(`{}`), nil)
	mockState.On("SendInitRequest", mock.Anything).Return(mockResp, nil)
	mockState.On("ProcessInitializeResponse", mock.Anything).Return(nil)

	// Note: no negotiated version or server info
	client := &MCPClient{}
	client.SetClientState(mockState)

	err := client.Handshake()
	assert.ErrorContains(t, err, "client handshake failed. no negotiated version")
}
