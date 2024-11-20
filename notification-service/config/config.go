package config

import "os"

type Config struct {
	KafkaServiceURL string
}

func LoadConfig() *Config {
	return &Config{
		KafkaServiceURL: os.Getenv("KAFKA_BROKER"),
	}
}
