package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"splitter/internal/config"
	"splitter/internal/federation"
	"splitter/internal/models"
	"splitter/internal/repository"

	"github.com/labstack/echo/v4"
)

// InteractionHandler handles post interactions (likes, reposts, bookmarks)
type InteractionHandler struct {
	interactionRepo *repository.InteractionRepository
	userRepo        *repository.UserRepository
	postRepo        *repository.PostRepository
	cfg             *config.Config
}

// NewInteractionHandler creates a new InteractionHandler
func NewInteractionHandler(interactionRepo *repository.InteractionRepository, userRepo *repository.UserRepository, postRepo *repository.PostRepository, cfg *config.Config) *InteractionHandler {
	return &InteractionHandler{
		interactionRepo: interactionRepo,
		userRepo:        userRepo,
		postRepo:        postRepo,
		cfg:             cfg,
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

	h.dispatchFederatedInteraction(c, currentUser, postID, "Like")

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

	h.dispatchFederatedInteraction(c, currentUser, postID, "Announce")

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

func (h *InteractionHandler) dispatchFederatedInteraction(c echo.Context, currentUser *models.User, postID, activityType string) {
	if h.cfg == nil || !h.cfg.Federation.Enabled {
		return
	}

	meta, err := h.postRepo.GetFederationMeta(c.Request().Context(), postID)
	if err != nil || meta == nil {
		return
	}

	if !meta.IsRemote || !strings.HasPrefix(meta.AuthorDID, "http") {
		return
	}

	objectURI := meta.OriginalPostURI
	if strings.TrimSpace(objectURI) == "" {
		objectURI = fmt.Sprintf("%s/posts/%s", h.cfg.Federation.URL, postID)
	}

	actorURI := fmt.Sprintf("%s/ap/users/%s", h.cfg.Federation.URL, currentUser.Username)

	go func(targetActorURI string) {
		var activity *federation.Activity
		switch activityType {
		case "Like":
			activity = federation.BuildLikeActivity(actorURI, objectURI)
		case "Announce":
			activity = federation.BuildAnnounceActivity(actorURI, objectURI)
		default:
			return
		}

		if err := federation.DeliverToActor(activity, targetActorURI); err != nil {
			log.Printf("[Federation] Failed to federate %s for post %s to %s: %v", activityType, postID, targetActorURI, err)
		}
	}(meta.AuthorDID)
}
