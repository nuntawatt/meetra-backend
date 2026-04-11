package config

import "fmt"

// DBConfig holds PostgreSQL connection and pool settings.
type DBConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	TimeZone        string // e.g. "Asia/Bangkok" — applied to every connection session
	MaxOpenConns    int    // max open connections in the pool
	MaxIdleConns    int    // max idle connections in the pool
	ConnMaxLifetime int    // connection max lifetime in minutes
}

// LoadDB reads PostgreSQL config from environment variables.
func LoadDB() DBConfig {
	return DBConfig{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnv("DB_PORT", "5432"),
		User:            getEnv("DB_USER", "postgres"),
		Password:        getEnv("DB_PASSWORD", "postgres"),
		DBName:          getEnv("DB_NAME", "wego"),
		SSLMode:         getEnv("DB_SSLMODE", "disable"),
		TimeZone:        getEnv("DB_TIMEZONE", "Asia/Bangkok"),
		MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
		ConnMaxLifetime: getEnvAsInt("DB_CONN_MAX_LIFETIME", 5),
	}
}

// DSN returns the PostgreSQL libpq connection string.
//
// Why timezone here?
// PostgreSQL stores TIMESTAMPTZ as UTC internally, but when you call now() or
// display a TIMESTAMPTZ value, it uses the *session* timezone.
// Setting "timezone=Asia/Bangkok" here means every connection in the pool will
// automatically use Thai time (UTC+7) for now(), CURRENT_TIMESTAMP, etc.
// Go's time.Time will carry the +07:00 offset correctly.
func (c DBConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s timezone=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode, c.TimeZone,
	)
}
