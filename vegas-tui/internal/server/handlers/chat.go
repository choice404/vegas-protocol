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

func (h *ChatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Prompt == "" {
		http.Error(w, `{"error":"prompt is required"}`, http.StatusBadRequest)
		return
	}

	model := req.Model
	if model == "" {
		model = "llama3"
	}

	ollamaReq := ollamaGenerateRequest{
		Model:  model,
		Prompt: req.Prompt,
		System: req.System,
		Stream: false,
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		http.Error(w, `{"error":"failed to marshal request"}`, http.StatusInternalServerError)
		return
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(h.OllamaURL+"/api/generate", "application/json", bytes.NewReader(body))
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"ollama unreachable: %s"}`, err.Error()), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, `{"error":"failed to read ollama response"}`, http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf(`{"error":"ollama error (%d): %s"}`, resp.StatusCode, string(respBody)), resp.StatusCode)
		return
	}

	var ollamaResp ollamaGenerateResponse
	if err := json.Unmarshal(respBody, &ollamaResp); err != nil {
		http.Error(w, `{"error":"failed to parse ollama response"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"response": ollamaResp.Response,
	})
}
