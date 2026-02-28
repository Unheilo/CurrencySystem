package currency

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"my-currency-service/currency/internal/config"
	"my-currency-service/currency/internal/dto"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Currency struct {
	baseURL    *url.URL
	httpClient *http.Client
	logger     *slog.Logger
}

func New(cfg config.APIConfig, logger *slog.Logger) (Currency, error) {
	baseURL, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return Currency{}, fmt.Errorf("invalid base URL: %w", err)
	}

	return Currency{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // TODO: сделать конфигурируемым, такое значение не для прода
			},
		},
		logger: logger,
	}, nil
}

func EurounionRequestMessage(ReqData *dto.ExchangeRateRequestDTO) (string, error) {

	if ReqData.BasicCurrency == "" && ReqData.ExchangeCurrency == "" && ReqData.StartPeriod == "" && ReqData.EndPeriod == "" {
		return "", fmt.Errorf("Found zero value in CurrencyEurounionRequest")
	}

	sentence := fmt.Sprintf(dto.DefaultEurounionExchangeAdress,
		ReqData.BasicCurrency, ReqData.ExchangeCurrency, ReqData.StartPeriod, ReqData.EndPeriod)

	return sentence, nil

}

func RequestMessage(ReqData *dto.CurrencyRequestDTO) (string, error) {

	if ReqData.BaseCurrency == "" || ReqData.TargetCurrency == "" ||
		ReqData.DateFrom.IsZero() || ReqData.DateTo.IsZero() {
		return "", fmt.Errorf("Found zero value in CurrencyEurounionRequest: "+
			"BaseCurrency %w, TargetCurrency %w, DateFrom %w, DateTo %w",
			ReqData.BaseCurrency, ReqData.TargetCurrency, ReqData.DateFrom, ReqData.DateTo)
	}

	sentence := fmt.Sprintf(dto.DefaultEurounionExchangeAdress,
		ReqData.BaseCurrency, ReqData.TargetCurrency,
		ReqData.DateFrom.Format("2006-01-02"), ReqData.DateTo.Format("2006-01-02"))

	return sentence, nil

}

func (c *Currency) FetchCurrentRates(ctx context.Context, ReqData *dto.CurrencyRequestDTO) (map[string]float64, error) {

	messageUrl, err := RequestMessage(ReqData)

	if err != nil {
		return nil, err
	}

	c.logger.DebugContext(ctx, "sending request", slog.String("url", messageUrl))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, messageUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// TODO: Вынести в конфиг формат xml
	req.Header.Add("Accept", "application/vnd.sdmx.structurespecificdata+xml;version=2.1")

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return nil, fmt.Errorf("Error while execute request: %v\n", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Server returned error: %s\n", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error while reading body: %v\n", err)
	}

	points, err := extractObs(bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("Error while parsing XML: %v\n", err)
	}

	//for _, p := range points {
	//	fmt.Println(p.Date, p.Value)
	//}

	// TODO: add metrics for this method

	rates := make(map[string]float64, len(points))
	for _, p := range points {
		rates[p.Date.Format("2006-01-02")] = float64(p.Value)
	}

	return rates, nil

}

func MakeCurrencyRequest(ReqData *dto.ExchangeRateRequestDTO) (dto.CurrencyResponseDTO, error) {

	url, err := EurounionRequestMessage(ReqData)

	fmt.Println("URL:")
	fmt.Println(url)

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
		return dto.CurrencyResponseDTO{}, err
	}

	// 3. Устанавливаем заголовок Accept (Content Negotiation)
	req.Header.Add("Accept", "application/vnd.sdmx.structurespecificdata+xml;version=2.1")

	// 4. Выполняем запрос
	resp, err := client.Do(req)
	if err != nil {

		return dto.CurrencyResponseDTO{}, fmt.Errorf("Error while execute request: %v\n", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
		}
	}(resp.Body)

	// 5. Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		return dto.CurrencyResponseDTO{}, fmt.Errorf("Server returned error: %s\n", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return dto.CurrencyResponseDTO{}, fmt.Errorf("Error while reading body: %v\n", err)
	}

	points, err := extractObs(bytes.NewReader(body))
	if err != nil {
		return dto.CurrencyResponseDTO{}, fmt.Errorf("Error while parsing XML: %v\n", err)
	}

	for _, p := range points {
		fmt.Println(p.Date, p.Value)
	}

	return dto.CurrencyResponseDTO{Currency: ReqData.BasicCurrency, Rates: points}, nil

}

func extractObs(body io.Reader) ([]dto.RateRecordDTO, error) {
	var data StructureSpecificData
	decoder := xml.NewDecoder(body)
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode XML: %w", err)
	}

	RateRecords := make([]dto.RateRecordDTO, 0, len(data.DataSet.Series.Obs))
	for _, obs := range data.DataSet.Series.Obs {
		if obs.TimePeriod == "" || obs.ObsValue == "" {
			continue
		}

		date, err := time.Parse("2006-01-02", obs.TimePeriod)
		if err != nil {
			return nil, fmt.Errorf("failed to parse date %q: %w", obs.TimePeriod, err)
		}

		val, err := strconv.ParseFloat(obs.ObsValue, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse value %q: %w", obs.ObsValue, err)
		}

		RateRecords = append(RateRecords, dto.RateRecordDTO{
			Date:  date,
			Value: float32(val),
		})

	}

	return RateRecords, nil
}
