package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

type Config struct {
	Port string

	MongoURI string
	MongoDB  string

	JWTSecret   string
	JWTClaimKey string
	JWKSURL     string

	UserInfoURL        string
	UserSyncTTLSeconds int

	PingIntervalSec int
	PongTimeoutSec  int
	MaxMessageBytes int

	OfflineQueueTTLDays int

	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int
	RedisUseTLS   bool

	WSTicketTTLSec int

	AllowedOrigins []string

	LogLevel string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:                getEnv("PORT", "8080"),
		MongoURI:            getEnv("MONGO_URI", ""),
		MongoDB:             getEnv("MONGO_DB", "gokan"),
		JWTSecret:           getEnv("JWT_SECRET", ""),
		JWTClaimKey:         getEnv("JWT_CLAIM_KEY", "sub"),
		JWKSURL:             getEnv("JWKS_URL", ""),
		UserInfoURL:         getEnv("USER_INFO_URL", ""),
		UserSyncTTLSeconds:  getEnvInt("USER_SYNC_TTL_SECONDS", 300),
		PingIntervalSec:     getEnvInt("PING_INTERVAL_SEC", 30),
		PongTimeoutSec:      getEnvInt("PONG_TIMEOUT_SEC", 10),
		MaxMessageBytes:     getEnvInt("MAX_MESSAGE_BYTES", 65536),
		OfflineQueueTTLDays: getEnvInt("OFFLINE_QUEUE_TTL_DAYS", 30),
		RedisHost:           getEnv("REDIS_HOST", "localhost"),
		RedisPort:           getEnv("REDIS_PORT", "6379"),
		RedisPassword:       getEnv("REDIS_PASSWORD", ""),
		RedisDB:             getEnvInt("REDIS_DB", 0),
		RedisUseTLS:         getEnvBool("REDIS_USE_TLS", false),
		WSTicketTTLSec:      getEnvInt("WS_TICKET_TTL_SEC", 30),
		AllowedOrigins:      getEnvStringSlice("ALLOWED_ORIGINS", []string{"*"}),
		LogLevel:            getEnv("LOG_LEVEL", "info"),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	if cfg.WSTicketTTLSec < 10 || cfg.WSTicketTTLSec > 300 {
		log.Warn().Int("value", cfg.WSTicketTTLSec).Msg("WS_TICKET_TTL_SEC out of range [10, 300], using default 30")
		cfg.WSTicketTTLSec = 30
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
		log.Warn().Str("key", key).Str("value", value).Msg("invalid integer value, using default")
	}
	return defaultValue
}

func (c *Config) UserSyncTTL() time.Duration {
	return time.Duration(c.UserSyncTTLSeconds) * time.Second
}

func (c *Config) PingInterval() time.Duration {
	return time.Duration(c.PingIntervalSec) * time.Second
}

func (c *Config) PongTimeout() time.Duration {
	return time.Duration(c.PongTimeoutSec) * time.Second
}

func (c *Config) OfflineQueueTTL() time.Duration {
	return time.Duration(c.OfflineQueueTTLDays) * 24 * time.Hour
}

func (c *Config) RedisAddr() string {
	return c.RedisHost + ":" + c.RedisPort
}

func (c *Config) WSTicketTTL() time.Duration {
	return time.Duration(c.WSTicketTTLSec) * time.Second
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
		log.Warn().Str("key", key).Str("value", value).Msg("invalid boolean value, using default")
	}
	return defaultValue
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		parts := make([]string, 0)
		for _, part := range strings.Split(value, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				parts = append(parts, trimmed)
			}
		}
		if len(parts) > 0 {
			return parts
		}
	}
	return defaultValue
}

func (c *Config) validate() error {
	if c.MongoURI == "" {
		return fmt.Errorf("MONGO_URI is required")
	}

	if c.JWTSecret == "" && c.JWKSURL == "" {
		return fmt.Errorf("JWT_SECRET or JWKS_URL is required")
	}

	if c.UserInfoURL == "" {
		return fmt.Errorf("USER_INFO_URL is required")
	}

	if port, err := strconv.Atoi(c.Port); err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("PORT must be a valid number between 1 and 65535, got: %s", c.Port)
	}

	if redisPort, err := strconv.Atoi(c.RedisPort); err != nil || redisPort < 1 || redisPort > 65535 {
		return fmt.Errorf("REDIS_PORT must be a valid number between 1 and 65535, got: %s", c.RedisPort)
	}

	if c.RedisHost == "" {
		return fmt.Errorf("REDIS_HOST cannot be empty")
	}

	validLogLevels := map[string]bool{
		"trace": true, "debug": true, "info": true, "warn": true, "error": true, "fatal": true, "panic": true,
	}
	if !validLogLevels[c.LogLevel] {
		return fmt.Errorf("LOG_LEVEL must be one of: trace, debug, info, warn, error, fatal, panic, got: %s", c.LogLevel)
	}

	return nil
}
