package integration

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"splitter/internal/config"
	"splitter/internal/db"
	"splitter/internal/server"
	"splitter/tests/testdb"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	TestServer *httptest.Server
	SchemaName string
)

// SetupTestEnv requires explicit disposable PostgreSQL credentials. It never
// loads application .env files or falls back to the configured application DB.
func SetupTestEnv(t *testing.T) func() {
	t.Helper()
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("set TEST_DATABASE_URL to a loopback splitter_repair* database to run integration tests")
	}
	if _, err := testdb.ParseLocalURL(raw); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	poolConfig, err := pgxpool.ParseConfig(raw)
	if err != nil {
		t.Fatal("invalid disposable database connection configuration")
	}
	poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	poolConfig.MaxConns = 4
	poolConfig.MinConns = 0
	adminPool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		t.Fatal("cannot initialize disposable database pool")
	}
	SchemaName = fmt.Sprintf("test_schema_%d_%d", os.Getpid(), time.Now().UnixNano())
	schema := SchemaName
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+quotedSchema); err != nil {
		adminPool.Close()
		t.Fatalf("create isolated test schema: %v", err)
	}
	cleanup := func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if _, err := adminPool.Exec(cleanupCtx, "DROP SCHEMA IF EXISTS "+quotedSchema+" CASCADE"); err != nil {
			t.Errorf("clean isolated test schema: %v", err)
		}
		adminPool.Close()
	}
	success := false
	defer func() {
		if !success {
			cleanup()
		}
	}()
	testConfig := poolConfig.Copy()
	testConfig.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	testPool, err := pgxpool.NewWithConfig(ctx, testConfig)
	if err != nil {
		t.Fatal("cannot initialize isolated test pool")
	}
	t.Cleanup(testPool.Close)
	if err := db.ApplyMigrations(ctx, testPool, "../../migrations"); err != nil {
		t.Fatalf("prepare disposable schema: %v", err)
	}

	previous := db.DB
	db.DB = testPool
	cfg := &config.Config{Server: config.ServerConfig{Env: "test", BaseURL: "http://localhost:3000"}, JWT: config.JWTConfig{Secret: "disposable-integration-secret-only", Expiration: 24}, Federation: config.FederationConfig{Domain: "test.local", URL: "http://127.0.0.1", Enabled: false}}
	TestServer = httptest.NewServer(server.NewServer(cfg).Echo())
	srv := TestServer
	success = true
	return func() { srv.Close(); testPool.Close(); db.DB = previous; cleanup() }
}
