package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"splitter/internal/auth"
	"splitter/internal/config"
	"splitter/internal/federation"
	"splitter/internal/models"
	"splitter/internal/repository"
	"splitter/internal/service"

	"github.com/labstack/echo/v4"
)

type authUserRepository interface {
	RegisterInitialKey(context.Context, string, string) error
	Create(ctx context.Context, user *models.UserCreate, passwordHash string) (*models.User, error)
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, string, error)
	GetByDID(ctx context.Context, did string) (*models.User, error)
	UsernameExists(ctx context.Context, username string) (bool, error)
	EmailExists(ctx context.Context, email string) (bool, error)
	UpdateAvatar(ctx context.Context, userID string, avatarData []byte, mediaType, avatarURL string) (*models.User, error)
	RotateKey(ctx context.Context, userID, oldPublicKey, newPublicKey, nonce, reason string) error
	GetKeyHistory(ctx context.Context, userID string) ([]*models.KeyRotation, error)
	GetKeyHistoryByDID(ctx context.Context, did string) ([]*models.KeyRotation, error)
	IsKeyRevoked(ctx context.Context, publicKey string) (bool, error)
	RevokeCurrentKey(ctx context.Context, userID, oldKey string) error
	ListDeviceKeys(ctx context.Context, userID string) ([]*models.DeviceKey, error)
	ListApprovedDeviceKeysByDID(ctx context.Context, did string) ([]*models.DeviceKey, error)
	RequestDeviceKey(ctx context.Context, userID, deviceID, deviceLabel, encryptionPublicKey string) (*models.DeviceKey, error)
	ApproveDeviceKey(ctx context.Context, userID, targetDeviceID, approverDeviceID string) (*models.DeviceKey, error)
}

// AuthHandler handles authentication-related requests
type AuthHandler struct {
	userRepo   authUserRepository
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
			avatarBytes, avatarMediaType, err = service.ReadAndValidateAvatar(avatarFile, 5*1024*1024)
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
			"error": "Failed to create user",
		})
	}

	if len(avatarBytes) > 0 {
		updatedUser, err := h.userRepo.UpdateAvatar(c.Request().Context(), user.ID, avatarBytes, avatarMediaType, "/api/v1/users/"+user.ID+"/avatar")
		if err != nil {
			log.Printf("Update Avatar DB error: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "User created but failed to save avatar",
			})
		}
		user = updatedUser
	}

	// Generate JWT token
	token, err := auth.GenerateToken(user.ID, user.DID, user.Username, user.Role, h.jwtSecret, h.tokenLifetime())
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
	token, err := auth.GenerateToken(user.ID, user.DID, user.Username, user.Role, h.jwtSecret, h.tokenLifetime())
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
	req.DID = strings.TrimSpace(req.DID)
	if req.DID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "DID is required"})
	}
	user, err := h.userRepo.GetByDID(c.Request().Context(), req.DID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "DID not found",
		})
	}

	if user.IsSuspended {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Account is suspended"})
	}
	if _, err := auth.DecodeEd25519PublicKey(user.PublicKey); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Account has no usable signing key"})
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
	h.cleanupExpiredChallenges(time.Now())
	if len(h.challenges) >= 4096 && h.challenges[req.DID] == nil {
		h.mu.Unlock()
		return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "Authentication capacity reached; retry shortly"})
	}
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

	if req.DID == "" || req.Challenge == "" || req.Signature == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "A complete signed challenge is required"})
	}
	// Snapshot under the lock, but do not hold it during the database lookup.
	h.mu.RLock()
	storedChallenge, exists := h.challenges[req.DID]
	h.mu.RUnlock()
	if !exists || storedChallenge.Challenge != req.Challenge || !time.Now().Before(storedChallenge.ExpiresAt) {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid or expired challenge"})
	}
	user, err := h.userRepo.GetByDID(c.Request().Context(), req.DID)
	if err != nil || user == nil || user.DID != req.DID {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid authentication proof"})
	}
	if user.IsSuspended {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Account is suspended"})
	}
	if err := auth.VerifyEd25519Signature(user.PublicKey, req.Challenge, req.Signature); err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid authentication proof"})
	}
	// Compare and consume atomically. Concurrent verifications, expiration, and a
	// newer challenge must never allow the same nonce to mint a second token.
	h.mu.Lock()
	current, exists := h.challenges[req.DID]
	if !exists || current != storedChallenge || !time.Now().Before(current.ExpiresAt) {
		h.mu.Unlock()
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid or expired challenge"})
	}
	delete(h.challenges, req.DID)
	h.mu.Unlock()

	// Generate JWT token
	token, err := auth.GenerateToken(user.ID, user.DID, user.Username, user.Role, h.jwtSecret, h.tokenLifetime())
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

