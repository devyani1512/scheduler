package config

import "os"

type Config struct {
	DatabaseURL string
	ServerPort  string
	LogLevel    string
}

func Load() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://user:password@postgres:5432/db?sslmode=disable"),
		ServerPort:  getEnv("SERVER_PORT", "8080"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
