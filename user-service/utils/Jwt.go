package utils

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"os"
	"time"
)

var JwtSecret = []byte(os.Getenv("APP_SECRET"))
var refreshSecret = []byte(os.Getenv("REFRESH_SECRET"))

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateJWT generates a JWT token for a user.
func GenerateJWT(userID uint) (string, time.Time, error) {
	expiresAt := time.Now().Add(15 * time.Minute)
	clientExpiresAt := expiresAt.Add(-15 * time.Second)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(JwtSecret)

	return tokenString, clientExpiresAt, err
}

// GenerateRefreshToken generates a long-lived refresh JWT token.
func GenerateRefreshToken(userID uint) (string, time.Time, error) {
	// 7 days for refresh token
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(refreshSecret)

	return tokenString, expiresAt, err
}

// VerifyRefreshToken verifies the refresh JWT token.
func VerifyRefreshToken(tokenString string, userID uint) (bool, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return refreshSecret, nil
	})
	if err != nil || !token.Valid {
		return false, err
	}

	if claims.UserID != userID {
		return false, errors.New("invalid userID in token")
	}
	if claims.ExpiresAt.Unix() < time.Now().Unix() {
		return false, errors.New("token expired")
	}

	return true, nil
}
