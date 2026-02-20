package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

type AuthHandler struct {
	SupabaseURL     string
	SupabaseAnonKey string
}

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	h.proxyAuth(w, r, "/auth/v1/signup")
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	h.proxyAuth(w, r, "/auth/v1/token?grant_type=password")
}

func (h *AuthHandler) proxyAuth(w http.ResponseWriter, r *http.Request, path string) {
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	body, _ := json.Marshal(req)
	proxyReq, err := http.NewRequestWithContext(
		r.Context(),
		http.MethodPost,
		h.SupabaseURL+path,
		bytes.NewReader(body),
	)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("apikey", h.SupabaseAnonKey)

	resp, err := http.DefaultClient.Do(proxyReq)
	if err != nil {
		http.Error(w, `{"error":"failed to reach auth service"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
