package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"splitter/internal/repository"

	"github.com/labstack/echo/v4"
)

// MediaHandler handles media retrieval requests.
type MediaHandler struct {
	postRepo *repository.PostRepository
}

// NewMediaHandler creates a new MediaHandler.
func NewMediaHandler(postRepo *repository.PostRepository) *MediaHandler {
	return &MediaHandler{postRepo: postRepo}
}

// GetMediaContent serves post media from DB, with fallback for legacy disk URLs.
func (h *MediaHandler) GetMediaContent(c echo.Context) error {
	mediaID := c.Param("id")
	if mediaID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid media ID"})
	}

	mediaData, mediaType, mediaURL, err := h.postRepo.GetMediaContent(c.Request().Context(), mediaID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Media not found"})
	}

	if len(mediaData) > 0 {
		if mediaType == "" {
			mediaType = "application/octet-stream"
		}
		return c.Blob(http.StatusOK, mediaType, mediaData)
	}

	if strings.HasPrefix(mediaURL, "/uploads/") {
		legacyPath := filepath.Join(".", filepath.FromSlash(strings.TrimPrefix(mediaURL, "/")))
		legacyData, readErr := os.ReadFile(legacyPath)
		if readErr == nil && len(legacyData) > 0 {
			detectedType := http.DetectContentType(legacyData)
			return c.Blob(http.StatusOK, detectedType, legacyData)
		}
	}

	return c.JSON(http.StatusNotFound, map[string]string{"error": "Media content not found"})
}
