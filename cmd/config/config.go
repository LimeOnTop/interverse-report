package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port    string
	DevMode bool
	DatabaseConfig
	RedisConfig
	KafkaConfig
	GeminiAPIKey        string
	GeminiModel         string
	InterviewServiceURL string
	QuestionServiceURL  string
}

type DatabaseConfig struct {
	DatabaseURL string
}

type RedisConfig struct {
	RedisAddr      string
	RedisAnswersDB int
}

type KafkaConfig struct {
	KafkaBrokers         string
	KafkaTopicReports    string
	KafkaTopicReportsDLQ string
	KafkaGroupID         string
}

func Load() *Config {
	_ = godotenv.Load()

	return &Config{
		Port:    getEnv("PORT", "50054"),
		DevMode: getEnvBool("DEV_MODE", false),
		DatabaseConfig: DatabaseConfig{
			DatabaseURL: getEnv("DATABASE_URL", "postgres://user:password@localhost/interverse?sslmode=disable"),
		},
		RedisConfig: RedisConfig{
			RedisAddr:      getEnv("REDIS_ADDR", "localhost:6379"),
			RedisAnswersDB: getEnvInt("REDIS_ANSWERS_DB", 2),
		},
		KafkaConfig: KafkaConfig{
			KafkaBrokers:         getEnv("KAFKA_BROKERS", "localhost:9092"),
			KafkaTopicReports:    getEnv("KAFKA_TOPIC_REPORTS", "training.report.analyze"),
			KafkaTopicReportsDLQ: getEnv("KAFKA_TOPIC_REPORTS_DLQ", "training.report.analyze.dlq"),
			KafkaGroupID:         getEnv("KAFKA_GROUP_ID", "report-workers"),
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

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}