// tokenLifetime uses the same configured lifetime for all login methods.
func (h *AuthHandler) tokenLifetime() time.Duration {
	if h.cfg != nil && h.cfg.JWT.Expiration > 0 {
		return time.Duration(h.cfg.JWT.Expiration) * time.Hour
	}
	return 24 * time.Hour
}

// cleanupExpiredChallenges is request-triggered: idle instances do no work.
// The caller holds h.mu.
func (h *AuthHandler) cleanupExpiredChallenges(now time.Time) {
	for did, challenge := range h.challenges {
		if !now.Before(challenge.ExpiresAt) {
			delete(h.challenges, did)
		}
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
	var req struct {
		models.KeyRotationRequest
		CurrentPassword string `json:"current_password"`
	}
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

	// Check timestamp expiration (5 minute window)
	if req.Timestamp == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "timestamp is required",
		})
	}
	now := time.Now().Unix()
	if now-req.Timestamp > 300 || req.Timestamp-now > 300 {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Request timestamp is stale or too far in the future",
		})
	}

	// Use a nonce derived from the new key to record rotation (replay safety kept)
	rotationNonce := req.Nonce
	if rotationNonce == "" {
		rotationNonce = req.NewPublicKey // fallback unique identifier
	}

	// Verify the signature is valid. The client must sign: "{new_public_key}|{nonce}|{timestamp}"
	// using their CURRENT (old) active private key.
	if req.Signature == "" && req.CurrentPassword == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "signature is required",
		})
	}

	// Reconstruct the message that was signed
	msg := fmt.Sprintf("%s|%s|%d", req.NewPublicKey, rotationNonce, req.Timestamp)

	// Verify using the old public key
	err = auth.VerifyEd25519Signature(user.PublicKey, msg, req.Signature)
	// Explicit password proof recovers legacy nonstandard browser signing keys.
	// A session token alone never authorizes replacing an existing signing key.
	if err != nil && req.CurrentPassword != "" {
		account, hash, passwordErr := h.userRepo.GetByUsername(c.Request().Context(), user.Username)
		if passwordErr == nil && account != nil && account.ID == userID && !account.IsSuspended && auth.CheckPasswordHash(req.CurrentPassword, hash) {
			err = nil
		}
	}
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid signing proof or current password.",
		})
	}

	// Perform the rotation atomically
	rotationReason := req.Reason
	if rotationReason == "" {
		rotationReason = "rotated"
	}
	if err := h.userRepo.RotateKey(c.Request().Context(), userID, user.PublicKey, req.NewPublicKey, rotationNonce, rotationReason); err != nil {
		if errors.Is(err, repository.ErrKeyConflict) {
			return c.JSON(http.StatusConflict, map[string]string{"error": "Signing key changed; reload before rotating"})
		}
		log.Printf("RotateKey DB error for user %s: %v", userID, err)
		// Check if it's a unique constraint violation on the nonce
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint \"key_rotations_nonce_key\"") {
			return c.JSON(http.StatusConflict, map[string]string{
				"error": "Nonce has already been used for rotation. Request rejected to prevent replay attack.",
			})
		}
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

	// Compare-and-set in the repository closes the check/write race between
	// concurrent key registrations from different devices.
	if err := h.userRepo.RegisterInitialKey(c.Request().Context(), userID, req.PublicKey); err != nil {
		if errors.Is(err, repository.ErrKeyConflict) {
			return c.JSON(http.StatusConflict, map[string]string{"error": "Account signing key changed; reload before continuing"})
		}
		log.Printf("RegisterKey DB error for user %s: %v", userID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to register key"})
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

// RequestDeviceKey registers (or refreshes) a secondary device encryption key.
// First device for a user is bootstrap-approved; subsequent devices require approval.
// Endpoint: POST /api/v1/auth/devices/request (authenticated)
func (h *AuthHandler) RequestDeviceKey(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	var req struct {
		DeviceID            string `json:"device_id"`
		DeviceLabel         string `json:"device_label"`
		EncryptionPublicKey string `json:"encryption_public_key"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	req.DeviceID = strings.TrimSpace(req.DeviceID)
	req.DeviceLabel = strings.TrimSpace(req.DeviceLabel)
	req.EncryptionPublicKey = strings.TrimSpace(req.EncryptionPublicKey)

	if req.DeviceID == "" || req.EncryptionPublicKey == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "device_id and encryption_public_key are required",
		})
	}

	deviceKey, err := h.userRepo.RequestDeviceKey(c.Request().Context(), userID, req.DeviceID, req.DeviceLabel, req.EncryptionPublicKey)
	if err != nil {
		if errors.Is(err, repository.ErrKeyConflict) {
			return c.JSON(http.StatusConflict, map[string]string{"error": "Device already has a different key; import its recovery material or use another device ID"})
		}
		log.Printf("RequestDeviceKey error for user %s: %v", userID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to register device key"})
	}

	statusCode := http.StatusAccepted
	if deviceKey.Status == "approved" {
		statusCode = http.StatusCreated
	}

	return c.JSON(statusCode, map[string]interface{}{
		"message":    "device key registered",
		"device_key": deviceKey,
	})
}

// ApproveDeviceKey approves a pending device key using an already-authorized device.
// Endpoint: POST /api/v1/auth/devices/:deviceId/approve (authenticated)
func (h *AuthHandler) ApproveDeviceKey(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	targetDeviceID := strings.TrimSpace(c.Param("deviceId"))
	if targetDeviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "deviceId is required"})
	}

	var req struct {
		ApproverDeviceID string `json:"approver_device_id"`
		Signature        string `json:"signature"`
		Timestamp        int64  `json:"timestamp"`
		Password         string `json:"password"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	req.ApproverDeviceID = strings.TrimSpace(req.ApproverDeviceID)
	if req.ApproverDeviceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "approver_device_id is required"})
	}

	if req.ApproverDeviceID == targetDeviceID {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "approver and target devices must be different"})
	}

	user, err := h.userRepo.GetByID(c.Request().Context(), userID)
	if err != nil || user == nil || user.IsSuspended {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Account unavailable"})
	}
	keys, err := h.userRepo.ListDeviceKeys(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Device keys temporarily unavailable"})
	}
	var target *models.DeviceKey
	approverApproved := false
	for _, key := range keys {
		if key.DeviceID == targetDeviceID && key.Status == "pending" {
			target = key
		}
		if key.DeviceID == req.ApproverDeviceID && key.Status == "approved" {
			approverApproved = true
		}
	}
	if target == nil || !approverApproved {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "A pending target and approved authorizing device are required"})
	}
	// A caller-supplied device ID is not proof of possession. Require a fresh
	// signature from the account signing key, or explicit password reauthentication.
	now := time.Now().Unix()
	if req.Timestamp == 0 || now-req.Timestamp > 300 || req.Timestamp-now > 60 {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "A fresh approval proof is required"})
	}
	validProof := false
	if req.Signature != "" {
		message := fmt.Sprintf("approve-device|%s|%s|%s|%s|%d", userID, targetDeviceID, req.ApproverDeviceID, target.EncryptionPublicKey, req.Timestamp)
		validProof = auth.VerifyEd25519Signature(user.PublicKey, message, req.Signature) == nil
	} else if req.Password != "" {
		account, hash, err := h.userRepo.GetByUsername(c.Request().Context(), user.Username)
		validProof = err == nil && account != nil && account.ID == userID && !account.IsSuspended && auth.CheckPasswordHash(req.Password, hash)
	}
	if !validProof {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Device approval requires a valid account signature or current password"})
	}
	deviceKey, err := h.userRepo.ApproveDeviceKey(c.Request().Context(), userID, targetDeviceID, req.ApproverDeviceID)
	if err != nil {
		errText := strings.ToLower(err.Error())
		switch {
		case strings.Contains(errText, "not found"):
			return c.JSON(http.StatusNotFound, map[string]string{"error": err.Error()})
		case strings.Contains(errText, "not authorized"):
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		default:
			log.Printf("ApproveDeviceKey error for user %s: %v", userID, err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to approve device key"})
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":    "device key approved",
		"device_key": deviceKey,
	})
}

