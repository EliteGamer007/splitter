package db

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/url"
	"time"

	"splitter/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

// InitDB initializes the PostgreSQL database connection pool
func InitDB(cfg *config.Config) error {
	poolConfig, err := BuildPoolConfig(cfg)
	if err != nil {
		return err
	}
	// Create connection pool
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return fmt.Errorf("unable to ping database: %w", err)
	}

	DB = pool
	log.Println("Database connection established successfully")
	return nil
}

// Close closes the database connection pool
func Close() {
	if DB != nil {
		DB.Close()
		log.Println("Database connection closed")
	}
}

// GetDB returns the database connection pool
func GetDB() *pgxpool.Pool {
	return DB
}

// BuildPoolConfig uses escaped URI components and bounded, idle-friendly settings.
func BuildPoolConfig(cfg *config.Config) (*pgxpool.Config, error) {
	u := url.URL{Scheme: "postgres", Host: net.JoinHostPort(cfg.Database.Host, cfg.Database.Port), Path: "/" + cfg.Database.Name, User: url.UserPassword(cfg.Database.User, cfg.Database.Password)}
	q := u.Query()
	q.Set("sslmode", cfg.Database.SSLMode)
	q.Set("connect_timeout", "5")
	u.RawQuery = q.Encode()
	p, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		return nil, fmt.Errorf("invalid database configuration")
	}
	p.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeExec
	p.MaxConns = cfg.Database.MaxConns
	if p.MaxConns < 1 || p.MaxConns > 8 {
		p.MaxConns = 4
	}
	p.MinConns = 0
	p.MaxConnIdleTime = time.Minute
	p.MaxConnLifetime = 30 * time.Minute
	p.HealthCheckPeriod = 30 * time.Second
	p.ConnConfig.RuntimeParams["statement_timeout"] = "10000"
	p.ConnConfig.RuntimeParams["idle_in_transaction_session_timeout"] = "10000"
	return p, nil
}
