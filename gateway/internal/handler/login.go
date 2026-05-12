package handler

import (
	"encoding/json"
	"log/slog"
	client "my-currency-service/gateway/internal/clients/auth"
	"net/http"
)

type LoginHandler struct {
	log    *slog.Logger
	client *client.Auth
}

func NewLoginHandler(log *slog.Logger, c *client.Auth) *LoginHandler {
	return &LoginHandler{log: log, client: c}
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func (h *LoginHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if req.Login == "" || req.Password == "" {
		http.Error(w, "login and password required", http.StatusBadRequest)
		return
	}
	tok, err := h.client.Authenticate(r.Context(), req.Login, req.Password)
	if err != nil {
		h.log.WarnContext(r.Context(), "auth failed", slog.Any("error", err))
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(loginResponse{Token: tok})
}
