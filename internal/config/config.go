package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds application configuration values.
type Config struct {
	Port            string
	DBPath          string
	JWTSecret       string
	DefaultCurrency string
	SMTPHost        string
	SMTPPort        string
	SMTPUser        string
	SMTPPass        string
	SMTPFrom        string
	AppBaseURL      string
	MasterKey       string
	TrustedProxies  string
}

func Load() (*Config, error) {
	// Attempt to load .env file, ignore if it doesn't exist
	_ = godotenv.Load()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	return &Config{
		Port:            getEnv("PORT", ":8080"),
		DBPath:          getEnv("DB_PATH", "expense.db"),
		JWTSecret:       jwtSecret,
		DefaultCurrency: getEnv("DEFAULT_CURRENCY", "USD"),
		SMTPHost:        getEnv("SMTP_HOST", "auth.smtp.1and1.fr"), // Preserving IT context if possible or generic
		SMTPPort:        getEnv("SMTP_PORT", "587"),
		SMTPUser:        getEnv("SMTP_USER", ""),
		SMTPPass:        getEnv("SMTP_PASS", ""),
		SMTPFrom:        getEnv("SMTP_FROM", "no-reply@personalexpense.it"),
		AppBaseURL:      getEnv("APP_BASE_URL", "http://localhost:8089"),
		MasterKey:       getEnv("MASTER_KEY", "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"),
		TrustedProxies:  getEnv("TRUSTED_PROXIES", ""),
	}, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
