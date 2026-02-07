package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

func main() {
	connString := "postgresql://neondb_owner:npg_9yWhGxj7OYsp@ep-falling-mode-a1k832j8-pooler.ap-southeast-1.aws.neon.tech:5432/neondb?sslmode=require"

	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(context.Background())

	fmt.Println("✅ Database connection successful!\n")

	// Check for tables
	var tableCount int
	err = conn.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'").Scan(&tableCount)
	if err != nil {
		log.Fatalf("Error checking tables: %v\n", err)
	}
	fmt.Printf("📊 Number of tables: %d\n\n", tableCount)

	// List all tables
	rows, err := conn.Query(context.Background(),
		"SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' ORDER BY table_name")
	if err != nil {
		log.Fatalf("Error listing tables: %v\n", err)
	}
	defer rows.Close()

	fmt.Println("📋 Tables in database:")
	for rows.Next() {
		var tableName string
		rows.Scan(&tableName)
		fmt.Printf("  - %s\n", tableName)
	}

	// Check user count
	var userCount int
	err = conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		fmt.Printf("⚠️  Error checking users: %v\n", err)
	} else {
		fmt.Printf("\n👥 Number of users: %d\n", userCount)
	}

	// Check post count
	var postCount int
	err = conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM posts").Scan(&postCount)
	if err != nil {
		fmt.Printf("⚠️  Error checking posts: %v\n", err)
	} else {
		fmt.Printf("📝 Number of posts: %d\n", postCount)
	}

	// List some users
	userRows, err := conn.Query(context.Background(),
		"SELECT username, email, role, created_at FROM users LIMIT 5")
	if err != nil {
		fmt.Printf("⚠️  Error listing users: %v\n", err)
	} else {
		defer userRows.Close()
		fmt.Println("\n👤 Sample users:")
		for userRows.Next() {
			var username, email, role string
			var createdAt string
			userRows.Scan(&username, &email, &role, &createdAt)
			fmt.Printf("  - %s (%s) [%s] - Created: %s\n", username, email, role, createdAt)
		}
	}

	fmt.Println("\n✅ Database check complete!")
}
