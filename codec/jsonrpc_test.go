package codec

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseJSONRPCRequest(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid request",
			body:        `{"jsonrpc":"2.0", "method":"sum", "params":[1,2], "id":1}`,
			expectError: false,
		},
		{
			name:        "invalid json",
			body:        `{"jsonrpc": "2.0", "method": "sum",`,
			expectError: true,
		},
		{
			name:        "invalid version",
			body:        `{"jsonrpc": "1.0", "method": "sum", "id":1}`,
			expectError: true,
			errorMsg:    "invalid jsonrpc version",
		},
		{
			name:        "missing method",
			body:        `{"jsonrpc": "2.0", "id":1}`,
			expectError: true,
			errorMsg:    "missing method",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.body))
			result, err := ParseJSONRPCRequest(req)
			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("expected error %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result.Method != "sum" {
					t.Errorf("expected method 'sum', got '%s'", result.Method)
				}
			}
		})
	}
}

func TestWriteJSONRPCResponse(t *testing.T) {
	rr := httptest.NewRecorder()
	result := JSONRPCResponse{
		JSONRPC: JsonRPCVersion,
		Result:  []byte("2"),
		Error:   nil,
		ID:      1,
	}
	r, err := json.Marshal(result)
	if err != nil {
		t.Error(err)
	}

	err = WriteJSONRPCResponse(rr, r, 1)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp := rr.Result()
	defer resp.Body.Close()

	if contentType := resp.Header.Get("Content-Type"); contentType != "application/json" {
		t.Errorf("expected content-type application/json, got %s", contentType)
	}

	var jsonResp JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		t.Fatalf("error decoding response: %v", err)
	}

	if jsonResp.JSONRPC != JsonRPCVersion {
		t.Errorf("expected version %s, got %s", JsonRPCVersion, jsonResp.JSONRPC)
	}

	var testResult map[string]interface{}
	if err := json.Unmarshal(jsonResp.Result, &testResult); err != nil {
		t.Errorf("failed to unmarshal test result: %v", err)
	}

	if testResult["result"].(float64) != 2 {
		t.Errorf("expected result '2', got %v", jsonResp.Result)
	}

	if jsonResp.Error != nil {
		t.Errorf("expected no error, got %v", jsonResp.Error)
	}
}

func TestWriteJSONRPCError(t *testing.T) {
	rr := httptest.NewRecorder()
	err := WriteJSONRPCError(rr, -32600, "", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	resp := rr.Result()
	defer resp.Body.Close()

	var jsonResp JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		t.Fatalf("error decoding response: %v", err)
	}

	if jsonResp.Error == nil {
		t.Fatal("expected error in response")
	}
	if jsonResp.Error.Code != -32600 {
		t.Errorf("expected error code -32600, got %d", jsonResp.Error.Code)
	}
	if jsonResp.Error.Message != rpcErrorMessages[-32600] {
		t.Errorf("expected message %q, got %q", rpcErrorMessages[-32600], jsonResp.Error.Message)
	}
}
