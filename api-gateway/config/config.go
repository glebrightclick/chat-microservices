package config

import (
	"os"
)

type Config struct {
	UserServiceURL string
}

func LoadConfig() *Config {
	// In Kubernetes, the service URL for user-service can be reached with the DNS name
	userServiceURL := os.Getenv("USER_SERVICE_URL")
	return &Config{UserServiceURL: userServiceURL}
}
