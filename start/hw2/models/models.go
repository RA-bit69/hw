package models

import (
	"sync"
)

type PriceRecord struct {
	Price     float64 `json:"price"`
	Timestamp string   `json:"timestamp"`
}

type Crypto struct {
	Symbol       string        `json:"symbol"`
	Name         string        `json:"name"`
	CurrentPrice float64       `json:"current_price"` 
	LastUpdated  string         `json:"last_updated"` 
	History      []PriceRecord `json:"-"`           
}

type GeckoCoin struct {
	ID     string `json:"id"`
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ScheduleSettings struct {
	Enabled         bool `json:"enabled"`
	IntervalSeconds int  `json:"interval_seconds"`
	LastUpdate      string `json:"last_update,omitempty"`
	NextUpdate      string `json:"next_update,omitempty"`
}

var (
	// ключ - логин, значение - захешированный пароль
	UserStorage = make(map[string]string)
	UserMutex sync.RWMutex
)

var JwtKey = []byte("secret_key_123")