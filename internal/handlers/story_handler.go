package handlers

import (
	"fmt"
	"net/http"

	"splitter/internal/repository"
	"splitter/internal/service"

	"github.com/labstack/echo/v4"
)

// StoryHandler handles story-related requests
type StoryHandler struct {
	storyRepo *repository.StoryRepository
}

// NewStoryHandler creates a new StoryHandler
func NewStoryHandler(storyRepo *repository.StoryRepository) *StoryHandler {
	return &StoryHandler{storyRepo: storyRepo}
}

// CreateStory uploads a new story image
func (h *StoryHandler) CreateStory(c echo.Context) error {
	did, ok := c.Get("did").(string)
	if !ok || did == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Image file is required",
		})
	}

	// Validate and read image (5MB limit, same as posts)
	mediaData, mediaType, err := service.ReadAndValidateImage(file, 5*1024*1024)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid image: %v", err),
		})
	}

	story, err := h.storyRepo.Create(c.Request().Context(), did, mediaData, mediaType)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create story",
		})
	}

	return c.JSON(http.StatusCreated, story)
}

// GetStoryFeed returns stories from followed users + own stories, grouped by user
func (h *StoryHandler) GetStoryFeed(c echo.Context) error {
	did, ok := c.Get("did").(string)
	if !ok || did == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	storyUsers, err := h.storyRepo.GetFeedForUser(c.Request().Context(), did)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve stories",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"stories": storyUsers,
	})
}

// GetStoryMedia serves the binary image content for a story
func (h *StoryHandler) GetStoryMedia(c echo.Context) error {
	storyID := c.Param("id")
	if storyID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid story ID",
		})
	}

	mediaData, mediaType, err := h.storyRepo.GetMediaContent(c.Request().Context(), storyID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Story not found or expired",
		})
	}

	return c.Blob(http.StatusOK, mediaType, mediaData)
}

// DeleteStory removes a story (author only)
func (h *StoryHandler) DeleteStory(c echo.Context) error {
	did, ok := c.Get("did").(string)
	if !ok || did == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	storyID := c.Param("id")
	if storyID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid story ID",
		})
	}

	if err := h.storyRepo.Delete(c.Request().Context(), storyID, did); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete story",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Story deleted successfully",
	})
}
