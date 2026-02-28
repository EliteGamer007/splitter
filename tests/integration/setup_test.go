package integration

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"splitter/internal/config"
	"splitter/internal/db"
	"splitter/internal/server"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var (
	TestServer *httptest.Server
	SchemaName string // Exported for cleanup
)

func TestMain(m *testing.M) {
	// Load .env from root
	// We are in tests/integration, so root is ../../
	// Load .env from root
	// We are in tests/integration, so root is ../../
	if err := godotenv.Load("../../.env"); err != nil {
		fmt.Printf("Error loading .env: %v\n", err)
	} else {
		fmt.Println(".env loaded successfully")
	}

	// Set ENV to test
	os.Setenv("ENV", "test")

	code := m.Run()
	os.Exit(code)
}

// SetupTestEnv initializes the test environment
// It returns a cleanup function that should be deferred
func SetupTestEnv(t *testing.T) func() {
	// Load Config
	cfg := config.Load()

	// Generate unique schema name
	SchemaName = fmt.Sprintf("test_schema_%d", os.Getpid())

	// 1. Connect to DB to create Schema (using default public schema initially)
	// We need to parse the config to get connection string
	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
	t.Logf("Connecting with: %s", connString)

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	adminPool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}

	// Drop if exists (cleanup from previous fail) and Create Schema
	_, err = adminPool.Exec(context.Background(), fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", SchemaName))
	if err != nil {
		t.Logf("Warning dropping schema: %v", err)
	}
	_, err = adminPool.Exec(context.Background(), fmt.Sprintf("CREATE SCHEMA %s", SchemaName))
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// 2. Run Migrations on this schema
	// We read the SQL files and execute them
	// search_path is critical here
	migrationSQL := fmt.Sprintf("SET search_path TO %s, public;", SchemaName)
	migrationSQL += `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`

	files := []string{
		"../../migrations/001_initial_schema.sql",
		"../../migrations/002_upgrade_to_current.sql",
		"../../migrations/014_add_key_rotations.sql",
		"../../migrations/015_revocation_reason.sql",
		"../../migrations/016_add_offline_message_sync.sql",
		"../../migrations/017_add_multi_device_and_federated_dm_encryption.sql",
	}

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("Failed to read migration file %s: %v", f, err)
		}
		sContent := string(content)
		migrationSQL += sContent
	}

	// execute all migrations in one go or separate transactions?
	// one go is fine for setup
	_, err = adminPool.Exec(context.Background(), migrationSQL)
	if err != nil {
		fmt.Printf("Migration Error: %v\n", err) // Force stdout print
		t.Fatalf("Failed to run migrations: %v", err)
	}
	adminPool.Close()

	// 3. Initialize App DB with specific search_path
	// This ensures the application uses the test schema by default
	testConnString := fmt.Sprintf("%s&search_path=%s,public", connString, SchemaName)

	// HACK: We need to set the global DB pool in internal/db package
	// But InitDB takes config. We can modify InitDB or manually set db.DB
	// Let's manually set it to ensure isolation

	testPoolConfig, _ := pgxpool.ParseConfig(testConnString)
	testPool, err := pgxpool.NewWithConfig(context.Background(), testPoolConfig)
	if err != nil {
		t.Fatalf("Failed to connect to test schema: %v", err)
	}

	// Override the global DB variable
	// Note: db.DB field must be exported (capitalized) in internal/db/postgres.go
	// Checking previous view_file... yes, "var DB *pgxpool.Pool" is exported!
	db.DB = testPool

	// 4. Start Server
	// Provide the config
	srv := server.NewServer(cfg)
	TestServer = httptest.NewServer(srv.Echo())

	// Return Cleanup Function
	return func() {
		TestServer.Close()
		db.DB.Close() // Close test pool

		// Clean schema using admin connection
		cleanupPool, _ := pgxpool.New(context.Background(), connString)
		_, _ = cleanupPool.Exec(context.Background(), fmt.Sprintf("DROP SCHEMA %s CASCADE", SchemaName))
		cleanupPool.Close()
	}
}
