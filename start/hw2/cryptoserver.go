package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	cryptoStorage = map[string]Crypto{
		"BTC":  {Symbol: "BTC", Name: "Bitcoin"},
		"ETH":  {Symbol: "ETH", Name: "Ethereum"},
		"USDT": {Symbol: "USDT", Name: "Tether"},
	}
	storageMutex sync.RWMutex
)
var symbolToGeckoID = make(map[string]string)

type GeckoCoin struct {
	ID     string `json:"id"`
	Symbol string `json:"symbol"`
}

type Crypto struct {
	Symbol string  `json:"symbol"`
	Name   string  `json:"name"`
	Price  float64 `json:"price"`
}

func getCryptoListHandler(w http.ResponseWriter, r *http.Request) {
	storageMutex.RLock()
	cryptoList := make([]Crypto, 0, len(cryptoStorage))
	for _, coin := range cryptoStorage {
		cryptoList = append(cryptoList, coin)
	}
	storageMutex.RUnlock()

	response := map[string][]Crypto{
		"cryptos": cryptoList,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func createCryptoHandler(w http.ResponseWriter, r *http.Request) {
	var newCoin Crypto

	err := json.NewDecoder(r.Body).Decode(&newCoin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	storageMutex.Lock()
	cryptoStorage[newCoin.Symbol] = newCoin
	storageMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := map[string]Crypto{
		"crypto": newCoin,
	}
	json.NewEncoder(w).Encode(response)
}

func getCryptoBySymbolHandler(w http.ResponseWriter, r *http.Request) {
	coinName := r.PathValue("symbol")

	storageMutex.RLock()
	defer storageMutex.RUnlock()

	val, ok := cryptoStorage[coinName]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(val)

}

func deleteCryptoHandler(w http.ResponseWriter, r *http.Request) {
	coinName := r.PathValue("symbol")

	storageMutex.Lock()
	defer storageMutex.Unlock()

	_, ok := cryptoStorage[coinName]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	delete(cryptoStorage, coinName)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{})
}

func fetchPrices() {
	var ids []string

	storageMutex.RLock()
	for symbol := range cryptoStorage {
		geckoID, exists := symbolToGeckoID[symbol]
		if exists {
			ids = append(ids, geckoID)
		}
	}
	storageMutex.RUnlock()

	if len(ids) == 0 {
		return
	}

	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd", strings.Join(ids, ","))

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Ошибка сети:", err)
		return
	}

	defer resp.Body.Close()

	fmt.Println("Статус CoinGecko:", resp.StatusCode)

	priceList := make(map[string]map[string]float64)
	err = json.NewDecoder(resp.Body).Decode(&priceList)
	if err != nil {
		fmt.Println("Ошибка чтение JSON:", err)
	}

	storageMutex.Lock()
	for symbol, coin := range cryptoStorage {
		geckoID := symbolToGeckoID[symbol]

		if priceData, ok := priceList[geckoID]; ok {
			coin.Price = priceData["usd"]
			cryptoStorage[symbol] = coin
		} else {
			fmt.Println("Ошибка получения цены для", symbol)
		}
	}
	storageMutex.Unlock()
}

func updatePrices() {
	for {
		fetchPrices()
		time.Sleep(30 * time.Second)
	}
}

func initCoinDictionary() {
	resp, err := http.Get("https://api.coingecko.com/api/v3/coins/list")
	if err != nil {
		fmt.Println("Ошибка: не удалось скачать справочник", err)
		return
	}

	defer resp.Body.Close()

	var coins []GeckoCoin
	err = json.NewDecoder(resp.Body).Decode(&coins)
	if err != nil {
		fmt.Println("Ошибка чтения JSON:", err)
		return
	}

	for _, coin := range coins {
		upperSumbol := strings.ToUpper(coin.Symbol)
		symbolToGeckoID[upperSumbol] = coin.ID
	}
	fmt.Printf("В словарь загружено %d монет. \n", len(symbolToGeckoID))
}

func main() {
	initCoinDictionary()
	go updatePrices()
	http.HandleFunc("GET /crypto", getCryptoListHandler)
	http.HandleFunc("GET /crypto/{symbol}", getCryptoBySymbolHandler)
	http.HandleFunc("POST /crypto", createCryptoHandler)
	http.HandleFunc("DELETE /crypto/{symbol}", deleteCryptoHandler)

	http.ListenAndServe(":8080", nil)

}
