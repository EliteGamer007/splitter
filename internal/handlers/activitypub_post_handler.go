package handlers

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"splitter/internal/config"
	"splitter/internal/federation"
	"splitter/internal/repository"
	"strings"
	"time"
)

// FederationViewer authenticates signed remote reads without ever forwarding a
// local bearer token. Local JWT identity, if already established, takes priority.
func FederationViewer(cfg *config.Config) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if viewerDID(c) != "" || c.Request().Header.Get("Signature") == "" {
				return next(c)
			}
			if cfg == nil || !cfg.Federation.Enabled {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Signed federation reads are disabled"})
			}
			keyID := parseSignatureKeyID(c.Request().Header.Get("Signature"))
			actorURI := stripFragment(keyID)
			if actorURI == keyID {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid signing key"})
			}
			actor, err := resolveActorFromURI(actorURI)
			if err != nil || actor == nil || actor.ActorURI != actorURI || federation.VerifyRequest(c.Request(), actor.PublicKeyPEM) != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid federation signature"})
			}
			c.Set("did", actorURI)
			if user, err := repository.NewUserRepository().GetByDID(c.Request().Context(), actorURI); err == nil {
				c.Set("user_id", user.ID)
			}
			return next(c)
		}
	}
}

// GetActivityPubPost serves the canonical URI published in outgoing Note IDs.
func (h *PostHandler) GetActivityPubPost(c echo.Context) error {
	post, err := h.postRepo.GetByIDForViewer(c.Request().Context(), c.Param("id"), viewerDID(c))
	if err != nil || post.IsRemote {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Note not found"})
	}
	user, err := h.userRepo.GetByDID(c.Request().Context(), post.AuthorDID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Author not found"})
	}
	base := strings.TrimRight(h.cfg.Federation.URL, "/")
	note := federation.Note{ID: base + "/posts/" + post.ID, Type: "Note", AttributedTo: base + "/ap/users/" + user.Username, Content: post.Content, Published: post.CreatedAt.UTC().Format(time.RFC3339), InReplyTo: post.InReplyToURI, Visibility: post.Visibility}
	if post.Visibility == "public" {
		note.To = []string{"https://www.w3.org/ns/activitystreams#Public"}
	} else {
		note.To = []string{viewerDID(c)}
	}
	for _, m := range post.Media {
		raw := m.MediaURL
		if strings.HasPrefix(raw, "/") {
			raw = base + raw
		}
		note.Attachment = append(note.Attachment, federation.Attachment{Type: "Document", URL: raw, MediaType: m.MediaType})
	}
	c.Response().Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	c.Response().Header().Set("Cache-Control", "private, no-store")
	return c.JSON(http.StatusOK, note)
}
