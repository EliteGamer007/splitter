package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // Use pgx via database/sql
	"github.com/joho/godotenv"
)

func main() {
	log.Println("⚡ Starting database migration...")

	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ Warning: .env file not found, relying on environment variables")
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	// Mask password for logging
	safeConnStr := fmt.Sprintf("postgres://%s:***@%s:%s/%s?sslmode=require",
		os.Getenv("DB_USER"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)
	log.Printf("🔌 Connecting to: %s", safeConnStr)

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("❌ Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Failed to ping database: %v", err)
	}
	log.Println("✅ Connected to database")

	// Migration Queries
	queries := []string{
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS encryption_public_key TEXT DEFAULT '';`,
		`ALTER TABLE messages ADD COLUMN IF NOT EXISTS ciphertext TEXT;`,
	}

	for _, query := range queries {
		log.Printf("▶️ Executing: %s", query)
		_, err := db.ExecContext(context.Background(), query)
		if err != nil {
			log.Fatalf("❌ Migration failed: %v", err)
		}
	}

	log.Println("✅ Migration completed successfully! 🚀")
}
