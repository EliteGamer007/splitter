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

	// Initialize database connection
	if err := db.InitDB(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.RunMigrations("migrations"); err != nil {
		log.Printf("Warning: Failed to run migrations: %v", err)
		// Don't fail fatal here, maybe migrations are already applied or dir missing in prod
	}

	// Ensure admin user exists
	if err := ensureAdminUser(); err != nil {
		log.Printf("Warning: Failed to ensure admin user: %v", err)
	}

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
		log.Println("Admin user already exists")
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
		ON CONFLICT (username) DO UPDATE SET role = 'admin'
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
