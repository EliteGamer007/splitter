// Package auth_test — logger bridge for unit tests.
package auth_test

import (
	"splitter/internal/auth"
	"strings"
	"testing"
	"time"

	"splitter/tests/testlogger"

	"github.com/golang-jwt/jwt/v5"
)

func TestHashPasswordLogged(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "unit", start, testErr) }()

	password := "securepassword123"
	hash1, err := auth.HashPassword(password)
	if err != nil {
		testErr = err
		t.Fatalf("Failed to hash password: %v", err)
	}
	hash2, err := auth.HashPassword(password)
	if err != nil {
		testErr = err
		t.Fatalf("Failed to hash password second time: %v", err)
	}
	if hash1 == hash2 {
		t.Error("Hashing should produce different outputs (salting)")
	}
	if !auth.CheckPasswordHash(password, hash1) {
		t.Error("CheckPasswordHash failed for valid password")
	}
	if auth.CheckPasswordHash("wrongpassword", hash1) {
		t.Error("CheckPasswordHash succeeded for invalid password")
	}
}

func TestGenerateTokenLogged(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "unit", start, testErr) }()

	secret := "test-secret-key"
	userID := "user-123"
	did := "did:splitter:user-123"
	username := "testuser"
	role := "admin"

	tokenString, err := auth.GenerateToken(userID, did, username, role, secret)
	if err != nil {
		testErr = err
		t.Fatalf("Failed to generate token: %v", err)
	}
	if tokenString == "" {
		t.Fatal("Generated token is empty")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		testErr = err
		t.Fatalf("Failed to parse token: %v", err)
	}
	if !token.Valid {
		t.Fatal("Token is invalid")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("Failed to parse claims")
	}
	if claims["sub"] != userID {
		t.Errorf("Expected sub %s, got %v", userID, claims["sub"])
	}
	exp, ok := claims["exp"].(float64)
	if !ok {
		t.Fatal("exp claim missing")
	}
	if time.Unix(int64(exp), 0).Before(time.Now()) {
		t.Error("Token is already expired")
	}
}

func TestGenerateSimpleDIDLogged(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "unit", start, testErr) }()

	username := "testuser"
	did := auth.GenerateSimpleDID(username)
	if !strings.HasPrefix(did, "did:splitter:testuser-") {
		t.Errorf("DID %s does not match expected format", did)
	}
}
