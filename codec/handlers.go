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
	if req.JSONRPC != JsonRPCVersion {
		return nil, errors.New("invalid jsonrpc version")
	}
	if req.Method == "" {
		return nil, errors.New("missing method")
	}
	return &req, nil
}

func WriteJSONRPCResponse(w http.ResponseWriter, result json.RawMessage, id int64) error {
	resp := JSONRPCResponse{
		JSONRPC: JsonRPCVersion,
		Result:  result,
		ID:      id,
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(resp)
}

func WriteJSONRPCError(w http.ResponseWriter, code int, message string, id int64) error {
	if message == "" {
		message = rpcErrorMessages[code]
	}
	resp := JSONRPCResponse{
		JSONRPC: JsonRPCVersion,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
		},
		ID: id,
	}
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(resp)
}
