package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ChatHandler struct {
	OllamaURL string
}

type chatRequest struct {
	Prompt string `json:"prompt"`
	Model  string `json:"model"`
	System string `json:"system"`
	Stream bool   `json:"stream"`
}

type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	System string `json:"system"`
	Stream bool   `json:"stream"`
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
}

func jsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func (h *ChatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Prompt == "" {
		jsonError(w, "prompt is required", http.StatusBadRequest)
		return
	}

	model := req.Model
	if model == "" {
		model = "llama3.1:8b"
	}

	ollamaReq := ollamaGenerateRequest{
		Model:  model,
		Prompt: req.Prompt,
		System: req.System,
		Stream: false,
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		jsonError(w, "failed to marshal request", http.StatusInternalServerError)
		return
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(h.OllamaURL+"/api/generate", "application/json", bytes.NewReader(body))
	if err != nil {
		jsonError(w, fmt.Sprintf("ollama unreachable: %s", err.Error()), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		jsonError(w, "failed to read ollama response", http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != http.StatusOK {
		jsonError(w, fmt.Sprintf("ollama error (%d): %s", resp.StatusCode, string(respBody)), resp.StatusCode)
		return
	}

	var ollamaResp ollamaGenerateResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		jsonError(w, "failed to parse ollama response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"response": ollamaResp.Response,
	})
}
