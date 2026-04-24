package main

import (
	"fmt"
	"log/slog"
	currencyClient "my-currency-service/currency/internal/clients/currency"
	"my-currency-service/currency/internal/config"
	"my-currency-service/currency/internal/handler"
	"my-currency-service/currency/internal/logger"
	"my-currency-service/currency/internal/repository"
	"my-currency-service/currency/internal/service"
	"my-currency-service/pkg/currency"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"my-currency-service/currency/internal/db"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

var (
	requestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "currency_requests_total",
			Help: "Total number of requets handled by the currency service",
		},
		[]string{"method"},
	)

	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "currency_request_duration_seconds",
			Help:    "Histogram of repsonse times for requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method"},
	)

	appUptime = prometheus.NewGauge(
		prometheus.GaugeOpts{Name: "currency_service_uptime_seconds",
			Help: "Time since service start in seconds"},
	)
)

// metrics registration
func init() {
	prometheus.MustRegister(requestCount)
	prometheus.MustRegister(requestDuration)
	prometheus.MustRegister(appUptime)
}

func main() {

	cfg := config.MustLoad()

	log, err := logger.SetupLogger(cfg.Service.Env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to setup logger: %v\n", err)
		os.Exit(1)
	}

	log.Info("Starting application",
		slog.String("config", cfg.Service.Env),
		slog.Int("grpc_port", cfg.Service.ServerPort),
	)

	conn, err := db.NewDatabaseConnection(cfg.Database)
	if err != nil {
		log.Error("db connect", slog.Any("error", err))
		os.Exit(1)
	}

	repo := repository.NewPostgresRepository(conn)
	CurrencyClient, err := currencyClient.New(cfg.API, log)
	if err != nil {
		log.Error("error while create client", slog.Any("error", err))
		os.Exit(1)
	}

	svc := service.NewCurrency(repo, CurrencyClient, log)

	//middleware

	currencyServer := handler.NewCurrencyServer(svc,
		log,
		requestCount,
		requestDuration,
		&appUptime,
		/*metrics*/) // TODO: implement metrics

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Info("Prometheus metrics server running on :8081") //TODO: сделать в конфиге порт прометея
		if err := http.ListenAndServe(":8081", nil); err != nil {
			log.Error("error starting Prometheus metrics server", slog.Any("error", err))
		}
	}()

	application := New(log, currencyServer, cfg.Service.ServerPort)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	errCh := make(chan error, 1)
	// Starting Application
	go func() { errCh <- application.Run() }()
	select {
	case sig := <-stop:
		// Graceful shutdown
		log.Info("stopping application", slog.String("signal", sig.String()))
		log.Info("signal", slog.String("signal", sig.String()))
	case err := <-errCh:
		log.Error("server failed", slog.Any("error", err))
		os.Exit(1)
	}

	application.Stop()
	log.Info("application stopped")

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
) *App {
	gRPCServer := grpc.NewServer()

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
