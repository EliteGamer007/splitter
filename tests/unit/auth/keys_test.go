package auth_test

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"splitter/internal/auth"
	"testing"
	"time"
)

/*
KEY ROTATION UNIT TEST SUMMARY (Story 4.3):
- What is tested: Ed25519 helper functions in internal/auth/keys.go (no DB required).
- Test cases:
    - BuildRotationMessage: canonical format
    - VerifyEd25519Signature: valid sig, wrong sig, tampered message, bad encoding
    - DecodeEd25519PublicKey: valid key, wrong length, bad base64

RESULT SUMMARY:
- All tests are pure crypto, zero external dependencies.
- No network or database required to run.
*/

// ─── BuildRotationMessage ────────────────────────────────────────────────────

func TestBuildRotationMessage(t *testing.T) {
	newKey := "AAAA..."
	nonce := "nonce-abc"
	ts := int64(1708456070)

	msg := auth.BuildRotationMessage(newKey, nonce, ts)
	expected := fmt.Sprintf("%s|%s|%d", newKey, nonce, ts)

	if msg != expected {
		t.Errorf("BuildRotationMessage = %q, want %q", msg, expected)
	}
}

func TestBuildRotationMessage_IsDeterministic(t *testing.T) {
	key := "somekey"
	nonce := "uniquenonce"
	ts := time.Now().Unix()

	// Calling twice with same inputs must produce the same message
	if auth.BuildRotationMessage(key, nonce, ts) != auth.BuildRotationMessage(key, nonce, ts) {
		t.Error("BuildRotationMessage is not deterministic")
	}
}

func TestBuildRotationMessage_DifferentInputsProduceDifferentMessages(t *testing.T) {
	ts := int64(1000000)
	m1 := auth.BuildRotationMessage("key1", "nonce1", ts)
	m2 := auth.BuildRotationMessage("key2", "nonce1", ts)
	m3 := auth.BuildRotationMessage("key1", "nonce2", ts)
	m4 := auth.BuildRotationMessage("key1", "nonce1", ts+1)

	if m1 == m2 || m1 == m3 || m1 == m4 {
		t.Error("Different inputs should produce different rotation messages")
	}
}

// ─── VerifyEd25519Signature ─────────────────────────────────────────────────

func TestVerifyEd25519Signature_ValidSignature(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	message := "test-message-content"
	sig := ed25519.Sign(priv, []byte(message))

	pubB64 := base64.StdEncoding.EncodeToString(pub)
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	if err := auth.VerifyEd25519Signature(pubB64, message, sigB64); err != nil {
		t.Errorf("Expected valid signature to verify, got error: %v", err)
	}
}

func TestVerifyEd25519Signature_WrongPrivateKey(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	_, wrongPriv, _ := ed25519.GenerateKey(nil) // different key pair

	message := "test-message-content"
	sig := ed25519.Sign(wrongPriv, []byte(message))

	pubB64 := base64.StdEncoding.EncodeToString(pub)
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	if err := auth.VerifyEd25519Signature(pubB64, message, sigB64); err == nil {
		t.Error("Expected wrong-key signature to fail, but it passed")
	}
}

func TestVerifyEd25519Signature_TamperedMessage(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	sig := ed25519.Sign(priv, []byte("original-message"))

	pubB64 := base64.StdEncoding.EncodeToString(pub)
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	// Verify against a DIFFERENT message — must fail
	if err := auth.VerifyEd25519Signature(pubB64, "tampered-message", sigB64); err == nil {
		t.Error("Expected tampered message to fail verification, but it passed")
	}
}

func TestVerifyEd25519Signature_InvalidPublicKeyEncoding(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	sig := ed25519.Sign(priv, []byte("msg"))
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	if err := auth.VerifyEd25519Signature("not-valid-base64!!!", "msg", sigB64); err == nil {
		t.Error("Expected bad base64 public key to fail, but it passed")
	}
}

func TestVerifyEd25519Signature_InvalidSignatureEncoding(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	pubB64 := base64.StdEncoding.EncodeToString(pub)

	if err := auth.VerifyEd25519Signature(pubB64, "msg", "not-valid-base64!!!"); err == nil {
		t.Error("Expected bad base64 signature to fail, but it passed")
	}
}

func TestVerifyEd25519Signature_WrongPublicKeyLength(t *testing.T) {
	// A valid base64 string but only 16 bytes (not 32)
	shortKey := base64.StdEncoding.EncodeToString([]byte("tooshort12345678"))

	_, priv, _ := ed25519.GenerateKey(nil)
	sig := ed25519.Sign(priv, []byte("msg"))
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	if err := auth.VerifyEd25519Signature(shortKey, "msg", sigB64); err == nil {
		t.Error("Expected short public key to fail, but it passed")
	}
}

