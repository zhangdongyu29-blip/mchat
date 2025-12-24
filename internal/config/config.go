package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// Config holds application configuration.
type Config struct {
	AppPort          string
	MySQLHost        string
	MySQLPort        string
	MySQLUser        string
	MySQLPassword    string
	MySQLDB          string
	OpenRouterAPIKey string
	OpenRouterURL    string
}

// Load reads configuration from environment variables with sensible defaults.
func Load() Config {
	_ = godotenv.Load()

	cfg := Config{
		AppPort:          getEnv("APP_PORT", "5555"),
		MySQLHost:        getEnv("MYSQL_HOST", "mysql"),
		MySQLPort:        getEnv("MYSQL_PORT", "3306"),
		MySQLUser:        getEnv("MYSQL_USER", "root"),
		MySQLPassword:    getEnv("MYSQL_PASSWORD", "secret"),
		MySQLDB:          getEnv("MYSQL_DB", "mchat"),
		OpenRouterAPIKey: getEnv("OPENROUTER_API_KEY", ""),
		OpenRouterURL:    getEnv("OPENROUTER_URL", "https://openrouter.ai/api/v1/chat/completions"),
	}

	if cfg.OpenRouterAPIKey == "" {
		log.Println("[warn] OPENROUTER_API_KEY 未设置，调用大模型将失败")
	}

	return cfg
}

func (c Config) MySQLDSN() string {
	return c.MySQLUser + ":" + c.MySQLPassword + "@tcp(" + c.MySQLHost + ":" + c.MySQLPort + ")/" + c.MySQLDB + "?charset=utf8mb4&parseTime=True&loc=Local"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
