package currency

import (
	"bytes"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
)

func EurounionRequestMessage(ReqData *ExchangeRateRequest) (string, error) {

	if ReqData.BasicCurrency == "" && ReqData.ExchangeCurrency == "" && ReqData.StartPeriod == "" && ReqData.EndPeriod == "" {
		return "", fmt.Errorf("Found zero value in CurrencyEurounionRequest")
	}

	sentence := fmt.Sprintf("https://data-api.ecb.europa.eu/service/data/EXR/D.%s.%s.SP00.A?startPeriod=%s&endPeriod=%s",
		ReqData.BasicCurrency, ReqData.ExchangeCurrency, ReqData.StartPeriod, ReqData.EndPeriod)

	return sentence, nil

}

type ExchangeRateRequest struct {
	BasicCurrency    string
	ExchangeCurrency string
	StartPeriod      string
	EndPeriod        string
}

func MakeCurrencyRequest(ReqData *ExchangeRateRequest) ([]byte, error) {

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
		return []byte(""), err
	}

	// 3. Устанавливаем заголовок Accept (Content Negotiation)
	req.Header.Add("Accept", "application/vnd.sdmx.structurespecificdata+xml;version=2.1")

	// 4. Выполняем запрос
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error while execute request: %v\n", err)
		return []byte(""), err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
		}
	}(resp.Body)

	// 5. Проверяем статус ответа
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Server returned error: %s\n", resp.Status)
		return []byte(""), err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error while reading body: %v\n", err)
		return []byte(""), err
	}

	points, err := extractObs(bytes.NewReader(body))
	if err != nil {
		fmt.Printf("Error while parsing XML: %v\n", err)
		return body, err
	}

	for _, p := range points {
		fmt.Println(p.Date, p.Value)
	}

	return body, nil

}

type Point struct {
	Date  string
	Value string
}

// XML-структуры для SDMX StructureSpecificData формата ECB
type StructureSpecificData struct {
	XMLName xml.Name                 `xml:"StructureSpecificData"`
	DataSet StructureSpecificDataSet `xml:"DataSet"`
}

type StructureSpecificDataSet struct {
	Series StructureSpecificSeries `xml:"Series"`
}

type StructureSpecificSeries struct {
	Obs []StructureSpecificObs `xml:"Obs"`
}

type StructureSpecificObs struct {
	TimePeriod string `xml:"TIME_PERIOD,attr"`
	ObsValue   string `xml:"OBS_VALUE,attr"`
}

func extractObs(body io.Reader) ([]Point, error) {
	var data StructureSpecificData
	decoder := xml.NewDecoder(body)
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode XML: %w", err)
	}

	points := make([]Point, 0, len(data.DataSet.Series.Obs))
	for _, obs := range data.DataSet.Series.Obs {
		if obs.TimePeriod == "" || obs.ObsValue == "" {
			continue
		}
		points = append(points, Point{
			Date:  obs.TimePeriod,
			Value: obs.ObsValue,
		})
	}

	return points, nil
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
