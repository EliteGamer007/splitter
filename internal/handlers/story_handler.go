package handlers

import (
	"context"
	"errors"
	"net/http"
	"splitter/internal/db"

	"splitter/internal/models"
	"splitter/internal/service"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type StoryHandler struct {
	service *service.StoryService
}

func NewStoryHandler(svc *service.StoryService) *StoryHandler {
	return &StoryHandler{service: svc}
}

func (h *StoryHandler) CreateStory(c echo.Context) error {
	userIDStr, ok := c.Get("user_id").(string)
	if !ok || userIDStr == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid user ID format"})
	}

	var req struct {
		MediaURL string `json:"media_url"`
	}

	var mediaURL string
	var mediaData []byte
	var mediaType string

	if err := c.Bind(&req); err == nil && req.MediaURL != "" {
		mediaURL = req.MediaURL
	} else {

		file, err := c.FormFile("file")
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid request body",
			})
		}

		mediaData, mediaType, err = service.ReadAndValidateImage(file, service.MaxImageInput)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
	}

	c.Logger().Infof(
		"Story upload request: user=%s mediaType=%s size=%d bytes",
		userID.String(),
		mediaType,
		len(mediaData),
	)

	if err := h.service.CreateStoryWithMedia(c.Request().Context(), userID, mediaURL, mediaType, mediaData); err != nil {
		c.Logger().Errorf("CreateStory failed: %v", err)
		if errors.Is(err, db.ErrStorageQuota) {
			return c.JSON(http.StatusInsufficientStorage, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Unable to create story",
		})
	}

	return c.JSON(http.StatusCreated, map[string]string{"status": "story created"})
}

func (h *StoryHandler) GetStories(c echo.Context) error {
	ctx := storyContext(c)

	userIDStr, ok := c.Get("user_id").(string)
	if ok && userIDStr != "" {
		if viewerID, err := uuid.Parse(userIDStr); err == nil {
			ctx = context.WithValue(ctx, "viewer_id", viewerID)
		}
	}

	limit, offset := requestPage(c, 20)
	stories, err := h.service.GetStoriesPage(ctx, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch stories"})
	}

	if stories == nil {
		stories = make([]models.Story, 0)
	}

	return c.JSON(http.StatusOK, stories)
}

func (h *StoryHandler) DeleteStory(c echo.Context) error {
	userIDStr, ok := c.Get("user_id").(string)
	if !ok || userIDStr == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid user id",
		})
	}

	storyIDStr := c.Param("id")

	storyID, err := uuid.Parse(storyIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid story id",
		})
	}

	err = h.service.DeleteStory(c.Request().Context(), storyID, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to delete story",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status": "story deleted",
	})
}

func (h *StoryHandler) ViewStory(c echo.Context) error {
	userIDStr, ok := c.Get("user_id").(string)
	if !ok || userIDStr == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
	}

	viewerID, err := uuid.Parse(userIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid user id",
		})
	}

	storyIDStr := c.Param("id")

	storyID, err := uuid.Parse(storyIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid story id",
		})
	}

	err = h.service.RecordStoryView(c.Request().Context(), storyID, viewerID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to record story view",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status": "view recorded",
	})
}

func (h *StoryHandler) GetStoryFeed(c echo.Context) error {
	ctx := storyContext(c)

	limit, offset := requestPage(c, 20)
	stories, err := h.service.GetStoryFeedPage(ctx, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Unable to load story feed",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"stories": stories,
	})
}

func (h *StoryHandler) GetStoryMedia(c echo.Context) error {
	c.Response().Header().Set("Cache-Control", "private, no-store")
	id := c.Param("id")

	media, contentType, err := h.service.GetStoryMedia(storyContext(c), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "story media not found",
		})
	}

	return c.Blob(http.StatusOK, contentType, media)
}

func storyContext(c echo.Context) context.Context {
	ctx := c.Request().Context()
	if raw, ok := c.Get("user_id").(string); ok {
		if id, err := uuid.Parse(raw); err == nil {
			return context.WithValue(ctx, "viewer_id", id)
		}
	}
	return ctx
}
func (h *StoryHandler) GetLegacyStoryMedia(c echo.Context) error {
	c.Response().Header().Set("Cache-Control", "private, no-store")
	data, kind, err := h.service.GetLegacyMedia(storyContext(c), c.Request().URL.Path)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Story media not found"})
	}
	return c.Blob(http.StatusOK, kind, data)
}
