package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"splitter/internal/config"
	"splitter/internal/db"
	"splitter/internal/federation"
	"splitter/internal/models"
	"splitter/internal/repository"
	"splitter/internal/service"

	"github.com/labstack/echo/v4"
)

type userProfileRepository interface {
	GetByID(context.Context, string) (*models.User, error)
	GetByDID(context.Context, string) (*models.User, error)
	Update(context.Context, string, *models.UserUpdate) (*models.User, error)
	UpdateAvatar(context.Context, string, []byte, string, string) (*models.User, error)
	GetAvatarContentByUserID(context.Context, string) ([]byte, string, string, error)
	Delete(context.Context, string) error
	UpdateEncryptionKey(context.Context, string, string) error
}

// UserHandler handles user-related requests
type UserHandler struct {
	userRepo   userProfileRepository
	circleRepo *repository.CircleRepository
	cfg        *config.Config
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userRepo *repository.UserRepository, cfg *config.Config) *UserHandler {
	return &UserHandler{
		userRepo:   userRepo,
		circleRepo: repository.NewCircleRepository(),
		cfg:        cfg,
	}
}

// GetProfile retrieves a user's profile by UUID
func (h *UserHandler) GetProfile(c echo.Context) error {
	id := c.Param("id") // UUID string

	user, err := h.userRepo.GetByID(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	return c.JSON(http.StatusOK, models.ToPublicUser(user))
}

// GetProfileByDID retrieves a user's profile by DID
func (h *UserHandler) GetProfileByDID(c echo.Context) error {
	did := c.QueryParam("did")
	if did == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "DID parameter required",
		})
	}

	user, err := h.userRepo.GetByDID(c.Request().Context(), did)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	return c.JSON(http.StatusOK, models.ToPublicUser(user))
}

// GetCurrentUser retrieves the authenticated user's profile
func (h *UserHandler) GetCurrentUser(c echo.Context) error {
	// Get DID from JWT token (set by auth middleware)
	did := c.Get("did").(string)

	user, err := h.userRepo.GetByDID(c.Request().Context(), did)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	return c.JSON(http.StatusOK, user)
}

