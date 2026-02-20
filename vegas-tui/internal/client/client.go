package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	BaseURL    string
	Token      string
	httpClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		BaseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

type AuthRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func (c *Client) Signup(email, password string) (*AuthResponse, error) {
	return c.doAuth("/auth/signup", email, password)
}

func (c *Client) Login(email, password string) (*AuthResponse, error) {
	return c.doAuth("/auth/login", email, password)
}

func (c *Client) doAuth(path, email, password string) (*AuthResponse, error) {
	body, _ := json.Marshal(AuthRequest{Email: email, Password: password})
	resp, err := c.httpClient.Post(c.BaseURL+path, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("auth error (%d): %s", resp.StatusCode, respBody)
	}

	var authResp AuthResponse
	if err := json.Unmarshal(respBody, &authResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	c.Token = authResp.AccessToken
	return &authResp, nil
}

func (c *Client) Do(method, path string, payload any) (*http.Response, error) {
	var body io.Reader
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshalling payload: %w", err)
		}
		body = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	return c.httpClient.Do(req)
}
