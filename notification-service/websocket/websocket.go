package websocket

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	UserID string
	Conn   *websocket.Conn
}

type Hub struct {
	clients   map[string]*Client // Maps user IDs to WebSocket connections
	mu        sync.RWMutex       // Protects the `clients` map
	broadcast chan KafkaMessage  // Channel for broadcasting Kafka messages
}

type KafkaMessage struct {
	Receiver string
	Content  string
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func NewHub() *Hub {
	return &Hub{
		clients:   make(map[string]*Client),
		broadcast: make(chan KafkaMessage),
	}
}

// WebSocketHandler establishes WebSocket connections
func (h *Hub) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		http.Error(w, "Failed to upgrade to WebSocket", http.StatusInternalServerError)
		return
	}

	// Extract user ID from query parameters
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		log.Println("Missing user_id in query parameters")
		conn.Close()
		http.Error(w, "Missing user_id query parameter", http.StatusBadRequest)
		return
	}

	// Add client to the hub
	client := &Client{UserID: userID, Conn: conn}
	h.mu.Lock()
	h.clients[userID] = client
	h.mu.Unlock()

	log.Printf("User %s connected via WebSocket", userID)

	// Listen for WebSocket close events
	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.clients, userID)
			h.mu.Unlock()
			conn.Close()
			log.Printf("User %s disconnected", userID)
		}()

		for {
			// ignoring client messages
			_, _, err := conn.ReadMessage()
			if err != nil {
				log.Printf("Error reading from WebSocket for user %s: %v", userID, err)
				break
			}
		}
	}()
}

func (h *Hub) AddMessage(message KafkaMessage) {
	h.broadcast <- message
}

// Broadcast sends Kafka messages to connected WebSocket clients
func (h *Hub) Broadcast() {
	for msg := range h.broadcast {
		h.mu.RLock()
		client, exists := h.clients[msg.Receiver]
		h.mu.RUnlock()

		if exists {
			err := client.Conn.WriteMessage(websocket.TextMessage, []byte(msg.Content))
			if err != nil {
				log.Printf("Failed to send message to user %s: %v", msg.Receiver, err)
			} else {
				log.Printf("Sent message to user %s", msg.Receiver)
			}
		} else {
			// todo send push notification (apple / android) if user uses mobile app
			log.Printf("User %s is not connected;", msg.Receiver)
		}
	}
}
