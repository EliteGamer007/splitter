package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func migrationTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("requires explicit disposable TEST_DATABASE_URL")
	}
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() != "127.0.0.1" || !strings.HasPrefix(u.Path, "/splitter_repair_") {
		t.Fatal("refusing non-disposable database")
	}
	ctx := context.Background()
	base, err := pgxpool.New(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("migration_test_%d", time.Now().UnixNano())
	if _, err = base.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	old := DB
	DB = pool
	t.Cleanup(func() { DB = old; pool.Close(); base.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); base.Close() })
	return pool
}

func TestMigrationsFreshAndRepeated(t *testing.T) {
	p := migrationTestPool(t)
	ctx := context.Background()
	dir := filepath.Join("..", "..", "migrations")
	if err := RunMigrations(dir); err != nil {
		t.Fatalf("fresh install failed: %v", err)
	}
	if err := RunMigrations(dir); err != nil {
		t.Fatalf("repeat must be idempotent: %v", err)
	}
	for _, table := range []string{"users", "posts", "stories", "story_views", "circle_members", "user_device_keys", "appeals"} {
		var exists bool
		if err := p.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", table).Scan(&exists); err != nil || !exists {
			t.Fatalf("missing %s: %v", table, err)
		}
	}
}

func TestFailedMigrationRollsBackAndIsNotRecorded(t *testing.T) {
	p := migrationTestPool(t)
	ctx := context.Background()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "003_failure.sql"), []byte("CREATE TABLE should_rollback(id int); SELECT definitely_missing_function();"), 0600)
	if err := RunMigrations(dir); err == nil {
		t.Fatal("broken migration reported success")
	}
	var exists bool
	if err := p.QueryRow(ctx, "SELECT to_regclass('should_rollback') IS NOT NULL").Scan(&exists); err != nil || exists {
		t.Fatalf("partial migration retained: %v", err)
	}
	var count int
	if err := p.QueryRow(ctx, "SELECT count(*) FROM schema_migrations WHERE version='003_failure.sql'").Scan(&count); err != nil || count != 0 {
		t.Fatalf("failed migration recorded (or no ledger): %v", err)
	}
}

func TestMigrationsDoNotBorrowPublicSchemaState(t *testing.T) {
	p := migrationTestPool(t)
	ctx := context.Background()
	base, err := pgxpool.New(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer base.Close()
	if err = ApplyMigrations(ctx, base, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	if err = RunMigrations(filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	var own bool
	if err = p.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema=current_schema() AND table_name='users')").Scan(&own); err != nil || !own {
		t.Fatalf("new schema borrowed public users/migration ledger: %v", err)
	}
}
