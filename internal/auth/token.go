package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateToken generates a JWT token for a user
func GenerateToken(userID, did, username, role, secret string) (string, error) {
	if secret == "" {
		return "", errors.New("JWT secret cannot be empty")
	}

	if role == "" {
		role = "user"
	}

	claims := jwt.MapClaims{
		"sub":      userID,
		"did":      did,
		"username": username,
		"role":     role,
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days
		"iat":      time.Now().Unix(),
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
