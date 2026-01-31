package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// RunMigrations executes SQL migration files from the migrations directory
func RunMigrations(migrationsDir string) error {
	log.Printf("Checking for migrations in: %s", migrationsDir)

	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var sqlFiles []string
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
			sqlFiles = append(sqlFiles, f.Name())
		}
	}

	sort.Strings(sqlFiles)

	for _, file := range sqlFiles {
		log.Printf("Running migration: %s", file)
		content, err := os.ReadFile(filepath.Join(migrationsDir, file))
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		// Split by semicolons maybe? Or just run the whole file?
		// pgx Execute can run multiple statements usually.
		_, err = DB.Exec(context.Background(), string(content))
		if err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}
	}

	log.Println("Migrations completed successfully")
	return nil
}
