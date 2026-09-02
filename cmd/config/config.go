package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port string
	DatabaseConfig
	GeminiAPIKey         string
	GeminiModel          string
	InterviewServiceURL  string
	QuestionServiceURL   string
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
		GeminiAPIKey:        getEnv("GEMINI_API_KEY", ""),
		GeminiModel:         getEnv("GEMINI_MODEL", "gemini-flash-latest"),
		InterviewServiceURL: getEnv("INTERVIEW_SERVICE_URL", "interview-service:50052"),
		QuestionServiceURL:  getEnv("QUESTION_SERVICE_URL", "question-service:50056"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
