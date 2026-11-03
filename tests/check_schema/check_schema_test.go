package main_test

import (
	"context"
	"splitter/tests/testenv"
	"testing"
	"time"

	"splitter/tests/testlogger"

	"github.com/jackc/pgx/v5/pgxpool"
)

func getSchemaConnStr(t *testing.T) string {
	t.Helper()
	return testenv.DatabaseURL(t)
}

// TestSchemaTablesExist validates that all required tables exist.
func TestSchemaTablesExist(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "schema", start, testErr) }()

	conn, err := pgxpool.New(context.Background(), getSchemaConnStr(t))
	if err != nil {
		testErr = err
		t.Fatalf("Cannot connect: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	tables := []string{
		"users", "posts", "replies", "follows", "interactions",
		"messages", "stories", "story_views",
	}
	for _, tbl := range tables {
		var exists bool
		err := conn.QueryRow(ctx,
			"SELECT EXISTS(SELECT FROM information_schema.tables WHERE table_schema='public' AND table_name=$1)",
			tbl,
		).Scan(&exists)
		if err != nil {
			testErr = err
			t.Errorf("Query failed for table %s: %v", tbl, err)
			continue
		}
		if !exists {
			t.Errorf("Table '%s' does not exist", tbl)
		}
	}
}

// TestSchemaColumnTypes validates critical column data types.
func TestSchemaColumnTypes(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "schema", start, testErr) }()

	conn, err := pgxpool.New(context.Background(), getSchemaConnStr(t))
	if err != nil {
		testErr = err
		t.Fatalf("Cannot connect: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()

	checks := []struct {
		table, column, wantType string
	}{
		{"users", "id", "uuid"},
		{"posts", "id", "uuid"},
		{"stories", "id", "uuid"},
		{"messages", "deleted_at", "timestamp with time zone"},
		{"messages", "edited_at", "timestamp with time zone"},
	}

	for _, c := range checks {
		var dataType string
		err := conn.QueryRow(ctx,
			"SELECT data_type FROM information_schema.columns WHERE table_schema='public' AND table_name=$1 AND column_name=$2",
			c.table, c.column,
		).Scan(&dataType)
		if err != nil {
			testErr = err
			t.Errorf("Failed to query %s.%s: %v", c.table, c.column, err)
			continue
		}
		if dataType != c.wantType {
			t.Errorf("%s.%s: expected type '%s', got '%s'", c.table, c.column, c.wantType, dataType)
		}
	}
}

// TestSchemaIndexesExist verifies that performance-critical indexes are present.
func TestSchemaIndexesExist(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "schema", start, testErr) }()

	conn, err := pgxpool.New(context.Background(), getSchemaConnStr(t))
	if err != nil {
		testErr = err
		t.Fatalf("Cannot connect: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()

	indexes := []string{
		"idx_messages_deleted_at",
	}
	for _, idx := range indexes {
		var exists bool
		err := conn.QueryRow(ctx,
			"SELECT EXISTS(SELECT FROM pg_indexes WHERE schemaname='public' AND indexname=$1)",
			idx,
		).Scan(&exists)
		if err != nil {
			testErr = err
			t.Errorf("Query failed for index %s: %v", idx, err)
			continue
		}
		if !exists {
			t.Errorf("Index '%s' does not exist", idx)
		}
	}
}
