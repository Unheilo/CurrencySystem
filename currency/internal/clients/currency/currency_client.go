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
	"strconv"
	"time"
)

type Currency struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

func New(cfg config.APIConfig, logger *slog.Logger) (Currency, error) {
	return Currency{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.SkipVerify},
			},
		},
		logger: logger,
	}, nil
}

func (c *Currency) buildURL(ReqData *dto.CurrencyRequestDTO) (string, error) {
	if ReqData.BaseCurrency == "" || ReqData.TargetCurrency == "" ||
		ReqData.DateFrom.IsZero() || ReqData.DateTo.IsZero() {
		return "", fmt.Errorf("found zero value in request: BaseCurrency %s, TargetCurrency %s, DateFrom %s, DateTo %s",
			ReqData.BaseCurrency, ReqData.TargetCurrency, ReqData.DateFrom, ReqData.DateTo)
	}
	return fmt.Sprintf(c.baseURL,
		ReqData.BaseCurrency, ReqData.TargetCurrency,
		ReqData.DateFrom.Format("2006-01-02"), ReqData.DateTo.Format("2006-01-02")), nil
}

func (c *Currency) FetchCurrentRates(ctx context.Context, ReqData *dto.CurrencyRequestDTO) (map[string]float64, error) {

	messageUrl, err := c.buildURL(ReqData)
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

	// TODO: add metrics for this method

	rates := make(map[string]float64, len(points))
	for _, p := range points {
		rates[p.Date.Format("2006-01-02")] = float64(p.Value)
	}

	return rates, nil

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
