package storage

import (
	"cryptoserver/models"
	"sync"
)

var (
	data = map[string]models.Crypto{
		// "BTC":  {Symbol: "BTC", Name: "Bitcoin"},
		// "ETH":  {Symbol: "ETH", Name: "Ethereum"},
		// "USDT": {Symbol: "USDT", Name: "Tether"},
	}
	mu sync.RWMutex
)

func GetAll() []models.Crypto {
	mu.RLock()
	defer mu.RUnlock()
	list := make([]models.Crypto, 0, len(data))
	for _, coin := range data {
		list = append(list, coin)
	}
	return list
}

func Get(symbol string) (models.Crypto, bool) {
	mu.RLock()
	defer mu.RUnlock()
	val, ok := data[symbol]
	return val, ok
}

func Set(coin models.Crypto) bool {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := data[coin.Symbol]; ok {
		return false
	}
	data[coin.Symbol] = coin
	return true
}

func Delete(symbol string) bool {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := data[symbol]; !ok {
		return false
	}
	delete(data, symbol)
	return true
}

func UpdatePrices(symbol string, price float64, timestamp int64) {
	mu.Lock()
	defer mu.Unlock()
	if coin, ok := data[symbol]; ok {
		coin.CurrentPrice = price
		coin.LastUpdated = timestamp

		coin.History = append(coin.History, models.PriceRecord{
			Price: price,
			Timestamp: timestamp,
		})
		if len(coin.History) > 100 {
			coin.History = coin.History[1:]
		}
		data[symbol] = coin
	}
}
