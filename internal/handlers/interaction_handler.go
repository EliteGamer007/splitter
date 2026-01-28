package handlers

import (
	"net/http"

	"splitter/internal/repository"

	"github.com/labstack/echo/v4"
)

// InteractionHandler handles post interactions (likes, reposts, bookmarks)
type InteractionHandler struct {
	interactionRepo *repository.InteractionRepository
	userRepo        *repository.UserRepository
}

// NewInteractionHandler creates a new InteractionHandler
func NewInteractionHandler(interactionRepo *repository.InteractionRepository, userRepo *repository.UserRepository) *InteractionHandler {
	return &InteractionHandler{
		interactionRepo: interactionRepo,
		userRepo:        userRepo,
	}
}

// LikePost likes a post
func (h *InteractionHandler) LikePost(c echo.Context) error {
	// Get authenticated user's DID from JWT
	did := c.Get("did").(string)

	// Get current user
	currentUser, err := h.userRepo.GetByDID(c.Request().Context(), did)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Current user not found",
		})
	}

	// Get post ID from path
	postID := c.Param("id")

	// Create like
	if err := h.interactionRepo.CreateLike(c.Request().Context(), postID, currentUser.DID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to like post",
		})
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"message": "Post liked successfully",
	})
}

// UnlikePost removes a like from a post
func (h *InteractionHandler) UnlikePost(c echo.Context) error {
	// Get authenticated user's DID from JWT
	did := c.Get("did").(string)

	// Get current user
	currentUser, err := h.userRepo.GetByDID(c.Request().Context(), did)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Current user not found",
		})
	}

	// Get post ID from path
	postID := c.Param("id")

	// Delete like
	if err := h.interactionRepo.DeleteLike(c.Request().Context(), postID, currentUser.DID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to unlike post",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Post unliked successfully",
	})
}

// RepostPost reposts/boosts a post
func (h *InteractionHandler) RepostPost(c echo.Context) error {
	// Get authenticated user's DID from JWT
	did := c.Get("did").(string)

	// Get current user
	currentUser, err := h.userRepo.GetByDID(c.Request().Context(), did)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Current user not found",
		})
	}

	// Get post ID from path
	postID := c.Param("id")

	// Create repost
	if err := h.interactionRepo.CreateRepost(c.Request().Context(), postID, currentUser.DID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to repost",
		})
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"message": "Post reposted successfully",
	})
}

// UnrepostPost removes a repost
func (h *InteractionHandler) UnrepostPost(c echo.Context) error {
	// Get authenticated user's DID from JWT
	did := c.Get("did").(string)

	// Get current user
	currentUser, err := h.userRepo.GetByDID(c.Request().Context(), did)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Current user not found",
		})
	}

	// Get post ID from path
	postID := c.Param("id")

	// Delete repost
	if err := h.interactionRepo.DeleteRepost(c.Request().Context(), postID, currentUser.DID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to remove repost",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Repost removed successfully",
	})
}

// BookmarkPost bookmarks a post (private)
func (h *InteractionHandler) BookmarkPost(c echo.Context) error {
	// Get authenticated user's DID from JWT
	did := c.Get("did").(string)

	// Get current user
	currentUser, err := h.userRepo.GetByDID(c.Request().Context(), did)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Current user not found",
		})
	}

	// Get post ID from path
	postID := c.Param("id")

	// Create bookmark
	if err := h.interactionRepo.CreateBookmark(c.Request().Context(), currentUser.ID, postID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to bookmark post",
		})
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"message": "Post bookmarked successfully",
	})
}

// UnbookmarkPost removes a bookmark
func (h *InteractionHandler) UnbookmarkPost(c echo.Context) error {
	// Get authenticated user's DID from JWT
	did := c.Get("did").(string)

	// Get current user
	currentUser, err := h.userRepo.GetByDID(c.Request().Context(), did)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Current user not found",
		})
	}

	// Get post ID from path
	postID := c.Param("id")

	// Delete bookmark
	if err := h.interactionRepo.DeleteBookmark(c.Request().Context(), currentUser.ID, postID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to remove bookmark",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Bookmark removed successfully",
	})
}

// GetBookmarks retrieves user's bookmarked posts
func (h *InteractionHandler) GetBookmarks(c echo.Context) error {
	// Get authenticated user's DID from JWT
	did := c.Get("did").(string)

	// Get current user
	currentUser, err := h.userRepo.GetByDID(c.Request().Context(), did)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Current user not found",
		})
	}

	// Get bookmarked posts
	posts, err := h.interactionRepo.GetBookmarks(c.Request().Context(), currentUser.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve bookmarks",
		})
	}

	return c.JSON(http.StatusOK, posts)
}
