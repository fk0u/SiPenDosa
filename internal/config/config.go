package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config represents the application configuration
type Config struct {
	Port            string
	Host            string
	AppEnv          string
	SessionSecret   string
	DBPath          string
	WASessionPath   string
	DefaultTimezone string
	SendWindowStart string
	SendWindowEnd   string
	RateLimitMinSec int
	RateLimitMaxSec int
	MaxRetries      int
	GlobalDryRun    bool
}

// Load loads configuration from .env file and environment variables
func Load() (*Config, error) {
	// Try loading .env file, ignore error if it does not exist
	_ = godotenv.Load()

	cfg := &Config{
		Port:            getEnv("PORT", "8473"),
		Host:            getEnv("HOST", "0.0.0.0"),
		AppEnv:          getEnv("APP_ENV", "production"),
		SessionSecret:   getEnv("SESSION_SECRET", "sipen-secure-session-key-must-be-changed-in-production"),
		DBPath:          getEnv("DB_PATH", "data/sipen.db"),
		WASessionPath:   getEnv("WA_SESSION_PATH", "session/whatsapp.db"),
		DefaultTimezone: getEnv("DEFAULT_TIMEZONE", "Asia/Makassar"),
		SendWindowStart: getEnv("SEND_WINDOW_START", "08:00"),
		SendWindowEnd:   getEnv("SEND_WINDOW_END", "16:00"),
		RateLimitMinSec: getEnvInt("RATE_LIMIT_MIN_SEC", 5),
		RateLimitMaxSec: getEnvInt("RATE_LIMIT_MAX_SEC", 15),
		MaxRetries:      getEnvInt("MAX_RETRIES", 3),
		GlobalDryRun:    getEnvBool("GLOBAL_DRY_RUN", false),
	}

	return cfg, nil
}

// GetLocation returns the time.Location based on DefaultTimezone
func (c *Config) GetLocation() *time.Location {
	loc, err := time.LoadLocation(c.DefaultTimezone)
	if err != nil {
		// Fallback to UTC+8 if timezone loading fails
		return time.FixedZone("WITA", 8*3600)
	}
	return loc
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if val, ok := os.LookupEnv(key); ok {
		if boolVal, err := strconv.ParseBool(val); err == nil {
			return boolVal
		}
	}
	return fallback
}
