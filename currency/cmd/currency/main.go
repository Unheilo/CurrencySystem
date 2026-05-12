package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	currencyClient "my-currency-service/currency/internal/clients/currency"
	"my-currency-service/currency/internal/config"
	"my-currency-service/currency/internal/handler"
	"my-currency-service/currency/internal/interceptors"
	"my-currency-service/pkg/logger"
	"my-currency-service/currency/internal/metrics"
	"my-currency-service/currency/internal/repository"
	"my-currency-service/currency/internal/service"
	"my-currency-service/pkg/currency"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"my-currency-service/currency/internal/db"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {

	cfg := config.MustLoad()

	log, err := logger.Setup(cfg.Service.Env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to setup logger: %v\n", err)
		os.Exit(1)
	}

	log.Info("Starting application",
		slog.String("config", cfg.Service.Env),
		slog.Int("grpc_port", cfg.Service.ServerPort),
		slog.Int("metrics_port", cfg.Service.MetricsPort),
	)

	conn, err := db.NewDatabaseConnection(cfg.Database)
	if err != nil {
		log.Error("db connect", slog.Any("error", err))
		os.Exit(1)
	}

	m := metrics.New()

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{}))

	metricsSrv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Service.MetricsPort),
		Handler:           metricsMux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("Prometheus metrics server running", slog.String("addr", metricsSrv.Addr))
		if err := metricsSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("metrics server", slog.Any("error", err))
		}
	}()

	repo := repository.NewPostgresRepository(conn, repository.RepositoryMetrics{
		DBQueryDuration: m.DBQueryDuration,
	})

	CurrencyClient, err := currencyClient.New(cfg.API, log, currencyClient.ECBMetrics{
		RequestDuration: m.ECBRequestDuration,
		Errors:          m.ECBErrors,
	})

	if err != nil {
		log.Error("error while create client", slog.Any("error", err))
		_ = conn.Close()
		os.Exit(1)
	}

	svc := service.NewCurrency(repo, CurrencyClient, log)

	currencyServer := handler.NewCurrencyServer(svc,
		log,
		cfg.Worker.CurrencyPair.BaseCurrency)

	application := New(log, currencyServer, cfg.Service.ServerPort, m)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	errCh := make(chan error, 1)
	go func() { errCh <- application.Run() }()

	exitCode := 0
	select {
	case sig := <-stop:
		log.Info("stopping application", slog.String("signal", sig.String()))
	case err := <-errCh:
		log.Error("server failed", slog.Any("error", err))
		exitCode = 1
	}

	application.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("metrics server shutdown", slog.Any("error", err))
	}

	if err := conn.Close(); err != nil {
		log.Error("db close", slog.Any("error", err))
	}

	log.Info("application stopped")
	os.Exit(exitCode)
}

type App struct {
	log            *slog.Logger
	currencyServer *handler.CurrencyServer
	gRPCServer     *grpc.Server
	port           int
}

// New creates new gRPC server app.
func New(
	log *slog.Logger,
	currencyServer *handler.CurrencyServer,
	//authService authgrpc.Auth,
	port int,
	m *metrics.Metrics,
) *App {
	gRPCServer := grpc.NewServer(grpc.ChainUnaryInterceptor(
		interceptors.Recovery(log),
		interceptors.RequestID(),
		interceptors.Logging(log),
		interceptors.Metrics(m.ReqCount, m.ReqDuration),
	),
	)

	currency.RegisterCurrencyServiceServer(gRPCServer, currencyServer)

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(gRPCServer, healthServer)
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	reflection.Register(gRPCServer)

	//authgrpc.Register(gRPCServer, authService) //TODO: авторизация

	return &App{
		log:            log,
		currencyServer: currencyServer,
		gRPCServer:     gRPCServer,
		port:           port,
	}
}

func (a *App) Run() error {
	const op = "grpcapp.Run"

	log := a.log.With(
		slog.String("op", op),
		slog.Int("port", a.port),
	)

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Info("gRPC server is running", slog.String("addr", l.Addr().String()))

	if err := a.gRPCServer.Serve(l); err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// Stop stops gRPC server
func (a *App) Stop() {
	const op = "grpcapp.Stop"

	a.log.With(slog.String("op", op)).
		Info("stopping gRPC server", slog.Int("port", a.port))

	a.gRPCServer.GracefulStop()
}
