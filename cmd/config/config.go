package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port string
	DatabaseConfig
}

type DatabaseConfig struct {
	DatabaseURL string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		Port: getEnv("PORT", "50054"),
		DatabaseConfig: DatabaseConfig{
			DatabaseURL: getEnv("DATABASE_URL", "postgres://user:password@localhost/interverse?sslmode=disable"),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
