package updater

import (
	"cryptoserver/models"
	"cryptoserver/storage"
	"encoding/json"
	"fmt"
	"sync"
	"net/http"
	"strings"
	"time"
)

var (
	Schedule = models.ScheduleSettings{Enabled: true, IntervalSeconds: 30}
	schedMu  sync.RWMutex
)

func GetSchedule() models.ScheduleSettings {
	schedMu.RLock()
	defer schedMu.RUnlock()
	return Schedule
}

func UpdateSchedule(newSettings models.ScheduleSettings) error {
	if newSettings.IntervalSeconds < 10 { // Тест требует 400 ошибку для интервала < 10
		return fmt.Errorf("слишком маленький интервал")
	}
	schedMu.Lock()
	defer schedMu.Unlock()
	Schedule = newSettings
	return nil
}

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

func fetchPrices() int {
	var ids []string
	updatedCount := 0

	for _, coin := range storage.GetAll() {

		if geckoID, exists := symbolToGeckoID[coin.Symbol]; exists {
			ids = append(ids, geckoID)
		}
	}
	if len(ids) == 0 {
		return 0
	}

	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd", strings.Join(ids, ","))

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Ошибка сети:", err)
		return 0
	}
	defer resp.Body.Close()

	priceList := make(map[string]map[string]float64)
	err = json.NewDecoder(resp.Body).Decode(&priceList)
	if err != nil {
		fmt.Println("Ошибка чтение JSON:", err)
		return 0
	}

	for _, coin := range storage.GetAll() {
		geckoID := symbolToGeckoID[coin.Symbol]
		if priceData, ok := priceList[geckoID]; ok {
			storage.UpdatePrices(coin.Symbol, priceData["usd"], time.Now().Unix())
			updatedCount++
		}
	}
	return updatedCount
}

func RefreshPrice(symbol string) error {
	geckoID, ok := symbolToGeckoID[symbol]
	if !ok {
		return fmt.Errorf("монета не найдена")
	}

	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd", geckoID)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	priceList := make(map[string]map[string]float64)
	if err := json.NewDecoder(resp.Body).Decode(&priceList); err != nil {
		return err
	}

	if priceData, ok := priceList[geckoID]; ok {
		storage.UpdatePrices(symbol, priceData["usd"], time.Now().Unix())
		return nil
	}
	return fmt.Errorf("цена не получена")
}

func TriggerUpdate() int {
	return fetchPrices()
}

func Start() {
	for {
		schedMu.RLock()
		enabled := Schedule.Enabled
		interval := time.Duration(Schedule.IntervalSeconds) * time.Second
		schedMu.RUnlock()

		if enabled {
			fetchPrices()
		}
		time.Sleep(interval)
	}
}
