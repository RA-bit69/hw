package main

import (
	"cryptoserver/handlers"
	"cryptoserver/storage"
	"cryptoserver/updater"
	"net/http"
)

func main() {
	dsn := "postgres://admin:secretpassword@localhost:5433/cryptodb?sslmode=disable"
	err := storage.InitDB(dsn)
	if err != nil {
		panic(err)
	}

	updater.InitCoinDictionary()
	go updater.Start()

	http.HandleFunc("POST /auth/register", handlers.RegisterHandler)
	http.HandleFunc("POST /auth/login", handlers.LoginHandler)

	http.HandleFunc("GET /crypto", handlers.AuthMiddleware(handlers.GetList))
	http.HandleFunc("GET /crypto/{symbol}", handlers.AuthMiddleware(handlers.GetBySymbol))
	http.HandleFunc("POST /crypto", handlers.AuthMiddleware(handlers.Create))
	http.HandleFunc("DELETE /crypto/{symbol}", handlers.AuthMiddleware(handlers.Delete))
	http.HandleFunc("GET /crypto/{symbol}/history", handlers.AuthMiddleware(handlers.GetHistory))
	http.HandleFunc("GET /crypto/{symbol}/stats", handlers.AuthMiddleware(handlers.GetStats))
	http.HandleFunc("PUT /crypto/{symbol}/refresh", handlers.AuthMiddleware(handlers.Refresh))
	http.HandleFunc("GET /schedule", handlers.AuthMiddleware(handlers.GetSchedule))
	http.HandleFunc("PUT /schedule", handlers.AuthMiddleware(handlers.UpdateSchedule))
	http.HandleFunc("POST /schedule/trigger", handlers.AuthMiddleware(handlers.TriggerSchedule))

	http.ListenAndServe(":8080", nil)

}
