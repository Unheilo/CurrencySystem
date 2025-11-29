package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"golang.org/x/net/html/charset"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"io"
	"log"
	"log/slog"
	"my-currency-service/currency/internal/config"
	"net"
	"net/http"
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

	testquery()

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

func testquery() {
	response, err := http.Get("https://www.cbr.ru/scripts/XML_daily_eng.asp?date_req=22/01/2007")

	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	ParseCurrencyXMLtoGolangStructure(string(body))

}

type RawCurrency struct {
	XMLName xml.Name `xml:"ValCurs"`
	Date    string   `xml:"Date"`
	Name    string   `xml:"Name"`

	Valute []struct {
		ID        string `xml:"ID,attr"`
		NumCode   int    `xml:"NumCode"`
		CharCode  string `xml:"CharCode"`
		Nominal   int    `xml:"Nominal"`
		Name      string `xml:"Name"`
		Value     string `xml:"Value"`
		VunitRate string `xml:"VunitRate"`
	} `xml:"Valute"`
}

func ParseCurrencyXMLtoGolangStructure(data string) {
	//result := new(RawCurrency)
	//err := xml.Unmarshal([]byte(data), result)
	//if err != nil {
	//	fmt.Printf("error: %v", err)
	//	return
	//}
	//
	//fmt.Printf("--- Unmarshal ---\n\n")
	//for _, CurrencyNode := range result.Valute {
	//	fmt.Printf("Name : %s\n", CurrencyNode.CharCode)
	//	fmt.Printf("Value  %s\n", CurrencyNode.Value)
	//	fmt.Printf("ValueRate %s\n", CurrencyNode.VunitRate)
	//}

	filmsDB := new(RawCurrency)
	r := bytes.NewReader([]byte(data))
	d := xml.NewDecoder(r)
	d.CharsetReader = charset.NewReaderLabel
	err := d.Decode(&filmsDB)
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}

	fmt.Printf("--- Unmarshal ---\n\n")
	for _, CurrencyNode := range filmsDB.Valute {
		fmt.Printf("Name : %s\n", CurrencyNode.CharCode)
		fmt.Printf("Value  %s\n", CurrencyNode.Value)
		fmt.Printf("ValueRate %s\n", CurrencyNode.VunitRate)
	}
}
