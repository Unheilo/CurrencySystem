package main

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"golang.org/x/net/html/charset"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"io"
	"io/ioutil"
	"log"
	"log/slog"
	"my-currency-service/currency/internal/config"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
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

	Message, err := MakeCurrencyRequest()

	if err != nil {
		fmt.Printf("Error while execute request: %v\n", err)
	}
	if err == nil {
		fmt.Printf(Message)
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

func testquery() {
	response, err := http.Get("https://www.cbr.ru/scripts/XML_daily_eng.asp?date_req=22/01/2006")

	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf(string(body))
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

func CurrencyTBank() {
	// подходящая строка /D.USD.EUR.SP00.A?startPeriod=2024-05-01&endPeriod=2024-05-31
	url := "https://sandbox-invest-public-api.tbank.ru/rest/tinkoff.public.invest.api.contract.v1.InstrumentsService/Currencies"
	method := "POST"

	payload := strings.NewReader(`{

	"instrumentStatus":
	
	"INSTRUMENT_STATUS_UNSPECIFIED",
	
	"instrumentExchange":
	
	"INSTRUMENT_EXCHANGE_UNSPECIFIED"
	
	}`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer t.aCX7w6w4SUGHALYCdzEqwBbif7cz13ZYu6Jwrboin07kjfGEC7B498J8uRa7YEwWOwCRWAiWq08Xx9ISwEoULA")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(body))
}

func CurrencyEurounionRequestMessage(BasicCurrency string, ExchangeCurrency string, StartPeriod string, EndPeriod string) (string, error) {

	if BasicCurrency == "" && ExchangeCurrency == "" && StartPeriod == "" && EndPeriod == "" {
		return "", fmt.Errorf("Found zero value in CurrencyEurounionRequest")
	}

	sentence := fmt.Sprintf("https://data-api.ecb.europa.eu/service/data/EXR/D.%s.%s.SP00.A?startPeriod=%s&endPeriod=%s",
		BasicCurrency, ExchangeCurrency, StartPeriod, EndPeriod)

	return sentence, nil

}

func MakeCurrencyRequest() (string, error) {

	BasicCurrency := "USD"
	ExchangeCurrency := "EUR"
	StartPeriod := "2024-05-01"
	EndPeriod := "2024-05-31"

	url, err := CurrencyEurounionRequestMessage(BasicCurrency, ExchangeCurrency, StartPeriod, EndPeriod)

	// 1. Настраиваем транспорт, чтобы игнорировать проверку SSL-сертификатов
	// Это аналог -k в curl или verify=False в Python
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	// 2. Создаем новый GET запрос
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("Error while making request: %v\n", err)
		return "", err
	}

	// 3. Устанавливаем заголовок Accept (Content Negotiation)
	req.Header.Add("Accept", "application/vnd.sdmx.structurespecificdata+xml;version=2.1")

	// 4. Выполняем запрос
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error while execute request: %v\n", err)
		return "", err
	}
	defer resp.Body.Close()

	// 5. Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Server returned error: %s\n", resp.Status)
		return "", err
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		fmt.Printf("Error while reading body: %v\n", err)
		return "", err
	}

	return string(body), nil

}