// UpdateProfile updates the authenticated user's profile
func (h *UserHandler) UpdateProfile(c echo.Context) error {
	// Get DID from JWT token
	did := c.Get("did").(string)

	// First get the user to retrieve their UUID
	user, err := h.userRepo.GetByDID(c.Request().Context(), did)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	var req struct {
		DisplayName       *string `json:"display_name,omitempty"`
		Bio               *string `json:"bio,omitempty"`
		AvatarURL         *string `json:"avatar_url,omitempty"`
		DefaultVisibility *string `json:"default_visibility"`
		MessagePrivacy    *string `json:"message_privacy"`
		AccountLocked     *bool   `json:"account_locked"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	// Validate every setting before performing any profile mutation.
	if req.MessagePrivacy != nil && *req.MessagePrivacy != "everyone" && *req.MessagePrivacy != "followers" && *req.MessagePrivacy != "none" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid message_privacy value"})
	}
	if req.DefaultVisibility != nil && *req.DefaultVisibility != "public" && *req.DefaultVisibility != "followers" && *req.DefaultVisibility != "circle" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid default_visibility value"})
	}
	profileReq := &models.UserUpdate{
		DisplayName: req.DisplayName,
		Bio:         req.Bio,
		AvatarURL:   req.AvatarURL,
	}

	updatedUser, err := h.userRepo.Update(c.Request().Context(), user.ID, profileReq)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update profile",
		})
	}

	if req.DefaultVisibility != nil || req.MessagePrivacy != nil || req.AccountLocked != nil {
		defaultVisibility := ""
		if req.DefaultVisibility != nil {
			defaultVisibility = strings.TrimSpace(*req.DefaultVisibility)
		}

		messagePrivacy := ""
		if req.MessagePrivacy != nil {
			messagePrivacy = strings.TrimSpace(*req.MessagePrivacy)
		}

		var accountLocked *bool
		if req.AccountLocked != nil {
			accountLocked = req.AccountLocked
		}

		if messagePrivacy != "" && messagePrivacy != "everyone" && messagePrivacy != "followers" && messagePrivacy != "none" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid message_privacy value"})
		}

		if defaultVisibility != "" && defaultVisibility != "public" && defaultVisibility != "followers" && defaultVisibility != "circle" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid default_visibility value"})
		}

		_, err = db.GetDB().Exec(c.Request().Context(),
			`UPDATE users
			 SET default_visibility = COALESCE(NULLIF($1, ''), default_visibility),
			     message_privacy = COALESCE(NULLIF($2, ''), message_privacy),
			     is_locked = COALESCE($3, is_locked),
			     updated_at = NOW()
			 WHERE id = $4`,
			defaultVisibility,
			messagePrivacy,
			accountLocked,
			user.ID,
		)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update privacy settings"})
		}
	}

	if h.cfg != nil && h.cfg.Federation.Enabled {
		actorURI := fmt.Sprintf("%s/ap/users/%s", h.cfg.Federation.URL, user.Username)
		activity := federation.BuildUpdateActorActivity(
			actorURI,
			user.Username,
			updatedUser.DisplayName,
			updatedUser.Bio,
			updatedUser.AvatarURL,
			updatedUser.PublicKey,
			updatedUser.EncryptionPublicKey,
		)
		go federation.DeliverToFollowers(activity, user.DID)
	}

	// Return the saved settings as part of the owner's private account response.
	if refreshed, readErr := h.userRepo.GetByID(c.Request().Context(), user.ID); readErr == nil {
		updatedUser = refreshed
	}
	return c.JSON(http.StatusOK, updatedUser)
}

// UploadAvatar uploads and stores avatar image in DB for authenticated user.
func (h *UserHandler) UploadAvatar(c echo.Context) error {
	did := c.Get("did").(string)

	user, err := h.userRepo.GetByDID(c.Request().Context(), did)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	file, fileErr := c.FormFile("avatar")
	if fileErr == http.ErrMissingFile {
		file, fileErr = c.FormFile("file")
	}
	if fileErr != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Avatar file is required",
		})
	}

	avatarBytes, mediaType, err := service.ReadAndValidateAvatar(file, service.MaxImageInput)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid avatar image: %v", err),
		})
	}

	updatedUser, err := h.userRepo.UpdateAvatar(c.Request().Context(), user.ID, avatarBytes, mediaType, "")
	if err != nil {
		if errors.Is(err, db.ErrStorageQuota) {
			return c.JSON(http.StatusInsufficientStorage, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to upload avatar",
		})
	}

	if h.cfg != nil && h.cfg.Federation.Enabled {
		actorURI := fmt.Sprintf("%s/ap/users/%s", h.cfg.Federation.URL, user.Username)
		activity := federation.BuildUpdateActorActivity(
			actorURI,
			user.Username,
			updatedUser.DisplayName,
			updatedUser.Bio,
			updatedUser.AvatarURL,
			updatedUser.PublicKey,
			updatedUser.EncryptionPublicKey,
		)
		go federation.DeliverToFollowers(activity, user.DID)
	}

	return c.JSON(http.StatusOK, updatedUser)
}

// GetAvatar serves user avatar content from DB, with fallback for legacy disk URLs.
func (h *UserHandler) GetAvatar(c echo.Context) error {
	userID := c.Param("id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
	}

	avatarData, mediaType, avatarURL, err := h.userRepo.GetAvatarContentByUserID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Avatar not found"})
	}

	if len(avatarData) > 0 {
		if mediaType == "" {
			mediaType = "image/jpeg"
		}
		return c.Blob(http.StatusOK, mediaType, avatarData)
	}

	if strings.HasPrefix(avatarURL, "/uploads/") {
		legacyPath, valid := safeLegacyUploadPath(avatarURL)
		if !valid {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Avatar not found"})
		}
		legacyData, readErr := os.ReadFile(legacyPath)
		if readErr == nil && len(legacyData) > 0 {
			detectedType := http.DetectContentType(legacyData)
			return c.Blob(http.StatusOK, detectedType, legacyData)
		}
	}

	return c.JSON(http.StatusNotFound, map[string]string{"error": "Avatar content not found"})
}

// DeleteAccount deletes the authenticated user's account
func (h *UserHandler) DeleteAccount(c echo.Context) error {
	// Get DID from JWT token
	did := c.Get("did").(string)

	// First get the user to retrieve their UUID
	user, err := h.userRepo.GetByDID(c.Request().Context(), did)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	var postIDs []string
	rows, qErr := db.GetDB().Query(c.Request().Context(), `SELECT id::text FROM posts WHERE author_did = $1 AND deleted_at IS NULL`, user.DID)
	if qErr == nil {
		defer rows.Close()
		for rows.Next() {
			var postID string
			if err := rows.Scan(&postID); err == nil {
				postIDs = append(postIDs, postID)
			}
		}
	}

	if err := h.userRepo.Delete(c.Request().Context(), user.ID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete account",
		})
	}

	if h.cfg != nil && h.cfg.Federation.Enabled {
		actorURI := fmt.Sprintf("%s/ap/users/%s", h.cfg.Federation.URL, user.Username)
		for _, postID := range postIDs {
			postDelete := federation.BuildDeleteActivity(actorURI, fmt.Sprintf("%s/posts/%s", h.cfg.Federation.URL, postID))
			go federation.DeliverToFollowers(postDelete, user.DID)
		}

		accountDelete := federation.BuildDeleteActivity(actorURI, actorURI)
		go federation.DeliverToFollowers(accountDelete, user.DID)
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Account deleted successfully",
	})
}

// UpdateEncryptionKey updates the user's encryption public key
// This allows existing users without keys to generate and add them
func (h *UserHandler) UpdateEncryptionKey(c echo.Context) error {
	// Get user ID from JWT token
	userID := c.Get("user_id").(string)

	var req struct {
		EncryptionPublicKey string `json:"encryption_public_key" validate:"required"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.EncryptionPublicKey == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "encryption_public_key is required",
		})
	}

	// Update only the encryption_public_key field
	err := h.userRepo.UpdateEncryptionKey(c.Request().Context(), userID, req.EncryptionPublicKey)
	if err != nil {
		if errors.Is(err, repository.ErrInvalidEncryptionKey) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if errors.Is(err, repository.ErrKeyConflict) {
			return c.JSON(http.StatusConflict, map[string]string{"error": "This account already has an encryption key. Import your recovery keys instead of replacing it."})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update encryption key"})
	}

	if h.cfg != nil && h.cfg.Federation.Enabled {
		if user, fetchErr := h.userRepo.GetByID(c.Request().Context(), userID); fetchErr == nil && user != nil {
			actorURI := fmt.Sprintf("%s/ap/users/%s", h.cfg.Federation.URL, user.Username)
			activity := federation.BuildUpdateActorActivity(
				actorURI,
				user.Username,
				user.DisplayName,
				user.Bio,
				user.AvatarURL,
				user.PublicKey,
				req.EncryptionPublicKey,
			)
			go federation.DeliverToFollowers(activity, user.DID)
		}
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Encryption key updated successfully",
	})
}

