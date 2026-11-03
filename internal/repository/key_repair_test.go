package repository

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"splitter/internal/db"
	"splitter/tests/testdb"
)

func keyRepairDB(t *testing.T) *pgxpool.Pool {
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
	schema := fmt.Sprintf("keys_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 4
	cfg.MinConns = 0
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	previous := db.DB
	db.DB = pool
	t.Cleanup(func() {
		db.DB = previous
		pool.Close()
		admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
		admin.Close()
	})
	if err := db.ApplyMigrations(ctx, pool, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	return pool
}

const keyTestID = "11111111-1111-4111-8111-111111111111"

func seedKeyUser(t *testing.T, pool *pgxpool.Pool, signing, encryption string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `INSERT INTO users(id,username,email,instance_domain,did,public_key,encryption_public_key) VALUES($1,'key-user','key@example.test','test.local','did:key-user',$2,$3)`, keyTestID, signing, encryption)
	if err != nil {
		t.Fatal(err)
	}
}
func TestInitialSigningKeyCannotOverwriteExistingKey(t *testing.T) {
	p := keyRepairDB(t)
	seedKeyUser(t, p, "existing-key", "")
	r := NewUserRepository()
	if err := r.RegisterInitialKey(context.Background(), keyTestID, "attacker-key"); err == nil {
		t.Fatal("existing signing key accepted replacement")
	}
	var key string
	p.QueryRow(context.Background(), "SELECT public_key FROM users WHERE id=$1", keyTestID).Scan(&key)
	if key != "existing-key" {
		t.Fatalf("signing key overwritten: %q", key)
	}
}
func TestInitialSigningKeyConcurrentRegistrationHasOneWinner(t *testing.T) {
	p := keyRepairDB(t)
	seedKeyUser(t, p, "", "")
	r := NewUserRepository()
	var wg sync.WaitGroup
	wins := make(chan bool, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			wins <- r.RegisterInitialKey(context.Background(), keyTestID, fmt.Sprintf("key-%d", i)) == nil
		}(i)
	}
	wg.Wait()
	close(wins)
	count := 0
	for win := range wins {
		if win {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("concurrent initial registration winners: %d", count)
	}
}
func TestStaleKeyRotationCannotReplaceCurrentKey(t *testing.T) {
	p := keyRepairDB(t)
	seedKeyUser(t, p, "current-key", "")
	r := NewUserRepository()
	if err := r.RotateKey(context.Background(), keyTestID, "stale-key", "attacker-key", "nonce-stale", "rotated"); err == nil {
		t.Fatal("rotation using superseded key was accepted")
	}
	var key string
	p.QueryRow(context.Background(), "SELECT public_key FROM users WHERE id=$1", keyTestID).Scan(&key)
	if key != "current-key" {
		t.Fatalf("current key overwritten: %q", key)
	}
	var count int
	p.QueryRow(context.Background(), "SELECT count(*) FROM key_rotations").Scan(&count)
	if count != 0 {
		t.Fatalf("rejected rotation left %d history entries", count)
	}
}
func TestStaleKeyRevocationCannotClearCurrentKey(t *testing.T) {
	p := keyRepairDB(t)
	seedKeyUser(t, p, "current-key", "")
	r := NewUserRepository()
	if err := r.RevokeCurrentKey(context.Background(), keyTestID, "stale-key"); err == nil {
		t.Fatal("revocation using superseded key was accepted")
	}
	var key string
	p.QueryRow(context.Background(), "SELECT public_key FROM users WHERE id=$1", keyTestID).Scan(&key)
	if key != "current-key" {
		t.Fatalf("current key cleared: %q", key)
	}
}
func TestNewDeviceCannotClaimExistingAccountEncryptionKey(t *testing.T) {
	p := keyRepairDB(t)
	seedKeyUser(t, p, "", "account-key")
	r := NewUserRepository()
	ctx := context.Background()
	unknown, err := r.RequestDeviceKey(ctx, keyTestID, "unknown", "New laptop", "other-key")
	if err != nil {
		t.Fatal(err)
	}
	if unknown.Status != "pending" {
		t.Fatalf("fresh device self-approved with different key: %s", unknown.Status)
	}
	known, err := r.RequestDeviceKey(ctx, keyTestID, "primary", "Primary", "account-key")
	if err != nil {
		t.Fatal(err)
	}
	if known.Status != "approved" {
		t.Fatalf("existing account key not bootstrapped: %s", known.Status)
	}
	if _, err := r.RequestDeviceKey(ctx, keyTestID, "primary", "Claimed primary", "attacker-key"); err == nil {
		t.Fatal("device ID accepted a replacement encryption key")
	}
	var key string
	p.QueryRow(ctx, "SELECT encryption_public_key FROM user_device_keys WHERE user_id=$1 AND device_id='primary'", keyTestID).Scan(&key)
	if key != "account-key" {
		t.Fatalf("approved device key replaced: %q", key)
	}
}
