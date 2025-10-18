package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                string
	DatabaseURL         string
	UserServiceURL      string
	InterviewServiceURL string
}

func Load() *Config {
	// Load .env file if exists
	godotenv.Load()

	return &Config{
		Port:                getEnv("PORT", "50054"),
		DatabaseURL:         getEnv("DATABASE_URL", "postgres://user:password@localhost/interverse?sslmode=disable"),
		UserServiceURL:      getEnv("USER_SERVICE_URL", "user-service:50051"),
		InterviewServiceURL: getEnv("INTERVIEW_SERVICE_URL", "interview-service:50052"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

