package handlers

import (
	"cryptoserver/models"
	"cryptoserver/storage"
	"cryptoserver/updater"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func GetList(w http.ResponseWriter, r *http.Request) {
	response := map[string][]models.Crypto{
		"cryptos": storage.GetAll(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func Create(w http.ResponseWriter, r *http.Request) {
	var newCoin models.Crypto

	err := json.NewDecoder(r.Body).Decode(&newCoin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if ok := storage.Set(newCoin); !ok {
		http.Error(w, "Криптовалюта уже существует", http.StatusConflict)
		return 
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := map[string]models.Crypto{
		"crypto": newCoin,
	}
	json.NewEncoder(w).Encode(response)
}

func GetBySymbol(w http.ResponseWriter, r *http.Request) {
	coinName := r.PathValue("symbol")

	val, ok := storage.Get(coinName)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(val)

}

func Delete(w http.ResponseWriter, r *http.Request) {
	if ok := storage.Delete(r.PathValue("symbol")); !ok {
		w.WriteHeader(http.StatusNotFound)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{})
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var creds models.User
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		return
	}

	models.UserMutex.RLock()
	_, exists := models.UserStorage[creds.Username]
	models.UserMutex.RUnlock()

	if exists {
		http.Error(w, "Пользователь с таким логином уже существует", http.StatusConflict)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), 14)
	if err != nil {
		http.Error(w, "Ошибка шифрования", http.StatusInternalServerError)
		return
	}

	models.UserMutex.Lock()
	models.UserStorage[creds.Username] = string(hashedPassword)
	models.UserMutex.Unlock()

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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	models.UserMutex.RLock()
	hashedPassword, exists := models.UserStorage[creds.Username]
	models.UserMutex.RUnlock()

	if !exists {
		http.Error(w, "Неверный логин или пароль", http.StatusUnauthorized)
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(creds.Password))
	if err != nil {
		http.Error(w, "Неверный логин или пароль", http.StatusUnauthorized)
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": creds.Username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString(models.JwtKey)
	if err != nil {
		http.Error(w, "Ошибка генерации токена", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token": tokenString,
	})
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == ""{
			http.Error(w, "Токен не предоставлен", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Неверный формат токена", http.StatusUnauthorized)
			return
		}
		tokenString := parts[1]

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error){
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok{
				return nil, fmt.Errorf("Неожиданный метод подписи")
			}
			return models.JwtKey, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Неверный или просроченный токен", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}

func GetHistory(w http.ResponseWriter, r *http.Request) {
	val, ok := storage.Get(r.PathValue("symbol"))
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"symbol":  val.Symbol,
		"history": val.History,
	})
}

func Refresh(w http.ResponseWriter, r *http.Request) {
	coinName := r.PathValue("symbol")
	if err := updater.RefreshPrice(coinName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	val, _ := storage.Get(coinName)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]models.Crypto{"crypto": val})
}

func GetStats(w http.ResponseWriter, r *http.Request) {
	val, ok := storage.Get(r.PathValue("symbol"))
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if len(val.History) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"symbol":        val.Symbol,
			"current_price": val.CurrentPrice,
			"stats":         "Недостаточно данных",
		})
		return
	}

	min, max := val.History[0].Price, val.History[0].Price
	var sum float64

	for _, record := range val.History {
		if record.Price < min { min = record.Price }
		if record.Price > max { max = record.Price }
		sum += record.Price
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"symbol":        val.Symbol,
		"current_price": val.CurrentPrice,
		"stats": map[string]any{
			"min_price":     min,
			"max_price":     max,
			"avg_price":     sum / float64(len(val.History)),
			"records_count": len(val.History),
		},
	})
}

func GetSchedule(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updater.GetSchedule())
}

func UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	var settings models.ScheduleSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	if err := updater.UpdateSchedule(settings); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		"timestamp":     time.Now().Unix(),
	})
}
