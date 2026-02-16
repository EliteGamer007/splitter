package main

import (
	"context"
	"log"

	"splitter/internal/config"
	"splitter/internal/db"
)

func main() {
	log.Println("🔍 Checking Users and Encryption Keys in Database...")

	// Load config and connect to DB
	cfg := config.Load()
	if err := db.InitDB(cfg); err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Get detailed user information
	log.Println("\n📋 Listing ALL users with encryption key details:")
	log.Println("═══════════════════════════════════════════════════════════════════════════════")

	rows, err := db.GetDB().Query(ctx, `
		SELECT 
			id, 
			username, 
			email, 
			role,
			CASE 
				WHEN encryption_public_key IS NULL THEN 'NULL'
				WHEN encryption_public_key = '' THEN 'EMPTY STRING'
				ELSE 'HAS KEY (' || LENGTH(encryption_public_key) || ' chars)'
			END as key_status,
			COALESCE(SUBSTRING(encryption_public_key, 1, 50), 'N/A') as key_preview,
			created_at
		FROM users 
		ORDER BY created_at DESC
	`)
	if err != nil {
		log.Fatalf("❌ Failed to query users: %v", err)
	}
	defer rows.Close()

	userCount := 0
	usersWithKeys := 0
	usersWithoutKeys := 0

	for rows.Next() {
		var id, username, email, role, keyStatus, keyPreview string
		var createdAt interface{}
		rows.Scan(&id, &username, &email, &role, &keyStatus, &keyPreview, &createdAt)

		status := "❌"
		if keyStatus != "NULL" && keyStatus != "EMPTY STRING" {
			status = "✅"
			usersWithKeys++
		} else {
			usersWithoutKeys++
		}

		log.Printf("%s User: %-20s | Role: %-10s | %s", status, username, role, keyStatus)
		if keyPreview != "N/A" {
			log.Printf("   └─ Key preview: %s...", keyPreview)
		}
		log.Println()

		userCount++
	}

	log.Println("═══════════════════════════════════════════════════════════════════════════════")
	log.Printf("📊 Summary:")
	log.Printf("   Total users: %d", userCount)
	log.Printf("   ✅ Users WITH encryption keys: %d", usersWithKeys)
	log.Printf("   ❌ Users WITHOUT encryption keys: %d", usersWithoutKeys)
	log.Println("═══════════════════════════════════════════════════════════════════════════════")

	if usersWithoutKeys > 0 {
		log.Println("\n⚠️  WARNING: Some users are missing encryption keys!")
		log.Println("   This means:")
		log.Println("   1. The signup process is NOT generating/sending encryption keys")
		log.Println("   2. OR the backend is NOT storing them properly")
		log.Println("   3. OR old users created before the E2E implementation still exist")
	}

	if userCount == 0 {
		log.Println("\n⚠️  No users found in database!")
	}

	// Check column type
	log.Println("\n📋 Verifying column types:")
	var dataType string
	err = db.GetDB().QueryRow(ctx, `
		SELECT data_type 
		FROM information_schema.columns 
		WHERE table_name = 'users' AND column_name = 'encryption_public_key'
	`).Scan(&dataType)
	if err == nil {
		log.Printf("   ✅ encryption_public_key column type: %s", dataType)
	}

	err = db.GetDB().QueryRow(ctx, `
		SELECT data_type 
		FROM information_schema.columns 
		WHERE table_name = 'messages' AND column_name = 'ciphertext'
	`).Scan(&dataType)
	if err == nil {
		log.Printf("   ✅ ciphertext column type: %s", dataType)
	}

	log.Println("\n✅ Diagnostic complete!")
}
