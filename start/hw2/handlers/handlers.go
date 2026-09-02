package handlers

import (
	"cryptoserver/models"
	"cryptoserver/storage"
	"cryptoserver/updater"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func writeJSONError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var creds models.User
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		writeJSONError(w, "Неверный формат JSON", http.StatusBadRequest)
		return
	}

	creds.Username = strings.TrimSpace(creds.Username)
	creds.Password = strings.TrimSpace(creds.Password)
	if creds.Username == "" || creds.Password == "" {
		writeJSONError(w, "Имя пользователя и пароль не могут быть пустыми", http.StatusBadRequest)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), 14)
	if err != nil {
		writeJSONError(w, "Ошибка шифрования", http.StatusInternalServerError)
		return
	}

	if ok := storage.SaveUser(creds.Username, string(hashedPassword)); !ok {
		writeJSONError(w, "Пользователь с таким логином уже существует", http.StatusConflict)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": creds.Username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, _ := token.SignedString(models.JwtKey)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Пользователь успешно зарегистрирован",
		"token":   tokenString,
	})
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var creds models.User
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		writeJSONError(w, "Неверный формат JSON", http.StatusBadRequest)
		return
	}

	creds.Username = strings.TrimSpace(creds.Username)
	creds.Password = strings.TrimSpace(creds.Password)
	if creds.Username == "" || creds.Password == "" {
		writeJSONError(w, "Логин и пароль обязательны", http.StatusBadRequest)
		return
	}

	hashedPassword, exists := storage.GetUserPasswordHash(creds.Username)
	if !exists {
		writeJSONError(w, "Неверный логин или пароль", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(creds.Password)); err != nil {
		writeJSONError(w, "Неверный логин или пароль", http.StatusUnauthorized)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": creds.Username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	})
	tokenString, err := token.SignedString(models.JwtKey)
	if err != nil {
		writeJSONError(w, "Ошибка генерации токена", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": tokenString,
	})
}

func Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Symbol string `json:"symbol"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, "Неверный формат JSON", http.StatusBadRequest)
		return
	}

	symbol := strings.ToUpper(strings.TrimSpace(req.Symbol))
	if symbol == "" {
		writeJSONError(w, "Поле symbol обязательно", http.StatusBadRequest)
		return
	}

	coinInfo, exists := updater.FindCoin(symbol)
	if !exists {
		writeJSONError(w, "Неизвестная криптовалюта", http.StatusBadRequest)
		return
	}

	newCoin := models.Crypto{
		Symbol: symbol,
		Name:   coinInfo.Name, 
	}

	if ok := storage.Set(newCoin); !ok {
		writeJSONError(w, "Криптовалюта уже существует", http.StatusConflict)
		return
	}

	_ = updater.RefreshPrice(symbol)
	savedCoin, _ := storage.Get(symbol)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]models.Crypto{
		"crypto": savedCoin,
	})
}

func Delete(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(r.PathValue("symbol"))
	if ok := storage.Delete(symbol); !ok {
		writeJSONError(w, "Криптовалюта не найдена", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Криптовалюта успешно удалена"})
}

func GetStats(w http.ResponseWriter, r *http.Request) {
	val, ok := storage.Get(strings.ToUpper(r.PathValue("symbol")))
	if !ok {
		writeJSONError(w, "Криптовалюта не найдена", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if len(val.History) == 0 {
		json.NewEncoder(w).Encode(map[string]any{
			"symbol":        val.Symbol,
			"current_price": val.CurrentPrice,
			"stats":         "Недостаточно данных для статистики",
		})
		return
	}

	minPrice, maxPrice := val.History[0].Price, val.History[0].Price
	var sum float64

	for _, rec := range val.History {
		if rec.Price < minPrice {
			minPrice = rec.Price
		}
		if rec.Price > maxPrice {
			maxPrice = rec.Price
		}
		sum += rec.Price
	}

	firstPrice := val.History[0].Price
	lastPrice := val.History[len(val.History)-1].Price
	priceChange := lastPrice - firstPrice

	var changePercent float64
	if firstPrice != 0 {
		changePercent = (priceChange / firstPrice) * 100
	}

	changePercent = math.Round(changePercent*100) / 100

	json.NewEncoder(w).Encode(map[string]any{
		"symbol":        val.Symbol,
		"current_price": val.CurrentPrice,
		"stats": map[string]any{
			"min_price":            minPrice,
			"max_price":            maxPrice,
			"avg_price":            sum / float64(len(val.History)),
			"price_change":         priceChange,
			"price_change_percent": changePercent,
			"records_count":        len(val.History),
		},
	})
}

func GetList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string][]models.Crypto{"cryptos": storage.GetAll()})
}

func GetBySymbol(w http.ResponseWriter, r *http.Request) {
	val, ok := storage.Get(strings.ToUpper(r.PathValue("symbol")))
	if !ok {
		writeJSONError(w, "Криптовалюта не найдена", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(val)
}

func GetHistory(w http.ResponseWriter, r *http.Request) {
	val, ok := storage.Get(strings.ToUpper(r.PathValue("symbol")))
	if !ok {
		writeJSONError(w, "Криптовалюта не найдена", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"symbol":  val.Symbol,
		"history": val.History,
	})
}

func Refresh(w http.ResponseWriter, r *http.Request) {
	symbol := strings.ToUpper(strings.TrimSpace(r.PathValue("symbol")))
	if symbol == "" {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) >= 2 {
			symbol = strings.ToUpper(parts[1])
		}
	}

	if symbol == "" {
		writeJSONError(w, "Символ криптовалюты не указан", http.StatusBadRequest)
		return
	}

	_, exists := storage.Get(symbol)
	if !exists {
		writeJSONError(w, "Криптовалюта не найдена", http.StatusNotFound)
		return
	}

	if err := updater.RefreshPrice(symbol); err != nil {
		writeJSONError(w, fmt.Sprintf("Ошибка обновления цены: %v", err), http.StatusBadRequest)
		return
	}

	updatedVal, _ := storage.Get(symbol)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message": "Цена успешно обновлена",
		"crypto":  updatedVal,
	})
}

func GetSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updater.GetSchedule())
}

func UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	var settings models.ScheduleSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		writeJSONError(w, "Неверный JSON", http.StatusBadRequest)
		return
	}
	if err := updater.UpdateSchedule(settings); err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func TriggerSchedule(w http.ResponseWriter, r *http.Request) {
	count := updater.TriggerUpdate()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"updated_count": count,
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	})
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeJSONError(w, "Токен не предоставлен", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			writeJSONError(w, "Неверный формат токена", http.StatusUnauthorized)
			return
		}

		token, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("неожиданный метод подписи")
			}
			return models.JwtKey, nil
		})

		if err != nil || !token.Valid {
			writeJSONError(w, "Неверный или просроченный токен", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}
