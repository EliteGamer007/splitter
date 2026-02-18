package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"splitter/internal/config"
	"splitter/internal/federation"
	"splitter/internal/repository"

	"github.com/labstack/echo/v4"
)

// ActorHandler handles ActivityPub Actor endpoints
type ActorHandler struct {
	userRepo *repository.UserRepository
	cfg      *config.Config
}

// NewActorHandler creates a new ActorHandler
func NewActorHandler(userRepo *repository.UserRepository, cfg *config.Config) *ActorHandler {
	return &ActorHandler{userRepo: userRepo, cfg: cfg}
}

// ActorResponse represents an ActivityPub Actor (Person) object
type ActorResponse struct {
	Context           interface{}     `json:"@context"`
	ID                string          `json:"id"`
	Type              string          `json:"type"`
	PreferredUsername string          `json:"preferredUsername"`
	Name              string          `json:"name"`
	Summary           string          `json:"summary,omitempty"`
	Inbox             string          `json:"inbox"`
	Outbox            string          `json:"outbox"`
	Followers         string          `json:"followers"`
	Following         string          `json:"following"`
	Icon              *ActorIcon      `json:"icon,omitempty"`
	PublicKey         *ActorPublicKey `json:"publicKey"`
}

// ActorIcon represents the actor's avatar
type ActorIcon struct {
	Type      string `json:"type"`
	MediaType string `json:"mediaType"`
	URL       string `json:"url"`
}

// ActorPublicKey represents the actor's public key for signature verification
type ActorPublicKey struct {
	ID           string `json:"id"`
	Owner        string `json:"owner"`
	PublicKeyPEM string `json:"publicKeyPem"`
}

// GetActor returns the ActivityPub Actor object for a user
// GET /users/:username (content-negotiated)
func (h *ActorHandler) GetActor(c echo.Context) error {
	username := c.Param("username")

	// Check Accept header — if requesting JSON, serve ActivityPub Actor
	accept := c.Request().Header.Get("Accept")
	if !strings.Contains(accept, "application/activity+json") &&
		!strings.Contains(accept, "application/ld+json") &&
		!strings.Contains(accept, "application/json") {
		// Not an ActivityPub request — could redirect to profile page
		return c.JSON(http.StatusNotAcceptable, map[string]string{
			"error": "This endpoint serves ActivityPub Actor data. Set Accept: application/activity+json",
		})
	}

	// Look up local user
	user, _, err := h.userRepo.GetByUsername(c.Request().Context(), username)
	if err != nil || user == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "user not found",
		})
	}

	baseURL := h.cfg.Federation.URL
	actorID := fmt.Sprintf("%s/ap/users/%s", baseURL, username)

	// Use instance public key for federation signatures
	publicKeyPEM := federation.GetInstancePublicKeyPEM()

	actor := ActorResponse{
		Context: []interface{}{
			"https://www.w3.org/ns/activitystreams",
			"https://w3id.org/security/v1",
		},
		ID:                actorID,
		Type:              "Person",
		PreferredUsername: username,
		Name:              user.DisplayName,
		Summary:           user.Bio,
		Inbox:             fmt.Sprintf("%s/ap/users/%s/inbox", baseURL, username),
		Outbox:            fmt.Sprintf("%s/ap/users/%s/outbox", baseURL, username),
		Followers:         fmt.Sprintf("%s/ap/users/%s/followers", baseURL, username),
		Following:         fmt.Sprintf("%s/ap/users/%s/following", baseURL, username),
		PublicKey: &ActorPublicKey{
			ID:           actorID + "#main-key",
			Owner:        actorID,
			PublicKeyPEM: publicKeyPEM,
		},
	}

	if user.AvatarURL != "" {
		actor.Icon = &ActorIcon{
			Type:      "Image",
			MediaType: "image/jpeg",
			URL:       user.AvatarURL,
		}
	}

	c.Response().Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	return c.JSON(http.StatusOK, actor)
}
