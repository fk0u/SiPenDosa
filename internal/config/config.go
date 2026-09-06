package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
	var envDir string

	// 1. Cek .env di current working directory
	if _, err := os.Stat(".env"); err == nil {
		_ = godotenv.Load(".env")
		if pwd, err := os.Getwd(); err == nil {
			envDir = pwd
		}
	}

	// 2. Cek .env di direktori executable binary jika belum dimuat
	if envDir == "" {
		if exe, err := os.Executable(); err == nil {
			exeDir := filepath.Dir(exe)
			envCandidate := filepath.Join(exeDir, ".env")
			if _, err := os.Stat(envCandidate); err == nil {
				_ = godotenv.Load(envCandidate)
				envDir = exeDir
			}
		}
	}

	// 3. Cek .env di direktori standar instalasi Termux/Linux user (~/.sipendosa/.env)
	if envDir == "" {
		if home, err := os.UserHomeDir(); err == nil {
			homeEnv := filepath.Join(home, ".sipendosa", ".env")
			if _, err := os.Stat(homeEnv); err == nil {
				_ = godotenv.Load(homeEnv)
				envDir = filepath.Join(home, ".sipendosa")
			}
		}
	}

	rawDBPath := getEnv("DB_PATH", "data/sipen.db")
	rawWASessionPath := getEnv("WA_SESSION_PATH", "session/whatsapp.db")

	cfg := &Config{
		Port:            getEnv("PORT", "8473"),
		Host:            getEnv("HOST", "0.0.0.0"),
		AppEnv:          getEnv("APP_ENV", "production"),
		SessionSecret:   getEnv("SESSION_SECRET", "sipen-secure-session-key-must-be-changed-in-production"),
		DBPath:          resolvePath(rawDBPath, envDir),
		WASessionPath:   resolvePath(rawWASessionPath, envDir),
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

func resolvePath(p, baseDir string) string {
	if p == "" {
		return p
	}
	// Ekspansi home direktori jika menggunakan tilde ~
	if strings.HasPrefix(p, "~/") || p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				return home
			}
			return filepath.Join(home, p[2:])
		}
	}
	// Jika path absolut, biarkan apa adanya
	if filepath.IsAbs(p) {
		return p
	}
	// Jika baseDir ditemukan dan path relatif, gabungkan dengan baseDir
	if baseDir != "" {
		return filepath.Join(baseDir, p)
	}
	// Fallback ke ~/.sipendosa jika folder ~/.sipendosa ada
	if home, err := os.UserHomeDir(); err == nil {
		sipenDir := filepath.Join(home, ".sipendosa")
		if fi, err := os.Stat(sipenDir); err == nil && fi.IsDir() {
			return filepath.Join(sipenDir, p)
		}
	}
	return p
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
