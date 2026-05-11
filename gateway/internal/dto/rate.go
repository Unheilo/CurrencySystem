package dto

type RateResponse struct {
	Currency string      `json:"currency"`
	Rates    []RatePoint `json:"rates"`
}

type RatePoint struct {
	Date string  `json:"date"`
	Rate float32 `json:"rate"`
}
