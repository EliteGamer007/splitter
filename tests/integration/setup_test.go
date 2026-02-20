package integration

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"splitter/internal/config"
	"splitter/internal/db"
	"splitter/internal/server"
	"strings"
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
		"../../migrations/004_consolidated_fixes.sql",
		"../../migrations/005_create_replies_table.sql",
	}

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("Failed to read migration file %s: %v", f, err)
		}
		sContent := string(content)
		// Patch for 004 to avoid "column already exists" errors in test environment
		if strings.Contains(f, "004_consolidated_fixes.sql") {
			// Fix 1: Scope checks to current schema
			sContent = strings.ReplaceAll(sContent, "WHERE table_name='", "WHERE table_schema = current_schema() AND table_name='")

			// Fix 2: Use IF NOT EXISTS for columns just in case
			sContent = strings.ReplaceAll(sContent, "ALTER TABLE users ADD COLUMN email", "ALTER TABLE users ADD COLUMN IF NOT EXISTS email")
			sContent = strings.ReplaceAll(sContent, "ALTER TABLE users ADD COLUMN password_hash", "ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash")
			sContent = strings.ReplaceAll(sContent, "ALTER TABLE users ADD COLUMN role", "ALTER TABLE users ADD COLUMN IF NOT EXISTS role")
		}
		migrationSQL += sContent
	}

	// Bring integration schema in sync with runtime migration additions used by
	// recently implemented auth/privacy/media features.
	migrationSQL += `
		ALTER TABLE users ADD COLUMN IF NOT EXISTS encryption_public_key TEXT DEFAULT '';
		ALTER TABLE users ADD COLUMN IF NOT EXISTS message_privacy TEXT DEFAULT 'everyone';
		ALTER TABLE users ADD COLUMN IF NOT EXISTS default_visibility TEXT DEFAULT 'public';
		ALTER TABLE media ADD COLUMN IF NOT EXISTS media_data BYTEA;
		ALTER TABLE messages ADD COLUMN IF NOT EXISTS ciphertext TEXT;
	`

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
