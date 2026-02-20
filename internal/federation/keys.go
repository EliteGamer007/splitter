package federation

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"sync"

	"splitter/internal/db"
)

var (
	instancePrivateKey *rsa.PrivateKey
	instancePublicPEM  string
	instanceDomain     string
	keyMu              sync.Mutex
)

// EnsureInstanceKeys generates or loads RSA-2048 keypair for this instance
func EnsureInstanceKeys(domain string) error {
	keyMu.Lock()
	defer keyMu.Unlock()

	instanceDomain = domain
	ctx := context.Background()

	// Check if keys already exist in DB
	var pubPEM, privPEM string
	err := db.GetDB().QueryRow(ctx,
		`SELECT public_key_pem, private_key_pem FROM instance_keys WHERE domain = $1`, domain,
	).Scan(&pubPEM, &privPEM)

	if err == nil {
		// Keys exist — load them
		block, _ := pem.Decode([]byte(privPEM))
		if block == nil {
			return fmt.Errorf("failed to decode private key PEM")
		}
		privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}
		instancePrivateKey = privKey
		instancePublicPEM = pubPEM
		log.Printf("[Federation] Loaded existing RSA keypair for domain '%s'", domain)
		return nil
	}

	// Generate new RSA-2048 keypair
	log.Printf("[Federation] Generating new RSA-2048 keypair for domain '%s'...", domain)
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Encode private key to PEM
	privBytes := x509.MarshalPKCS1PrivateKey(privKey)
	privBlock := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes}
	privPEM = string(pem.EncodeToMemory(privBlock))

	// Encode public key to PEM
	pubBytes, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}
	pubBlock := &pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes}
	pubPEM = string(pem.EncodeToMemory(pubBlock))

	// Store in DB
	_, err = db.GetDB().Exec(ctx,
		`INSERT INTO instance_keys (domain, public_key_pem, private_key_pem)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (domain) DO UPDATE SET public_key_pem = $2, private_key_pem = $3`,
		domain, pubPEM, privPEM,
	)
	if err != nil {
		return fmt.Errorf("failed to store instance keys: %w", err)
	}

	instancePrivateKey = privKey
	instancePublicPEM = pubPEM
	log.Printf("[Federation] Generated and stored RSA keypair for domain '%s'", domain)
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
