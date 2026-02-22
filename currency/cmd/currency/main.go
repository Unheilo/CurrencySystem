package main

import (
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"log/slog"
	"my-currency-service/currency/internal/clients/currency"
	"my-currency-service/currency/internal/config"
	"net"
	"os"
	"os/signal"
	"syscall"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {

	cfg := config.MustLoad()

	fmt.Println(cfg)

	log := setupLogger(cfg.Service.Env)

	log.Info("Starting application",
		slog.Any("config", cfg),
	)

	log.Debug("debug message")

	log.Error("error message")

	log.Warn("warn message")

	fmt.Println(cfg.Service.ServerPort)
	// TODO: инициализировать приложение (app)
	application := New(log, 8303)

	// TODO: запустить gRPC-сервер приложения
	go application.MustRun()

	CurrencyRequestParameters := currency.ExchangeRateRequest{
		BasicCurrency:    "USD",
		ExchangeCurrency: "EUR",
		StartPeriod:      "2024-05-01",
		EndPeriod:        "2024-05-31",
	}

	Message, err := currency.MakeCurrencyRequest(&CurrencyRequestParameters)

	if err != nil {
		fmt.Printf("Error while execute request: %v\n", err)
	}
	if err != nil {
		fmt.Printf(string(Message))
	}

	// TODO: Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	sign := <-stop

	log.Info("stopping application", slog.String("signal", sign.String()))

	application.Stop()

	log.Info("application stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

type App struct {
	log        *slog.Logger
	gRPCServer *grpc.Server
	port       int
}

// New creates new gRPC server app.
func New(
	log *slog.Logger,
	//authService authgrpc.Auth,
	port int,
) *App {
	gRPCServer := grpc.NewServer()

	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(gRPCServer, healthServer)

	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	reflection.Register(gRPCServer)

	//authgrpc.Register(gRPCServer, authService)

	return &App{
		log:        log,
		gRPCServer: gRPCServer,
		port:       port,
	}
}

func (a *App) MustRun() {
	if err := a.Run(); err != nil {
		panic(err)
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
