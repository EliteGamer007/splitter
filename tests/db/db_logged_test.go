// Package db contains database tests.
// Unit tests must be instrumented with testlogger.
package db

import (
	"context"
	"testing"
	"time"

	"splitter/tests/testlogger"

	"github.com/jackc/pgx/v5"
)

// TestDatabaseConnectionLogged wraps the connection test with result logging.
func TestDatabaseConnectionLogged(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "db", start, testErr) }()

	connString := getConnectionString(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		testErr = err
		t.Fatalf("Expected successful connection, got error: %v", err)
	}
	defer conn.Close(ctx)

	if err = conn.Ping(ctx); err != nil {
		testErr = err
		t.Fatalf("Expected successful ping, got error: %v", err)
	}
}

// TestSchemaChecksLogged wraps the schema validation with result logging.
func TestSchemaChecksLogged(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "db", start, testErr) }()

	connString := getConnectionString(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		testErr = err
		t.Fatalf("Could not connect to database: %v", err)
	}
	defer conn.Close(ctx)

	tables := []string{"users", "posts", "replies", "stories", "story_views", "messages"}
	for _, table := range tables {
		var exists bool
		query := "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1)"
		if err := conn.QueryRow(ctx, query, table).Scan(&exists); err != nil {
			testErr = err
			t.Errorf("Failed to query for table %s: %v", table, err)
			continue
		}
		if !exists {
			t.Errorf("Table '%s' is missing", table)
		}
	}

	// Check stories table columns
	storyColumns := []string{"id", "user_id", "media_url", "created_at", "expires_at"}
	for _, col := range storyColumns {
		var exists bool
		query := "SELECT EXISTS (SELECT FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'stories' AND column_name = $1)"
		if err := conn.QueryRow(ctx, query, col).Scan(&exists); err != nil {
			testErr = err
			t.Errorf("Failed to query column stories.%s: %v", col, err)
			continue
		}
		if !exists {
			t.Errorf("Column 'stories.%s' is missing", col)
		}
	}
}

// TestDatabaseTransactionRollback validates that failed transactions roll back correctly.
func TestDatabaseTransactionRollback(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "db", start, testErr) }()

	connString := getConnectionString(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		testErr = err
		t.Fatalf("Could not connect: %v", err)
	}
	defer conn.Close(ctx)

	tx, err := conn.Begin(ctx)
	if err != nil {
		testErr = err
		t.Fatalf("Failed to begin transaction: %v", err)
	}

	// Execute a query that will intentionally fail (insert into non-existent table)
	_, errExec := tx.Exec(ctx, "SELECT 1 FROM nonexistent_table_abc123")
	if errExec == nil {
		// Query shouldn't succeed; roll back and mark fail
		_ = tx.Rollback(ctx)
		t.Fatal("Expected query to fail, but it succeeded")
	}

	// Rollback must succeed after a failed statement in a tx
	if rollErr := tx.Rollback(ctx); rollErr != nil {
		testErr = rollErr
		t.Fatalf("Rollback failed: %v", rollErr)
	}

	// Connection should still be usable after rollback
	var one int
	if err := conn.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil || one != 1 {
		testErr = err
		t.Fatalf("Connection unusable after rollback: %v", err)
	}
}
