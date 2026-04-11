package config

import "time"

// AuthConfig holds JWT signing secrets and token expiry settings.
type AuthConfig struct {
	AccessSecret   string        // HMAC-SHA256 secret for access token
	RefreshSecret  string        // HMAC-SHA256 secret for refresh token
	AccessExpiry   time.Duration // how long an access token lives
	RefreshExpiry  time.Duration // how long a refresh token lives
}

// LoadAuth reads auth config from environment variables.
func LoadAuth() AuthConfig {
	accessMins := getEnvAsInt("JWT_ACCESS_EXPIRY_MIN", 15)
	refreshDays := getEnvAsInt("JWT_REFRESH_EXPIRY_DAYS", 7)

	return AuthConfig{
		AccessSecret:  getEnv("JWT_ACCESS_SECRET", "change-me-access-secret"),
		RefreshSecret: getEnv("JWT_REFRESH_SECRET", "change-me-refresh-secret"),
		AccessExpiry:  time.Duration(accessMins) * time.Minute,
		RefreshExpiry: time.Duration(refreshDays) * 24 * time.Hour,
	}
}
