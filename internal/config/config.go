package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

func init() {
	// Load env file: use ENV_FILE if set (for instance 2), otherwise default .env
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	if err := godotenv.Load(envFile); err != nil {
		log.Printf("Warning: Could not load env file %s: %v", envFile, err)
	}
}

// Config holds all application configuration
type Config struct {
	Database   DatabaseConfig
	Server     ServerConfig
	JWT        JWTConfig
	Federation FederationConfig
	Worker     WorkerConfig
	Bot        BotConfig
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
	MaxConns int32
	MinConns int32
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port    string
	Env     string
	BaseURL string
}

// JWTConfig holds JWT-related configuration
type JWTConfig struct {
	Secret     string
	Expiration int // in hours
}

// FederationConfig holds federation-related configuration
type FederationConfig struct {
	Domain  string // Instance domain identifier (e.g., "splitter-1")
	URL     string // Full URL of this instance (e.g., "http://localhost:8000")
	Enabled bool   // Whether federation is enabled
}

// WorkerConfig holds background worker configuration
type WorkerConfig struct {
	RetryIntervalSeconds      int
	ReputationIntervalSeconds int
	CircuitCooldownSeconds    int
	MaxRetryCount             int
	CircuitFailureThreshold   int
}

// BotConfig holds configuration for the Split AI reply bot
type BotConfig struct {
	ApiKey string
}

// Load reads configuration from environment variables
func Load() *Config {
	return &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "splitter_user"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     getEnv("DB_NAME", "splitter_db"),
			SSLMode:  getEnv("DB_SSLMODE", "require"),
			MaxConns: int32(boundedEnvInt("DB_MAX_CONNS", 4, 1, 8)),
			MinConns: 0,
		},
		Server: ServerConfig{
			Port:    getEnv("PORT", "8000"),
			Env:     getEnv("ENV", "development"),
			BaseURL: getEnv("BASE_URL", "http://localhost:8000"),
		},
		JWT: JWTConfig{
			Secret:     jwtSecret(),
			Expiration: boundedEnvInt("JWT_EXPIRATION_HOURS", 24, 1, 24),
		},
		Federation: FederationConfig{
			Domain:  getEnv("FEDERATION_DOMAIN", "localhost"),
			URL:     getEnv("FEDERATION_URL", "http://localhost:8000"),
			Enabled: strings.EqualFold(getEnv("FEDERATION_ENABLED", "true"), "true"),
		},
		Worker: WorkerConfig{
			RetryIntervalSeconds:      getEnvAsInt("WORKER_RETRY_INTERVAL_SECONDS", 15),
			ReputationIntervalSeconds: getEnvAsInt("WORKER_REPUTATION_INTERVAL_SECONDS", 60),
			CircuitCooldownSeconds:    getEnvAsInt("WORKER_CIRCUIT_COOLDOWN_SECONDS", 300),
			MaxRetryCount:             getEnvAsInt("WORKER_MAX_RETRY_COUNT", 6),
			CircuitFailureThreshold:   getEnvAsInt("WORKER_CIRCUIT_FAILURE_THRESHOLD", 5),
		},
		Bot: BotConfig{
			ApiKey: getEnvWithFallback("SPLIT_BOT_API_KEY", "GEMINI_API_KEY"),
		},
	}
}

var devSecretOnce sync.Once
var devSecret string

func jwtSecret() string {
	if secret := strings.TrimSpace(os.Getenv("JWT_SECRET")); secret != "" {
		return secret
	}
	if strings.EqualFold(os.Getenv("ENV"), "production") {
		return ""
	}
	devSecretOnce.Do(func() {
		var b [32]byte
		if _, err := rand.Read(b[:]); err != nil {
			panic("cannot initialize development JWT key")
		}
		devSecret = hex.EncodeToString(b[:])
	})
	return devSecret
}

func boundedEnvInt(name string, fallback, minimum, maximum int) int {
	n, err := strconv.Atoi(os.Getenv(name))
	if err != nil {
		return fallback
	}
	if n < minimum {
		return minimum
	}
	if n > maximum {
		return maximum
	}
	return n
}

// Validate rejects unsafe production defaults before opening a database connection.
func (c *Config) Validate() error {
	if c.Database.Host == "" || c.Database.User == "" || c.Database.Name == "" {
		return fmt.Errorf("database host, user and name are required")
	}
	if strings.EqualFold(c.Server.Env, "production") {
		if len(c.JWT.Secret) < 32 || c.JWT.Secret == "your-secret-key" {
			return fmt.Errorf("JWT_SECRET must contain at least 32 characters in production")
		}
		u, err := url.Parse(c.Server.BaseURL)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil {
			return fmt.Errorf("BASE_URL must be a public HTTPS origin in production")
		}
		if c.Database.SSLMode == "disable" {
			return fmt.Errorf("database TLS cannot be disabled in production")
		}
	}
	return nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvWithFallback tries the primary key, then falls back to a secondary key
func getEnvWithFallback(primary, fallback string) string {
	if value := os.Getenv(primary); value != "" {
		return value
	}
	return os.Getenv(fallback)
}

func getEnvAsInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed := 0
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return defaultValue
		}
		parsed = parsed*10 + int(ch-'0')
	}
	if parsed <= 0 {
		return defaultValue
	}
	return parsed
}
