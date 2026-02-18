package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"splitter/internal/config"
	"splitter/internal/db"

	"github.com/joho/godotenv"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run scripts/apply_federation_migration_go.go <instance>")
		fmt.Println("  instance: 1 or 2")
		os.Exit(1)
	}

	instance := os.Args[1]
	var envFile string
	var instanceName string

	switch instance {
	case "1":
		envFile = ".env"
		instanceName = "Instance 1 (splitter-1)"
	case "2":
		envFile = ".env.instance2"
		instanceName = "Instance 2 (splitter-2)"
	case "both":
		// Apply to both instances
		applyToInstance(".env", "Instance 1 (splitter-1)")
		applyToInstance(".env.instance2", "Instance 2 (splitter-2)")
		fmt.Println("\n✓ Migration applied to both instances successfully!")
		return
	default:
		log.Fatalf("Invalid instance: %s (use 1, 2, or both)", instance)
	}

	applyToInstance(envFile, instanceName)
	fmt.Printf("\n✓ Migration applied to %s successfully!\n", instanceName)
}

func applyToInstance(envFile, instanceName string) {
	// Clear all environment variables to avoid carry-over
	os.Clearenv()

	// Load environment
	if err := godotenv.Overload(envFile); err != nil {
		log.Fatalf("Error loading %s: %v", envFile, err)
	}

	cfg := config.Load()
	fmt.Printf("\nApplying migration to %s...\n", instanceName)
	fmt.Printf("Database: %s on %s\n", cfg.Database.Name, cfg.Database.Host)

	// Initialize database
	if err := db.InitDB(cfg); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	conn := db.GetDB()

	// Read migration file
	migrationSQL := `
-- Migration 010: Federation Fix - Create instance_keys table

-- Instance RSA keypairs for HTTP Signature signing
CREATE TABLE IF NOT EXISTS instance_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    domain TEXT UNIQUE NOT NULL,
    public_key_pem TEXT NOT NULL,
    private_key_pem TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);
`

	// Execute migration
	_, err := conn.Exec(ctx, migrationSQL)
	if err != nil {
		log.Fatalf("Failed to execute migration: %v", err)
	}

	// Ensure users table has instance_domain column
	alterUsersSQL := `
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='users' AND column_name='instance_domain') THEN
        ALTER TABLE users ADD COLUMN instance_domain TEXT DEFAULT 'localhost';
    END IF;
END $$;

-- Update instance_domain for existing local users if null  
UPDATE users SET instance_domain = COALESCE(instance_domain, 'localhost') WHERE instance_domain IS NULL OR instance_domain = '';
`
	_, err = conn.Exec(ctx, alterUsersSQL)
	if err != nil {
		log.Printf("Warning: Failed to alter users table: %v", err)
	}

	// Ensure posts table has federation columns
	alterPostsSQL := `
DO $$ 
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='posts' AND column_name='is_remote') THEN
        ALTER TABLE posts ADD COLUMN is_remote BOOLEAN DEFAULT FALSE;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='posts' AND column_name='original_post_uri') THEN
        ALTER TABLE posts ADD COLUMN original_post_uri TEXT;
    END IF;
END $$;
`
	_, err = conn.Exec(ctx, alterPostsSQL)
	if err != nil {
		log.Printf("Warning: Failed to alter posts table: %v", err)
	}

	fmt.Printf("✓ Migration completed for %s\n", instanceName)
}
