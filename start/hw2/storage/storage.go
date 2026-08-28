package storage

import (
	"cryptoserver/models"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var db *sql.DB

func InitDB(dsn string) error {
	var err error
	db, err = sql.Open("pgx", dsn)
	if err != nil{
		return fmt.Errorf("Ошибка открытия бд: %w", err)
	}

	if err = db.Ping(); err != nil {
		return fmt.Errorf("Бд недоступна: %w", err)
	}

	query := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(50) UNIQUE NOT NULL,
		password_hash TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS cryptos (
		symbol VARCHAR(10) PRIMARY KEY,
		name TEXT NOT NULL,
		current_price DOUBLE PRECISION DEFAULT 0,
		last_updated BIGINT DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS price_history (
		id SERIAL PRIMARY KEY,
		symbol VARCHAR(10) REFERENCES cryptos(symbol) ON DELETE CASCADE,
		price DOUBLE PRECISION NOT NULL,
		timestamp BIGINT NOT NULL
	);
	`
	_, err = db.Exec(query)
	if err != nil {
		return fmt.Errorf("Ошибка создания таблиц: %w", err)
	}
	fmt.Println("Бд подключена, таблицы готовы к работе")
	return nil
}


var (
	data = map[string]models.Crypto{
		// "BTC":  {Symbol: "BTC", Name: "Bitcoin"},
		// "ETH":  {Symbol: "ETH", Name: "Ethereum"},
		// "USDT": {Symbol: "USDT", Name: "Tether"},
	}
	mu sync.RWMutex
)

func GetAll() []models.Crypto {
	rows, err := db.Query("SELECT symbol, name, current_price, last_updated FROM cryptos")
	if err != nil {
		return []models.Crypto{}
	}
	defer rows.Close()

	var list []models.Crypto
	for rows.Next() {
		var coin models.Crypto
		if err := rows.Scan(&coin.Symbol, &coin.Name, &coin.CurrentPrice, &coin.LastUpdated); err == nil {
			list = append(list, coin)
		}
	}
	return list
}

func Get(symbol string) (models.Crypto, bool) {
	var coin models.Crypto
	err := db.QueryRow("SELECT symbol, name, current_price, last_updated FROM cryptos WHERE symbol = $1", symbol).
		Scan(&coin.Symbol, &coin.Name, &coin.CurrentPrice, &coin.LastUpdated)
	if err != nil {
		return coin, false 
	}

	rows, err := db.Query("SELECT price, timestamp FROM price_history WHERE symbol = $1 ORDER BY timestamp ASC", symbol)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var rec models.PriceRecord
			if err := rows.Scan(&rec.Price, &rec.Timestamp); err == nil {
				coin.History = append(coin.History, rec)
			}
		}
	}

	return coin, true
}

func Set(coin models.Crypto) bool {
	res, err := db.Exec("INSERT INTO cryptos (symbol, name, current_price, last_updated) VALUES ($1, $2, $3, $4) ON CONFLICT (symbol) DO NOTHING",
		coin.Symbol, coin.Name, coin.CurrentPrice, coin.LastUpdated)
	if err != nil {
		return false
	}
	rowsAffected, _ := res.RowsAffected()
	return rowsAffected > 0 // Если 0 — запись уже существовала (сработал ON CONFLICT)
}

func Delete(symbol string) bool {
	res, err := db.Exec("DELETE FROM cryptos WHERE symbol = $1", symbol)
	if err != nil {
		return false
	}
	rowsAffected, _ := res.RowsAffected()
	return rowsAffected > 0
}

func UpdatePrices(symbol string, price float64, timestamp int64) {
	db.Exec("UPDATE cryptos SET current_price = $1, last_updated = $2 WHERE symbol = $3", price, timestamp, symbol)

	db.Exec("INSERT INTO price_history (symbol, price, timestamp) VALUES ($1, $2, $3)", symbol, price, timestamp)
}

func SaveUser(username, passwordHash string) bool {
	res, err := db.Exec("INSERT INTO users (username, password_hash) VALUES ($1, $2) ON CONFLICT (username) DO NOTHING", username, passwordHash)
	if err != nil {
		return false
	}
	rowsAffected, _ := res.RowsAffected()
	return rowsAffected > 0
}

func GetUserPasswordHash(username string) (string, bool) {
	var hash string
	err := db.QueryRow("SELECT password_hash FROM users WHERE username = $1", username).Scan(&hash)
	if err != nil {
		return "", false
	}
	return hash, true
}