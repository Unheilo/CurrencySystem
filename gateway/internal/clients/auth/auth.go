package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Auth struct {
	baseURL    string
	httpClient *http.Client
}

func NewAuth(baseURL string) *Auth {
	return &Auth{baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type authRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
}

func (a *Auth) Authenticate(ctx context.Context, login, password string) (string, error) {
	body, _ := json.Marshal(authRequest{Login: login, Password: password})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/auth", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("auth failed: %s, body: %s", resp.Status, msg)
	}

	var out authResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode :%w", err)
	}
	return out.Token, nil
}
