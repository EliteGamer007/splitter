package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateToken generates a JWT token for a user
func GenerateToken(userID, did, username, role, secret string, lifetimes ...time.Duration) (string, error) {
	if secret == "" {
		return "", errors.New("JWT secret cannot be empty")
	}

	if role == "" {
		role = "user"
	}

	lifetime := 24 * time.Hour
	if len(lifetimes) > 0 {
		lifetime = lifetimes[0]
	}
	if lifetime <= 0 {
		return "", errors.New("JWT lifetime must be positive")
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":      userID,
		"did":      did,
		"username": username,
		"role":     role,
		"exp":      now.Add(lifetime).Unix(),
		"iat":      now.Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateSimpleDID creates a simple DID from username
func GenerateSimpleDID(username string) string {
	randBytes := make([]byte, 8)
	rand.Read(randBytes)
	return fmt.Sprintf("did:splitter:%s-%x", username, randBytes)
}
