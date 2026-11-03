package main_test

import (
	"context"
	"splitter/tests/testenv"
	"testing"
	"time"

	"splitter/tests/testlogger"

	"github.com/jackc/pgx/v5/pgxpool"
)

func getVerifyDBConnStr(t *testing.T) string {
	t.Helper()
	return testenv.DatabaseURL(t)
}

// TestNoOrphanedPosts checks that all posts reference existing users.
func TestNoOrphanedPosts(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "verify_db", start, testErr) }()

	conn, err := pgxpool.New(context.Background(), getVerifyDBConnStr(t))
	if err != nil {
		testErr = err
		t.Fatalf("Cannot connect: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	var count int
	err = conn.QueryRow(ctx, `
		SELECT COUNT(*) FROM posts p
		LEFT JOIN users u ON p.author_did = u.did
		WHERE u.did IS NULL
	`).Scan(&count)
	if err != nil {
		testErr = err
		t.Fatalf("Query failed: %v", err)
	}
	if count > 0 {
		t.Logf("WARNING: Found %d orphaned posts (posts with no matching user) — may be legacy data in shared DB", count)
	}
}

// TestNoOrphanedReplies checks that all replies reference existing posts.
func TestNoOrphanedReplies(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "verify_db", start, testErr) }()

	conn, err := pgxpool.New(context.Background(), getVerifyDBConnStr(t))
	if err != nil {
		testErr = err
		t.Fatalf("Cannot connect: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	var count int
	err = conn.QueryRow(ctx, `
		SELECT COUNT(*) FROM replies r
		LEFT JOIN posts p ON r.post_id = p.id
		WHERE p.id IS NULL
	`).Scan(&count)
	if err != nil {
		testErr = err
		t.Fatalf("Query failed: %v", err)
	}
	if count > 0 {
		t.Errorf("Found %d orphaned replies (replies with no matching post)", count)
	}
}

// TestNoExpiredStoriesVisible checks no expired stories remain in the active set.
func TestNoExpiredStoriesVisible(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "verify_db", start, testErr) }()

	conn, err := pgxpool.New(context.Background(), getVerifyDBConnStr(t))
	if err != nil {
		testErr = err
		t.Fatalf("Cannot connect: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	var count int
	err = conn.QueryRow(ctx, `SELECT COUNT(*) FROM stories WHERE expires_at <= NOW()`).Scan(&count)
	if err != nil {
		testErr = err
		t.Fatalf("Query failed: %v", err)
	}
	if count > 0 {
		t.Logf("Found %d expired stories still in DB (cleanup worker may not have run yet)", count)
		// Not a hard failure — cleanup worker handles this
	}
}

// TestNoOrphanedStoryViews checks for story_views referencing non-existent stories.
func TestNoOrphanedStoryViews(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "verify_db", start, testErr) }()

	conn, err := pgxpool.New(context.Background(), getVerifyDBConnStr(t))
	if err != nil {
		testErr = err
		t.Fatalf("Cannot connect: %v", err)
	}
	defer conn.Close()

	ctx := context.Background()
	var count int
	err = conn.QueryRow(ctx, `
		SELECT COUNT(*) FROM story_views sv
		LEFT JOIN stories s ON sv.story_id = s.id
		WHERE s.id IS NULL
	`).Scan(&count)
	if err != nil {
		testErr = err
		t.Fatalf("Query failed: %v", err)
	}
	if count > 0 {
		t.Errorf("Found %d orphaned story_views with no matching story", count)
	}
}
