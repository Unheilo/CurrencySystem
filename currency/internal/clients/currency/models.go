package currency

import "encoding/xml"

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
