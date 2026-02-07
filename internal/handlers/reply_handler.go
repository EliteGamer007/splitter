package handlers

import (
	"net/http"

	"splitter/internal/models"
	"splitter/internal/repository"

	"github.com/labstack/echo/v4"
)

type ReplyHandler struct {
	Repo     *repository.ReplyRepository
	PostRepo *repository.PostRepository
}

func NewReplyHandler() *ReplyHandler {
	return &ReplyHandler{
		Repo:     repository.NewReplyRepository(),
		PostRepo: repository.NewPostRepository(),
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

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
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

	replies, err := h.Repo.GetByPostID(c.Request().Context(), postID, userDID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch replies"})
	}

	return c.JSON(http.StatusOK, replies)
}
