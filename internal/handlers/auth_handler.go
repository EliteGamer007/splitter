package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"splitter/internal/auth"
	"splitter/internal/models"
	"splitter/internal/repository"
	"splitter/internal/service"

	"github.com/labstack/echo/v4"
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

	contentType := c.Request().Header.Get("Content-Type")
	var avatarBytes []byte
	var avatarMediaType string

	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := c.Request().ParseMultipartForm(6 * 1024 * 1024); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid multipart form data",
			})
		}

		multipartForm, _ := c.MultipartForm()

		formValue := func(keys ...string) string {
			for _, key := range keys {
				if multipartForm != nil {
					if values, ok := multipartForm.Value[key]; ok && len(values) > 0 {
						value := strings.TrimSpace(values[0])
						if value != "" {
							return value
						}
					}
				}
				if value := strings.TrimSpace(c.FormValue(key)); value != "" {
					return value
				}
				if value := strings.TrimSpace(c.Request().PostFormValue(key)); value != "" {
					return value
				}
			}
			return ""
		}

		req = models.UserCreate{
			Username:            formValue("username", "userName"),
			Email:               formValue("email"),
			Password:            formValue("password"),
			DisplayName:         formValue("display_name", "displayName"),
			Bio:                 formValue("bio"),
			InstanceDomain:      formValue("instance_domain", "instanceDomain"),
			DID:                 formValue("did"),
			PublicKey:           formValue("public_key", "publicKey"),
			EncryptionPublicKey: formValue("encryption_public_key", "encryptionPublicKey"),
		}

		avatarFile, fileErr := c.FormFile("avatar")
		if fileErr == http.ErrMissingFile {
			avatarFile, fileErr = c.FormFile("file")
		}
		if fileErr == nil {
			var err error
			avatarBytes, avatarMediaType, err = service.ReadAndValidateImage(avatarFile, 5*1024*1024)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "Invalid avatar image: " + err.Error(),
				})
			}
		}
	} else {
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request body",
			})
		}
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	req.Password = strings.TrimSpace(req.Password)
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	req.Bio = strings.TrimSpace(req.Bio)
	req.InstanceDomain = strings.TrimSpace(req.InstanceDomain)
	req.DID = strings.TrimSpace(req.DID)
	req.PublicKey = strings.TrimSpace(req.PublicKey)
	req.EncryptionPublicKey = strings.TrimSpace(req.EncryptionPublicKey)

	// Validate using model method
	if err := req.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	// Check if username already exists
	exists, err := h.userRepo.UsernameExists(c.Request().Context(), req.Username)
	if err != nil {
		log.Printf("UsernameExists DB error: %v", err)
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
		log.Printf("EmailExists DB error: %v", err)
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
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to hash password",
		})
	}

	// Auto-generate DID if not provided (optional for basic users)
	if req.DID == "" {
		req.DID = auth.GenerateSimpleDID(req.Username)
	}

	// Set display name to username if not provided
	if req.DisplayName == "" {
		req.DisplayName = req.Username
	}

	// Create user
	user, err := h.userRepo.Create(c.Request().Context(), &req, passwordHash)
	if err != nil {
		log.Printf("Create User DB error: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create user: " + err.Error(),
		})
	}

	if len(avatarBytes) > 0 {
		updatedUser, err := h.userRepo.UpdateAvatar(c.Request().Context(), user.ID, avatarBytes, avatarMediaType)
		if err != nil {
			log.Printf("Update Avatar DB error: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "User created but failed to save avatar",
			})
		}
		user = updatedUser
	}

	// Generate JWT token
	token, err := auth.GenerateToken(user.ID, user.DID, user.Username, user.Role, h.jwtSecret)
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
	if !auth.CheckPasswordHash(req.Password, passwordHash) {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid username or password",
		})
	}

	// Generate JWT token
	token, err := auth.GenerateToken(user.ID, user.DID, user.Username, user.Role, h.jwtSecret)
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
	token, err := auth.GenerateToken(user.ID, user.DID, user.Username, user.Role, h.jwtSecret)
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
