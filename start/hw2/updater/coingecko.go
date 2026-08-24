package updater

import (
	"cryptoserver/models"
	"cryptoserver/storage"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var symbolToGeckoID = make(map[string]string)

func InitCoinDictionary() {
	resp, err := http.Get("https://api.coingecko.com/api/v3/coins/list")
	if err != nil {
		fmt.Println("Ошибка: не удалось скачать справочник", err)
		return
	}

	defer resp.Body.Close()

	var coins []models.GeckoCoin
	err = json.NewDecoder(resp.Body).Decode(&coins)
	if err != nil {
		fmt.Println("Ошибка чтения JSON1:", err)
		return
	}

	for _, coin := range coins {
		upperSumbol := strings.ToUpper(coin.Symbol)
		symbolToGeckoID[upperSumbol] = coin.ID
	}
	fmt.Printf("В словарь загружено %d монет. \n", len(symbolToGeckoID))
}

func fetchPrices() {
	var ids []string

	for _, coin := range storage.GetAll() {

		if geckoID, exists := symbolToGeckoID[coin.Symbol]; exists {
			ids = append(ids, geckoID)
		}
	}
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

	priceList := make(map[string]map[string]float64)
	err = json.NewDecoder(resp.Body).Decode(&priceList)
	if err != nil {
		fmt.Println("Ошибка чтение JSON:", err)
		return
	}

	for _, coin := range storage.GetAll() {
		geckoID := symbolToGeckoID[coin.Symbol]

		if priceData, ok := priceList[geckoID]; ok {
			storage.UpdatePrices(coin.Symbol, priceData["usd"])
		}
	}
}

func Start() {
	for {
		fetchPrices()
		time.Sleep(30 * time.Second)
	}
}
