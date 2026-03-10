package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"splitter/internal/config"
	"splitter/internal/db"
	"splitter/internal/federation"
	"splitter/internal/repository"
	"splitter/internal/server"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// DEBUG: Print DB Config to verify isolation
	log.Printf("DEBUG: Loaded Config -- DB_HOST=%s, DB_NAME=%s, FEDERATION_DOMAIN=%s",
		cfg.Database.Host, cfg.Database.Name, cfg.Federation.Domain)

	// Initialize database connection
	if err := db.InitDB(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Migrations are already applied manually to Neon
	// Skip automatic migrations to avoid "relation already exists" errors
	// if err := db.RunMigrations("migrations"); err != nil {
	// 	log.Printf("Warning: Failed to run migrations: %v", err)
	// }

	// Ensure migration 015 is applied (manual fix for Revoke Key)
	db.GetDB().Exec(context.Background(), "ALTER TABLE key_rotations ADD COLUMN IF NOT EXISTS reason TEXT NOT NULL DEFAULT 'rotated';")

	// Ensure migration 023 is applied (remote actor encryption key for E2E DMs)
	db.GetDB().Exec(context.Background(), "ALTER TABLE remote_actors ADD COLUMN IF NOT EXISTS encryption_public_key TEXT;")

	// Ensure encryption_public_key column exists on users table (may already exist from master schema)
	db.GetDB().Exec(context.Background(), "ALTER TABLE users ADD COLUMN IF NOT EXISTS encryption_public_key TEXT DEFAULT '';")

	// One-time reset: clear all broken/mismatched encryption keys so every user
	// gets a fresh key pair on next login.  The flag row prevents this from
	// running more than once.
	db.GetDB().Exec(context.Background(), "CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ DEFAULT NOW())")
	resetTag := "e2e_key_reset_v1"
	var exists bool
	err := db.GetDB().QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", resetTag).Scan(&exists)
	if err != nil || !exists {
		log.Println("[E2E] Resetting all encryption_public_key values to force fresh key generation...")
		db.GetDB().Exec(context.Background(), "UPDATE users SET encryption_public_key = '' WHERE encryption_public_key IS NOT NULL AND encryption_public_key != ''")
		db.GetDB().Exec(context.Background(), "UPDATE remote_actors SET encryption_public_key = '' WHERE encryption_public_key IS NOT NULL AND encryption_public_key != ''")
		// Record that we ran this so it never runs again
		db.GetDB().Exec(context.Background(),
			"INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING", resetTag)
		log.Println("[E2E] All encryption keys cleared. Users will get fresh keys on next login.")
	}

	// Ensure admin user exists
	ensureAdminUser(cfg) // Silent check, no logging needed

	// One-time: deduplicate remote posts and add unique index to prevent future duplicates
	dedupTag := "dedup_remote_posts_v1"
	var dedupExists bool
	db.GetDB().QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", dedupTag).Scan(&dedupExists)
	if !dedupExists {
		log.Println("[Migration] Deduplicating remote posts and adding unique index on original_post_uri...")
		// Delete duplicate remote posts keeping the earliest one per original_post_uri
		db.GetDB().Exec(context.Background(), `
			DELETE FROM posts WHERE id IN (
				SELECT id FROM (
					SELECT id, ROW_NUMBER() OVER (PARTITION BY original_post_uri ORDER BY created_at ASC) AS rn
					FROM posts
					WHERE is_remote = true AND original_post_uri IS NOT NULL AND original_post_uri != ''
				) dupes WHERE dupes.rn > 1
			)`)
		// Create unique partial index for non-null, non-empty original_post_uri
		db.GetDB().Exec(context.Background(),
			"CREATE UNIQUE INDEX IF NOT EXISTS idx_posts_original_post_uri ON posts (original_post_uri) WHERE original_post_uri IS NOT NULL AND original_post_uri != ''")
		db.GetDB().Exec(context.Background(),
			"INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING", dedupTag)
		log.Println("[Migration] Remote post dedup complete.")
	}

	// Ensure Split bot user exists
	ensureSplitBotUser(cfg)

	// Initialize and start server
	federation.ConfigureDeliveryPolicy(
		cfg.Worker.MaxRetryCount,
		cfg.Worker.CircuitFailureThreshold,
		time.Duration(cfg.Worker.CircuitCooldownSeconds)*time.Second,
	)

	srv := server.NewServer(cfg)

	// --- Start background worker loops in-process (goroutine) ---
	if cfg.Federation.Enabled {
		go runWorkerLoops(cfg)
		go federation.RunHealthCheckLoop(context.Background(), cfg.Federation.Domain)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	log.Printf("Starting server on port %s", port)
	if err := srv.Start(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// ensureAdminUser creates a domain-specific admin user (admin1 for splitter-1,
// admin2 for splitter-2) and always keeps the password and role in sync.
func ensureAdminUser(cfg *config.Config) error {
	ctx := context.Background()
	userRepo := repository.NewUserRepository()

	// Determine admin username from federation domain
	adminUsername := "admin1" // default for primary instance
	domain := cfg.Federation.Domain
	if strings.Contains(domain, "-2") || strings.HasSuffix(domain, "2") || strings.Contains(domain, ":8001") {
		adminUsername = "admin2"
	}
	adminEmail := adminUsername + "@" + cfg.Federation.Domain
	adminDID := "did:key:" + adminUsername

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("splitteradmin"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Check if this domain-specific admin already exists
	existingAdmin, _, _ := userRepo.GetByUsername(ctx, adminUsername)
	if existingAdmin != nil {
		updateQuery := `UPDATE users SET role = 'admin', password_hash = $1, instance_domain = $2 WHERE username = $3`
		db.GetDB().Exec(ctx, updateQuery, string(passwordHash), cfg.Federation.Domain, adminUsername)
		return nil
	}

	query := `
		INSERT INTO users (username, email, password_hash, instance_domain, display_name, role, did, public_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (email) DO UPDATE SET role = 'admin', password_hash = EXCLUDED.password_hash
	`
	_, err = db.GetDB().Exec(ctx, query,
		adminUsername,
		adminEmail,
		string(passwordHash),
		cfg.Federation.Domain,
		"System Admin",
		"admin",
		adminDID,
		"",
	)
	if err != nil {
		return err
	}

	log.Printf("Admin user created (username: %s, password: splitteradmin, domain: %s)", adminUsername, cfg.Federation.Domain)
	return nil
}

// ensureSplitBotUser creates the split bot user if it doesn't exist
func ensureSplitBotUser(cfg *config.Config) error {
	ctx := context.Background()
	userRepo := repository.NewUserRepository()

	existingBot, _, _ := userRepo.GetByUsername(ctx, "split")
	if existingBot != nil {
		return nil
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("splitbotpass!"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO users (username, email, password_hash, instance_domain, display_name, role, did, public_key, bio)
		VALUES ($1, $2, $3, $4, $5, 'user', $6, $7, $8)
		ON CONFLICT (did) DO NOTHING
	`

	_, err = db.GetDB().Exec(ctx, query,
		"split",
		"split@bot.local",
		string(passwordHash),
		cfg.Federation.Domain,
		"Split AI",
		"did:key:bot_split",
		"bot_key",
		"I am Split, the AI assistant! Mention @split in a post to talk to me. 🤖",
	)

	if err == nil {
		log.Println("[SplitBot] Auto-created 'split' user account.")
	}
	return err
}

// runWorkerLoops runs the federation background worker loops (retry + reputation + migration)
// inside the same process as the web server, using goroutines.
func runWorkerLoops(cfg *config.Config) {
	ctx := context.Background()

	retryInterval := time.Duration(cfg.Worker.RetryIntervalSeconds) * time.Second
	reputationInterval := time.Duration(cfg.Worker.ReputationIntervalSeconds) * time.Second

	// Clamp minimum intervals to avoid tight loops if config is 0
	if retryInterval < 10*time.Second {
		retryInterval = 30 * time.Second
	}
	if reputationInterval < 10*time.Second {
		reputationInterval = 60 * time.Second
	}

	retryTicker := time.NewTicker(retryInterval)
	reputationTicker := time.NewTicker(reputationInterval)
	migrationTicker := time.NewTicker(6 * time.Hour) // Check migration every 6 hours
	defer retryTicker.Stop()
	defer reputationTicker.Stop()
	defer migrationTicker.Stop()

	log.Printf("[InProcessWorker] Started: retry every %s, reputation every %s, migration check every 6h", retryInterval, reputationInterval)

	// Ensure migration table exists
	if err := federation.EnsureMigrationTable(ctx); err != nil {
		log.Printf("[InProcessWorker] Failed to ensure migration table: %v", err)
	}

	// Initial reputation calculation
	if err := federation.RecalculateInstanceReputation(ctx); err != nil {
		log.Printf("[InProcessWorker] Initial reputation calc failed: %v", err)
	}

	for {
		select {
		case <-retryTicker.C:
			processed, failed, err := federation.RetryOutboxBatch(ctx, 50)
			if err != nil {
				log.Printf("[InProcessWorker] Retry batch failed: %v", err)
				continue
			}
			if processed > 0 {
				log.Printf("[InProcessWorker] Retry batch processed=%d failed=%d", processed, failed)
			}
		case <-reputationTicker.C:
			if err := federation.RecalculateInstanceReputation(ctx); err != nil {
				log.Printf("[InProcessWorker] Reputation recalculation failed: %v", err)
			}
		case <-migrationTicker.C:
			federation.CheckAndMigrateUsers(ctx, cfg.Federation.Domain)
		}
	}
}
