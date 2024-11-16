package utils

import (
	"context"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
	"user-service/controllers"
	"user-service/models"
	"user-service/proto"
	"user-service/test"
	"user-service/utils"
)

func generateExpiredToken(userID uint, expiration time.Time) string {
	claims := &utils.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiration),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredToken, _ := token.SignedString(utils.JwtSecret)
	return expiredToken
}

// Mock time setup for expired tokens
var expiredTokenTime = time.Now().Add(-24 * time.Hour) // 24 hours in the past

// TestRefreshTokenSuccess checks correct refresh token attempt
func TestRefreshTokenSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userID := uint(1)
	repository := test.NewMockUserRepository()
	repository.CreateUser(&models.User{
		ID:   userID,
		Name: "test@example.com",
	})

	server := &controllers.UserServiceServer{UserRepo: repository}
	conn := test.InitGrpcServer(t, server)
	defer conn.Close()

	// Generate a valid refresh token for the user
	refreshToken, _, err := utils.GenerateRefreshToken(userID)
	assert.NoError(t, err)

	request := proto.RefreshTokenRequest{
		UserId:       int32(userID),
		RefreshToken: refreshToken,
	}

	client := proto.NewUserServiceClient(conn)
	response, err := client.RefreshToken(context.Background(), &request)
	if err != nil {
		t.Fatalf("Refresh token failed: %v", err)
	}

	// Check tokens and expiration dates
	assert.NotEmpty(t, response.AccessToken)
	assert.NotEmpty(t, response.RefreshToken)

	atExpiresAt, rtExpiresAt := time.Unix(response.AccessTokenExpiresAt, 0), time.Unix(response.RefreshTokenExpiresAt, 0)

	assert.True(t, atExpiresAt.After(time.Now()))
	assert.True(t, rtExpiresAt.After(time.Now().Add(24*time.Hour)))
}

// TestRefreshTokenExpiredToken checks expired token case
func TestRefreshTokenExpiredToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userID := uint(1)
	repository := test.NewMockUserRepository()
	repository.CreateUser(&models.User{
		ID:   userID,
		Name: "test@example.com",
	})

	server := &controllers.UserServiceServer{UserRepo: repository}
	conn := test.InitGrpcServer(t, server)
	defer conn.Close()

	expiredToken := generateExpiredToken(userID, expiredTokenTime)

	request := proto.RefreshTokenRequest{
		UserId:       int32(userID),
		RefreshToken: expiredToken,
	}

	client := proto.NewUserServiceClient(conn)
	_, err := client.RefreshToken(context.Background(), &request)

	assert.Contains(t, err.Error(), "invalid or expired refresh token")
}

// TestRefreshTokenInvalidToken checks incorrect token
func TestRefreshTokenInvalidToken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repository := test.NewMockUserRepository()

	server := &controllers.UserServiceServer{UserRepo: repository}
	conn := test.InitGrpcServer(t, server)
	defer conn.Close()

	invalidRefreshToken := "invalid.token.string"
	request := proto.RefreshTokenRequest{
		UserId:       int32(1),
		RefreshToken: invalidRefreshToken,
	}

	client := proto.NewUserServiceClient(conn)
	_, err := client.RefreshToken(context.Background(), &request)

	assert.Contains(t, err.Error(), "invalid or expired refresh token")
}
