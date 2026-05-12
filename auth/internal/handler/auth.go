package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"my-currency-service/auth/internal/users"

	"github.com/golang-jwt/jwt/v5"
)

type Server struct {
	log    *slog.Logger
	users  *users.Store
	secret []byte
	ttl    time.Duration
	issuer string
}

func NewServer(log *slog.Logger, store *users.Store, secret []byte, ttl time.Duration, issuer string) *Server {
	return &Server{log: log, users: store, secret: secret, ttl: ttl, issuer: issuer}
}

type authRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
}

type errorResponse struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (s *Server) Auth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "use POST")
		return
	}
	if r.Header.Get("Content-Type") != "application/json" {
		writeErr(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA", "use application/json")
		return
	}

	var req authRequest
	dec := json.NewDecoder(io.LimitReader(r.Body, 4*1024)) // защита от гигантского body
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "BAD_JSON", "invalid request body")
		return
	}
	if req.Login == "" || req.Password == "" {
		writeErr(w, http.StatusBadRequest, "MISSING_FIELDS", "login and password are required")
		return
	}

	if !s.users.Verify(req.Login, req.Password) {
		s.log.InfoContext(r.Context(), "auth rejected",
			slog.String("login", req.Login),
		)
		writeErr(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid login or password")
		return
	}

	token, err := s.issueToken(req.Login)
	if err != nil {
		s.log.ErrorContext(r.Context(), "issue token", slog.Any("error", err))
		writeErr(w, http.StatusInternalServerError, "INTERNAL", "internal server error")
		return
	}

	s.log.InfoContext(r.Context(), "auth granted", slog.String("login", req.Login))
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(authResponse{Token: token})
}

func (s *Server) issueToken(sub string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   sub,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		NotBefore: jwt.NewNumericDate(now),
		Issuer:    s.issuer,
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(s.secret)
	if err != nil {
		return "", errors.New("sign failed")
	}
	return signed, nil
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: errorDetail{Code: code, Message: msg}})
}

func Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
