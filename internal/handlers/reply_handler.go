package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"splitter/internal/config"
	"splitter/internal/federation"
	"splitter/internal/models"
	"splitter/internal/repository"
	"splitter/internal/service"

	"github.com/labstack/echo/v4"
)

type ReplyHandler struct {
	Repo     *repository.ReplyRepository
	PostRepo *repository.PostRepository
	userRepo *repository.UserRepository
	cfg      *config.Config
}

func NewReplyHandler(cfg *config.Config, userRepo *repository.UserRepository) *ReplyHandler {
	return &ReplyHandler{
		Repo:     repository.NewReplyRepository(),
		PostRepo: repository.NewPostRepository(),
		userRepo: userRepo,
		cfg:      cfg,
	}
}

// CreateReply handles the creation of a new reply
func (h *ReplyHandler) CreateReply(c echo.Context) error {
	authorDID, ok := c.Get("did").(string)
	if !ok || authorDID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	var req models.ReplyCreate
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	// Get post ID from URL and override/set in request
	postID := c.Param("id")
	if postID != "" {
		req.PostID = postID
	}

	if err := req.Validate(); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	allowed, accessErr := h.PostRepo.CanView(c.Request().Context(), req.PostID, authorDID)
	if accessErr != nil || !allowed {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Post not found"})
	}
	// Calculate and validate depth
	depth := 1 // Default depth (reply to post)
	if req.ParentID != nil {
		// Fetch parent reply to check its depth
		parent, err := h.Repo.GetByID(c.Request().Context(), *req.ParentID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Parent reply not found"})
		}

		// If req.ParentID is provided, it must be the immediate parent.
		// New depth = Parent.Depth + 1
		depth = parent.Depth + 1

		if depth > 3 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Maximum reply depth exceeded"})
		}

		// Integrity check: Ensure parent belongs to the same PostID
		if parent.PostID != req.PostID {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Parent reply does not belong to the specified post"})
		}
	} else {
		// Reply to root post. Verify post exists.
		_, err := h.PostRepo.GetByID(c.Request().Context(), req.PostID)
		if err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Post not found"})
		}
	}

	reply, err := h.Repo.Create(c.Request().Context(), authorDID, &req, depth)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create reply"})
	}

	// Federation Hook: Deliver reply to remote instances
	if h.cfg != nil && h.cfg.Federation.Enabled {
		go func() {
			user, err := h.userRepo.GetByDID(context.Background(), authorDID)
			if err != nil {
				log.Printf("[Federation] Failed to fetch user %s for reply delivery: %v", authorDID, err)
				return
			}
			actorURI := fmt.Sprintf("%s/ap/users/%s", h.cfg.Federation.URL, user.Username)

			// Determine the correct inReplyTo URI
			// For remote posts, use the original_post_uri; for local posts, build a local URL
			parentPost, pErr := h.PostRepo.GetByID(context.Background(), req.PostID)
			var inReplyTo string
			if pErr == nil && parentPost.IsRemote && parentPost.OriginalPostURI != "" {
				inReplyTo = parentPost.OriginalPostURI
			} else {
				inReplyTo = fmt.Sprintf("%s/posts/%s", strings.TrimRight(h.cfg.Federation.URL, "/"), req.PostID)
			}

			activity := federation.BuildCreateNoteActivity(actorURI, reply.ID, reply.Content, reply.CreatedAt, "", inReplyTo)
			if pErr != nil || parentPost == nil {
				return
			}
			audienceOwner := authorDID
			if parentPost.Visibility != "public" {
				audienceOwner = parentPost.AuthorDID
			}
			federation.DeliverPostToAudience(activity, audienceOwner, parentPost.Visibility)

			// If replying to a remote post, also deliver directly to the remote author
			if pErr == nil && parentPost.IsRemote && parentPost.AuthorDID != "" {
				if parentPost.Visibility != "public" {
					activity.To = []string{parentPost.AuthorDID}
					activity.CC = nil
					if note, ok := activity.Object.(federation.Note); ok {
						note.To = activity.To
						note.CC = nil
						activity.Object = note
					}
				}

				if dErr := federation.DeliverToActor(activity, parentPost.AuthorDID); dErr != nil {
					log.Printf("[Federation] Failed to deliver reply to remote author %s: %v", parentPost.AuthorDID, dErr)
				} else {
					log.Printf("[Federation] Reply %s delivered to remote author %s", reply.ID, parentPost.AuthorDID)
				}
			}

			log.Printf("[Federation] Reply %s to post %s delivered", reply.ID, req.PostID)
		}()
	}

	// The service checks public visibility, author status and persistent budgets
	// before sending an actual mention to the configured provider.
	service.CheckAndHandleSplitBot(reply.Content, req.PostID, &reply.ID, h.cfg, h.Repo)

	return c.JSON(http.StatusCreated, reply)
}

// GetReplies retrieves replies for a post
func (h *ReplyHandler) GetReplies(c echo.Context) error {
	postID := c.Param("id")

	// Get optional user context for liked status
	var userDID string
	if did, ok := c.Get("did").(string); ok {
		userDID = did
	}

	allowed, accessErr := h.PostRepo.CanView(c.Request().Context(), postID, userDID)
	if accessErr != nil || !allowed {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Post not found"})
	}
	limit, offset := 20, 0
	if n, err := strconv.Atoi(c.QueryParam("limit")); err == nil && n > 0 {
		limit = n
	}
	if limit > 50 {
		limit = 50
	}
	if n, err := strconv.Atoi(c.QueryParam("offset")); err == nil && n >= 0 {
		offset = n
	}
	replies, err := h.Repo.GetByPostIDPage(c.Request().Context(), postID, userDID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch replies"})
	}

	if len(replies) == limit {
		c.Response().Header().Set("X-Next-Offset", strconv.Itoa(offset+limit))
	}
	return c.JSON(http.StatusOK, replies)
}
