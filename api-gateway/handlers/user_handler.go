package handlers

import (
	"api-gateway/config"
	"api-gateway/proto"
	"context"
	"encoding/json"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net/http"
	_ "net/url"
)

type UserHandler struct {
	Config *config.Config
	Client proto.UserServiceClient
}

func (h *UserHandler) InitUserServiceClient() {
	connection, err := grpc.NewClient(h.Config.UserServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to user service: %v", err)
	}
	h.Client = proto.NewUserServiceClient(connection)
}

func NewUserHandler(cfg *config.Config) *UserHandler {
	handler := UserHandler{Config: cfg}
	handler.InitUserServiceClient()
	return &handler
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var loginReq proto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	grpcReq := &proto.LoginRequest{
		Name:     loginReq.Name,
		Password: loginReq.Password,
	}

	forwardGrpcRequest(
		w,
		grpcReq,
		func(ctx context.Context, req interface{}) (interface{}, error) {
			return h.Client.Login(ctx, req.(*proto.LoginRequest))
		},
		func(resp interface{}) (map[string]interface{}, error) {
			grpcResp := resp.(*proto.LoginResponse)
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

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var registerReq proto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&registerReq); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	grpcReq := &proto.RegisterRequest{
		Name:     registerReq.Name,
		Password: registerReq.Password,
	}

	forwardGrpcRequest(
		w,
		grpcReq,
		func(ctx context.Context, req interface{}) (interface{}, error) {
			return h.Client.Register(ctx, req.(*proto.RegisterRequest))
		},
		func(resp interface{}) (map[string]interface{}, error) {
			grpcResp := resp.(*proto.RegisterResponse)
			return map[string]interface{}{
				"userId":  grpcResp.UserId,
				"message": grpcResp.Message,
			}, nil
		},
	)
}

func (h *UserHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var refreshTokenReq proto.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&refreshTokenReq); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Prepare the gRPC request
	grpcReq := &proto.RefreshTokenRequest{
		UserId:       refreshTokenReq.UserId,
		RefreshToken: refreshTokenReq.RefreshToken,
	}

	// Forward the request to the user service
	forwardGrpcRequest(
		w,
		grpcReq,
		func(ctx context.Context, req interface{}) (interface{}, error) {
			return h.Client.RefreshToken(ctx, req.(*proto.RefreshTokenRequest))
		},
		func(resp interface{}) (map[string]interface{}, error) {
			grpcResp := resp.(*proto.RefreshTokenResponse)
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
