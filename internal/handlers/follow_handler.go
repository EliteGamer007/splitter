package handlers

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"splitter/internal/config"
	"splitter/internal/federation"
	"strconv"
	"strings"
	"time"

	"splitter/internal/models"
	"splitter/internal/repository"

	"github.com/labstack/echo/v4"
)

// FollowHandler handles follow-related requests
type FollowHandler struct {
	followRepo      *repository.FollowRepository
	userRepo        *repository.UserRepository
	cfg             *config.Config
	deliverAccepted func(*federation.Activity, string) error
}

// NewFollowHandler creates a new FollowHandler
func NewFollowHandler(followRepo *repository.FollowRepository, userRepo *repository.UserRepository, configs ...*config.Config) *FollowHandler {
	var cfg *config.Config
	if len(configs) > 0 {
		cfg = configs[0]
	}
	return &FollowHandler{
		followRepo:      followRepo,
		cfg:             cfg,
		deliverAccepted: federation.DeliverToActor,
		userRepo:        userRepo,
	}
}

// FollowUser follows a user
func (h *FollowHandler) FollowUser(c echo.Context) error {
	// Get authenticated user's DID from JWT
	did := c.Get("did").(string)

	// Get current user
	currentUser, err := h.userRepo.GetByDID(c.Request().Context(), did)
	if err != nil {
		c.Logger().Errorf("Failed to get current user by DID %s: %v", did, err)
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Current user not found",
		})
	}

	// Get target user ID from path
	targetIDStr := c.Param("id")
	c.Logger().Infof("Follow request: current user DID=%s, target ID=%s", did, targetIDStr)

	// Try to parse as UUID first (user ID), or lookup by DID
	targetUser, err := h.userRepo.GetByID(c.Request().Context(), targetIDStr)
	if err != nil {
		c.Logger().Infof("Target not found by ID, trying DID: %v", err)
		// Try as DID
		targetUser, err = h.userRepo.GetByDID(c.Request().Context(), targetIDStr)
		if err != nil {
			c.Logger().Errorf("Target user not found by ID or DID: %v", err)
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Target user not found",
			})
		}
	}

	c.Logger().Infof("Following: %s -> %s", currentUser.DID, targetUser.DID)

	// Create follow relationship
	follow, err := h.followRepo.Create(c.Request().Context(), currentUser.DID, targetUser.DID)
	if err != nil {
		c.Logger().Errorf("Failed to create follow: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to follow user",
		})
	}

	return c.JSON(http.StatusCreated, follow)
}

// UnfollowUser unfollows a user
func (h *FollowHandler) UnfollowUser(c echo.Context) error {
	// Get authenticated user's DID from JWT
	did := c.Get("did").(string)

	// Get current user
	currentUser, err := h.userRepo.GetByDID(c.Request().Context(), did)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Current user not found",
		})
	}

	// Get target user ID from path
	targetIDStr := c.Param("id")

	// Try to parse as UUID first, or lookup by DID
	targetUser, err := h.userRepo.GetByID(c.Request().Context(), targetIDStr)
	if err != nil {
		targetUser, err = h.userRepo.GetByDID(c.Request().Context(), targetIDStr)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Target user not found",
			})
		}
	}

	// Delete follow relationship
	if err := h.followRepo.Delete(c.Request().Context(), currentUser.DID, targetUser.DID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to unfollow user",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Successfully unfollowed user",
	})
}

// GetFollowers retrieves a user's followers
func (h *FollowHandler) GetFollowers(c echo.Context) error {
	targetIDStr := c.Param("id")

	// Parse pagination parameters
	limit := 20
	offset := 0
	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			if parsed > 50 {
				parsed = 50
			}
			limit = parsed
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Get target user
	targetUser, err := h.userRepo.GetByID(c.Request().Context(), targetIDStr)
	if err != nil {
		targetUser, err = h.userRepo.GetByDID(c.Request().Context(), targetIDStr)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "User not found",
			})
		}
	}

	// Get followers
	followers, err := h.followRepo.GetFollowers(c.Request().Context(), targetUser.DID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve followers",
		})
	}

	return c.JSON(http.StatusOK, models.ToPublicUsers(followers))
}

