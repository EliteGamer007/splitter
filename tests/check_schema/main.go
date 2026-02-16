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

	// Check messages table schema
	query := `
		SELECT column_name, data_type, is_nullable
		FROM information_schema.columns
		WHERE table_name = 'messages'
		ORDER BY ordinal_position;
	`

	rows, err := db.GetDB().Query(ctx, query)
	if err != nil {
		log.Fatalf("Failed to query schema: %v", err)
	}
	defer rows.Close()

	fmt.Println("\n=== Messages Table Schema ===")
	for rows.Next() {
		var colName, dataType, nullable string
		if err := rows.Scan(&colName, &dataType, &nullable); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("  - %s: %s (nullable: %s)\n", colName, dataType, nullable)
	}

	// Check for deleted_at and edited_at specifically
	checkQuery := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'messages' AND column_name = 'deleted_at'
		) as has_deleted_at,
		EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'messages' AND column_name = 'edited_at'
		) as has_edited_at;
	`

	var hasDeletedAt, hasEditedAt bool
	err = db.GetDB().QueryRow(ctx, checkQuery).Scan(&hasDeletedAt, &hasEditedAt)
	if err != nil {
		log.Fatalf("Failed to check columns: %v", err)
	}

	fmt.Println("\n=== Column Check ===")
	fmt.Printf("  deleted_at exists: %v\n", hasDeletedAt)
	fmt.Printf("  edited_at exists: %v\n", hasEditedAt)

	if !hasDeletedAt || !hasEditedAt {
		fmt.Println("\n⚠️  MISSING COLUMNS! Run migration 008 manually:")
		fmt.Println("    ALTER TABLE messages ADD COLUMN deleted_at TIMESTAMPTZ DEFAULT NULL;")
		fmt.Println("    ALTER TABLE messages ADD COLUMN edited_at TIMESTAMPTZ DEFAULT NULL;")
	} else {
		fmt.Println("\n✅ Schema is correct!")
	}
}
