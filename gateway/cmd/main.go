package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	client "my-currency-service/gateway/internal/clients"
	"my-currency-service/gateway/internal/config"
	"my-currency-service/gateway/internal/logger"
	"my-currency-service/gateway/internal/handler"
	"my-currency-service/gateway/internal/middleware"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
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

	currencyClient, err := client.NewCurrency(cfg.CurrencyGRPCAddr)
	if err != nil {
		return fmt.Errorf("currency client: %w", err)
	}
	defer currencyClient.Close()

	rateH := handler.NewRateHandler(l, currencyClient)

	r := chi.NewRouter()
	r.Use(middleware.Recovery(l))
	r.Use(middleware.RequestID)
	r.Use(middleware.Logging(l))

	r.Get("/healthz", handler.Health)
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/rate", rateH.Get)
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		l.Info("gateway http listening", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	l.Info("stopped")
	return nil
}
