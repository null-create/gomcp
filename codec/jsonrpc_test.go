package codec

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
)

func TestParseJSONRPCRequest(t *testing.T) {
	requestData := JSONRPCRequest{
		JSONRPC: JsonRPCVersion,
		Method:  "test_method",
		Params:  json.RawMessage(`{"key":"value"}`),
		ID:      1,
	}
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(requestData)
	if err != nil {
		t.Fatalf("failed to encode request: %v", err)
	}
	r := httptest.NewRequest("POST", "/rpc", buf)

	parsedReq, err := ParseJSONRPCRequest(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsedReq.Method != requestData.Method {
		t.Errorf("expected method %s, got %s", requestData.Method, parsedReq.Method)
	}
	if parsedReq.JSONRPC != JsonRPCVersion {
		t.Errorf("expected jsonrpc %s, got %s", JsonRPCVersion, parsedReq.JSONRPC)
	}
}

func TestWriteJSONRPCResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	WriteJSONRPCResponse(recorder, map[string]string{"result": "ok"}, 42)

	res := recorder.Result()
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var response JSONRPCResponse
	err := json.Unmarshal(body, &response)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.JSONRPC != JsonRPCVersion {
		t.Errorf("expected jsonrpc %s, got %s", JsonRPCVersion, response.JSONRPC)
	}
	if response.ID.(float64) != 42 {
		t.Errorf("expected 42, got %v", response.ID)
	}
	if response.Result == nil {
		t.Errorf("expected result, got nil")
	}
}

func TestWriteJSONRPCError(t *testing.T) {
	recorder := httptest.NewRecorder()
	WriteJSONRPCError(recorder, -32601, "Method not found", "abc")

	res := recorder.Result()
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	var response JSONRPCResponse
	err := json.Unmarshal(body, &response)
	if err != nil {
		t.Fatalf("failed to unmarshal error response: %v", err)
	}

	if response.JSONRPC != JsonRPCVersion {
		t.Errorf("expected jsonrpc %s, got %s", JsonRPCVersion, response.JSONRPC)
	}
	if response.Error == nil {
		t.Fatal("expected error object, got nil")
	}
	if response.Error.Code != -32601 {
		t.Errorf("expected error code -32601, got %d", response.Error.Code)
	}
	if response.ID != "abc" {
		t.Errorf("expected id 'abc', got %v", response.ID)
	}
}
