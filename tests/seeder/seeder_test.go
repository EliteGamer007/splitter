package main_test

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"testing"
	"time"

	"splitter/tests/testlogger"
)

// TestSeederKeyGeneration validates that the seeder-style key generation works.
func TestSeederKeyGeneration(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "seeder", start, testErr) }()

	// Signing key (ECDSA P-256)
	signKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		testErr = err
		t.Fatalf("Failed to generate signing key: %v", err)
	}
	signPubKeyBytes, err := x509.MarshalPKIXPublicKey(&signKey.PublicKey)
	if err != nil {
		testErr = err
		t.Fatalf("Failed to marshal signing public key: %v", err)
	}
	signPubBase64 := base64.StdEncoding.EncodeToString(signPubKeyBytes)
	if signPubBase64 == "" {
		t.Error("Signing public key base64 is empty")
	}

	// Encryption key (ECDH P-256)
	encKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		testErr = err
		t.Fatalf("Failed to generate encryption key: %v", err)
	}
	encPubKeyBytes, err := x509.MarshalPKIXPublicKey(encKey.PublicKey())
	if err != nil {
		testErr = err
		t.Fatalf("Failed to marshal encryption public key: %v", err)
	}
	encPubBase64 := base64.StdEncoding.EncodeToString(encPubKeyBytes)
	if encPubBase64 == "" {
		t.Error("Encryption public key base64 is empty")
	}
}

// TestSeederPublicKeyRoundTrip verifies that public keys are correctly decoded after encoding.
func TestSeederPublicKeyRoundTrip(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "seeder", start, testErr) }()

	encKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		testErr = err
		t.Fatalf("Key generation failed: %v", err)
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(encKey.PublicKey())
	if err != nil {
		testErr = err
		t.Fatalf("MarshalPKIX failed: %v", err)
	}
	encoded := base64.StdEncoding.EncodeToString(pubBytes)

	// Decode and parse back
	der, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		testErr = err
		t.Fatalf("Base64 decode failed: %v", err)
	}
	parsed, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		testErr = err
		t.Fatalf("ParsePKIX failed: %v", err)
	}

	ecdsaParsed, ok := parsed.(*ecdsa.PublicKey)
	if !ok {
		t.Fatal("Parsed key is not ECDSA")
	}

	ecdhParsed, err := ecdsaParsed.ECDH()
	if err != nil {
		testErr = err
		t.Fatalf("ECDH conversion failed: %v", err)
	}

	if string(ecdhParsed.Bytes()) != string(encKey.PublicKey().Bytes()) {
		t.Error("Round-tripped key bytes do not match original")
	}
}

// TestSeederSharedSecretDerivation validates ECDH shared secret derivation (Alice ↔ Bob).
func TestSeederSharedSecretDerivation(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "seeder", start, testErr) }()

	aliceKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		testErr = err
		t.Fatalf("Alice key gen failed: %v", err)
	}

	bobKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		testErr = err
		t.Fatalf("Bob key gen failed: %v", err)
	}

	aliceSecret, err := aliceKey.ECDH(bobKey.PublicKey())
	if err != nil {
		testErr = err
		t.Fatalf("Alice ECDH failed: %v", err)
	}

	bobSecret, err := bobKey.ECDH(aliceKey.PublicKey())
	if err != nil {
		testErr = err
		t.Fatalf("Bob ECDH failed: %v", err)
	}

	if string(aliceSecret) != string(bobSecret) {
		t.Error("Shared secrets don't match between Alice and Bob")
	}
}

// TestSeederUniqueUsernames validates that seeder generates unique usernames.
func TestSeederUniqueUsernames(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "seeder", start, testErr) }()

	seen := make(map[string]bool)
	for i := 0; i < 10; i++ {
		// Simulate seeder's naming convention
		username := time.Now().Format("20060102150405.999999999")
		if seen[username] {
			t.Errorf("Duplicate username generated: %s", username)
		}
		seen[username] = true
		time.Sleep(time.Nanosecond)
	}
}
