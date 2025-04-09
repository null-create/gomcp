package codec

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

const (
	// DefaultProtocolVersion defines a fallback or standard version if negotiation fails simply.
	// In reality, the server dictates the chosen version based on the client's offer.
	DefaultProtocolVersion = "2024-11-05"
	JsonRPCVersion         = "2.0"
)

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	ID      any             `json:"id"`
}

type JSONRPCResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
	ID      any       `json:"id"`
}

func (j *JSONRPCResponse) Bytes() []byte {
	b, err := json.Marshal(j.Result)
	if err != nil {
		log.Fatal(err)
	}
	return b
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"` // Often null/omitted for simple notifications
}

// JSON-RPC 2.0 standard error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

var rpcErrorMessages = map[int]string{
	ParseError:     "Parse error",
	InvalidRequest: "Invalid Request",
	MethodNotFound: "Method not found",
	InvalidParams:  "Invalid params",
	InternalError:  "Internal error",
}

func ParseJSONRPCRequest(r *http.Request) (*JSONRPCRequest, error) {
	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	if req.JSONRPC != "2.0" {
		return nil, errors.New("invalid jsonrpc version")
	}
	if req.Method == "" {
		return nil, errors.New("missing method")
	}
	return &req, nil
}

func WriteJSONRPCResponse(w http.ResponseWriter, result any, id any) error {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(resp)
}

func WriteJSONRPCError(w http.ResponseWriter, code int, message string, id any) error {
	if message == "" {
		message = rpcErrorMessages[code]
	}
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
		ID: id,
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(resp)
}
