package updater

import (
	"cryptoserver/models"
	"cryptoserver/storage"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type CoinInfo struct {
	ID   string
	Name string
}

var (
	symbolDictionary = make(map[string]CoinInfo)
	dictMu           sync.RWMutex

	Schedule = models.ScheduleSettings{Enabled: true, IntervalSeconds: 30}
	lastRun  time.Time
	schedMu  sync.RWMutex
)

func InitCoinDictionary() {
	resp, err := http.Get("https://api.coingecko.com/api/v3/coins/list")
	if err != nil {
		fmt.Println("Ошибка: не удалось скачать справочник", err)
		return
	}
	defer resp.Body.Close()

	var coins []models.GeckoCoin
	if err = json.NewDecoder(resp.Body).Decode(&coins); err != nil {
		fmt.Println("Ошибка чтения JSON:", err)
		return
	}

	dictMu.Lock()
	for _, coin := range coins {
		upper := strings.ToUpper(coin.Symbol)
		symbolDictionary[upper] = CoinInfo{
			ID:   coin.ID,
			Name: coin.Name,
		}
	}
	dictMu.Unlock()
	fmt.Printf("В словарь загружено %d монет.\n", len(symbolDictionary))
}

func FindCoin(symbol string) (CoinInfo, bool) {
	dictMu.RLock()
	defer dictMu.RUnlock()
	info, ok := symbolDictionary[strings.ToUpper(symbol)]
	return info, ok
}

func fetchPrices() int {
	var ids []string
	dictMu.RLock()
	for _, coin := range storage.GetAll() {
		if info, exists := symbolDictionary[coin.Symbol]; exists {
			ids = append(ids, info.ID)
		}
	}
	dictMu.RUnlock()

	if len(ids) == 0 {
		return 0
	}

	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd", strings.Join(ids, ","))
	resp, err := http.Get(url)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	priceList := make(map[string]map[string]float64)
	if err = json.NewDecoder(resp.Body).Decode(&priceList); err != nil {
		return 0
	}

	nowISO := time.Now().UTC().Format(time.RFC3339)
	updatedCount := 0

	dictMu.RLock()
	for _, coin := range storage.GetAll() {
		if info, ok := symbolDictionary[coin.Symbol]; ok {
			if priceData, found := priceList[info.ID]; found {
				storage.UpdatePrices(coin.Symbol, priceData["usd"], nowISO)
				updatedCount++
			}
		}
	}
	dictMu.RUnlock()

	schedMu.Lock()
	lastRun = time.Now().UTC()
	schedMu.Unlock()

	return updatedCount
}

func RefreshPrice(symbol string) error {
	info, ok := FindCoin(symbol)
	if !ok {
		return fmt.Errorf("монета не найдена в CoinGecko")
	}

	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd", info.ID)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	priceList := make(map[string]map[string]float64)
	if err = json.NewDecoder(resp.Body).Decode(&priceList); err != nil {
		return err
	}

	if priceData, found := priceList[info.ID]; found {
		nowISO := time.Now().UTC().Format(time.RFC3339)
		storage.UpdatePrices(symbol, priceData["usd"], nowISO)
		return nil
	}
	return fmt.Errorf("цена не получена")
}

func GetSchedule() models.ScheduleSettings {
	schedMu.RLock()
	defer schedMu.RUnlock()

	s := Schedule
	if !lastRun.IsZero() {
		s.LastUpdate = lastRun.Format(time.RFC3339)
		if s.Enabled {
			s.NextUpdate = lastRun.Add(time.Duration(s.IntervalSeconds) * time.Second).Format(time.RFC3339)
		}
	}
	return s
}

func UpdateSchedule(newSettings models.ScheduleSettings) error {
	if newSettings.IntervalSeconds < 10 {
		return fmt.Errorf("слишком маленький интервал")
	}
	schedMu.Lock()
	defer schedMu.Unlock()
	Schedule.Enabled = newSettings.Enabled
	Schedule.IntervalSeconds = newSettings.IntervalSeconds
	return nil
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
