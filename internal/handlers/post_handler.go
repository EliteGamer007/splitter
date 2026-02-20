package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"splitter/internal/config"
	"splitter/internal/federation"
	"splitter/internal/models"
	"splitter/internal/repository"
	"splitter/internal/service"

	"github.com/labstack/echo/v4"
)

// PostHandler handles post-related requests
type PostHandler struct {
	postRepo *repository.PostRepository
	userRepo *repository.UserRepository
	cfg      *config.Config
}

// NewPostHandler creates a new PostHandler
func NewPostHandler(postRepo *repository.PostRepository, userRepo *repository.UserRepository, cfg *config.Config) *PostHandler {
	return &PostHandler{
		postRepo: postRepo,
		userRepo: userRepo,
		cfg:      cfg,
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

	// Parse multipart form
	// Limit handled by middleware, but good to have fallback checks
	content := c.FormValue("content")
	visibility := c.FormValue("visibility")

	// Handle file upload check first to validate
	file, fileErr := c.FormFile("file")

	// Validate content or media presence
	if len(content) > 500 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Content too long (max 500 characters)",
		})
	}
	// Check if both content is empty and no file is provided
	// fileErr will be nil if file exists
	if content == "" && fileErr != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Either content or media is required",
		})
	}

	// Handle file upload
	var mediaData []byte
	var mediaType string
	if fileErr == nil {
		// File present, validate and read
		bytes, mType, err := service.ReadAndValidateImage(file, 5*1024*1024)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": fmt.Sprintf("Failed to upload file: %v", err),
			})
		}
		mediaData = bytes
		mediaType = mType
	} else if fileErr != http.ErrMissingFile {
		// Real error occurred
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Failed to process file: %v", fileErr),
		})
	}

	req := models.PostCreate{
		Content:    content,
		Visibility: visibility,
	}

	post, err := h.postRepo.Create(c.Request().Context(), did, &req, mediaData, mediaType)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to create post",
		})
	}

	// Federation Hook: Deliver to remote followers
	if h.cfg.Federation.Enabled {
		log.Printf("[Federation] Post created by %s, triggering delivery...", did)
		go func() {
			// Fetch user to get username
			user, err := h.userRepo.GetByDID(context.Background(), did)
			if err != nil {
				log.Printf("[Federation] Failed to fetch user %s for delivery: %v", did, err)
				return
			}

			// Construct Actor URI: base_url/ap/users/username
			actorURI := fmt.Sprintf("%s/ap/users/%s", h.cfg.Federation.URL, user.Username)
			log.Printf("[Federation] Building Create activity for %s (post %s)", actorURI, post.ID)

			// Build Create activity
			activity := federation.BuildCreateNoteActivity(actorURI, post.ID, post.Content, post.CreatedAt)

			// Deliver to followers
			federation.DeliverToFollowers(activity, did)
		}()
	} else {
		log.Printf("[Federation] Federation disabled, skipping delivery for post %s", post.ID)
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

// GetPublicFeed retrieves public posts for unauthenticated users or authenticated users viewing public timeline
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

	// Check if user is authenticated to include liked status
	userDID, _ := c.Get("did").(string)

	localOnly := false
	if localParam := c.QueryParam("local_only"); localParam == "true" || localParam == "1" {
		localOnly = true
	}

	posts, err := h.postRepo.GetPublicFeedWithUser(c.Request().Context(), userDID, limit, offset, localOnly)
	if err != nil {
		c.Logger().Errorf("Failed to retrieve public feed: %v", err)
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

	// Check if user is admin or moderator - they can delete any post
	role, _ := c.Get("role").(string)
	isAdmin := role == "admin" || role == "moderator"

	meta, _ := h.postRepo.GetFederationMeta(c.Request().Context(), postID)

	if err := h.postRepo.Delete(c.Request().Context(), postID, did, isAdmin); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete post",
		})
	}

	if h.cfg.Federation.Enabled && !isAdmin {
		if user, err := h.userRepo.GetByDID(c.Request().Context(), did); err == nil && user != nil {
			actorURI := fmt.Sprintf("%s/ap/users/%s", h.cfg.Federation.URL, user.Username)
			objectURI := fmt.Sprintf("%s/posts/%s", h.cfg.Federation.URL, postID)
			if meta != nil && meta.OriginalPostURI != "" {
				objectURI = meta.OriginalPostURI
			}

			deleteActivity := federation.BuildDeleteActivity(actorURI, objectURI)
			go federation.DeliverToFollowers(deleteActivity, did)
		}
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Post deleted successfully",
	})
}