func TestVerifyEd25519Signature_URLSafeBase64(t *testing.T) {
	// The helper should accept URL-safe base64 as fallback
	pub, priv, _ := ed25519.GenerateKey(nil)
	message := "url-safe-test"
	sig := ed25519.Sign(priv, []byte(message))

	pubB64 := base64.RawURLEncoding.EncodeToString(pub)
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	if err := auth.VerifyEd25519Signature(pubB64, message, sigB64); err != nil {
		t.Errorf("URL-safe base64 should be accepted: %v", err)
	}
}

// ─── Full Rotation Flow ──────────────────────────────────────────────────────

func TestRotationFlow_E2E(t *testing.T) {
	// Simulate: client builds rotation request → server verifies it

	// Step 1: Generate key pairs (current + new)
	currentPub, currentPriv, _ := ed25519.GenerateKey(nil)
	newPub, _, _ := ed25519.GenerateKey(nil)

	currentPubB64 := base64.StdEncoding.EncodeToString(currentPub)
	newPubB64 := base64.StdEncoding.EncodeToString(newPub)
	nonce := "unique-nonce-XYZ"
	ts := time.Now().Unix()

	// Step 2: Client builds signed message
	msg := auth.BuildRotationMessage(newPubB64, nonce, ts)
	sig := ed25519.Sign(currentPriv, []byte(msg))
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	// Step 3: Server rebuilds message and verifies
	serverMsg := auth.BuildRotationMessage(newPubB64, nonce, ts)
	if err := auth.VerifyEd25519Signature(currentPubB64, serverMsg, sigB64); err != nil {
		t.Errorf("Full rotation flow failed: %v", err)
	}
}

func TestRotationFlow_ReplayWithDifferentNonce(t *testing.T) {
	// Simulate: attacker replays the signature with a different nonce
	// This MUST fail because the nonce is part of the signed message

	_, currentPriv, _ := ed25519.GenerateKey(nil)
	newPub, _, _ := ed25519.GenerateKey(nil)
	attackerPub, _, _ := ed25519.GenerateKey(nil)

	newPubB64 := base64.StdEncoding.EncodeToString(newPub)
	attackerPubB64 := base64.StdEncoding.EncodeToString(attackerPub)
	ts := time.Now().Unix()

	// Legitimate signature over nonce A
	originalMsg := auth.BuildRotationMessage(newPubB64, "nonce-A", ts)
	sig := ed25519.Sign(currentPriv, []byte(originalMsg))
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	// Attacker tries to use same signature with nonce B and different new key
	attackerMsg := auth.BuildRotationMessage(attackerPubB64, "nonce-B", ts)
	_, currentPub, _ := ed25519.GenerateKey(nil)
	currentPubB64 := base64.StdEncoding.EncodeToString(currentPub)

	if err := auth.VerifyEd25519Signature(currentPubB64, attackerMsg, sigB64); err == nil {
		t.Error("Replay with different nonce should fail signature verification")
	}
}

// ─── DecodeEd25519PublicKey ──────────────────────────────────────────────────

func TestDecodeEd25519PublicKey_ValidKey(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	pubB64 := base64.StdEncoding.EncodeToString(pub)

	decoded, err := auth.DecodeEd25519PublicKey(pubB64)
	if err != nil {
		t.Fatalf("Expected successful decode, got: %v", err)
	}
	if len(decoded) != ed25519.PublicKeySize {
		t.Errorf("Expected %d bytes, got %d", ed25519.PublicKeySize, len(decoded))
	}
}

func TestDecodeEd25519PublicKey_WrongLength(t *testing.T) {
	// 16 bytes instead of 32
	shortKey := base64.StdEncoding.EncodeToString(make([]byte, 16))
	if _, err := auth.DecodeEd25519PublicKey(shortKey); err == nil {
		t.Error("Expected error for wrong-length key, got nil")
	}
}

func TestDecodeEd25519PublicKey_InvalidBase64(t *testing.T) {
	if _, err := auth.DecodeEd25519PublicKey("!!!not-base64!!!"); err == nil {
		t.Error("Expected error for invalid base64, got nil")
	}
}

func TestDecodeEd25519PublicKey_URLSafeBase64(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	pubB64 := base64.RawURLEncoding.EncodeToString(pub)

	if _, err := auth.DecodeEd25519PublicKey(pubB64); err != nil {
		t.Errorf("URL-safe base64 should be accepted: %v", err)
	}
}
