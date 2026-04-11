package config

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// LoadRedis reads Redis config from environment variables.
func LoadRedis() RedisConfig {
	return RedisConfig{
		Host:     getEnv("REDIS_HOST", "localhost"),
		Port:     getEnv("REDIS_PORT", "6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       getEnvAsInt("REDIS_DB", 0),
	}
}

// Addr returns the Redis address "host:port".
func (c RedisConfig) Addr() string {
	return c.Host + ":" + c.Port
}
