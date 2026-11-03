package main_test

import (
	"context"
	"splitter/tests/testenv"
	"testing"
	"time"

	"splitter/tests/testlogger"

	"github.com/jackc/pgx/v5/pgxpool"
)

func getConnStr(t *testing.T) string {
	t.Helper()
	return testenv.DatabaseURL(t)
}

// TestMigrationColumnAddition validates that the migration added required columns.
func TestMigrationColumnAddition(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "migration", start, testErr) }()

	conn, err := pgxpool.New(context.Background(), getConnStr(t))
	if err != nil {
		testErr = err
		t.Fatalf("Cannot connect to DB: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()

	// Verify messages table has deleted_at and edited_at (from migration 018)
	checkColumns := []struct{ table, column string }{
		{"messages", "deleted_at"},
		{"messages", "edited_at"},
	}
	for _, c := range checkColumns {
		var exists bool
		err := conn.QueryRow(ctx,
			"SELECT EXISTS(SELECT FROM information_schema.columns WHERE table_schema='public' AND table_name=$1 AND column_name=$2)",
			c.table, c.column,
		).Scan(&exists)
		if err != nil {
			testErr = err
			t.Errorf("Column query failed for %s.%s: %v", c.table, c.column, err)
			continue
		}
		if !exists {
			t.Errorf("Column %s.%s is missing — migration 018 may not have applied", c.table, c.column)
		}
	}
}

// TestMigrationStoriesTable checks that the stories migration applied correctly.
func TestMigrationStoriesTable(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "migration", start, testErr) }()

	conn, err := pgxpool.New(context.Background(), getConnStr(t))
	if err != nil {
		testErr = err
		t.Fatalf("Cannot connect to DB: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()

	tables := []string{"stories", "story_views"}
	for _, tbl := range tables {
		var exists bool
		err := conn.QueryRow(ctx,
			"SELECT EXISTS(SELECT FROM information_schema.tables WHERE table_schema='public' AND table_name=$1)",
			tbl,
		).Scan(&exists)
		if err != nil {
			testErr = err
			t.Errorf("Table query failed for %s: %v", tbl, err)
			continue
		}
		if !exists {
			t.Errorf("Table %s is missing — migration 019/020 may not have applied", tbl)
		}
	}
}

// TestMigrationIdempotent verifies that running IF NOT EXISTS migrations twice doesn't error.
func TestMigrationIdempotent(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "migration", start, testErr) }()

	conn, err := pgxpool.New(context.Background(), getConnStr(t))
	if err != nil {
		testErr = err
		t.Fatalf("Cannot connect to DB: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()

	idempotentSQL := `ALTER TABLE messages ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ DEFAULT NULL;`
	if _, err := conn.Exec(ctx, idempotentSQL); err != nil {
		testErr = err
		t.Fatalf("Idempotent migration failed on second run: %v", err)
	}
}
