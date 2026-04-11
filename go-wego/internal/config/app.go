package config

import (
	"os"
	"strconv"
	"time"
)

// AppConfig holds general application configuration.
type AppConfig struct {
	Env        string // "development" | "production"
	Port       string // HTTP listen port, e.g. "8080"
	UploadPath string // Local directory to store uploaded files
	TimeZone   string // IANA timezone name for app-level time formatting
}

// LoadApp reads application config from environment variables with safe defaults.
func LoadApp() AppConfig {
	return AppConfig{
		Env:        getEnv("APP_ENV", "development"),
		Port:       getEnv("APP_PORT", "8080"),
		UploadPath: getEnv("UPLOAD_PATH", "./uploads"),
		TimeZone:   getEnv("APP_TIMEZONE", "Asia/Bangkok"),
	}
}

// Location returns the parsed *time.Location for this app's timezone.
// Falls back to UTC if the timezone name is invalid (never panics).
//
// Usage: time.Now().In(appCfg.Location())
func (c AppConfig) Location() *time.Location {
	loc, err := time.LoadLocation(c.TimeZone)
	if err != nil {
		return time.UTC
	}
	return loc
}

// NowInTZ returns the current time in the configured application timezone.
// Use this instead of time.Now() when you need Thai local time.
func (c AppConfig) NowInTZ() time.Time {
	return time.Now().In(c.Location())
}

// getEnv returns the value of the env var key, or fallback if unset.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvAsInt returns the env var as int, or fallback if unset / invalid.
func getEnvAsInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
