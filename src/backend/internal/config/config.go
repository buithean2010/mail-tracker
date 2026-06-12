package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	// Microsoft Entra ID
	AzureClientID     string
	AzureClientSecret string
	AzureTenantID     string
	AzureRedirectURI  string

	// Database
	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string

	// Security
	AESKey        string
	SessionSecret string

	// App
	AppEnv              string
	BaseURL             string
	SyncIntervalMinutes int
}

func (c *Config) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}

func Load() (*Config, error) {
	// Load .env in development (ignored if file doesn't exist)
	_ = godotenv.Load()

	cfg := &Config{
		AzureClientID:     mustEnv("AZURE_CLIENT_ID"),
		AzureClientSecret: mustEnv("AZURE_CLIENT_SECRET"),
		AzureTenantID:     mustEnv("AZURE_TENANT_ID"),
		AzureRedirectURI:  mustEnv("AZURE_REDIRECT_URI"),

		DBHost:     getEnv("DB_HOST", "postgres"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBName:     mustEnv("DB_NAME"),
		DBUser:     mustEnv("DB_USER"),
		DBPassword: mustEnv("DB_PASSWORD"),

		AESKey:        mustEnv("AES_KEY"),
		SessionSecret: mustEnv("SESSION_SECRET"),

		AppEnv:  getEnv("APP_ENV", "development"),
		BaseURL: getEnv("BASE_URL", "http://localhost"),
	}

	interval, err := strconv.Atoi(getEnv("SYNC_INTERVAL_MINUTES", "30"))
	if err != nil {
		return nil, fmt.Errorf("SYNC_INTERVAL_MINUTES must be an integer: %w", err)
	}
	cfg.SyncIntervalMinutes = interval

	return cfg, nil
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
