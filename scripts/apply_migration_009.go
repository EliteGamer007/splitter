package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
)

func main() {
	migration := `
CREATE TABLE IF NOT EXISTS remote_actors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    actor_uri TEXT UNIQUE NOT NULL,
    username TEXT NOT NULL,
    domain TEXT NOT NULL,
    inbox_url TEXT NOT NULL,
    outbox_url TEXT,
    public_key_pem TEXT,
    display_name TEXT DEFAULT '',
    avatar_url TEXT DEFAULT '',
    last_fetched_at TIMESTAMPTZ DEFAULT now(),
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS instance_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    domain TEXT UNIQUE NOT NULL,
    public_key_pem TEXT NOT NULL,
    private_key_pem TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_remote_actors_domain ON remote_actors(domain);
CREATE INDEX IF NOT EXISTS idx_remote_actors_username_domain ON remote_actors(username, domain);
`

	databases := []struct {
		name   string
		dbName string
	}{
		{"Instance 1", "neondb"},
		{"Instance 2", "neondb_2"},
	}

	host := "ep-falling-mode-a1k832j8-pooler.ap-southeast-1.aws.neon.tech"
	user := "neondb_owner"
	password := "npg_doQ6W7BuhytJ"

	for _, db := range databases {
		fmt.Printf("=== Applying migration to %s (%s) ===\n", db.name, db.dbName)

		connStr := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=require",
			user, password, host, db.dbName)

		conn, err := pgx.Connect(context.Background(), connStr)
		if err != nil {
			log.Printf("ERROR connecting to %s: %v\n", db.name, err)
			continue
		}

		_, err = conn.Exec(context.Background(), migration)
		if err != nil {
			log.Printf("ERROR executing migration on %s: %v\n", db.name, err)
		} else {
			fmt.Printf("  ✓ Migration applied successfully to %s\n", db.name)
		}

		// Verify tables exist
		var exists bool
		conn.QueryRow(context.Background(),
			"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name='instance_keys')").Scan(&exists)
		fmt.Printf("  instance_keys exists: %v\n", exists)

		conn.QueryRow(context.Background(),
			"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name='remote_actors')").Scan(&exists)
		fmt.Printf("  remote_actors exists: %v\n", exists)

		conn.Close(context.Background())
		fmt.Println()
	}

	fmt.Println("Migration complete!")
	os.Exit(0)
}
