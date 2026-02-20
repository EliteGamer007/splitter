package main

import (
	"context"
	"log"
	"os"

	"splitter/internal/config"
	"splitter/internal/db"
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

	// Ensure admin user exists
	ensureAdminUser() // Silent check, no logging needed

	// Initialize and start server
	srv := server.NewServer(cfg)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	log.Printf("Starting server on port %s", port)
	if err := srv.Start(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// ensureAdminUser creates the admin user if it doesn't exist
func ensureAdminUser() error {
	ctx := context.Background()
	userRepo := repository.NewUserRepository()

	// Check if admin already exists
	existingAdmin, _, _ := userRepo.GetByUsername(ctx, "admin")
	if existingAdmin != nil {
		// Silently ensure admin role is set
		updateQuery := `UPDATE users SET role = 'admin' WHERE username = 'admin'`
		db.GetDB().Exec(ctx, updateQuery)
		return nil
	}

	// Create admin user with password "splitteradmin"
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("splitteradmin"), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO users (username, email, password_hash, instance_domain, display_name, role, did, public_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (email) DO UPDATE SET role = 'admin'
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
