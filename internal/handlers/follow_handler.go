package handlers

import (
	"net/http"
	"strconv"

	"splitter/internal/repository"

	"github.com/labstack/echo/v4"
)

// FollowHandler handles follow-related requests
type FollowHandler struct {
	followRepo *repository.FollowRepository
	userRepo   *repository.UserRepository
}

// NewFollowHandler creates a new FollowHandler
func NewFollowHandler(followRepo *repository.FollowRepository, userRepo *repository.UserRepository) *FollowHandler {
	return &FollowHandler{
		followRepo: followRepo,
		userRepo:   userRepo,
	}
}

// FollowUser follows a user
func (h *FollowHandler) FollowUser(c echo.Context) error {
	// Get authenticated user's DID from JWT
	did := c.Get("did").(string)

	// Get current user
	currentUser, err := h.userRepo.GetByDID(c.Request().Context(), did)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Current user not found",
		})
	}

	// Get target user ID from path
	targetIDStr := c.Param("id")

	// Try to parse as UUID first (user ID), or lookup by DID
	targetUser, err := h.userRepo.GetByID(c.Request().Context(), targetIDStr)
	if err != nil {
		// Try as DID
		targetUser, err = h.userRepo.GetByDID(c.Request().Context(), targetIDStr)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Target user not found",
			})
		}
	}

	// Create follow relationship
	follow, err := h.followRepo.Create(c.Request().Context(), currentUser.DID, targetUser.DID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to follow user",
		})
	}

	return c.JSON(http.StatusCreated, follow)
}

// UnfollowUser unfollows a user
func (h *FollowHandler) UnfollowUser(c echo.Context) error {
	// Get authenticated user's DID from JWT
	did := c.Get("did").(string)

	// Get current user
	currentUser, err := h.userRepo.GetByDID(c.Request().Context(), did)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Current user not found",
		})
	}

	// Get target user ID from path
	targetIDStr := c.Param("id")

	// Try to parse as UUID first, or lookup by DID
	targetUser, err := h.userRepo.GetByID(c.Request().Context(), targetIDStr)
	if err != nil {
		targetUser, err = h.userRepo.GetByDID(c.Request().Context(), targetIDStr)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Target user not found",
			})
		}
	}

	// Delete follow relationship
	if err := h.followRepo.Delete(c.Request().Context(), currentUser.DID, targetUser.DID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to unfollow user",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Successfully unfollowed user",
	})
}

// GetFollowers retrieves a user's followers
func (h *FollowHandler) GetFollowers(c echo.Context) error {
	targetIDStr := c.Param("id")

	// Parse pagination parameters
	limit := 50
	offset := 0
	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Get target user
	targetUser, err := h.userRepo.GetByID(c.Request().Context(), targetIDStr)
	if err != nil {
		targetUser, err = h.userRepo.GetByDID(c.Request().Context(), targetIDStr)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "User not found",
			})
		}
	}

	// Get followers
	followers, err := h.followRepo.GetFollowers(c.Request().Context(), targetUser.DID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve followers",
		})
	}

	return c.JSON(http.StatusOK, followers)
}

// GetFollowing retrieves users followed by a user
func (h *FollowHandler) GetFollowing(c echo.Context) error {
	targetIDStr := c.Param("id")

	// Parse pagination parameters
	limit := 50
	offset := 0
	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Get target user
	targetUser, err := h.userRepo.GetByID(c.Request().Context(), targetIDStr)
	if err != nil {
		targetUser, err = h.userRepo.GetByDID(c.Request().Context(), targetIDStr)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "User not found",
			})
		}
	}

	// Get following
	following, err := h.followRepo.GetFollowing(c.Request().Context(), targetUser.DID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve following",
		})
	}

	return c.JSON(http.StatusOK, following)
}

// GetFollowStats retrieves follow statistics for a user
func (h *FollowHandler) GetFollowStats(c echo.Context) error {
	targetIDStr := c.Param("id")

	// Get target user
	targetUser, err := h.userRepo.GetByID(c.Request().Context(), targetIDStr)
	if err != nil {
		targetUser, err = h.userRepo.GetByDID(c.Request().Context(), targetIDStr)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "User not found",
			})
		}
	}

	// Get stats
	stats, err := h.followRepo.GetStats(c.Request().Context(), targetUser.DID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve stats",
		})
	}

	return c.JSON(http.StatusOK, stats)
}
