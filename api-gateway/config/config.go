package config

import (
	"os"
)

type Config struct {
	UserServiceURL         string
	NotificationServiceURL string
	KafkaServiceURL        string
}

func LoadConfig() *Config {
	return &Config{
		UserServiceURL:         os.Getenv("USER_SERVICE_URL"),
		NotificationServiceURL: os.Getenv("NOTIFICATION_SERVICE_URL"),
		KafkaServiceURL:        os.Getenv("KAFKA_BROKER"),
	}
}
