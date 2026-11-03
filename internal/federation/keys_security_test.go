package federation

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"splitter/internal/db"
	"splitter/tests/testdb"
	"testing"
	"time"
)

func keyStoreFixture(t *testing.T) *pgxpool.Pool {
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
	schema := fmt.Sprintf("instance_keys_%d", time.Now().UnixNano())
	if _, err := base.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.ApplyMigrations(ctx, pool, "../../migrations"); err != nil {
		t.Fatal(err)
	}
	oldPrivate, oldPublic, oldDomain := instancePrivateKey, instancePublicPEM, instanceDomain
	t.Cleanup(func() {
		instancePrivateKey, instancePublicPEM, instanceDomain = oldPrivate, oldPublic, oldDomain
		pool.Close()
		base.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
		base.Close()
	})
	return pool
}
func storedKeyPair(t *testing.T) (string, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pub})), string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
}

type failedKeyRead struct{ *pgxpool.Pool }
type failedKeyRow struct{}

func (failedKeyRow) Scan(...any) error                                   { return errors.New("transient key read failure") }
func (s failedKeyRead) QueryRow(context.Context, string, ...any) pgx.Row { return failedKeyRow{} }
func TestInstanceKeyReadFailureNeverReplacesPersistedIdentity(t *testing.T) {
	p := keyStoreFixture(t)
	ctx := context.Background()
	pub, priv := storedKeyPair(t)
	if _, err := p.Exec(ctx, "INSERT INTO instance_keys(domain,public_key_pem,private_key_pem)VALUES('test.local',$1,$2)", pub, priv); err != nil {
		t.Fatal(err)
	}
	if err := ensureInstanceKeys(ctx, "test.local", failedKeyRead{p}); err == nil {
		t.Error("transient read error was treated as a missing key")
	}
	var after string
	if err := p.QueryRow(ctx, "SELECT public_key_pem FROM instance_keys WHERE domain='test.local'").Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != pub {
		t.Fatal("transient database error replaced persisted instance identity")
	}
}

type racingKeyStore struct {
	*pgxpool.Pool
	pub, priv string
	first     bool
}
type raceKeyRow struct {
	pgx.Row
	afterMissing func()
}

func (r raceKeyRow) Scan(dest ...any) error {
	err := r.Row.Scan(dest...)
	if errors.Is(err, pgx.ErrNoRows) {
		r.afterMissing()
	}
	return err
}
func (s *racingKeyStore) QueryRow(ctx context.Context, q string, args ...any) pgx.Row {
	row := s.Pool.QueryRow(ctx, q, args...)
	if s.first {
		s.first = false
		return raceKeyRow{row, func() {
			_, err := s.Pool.Exec(ctx, "INSERT INTO instance_keys(domain,public_key_pem,private_key_pem)VALUES('test.local',$1,$2)", s.pub, s.priv)
			if err != nil {
				panic(err)
			}
		}}
	}
	return row
}
func TestConcurrentBootLoadsWinnerWithoutOverwritingItsKeys(t *testing.T) {
	p := keyStoreFixture(t)
	ctx := context.Background()
	pub, priv := storedKeyPair(t)
	store := &racingKeyStore{Pool: p, pub: pub, priv: priv, first: true}
	if err := ensureInstanceKeys(ctx, "test.local", store); err != nil {
		t.Fatal(err)
	}
	var after string
	if err := p.QueryRow(ctx, "SELECT public_key_pem FROM instance_keys WHERE domain='test.local'").Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != pub || GetInstancePublicKeyPEM() != pub {
		t.Fatal("concurrent startup replaced the first persisted RSA identity")
	}
}