// GetCircle returns the list of users in the authenticated user's circle.
// GET /api/v1/users/me/circle
func (h *UserHandler) GetCircle(c echo.Context) error {
	did := c.Get("did").(string)

	user, err := h.userRepo.GetByDID(c.Request().Context(), did)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	limit, offset := requestPage(c, 20)
	members, err := h.circleRepo.GetCircleMembersPage(c.Request().Context(), user.ID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get circle members"})
	}

	if members == nil {
		members = []*models.User{}
	}

	return c.JSON(http.StatusOK, models.ToPublicUsers(members))
}

// AddToCircle adds a user to the authenticated user's circle.
// POST /api/v1/users/me/circle/:id   (id = UUID of the user to add)
func (h *UserHandler) AddToCircle(c echo.Context) error {
	did := c.Get("did").(string)
	memberID := c.Param("id")
	if memberID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "User ID is required"})
	}

	owner, err := h.userRepo.GetByDID(c.Request().Context(), did)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	// Verify the target user exists and normalize ID
	targetUser, err := h.userRepo.GetByID(c.Request().Context(), memberID)
	if err != nil {
		targetUser, err = h.userRepo.GetByDID(c.Request().Context(), memberID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Target user not found"})
		}
	}
	actualMemberID := targetUser.ID

	if owner.ID == actualMemberID {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Cannot add yourself to your own circle"})
	}

	if err := h.circleRepo.AddCircleMember(c.Request().Context(), owner.ID, actualMemberID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to add to circle"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "User added to circle"})
}

// RemoveFromCircle removes a user from the authenticated user's circle.
// DELETE /api/v1/users/me/circle/:id   (id = UUID of the user to remove)
func (h *UserHandler) RemoveFromCircle(c echo.Context) error {
	did := c.Get("did").(string)
	memberID := c.Param("id")
	if memberID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "User ID is required"})
	}

	owner, err := h.userRepo.GetByDID(c.Request().Context(), did)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	// Verify the target user exists and normalize ID
	targetUser, err := h.userRepo.GetByID(c.Request().Context(), memberID)
	if err != nil {
		targetUser, err = h.userRepo.GetByDID(c.Request().Context(), memberID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Target user not found"})
		}
	}
	actualMemberID := targetUser.ID

	if err := h.circleRepo.RemoveCircleMember(c.Request().Context(), owner.ID, actualMemberID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to remove from circle"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "User removed from circle"})
}

// IsInCircle checks whether a given user is in the authenticated user's circle.
// GET /api/v1/users/me/circle/:id/check
func (h *UserHandler) IsInCircle(c echo.Context) error {
	did := c.Get("did").(string)
	memberID := c.Param("id")
	if memberID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "User ID is required"})
	}

	owner, err := h.userRepo.GetByDID(c.Request().Context(), did)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	// Verify the target user exists and normalize ID
	targetUser, err := h.userRepo.GetByID(c.Request().Context(), memberID)
	if err != nil {
		targetUser, err = h.userRepo.GetByDID(c.Request().Context(), memberID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Target user not found"})
		}
	}
	actualMemberID := targetUser.ID

	inCircle, err := h.circleRepo.IsInCircle(c.Request().Context(), owner.ID, actualMemberID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to check circle membership"})
	}

	return c.JSON(http.StatusOK, map[string]bool{"in_circle": inCircle})
}
