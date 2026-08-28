package models

import (
	"sync"
)

type PriceRecord struct {
	Price     float64 `json:"price"`
	Timestamp int64   `json:"timestamp"`
}

type Crypto struct {
	Symbol       string        `json:"symbol"`
	Name         string        `json:"name"`
	CurrentPrice float64       `json:"current_price"` 
	LastUpdated  int64         `json:"last_updated"` 
	History      []PriceRecord `json:"-"`           
}

type GeckoCoin struct {
	ID     string `json:"id"`
	Symbol string `json:"symbol"`
}

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ScheduleSettings struct {
	Enabled         bool `json:"enabled"`
	IntervalSeconds int  `json:"interval_seconds"`
}

var (
	// ключ - логин, значение - захешированный пароль
	UserStorage = make(map[string]string)
	UserMutex sync.RWMutex
)

var JwtKey = []byte("secret_key_123")