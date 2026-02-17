package main

import (
	"context"
	"fmt"
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

	// Check all users and their encryption keys
	query := `
		SELECT id, username, email, 
		       COALESCE(encryption_public_key, 'NULL') as encryption_key,
		       LENGTH(COALESCE(encryption_public_key, '')) as key_length
		FROM users
		ORDER BY created_at DESC
		LIMIT 10;
	`

	rows, err := db.GetDB().Query(ctx, query)
	if err != nil {
		log.Fatalf("Failed to query users: %v", err)
	}
	defer rows.Close()

	fmt.Println("\n=== Recent Users Encryption Key Status ===")
	fmt.Printf("%-36s %-20s %-30s %-10s %s\n", "ID", "Username", "Email", "Key Length", "Has Key")
	fmt.Println(string(make([]byte, 120)))

	for rows.Next() {
		var id, username, email, encKey string
		var keyLen int
		if err := rows.Scan(&id, &username, &email, &encKey, &keyLen); err != nil {
			log.Fatal(err)
		}

		hasKey := "❌ NO"
		if keyLen > 0 {
			hasKey = "✅ YES"
		}

		emailDisplay := email
		if len(email) > 28 {
			emailDisplay = email[:25] + "..."
		}

		fmt.Printf("%-36s %-20s %-30s %-10d %s\n", id, username, emailDisplay, keyLen, hasKey)
	}

	fmt.Println("\n✅ Check complete!")
}
