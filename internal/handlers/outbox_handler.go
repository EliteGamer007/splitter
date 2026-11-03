package handlers

import (
	"net/http"
	"strings"
	"time"

	"splitter/internal/config"
	"splitter/internal/federation"
	"splitter/internal/repository"

	"github.com/labstack/echo/v4"
)

// OutboxHandler publishes the actor's currently public local notes. The delivery
// queue also contains private messages and is never a public read model.
type OutboxHandler struct {
	userRepo *repository.UserRepository
	cfg      *config.Config
}

func NewOutboxHandler(userRepo *repository.UserRepository, cfg *config.Config) *OutboxHandler {
	return &OutboxHandler{userRepo: userRepo, cfg: cfg}
}
func (h *OutboxHandler) GetOutbox(c echo.Context) error {
	username := c.Param("username")
	user, _, err := h.userRepo.GetByUsername(c.Request().Context(), username)
	if err != nil || user == nil || user.IsSuspended {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}
	repo := repository.NewPostRepository()
	posts, err := repo.GetByAuthorDIDForViewer(c.Request().Context(), user.DID, "", 20, 0)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get outbox"})
	}
	base := strings.TrimRight(h.cfg.Federation.URL, "/")
	actor := base + "/ap/users/" + username
	items := make([]*federation.Activity, 0, len(posts))
	for _, post := range posts {
		if post.IsRemote || post.Visibility != "public" {
			continue
		}
		note := federation.Note{ID: base + "/posts/" + post.ID, Type: "Note", AttributedTo: actor, Content: post.Content, Published: post.CreatedAt.UTC().Format(time.RFC3339), To: []string{"https://www.w3.org/ns/activitystreams#Public"}, InReplyTo: post.InReplyToURI}
		for _, media := range post.Media {
			mediaURL := media.MediaURL
			if strings.HasPrefix(mediaURL, "/") {
				mediaURL = base + mediaURL
			}
			note.Attachment = append(note.Attachment, federation.Attachment{Type: "Document", MediaType: media.MediaType, URL: mediaURL})
		}
		items = append(items, &federation.Activity{Context: "https://www.w3.org/ns/activitystreams", ID: base + "/activities/create-" + post.ID, Type: "Create", Actor: actor, Object: note, To: note.To})
	}
	collection := map[string]any{"@context": "https://www.w3.org/ns/activitystreams", "id": actor + "/outbox", "type": "OrderedCollection", "totalItems": len(items), "orderedItems": items}
	c.Response().Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	return c.JSON(http.StatusOK, collection)
}
