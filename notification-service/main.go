package main

import (
	"net/http"
	"notification-service/config"
	"notification-service/consumer"
	"notification-service/websocket"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	config := config.LoadConfig()
	hub := websocket.NewHub()

	go hub.Broadcast()

	go consumer.StartWebsocketConsumer(
		[]string{config.KafkaServiceURL},
		"default",
		"notification-service-group",
		hub,
	)

	// register websocket registration function
	http.HandleFunc("/ws", hub.WebSocketHandler)

	go http.ListenAndServe(":"+os.Getenv("PORT"), nil)

	interruption := make(chan os.Signal, 1)
	signal.Notify(interruption, syscall.SIGINT, syscall.SIGTERM)
	<-interruption
}
