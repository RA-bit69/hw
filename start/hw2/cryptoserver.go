package main

import (
	"cryptoserver/updater"
	"cryptoserver/handlers"
	"net/http"
)

func main() {
	updater.InitCoinDictionary()
	go updater.Start()
	
	http.HandleFunc("GET /crypto", handlers.GetList)
	http.HandleFunc("GET /crypto/{symbol}", handlers.GetBySymbol)
	http.HandleFunc("POST /crypto", handlers.Create)
	http.HandleFunc("DELETE /crypto/{symbol}", handlers.Delete)

	http.ListenAndServe(":8080", nil)

}
