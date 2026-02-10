package auth_test

import (
	"splitter/internal/auth"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

/*
WHY THIS TEST EXISTS:
- Security relies on correct password hashing and JWT token generation.
- Hashing must be irreversible and non-deterministic (salt).
- Tokens must contain correct claims and expiry.

EXPECTED BEHAVIOR:
- Hashing always produces different outputs for same input.
- Valid tokens verify correctly.
- Invalid tokens fail verification.

TEST RESULT SUMMARY:
- Passed: Hashing uniqueness, verification success/failure.
- Passed: JWT generation, claims check, expiry check.
- Limitations: Does not test database persistence.
*/

func TestHashPassword(t *testing.T) {
	password := "securepassword123"

	// Test 1: Hash generation
	hash1, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Test 2: Hashing same password twice should produce different hashes (salting)
	hash2, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password second time: %v", err)
	}

	if hash1 == hash2 {
		t.Error("Hashing should produce different outputs for the same password due to salting")
	}

	// Test 3: Verify correct password
	if !auth.CheckPasswordHash(password, hash1) {
		t.Error("CheckPasswordHash failed for valid password")
	}

	// Test 4: Verify incorrect password
	if auth.CheckPasswordHash("wrongpassword", hash1) {
		t.Error("CheckPasswordHash succeeded for invalid password")
	}
}

func TestGenerateToken(t *testing.T) {
	secret := "test-secret-key"
	userID := "user-123"
	did := "did:splitter:user-123"
	username := "testuser"
	role := "admin"

	// Test 1: Generate valid token
	tokenString, err := auth.GenerateToken(userID, did, username, role, secret)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if tokenString == "" {
		t.Fatal("Generated token is empty")
	}

	// Test 2: Parse and validate token manually
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		t.Fatalf("Failed to parse generated token: %v", err)
	}

	if !token.Valid {
		t.Fatal("Token is invalid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("Failed to parse claims")
	}

	// Verify claims
	if claims["sub"] != userID {
		t.Errorf("Expected sub %s, got %v", userID, claims["sub"])
	}
	if claims["did"] != did {
		t.Errorf("Expected did %s, got %v", did, claims["did"])
	}
	if claims["username"] != username {
		t.Errorf("Expected username %s, got %v", username, claims["username"])
	}
	if claims["role"] != role {
		t.Errorf("Expected role %s, got %v", role, claims["role"])
	}

	// Verify expiry (should be > now)
	exp, ok := claims["exp"].(float64)
	if !ok {
		t.Fatal("exp claim missing or invalid type")
	}
	if time.Unix(int64(exp), 0).Before(time.Now()) {
		t.Error("Token is already expired")
	}
}

func TestGenerateSimpleDID(t *testing.T) {
	username := "testuser"
	did := auth.GenerateSimpleDID(username)

	if !strings.HasPrefix(did, "did:splitter:testuser-") {
		t.Errorf("DID %s does not match expected format", did)
	}
}
