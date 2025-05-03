package models

type TickerPrice struct {
	Symbol    string `json:"symbol"`
	Price     string `json:"price"`
	Timestamp string `json:"timestamp"`
}
