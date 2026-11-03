package handlers

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"splitter/internal/db"
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
	c.Response().Header().Set("Cache-Control", "private, no-store")
	c.Response().Header().Set("Vary", "Authorization, Cookie")
	mediaID := c.Param("id")
	if mediaID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid media ID"})
	}

	mediaData, mediaType, mediaURL, err := h.postRepo.GetMediaContentForViewer(c.Request().Context(), mediaID, viewerDID(c))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Media not found"})
	}

	return servePostMedia(c, mediaData, mediaType, mediaURL)
}

// GetLegacyPostMedia accepts only exact database-owned URLs and clean paths.
func (h *MediaHandler) GetLegacyPostMedia(c echo.Context) error {
	c.Response().Header().Set("Cache-Control", "private, no-store")
	c.Response().Header().Set("Vary", "Authorization, Cookie")
	if db.GetDB() == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Media temporarily unavailable"})
	}
	legacyURL := c.Request().URL.Path
	if !strings.HasPrefix(legacyURL, "/uploads/posts/") && !strings.HasPrefix(legacyURL, "/media/") {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Media not found"})
	}
	if path.Clean(legacyURL) != legacyURL || strings.Contains(legacyURL, "\\") {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Media not found"})
	}
	data, kind, savedURL, err := h.postRepo.GetLegacyMediaURL(c.Request().Context(), legacyURL, viewerDID(c))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Media not found"})
	}
	return servePostMedia(c, data, kind, savedURL)
}
func servePostMedia(c echo.Context, mediaData []byte, mediaType, mediaURL string) error {
	if len(mediaData) > 0 {
		if mediaType == "" {
			mediaType = "application/octet-stream"
		}
		return c.Blob(http.StatusOK, mediaType, mediaData)
	}

	if strings.HasPrefix(mediaURL, "/uploads/") {
		legacyPath, valid := safeLegacyUploadPath(mediaURL)
		if !valid {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Media not found"})
		}
		legacyData, readErr := os.ReadFile(legacyPath)
		if readErr == nil && len(legacyData) > 0 {
			detectedType := http.DetectContentType(legacyData)
			return c.Blob(http.StatusOK, detectedType, legacyData)
		}
	}

	return c.JSON(http.StatusNotFound, map[string]string{"error": "Media content not found"})
}

func safeLegacyUploadPath(mediaURL string) (string, bool) {
	if !strings.HasPrefix(mediaURL, "/uploads/") || path.Clean(mediaURL) != mediaURL || strings.Contains(mediaURL, "\\") {
		return "", false
	}
	root, err := filepath.Abs("uploads")
	if err != nil {
		return "", false
	}
	resolved, err := filepath.Abs(filepath.FromSlash(strings.TrimPrefix(mediaURL, "/")))
	if err != nil {
		return "", false
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return "", false
	}
	return resolved, true
}
