// Package testenv provides opt-in migrated disposable PostgreSQL fixtures.
package testenv

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"splitter/internal/db"
	"splitter/tests/testdb"
)

func DatabaseURL(t *testing.T) string {
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
	pool, err := pgxpool.New(ctx, raw)
	if err != nil {
		t.Fatal("cannot initialize disposable database")
	}
	defer pool.Close()
	_, source, _, _ := runtime.Caller(0)
	if err := db.ApplyMigrations(ctx, pool, filepath.Join(filepath.Dir(source), "..", "..", "migrations")); err != nil {
		t.Fatalf("prepare disposable database: %v", err)
	}
	return raw
}
