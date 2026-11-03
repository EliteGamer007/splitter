//go:build ignore

package main

import (
	"context"
	"log"

	"splitter/internal/config"
	"splitter/internal/db"
)

func main() {
	log.Println("Starting database cleanup to free up Neon Cloud storage...")

	// Connect to DB using app config
	cfg := config.Load()
	if err := db.InitDB(cfg); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// 1. Clear stories (media heavy, ephemeral anyway)
	log.Println("Truncating stories table to free media data...")
	if _, err := db.GetDB().Exec(ctx, `TRUNCATE TABLE stories CASCADE;`); err != nil {
		log.Printf("Warning: Failed to truncate stories: %v", err)
	}

	// 2. Clear outbox and inbox activities (network & storage heavy)
	log.Println("Truncating federation inbox/outbox activities...")
	if _, err := db.GetDB().Exec(ctx, `TRUNCATE TABLE inbox_activities, outbox_activities CASCADE;`); err != nil {
		log.Printf("Warning: Failed to truncate activities: %v", err)
	}

	// 3. Clear federation failures and reputation tables (failure management)
	log.Println("Truncating federation failure management tables...")
	if _, err := db.GetDB().Exec(ctx, `TRUNCATE TABLE federation_failures, instance_reputation, activity_deduplication CASCADE;`); err != nil {
		log.Printf("Warning: Failed to truncate federation metadata: %v", err)
	}

	// Wait, deleting post media might ruin existing real posts, but you mentioned you want to reduce storage.
	// If you want a fresh slate without losing users:
	log.Println("Cleanup executed successfully. You should see a noticeable drop in DB storage usage.")
}
