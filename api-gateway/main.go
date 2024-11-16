package main

import (
	"api-gateway/config"
	"api-gateway/handlers"
	"api-gateway/middleware"
	"encoding/json"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"net/http"
	"os"
)

func main() {
	cfg := config.LoadConfig()
	// Initialize handlers
	userHandler := handlers.NewUserHandler(cfg)

	router := mux.NewRouter()
	// Unprotected routes
	router.HandleFunc("/health", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
		json.NewEncoder(writer).Encode(map[string]string{"status": "ok"})
	}).Methods("GET")
	router.HandleFunc("/login", userHandler.Login).Methods("POST")
	router.HandleFunc("/register", userHandler.Register).Methods("POST")

	// Routes
	router.HandleFunc("/refresh", userHandler.RefreshToken).Methods("POST")

	// Apply middleware
	router.Use(middleware.TokenAuthMiddleware)

	// Start the API Gateway
	http.ListenAndServe(":"+os.Getenv("PORT"), router)
}
