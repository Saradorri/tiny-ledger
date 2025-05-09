package main

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"tiny-ledger/internal/handlers"
	"tiny-ledger/internal/services"
	"tiny-ledger/internal/store"
)

func main() {
	ledgerStore := store.NewLedgerStore()
	ledgerService := services.NewLedgerService(ledgerStore)
	ledgerHandler := handlers.NewLedgerHandler(ledgerService)

	r := mux.NewRouter()
	ledgerHandler.RegisterRoutes(r)

	log.Println("Server is running on port 8080")
	err := http.ListenAndServe(":8080", r)
	if err != nil {
		return
	}
}
