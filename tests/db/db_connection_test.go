package db

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

// getConnectionString loads environment variables and returns a proper connection string
func getConnectionString(t *testing.T) string {
	// Try loading .env from root if available.
	// We are in tests/db, so root is ../../
	_ = godotenv.Load("../../.env")

	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	if host == "" || port == "" || user == "" || password == "" || dbname == "" {
		t.Fatal("Missing required environment variables (DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME)")
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require", user, password, host, port, dbname)
}

// TestDatabaseConnection verifies that the application can connect to the database
// using the configuration provided in environment variables.
// Why: This is critical for all backend functionality to ensure connectivity.
// Expected Outcome: A connection is established and a ping is successful.
func TestDatabaseConnection(t *testing.T) {
	connString := getConnectionString(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		t.Fatalf("Expected successful connection, got error: %v", err)
	}
	defer conn.Close(ctx)

	err = conn.Ping(ctx)
	if err != nil {
		t.Fatalf("Expected successful ping, got error: %v", err)
	}
}

/*
TEST RESULT SUMMARY:
- Passed: Database connection established successfully via pgx
- Passed: Database ping successful
- Failed: None
- Limitations: Relies on external Neon DB connectivity and valid environment variables
*/
