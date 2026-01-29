package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"sync"
	"time"

	"splitter/internal/models"
	"splitter/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	userRepo   *repository.UserRepository
	jwtSecret  string
	challenges map[string]*models.AuthChallenge // In-memory challenge store
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

// Register handles user registration with username/email/password
func (h *AuthHandler) Register(c echo.Context) error {
	var req models.UserCreate
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Username, email, and password are required",
		})
	}

	// Check password length
	if len(req.Password) < 8 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Password must be at least 8 characters",
		})
	}

	// Check if username already exists
	exists, err := h.userRepo.UsernameExists(c.Request().Context(), req.Username)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to check username",
		})
	}
	if exists {
		return c.JSON(http.StatusConflict, map[string]string{
			"error": "Username already taken",
		})
	}

	// Check if email already exists
	exists, err = h.userRepo.EmailExists(c.Request().Context(), req.Email)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to check email",
		})
	}
	if exists {
		return c.JSON(http.StatusConflict, map[string]string{
			"error": "Email already registered",
		})
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to hash password",
		})
	}

	// Auto-generate DID if not provided (optional for basic users)
	if req.DID == "" {
		req.DID = generateSimpleDID(req.Username)
	}

	// Set display name to username if not provided
	if req.DisplayName == "" {
		req.DisplayName = req.Username
	}

	// Create user
	user, err := h.userRepo.Create(c.Request().Context(), &req, string(passwordHash))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create user: " + err.Error(),
		})
	}

	// Generate JWT token
	token, err := h.generateToken(user.ID, user.DID, user.Username, user.Role)
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

// Login handles user login with username/email and password
func (h *AuthHandler) Login(c echo.Context) error {
	var req models.LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.Username == "" || req.Password == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Username/email and password are required",
		})
	}

	// Get user by username or email
	user, passwordHash, err := h.userRepo.GetByUsername(c.Request().Context(), req.Username)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid username or password",
		})
	}

	// Check if account is suspended
	if user.IsSuspended {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Account is suspended",
		})
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid username or password",
		})
	}

	// Generate JWT token
	token, err := h.generateToken(user.ID, user.DID, user.Username, user.Role)
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

// GetChallenge generates a challenge nonce for DID-based authentication (optional advanced auth)
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
	expiresAt := time.Now().Add(5 * time.Minute)

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

// VerifyChallenge verifies the signed challenge and issues JWT (for DID-based login)
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

	// Get user
	user, err := h.userRepo.GetByDID(c.Request().Context(), req.DID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "User not found",
		})
	}

	// Remove used challenge
	h.mu.Lock()
	delete(h.challenges, req.DID)
	h.mu.Unlock()

	// Generate JWT token
	token, err := h.generateToken(user.ID, user.DID, user.Username, user.Role)
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

// generateToken generates a JWT token
func (h *AuthHandler) generateToken(userID, did, username, role string) (string, error) {
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
	return token.SignedString([]byte(h.jwtSecret))
}

// generateSimpleDID creates a simple DID from username
func generateSimpleDID(username string) string {
	randBytes := make([]byte, 8)
	rand.Read(randBytes)
	return fmt.Sprintf("did:splitter:%s-%x", username, randBytes)
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
