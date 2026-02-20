package handlers

import (
	"encoding/json"
	"net/http"

	"splitter/internal/config"
	"splitter/internal/db"
	"splitter/internal/repository"

	"github.com/labstack/echo/v4"
)

// OutboxHandler handles ActivityPub outbox queries
type OutboxHandler struct {
	userRepo *repository.UserRepository
	cfg      *config.Config
}

// NewOutboxHandler creates a new OutboxHandler
func NewOutboxHandler(userRepo *repository.UserRepository, cfg *config.Config) *OutboxHandler {
	return &OutboxHandler{userRepo: userRepo, cfg: cfg}
}

// GetOutbox returns the outbox for a user (OrderedCollection of activities)
// GET /users/:username/outbox
func (h *OutboxHandler) GetOutbox(c echo.Context) error {
	username := c.Param("username")

	// Verify user exists
	user, _, err := h.userRepo.GetByUsername(c.Request().Context(), username)
	if err != nil || user == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "user not found",
		})
	}

	// Get recent outbox activities
	rows, err := db.GetDB().Query(c.Request().Context(),
		`SELECT payload FROM outbox_activities
		 WHERE status = 'sent'
		 ORDER BY created_at DESC LIMIT 20`)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to get outbox",
		})
	}
	defer rows.Close()

	var items []interface{}
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err == nil {
			var item interface{}
			if err := json.Unmarshal(payload, &item); err == nil {
				items = append(items, item)
			}
		}
	}

	collection := map[string]interface{}{
		"@context":     "https://www.w3.org/ns/activitystreams",
		"id":           h.cfg.Federation.URL + "/users/" + username + "/outbox",
		"type":         "OrderedCollection",
		"totalItems":   len(items),
		"orderedItems": items,
	}

	c.Response().Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	return c.JSON(http.StatusOK, collection)
}