// ListDeviceKeys returns all device keys (pending/approved/revoked) for the authenticated user.
// Endpoint: GET /api/v1/auth/devices (authenticated)
func (h *AuthHandler) ListDeviceKeys(c echo.Context) error {
	userID, ok := c.Get("user_id").(string)
	if !ok || userID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	keys, err := h.userRepo.ListDeviceKeys(c.Request().Context(), userID)
	if err != nil {
		log.Printf("ListDeviceKeys error for user %s: %v", userID, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to list device keys"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"device_keys": keys,
		"count":       len(keys),
	})
}

// GetPublicDeviceKeys returns approved device encryption keys for a DID.
// Endpoint: GET /api/v1/dids/:did/device-keys (public)
func (h *AuthHandler) GetPublicDeviceKeys(c echo.Context) error {
	did := strings.TrimSpace(c.Param("did"))
	if did == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "DID is required"})
	}

	keys, err := h.userRepo.ListApprovedDeviceKeysByDID(c.Request().Context(), did)
	if err != nil {
		log.Printf("GetPublicDeviceKeys error for DID %s: %v", did, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to retrieve device keys"})
	}

	type publicKey struct {
		DeviceID            string `json:"device_id"`
		DeviceLabel         string `json:"device_label,omitempty"`
		EncryptionPublicKey string `json:"encryption_public_key"`
	}

	result := make([]publicKey, 0, len(keys))
	for _, item := range keys {
		result = append(result, publicKey{
			DeviceID:            item.DeviceID,
			DeviceLabel:         item.DeviceLabel,
			EncryptionPublicKey: item.EncryptionPublicKey,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"did":         did,
		"device_keys": result,
		"count":       len(result),
	})
}
