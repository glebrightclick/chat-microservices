package handlers

import (
	"api-gateway/config"
	userServiceProto "api-gateway/proto/user_service"
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net/http"
	"net/url"
	_ "net/url"
)

type Handler struct {
	Config      *config.Config
	UserClient  userServiceProto.UserServiceClient
	KafkaWriter *kafka.Writer
}

func (h *Handler) init() {
	var connection *grpc.ClientConn
	var err error

	// init user service client
	if connection, err = grpc.NewClient(h.Config.UserServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials())); err != nil {
		log.Fatalf("Failed to connect to user service: %v", err)
	}
	h.UserClient = userServiceProto.NewUserServiceClient(connection)

	// init kafka writer
	h.KafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(h.Config.KafkaServiceURL),
		Balancer: &kafka.LeastBytes{},
	}
}

func NewHandler(cfg *config.Config) *Handler {
	handler := Handler{Config: cfg}
	handler.init()
	return &handler
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var loginReq userServiceProto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	grpcReq := &userServiceProto.LoginRequest{
		Name:     loginReq.Name,
		Password: loginReq.Password,
	}

	forwardGrpcRequest(
		w,
		grpcReq,
		func(ctx context.Context, req interface{}) (interface{}, error) {
			return h.UserClient.Login(ctx, req.(*userServiceProto.LoginRequest))
		},
		func(resp interface{}) (map[string]interface{}, error) {
			grpcResp := resp.(*userServiceProto.LoginResponse)
			return map[string]interface{}{
				"userId":                grpcResp.UserId,
				"accessToken":           grpcResp.AccessToken,
				"accessTokenExpiresAt":  grpcResp.AccessTokenExpiresAt,
				"refreshToken":          grpcResp.RefreshToken,
				"refreshTokenExpiresAt": grpcResp.RefreshTokenExpiresAt,
				"message":               grpcResp.Message,
			}, nil
		},
	)
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var registerReq userServiceProto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&registerReq); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	grpcReq := &userServiceProto.RegisterRequest{
		Name:     registerReq.Name,
		Password: registerReq.Password,
	}

	forwardGrpcRequest(
		w,
		grpcReq,
		func(ctx context.Context, req interface{}) (interface{}, error) {
			return h.UserClient.Register(ctx, req.(*userServiceProto.RegisterRequest))
		},
		func(resp interface{}) (map[string]interface{}, error) {
			grpcResp := resp.(*userServiceProto.RegisterResponse)
			return map[string]interface{}{
				"userId":  grpcResp.UserId,
				"message": grpcResp.Message,
			}, nil
		},
	)
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var refreshTokenReq userServiceProto.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&refreshTokenReq); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Prepare the gRPC request
	grpcReq := &userServiceProto.RefreshTokenRequest{
		UserId:       refreshTokenReq.UserId,
		RefreshToken: refreshTokenReq.RefreshToken,
	}

	// Forward the request to the user service
	forwardGrpcRequest(
		w,
		grpcReq,
		func(ctx context.Context, req interface{}) (interface{}, error) {
			return h.UserClient.RefreshToken(ctx, req.(*userServiceProto.RefreshTokenRequest))
		},
		func(resp interface{}) (map[string]interface{}, error) {
			grpcResp := resp.(*userServiceProto.RefreshTokenResponse)
			return map[string]interface{}{
				"userId":                grpcResp.UserId,
				"accessToken":           grpcResp.AccessToken,
				"refreshToken":          grpcResp.RefreshToken,
				"accessTokenExpiresAt":  grpcResp.AccessTokenExpiresAt,
				"refreshTokenExpiresAt": grpcResp.RefreshTokenExpiresAt,
			}, nil
		},
	)
}

func (h *Handler) TestNotification(w http.ResponseWriter, r *http.Request) {
	type request struct {
		UserID  string `json:"user_id"`
		Message string `json:"message"`
	}

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid input", http.StatusBadRequest)
		return
	}

	msg, err := json.Marshal(req)
	if err != nil {
		http.Error(w, "Failed to serialize message", http.StatusInternalServerError)
		return
	}

	err = h.KafkaWriter.WriteMessages(r.Context(), kafka.Message{
		Topic: "default",
		Key:   []byte(req.UserID), // Use UserID as key for partitioning
		Value: msg,
	})
	if err != nil {
		http.Error(w, "Failed to send notification: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Notification sent to Kafka"))
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ProxyWebSocket redirects /ws requests to establish websocket connection
func (h *Handler) ProxyWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade the client connection
	clientConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade client WebSocket: %v", err)
		http.Error(w, "Failed to upgrade WebSocket", http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	// Dial the backend Notification Service
	notificationURL := url.URL{Scheme: "ws", Host: h.Config.NotificationServiceURL, Path: "/ws", RawQuery: r.URL.RawQuery}
	backendConn, _, err := websocket.DefaultDialer.Dial(notificationURL.String(), nil)
	if err != nil {
		log.Printf("Failed to connect to backend WebSocket: %v", err)
		http.Error(w, "Failed to connect to backend WebSocket", http.StatusInternalServerError)
		return
	}
	defer backendConn.Close()

	// Proxy messages between client and backend
	errChan := make(chan error, 2)

	go proxyMessages(clientConn, backendConn, errChan)
	go proxyMessages(backendConn, clientConn, errChan)

	<-errChan // Wait for any error
}

func proxyMessages(src, dest *websocket.Conn, errChan chan error) {
	for {
		messageType, message, err := src.ReadMessage()
		if err != nil {
			errChan <- err
			return
		}
		err = dest.WriteMessage(messageType, message)
		if err != nil {
			errChan <- err
			return
		}
	}
}

// Helper function to forward requests
func forwardGrpcRequest(
	w http.ResponseWriter,
	grpcReq interface{},
	grpcCall func(context.Context, interface{}) (interface{}, error),
	responseMapper func(interface{}) (map[string]interface{}, error),
) {
	ctx := context.Background()

	// Call the specified gRPC function
	grpcResp, err := grpcCall(ctx, grpcReq)
	if err != nil {
		http.Error(w, "Request failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Map the gRPC response to a JSON response
	response, err := responseMapper(grpcResp)
	if err != nil {
		http.Error(w, "Failed to map response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Send the JSON response to the client
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
