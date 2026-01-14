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
	// Get user ID from JWT token
	userID := c.Get("user_id").(int)

	var req models.PostCreate
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	post, err := h.postRepo.Create(c.Request().Context(), userID, &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create post",
		})
	}

	return c.JSON(http.StatusCreated, post)
}

// GetPost retrieves a post by ID
func (h *PostHandler) GetPost(c echo.Context) error {
	idParam := c.Param("id")
	postID, err := strconv.Atoi(idParam)
	if err != nil {
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

// GetUserPosts retrieves all posts by a specific user
func (h *PostHandler) GetUserPosts(c echo.Context) error {
	idParam := c.Param("id")
	userID, err := strconv.Atoi(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid user ID",
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

	posts, err := h.postRepo.GetByUserID(c.Request().Context(), userID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve posts",
		})
	}

	return c.JSON(http.StatusOK, posts)
}

// GetFeed retrieves the authenticated user's feed
func (h *PostHandler) GetFeed(c echo.Context) error {
	// Get user ID from JWT token
	userID := c.Get("user_id").(int)

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

	posts, err := h.postRepo.GetFeed(c.Request().Context(), userID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve feed",
		})
	}

	return c.JSON(http.StatusOK, posts)
}

// UpdatePost updates a post
func (h *PostHandler) UpdatePost(c echo.Context) error {
	// Get user ID from JWT token
	userID := c.Get("user_id").(int)

	idParam := c.Param("id")
	postID, err := strconv.Atoi(idParam)
	if err != nil {
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

	post, err := h.postRepo.Update(c.Request().Context(), postID, userID, &req)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update post",
		})
	}

	return c.JSON(http.StatusOK, post)
}

// DeletePost deletes a post
func (h *PostHandler) DeletePost(c echo.Context) error {
	// Get user ID from JWT token
	userID := c.Get("user_id").(int)

	idParam := c.Param("id")
	postID, err := strconv.Atoi(idParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid post ID",
		})
	}

	if err := h.postRepo.Delete(c.Request().Context(), postID, userID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete post",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Post deleted successfully",
	})
}
