package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"splitter/internal/config"
	"splitter/internal/repository"

	"github.com/labstack/echo/v4"
)

// WebFingerHandler handles WebFinger discovery
type WebFingerHandler struct {
	userRepo *repository.UserRepository
	cfg      *config.Config
}

// NewWebFingerHandler creates a new WebFingerHandler
func NewWebFingerHandler(userRepo *repository.UserRepository, cfg *config.Config) *WebFingerHandler {
	return &WebFingerHandler{userRepo: userRepo, cfg: cfg}
}

// WebFingerResponse represents the JRD (JSON Resource Descriptor) response
type WebFingerResponse struct {
	Subject string          `json:"subject"`
	Links   []WebFingerLink `json:"links"`
}

// WebFingerLink represents a link in the WebFinger response
type WebFingerLink struct {
	Rel  string `json:"rel"`
	Type string `json:"type,omitempty"`
	Href string `json:"href,omitempty"`
}

// Handle processes WebFinger requests
// GET /.well-known/webfinger?resource=acct:alice@splitter-1
func (h *WebFingerHandler) Handle(c echo.Context) error {
	resource := c.QueryParam("resource")
	if resource == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "resource parameter required",
		})
	}

	// Parse acct:username@domain
	if !strings.HasPrefix(resource, "acct:") {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "resource must use acct: scheme",
		})
	}

	acct := strings.TrimPrefix(resource, "acct:")
	parts := strings.SplitN(acct, "@", 2)
	if len(parts) != 2 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid acct format, expected user@domain",
		})
	}

	username := parts[0]
	domain := parts[1]

	// Verify this is our domain
	if domain != h.cfg.Federation.Domain {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("domain '%s' not served here (this is %s)", domain, h.cfg.Federation.Domain),
		})
	}

	// Look up user
	user, _, err := h.userRepo.GetByUsername(c.Request().Context(), username)
	if err != nil || user == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "user not found",
		})
	}

	// Build response
	actorURL := fmt.Sprintf("%s/ap/users/%s", h.cfg.Federation.URL, username)

	response := WebFingerResponse{
		Subject: resource,
		Links: []WebFingerLink{
			{
				Rel:  "self",
				Type: "application/activity+json",
				Href: actorURL,
			},
		},
	}

	c.Response().Header().Set("Content-Type", "application/jrd+json; charset=utf-8")
	return c.JSON(http.StatusOK, response)
}
