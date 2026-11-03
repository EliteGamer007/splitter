package db

import (
	"github.com/jackc/pgx/v5"
	"splitter/internal/config"
	"testing"
	"time"
)

func TestPoolConfigurationEscapesCredentialsAndAllowsIdleSleep(t *testing.T) {
	cfg := &config.Config{Database: config.DatabaseConfig{Host: "db.example", Port: "5432", User: "user@name", Password: "pass:@/?#", Name: "demo", SSLMode: "require", MaxConns: 4}}
	pool, err := BuildPoolConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if pool.ConnConfig.Password != cfg.Database.Password || pool.ConnConfig.User != cfg.Database.User {
		t.Fatal("credentials corrupted by URI construction")
	}
	if pool.MaxConns != 4 || pool.MinConns != 0 || pool.MaxConnIdleTime > time.Minute {
		t.Fatal("pool keeps database awake")
	}
	if pool.ConnConfig.DefaultQueryExecMode == pgx.QueryExecModeSimpleProtocol {
		t.Fatal("unnecessary simple protocol interpolation")
	}
}
