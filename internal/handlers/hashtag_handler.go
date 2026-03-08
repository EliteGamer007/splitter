package handlers

import (
	"net/http"
	"strconv"

	"splitter/internal/repository"

	"github.com/labstack/echo/v4"
)

// HashtagHandler handles hashtag-related requests
type HashtagHandler struct {
	postRepo *repository.PostRepository
}

// NewHashtagHandler creates a new HashtagHandler
func NewHashtagHandler(postRepo *repository.PostRepository) *HashtagHandler {
	return &HashtagHandler{postRepo: postRepo}
}

// GetTrendingHashtags returns the top trending hashtags from the last 24 hours
func (h *HashtagHandler) GetTrendingHashtags(c echo.Context) error {
	limitStr := c.QueryParam("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	trending, err := h.postRepo.GetTrendingHashtags(c.Request().Context(), limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get trending hashtags",
		})
	}

	return c.JSON(http.StatusOK, trending)
}

// GetPostsByHashtag returns all posts containing a specific hashtag
func (h *HashtagHandler) GetPostsByHashtag(c echo.Context) error {
	tag := c.Param("tag")
	if tag == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Hashtag is required",
		})
	}

	limitStr := c.QueryParam("limit")
	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	offsetStr := c.QueryParam("offset")
	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	// Get the current user DID if authenticated (for liked/reposted status)
	userDID, _ := c.Get("did").(string)

	posts, err := h.postRepo.GetPostsByHashtag(c.Request().Context(), tag, userDID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get posts by hashtag",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"hashtag": tag,
		"posts":   posts,
		"count":   len(posts),
	})
}

// SearchHashtags returns hashtags matching a query
func (h *HashtagHandler) SearchHashtags(c echo.Context) error {
	query := c.QueryParam("q")
	if len(query) < 1 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Search query too short (min 1 character)",
		})
	}

	limitStr := c.QueryParam("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	results, err := h.postRepo.SearchHashtags(c.Request().Context(), query, limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to search hashtags",
		})
	}

	return c.JSON(http.StatusOK, results)
}
