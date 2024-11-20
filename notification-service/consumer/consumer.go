package consumer

import (
	"context"
	"encoding/json"
	"github.com/segmentio/kafka-go"
	"log"
	"notification-service/websocket"
	"time"
)

type Message struct {
	Receiver string `json:"user_id"`
	Content  string `json:"message"`
}

func StartWebsocketConsumer(brokers []string, topic, groupID string, hub *websocket.Hub) {
	readerConfig := kafka.ReaderConfig{
		Brokers:        brokers,
		GroupID:        groupID,
		Topic:          topic,
		MinBytes:       10e3,        // 10KB
		MaxBytes:       10e6,        // 10MB
		CommitInterval: time.Second, // Commit offsets every second
	}

	reader := kafka.NewReader(readerConfig)
	defer reader.Close()

	// Continuously read messages from Kafka
	for {
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Error reading message: %v", err)
			continue
		}

		// Deserialize the Kafka message
		var kafkaMessage Message
		err = json.Unmarshal(msg.Value, &kafkaMessage)
		if err != nil {
			log.Printf("Failed to deserialize Kafka message: %v", err)
			continue
		}

		// Send message to WebSocket hub
		hub.AddMessage(websocket.KafkaMessage{Receiver: kafkaMessage.Receiver, Content: kafkaMessage.Content})
	}
}
