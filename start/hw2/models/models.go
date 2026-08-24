package models

type Crypto struct {
	Symbol string  `json:"symbol"`
	Name   string  `json:"name"`
	Price  float64 `json:"price"`
	History []float64 `json:"history"`
}

type GeckoCoin struct {
	ID     string `json:"id"`
	Symbol string `json:"symbol"`
}

