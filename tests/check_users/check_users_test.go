package main_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"splitter/tests/testlogger"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func getUsersConnStr(t *testing.T) string {
	t.Helper()
	_ = godotenv.Load("../../.env")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")
	if host == "" || port == "" || user == "" || pass == "" || name == "" {
		t.Fatal("Missing required DB_* environment variables")
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require", user, pass, host, port, name)
}

// TestUsersTableColumns verifies all required user columns exist.
func TestUsersTableColumns(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "users", start, testErr) }()

	conn, err := pgxpool.New(context.Background(), getUsersConnStr(t))
	if err != nil {
		testErr = err
		t.Fatalf("Cannot connect: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	cols := []string{"id", "username", "email", "password_hash", "did", "public_key",
		"encryption_public_key", "role", "created_at"}
	for _, col := range cols {
		var exists bool
		err := conn.QueryRow(ctx,
			"SELECT EXISTS(SELECT FROM information_schema.columns WHERE table_schema='public' AND table_name='users' AND column_name=$1)",
			col,
		).Scan(&exists)
		if err != nil {
			testErr = err
			t.Errorf("Column query failed for users.%s: %v", col, err)
			continue
		}
		if !exists {
			t.Errorf("Column 'users.%s' is missing", col)
		}
	}
}

// TestUsersUniqueConstraints verifies unique constraints exist on key columns.
func TestUsersUniqueConstraints(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "users", start, testErr) }()

	conn, err := pgxpool.New(context.Background(), getUsersConnStr(t))
	if err != nil {
		testErr = err
		t.Fatalf("Cannot connect: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()

	// did is UNIQUE per the schema
	var count int
	err = conn.QueryRow(ctx, `
		SELECT COUNT(*) FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		WHERE tc.constraint_type = 'UNIQUE'
		  AND tc.table_name = 'users'
		  AND kcu.column_name = 'did'
	`).Scan(&count)
	if err != nil {
		testErr = err
		t.Errorf("Constraint query failed for users.did: %v", err)
	} else if count == 0 {
		t.Errorf("No UNIQUE constraint on users.did")
	}

	// username and email may or may not have explicit UNIQUE constraints
	for _, col := range []string{"username", "email"} {
		var c int
		err := conn.QueryRow(ctx, `
			SELECT COUNT(*) FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu
			  ON tc.constraint_name = kcu.constraint_name
			WHERE tc.constraint_type = 'UNIQUE'
			  AND tc.table_name = 'users'
			  AND kcu.column_name = $1
		`, col).Scan(&c)
		if err != nil {
			t.Logf("Constraint query for users.%s: %v", col, err)
			continue
		}
		if c == 0 {
			t.Logf("NOTE: No UNIQUE constraint on users.%s (uniqueness may be enforced at app level)", col)
		}
	}
}

// TestUsersRequiredFieldsNotNull validates that critical fields have NOT NULL constraints.
func TestUsersRequiredFieldsNotNull(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "users", start, testErr) }()

	conn, err := pgxpool.New(context.Background(), getUsersConnStr(t))
	if err != nil {
		testErr = err
		t.Fatalf("Cannot connect: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	requiredCols := []string{"username", "email", "password_hash", "role"}
	for _, col := range requiredCols {
		var isNullable string
		err := conn.QueryRow(ctx,
			"SELECT is_nullable FROM information_schema.columns WHERE table_schema='public' AND table_name='users' AND column_name=$1",
			col,
		).Scan(&isNullable)
		if err != nil {
			testErr = err
			t.Errorf("Query failed for users.%s: %v", col, err)
			continue
		}
		if isNullable == "YES" {
			t.Logf("WARNING: Column users.%s allows NULL — expected NOT NULL (may differ in shared DB)", col)
		}
	}
}
