package config

import (
	"errors"
	"os"
	"time"
)

type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	TelegramToken   string
	LLMBaseURL      string
	LLMAPIKey       string
	LLMModel        string
	WebDir          string
	ShutdownTimeout time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        envOrDefault("HTTP_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		TelegramToken:   os.Getenv("TELEGRAM_BOT_TOKEN"),
		LLMBaseURL:      envOrDefault("LLM_BASE_URL", "https://api.openai.com/v1"),
		LLMAPIKey:       os.Getenv("LLM_API_KEY"),
		LLMModel:        envOrDefault("LLM_MODEL", "gpt-4o-mini"),
		WebDir:          envOrDefault("WEB_DIR", "web"),
		ShutdownTimeout: 10 * time.Second,
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}

	return cfg, nil
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
