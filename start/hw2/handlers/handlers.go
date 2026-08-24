package handlers

import (
	"cryptoserver/models"
	"cryptoserver/storage"
	"net/http"
	"encoding/json"
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

	storage.Set(newCoin)

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