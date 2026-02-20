package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	// Read connection details from .env (Instance 1's database)
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")

	if host == "" || user == "" || password == "" {
		log.Fatal("Missing DB_HOST, DB_USER, or DB_PASSWORD in .env")
	}

	newDBName := "neondb_2"

	// Connect to the default 'neondb' database to create the new one
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/neondb?sslmode=require",
		user, password, host, port)

	ctx := context.Background()

	fmt.Println("=== Splitter Federation: Second Instance Database Setup ===")
	fmt.Printf("Host: %s\n", host)
	fmt.Printf("Target DB: %s\n\n", newDBName)

	// Step 1: Check if neondb_2 already exists
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		log.Fatalf("Failed to connect to neondb: %v", err)
	}

	var exists bool
	err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", newDBName).Scan(&exists)
	if err != nil {
		log.Fatalf("Failed to check database existence: %v", err)
	}

	if exists {
		fmt.Printf("✅ Database '%s' already exists!\n", newDBName)
	} else {
		fmt.Printf("Creating database '%s'...\n", newDBName)
		// Neon doesn't allow CREATE DATABASE via pooler, so we'll inform the user
		fmt.Println("")
		fmt.Println("⚠️  Neon Cloud does not support CREATE DATABASE via the pooler connection.")
		fmt.Println("   Please create the database manually:")
		fmt.Println("")
		fmt.Println("   Option 1: Neon Console")
		fmt.Println("   → Go to https://console.neon.tech")
		fmt.Println("   → Select your project")
		fmt.Println("   → Go to 'Databases' tab")
		fmt.Println("   → Click 'New Database'")
		fmt.Printf("   → Name: %s, Owner: %s\n", newDBName, user)
		fmt.Println("")
		fmt.Println("   Option 2: Neon SQL Editor")
		fmt.Printf("   → Run: CREATE DATABASE %s;\n", newDBName)
		fmt.Println("")
		fmt.Println("   After creating the database, run this script again to apply the schema.")
		conn.Close(ctx)
		os.Exit(0)
	}
	conn.Close(ctx)

	// Step 2: Connect to neondb_2 and apply master schema
	newConnStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require",
		user, password, host, port, newDBName)

	conn2, err := pgx.Connect(ctx, newConnStr)
	if err != nil {
		log.Fatalf("Failed to connect to %s: %v", newDBName, err)
	}
	defer conn2.Close(ctx)

	// Check if schema is already applied by checking for the users table
	var tableExists bool
	err = conn2.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'users')").Scan(&tableExists)
	if err != nil {
		log.Fatalf("Failed to check table existence: %v", err)
	}

	if tableExists {
		fmt.Println("✅ Schema already applied to neondb_2!")

		// Verify table count
		var tableCount int
		err = conn2.QueryRow(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE'").Scan(&tableCount)
		if err == nil {
			fmt.Printf("   Tables found: %d\n", tableCount)
		}

		fmt.Println("\n✅ Instance 2 database is ready!")
		fmt.Println("   Run: .\\scripts\\run_instance2.ps1")
		return
	}

	// Read and apply master schema
	schemaPath := "migrations/000_master_schema.sql"
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		log.Fatalf("Failed to read master schema from %s: %v", schemaPath, err)
	}

	schema := string(schemaBytes)
	fmt.Printf("Applying master schema (%d bytes)...\n", len(schema))

	// Execute the entire schema as a single batch — this correctly handles
	// PL/pgSQL function bodies with $$ delimiters
	_, err = conn2.Exec(ctx, schema)
	if err != nil {
		log.Fatalf("❌ Schema application failed: %v", err)
	}

	// Verify
	var finalTableCount int
	err = conn2.QueryRow(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE'").Scan(&finalTableCount)
	if err == nil {
		fmt.Printf("   Tables created: %d\n", finalTableCount)
	}

	fmt.Println("\n✅ Instance 2 database setup complete!")
	fmt.Println("   Run: .\\scripts\\run_instance2.ps1")
}
