package controllers

import (
	"context"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"user-service/models"
	"user-service/proto"
	"user-service/repositories"
	"user-service/utils"
)

type UserServiceServer struct {
	proto.UnimplementedUserServiceServer
	UserRepo repositories.UserRepository
}

// Register a new user
func (c *UserServiceServer) Register(ctx context.Context, request *proto.RegisterRequest) (*proto.RegisterResponse, error) {
	// Simple check for username
	if len(request.Name) < 1 {
		return nil, fmt.Errorf("invalid name")
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %v", err)
	}

	// Create user in DB
	user := models.User{Name: request.Name, Password: string(hashedPassword)}
	if _, err := c.UserRepo.FindUserByName(user.Name); err == nil {
		return nil, fmt.Errorf("user already exists")
	}

	if err := c.UserRepo.CreateUser(&user); err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	// Return success response
	return &proto.RegisterResponse{
		Message: "user registered successfully",
		UserId:  int32(user.ID),
	}, nil
}

// Login an existing user
func (c *UserServiceServer) Login(ctx context.Context, request *proto.LoginRequest) (*proto.LoginResponse, error) {
	if !request.IsValid() {
		return nil, fmt.Errorf("invalid input")
	}

	user, err := c.UserRepo.FindUserByName(request.Name)
	if err != nil {
		return nil, fmt.Errorf("user not found: %v", err)
	}

	// Compare password with hashed password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password)); err != nil {
		return nil, fmt.Errorf("invalid credentials: %v", err)
	}

	// Generate access and refresh tokens
	accessToken, accessTokenExpiresAt, err := utils.GenerateJWT(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %v", err)
	}

	refreshToken, refreshTokenExpiresAt, err := utils.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %v", err)
	}

	// Return login response with tokens
	return &proto.LoginResponse{
		UserId:                int32(user.ID),
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessTokenExpiresAt.Unix(),
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshTokenExpiresAt.Unix(),
		Message:               "login successful",
	}, nil
}

func (c *UserServiceServer) RefreshToken(ctx context.Context, request *proto.RefreshTokenRequest) (*proto.RefreshTokenResponse, error) {
	userID, refreshToken := uint(request.UserId), request.RefreshToken
	// Verify the provided refresh token
	if isVerified, _ := utils.VerifyRefreshToken(refreshToken, userID); !isVerified {
		return nil, fmt.Errorf("invalid or expired refresh token")
	}

	// Generate a new access token
	newAccessToken, newAccessTokenExpiresAt, err := utils.GenerateJWT(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %v", err)
	}

	// Generate a new refresh token
	newRefreshToken, newRefreshTokenExpiresAt, err := utils.GenerateRefreshToken(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %v", err)
	}

	return &proto.RefreshTokenResponse{
		UserId:                int32(userID),
		AccessToken:           newAccessToken,
		AccessTokenExpiresAt:  newAccessTokenExpiresAt.Unix(),
		RefreshToken:          newRefreshToken,
		RefreshTokenExpiresAt: newRefreshTokenExpiresAt.Unix(),
	}, nil
}
