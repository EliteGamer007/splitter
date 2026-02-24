package auth

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
)

// BuildRotationMessage creates the canonical signed message for a key rotation request.
// Format: "{newPublicKey}|{nonce}|{timestamp}"
// Both client and server must use this exact format.
func BuildRotationMessage(newPublicKey, nonce string, timestamp int64) string {
	return fmt.Sprintf("%s|%s|%d", newPublicKey, nonce, timestamp)
}

// VerifyEd25519Signature verifies an Ed25519 signature over a message.
//
// Parameters:
//   - publicKeyB64: base64-encoded Ed25519 public key (32 bytes raw)
//   - message:      the plaintext message that was signed
//   - signatureB64: base64-encoded Ed25519 signature (64 bytes raw)
//
// Returns nil if the signature is valid, an error otherwise.
func VerifyEd25519Signature(publicKeyB64, message, signatureB64 string) error {
	// Decode public key
	pubKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		// Try URL-safe base64 as fallback
		pubKeyBytes, err = base64.RawURLEncoding.DecodeString(publicKeyB64)
		if err != nil {
			return fmt.Errorf("invalid public key encoding: %w", err)
		}
	}
	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key length: expected %d bytes, got %d", ed25519.PublicKeySize, len(pubKeyBytes))
	}

	// Decode signature
	sigBytes, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		sigBytes, err = base64.RawURLEncoding.DecodeString(signatureB64)
		if err != nil {
			return fmt.Errorf("invalid signature encoding: %w", err)
		}
	}
	if len(sigBytes) != ed25519.SignatureSize {
		return fmt.Errorf("invalid signature length: expected %d bytes, got %d", ed25519.SignatureSize, len(sigBytes))
	}

	// Verify
	pubKey := ed25519.PublicKey(pubKeyBytes)
	if !ed25519.Verify(pubKey, []byte(message), sigBytes) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// DecodeEd25519PublicKey decodes a base64-encoded Ed25519 public key.
// Accepts both standard and URL-safe base64.
func DecodeEd25519PublicKey(publicKeyB64 string) (ed25519.PublicKey, error) {
	pubKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		pubKeyBytes, err = base64.RawURLEncoding.DecodeString(publicKeyB64)
		if err != nil {
			return nil, fmt.Errorf("invalid public key encoding: %w", err)
		}
	}
	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid Ed25519 public key: expected %d bytes, got %d", ed25519.PublicKeySize, len(pubKeyBytes))
	}
	return ed25519.PublicKey(pubKeyBytes), nil
}
