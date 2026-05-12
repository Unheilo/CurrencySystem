package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"my-currency-service/auth/internal/config"
	"my-currency-service/auth/internal/handler"
	"my-currency-service/auth/internal/users"
	"my-currency-service/pkg/logger"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	l, err := logger.Setup(cfg.Env)
	if err != nil {
		return fmt.Errorf("logger: %w", err)
	}

	store := users.New(cfg.Users)
	srv := handler.NewServer(
		l,
		store,
		[]byte(cfg.JWTSecret),
		time.Duration(cfg.TokenTTLMinutes)*time.Minute,
		cfg.Issuer,
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/auth", srv.Auth)
	mux.HandleFunc("/healthz", handler.Health)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		l.Info("auth http listening", slog.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		l.Info("shutdown signal received")
	case err := <-errCh:
		return fmt.Errorf("listen: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	l.Info("stopped")
	return nil
}
