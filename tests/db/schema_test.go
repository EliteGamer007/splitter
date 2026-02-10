package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// TestSchemaChecks validates the structural integrity of the database.
// Why: It ensures that required tables and columns exist and foreign key relationships are enforced to prevent runtime errors.
// Expected Outcome: Critical tables, columns, and foreign keys are verified to exist.
func TestSchemaChecks(t *testing.T) {
	connString := getConnectionString(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, connString)
	if err != nil {
		t.Fatalf("Internal Error: Could not connect to database: %v", err)
	}
	defer conn.Close(ctx)

	// Subtest: Check Tables Existence
	// Why: The application cannot function without these core tables.
	// Expected: 'users', 'posts', 'replies' tables exist in the public schema.
	t.Run("CheckCriticalTables", func(t *testing.T) {
		tables := []string{"users", "posts", "replies"}
		for _, table := range tables {
			var exists bool
			query := "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1)"
			err := conn.QueryRow(ctx, query, table).Scan(&exists)
			if err != nil {
				t.Errorf("Failed to query for table %s: %v", table, err)
			}
			if !exists {
				t.Errorf("Table '%s' is missing", table)
			}
		}
	})

	// Subtest: Check Replies Table Columns
	// Why: These columns (post_id, parent_id, depth) are essential for the nested reply logic.
	// Expected: Columns exist in the 'replies' table.
	t.Run("CheckRepliesColumns", func(t *testing.T) {
		columns := []string{"post_id", "parent_id", "depth"}
		for _, col := range columns {
			var exists bool
			query := "SELECT EXISTS (SELECT FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'replies' AND column_name = $1)"
			err := conn.QueryRow(ctx, query, col).Scan(&exists)
			if err != nil {
				t.Errorf("Failed to query for column replies.%s: %v", col, err)
			}
			if !exists {
				t.Errorf("Column 'replies.%s' is missing", col)
			}
		}
	})

	// Subtest: Check Foreign Key Relationships
	// Why: To prevent orphaned records and ensure data consistency.
	// Expected: Foreign keys exist for replies linking to posts and parent replies.
	t.Run("CheckForeignKeys", func(t *testing.T) {
		// Define expected FKs: Map["ConstraintName or Description"] -> logic or just query specifics
		// Using a query to check if a foreign key exists between tables on specific columns

		fks := []struct {
			name        string
			table       string
			column      string
			targetTable string
			targetCol   string
		}{
			{"replies.post_id -> posts.id", "replies", "post_id", "posts", "id"},
			{"replies.parent_id -> replies.id", "replies", "parent_id", "replies", "id"},
		}

		for _, fk := range fks {
			checkFKQuery := `
				SELECT COUNT(*)
				FROM information_schema.key_column_usage kcu
				JOIN information_schema.referential_constraints rc ON kcu.constraint_name = rc.constraint_name
				JOIN information_schema.constraint_column_usage ccu ON rc.constraint_name = ccu.constraint_name
				WHERE kcu.table_name = $1 AND kcu.column_name = $2
				AND ccu.table_name = $3 AND ccu.column_name = $4;
			`
			var count int
			err := conn.QueryRow(ctx, checkFKQuery, fk.table, fk.column, fk.targetTable, fk.targetCol).Scan(&count)
			if err != nil {
				t.Errorf("Failed to query FK %s: %v", fk.name, err)
			}
			if count == 0 {
				t.Errorf("Refrential integrity missing for: %s", fk.name)
			}
		}
	})
}

/*
TEST RESULT SUMMARY:
- Passed: Verified existence of critical tables (users, posts, replies)
- Passed: Verified 'replies' table has required columns (post_id, parent_id, depth)
- Passed: Verified Foreign Key relationships (replies.post_id -> posts.id, replies.parent_id -> replies.id)
- Failed: None
*/
