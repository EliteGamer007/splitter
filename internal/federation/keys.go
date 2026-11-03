package federation

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"splitter/internal/db"
)

var (
	instancePrivateKey *rsa.PrivateKey
	instancePublicPEM  string
	instanceDomain     string
	keyMu              sync.Mutex
)

type instanceKeyStore interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

// EnsureInstanceKeys generates or loads RSA-2048 keypair for this instance
func EnsureInstanceKeys(domain string) error {
	if db.GetDB() == nil {
		return errors.New("instance key database is unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return ensureInstanceKeys(ctx, domain, db.GetDB())
}

func ensureInstanceKeys(ctx context.Context, domain string, store instanceKeyStore) error {
	keyMu.Lock()
	defer keyMu.Unlock()
	if strings.TrimSpace(domain) == "" || len(domain) > 253 {
		return errors.New("invalid instance domain")
	}
	var pubPEM, privPEM string
	err := store.QueryRow(ctx, `SELECT public_key_pem, private_key_pem FROM instance_keys WHERE domain=$1`, domain).Scan(&pubPEM, &privPEM)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("cannot read persisted instance keys: %w", err)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return errors.New("cannot generate instance signing key")
		}
		pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
		if err != nil {
			return errors.New("cannot encode instance public key")
		}
		pubPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes}))
		privPEM = string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
		// Another process may have won after our read. Never update a persisted
		// identity during startup; always reload the winning database row.
		if _, err := store.Exec(ctx, `INSERT INTO instance_keys(domain,public_key_pem,private_key_pem) VALUES($1,$2,$3) ON CONFLICT(domain) DO NOTHING`, domain, pubPEM, privPEM); err != nil {
			return fmt.Errorf("cannot persist instance keys: %w", err)
		}
		if err := store.QueryRow(ctx, `SELECT public_key_pem, private_key_pem FROM instance_keys WHERE domain=$1`, domain).Scan(&pubPEM, &privPEM); err != nil {
			return fmt.Errorf("cannot reload persisted instance keys: %w", err)
		}
	}
	block, _ := pem.Decode([]byte(privPEM))
	if block == nil {
		return errors.New("invalid persisted instance private key")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil || key.Validate() != nil || key.N.BitLen() < 2048 {
		return errors.New("invalid persisted instance private key")
	}
	publicBlock, _ := pem.Decode([]byte(pubPEM))
	if publicBlock == nil {
		return errors.New("invalid persisted instance public key")
	}
	parsed, err := x509.ParsePKIXPublicKey(publicBlock.Bytes)
	if err != nil {
		return errors.New("invalid persisted instance public key")
	}
	public, ok := parsed.(*rsa.PublicKey)
	if !ok || public.N.Cmp(key.N) != 0 || public.E != key.E {
		return errors.New("persisted instance key pair does not match")
	}
	instancePrivateKey, instancePublicPEM, instanceDomain = key, pubPEM, domain
	log.Printf("[Federation] Loaded persisted instance signing key for %s", domain)
	return nil
}

// GetInstancePrivateKey returns the instance's RSA private key
func GetInstancePrivateKey() *rsa.PrivateKey {
	keyMu.Lock()
	defer keyMu.Unlock()
	return instancePrivateKey
}

// GetInstancePublicKeyPEM returns the instance's public key in PEM format
func GetInstancePublicKeyPEM() string {
	keyMu.Lock()
	defer keyMu.Unlock()
	return instancePublicPEM
}

// GetInstanceDomain returns the configured domain
func GetInstanceDomain() string {
	keyMu.Lock()
	defer keyMu.Unlock()
	return instanceDomain
}
