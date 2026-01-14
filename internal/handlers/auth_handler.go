package handlers

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"splitter/internal/models"
	"splitter/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// AuthHandler handles authentication-related requests with DID challenge-response
type AuthHandler struct {
	userRepo   *repository.UserRepository
	jwtSecret  string
	challenges map[string]*models.AuthChallenge // In-memory challenge store (use Redis in production)
	mu         sync.RWMutex
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(userRepo *repository.UserRepository, jwtSecret string) *AuthHandler {
	handler := &AuthHandler{
		userRepo:   userRepo,
		jwtSecret:  jwtSecret,
		challenges: make(map[string]*models.AuthChallenge),
	}

	// Start challenge cleanup goroutine
	go handler.cleanupExpiredChallenges()

	return handler
}

// Register handles user registration with DID and public key
func (h *AuthHandler) Register(c echo.Context) error {
	var req models.UserCreate
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate that DID is not already registered
	existingUser, _ := h.userRepo.GetByDID(c.Request().Context(), req.DID)
	if existingUser != nil {
		return c.JSON(http.StatusConflict, map[string]string{
			"error": "DID already registered",
		})
	}

	// Create user
	user, err := h.userRepo.Create(c.Request().Context(), &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create user",
		})
	}

	// Generate JWT token with DID as subject
	token, err := h.generateToken(user.DID, user.Username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to generate token",
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"user":  user,
		"token": token,
	})
}

// GetChallenge generates a challenge nonce for authentication
func (h *AuthHandler) GetChallenge(c echo.Context) error {
	var req models.ChallengeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Verify that the DID exists
	_, err := h.userRepo.GetByDID(c.Request().Context(), req.DID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "DID not found",
		})
	}

	// Generate random challenge nonce (32 bytes)
	challengeBytes := make([]byte, 32)
	if _, err := rand.Read(challengeBytes); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to generate challenge",
		})
	}

	challenge := base64.StdEncoding.EncodeToString(challengeBytes)
	expiresAt := time.Now().Add(5 * time.Minute) // 5 minute expiry

	// Store challenge
	h.mu.Lock()
	h.challenges[req.DID] = &models.AuthChallenge{
		DID:       req.DID,
		Challenge: challenge,
		ExpiresAt: expiresAt,
	}
	h.mu.Unlock()

	return c.JSON(http.StatusOK, models.ChallengeResponse{
		Challenge: challenge,
		ExpiresAt: expiresAt.Unix(),
	})
}

// VerifyChallenge verifies the signed challenge and issues JWT
func (h *AuthHandler) VerifyChallenge(c echo.Context) error {
	var req models.VerifyChallengeRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Retrieve stored challenge
	h.mu.RLock()
	storedChallenge, exists := h.challenges[req.DID]
	h.mu.RUnlock()

	if !exists {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Challenge not found or expired",
		})
	}

	// Check if challenge matches
	if storedChallenge.Challenge != req.Challenge {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid challenge",
		})
	}

	// Check if challenge has expired
	if time.Now().After(storedChallenge.ExpiresAt) {
		h.mu.Lock()
		delete(h.challenges, req.DID)
		h.mu.Unlock()
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Challenge expired",
		})
	}

	// Get user to retrieve public key
	user, err := h.userRepo.GetByDID(c.Request().Context(), req.DID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "User not found",
		})
	}

	// Verify signature
	publicKeyBytes, err := base64.StdEncoding.DecodeString(user.PublicKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Invalid public key format",
		})
	}

	signatureBytes, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid signature format",
		})
	}

	challengeBytes, err := base64.StdEncoding.DecodeString(req.Challenge)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid challenge format",
		})
	}

	// Verify Ed25519 signature
	if len(publicKeyBytes) != ed25519.PublicKeySize {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid public key size",
		})
	}

	publicKey := ed25519.PublicKey(publicKeyBytes)
	if !ed25519.Verify(publicKey, challengeBytes, signatureBytes) {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid signature",
		})
	}

	// Remove used challenge
	h.mu.Lock()
	delete(h.challenges, req.DID)
	h.mu.Unlock()

	// Generate JWT token with DID as subject
	token, err := h.generateToken(user.DID, user.Username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to generate token",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user":  user,
		"token": token,
	})
}

// generateToken generates a JWT token with DID as subject
func (h *AuthHandler) generateToken(did string, username string) (string, error) {
	claims := jwt.MapClaims{
		"sub":      did, // Subject is now DID instead of user_id
		"username": username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // 24 hours
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}

// cleanupExpiredChallenges removes expired challenges periodically
func (h *AuthHandler) cleanupExpiredChallenges() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		h.mu.Lock()
		now := time.Now()
		for did, challenge := range h.challenges {
			if now.After(challenge.ExpiresAt) {
				delete(h.challenges, did)
			}
		}
		h.mu.Unlock()
	}
}
