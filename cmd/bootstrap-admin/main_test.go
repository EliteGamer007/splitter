package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"splitter/internal/auth"
	"splitter/internal/db"
	"splitter/tests/testdb"
	"sync"
	"testing"
	"time"
)

func bootstrapFixture(t *testing.T) *pgxpool.Pool {
	t.Helper()
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("requires explicit disposable TEST_DATABASE_URL")
	}
	if _, err := testdb.ParseLocalURL(raw); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("bootstrap_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
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
	t.Cleanup(func() { pool.Close(); admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	if err := db.ApplyMigrations(ctx, pool, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	return pool
}
func TestBootstrapRequiresExplicitStrongCredentials(t *testing.T) {
	for _, tc := range []struct{ username, email, password string }{{"", "admin@example.test", "strong-test-password"}, {"admin", "", "strong-test-password"}, {"admin", "admin@example.test", "short"}} {
		if err := bootstrapAdmin(context.Background(), nil, tc.username, tc.email, tc.password, "test.local"); err == nil {
			t.Error("missing/weak bootstrap credentials accepted")
		}
	}
}
func TestBootstrapCreatesOneAdminAndNeverChangesExistingAccount(t *testing.T) {
	p := bootstrapFixture(t)
	ctx := context.Background()
	password := "supplied-test-password"
	if err := bootstrapAdmin(ctx, p, "owner", "owner@example.test", password, "test.local"); err != nil {
		t.Fatal(err)
	}
	var hash, role string
	if err := p.QueryRow(ctx, "SELECT password_hash,role FROM users WHERE username='owner'").Scan(&hash, &role); err != nil {
		t.Fatal(err)
	}
	if role != "admin" || !auth.CheckPasswordHash(password, hash) {
		t.Fatal("bootstrap did not create administrator with supplied password")
	}
	if err := bootstrapAdmin(ctx, p, "owner", "owner@example.test", "a-new-test-password", "other.local"); err == nil {
		t.Fatal("repeated bootstrap was allowed to modify an administrator")
	}
	var after, domain string
	p.QueryRow(ctx, "SELECT password_hash,instance_domain FROM users WHERE username='owner'").Scan(&after, &domain)
	if after != hash || domain != "test.local" {
		t.Fatal("existing administrator credentials/domain changed")
	}
}
func TestBootstrapNeverPromotesAnExistingAccount(t *testing.T) {
	p := bootstrapFixture(t)
	ctx := context.Background()
	_, err := p.Exec(ctx, `INSERT INTO users(username,email,password_hash,instance_domain,role) VALUES('ordinary','ordinary@example.test','original-hash','test.local','user')`)
	if err != nil {
		t.Fatal(err)
	}
	if err := bootstrapAdmin(ctx, p, "ordinary", "ordinary@example.test", "supplied-test-password", "test.local"); err == nil {
		t.Fatal("bootstrap claimed an existing account")
	}
	var role, hash string
	p.QueryRow(ctx, "SELECT role,password_hash FROM users WHERE username='ordinary'").Scan(&role, &hash)
	if role != "user" || hash != "original-hash" {
		t.Fatal("bootstrap mutated existing account")
	}
}
func TestBootstrapConcurrentInvocationsCreateOnlyOneAdministrator(t *testing.T) {
	p := bootstrapFixture(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results <- bootstrapAdmin(ctx, p, fmt.Sprintf("owner%d", i), fmt.Sprintf("owner%d@example.test", i), "supplied-test-password", "test.local")
		}(i)
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("concurrent bootstrap successes=%d", success)
	}
	var count int
	p.QueryRow(ctx, "SELECT count(*) FROM users WHERE role='admin'").Scan(&count)
	if count != 1 {
		t.Fatalf("bootstrap created %d administrators", count)
	}
}
