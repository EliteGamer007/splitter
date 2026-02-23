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
	log.Println("Starting database migration...")

	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println(" Warning: .env file not found, relying on environment variables")
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
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS email TEXT;`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT;`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS public_key TEXT;`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS encryption_public_key TEXT DEFAULT '';`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS message_privacy TEXT DEFAULT 'everyone';`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS default_visibility TEXT DEFAULT 'public';`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_data BYTEA;`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_media_type TEXT;`,
		`ALTER TABLE media ADD COLUMN IF NOT EXISTS media_data BYTEA;`,
		`ALTER TABLE messages ADD COLUMN IF NOT EXISTS ciphertext TEXT;`,
		`ALTER TABLE outbox_activities ADD COLUMN IF NOT EXISTS next_retry_at TIMESTAMPTZ;`,
		`ALTER TABLE outbox_activities ADD COLUMN IF NOT EXISTS last_attempt_at TIMESTAMPTZ;`,
		`ALTER TABLE outbox_activities ADD COLUMN IF NOT EXISTS last_error TEXT;`,
		`UPDATE outbox_activities SET next_retry_at = COALESCE(next_retry_at, now()) WHERE status IN ('pending','failed');`,
		`CREATE TABLE IF NOT EXISTS federation_connections (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			source_domain TEXT NOT NULL,
			target_domain TEXT NOT NULL,
			success_count INT DEFAULT 0,
			failure_count INT DEFAULT 0,
			last_status TEXT CHECK (last_status IN ('sent', 'failed', 'pending')),
			last_seen TIMESTAMPTZ DEFAULT now(),
			created_at TIMESTAMPTZ DEFAULT now(),
			updated_at TIMESTAMPTZ DEFAULT now(),
			UNIQUE(source_domain, target_domain)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_outbox_next_retry ON outbox_activities(next_retry_at) WHERE status IN ('pending','failed');`,
		`CREATE INDEX IF NOT EXISTS idx_federation_failures_circuit_until ON federation_failures(circuit_open_until);`,
		`CREATE INDEX IF NOT EXISTS idx_federation_connections_source ON federation_connections(source_domain);`,
		`CREATE INDEX IF NOT EXISTS idx_federation_connections_target ON federation_connections(target_domain);`,
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
