package config

import (
	"log/slog"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	PORT         string
	DATABASE_URL string
}

func NewConfig() *Config {
	err := godotenv.Load()

	if err != nil {
		slog.Error("Error loading .env file", err)
	}
	// Return the Config struct with values from environment variables or defaults.

	return &Config{
		PORT:         getEnv("PORT", "50051"),
		DATABASE_URL: getEnv("DATABASE_URL", "localhost:5432"),
	}
}
func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}
