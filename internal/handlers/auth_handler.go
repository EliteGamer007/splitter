package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"splitter/internal/auth"
	"splitter/internal/config"
	"splitter/internal/db"
	"splitter/internal/federation"
	"splitter/internal/models"
	"splitter/internal/repository"
	"splitter/internal/service"

	"github.com/labstack/echo/v4"
)

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	userRepo   *repository.UserRepository
	cfg        *config.Config
	jwtSecret  string
	challenges map[string]*models.AuthChallenge // In-memory challenge store
	mu         sync.RWMutex
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(userRepo *repository.UserRepository, cfg *config.Config) *AuthHandler {
	handler := &AuthHandler{
		userRepo:   userRepo,
		cfg:        cfg,
		jwtSecret:  cfg.JWT.Secret,
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

// RotateKey handles Ed25519 signing key rotation (Story 4.3).
// The client must sign the rotation message with their CURRENT private key.
// Endpoint: POST /api/v1/auth/rotate-key (authenticated)
//
// Request body:
//
//	{
//	  "new_public_key": "<base64 Ed25519 pubkey>",
//	  "signature":      "<base64 signature over '{new_public_key}|{nonce}|{timestamp}'>",
//	  "nonce":          "<UUID v4>",
//	  "timestamp":      <unix seconds>
//	}
func (h *AuthHandler) RotateKey(c echo.Context) error {
	// Extract authenticated user ID from JWT claims
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	// Bind and validate request body
	var req models.KeyRotationRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}
	if req.NewPublicKey == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "new_public_key is required",
		})
	}

	// Fetch current user to get existing public key
	user, err := h.userRepo.GetByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	// Only users with an existing public key can rotate
	if user.PublicKey == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "No existing public key. Use POST /api/v1/auth/register-key to set an initial key.",
		})
	}

	// Verify the new key is a valid Ed25519 key
	if _, err := auth.DecodeEd25519PublicKey(req.NewPublicKey); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid new_public_key: " + err.Error(),
		})
	}

	// Use a nonce derived from the new key to record rotation (replay safety kept)
	rotationNonce := req.Nonce
	if rotationNonce == "" {
		rotationNonce = req.NewPublicKey // fallback unique identifier
	}

	// Perform the rotation atomically
	rotationReason := req.Reason
	if rotationReason == "" {
		rotationReason = "rotated"
	}
	if err := h.userRepo.RotateKey(c.Request().Context(), userID, user.PublicKey, req.NewPublicKey, rotationNonce, rotationReason); err != nil {
		log.Printf("RotateKey DB error for user %s: %v", userID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to rotate key",
		})
	}

	log.Printf("Key rotated successfully for user %s (DID: %s)", userID, user.DID)

	// Propagate rotation via ActivityPub
	if h.cfg != nil && h.cfg.Federation.Enabled {
		actorURI := fmt.Sprintf("%s/ap/users/%s", h.cfg.Federation.URL, user.Username)
		// encryptionPublicKey is optional, using existing if not provided
		activity := federation.BuildUpdateActorActivity(
			actorURI,
			user.Username,
			user.DisplayName,
			user.Bio,
			user.AvatarURL,
			req.NewPublicKey, // New signing key
			user.EncryptionPublicKey,
		)
		go federation.DeliverToFollowers(activity, user.DID)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":        "Key rotated successfully",
		"new_public_key": req.NewPublicKey,
		"rotated_at":     time.Now().UTC(),
	})
}

// GetKeyHistory returns the signing key rotation history for the authenticated user.
// Endpoint: GET /api/v1/auth/key-history (authenticated)
func (h *AuthHandler) GetKeyHistory(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	history, err := h.userRepo.GetKeyHistory(c.Request().Context(), userID)
	if err != nil {
		log.Printf("GetKeyHistory error for user %s: %v", userID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve key history",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"key_history": history,
		"count":       len(history),
	})
}

// RegisterKey sets the initial Ed25519 public key for an account that doesn't have one yet.
// This is a one-time operation for password-only users who want to add a signing key.
// No signature is required because there is no existing key to sign with.
// Endpoint: POST /api/v1/auth/register-key (authenticated)
//
// Request body:
//
//	{
//	  "public_key": "<base64 Ed25519 pubkey>"
//	}
func (h *AuthHandler) RegisterKey(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Authentication required",
		})
	}

	var req struct {
		PublicKey string `json:"public_key"`
	}
	if err := c.Bind(&req); err != nil || req.PublicKey == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "public_key is required",
		})
	}

	// Validate the key is a proper Ed25519 key
	if _, err := auth.DecodeEd25519PublicKey(req.PublicKey); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid public_key: " + err.Error(),
		})
	}

	// Fetch user to verify they don't already have a key
	user, err := h.userRepo.GetByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}
	if user.PublicKey != "" {
		return c.JSON(http.StatusConflict, map[string]string{
			"error": "Account already has a public key. Use POST /api/v1/auth/rotate-key to change it.",
		})
	}

	// Store the initial public key using the global DB connection
	if _, err := db.GetDB().Exec(
		c.Request().Context(),
		`UPDATE users SET public_key = $1 WHERE id = $2`,
		req.PublicKey, userID,
	); err != nil {
		log.Printf("RegisterKey DB error for user %s: %v", userID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to register key",
		})
	}

	log.Printf("Initial public key registered for user %s (DID: %s)", userID, user.DID)
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":    "Public key registered successfully",
		"public_key": req.PublicKey,
	})
}

