package handlers

import (
	"net/http"
	"strconv"

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

// GetProfile retrieves a user's profile by ID
func (h *UserHandler) GetProfile(c echo.Context) error {
	idParam := c.Param("id")
	userID, err := strconv.Atoi(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID",
		})
	}

	user, err := h.userRepo.GetByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	return c.JSON(http.StatusOK, user)
}

// GetCurrentUser retrieves the authenticated user's profile
func (h *UserHandler) GetCurrentUser(c echo.Context) error {
	// Get user ID from JWT token (set by auth middleware)
	userID := c.Get("user_id").(int)

	user, err := h.userRepo.GetByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	return c.JSON(http.StatusOK, user)
}

// UpdateProfile updates the authenticated user's profile
func (h *UserHandler) UpdateProfile(c echo.Context) error {
	// Get user ID from JWT token
	userID := c.Get("user_id").(int)

	var req models.UserUpdate
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	user, err := h.userRepo.Update(c.Request().Context(), userID, &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update profile",
		})
	}

	return c.JSON(http.StatusOK, user)
}

// DeleteAccount deletes the authenticated user's account
func (h *UserHandler) DeleteAccount(c echo.Context) error {
	// Get user ID from JWT token
	userID := c.Get("user_id").(int)

	if err := h.userRepo.Delete(c.Request().Context(), userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete account",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Account deleted successfully",
	})
}
