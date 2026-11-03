package service

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"splitter/internal/db"
	"splitter/internal/repository"
	"splitter/tests/testdb"
	"testing"
	"time"
)

func botDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("explicit disposable database required")
	}
	if _, err := testdb.ParseLocalURL(raw); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	base, err := pgxpool.New(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("bot_test_%d", time.Now().UnixNano())
	if _, err := base.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 4
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.ApplyMigrations(ctx, pool, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	old := db.DB
	db.DB = pool
	t.Cleanup(func() { db.DB = old; pool.Close(); base.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); base.Close() })
	_, err = pool.Exec(ctx, `INSERT INTO users(username,instance_domain,did,display_name,public_key)VALUES('alice','test.local','did:bot-alice','Alice',''),('bob','test.local','did:bot-bob','Bob','');
 INSERT INTO posts(id,author_did,content,visibility)VALUES('30000000-0000-4000-8000-000000000001','did:bot-alice','@split explain one','public'),('30000000-0000-4000-8000-000000000002','did:bot-alice','@split explain two','public'),('30000000-0000-4000-8000-000000000003','did:bot-bob','@split explain three','public'),('30000000-0000-4000-8000-000000000004','did:bot-bob','@split private','followers');`)
	if err != nil {
		t.Fatal(err)
	}
	return pool
}
func TestBotPersistentDailyBudgetAndSingleMentionReservation(t *testing.T) {
	p := botDatabase(t)
	ctx := context.Background()
	store := databaseSplitBotStore{domain: "test.local", appLimit: 2, userLimit: 1, replyRepo: repository.NewReplyRepository()}
	first, err := store.Reserve(ctx, "30000000-0000-4000-8000-000000000001", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, first, "A useful public answer"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Reserve(ctx, "30000000-0000-4000-8000-000000000001", nil); err == nil {
		t.Error("same mention spent a second request")
	}
	if _, err := store.Reserve(ctx, "30000000-0000-4000-8000-000000000002", nil); err == nil {
		t.Error("per-user daily budget exceeded")
	}
	if _, err := store.Reserve(ctx, "30000000-0000-4000-8000-000000000003", nil); err != nil {
		t.Fatal(err)
	}
	// A new worker/store still observes persisted consumption after a process restart.
	restarted := databaseSplitBotStore{domain: "test.local", appLimit: 2, userLimit: 2, replyRepo: repository.NewReplyRepository()}
	if _, err := restarted.Reserve(ctx, "30000000-0000-4000-8000-000000000002", nil); err == nil {
		t.Error("app daily budget disappeared after restart")
	}
	var attempts, replies int
	p.QueryRow(ctx, "SELECT count(*) FROM ai_bot_requests").Scan(&attempts)
	p.QueryRow(ctx, "SELECT count(*) FROM replies WHERE author_did='did:key:bot_split'").Scan(&replies)
	if attempts != 2 || replies != 1 {
		t.Fatalf("actual AI attempts/replies: %d/%d", attempts, replies)
	}
}
func TestBotNeverReservesPrivateOrSuspendedAuthorContent(t *testing.T) {
	p := botDatabase(t)
	ctx := context.Background()
	store := databaseSplitBotStore{domain: "test.local", appLimit: 10, userLimit: 3}
	if _, err := store.Reserve(ctx, "30000000-0000-4000-8000-000000000004", nil); err == nil {
		t.Error("private content reserved provider call")
	}
	p.Exec(ctx, "UPDATE users SET is_suspended=true WHERE did='did:bot-alice'")
	if _, err := store.Reserve(ctx, "30000000-0000-4000-8000-000000000001", nil); err == nil {
		t.Error("suspended author reserved provider call")
	}
	var count int
	p.QueryRow(ctx, "SELECT count(*) FROM ai_bot_requests").Scan(&count)
	if count != 0 {
		t.Fatalf("forbidden content consumed %d reservations", count)
	}
}
