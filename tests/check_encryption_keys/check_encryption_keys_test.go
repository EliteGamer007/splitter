package main_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"

	"splitter/tests/testlogger"
)

// TestEd25519KeyGeneration verifies that a valid Ed25519 keypair can be generated.
func TestEd25519KeyGeneration(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "encryption", start, testErr) }()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		testErr = err
		t.Fatalf("Key generation failed: %v", err)
	}
	if len(pub) != ed25519.PublicKeySize {
		t.Errorf("Public key wrong size: want %d, got %d", ed25519.PublicKeySize, len(pub))
	}
	if len(priv) != ed25519.PrivateKeySize {
		t.Errorf("Private key wrong size: want %d, got %d", ed25519.PrivateKeySize, len(priv))
	}
}

// TestEd25519SignVerify verifies that a signature generated with a private key
// can be validated with the corresponding public key.
func TestEd25519SignVerify(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "encryption", start, testErr) }()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		testErr = err
		t.Fatalf("Key generation failed: %v", err)
	}

	message := []byte("test-message-for-signature")
	sig := ed25519.Sign(priv, message)

	if !ed25519.Verify(pub, message, sig) {
		t.Error("Signature verification failed with correct key")
	}
}

// TestEd25519RejectWrongKey verifies that a signature is rejected with the wrong public key.
func TestEd25519RejectWrongKey(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "encryption", start, testErr) }()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		testErr = err
		t.Fatalf("Key generation failed: %v", err)
	}
	wrongPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		testErr = err
		t.Fatalf("Second key generation failed: %v", err)
	}

	message := []byte("message-signed-with-different-key")
	sig := ed25519.Sign(priv, message)

	if ed25519.Verify(wrongPub, message, sig) {
		t.Error("Signature should have been rejected with wrong public key, but was accepted")
	}
}

// TestEd25519RejectTamperedMessage verifies that a signature is rejected if the message is tampered.
func TestEd25519RejectTamperedMessage(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "encryption", start, testErr) }()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		testErr = err
		t.Fatalf("Key generation failed: %v", err)
	}

	original := []byte("original message")
	sig := ed25519.Sign(priv, original)

	tampered := []byte("tampered message")
	if ed25519.Verify(pub, tampered, sig) {
		t.Error("Signature should have been rejected for tampered message, but was accepted")
	}
}

// TestBase64RoundTrip verifies that public keys survive base64 encode/decode.
func TestBase64RoundTrip(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "encryption", start, testErr) }()

	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		testErr = err
		t.Fatalf("Key generation failed: %v", err)
	}

	encoded := base64.StdEncoding.EncodeToString(pub)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		testErr = err
		t.Fatalf("Base64 decode failed: %v", err)
	}

	if string(pub) != string(decoded) {
		t.Error("Public key mismatch after base64 round trip")
	}
}

// TestEd25519KeyRotationFlow simulates a full key rotation flow.
func TestEd25519KeyRotationFlow(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "encryption", start, testErr) }()

	// Step 1: Old key signs the rotation message
	oldPub, oldPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		testErr = err
		t.Fatalf("Old key gen failed: %v", err)
	}

	newPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		testErr = err
		t.Fatalf("New key gen failed: %v", err)
	}

	nonce := "unique-nonce-12345"
	ts := "1710000000"
	newPubB64 := base64.StdEncoding.EncodeToString(newPub)
	message := newPubB64 + "|" + nonce + "|" + ts
	sig := ed25519.Sign(oldPriv, []byte(message))

	// Step 2: Verify with old public key
	if !ed25519.Verify(oldPub, []byte(message), sig) {
		t.Error("Key rotation signature should be valid with old public key")
	}

	// Step 3: Signature must be invalid with new public key
	if ed25519.Verify(newPub, []byte(message), sig) {
		t.Error("Key rotation signature should NOT be valid with new public key")
	}
}
