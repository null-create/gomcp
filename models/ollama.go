package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func QueryOllama(model string, prompt string) (string, error) {
	reqBody := map[string]any{
		"model":  model,
		"prompt": prompt,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	resp, err := http.Post("http://localhost:11434/api/generate", "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Response string `json:"response"`
	}

	data, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("bad ollama response: %s", data)
	}

	return result.Response, nil
}
