package main_test

import (
	"context"
	"splitter/tests/testenv"
	"testing"
	"time"

	"splitter/tests/testlogger"

	"github.com/jackc/pgx/v5/pgxpool"
)

func getFixDBConnStr(t *testing.T) string {
	t.Helper()
	return testenv.DatabaseURL(t)
}

// TestDeletedAtColumnExists verifies the deleted_at column is present (repair check).
func TestDeletedAtColumnExists(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "fix_db", start, testErr) }()

	conn, err := pgxpool.New(context.Background(), getFixDBConnStr(t))
	if err != nil {
		testErr = err
		t.Fatalf("Cannot connect: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	var exists bool
	err = conn.QueryRow(ctx,
		"SELECT EXISTS(SELECT FROM information_schema.columns WHERE table_schema='public' AND table_name='messages' AND column_name='deleted_at')",
	).Scan(&exists)
	if err != nil {
		testErr = err
		t.Fatalf("Query failed: %v", err)
	}
	if !exists {
		t.Error("messages.deleted_at is missing — run repair migration")
	}
}

// TestEditedAtColumnExists verifies the edited_at column is present.
func TestEditedAtColumnExists(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "fix_db", start, testErr) }()

	conn, err := pgxpool.New(context.Background(), getFixDBConnStr(t))
	if err != nil {
		testErr = err
		t.Fatalf("Cannot connect: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	var exists bool
	err = conn.QueryRow(ctx,
		"SELECT EXISTS(SELECT FROM information_schema.columns WHERE table_schema='public' AND table_name='messages' AND column_name='edited_at')",
	).Scan(&exists)
	if err != nil {
		testErr = err
		t.Fatalf("Query failed: %v", err)
	}
	if !exists {
		t.Error("messages.edited_at is missing — run repair migration")
	}
}

// TestRepairIndexExists verifies the repair index is in place.
func TestRepairIndexExists(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "fix_db", start, testErr) }()

	conn, err := pgxpool.New(context.Background(), getFixDBConnStr(t))
	if err != nil {
		testErr = err
		t.Fatalf("Cannot connect: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	var exists bool
	err = conn.QueryRow(ctx,
		"SELECT EXISTS(SELECT FROM pg_indexes WHERE schemaname='public' AND indexname='idx_messages_deleted_at')",
	).Scan(&exists)
	if err != nil {
		testErr = err
		t.Fatalf("Index query failed: %v", err)
	}
	if !exists {
		t.Error("idx_messages_deleted_at index is missing")
	}
}

// TestEncryptionPublicKeyColumnExists verifies the encryption key column is present.
func TestEncryptionPublicKeyColumnExists(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "fix_db", start, testErr) }()

	conn, err := pgxpool.New(context.Background(), getFixDBConnStr(t))
	if err != nil {
		testErr = err
		t.Fatalf("Cannot connect: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	var exists bool
	err = conn.QueryRow(ctx,
		"SELECT EXISTS(SELECT FROM information_schema.columns WHERE table_schema='public' AND table_name='users' AND column_name='encryption_public_key')",
	).Scan(&exists)
	if err != nil {
		testErr = err
		t.Fatalf("Query failed: %v", err)
	}
	if !exists {
		t.Error("users.encryption_public_key is missing — run repair migration")
	}
}