// GetFollowing retrieves users followed by a user
func (h *FollowHandler) GetFollowing(c echo.Context) error {
	targetIDStr := c.Param("id")

	// Parse pagination parameters
	limit := 20
	offset := 0
	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			if parsed > 50 {
				parsed = 50
			}
			limit = parsed
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Get target user
	targetUser, err := h.userRepo.GetByID(c.Request().Context(), targetIDStr)
	if err != nil {
		targetUser, err = h.userRepo.GetByDID(c.Request().Context(), targetIDStr)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "User not found",
			})
		}
	}

	// Get following
	following, err := h.followRepo.GetFollowing(c.Request().Context(), targetUser.DID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve following",
		})
	}

	return c.JSON(http.StatusOK, models.ToPublicUsers(following))
}

// GetFollowStats retrieves follow statistics for a user
func (h *FollowHandler) GetFollowStats(c echo.Context) error {
	targetIDStr := c.Param("id")

	// Get target user
	targetUser, err := h.userRepo.GetByID(c.Request().Context(), targetIDStr)
	if err != nil {
		targetUser, err = h.userRepo.GetByDID(c.Request().Context(), targetIDStr)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "User not found",
			})
		}
	}

	// Get stats
	stats, err := h.followRepo.GetStats(c.Request().Context(), targetUser.DID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to retrieve stats",
		})
	}

	return c.JSON(http.StatusOK, stats)
}

// GetFollowRequests lists pending requests addressed to the signed-in account.
func (h *FollowHandler) GetFollowRequests(c echo.Context) error {
	did, _ := c.Get("did").(string)
	if did == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}
	limit, offset := requestPage(c, 20)
	requests, err := h.followRepo.GetRequests(c.Request().Context(), did, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load follow requests"})
	}
	return c.JSON(http.StatusOK, requests)
}
func (h *FollowHandler) AcceptFollowRequest(c echo.Context) error {
	return h.reviewFollowRequest(c, true)
}
func (h *FollowHandler) RejectFollowRequest(c echo.Context) error {
	return h.reviewFollowRequest(c, false)
}
func (h *FollowHandler) reviewFollowRequest(c echo.Context, accept bool) error {
	did, _ := c.Get("did").(string)
	if did == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}
	follower, err := h.userRepo.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		follower, err = h.userRepo.GetByDID(c.Request().Context(), c.Param("id"))
	}
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Request not found"})
	}
	if accept {
		err = h.followRepo.AcceptRequest(c.Request().Context(), did, follower.DID)
	} else {
		err = h.followRepo.RejectRequest(c.Request().Context(), did, follower.DID)
	}
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Pending request not found"})
	}
	if accept && h.cfg != nil && h.cfg.Federation.Enabled {
		if owner, err := h.userRepo.GetByDID(c.Request().Context(), did); err == nil {
			h.notifyFollowAccepted(owner, follower)
		}
	}
	return c.JSON(http.StatusOK, map[string]bool{"accepted": accept})
}
func (h *FollowHandler) GetFollowStatus(c echo.Context) error {
	did, _ := c.Get("did").(string)
	if did == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
	}
	target, err := h.userRepo.GetByID(c.Request().Context(), c.Param("id"))
	if err != nil {
		target, err = h.userRepo.GetByDID(c.Request().Context(), c.Param("id"))
	}
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}
	status, err := h.followRepo.GetStatus(c.Request().Context(), did, target.DID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to load follow status"})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": status})
}

func (h *FollowHandler) notifyFollowAccepted(owner, follower *models.User) {
	if h.cfg == nil || !h.cfg.Federation.Enabled || owner == nil || follower == nil {
		return
	}
	remote, err := url.Parse(follower.DID)
	if err != nil || (remote.Scheme != "https" && remote.Scheme != "http") || remote.Host == "" {
		return
	}
	actor := strings.TrimRight(h.cfg.Federation.URL, "/") + "/ap/users/" + owner.Username
	activity := &federation.Activity{Context: "https://www.w3.org/ns/activitystreams", ID: fmt.Sprintf("%s/activities/accept-%d", strings.TrimRight(h.cfg.Federation.URL, "/"), time.Now().UnixNano()), Type: "Accept", Actor: actor, To: []string{follower.DID}, Object: map[string]interface{}{"type": "Follow", "actor": follower.DID, "object": actor}}
	deliver := h.deliverAccepted
	if deliver == nil {
		deliver = federation.DeliverToActor
	}
	// Outbound delivery owns bounded HTTP concurrency, deadlines and persisted retries.
	go func() {
		if err := deliver(activity, follower.DID); err != nil {
			log.Printf("[Federation] follow acceptance delivery pending")
		}
	}()
}
