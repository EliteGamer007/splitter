package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"splitter/internal/db"
	"splitter/internal/repository"

	"github.com/labstack/echo/v4"
)

// AdminAction represents an admin action in the audit log
type AdminAction struct {
	ID         string    `json:"id"`
	AdminID    string    `json:"admin_id"`
	ActionType string    `json:"action_type"`
	Target     string    `json:"target,omitempty"`
	Reason     string    `json:"reason,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// AdminHandler handles admin-related requests
type AdminHandler struct {
	userRepo *repository.UserRepository
}

// NewAdminHandler creates a new AdminHandler
func NewAdminHandler(userRepo *repository.UserRepository) *AdminHandler {
	return &AdminHandler{
		userRepo: userRepo,
	}
}

// logAdminAction logs an admin action to the audit log
func (h *AdminHandler) logAdminAction(adminID, actionType, target, reason string) error {
	query := `
		INSERT INTO admin_actions (admin_id, action_type, target, reason)
		VALUES ($1, $2, $3, $4)
	`
	_, err := db.GetDB().Exec(context.Background(), query, adminID, actionType, target, reason)
	return err
}

// requireAdmin checks if the current user is an admin
func (h *AdminHandler) requireAdmin(c echo.Context) error {
	role := c.Get("role")
	if role == nil || role.(string) != "admin" {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Admin access required",
		})
	}
	return nil
}

// requireModOrAdmin checks if the current user is a moderator or admin
func (h *AdminHandler) requireModOrAdmin(c echo.Context) error {
	role := c.Get("role")
	if role == nil || (role.(string) != "admin" && role.(string) != "moderator") {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Moderator or admin access required",
		})
	}
	return nil
}

// GetAllUsers returns all users (admin only)
func (h *AdminHandler) GetAllUsers(c echo.Context) error {
	if err := h.requireAdmin(c); err != nil {
		return err
	}

	limit := 50
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

	users, total, err := h.userRepo.GetAllUsers(c.Request().Context(), limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get users: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"users":  users,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetModerationRequests returns all pending moderation requests (admin only)
func (h *AdminHandler) GetModerationRequests(c echo.Context) error {
	if err := h.requireAdmin(c); err != nil {
		return err
	}

	users, err := h.userRepo.GetModerationRequests(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get moderation requests: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"requests": users,
	})
}

// ApproveModerationRequest approves a user's moderation request (admin only)
func (h *AdminHandler) ApproveModerationRequest(c echo.Context) error {
	if err := h.requireAdmin(c); err != nil {
		return err
	}

	userID := c.Param("id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "User ID required",
		})
	}

	err := h.userRepo.ApproveModerationRequest(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to approve request: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Moderation request approved",
	})
}

// RejectModerationRequest rejects a user's moderation request (admin only)
func (h *AdminHandler) RejectModerationRequest(c echo.Context) error {
	if err := h.requireAdmin(c); err != nil {
		return err
	}

	userID := c.Param("id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "User ID required",
		})
	}

	err := h.userRepo.RejectModerationRequest(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to reject request: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Moderation request rejected",
	})
}

// UpdateUserRole updates a user's role (admin only)
func (h *AdminHandler) UpdateUserRole(c echo.Context) error {
	if err := h.requireAdmin(c); err != nil {
		return err
	}

	userID := c.Param("id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "User ID required",
		})
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.Role != "user" && req.Role != "moderator" && req.Role != "admin" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid role. Must be 'user', 'moderator', or 'admin'",
		})
	}

	err := h.userRepo.UpdateUserRole(c.Request().Context(), userID, req.Role)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update role: " + err.Error(),
		})
	}

	// Log the role change action
	adminID := c.Get("user_id").(string)
	h.logAdminAction(adminID, "role_change", userID, "Role changed to "+req.Role)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "User role updated to " + req.Role,
	})
}

// SuspendUser suspends a user (moderator or admin)
func (h *AdminHandler) SuspendUser(c echo.Context) error {
	if err := h.requireModOrAdmin(c); err != nil {
		return err
	}

	userID := c.Param("id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "User ID required",
		})
	}

	// Parse optional reason from body
	var req struct {
		Reason string `json:"reason"`
	}
	c.Bind(&req) // Ignore errors, reason is optional

	err := h.userRepo.SuspendUser(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to suspend user: " + err.Error(),
		})
	}

	// Log the suspend action
	adminID := c.Get("user_id").(string)
	h.logAdminAction(adminID, "suspend", userID, req.Reason)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "User suspended",
	})
}

// UnsuspendUser unsuspends a user (moderator or admin)
func (h *AdminHandler) UnsuspendUser(c echo.Context) error {
	if err := h.requireModOrAdmin(c); err != nil {
		return err
	}

	userID := c.Param("id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "User ID required",
		})
	}

	err := h.userRepo.UnsuspendUser(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to unsuspend user: " + err.Error(),
		})
	}

	// Log the unsuspend action
	adminID := c.Get("user_id").(string)
	h.logAdminAction(adminID, "unsuspend", userID, "")

	return c.JSON(http.StatusOK, map[string]string{
		"message": "User unsuspended",
	})
}

// RequestModeration allows a user to request moderation privileges
func (h *AdminHandler) RequestModeration(c echo.Context) error {
	userID := c.Get("user_id").(string)

	err := h.userRepo.RequestModeration(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to submit request: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Moderation request submitted",
	})
}

// SearchUsers searches for users by username
func (h *AdminHandler) SearchUsers(c echo.Context) error {
	query := c.QueryParam("q")
	if query == "" || len(query) < 2 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Search query must be at least 2 characters",
		})
	}

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

	users, err := h.userRepo.SearchUsers(c.Request().Context(), query, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to search users: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"users": users,
	})
}

// GetAdminActions returns the admin action audit log
func (h *AdminHandler) GetAdminActions(c echo.Context) error {
	if err := h.requireAdmin(c); err != nil {
		return err
	}

	limit := 50
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

	query := `
		SELECT 
			a.id, 
			a.admin_id, 
			a.action_type, 
			a.target, 
			a.reason, 
			a.created_at,
			u.username as target_username
		FROM admin_actions a
		LEFT JOIN users u ON a.target = u.id::text
		ORDER BY a.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := db.GetDB().Query(c.Request().Context(), query, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get admin actions: " + err.Error(),
		})
	}
	defer rows.Close()

	var actions []AdminAction
	for rows.Next() {
		var action AdminAction
		var target, reason, targetUsername *string
		if err := rows.Scan(&action.ID, &action.AdminID, &action.ActionType, &target, &reason, &action.CreatedAt, &targetUsername); err != nil {
			continue
		}
		// Use username if available, otherwise fall back to UUID
		if targetUsername != nil && *targetUsername != "" {
			action.Target = "@" + *targetUsername
		} else if target != nil {
			action.Target = *target
		}
		if reason != nil {
			action.Reason = *reason
		}
		actions = append(actions, action)
	}

	if actions == nil {
		actions = []AdminAction{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"actions": actions,
	})
}

// GetSuspendedUsers returns all suspended users
func (h *AdminHandler) GetSuspendedUsers(c echo.Context) error {
	if err := h.requireModOrAdmin(c); err != nil {
		return err
	}

	limit := 50
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

	users, err := h.userRepo.GetSuspendedUsers(c.Request().Context(), limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get suspended users: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"users": users,
	})
}

// GetModerationQueue returns content moderation queue (stub/placeholder for Sprint 2+)
// TODO: Implement actual content moderation queue with reports, flagged content, etc.
func (h *AdminHandler) GetModerationQueue(c echo.Context) error {
	if err := h.requireModOrAdmin(c); err != nil {
		return err
	}

	// Placeholder response - content moderation queue not yet implemented
	return c.JSON(http.StatusOK, map[string]interface{}{
		"items": []interface{}{},
		"total": 0,
		"note":  "Content moderation queue is a placeholder feature for Sprint 2+",
	})
}