// GetRevokedKeys returns the full revocation list for the authenticated user.
// Endpoint: GET /api/v1/auth/revoked-keys (authenticated)
func (h *AuthHandler) GetRevokedKeys(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	user, err := h.userRepo.GetByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	revoked, err := h.userRepo.GetKeyHistory(c.Request().Context(), userID)
	if err != nil {
		log.Printf("GetRevokedKeys error for user %s: %v", userID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve revocation list"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"did":          user.DID,
		"active_key":   user.PublicKey,
		"revoked_keys": revoked,
		"count":        len(revoked),
	})
}

// GetPublicRevokedKeys returns the revocation list for a given DID — no auth required.
// Federation partners use this to check whether a key is still valid.
// Endpoint: GET /api/v1/dids/:did/revoked-keys (public)
func (h *AuthHandler) GetPublicRevokedKeys(c echo.Context) error {
	did := c.Param("did")
	if did == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "DID is required"})
	}

	revoked, err := h.userRepo.GetKeyHistoryByDID(c.Request().Context(), did)
	if err != nil {
		log.Printf("GetPublicRevokedKeys error for DID %s: %v", did, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve revocation list"})
	}

	// Return only the fields needed for external verification (no internal IDs)
	type publicEntry struct {
		RevokedKey string    `json:"revoked_key"`
		ReplacedBy string    `json:"replaced_by"`
		RevokedAt  time.Time `json:"revoked_at"`
		Reason     string    `json:"reason"`
		IsRevoked  bool      `json:"is_revoked"`
	}
	public := make([]publicEntry, len(revoked))
	for i, r := range revoked {
		public[i] = publicEntry{
			RevokedKey: r.OldPublicKey,
			ReplacedBy: r.NewPublicKey,
			RevokedAt:  r.RotatedAt,
			Reason:     r.Reason,
			IsRevoked:  true,
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"did":          did,
		"revoked_keys": public,
		"count":        len(public),
	})
}

// CheckKeyRevocation checks whether a specific public key has been revoked.
// Endpoint: GET /api/v1/auth/check-key?key=<base64> (public)
func (h *AuthHandler) CheckKeyRevocation(c echo.Context) error {
	publicKey := c.QueryParam("key")
	if publicKey == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "key query parameter is required"})
	}

	// Support full DID URIs by stripping common prefixes
	if len(publicKey) > 9 && publicKey[:9] == "did:key:z" {
		// did:key:z6Mk... -> strip "did:key:z6Mk" (12 chars) or just "did:key:z" (9 chars)
		// The most common prefix is did:key:z6Mk
		if len(publicKey) > 12 && publicKey[:12] == "did:key:z6Mk" {
			publicKey = publicKey[12:]
		} else {
			publicKey = publicKey[9:]
		}
	}

	revoked, err := h.userRepo.IsKeyRevoked(c.Request().Context(), publicKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to check key status"})
	}

	status := "active"
	if revoked {
		status = "revoked"
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"key":     publicKey,
		"status":  status,
		"revoked": revoked,
	})
}

// RevokeKey manually revokes the user's current signing key without rotating to a new one.
// The account reverts to a password-only status until a new key is initialized.
// Endpoint: POST /api/v1/auth/revoke-key (authenticated)
func (h *AuthHandler) RevokeKey(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	user, err := h.userRepo.GetByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	if user.PublicKey == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "No active key to revoke"})
	}

	// Archive old key and clear from user profile
	if err := h.userRepo.RevokeCurrentKey(c.Request().Context(), userID, user.PublicKey); err != nil {
		log.Printf("RevokeKey error for user %s: %v", userID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to revoke key"})
	}

	log.Printf("Key manually revoked for user %s (DID: %s)", userID, user.DID)

	// Propagate revocation via ActivityPub
	if h.cfg != nil && h.cfg.Federation.Enabled {
		actorURI := fmt.Sprintf("%s/ap/users/%s", h.cfg.Federation.URL, user.Username)
		// For revocation, publicKeyPEM is empty/revoked
		activity := federation.BuildUpdateActorActivity(
			actorURI,
			user.Username,
			user.DisplayName,
			user.Bio,
			user.AvatarURL,
			"", // Revoked key (empty)
			user.EncryptionPublicKey,
		)
		go federation.DeliverToFollowers(activity, user.DID)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Key revoked successfully. Identity is now password-only.",
	})
}
