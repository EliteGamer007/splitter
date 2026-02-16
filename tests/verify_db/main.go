package main

import (
	"context"
	"log"

	"splitter/internal/config"
	"splitter/internal/db"
)

func main() {
	log.Println("🔍 Verifying Database Schema and Users...")

	// Load config and connect to DB
	cfg := config.Load()
	if err := db.InitDB(cfg); err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Check column types
	log.Println("\n📋 Step 1: Verifying column types...")
	rows, err := db.GetDB().Query(ctx, `
		SELECT table_name, column_name, data_type 
		FROM information_schema.columns 
		WHERE table_name IN ('users', 'messages') 
		  AND column_name IN ('encryption_public_key', 'ciphertext')
		ORDER BY table_name, column_name
	`)
	if err != nil {
		log.Fatalf("❌ Failed to query column types: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, columnName, dataType string
		rows.Scan(&tableName, &columnName, &dataType)
		log.Printf("  ✅ %s.%s: %s", tableName, columnName, dataType)
	}

	// List all users with their encryption keys
	log.Println("\n📋 Step 2: Listing all users...")
	userRows, err := db.GetDB().Query(ctx, `
		SELECT id, username, email, role, 
		       CASE WHEN encryption_public_key IS NULL OR encryption_public_key = '' THEN false ELSE true END as has_key,
		       created_at
		FROM users 
		ORDER BY created_at DESC
		LIMIT 10
	`)
	if err != nil {
		log.Fatalf("❌ Failed to query users: %v", err)
	}
	defer userRows.Close()

	userCount := 0
	for userRows.Next() {
		var id, username, email, role string
		var hasKey bool
		var createdAt interface{}
		userRows.Scan(&id, &username, &email, &role, &hasKey, &createdAt)
		keyStatus := "❌ NO KEY"
		if hasKey {
			keyStatus = "✅ HAS KEY"
		}
		log.Printf("  %s - User: %-20s | Role: %-10s | %s", id, username, role, keyStatus)
		userCount++
	}

	log.Printf("\n✅ Found %d users in database", userCount)

	if userCount == 0 {
		log.Println("\n⚠️  No users found! You need to create an admin user.")
		log.Println("   You can either:")
		log.Println("   1. Sign up via the frontend (will have encryption keys)")
		log.Println("   2. Run the seeder script: go run cmd/seeder/main.go")
	}

	log.Println("\n✅ Verification complete!")
}
