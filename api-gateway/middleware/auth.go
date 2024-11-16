package middleware

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"os"
	"strings"
	"time"
)

func TokenAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Bypass token check for certain routes
		if isUnprotectedRoute(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract and validate JWT from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Unauthorized - No token", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		jwtSecret := []byte(os.Getenv("JWT_SECRET")) // Use the same jwtSecret

		// Parse token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Unauthorized - Invalid token", http.StatusUnauthorized)
			return
		}

		// Check token expiration
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			if exp, ok := claims["exp"].(float64); ok && time.Now().Unix() > int64(exp) {
				http.Error(w, "Unauthorized - Token expired", http.StatusUnauthorized)
				return
			}
		} else {
			http.Error(w, "Unauthorized - Invalid token claims", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isUnprotectedRoute(path string) bool {
	return path == "/health" || path == "/login" || path == "/register" || path == "/users/columns"
}
