package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	Server          ServerConfig
	WhatsApp        WhatsAppConfig
	Otomax          OtomaxConfig
	Security        SecurityConfig
	RateLimit       RateLimitConfig
	MessageTracking MessageTrackingConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port string
	Host string
}

// WhatsAppConfig holds WhatsApp configuration
type WhatsAppConfig struct {
	DBPath   string
	LogLevel string
}

// OtomaxConfig holds Otomax webhook configuration
type OtomaxConfig struct {
	WebhookURL     string
	WebhookTimeout time.Duration
	RetryCount     int
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	APIKey string
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	MaxMessagesPerSecond int
}

// MessageTrackingConfig holds message tracking configuration
type MessageTrackingConfig struct {
	TTL              time.Duration
	TrackingDBPath   string
	WebhookWhitelist []string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if exists (ignore error if not found)
	_ = godotenv.Load()

	config := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Host: getEnv("HOST", "0.0.0.0"),
		},
		WhatsApp: WhatsAppConfig{
			DBPath:   getEnv("WA_DB_PATH", "./db/whatsmeow.db"),
			LogLevel: getEnv("WA_LOG_LEVEL", "INFO"),
		},
		Otomax: OtomaxConfig{
			WebhookURL:     getEnv("OTOMAX_WEBHOOK_URL", ""),
			WebhookTimeout: parseDuration(getEnv("OTOMAX_WEBHOOK_TIMEOUT", "10s"), 10*time.Second),
			RetryCount:     parseInt(getEnv("OTOMAX_WEBHOOK_RETRY_COUNT", "3"), 3),
		},
		Security: SecurityConfig{
			APIKey: getEnv("API_KEY", ""),
		},
		RateLimit: RateLimitConfig{
			MaxMessagesPerSecond: parseInt(getEnv("MAX_MESSAGES_PER_SECOND", "5"), 5),
		},
		MessageTracking: MessageTrackingConfig{
			TTL:              parseDuration(getEnv("MESSAGE_TRACKING_TTL", "24h"), 24*time.Hour),
			TrackingDBPath:   getEnv("TRACKING_DB_PATH", "./db/tracking.db"),
			WebhookWhitelist: parseStringList(getEnv("WEBHOOK_WHITELIST_JIDS", "")),
		},
	}

	// Validate required fields
	if config.Otomax.WebhookURL == "" {
		return nil, fmt.Errorf("OTOMAX_WEBHOOK_URL is required")
	}

	return config, nil
}

// getEnv gets environment variable with default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// parseInt parses string to int with default value
func parseInt(value string, defaultValue int) int {
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

// parseDuration parses string to time.Duration with default value
func parseDuration(value string, defaultValue time.Duration) time.Duration {
	if value == "" {
		return defaultValue
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return duration
}

// parseStringList parses comma-separated string to slice
func parseStringList(value string) []string {
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

