package handlers

import (
	"net/http"
	"strconv"

	"splitter/internal/models"
	"splitter/internal/repository"

	"github.com/labstack/echo/v4"
)

// PostHandler handles post-related requests
type PostHandler struct {
	postRepo *repository.PostRepository
}

// NewPostHandler creates a new PostHandler
func NewPostHandler(postRepo *repository.PostRepository) *PostHandler {
	return &PostHandler{
		postRepo: postRepo,
	}
}

// CreatePost creates a new post
func (h *PostHandler) CreatePost(c echo.Context) error {
	// Get DID from JWT token (set by auth middleware)
	did, ok := c.Get("did").(string)
	if !ok || did == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	var req models.PostCreate
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	post, err := h.postRepo.Create(c.Request().Context(), did, &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create post",
		})
	}

	return c.JSON(http.StatusCreated, post)
}

// GetPost retrieves a post by ID
func (h *PostHandler) GetPost(c echo.Context) error {
	postID := c.Param("id")
	if postID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid post ID",
		})
	}

	post, err := h.postRepo.GetByID(c.Request().Context(), postID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Post not found",
		})
	}

	return c.JSON(http.StatusOK, post)
}

// GetUserPosts retrieves all posts by a specific user (by DID)
func (h *PostHandler) GetUserPosts(c echo.Context) error {
	authorDID := c.Param("did")
	if authorDID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user DID",
		})
	}

	// Parse pagination parameters
	limit := 20
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

	posts, err := h.postRepo.GetByAuthorDID(c.Request().Context(), authorDID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve posts",
		})
	}

	return c.JSON(http.StatusOK, posts)
}

// GetFeed retrieves the authenticated user's feed
func (h *PostHandler) GetFeed(c echo.Context) error {
	// Get DID from JWT token
	did, ok := c.Get("did").(string)
	if !ok || did == "" {
		// Return public feed for unauthenticated users
		return h.GetPublicFeed(c)
	}

	// Parse pagination parameters
	limit := 20
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

	posts, err := h.postRepo.GetFeed(c.Request().Context(), did, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve feed",
		})
	}

	return c.JSON(http.StatusOK, posts)
}

// GetPublicFeed retrieves public posts for unauthenticated users
func (h *PostHandler) GetPublicFeed(c echo.Context) error {
	// Parse pagination parameters
	limit := 20
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

	posts, err := h.postRepo.GetPublicFeed(c.Request().Context(), limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve public feed",
		})
	}

	return c.JSON(http.StatusOK, posts)
}

// UpdatePost updates a post
func (h *PostHandler) UpdatePost(c echo.Context) error {
	// Get DID from JWT token
	did, ok := c.Get("did").(string)
	if !ok || did == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	postID := c.Param("id")
	if postID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid post ID",
		})
	}

	var req models.PostUpdate
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	post, err := h.postRepo.Update(c.Request().Context(), postID, did, &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update post",
		})
	}

	return c.JSON(http.StatusOK, post)
}

// DeletePost deletes a post
func (h *PostHandler) DeletePost(c echo.Context) error {
	// Get DID from JWT token
	did, ok := c.Get("did").(string)
	if !ok || did == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	postID := c.Param("id")
	if postID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid post ID",
		})
	}

	if err := h.postRepo.Delete(c.Request().Context(), postID, did); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete post",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Post deleted successfully",
	})
}
