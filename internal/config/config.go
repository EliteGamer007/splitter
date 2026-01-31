package config

import (
	"os"

	_ "github.com/joho/godotenv/autoload"
)

// Config holds all application configuration
type Config struct {
	Database DatabaseConfig
	Server   ServerConfig
	JWT      JWTConfig
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
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

// Load reads configuration from environment variables
func Load() *Config {
	return &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "splitter_user"),
			Password: os.Getenv("DB_PASSWORD"),
			Name:     getEnv("DB_NAME", "splitter_db"),
			MaxConns: 25,
			MinConns: 5,
		},
		Server: ServerConfig{
			Port:    getEnv("PORT", "8080"),
			Env:     getEnv("ENV", "development"),
			BaseURL: getEnv("BASE_URL", "http://localhost:3000"),
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "your-secret-key"),
			Expiration: 24, // 24 hours
		},
	}
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
