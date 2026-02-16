package main

import (
	"context"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"time"

	"splitter/internal/auth"
	"splitter/internal/config"
	"splitter/internal/db"
	"splitter/internal/models"
	"splitter/internal/repository"
)

func main() {
	log.Println("🔧 Starting Database Fix and Admin Setup...")

	// Load config
	cfg := config.Load()
	if cfg == nil {
		log.Fatalf("❌ Failed to load config")
	}

	// Initialize database
	if err := db.InitDB(cfg); err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Step 1: Run the fix migration
	log.Println("📋 Step 1: Running database migration to fix column types...")
	migrationSQL, err := os.ReadFile("migrations/007_fix_e2ee_and_reset.sql")
	if err != nil {
		log.Fatalf("❌ Failed to read migration file: %v", err)
	}

	_, err = db.GetDB().Exec(ctx, string(migrationSQL))
	if err != nil {
		log.Fatalf("❌ Failed to run migration: %v", err)
	}
	log.Println("✅ Migration completed successfully")

	// Step 2: Verify column types
	log.Println("📋 Step 2: Verifying column types...")
	var dataType string
	err = db.GetDB().QueryRow(ctx, `
		SELECT data_type 
		FROM information_schema.columns 
		WHERE table_name = 'messages' AND column_name = 'ciphertext'
	`).Scan(&dataType)
	if err != nil {
		log.Fatalf("❌ Failed to verify ciphertext column: %v", err)
	}
	log.Printf("✅ ciphertext column type: %s", dataType)

	err = db.GetDB().QueryRow(ctx, `
		SELECT data_type 
		FROM information_schema.columns 
		WHERE table_name = 'users' AND column_name = 'encryption_public_key'
	`).Scan(&dataType)
	if err != nil {
		log.Fatalf("❌ Failed to verify encryption_public_key column: %v", err)
	}
	log.Printf("✅ encryption_public_key column type: %s", dataType)

	// Step 3: Generate keys for admin
	log.Println("📋 Step 3: Generating encryption keys for admin...")

	// Generate signing key (ECDSA P-256)
	signKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("❌ Failed to generate signing key: %v", err)
	}
	signPubKeyBytes, _ := x509.MarshalPKIXPublicKey(&signKey.PublicKey)
	signPubKeyBase64 := base64.StdEncoding.EncodeToString(signPubKeyBytes)

	// Generate encryption key (ECDH P-256)
	encKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("❌ Failed to generate encryption key: %v", err)
	}
	encPubKeyBytes, _ := x509.MarshalPKIXPublicKey(encKey.PublicKey())
	encPubKeyBase64 := base64.StdEncoding.EncodeToString(encPubKeyBytes)

	log.Println("✅ Keys generated successfully")

	// Step 4: Create admin user
	log.Println("📋 Step 4: Creating admin user...")

	adminUsername := fmt.Sprintf("admin_%d", time.Now().Unix())
	adminPassword := "admin123" // User should change this immediately
	adminEmail := fmt.Sprintf("admin_%d@localhost", time.Now().Unix())

	passwordHash, err := auth.HashPassword(adminPassword)
	if err != nil {
		log.Fatalf("❌ Failed to hash password: %v", err)
	}

	did := auth.GenerateSimpleDID(adminUsername)

	userCreate := &models.UserCreate{
		Username:            adminUsername,
		Email:               adminEmail,
		Password:            adminPassword,
		DisplayName:         "Admin",
		InstanceDomain:      "localhost",
		DID:                 did,
		PublicKey:           signPubKeyBase64,
		EncryptionPublicKey: encPubKeyBase64,
	}

	userRepo := repository.NewUserRepository()
	user, err := userRepo.Create(ctx, userCreate, passwordHash)
	if err != nil {
		log.Fatalf("❌ Failed to create admin user: %v", err)
	}

	// Step 5: Update role to admin
	_, err = db.GetDB().Exec(ctx, `UPDATE users SET role = 'admin' WHERE id = $1`, user.ID)
	if err != nil {
		log.Fatalf("❌ Failed to update admin role: %v", err)
	}

	log.Println("✅ Admin user created successfully")
	log.Println("")
	log.Println("═══════════════════════════════════════════════")
	log.Println("  📋 ADMIN LOGIN CREDENTIALS")
	log.Println("═══════════════════════════════════════════════")
	log.Printf("  Username: %s\n", adminUsername)
	log.Printf("  Password: %s\n", adminPassword)
	log.Printf("  User ID:  %s\n", user.ID)
	log.Println("═══════════════════════════════════════════════")
	log.Println("  ⚠️  IMPORTANT: Change the admin password immediately!")
	log.Println("═══════════════════════════════════════════════")
	log.Println("")
	log.Println("✅ Database fix and admin setup completed successfully!")
	log.Println("")
	log.Println("Next steps:")
	log.Println("1. Restart the backend server")
	log.Println("2. Login with the admin credentials above")
	log.Println("3. Create new test users (all new users will have encryption keys)")
	log.Println("4. Test E2E messaging between users")
}
