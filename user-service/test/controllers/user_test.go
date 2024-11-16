package controllers

import (
	"context"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"testing"
	"time"
	"user-service/controllers"
	"user-service/models"
	"user-service/proto"
	"user-service/test"
)

// TestRegisterSuccess checks if user registration is successful
func TestRegisterSuccess(t *testing.T) {
	// Mock gRPC server implementation
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repository := test.NewMockUserRepository()
	server := &controllers.UserServiceServer{UserRepo: repository}
	request := proto.RegisterRequest{
		Name:     "testuser",
		Password: "password",
	}
	conn := test.InitGrpcServer(t, server)
	defer conn.Close()

	client := proto.NewUserServiceClient(conn)
	response, err := client.Register(context.Background(), &request)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	assert.Contains(t, response.Message, "user registered successfully")
}

// TestRegisterInvalidInput checks if user registration fails with invalid input
func TestRegisterInvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repository := test.NewMockUserRepository()

	server := &controllers.UserServiceServer{UserRepo: repository}
	conn := test.InitGrpcServer(t, server)
	defer conn.Close()

	request := proto.RegisterRequest{
		Name:     "",
		Password: "password",
	}

	client := proto.NewUserServiceClient(conn)
	_, err := client.Register(context.Background(), &request)
	if err == nil {
		t.Error("Expected error for invalid input, got nil:")
	}

	assert.Contains(t, err.Error(), "invalid name")
}

// TestRegisterUserAlreadyExists rejects registration if user exists
func TestRegisterUserAlreadyExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	existingUserName := "existinguser"
	repository := test.NewMockUserRepository()
	repository.CreateUser(&models.User{Name: existingUserName})

	server := &controllers.UserServiceServer{UserRepo: repository}
	conn := test.InitGrpcServer(t, server)
	defer conn.Close()

	request := proto.RegisterRequest{
		Name:     existingUserName,
		Password: "password",
	}

	client := proto.NewUserServiceClient(conn)
	_, err := client.Register(context.Background(), &request)
	if err == nil {
		t.Error("Expected error for invalid input, got nil")
	}

	assert.Contains(t, err.Error(), "user already exists")
}

// TestLoginSuccess checks correct login attempt
func TestLoginSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userID, userName, password := uint(1), "testuser", "password123"
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	repository := test.NewMockUserRepository()
	repository.CreateUser(&models.User{
		ID:       userID,
		Name:     userName,
		Password: string(passwordHash),
	})

	server := &controllers.UserServiceServer{UserRepo: repository}
	conn := test.InitGrpcServer(t, server)
	defer conn.Close()

	request := proto.LoginRequest{
		Name:     userName,
		Password: password,
	}

	client := proto.NewUserServiceClient(conn)
	response, err := client.Login(context.Background(), &request)
	if err != nil {
		t.Errorf("Login failed: %v", err)
	}

	assert.NotEmpty(t, response.AccessToken)
	assert.NotEmpty(t, response.RefreshToken)

	atExpiresAt, rtExpiresAt := time.Unix(response.AccessTokenExpiresAt, 0), time.Unix(response.RefreshTokenExpiresAt, 0)

	assert.True(t, atExpiresAt.After(time.Now().Add(-5*time.Second)))
	assert.True(t, rtExpiresAt.After(time.Now().Add(7*24*time.Hour-5*time.Second)))

	assert.Contains(t, response.Message, "login successful")
}

// TestLoginInvalidInput tests that login fails when there are no required fields
func TestLoginInvalidInput(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repository := test.NewMockUserRepository()

	server := &controllers.UserServiceServer{UserRepo: repository}
	conn := test.InitGrpcServer(t, server)
	defer conn.Close()

	request := proto.LoginRequest{
		// Missing required password
		Name: "invalid",
	}

	client := proto.NewUserServiceClient(conn)
	_, err := client.Login(context.Background(), &request)
	if err == nil {
		t.Errorf("Expected error for invalid input, got nil")
	}

	assert.Contains(t, err.Error(), "invalid input")
}

// TestLoginInvalidCredentials checks wrong password case
func TestLoginInvalidCredentials(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userName, password := "testuser", "password123"
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	repository := test.NewMockUserRepository()
	repository.CreateUser(&models.User{
		Name:     userName,
		Password: string(passwordHash),
	})

	server := &controllers.UserServiceServer{UserRepo: repository}
	connection := test.InitGrpcServer(t, server)
	defer connection.Close()

	request := proto.LoginRequest{
		Name:     userName,
		Password: "wrongpassword",
	}

	client := proto.NewUserServiceClient(connection)
	_, err := client.Login(context.Background(), &request)
	if err == nil {
		t.Errorf("Expected error for invalid input, got nil")
	}

	assert.Contains(t, err.Error(), "invalid credentials")
}

// TestLoginUserNotFound checks case when user doesn't exist
func TestLoginUserNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repository := test.NewMockUserRepository()

	server := &controllers.UserServiceServer{UserRepo: repository}
	conn := test.InitGrpcServer(t, server)
	defer conn.Close()

	request := proto.LoginRequest{
		Name:     "nonexistent",
		Password: "password123",
	}

	client := proto.NewUserServiceClient(conn)
	_, err := client.Login(context.Background(), &request)
	if err == nil {
		t.Errorf("Expected error for invalid input, got nil")
	}

	assert.Contains(t, err.Error(), "user not found")
}
