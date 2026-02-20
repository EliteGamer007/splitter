package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DBTarget struct {
	Name       string
	ConnStr    string
	LocalLabel string
}

func main() {
	targets := []DBTarget{
		{
			Name:       "neondb",
			ConnStr:    "postgres://neondb_owner:npg_doQ6W7BuhytJ@ep-falling-mode-a1k832j8-pooler.ap-southeast-1.aws.neon.tech:5432/neondb?sslmode=require",
			LocalLabel: "splitter-1",
		},
		{
			Name:       "neondb_2",
			ConnStr:    "postgres://neondb_owner:npg_doQ6W7BuhytJ@ep-falling-mode-a1k832j8-pooler.ap-southeast-1.aws.neon.tech:5432/neondb_2?sslmode=require",
			LocalLabel: "splitter-2",
		},
	}

	for _, target := range targets {
		fmt.Printf("\n=== Processing %s (%s) ===\n", target.Name, target.LocalLabel)
		if err := processDB(target); err != nil {
			log.Fatalf("failed processing %s: %v", target.Name, err)
		}
	}

	fmt.Println("\nDone.")
}

func processDB(target DBTarget) error {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, target.ConnStr)
	if err != nil {
		return err
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// 1) Normalize old localhost labels to the current local server label
	res, err := tx.Exec(ctx,
		`UPDATE users
		 SET instance_domain = $1
		 WHERE COALESCE(instance_domain, '') = '' OR instance_domain = 'localhost'`,
		target.LocalLabel,
	)
	if err != nil {
		return fmt.Errorf("normalize instance_domain failed: %w", err)
	}
	fmt.Printf("normalized localhost/blank users: %d\n", res.RowsAffected())

	// keep a tiny baseline set of test users, delete most others
	keepList := []string{"testuser", "testuser2"}

	// 2) Delete dependent rows for test users first
	queries := []struct {
		name string
		sql  string
	}{
		{"messages", `DELETE FROM messages
		 WHERE sender_id IN (SELECT id FROM users WHERE username ILIKE '%test%' AND username <> ALL($1))
		    OR recipient_id IN (SELECT id FROM users WHERE username ILIKE '%test%' AND username <> ALL($1))`},
		{"message_threads", `DELETE FROM message_threads
		 WHERE participant_a_id IN (SELECT id FROM users WHERE username ILIKE '%test%' AND username <> ALL($1))
		    OR participant_b_id IN (SELECT id FROM users WHERE username ILIKE '%test%' AND username <> ALL($1))`},
		{"follows", `DELETE FROM follows
		 WHERE follower_did IN (SELECT did FROM users WHERE username ILIKE '%test%' AND username <> ALL($1))
		    OR following_did IN (SELECT did FROM users WHERE username ILIKE '%test%' AND username <> ALL($1))`},
		{"interactions", `DELETE FROM interactions
		 WHERE actor_did IN (SELECT did FROM users WHERE username ILIKE '%test%' AND username <> ALL($1))`},
		{"media", `DELETE FROM media
		 WHERE post_id IN (SELECT id FROM posts WHERE author_did IN (SELECT did FROM users WHERE username ILIKE '%test%' AND username <> ALL($1)))`},
		{"posts", `DELETE FROM posts
		 WHERE author_did IN (SELECT did FROM users WHERE username ILIKE '%test%' AND username <> ALL($1))`},
		{"remote_actors", `DELETE FROM remote_actors
		 WHERE username ILIKE '%test%' AND username <> ALL($1)`},
		{"users", `DELETE FROM users
		 WHERE username ILIKE '%test%' AND username <> ALL($1)`},
	}

	for _, q := range queries {
		r, err := tx.Exec(ctx, q.sql, keepList)
		if err != nil {
			return fmt.Errorf("delete %s failed: %w", q.name, err)
		}
		fmt.Printf("deleted %-14s rows: %d\n", q.name, r.RowsAffected())
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// 3) Print final summary
	var total, testCount int
	_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&total)
	_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE username ILIKE '%test%'").Scan(&testCount)
	fmt.Printf("final users: %d | remaining test users: %d\n", total, testCount)

	rows, _ := pool.Query(ctx, `SELECT COALESCE(instance_domain,'<NULL>'), COUNT(*) FROM users GROUP BY 1 ORDER BY 2 DESC`)
	fmt.Println("instance_domain summary:")
	for rows.Next() {
		var domain string
		var count int
		_ = rows.Scan(&domain, &count)
		fmt.Printf("  %s -> %d\n", domain, count)
	}
	rows.Close()

	return nil
}
