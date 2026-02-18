package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env.instance2
	if err := godotenv.Load(".env.instance2"); err != nil {
		log.Fatalf("Error loading .env.instance2: %v", err)
	}

	// Build connection string
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"))

	fmt.Printf("Connecting to database: %s...\n", os.Getenv("DB_NAME"))

	// Connect
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	// Test connection
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("✓ Connected successfully")

	// Create instance_keys table
	migrationSQL := `
CREATE TABLE IF NOT EXISTS instance_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain TEXT UNIQUE NOT NULL,
    public_key_pem TEXT NOT NULL,
    private_key_pem TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);
`

	fmt.Println("Creating instance_keys table...")
	_, err = pool.Exec(context.Background(), migrationSQL)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	fmt.Println("✓ Migration completed successfully!")
}
