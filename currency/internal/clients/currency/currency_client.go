package currency

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"my-currency-service/currency/internal/dto"
	"net/http"
	"strconv"
	"time"
)

func EurounionRequestMessage(ReqData *dto.ExchangeRateRequestDTO) (string, error) {

	if ReqData.BasicCurrency == "" && ReqData.ExchangeCurrency == "" && ReqData.StartPeriod == "" && ReqData.EndPeriod == "" {
		return "", fmt.Errorf("Found zero value in CurrencyEurounionRequest")
	}

	sentence := fmt.Sprintf(dto.DefaultEurounionExchangeAdress,
		ReqData.BasicCurrency, ReqData.ExchangeCurrency, ReqData.StartPeriod, ReqData.EndPeriod)

	return sentence, nil

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
		fmt.Printf("Error while execute request: %v\n", err)
		return dto.CurrencyResponseDTO{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
		}
	}(resp.Body)

	// 5. Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Server returned error: %s\n", resp.Status)
		return dto.CurrencyResponseDTO{}, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error while reading body: %v\n", err)
		return dto.CurrencyResponseDTO{}, err
	}

	points, err := extractObs(bytes.NewReader(body))
	if err != nil {
		fmt.Printf("Error while parsing XML: %v\n", err)
		return dto.CurrencyResponseDTO{}, err
	}

	for _, p := range points {
		fmt.Println(p.Date, p.Value)
	}

	return dto.CurrencyResponseDTO{Currency: dto.DefaultBaseExchangeCurrency, Rates: points}, nil

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
