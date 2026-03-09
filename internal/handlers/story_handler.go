package handlers

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"

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

	if err := c.Bind(&req); err == nil && req.MediaURL != "" {
		mediaURL = req.MediaURL
	} else {

		file, err := c.FormFile("file")
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid request body",
			})
		}

		src, err := file.Open()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to open uploaded file"})
		}
		defer src.Close()

		if err := os.MkdirAll("./uploads", os.ModePerm); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create uploads directory"})
		}

		filename := uuid.New().String() + filepath.Ext(file.Filename)
		path := "./uploads/" + filename

		dst, err := os.Create(path)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save file"})
		}
		defer dst.Close()

		if _, err = io.Copy(dst, src); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to copy file contents"})
		}

		mediaURL = "/media/" + filename
	}

	if err := h.service.CreateStory(c.Request().Context(), userID, mediaURL); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create story"})
	}

	return c.JSON(http.StatusCreated, map[string]string{"status": "story created"})
}

func (h *StoryHandler) GetStories(c echo.Context) error {
	ctx := c.Request().Context()

	userIDStr, ok := c.Get("user_id").(string)
	if ok && userIDStr != "" {
		if viewerID, err := uuid.Parse(userIDStr); err == nil {
			ctx = context.WithValue(ctx, "viewer_id", viewerID)
		}
	}

	stories, err := h.service.GetStories(ctx)
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
	ctx := c.Request().Context()

	stories, err := h.service.GetStoryFeed(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"stories": stories,
	})
}

func (h *StoryHandler) GetStoryMedia(c echo.Context) error {
	id := c.Param("id")

	media, contentType, err := h.service.GetStoryMedia(c.Request().Context(), id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "story media not found",
		})
	}

	return c.Blob(http.StatusOK, contentType, media)
}
