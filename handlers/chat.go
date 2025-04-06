package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gomcp/models"
	"github.com/gomcp/types"
)

func HandleChatCompletion(w http.ResponseWriter, r *http.Request) {
	var req types.ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	modelName := req.Model
	content := req.Messages[len(req.Messages)-1].Content

	// Check for tool invocation
	if len(req.Tools) > 0 && strings.Contains(strings.ToLower(content), "tool:") {
		response, err := models.HandleToolCall(req.Tools, content)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"tool_response": response,
		})
		return
	}

	if req.Stream {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		reader, err := models.StreamResponse(modelName, content)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer reader.Close()

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintf(w, "data: %s\n\n", line)
			flusher.Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
		return
	}

	var response string
	var err error

	switch {
	case modelName == "claude":
		response, err = models.QueryClaude(content)
	case modelName == "ollama" || modelName == "llama3":
		response, err = models.QueryOllama(modelName, content)
	default:
		http.Error(w, fmt.Sprintf("Unsupported model: %s", modelName), http.StatusNotImplemented)
		return
	}

	if err != nil {
		http.Error(w, "Error from model: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := map[string]any{
		"id":      "chatcmpl-mockid",
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   modelName,
		"choices": []map[string]any{
			{
				"index": 0,
				"message": map[string]string{
					"role":    "assistant",
					"content": response,
				},
				"finish_reason": "stop",
			},
		},
		"usage": map[string]int{
			"prompt_tokens":     0,
			"completion_tokens": 0,
			"total_tokens":      0,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
