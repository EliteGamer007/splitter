package main

import (
	"context"
	"log"

	"splitter/internal/config"
	"splitter/internal/db"
)

func main() {
	cfg := config.Load()
	if err := db.InitDB(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	log.Println("Adding deleted_at and edited_at columns...")

	// Add columns
	queries := []string{
		`ALTER TABLE messages ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ DEFAULT NULL;`,
		`ALTER TABLE messages ADD COLUMN IF NOT EXISTS edited_at TIMESTAMPTZ DEFAULT NULL;`,
		`CREATE INDEX IF NOT EXISTS idx_messages_deleted_at ON messages(deleted_at) WHERE deleted_at IS NOT NULL;`,
	}

	for _, query := range queries {
		_, err := db.GetDB().Exec(ctx, query)
		if err != nil {
			log.Printf("Warning: %v", err)
		} else {
			log.Printf("✅ Executed: %s", query)
		}
	}

	log.Println("\n✅ Migration complete!")
}
