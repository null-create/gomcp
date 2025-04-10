package codec

import (
	"encoding/json"
	"errors"
	"net/http"
)

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
