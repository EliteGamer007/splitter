package testenv

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"splitter/internal/config"
	"splitter/internal/db"
	"splitter/internal/server"
	"splitter/tests/testdb"
)

// ServerURL creates the application around an isolated disposable schema.
// HTTP tests never use a developer's already-running backend or application DB.
func ServerURL(t *testing.T) string {
	t.Helper()
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("requires explicit disposable TEST_DATABASE_URL")
	}
	if _, err := testdb.ParseLocalURL(raw); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("http_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 4
	cfg.MinConns = 0
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	_, source, _, _ := runtime.Caller(0)
	if err := db.ApplyMigrations(ctx, pool, filepath.Join(filepath.Dir(source), "..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	previous := db.DB
	db.DB = pool
	appConfig := &config.Config{Server: config.ServerConfig{Env: "test", BaseURL: "http://localhost:3000"}, JWT: config.JWTConfig{Secret: "disposable-http-suite-secret-only", Expiration: 24}, Federation: config.FederationConfig{Domain: "test.local", Enabled: false}}
	srv := httptest.NewServer(server.NewServer(appConfig).Echo())
	t.Cleanup(func() {
		srv.Close()
		db.DB = previous
		pool.Close()
		cleanupCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		admin.Exec(cleanupCtx, "DROP SCHEMA "+schema+" CASCADE")
		admin.Close()
	})
	return srv.URL
}
