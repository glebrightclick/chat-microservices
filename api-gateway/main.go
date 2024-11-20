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
	handler := handlers.NewHandler(cfg)
	defer handler.KafkaWriter.Close()

	router := mux.NewRouter()
	// Unprotected routes
	router.HandleFunc("/health", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
		json.NewEncoder(writer).Encode(map[string]string{"status": "ok"})
	}).Methods("GET")
	router.HandleFunc("/login", handler.Login).Methods("POST")
	router.HandleFunc("/register", handler.Register).Methods("POST")

	// Routes
	router.HandleFunc("/refresh", handler.RefreshToken).Methods("POST")
	router.HandleFunc("/send-notification", handler.TestNotification).Methods("POST")
	router.HandleFunc("/ws", handler.ProxyWebSocket)

	// Apply middleware
	router.Use(middleware.TokenAuthMiddleware)

	// Start the API Gateway
	http.ListenAndServe(":"+os.Getenv("PORT"), router)
}
