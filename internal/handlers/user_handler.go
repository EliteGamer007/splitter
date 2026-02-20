package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"splitter/internal/config"
	"splitter/internal/db"
	"splitter/internal/federation"
	"splitter/internal/models"
	"splitter/internal/repository"
	"splitter/internal/service"

	"github.com/labstack/echo/v4"
)

// UserHandler handles user-related requests
type UserHandler struct {
	userRepo *repository.UserRepository
	cfg      *config.Config
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userRepo *repository.UserRepository, cfg *config.Config) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
		cfg:      cfg,
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

	return c.JSON(http.StatusOK, user)
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

	return c.JSON(http.StatusOK, user)
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

	var req models.UserUpdate
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	updatedUser, err := h.userRepo.Update(c.Request().Context(), user.ID, &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update profile",
		})
	}

	if h.cfg != nil && h.cfg.Federation.Enabled {
		actorURI := fmt.Sprintf("%s/ap/users/%s", h.cfg.Federation.URL, user.Username)
		activity := federation.BuildUpdateActorActivity(
			actorURI,
			user.Username,
			updatedUser.DisplayName,
			updatedUser.Bio,
			updatedUser.PublicKey,
			updatedUser.EncryptionPublicKey,
		)
		go federation.DeliverToFollowers(activity, user.DID)
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

	avatarBytes, mediaType, err := service.ReadAndValidateImage(file, 5*1024*1024)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid avatar image: %v", err),
		})
	}

	updatedUser, err := h.userRepo.UpdateAvatar(c.Request().Context(), user.ID, avatarBytes, mediaType)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to upload avatar",
		})
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
		legacyPath := filepath.Join(".", filepath.FromSlash(strings.TrimPrefix(avatarURL, "/")))
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
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update encryption key: " + err.Error(),
		})
	}

	if h.cfg != nil && h.cfg.Federation.Enabled {
		if user, fetchErr := h.userRepo.GetByID(c.Request().Context(), userID); fetchErr == nil && user != nil {
			actorURI := fmt.Sprintf("%s/ap/users/%s", h.cfg.Federation.URL, user.Username)
			activity := federation.BuildUpdateActorActivity(
				actorURI,
				user.Username,
				user.DisplayName,
				user.Bio,
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
