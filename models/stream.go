package models

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func StreamResponse(model, prompt string) (io.ReadCloser, error) {
	switch model {
	case "ollama":
		return streamOllama(model, prompt)
	default:
		return nil, fmt.Errorf("streaming not implemented for model: %s", model)
	}
}

func streamOllama(model, prompt string) (io.ReadCloser, error) {
	body := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": true,
	}
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(body); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "http://localhost:11434/api/generate", buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("ollama returned status: %s", resp.Status)
	}
	return resp.Body, nil
}
