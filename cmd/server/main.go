package main

import (
	"context"
	"log"
	"os"
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

	// Ensure admin user exists
	ensureAdminUser() // Silent check, no logging needed

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

// ensureAdminUser creates the admin user if it doesn't exist, and always keeps
// the password and role in sync so login works after a fresh deploy.
func ensureAdminUser() error {
	ctx := context.Background()
	userRepo := repository.NewUserRepository()

	// Always regenerate hash so the known password is correct on every startup
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("splitteradmin"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// Check if admin already exists
	existingAdmin, _, _ := userRepo.GetByUsername(ctx, "admin")
	if existingAdmin != nil {
		// Ensure admin role and password are both correct
		updateQuery := `UPDATE users SET role = 'admin', password_hash = $1 WHERE username = 'admin'`
		db.GetDB().Exec(ctx, updateQuery, string(passwordHash))
		return nil
	}

	query := `
		INSERT INTO users (username, email, password_hash, instance_domain, display_name, role, did, public_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (email) DO UPDATE SET role = 'admin', password_hash = EXCLUDED.password_hash
	`

	_, err = db.GetDB().Exec(ctx, query,
		"admin",
		"admin@localhost",
		string(passwordHash),
		"localhost",
		"System Admin",
		"admin",
		"did:key:admin",
		"",
	)

	if err != nil {
		return err
	}

	log.Println("Admin user created successfully (username: admin, password: splitteradmin)")
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

// runWorkerLoops runs the federation background worker loops (retry + reputation)
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
	defer retryTicker.Stop()
	defer reputationTicker.Stop()

	log.Printf("[InProcessWorker] Started: retry every %s, reputation every %s", retryInterval, reputationInterval)

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
		}
	}
}
