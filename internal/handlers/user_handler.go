package handlers

import (
	"net/http"

	"splitter/internal/models"
	"splitter/internal/repository"

	"github.com/labstack/echo/v4"
)

// UserHandler handles user-related requests
type UserHandler struct {
	userRepo *repository.UserRepository
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userRepo *repository.UserRepository) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
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

	return c.JSON(http.StatusOK, updatedUser)
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

	if err := h.userRepo.Delete(c.Request().Context(), user.ID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete account",
		})
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

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Encryption key updated successfully",
	})
}
