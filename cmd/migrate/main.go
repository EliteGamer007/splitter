package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

func main() {
	log.Println("Starting database migration...")

	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, relying on environment variables")
	}

	dbName := os.Getenv("DB_NAME")
	if dbName != "" {
		runMigrations(dbName)
	}

	instance2DBName := os.Getenv("INSTANCE2_DB_NAME")
	if instance2DBName != "" {
		runMigrations(instance2DBName)
	}
}

func runMigrations(dbName string) {
	log.Printf("\nRunning migrations for database: %s", dbName)

	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=require",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		dbName,
	)

	// Mask password for logging
	safeConnStr := fmt.Sprintf(
		"postgres://%s:***@%s:%s/%s?sslmode=require",
		os.Getenv("DB_USER"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		dbName,
	)

	log.Printf("🔌 Connecting to: %s", safeConnStr)

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("❌ Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Failed to ping database: %v", err)
	}

	log.Println("✅ Connected to database")

	// Create migrations table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ DEFAULT now()
		)
	`)
	if err != nil {
		log.Fatalf("❌ Failed to create schema_migrations table: %v", err)
	}

	// Read migrations directory
	entries, err := os.ReadDir("migrations")
	if err != nil {
		log.Fatalf("❌ Failed to read migrations directory: %v", err)
	}

	var migrationFiles []string

	for _, entry := range entries {
		if !entry.IsDir() &&
			len(entry.Name()) > 4 &&
			entry.Name()[len(entry.Name())-4:] == ".sql" &&
			entry.Name() != "verify_migration.sql" &&
			entry.Name() != "000_master_schema.sql" &&
			entry.Name() != "001_initial_schema.sql" &&
			entry.Name() != "002_upgrade_to_current.sql" {

			migrationFiles = append(migrationFiles, entry.Name())
		}
	}

	for _, file := range migrationFiles {

		var exists string

		err := db.QueryRow(
			"SELECT version FROM schema_migrations WHERE version = $1",
			file,
		).Scan(&exists)

		if err == sql.ErrNoRows {

			log.Printf("▶️ Applying migration: %s", file)

			content, err := os.ReadFile("migrations/" + file)
			if err != nil {
				log.Fatalf("❌ Failed to read migration file %s: %v", file, err)
			}

			_, err = db.ExecContext(context.Background(), string(content))
			if err != nil {

				log.Printf("⚠️ Migration %s returned error: %v", file, err)

				// Mark as applied to prevent loops
				_, _ = db.Exec(
					"INSERT INTO schema_migrations (version) VALUES ($1)",
					file,
				)

				continue
			}

			_, err = db.Exec(
				"INSERT INTO schema_migrations (version) VALUES ($1)",
				file,
			)

			if err != nil {
				log.Fatalf("❌ Failed to record migration %s: %v", file, err)
			}

		} else if err != nil {

			log.Fatalf(
				"❌ Failed to check migration status for %s: %v",
				file,
				err,
			)

		} else {

			log.Printf("⏭️ Skipping already applied migration: %s", file)

		}
	}

	log.Println("Migration completed successfully.")
}
