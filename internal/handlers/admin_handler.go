package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"splitter/internal/db"
	"splitter/internal/repository"
	"splitter/internal/security"

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

	query := `
		SELECT
			r.id::text,
			COALESCE(r.post_id::text, ''),
			COALESCE(p.content, ''),
			COALESCE(u.id::text, ''),
			COALESCE(u.username, ''),
			COALESCE(u.instance_domain, ''),
			COALESCE(r.reason, 'reported'),
			COALESCE(p.is_remote, false),
			r.created_at
		FROM reports r
		LEFT JOIN posts p ON p.id = r.post_id
		LEFT JOIN users u ON u.did = p.author_did
		WHERE COALESCE(r.status, 'pending') = 'pending'
		ORDER BY r.created_at DESC
		LIMIT 200
	`

	rows, err := db.GetDB().Query(c.Request().Context(), query)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch moderation queue: " + err.Error(),
		})
	}
	defer rows.Close()

	items := make([]map[string]interface{}, 0)
	for rows.Next() {
		var (
			id, postID, preview, authorID, username, instanceDomain, reason string
			isFederated                                                     bool
			createdAt                                                       time.Time
		)

		if scanErr := rows.Scan(&id, &postID, &preview, &authorID, &username, &instanceDomain, &reason, &isFederated, &createdAt); scanErr != nil {
			continue
		}

		trimmedPreview := strings.TrimSpace(preview)
		if len(trimmedPreview) > 140 {
			trimmedPreview = trimmedPreview[:140] + "..."
		}

		items = append(items, map[string]interface{}{
			"id":           id,
			"post_id":      postID,
			"preview":      trimmedPreview,
			"content":      preview,
			"author_id":    authorID,
			"author":       username,
			"server":       instanceDomain,
			"reason":       reason,
			"is_federated": isFederated,
			"created_at":   createdAt,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"items": items,
		"total": len(items),
	})
}

// ApproveModerationItem marks a report as resolved without removing content
func (h *AdminHandler) ApproveModerationItem(c echo.Context) error {
	if err := h.requireModOrAdmin(c); err != nil {
		return err
	}

	reportID := c.Param("id")
	if reportID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Report ID required"})
	}

	result, err := db.GetDB().Exec(c.Request().Context(),
		`UPDATE reports SET status = 'resolved', resolved_at = now() WHERE id = $1`, reportID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to approve report: " + err.Error()})
	}
	if result.RowsAffected() == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Report not found"})
	}

	adminID := c.Get("user_id").(string)
	h.logAdminAction(adminID, "approve_report", reportID, "")

	return c.JSON(http.StatusOK, map[string]string{"message": "Report approved and resolved"})
}

// RemoveModerationContent removes flagged content and resolves report
func (h *AdminHandler) RemoveModerationContent(c echo.Context) error {
	if err := h.requireModOrAdmin(c); err != nil {
		return err
	}

	reportID := c.Param("id")
	if reportID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Report ID required"})
	}

	tx, err := db.GetDB().Begin(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to start transaction"})
	}
	defer tx.Rollback(c.Request().Context())

	var postID string
	if err := tx.QueryRow(c.Request().Context(), `SELECT COALESCE(post_id::text, '') FROM reports WHERE id = $1`, reportID).Scan(&postID); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Report not found"})
	}

	if postID != "" {
		if _, err := tx.Exec(c.Request().Context(), `UPDATE posts SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`, postID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to remove content"})
		}
	}

	if _, err := tx.Exec(c.Request().Context(), `UPDATE reports SET status = 'resolved', resolved_at = now() WHERE id = $1`, reportID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to resolve report"})
	}

	if err := tx.Commit(c.Request().Context()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to commit moderation action"})
	}

	adminID := c.Get("user_id").(string)
	h.logAdminAction(adminID, "remove_content", postID, "report:"+reportID)

	return c.JSON(http.StatusOK, map[string]string{"message": "Flagged content removed"})
}

// WarnUser logs a warning action for audit purposes
func (h *AdminHandler) WarnUser(c echo.Context) error {
	if err := h.requireModOrAdmin(c); err != nil {
		return err
	}

	userID := c.Param("id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "User ID required"})
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.Bind(&req)

	adminID := c.Get("user_id").(string)
	h.logAdminAction(adminID, "warn_user", userID, req.Reason)

	return c.JSON(http.StatusOK, map[string]string{"message": "User warning logged"})
}

// BlockDomain blocks a remote domain from federation delivery
func (h *AdminHandler) BlockDomain(c echo.Context) error {
	if err := h.requireModOrAdmin(c); err != nil {
		return err
	}

	var req struct {
		Domain string `json:"domain"`
		Reason string `json:"reason"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}

	domain := strings.TrimSpace(strings.ToLower(req.Domain))
	if domain == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Domain required"})
	}

	adminID := c.Get("user_id").(string)
	if _, err := db.GetDB().Exec(c.Request().Context(),
		`INSERT INTO blocked_domains(domain, reason, blocked_by, blocked_at)
		 VALUES ($1, $2, $3, now())
		 ON CONFLICT (domain) DO UPDATE SET
		   reason = EXCLUDED.reason,
		   blocked_by = EXCLUDED.blocked_by,
		   blocked_at = now()`,
		domain, req.Reason, adminID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to block domain: " + err.Error()})
	}

	h.logAdminAction(adminID, "block_domain", domain, req.Reason)

	return c.JSON(http.StatusOK, map[string]string{"message": "Domain blocked: " + domain})
}

// UnblockDomain removes a blocked domain from federation blocklist
func (h *AdminHandler) UnblockDomain(c echo.Context) error {
	if err := h.requireModOrAdmin(c); err != nil {
		return err
	}

	domain := strings.TrimSpace(strings.ToLower(c.Param("domain")))
	if domain == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Domain required"})
	}

	result, err := db.GetDB().Exec(c.Request().Context(), `DELETE FROM blocked_domains WHERE domain = $1`, domain)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to unblock domain: " + err.Error()})
	}

	if result.RowsAffected() == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Domain not found in block list"})
	}

	adminID := c.Get("user_id").(string)
	h.logAdminAction(adminID, "unblock_domain", domain, "")

	return c.JSON(http.StatusOK, map[string]string{"message": "Domain unblocked: " + domain})
}

// GetBlockedDomains returns currently blocked domains
func (h *AdminHandler) GetBlockedDomains(c echo.Context) error {
	if err := h.requireModOrAdmin(c); err != nil {
		return err
	}

	rows, err := db.GetDB().Query(c.Request().Context(),
		`SELECT domain, COALESCE(reason, ''), blocked_at, blocked_by
		 FROM blocked_domains
		 ORDER BY blocked_at DESC`)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch blocked domains: " + err.Error()})
	}
	defer rows.Close()

	domains := make([]map[string]interface{}, 0)
	for rows.Next() {
		var domain, reason, blockedBy string
		var blockedAt time.Time
		if scanErr := rows.Scan(&domain, &reason, &blockedAt, &blockedBy); scanErr != nil {
			continue
		}

		domains = append(domains, map[string]interface{}{
			"domain":     domain,
			"reason":     reason,
			"blocked_at": blockedAt.UTC().Format(time.RFC3339),
			"blocked_by": blockedBy,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"domains": domains,
	})
}

// GetFederationInspector returns live federation traffic and instance health metrics
func (h *AdminHandler) GetFederationInspector(c echo.Context) error {
	if err := h.requireModOrAdmin(c); err != nil {
		return err
	}

	ctx := c.Request().Context()

	incomingPerMinute := 0
	outgoingPerMinute := 0
	retryQueue := 0
	validationRate := 100.0

	_ = db.GetDB().QueryRow(ctx, `SELECT COUNT(*) FROM inbox_activities WHERE received_at > now() - interval '1 minute'`).Scan(&incomingPerMinute)
	_ = db.GetDB().QueryRow(ctx, `SELECT COUNT(*) FROM outbox_activities WHERE created_at > now() - interval '1 minute'`).Scan(&outgoingPerMinute)
	_ = db.GetDB().QueryRow(ctx, `SELECT COUNT(*) FROM outbox_activities WHERE status IN ('pending','failed')`).Scan(&retryQueue)
	_ = db.GetDB().QueryRow(ctx, `
		SELECT COALESCE(
			ROUND(
				(SUM(CASE WHEN status = 'sent' THEN 1 ELSE 0 END)::numeric / NULLIF(COUNT(*), 0)) * 100,
				2
			),
			100
		)
		FROM outbox_activities
		WHERE created_at > now() - interval '1 hour'
	`).Scan(&validationRate)

	type serverRow struct {
		Domain          string
		Blocked         bool
		LastSeen        *time.Time
		IncomingM       int
		OutgoingM       int
		FailedH         int
		RetryQueue      int
		CircuitOpen     bool
		ReputationScore int
	}

	serverRows, err := db.GetDB().Query(ctx, `
		WITH domains AS (
			SELECT DISTINCT domain FROM remote_actors WHERE domain IS NOT NULL AND domain != ''
			UNION
			SELECT DISTINCT domain FROM blocked_domains
			UNION
			SELECT DISTINCT domain FROM instance_reputation
			UNION
			SELECT DISTINCT domain FROM federation_failures
			UNION
			SELECT DISTINCT regexp_replace(target_inbox, '^https?://([^/]+)/?.*$', '\\1') AS domain
			FROM outbox_activities
			WHERE target_inbox ILIKE 'http%'
		)
		SELECT
			d.domain,
			EXISTS(SELECT 1 FROM blocked_domains b WHERE b.domain = d.domain) AS blocked,
			GREATEST(
				COALESCE((SELECT MAX(i.received_at) FROM inbox_activities i WHERE i.actor_uri ILIKE '%' || d.domain || '%'), 'epoch'::timestamptz),
				COALESCE((SELECT MAX(o.created_at) FROM outbox_activities o WHERE o.target_inbox ILIKE '%' || d.domain || '%'), 'epoch'::timestamptz),
				COALESCE((SELECT MAX(r.last_fetched_at) FROM remote_actors r WHERE r.domain = d.domain), 'epoch'::timestamptz)
			) AS last_seen,
			COALESCE((SELECT COUNT(*) FROM inbox_activities i WHERE i.actor_uri ILIKE '%' || d.domain || '%' AND i.received_at > now() - interval '1 minute'), 0) AS incoming_m,
			COALESCE((SELECT COUNT(*) FROM outbox_activities o WHERE o.target_inbox ILIKE '%' || d.domain || '%' AND o.created_at > now() - interval '1 minute'), 0) AS outgoing_m,
			COALESCE((SELECT COUNT(*) FROM outbox_activities o WHERE o.target_inbox ILIKE '%' || d.domain || '%' AND o.status = 'failed' AND o.created_at > now() - interval '1 hour'), 0) AS failed_h,
			COALESCE((SELECT COUNT(*) FROM outbox_activities o WHERE o.target_inbox ILIKE '%' || d.domain || '%' AND o.status IN ('pending','failed')), 0) AS retry_queue,
			COALESCE((SELECT ff.circuit_open_until > now() FROM federation_failures ff WHERE ff.domain = d.domain), false) AS circuit_open,
			COALESCE((SELECT ir.reputation_score FROM instance_reputation ir WHERE ir.domain = d.domain), 100) AS reputation_score
		FROM domains d
		WHERE d.domain IS NOT NULL AND d.domain != ''
		ORDER BY d.domain
	`)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch federation server stats: " + err.Error()})
	}
	defer serverRows.Close()

	servers := make([]map[string]interface{}, 0)
	for serverRows.Next() {
		row := serverRow{}
		if scanErr := serverRows.Scan(&row.Domain, &row.Blocked, &row.LastSeen, &row.IncomingM, &row.OutgoingM, &row.FailedH, &row.RetryQueue, &row.CircuitOpen, &row.ReputationScore); scanErr != nil {
			continue
		}

		status := "healthy"
		if row.Blocked {
			status = "blocked"
		} else if row.CircuitOpen {
			status = "circuit_open"
		} else if row.FailedH > 2 {
			status = "degraded"
		}

		lastSeen := "—"
		if row.LastSeen != nil && !row.LastSeen.IsZero() && row.LastSeen.Year() > 1970 {
			lastSeen = row.LastSeen.UTC().Format(time.RFC3339)
		}

		servers = append(servers, map[string]interface{}{
			"domain":           row.Domain,
			"status":           status,
			"reputation":       map[bool]string{true: "Blocked", false: "Trusted"}[row.Blocked],
			"reputation_score": row.ReputationScore,
			"last_seen":        lastSeen,
			"incoming_m":       row.IncomingM,
			"outgoing_m":       row.OutgoingM,
			"retry_queue":      row.RetryQueue,
			"failed_h":         row.FailedH,
			"circuit_open":     row.CircuitOpen,
			"activities_m":     row.IncomingM + row.OutgoingM,
		})
	}

	failingRows, err := db.GetDB().Query(ctx, `
		WITH parsed AS (
			SELECT
				regexp_replace(target_inbox, '^https?://([^/]+)/?.*$', '\\1') AS domain,
				retry_count,
				status,
				next_retry_at,
				last_error
			FROM outbox_activities
			WHERE target_inbox ILIKE 'http%'
			  AND status IN ('pending','failed')
		)
		SELECT
			domain,
			COUNT(*) AS queued,
			MAX(retry_count) AS max_retry_count,
			MAX(next_retry_at) AS next_retry_at,
			MAX(COALESCE(last_error, '')) AS last_error,
			COALESCE((SELECT ff.circuit_open_until FROM federation_failures ff WHERE ff.domain = parsed.domain), NULL) AS circuit_open_until
		FROM parsed
		WHERE COALESCE(domain, '') <> ''
		GROUP BY domain
		ORDER BY queued DESC, domain ASC
	`)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch failing domains: " + err.Error()})
	}
	defer failingRows.Close()

	failingDomains := make([]map[string]interface{}, 0)
	for failingRows.Next() {
		var domain string
		var queued, maxRetryCount int
		var nextRetryAt *time.Time
		var lastError string
		var circuitOpenUntil *time.Time

		if scanErr := failingRows.Scan(&domain, &queued, &maxRetryCount, &nextRetryAt, &lastError, &circuitOpenUntil); scanErr != nil {
			continue
		}

		nextRetry := "—"
		if nextRetryAt != nil {
			nextRetry = nextRetryAt.UTC().Format(time.RFC3339)
		}

		circuitUntil := "—"
		if circuitOpenUntil != nil {
			circuitUntil = circuitOpenUntil.UTC().Format(time.RFC3339)
		}

		failingDomains = append(failingDomains, map[string]interface{}{
			"domain":             domain,
			"queued":             queued,
			"max_retry_count":    maxRetryCount,
			"next_retry_at":      nextRetry,
			"last_error":         lastError,
			"circuit_open_until": circuitUntil,
		})
	}

	inboxRows, err := db.GetDB().Query(ctx, `
		SELECT actor_uri, activity_type, received_at
		FROM inbox_activities
		ORDER BY received_at DESC
		LIMIT 20
	`)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch inbox activity: " + err.Error()})
	}
	defer inboxRows.Close()

	recentIncoming := make([]map[string]interface{}, 0)
	for inboxRows.Next() {
		var actorURI, activityType string
		var receivedAt time.Time
		if scanErr := inboxRows.Scan(&actorURI, &activityType, &receivedAt); scanErr != nil {
			continue
		}
		recentIncoming = append(recentIncoming, map[string]interface{}{
			"direction": "incoming",
			"actor_uri": actorURI,
			"type":      activityType,
			"time":      receivedAt.UTC().Format(time.RFC3339),
		})
	}

	outboxRows, err := db.GetDB().Query(ctx, `
		SELECT target_inbox, activity_type, status, created_at
		FROM outbox_activities
		ORDER BY created_at DESC
		LIMIT 20
	`)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch outbox activity: " + err.Error()})
	}
	defer outboxRows.Close()

	recentOutgoing := make([]map[string]interface{}, 0)
	for outboxRows.Next() {
		var targetInbox, activityType, status string
		var createdAt time.Time
		if scanErr := outboxRows.Scan(&targetInbox, &activityType, &status, &createdAt); scanErr != nil {
			continue
		}
		recentOutgoing = append(recentOutgoing, map[string]interface{}{
			"direction":    "outgoing",
			"target_inbox": targetInbox,
			"type":         activityType,
			"status":       status,
			"time":         createdAt.UTC().Format(time.RFC3339),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"metrics": map[string]interface{}{
			"incoming_per_minute":  incomingPerMinute,
			"outgoing_per_minute":  outgoingPerMinute,
			"signature_validation": fmt.Sprintf("%.2f%%", validationRate),
			"retry_queue":          retryQueue,
			"failing_domains":      len(failingDomains),
		},
		"servers":         servers,
		"failing_domains": failingDomains,
		"recent_incoming": recentIncoming,
		"recent_outgoing": recentOutgoing,
	})
}

// GetInstanceReputation returns per-domain reputation metrics for admin governance.
func (h *AdminHandler) GetInstanceReputation(c echo.Context) error {
	if err := h.requireModOrAdmin(c); err != nil {
		return err
	}

	rows, err := db.GetDB().Query(c.Request().Context(), `
		SELECT domain, reputation_score, spam_count, failure_count, updated_at
		FROM instance_reputation
		ORDER BY reputation_score ASC, domain ASC
	`)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch instance reputation: " + err.Error()})
	}
	defer rows.Close()

	items := make([]map[string]interface{}, 0)
	for rows.Next() {
		var domain string
		var score, spamCount, failureCount int
		var updatedAt time.Time
		if scanErr := rows.Scan(&domain, &score, &spamCount, &failureCount, &updatedAt); scanErr != nil {
			continue
		}
		items = append(items, map[string]interface{}{
			"domain":           domain,
			"reputation_score": score,
			"spam_count":       spamCount,
			"failure_count":    failureCount,
			"updated_at":       updatedAt.UTC().Format(time.RFC3339),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"items": items})
}

// GetFederationNetwork returns graph-friendly server relationship data.
func (h *AdminHandler) GetFederationNetwork(c echo.Context) error {
	if err := h.requireModOrAdmin(c); err != nil {
		return err
	}

	ctx := c.Request().Context()

	type edgeRow struct {
		Source       string
		Target       string
		SuccessCount int
		FailureCount int
		LastStatus   string
		LastSeen     time.Time
	}

	rows, err := db.GetDB().Query(ctx, `
		SELECT source_domain, target_domain, success_count, failure_count, COALESCE(last_status, 'pending'), last_seen
		FROM federation_connections
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch federation graph: " + err.Error()})
	}
	defer rows.Close()

	nodeSet := map[string]bool{}
	edges := make([]map[string]interface{}, 0)

	for rows.Next() {
		r := edgeRow{}
		if scanErr := rows.Scan(&r.Source, &r.Target, &r.SuccessCount, &r.FailureCount, &r.LastStatus, &r.LastSeen); scanErr != nil {
			continue
		}

		if strings.TrimSpace(r.Source) == "" || strings.TrimSpace(r.Target) == "" {
			continue
		}

		nodeSet[r.Source] = true
		nodeSet[r.Target] = true

		weight := r.SuccessCount + r.FailureCount
		if weight < 1 {
			weight = 1
		}

		edges = append(edges, map[string]interface{}{
			"source":        r.Source,
			"target":        r.Target,
			"weight":        weight,
			"success_count": r.SuccessCount,
			"failure_count": r.FailureCount,
			"last_status":   r.LastStatus,
			"last_seen":     r.LastSeen.UTC().Format(time.RFC3339),
		})
	}

	selfDomain := strings.TrimSpace(c.QueryParam("self"))
	if selfDomain == "" {
		selfDomain = "local"
	}
	nodeSet[selfDomain] = true

	nodes := make([]map[string]interface{}, 0, len(nodeSet))
	for domain := range nodeSet {
		nodeType := "remote"
		if domain == selfDomain {
			nodeType = "local"
		}

		nodes = append(nodes, map[string]interface{}{
			"id":   domain,
			"type": nodeType,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	})
}

// GetMessagingSecurity returns messaging rate-limit and suspicious-event telemetry.
func (h *AdminHandler) GetMessagingSecurity(c echo.Context) error {
	if err := h.requireModOrAdmin(c); err != nil {
		return err
	}

	snapshot := security.GetMessagingGuard().Snapshot()

	return c.JSON(http.StatusOK, map[string]interface{}{
		"limits":        snapshot.Limits,
		"metrics":       snapshot.Metrics,
		"recent_events": snapshot.RecentEvents,
	})
}

// ─── AI Moderation ──────────────────────────────────────────────────────────

// aiScreenPost calls Gemini to classify post content as "remove" or "allow".
// Returns verdict and a brief reason. Falls back to "allow" on any error.
func aiScreenPost(content string) (verdict string, reason string) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "allow", "AI screening unavailable (no API key configured)"
	}

	prompt := `You are a strict content moderation AI for a social network.
Review the following post and respond ONLY with a JSON object in this exact format (no markdown, no code fences):
{"verdict":"remove","reason":"one sentence explanation","category":"category"}
OR
{"verdict":"allow","reason":"No violations","category":"none"}

Use verdict "remove" ONLY for: hate speech, explicit threats, graphic violence, severe harassment, or blatant spam.
For borderline or ambiguous content, use "allow".

Post content:
` + content

	type geminiPart struct {
		Text string `json:"text"`
	}
	type geminiContent struct {
		Parts []geminiPart `json:"parts"`
	}
	type geminiRequest struct {
		Contents         []geminiContent `json:"contents"`
		GenerationConfig map[string]any  `json:"generationConfig,omitempty"`
	}

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: map[string]any{
			"temperature":     0.1,
			"maxOutputTokens": 120,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "allow", "AI screening error: marshal failed"
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash-latest:generateContent?key=%s", apiKey)
	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "allow", "AI screening error: request creation failed"
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "allow", "AI screening error: API call failed"
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "allow", fmt.Sprintf("AI screening error: API returned %d", resp.StatusCode)
	}

	var gemResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(body, &gemResp); err != nil {
		return "allow", "AI screening error: parse failed"
	}

	if len(gemResp.Candidates) == 0 || len(gemResp.Candidates[0].Content.Parts) == 0 {
		return "allow", "AI screening: no response from model"
	}

	text := strings.TrimSpace(gemResp.Candidates[0].Content.Parts[0].Text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var result struct {
		Verdict string `json:"verdict"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return "allow", "AI screening error: could not parse verdict JSON"
	}

	if result.Verdict == "remove" {
		return "remove", result.Reason
	}
	return "allow", result.Reason
}

// ReportPost creates a moderation report for a post and triggers async AI screening.
// Route: POST /api/v1/posts/:id/report
func (h *AdminHandler) ReportPost(c echo.Context) error {
	postID := c.Param("id")
	if postID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Post ID required"})
	}

	reporterDID, _ := c.Get("did").(string)
	if reporterDID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.Bind(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Report reason required"})
	}

	validReasons := map[string]bool{
		"spam": true, "harassment": true, "inappropriate": true,
		"hate_speech": true, "misinformation": true,
	}
	if !validReasons[req.Reason] {
		req.Reason = "inappropriate"
	}

	var postContent string
	err := db.GetDB().QueryRow(c.Request().Context(),
		`SELECT COALESCE(content, '') FROM posts WHERE id = $1::uuid AND deleted_at IS NULL`, postID).Scan(&postContent)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Post not found"})
	}

	var reportID string
	err = db.GetDB().QueryRow(c.Request().Context(),
		`INSERT INTO reports (reporter_did, post_id, reason, status)
		 VALUES ($1, $2::uuid, $3, 'pending')
		 RETURNING id::text`,
		reporterDID, postID, req.Reason).Scan(&reportID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to submit report: " + err.Error()})
	}

	// Async AI screening
	capturedPostID := postID
	capturedReportID := reportID
	capturedContent := postContent
	go func() {
		verdict, aiReason := aiScreenPost(capturedContent)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if verdict == "remove" {
			_, _ = db.GetDB().Exec(ctx,
				`UPDATE posts SET deleted_at = now(), hidden_by_ai = true, hidden_reason = $1 WHERE id = $2::uuid AND deleted_at IS NULL`,
				aiReason, capturedPostID)
			_, _ = db.GetDB().Exec(ctx,
				`UPDATE reports SET status = 'ai_actioned', ai_verdict = 'remove', ai_reason = $1, ai_screened_at = now() WHERE id = $2::uuid`,
				aiReason, capturedReportID)
		} else {
			_, _ = db.GetDB().Exec(ctx,
				`UPDATE reports SET ai_verdict = 'allow', ai_reason = $1, ai_screened_at = now() WHERE id = $2::uuid`,
				aiReason, capturedReportID)
		}
	}()

	return c.JSON(http.StatusCreated, map[string]string{
		"message":   "Report submitted. AI moderation in progress.",
		"report_id": reportID,
	})
}

// GetAIActionsQueue returns posts auto-removed by AI moderation.
// Route: GET /admin/ai-actions
func (h *AdminHandler) GetAIActionsQueue(c echo.Context) error {
	if err := h.requireModOrAdmin(c); err != nil {
		return err
	}

	rows, err := db.GetDB().Query(c.Request().Context(), `
		SELECT
			r.id::text,
			COALESCE(r.post_id::text, ''),
			COALESCE(p.content, ''),
			COALESCE(u.id::text, ''),
			COALESCE(u.username, ''),
			COALESCE(u.instance_domain, ''),
			COALESCE(r.reason, 'reported'),
			COALESCE(r.ai_reason, ''),
			r.created_at,
			COALESCE(r.ai_screened_at, r.created_at),
			EXISTS(SELECT 1 FROM appeals a WHERE a.report_id = r.id AND a.status = 'pending') AS has_appeal
		FROM reports r
		LEFT JOIN posts p ON p.id = r.post_id
		LEFT JOIN users u ON u.did = p.author_did
		WHERE r.status = 'ai_actioned'
		ORDER BY r.created_at DESC
		LIMIT 200
	`)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch AI actions queue: " + err.Error()})
	}
	defer rows.Close()

	items := make([]map[string]interface{}, 0)
	for rows.Next() {
		var (
			id, postID, content, authorID, username, instanceDomain string
			reason, aiReason                                        string
			createdAt, aiScreenedAt                                 time.Time
			hasAppeal                                               bool
		)
		if scanErr := rows.Scan(&id, &postID, &content, &authorID, &username, &instanceDomain, &reason, &aiReason, &createdAt, &aiScreenedAt, &hasAppeal); scanErr != nil {
			continue
		}
		preview := strings.TrimSpace(content)
		if len(preview) > 140 {
			preview = preview[:140] + "..."
		}
		items = append(items, map[string]interface{}{
			"id":             id,
			"post_id":        postID,
			"preview":        preview,
			"content":        content,
			"author_id":      authorID,
			"author":         username,
			"server":         instanceDomain,
			"reason":         reason,
			"ai_reason":      aiReason,
			"created_at":     createdAt,
			"ai_screened_at": aiScreenedAt,
			"has_appeal":     hasAppeal,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"items": items,
		"total": len(items),
	})
}

// SubmitAppeal lets an authenticated user contest an AI-actioned content removal.
// Route: POST /api/v1/posts/:id/appeal
func (h *AdminHandler) SubmitAppeal(c echo.Context) error {
	postID := c.Param("id")
	if postID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Post ID required"})
	}

	userDID, _ := c.Get("did").(string)
	userID, _ := c.Get("user_id").(string)
	if userDID == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := c.Bind(&req); err != nil || strings.TrimSpace(req.Reason) == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Appeal reason required"})
	}

	// Find the ai_actioned report for this post
	var reportID string
	_ = db.GetDB().QueryRow(c.Request().Context(),
		`SELECT r.id::text FROM reports r
		 WHERE r.post_id = $1::uuid AND r.status = 'ai_actioned'
		 ORDER BY r.created_at DESC LIMIT 1`,
		postID).Scan(&reportID)

	if reportID == "" {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "No AI-actioned report found for this post"})
	}

	// Prevent duplicate pending appeals
	var existingID string
	_ = db.GetDB().QueryRow(c.Request().Context(),
		`SELECT id::text FROM appeals WHERE report_id = $1::uuid AND status = 'pending'`, reportID).Scan(&existingID)
	if existingID != "" {
		return c.JSON(http.StatusConflict, map[string]string{"error": "An appeal is already pending for this content"})
	}

	var appealID string
	err := db.GetDB().QueryRow(c.Request().Context(),
		`INSERT INTO appeals (report_id, post_id, appellant_did, appellant_id, appeal_reason)
		 VALUES ($1::uuid, $2::uuid, $3, $4::uuid, $5)
		 RETURNING id::text`,
		reportID, postID, userDID, userID, strings.TrimSpace(req.Reason)).Scan(&appealID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to submit appeal: " + err.Error()})
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"message":   "Appeal submitted. A moderator will review it shortly.",
		"appeal_id": appealID,
	})
}

// GetAppeals returns all content appeals for admin review.
// Route: GET /admin/appeals
func (h *AdminHandler) GetAppeals(c echo.Context) error {
	if err := h.requireModOrAdmin(c); err != nil {
		return err
	}

	rows, err := db.GetDB().Query(c.Request().Context(), `
		SELECT
			a.id::text,
			COALESCE(a.report_id::text, ''),
			COALESCE(a.post_id::text, ''),
			COALESCE(p.content, ''),
			COALESCE(u.id::text, ''),
			COALESCE(u.username, ''),
			COALESCE(r.ai_reason, ''),
			a.appeal_reason,
			a.status,
			a.created_at
		FROM appeals a
		LEFT JOIN reports r ON r.id = a.report_id
		LEFT JOIN posts p ON p.id = a.post_id
		LEFT JOIN users u ON u.did = a.appellant_did
		ORDER BY a.created_at DESC
		LIMIT 200
	`)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch appeals: " + err.Error()})
	}
	defer rows.Close()

	appeals := make([]map[string]interface{}, 0)
	for rows.Next() {
		var (
			id, reportID, postID, content, authorID, username string
			aiReason, appealReason, status                    string
			createdAt                                         time.Time
		)
		if scanErr := rows.Scan(&id, &reportID, &postID, &content, &authorID, &username, &aiReason, &appealReason, &status, &createdAt); scanErr != nil {
			continue
		}
		preview := strings.TrimSpace(content)
		if len(preview) > 140 {
			preview = preview[:140] + "..."
		}
		appeals = append(appeals, map[string]interface{}{
			"id":            id,
			"report_id":     reportID,
			"post_id":       postID,
			"preview":       preview,
			"author_id":     authorID,
			"author":        username,
			"ai_reason":     aiReason,
			"appeal_reason": appealReason,
			"status":        status,
			"created_at":    createdAt,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"appeals": appeals,
		"total":   len(appeals),
	})
}

// ResolveAppeal accepts or rejects a content appeal.
// Accept → restores the post. Reject → keeps it removed.
// Route: POST /admin/appeals/:id/resolve
func (h *AdminHandler) ResolveAppeal(c echo.Context) error {
	if err := h.requireModOrAdmin(c); err != nil {
		return err
	}

	appealID := c.Param("id")
	if appealID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Appeal ID required"})
	}

	var req struct {
		Decision string `json:"decision"` // "accept" or "reject"
		Note     string `json:"note"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
	}
	if req.Decision != "accept" && req.Decision != "reject" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Decision must be 'accept' or 'reject'"})
	}

	adminID := c.Get("user_id").(string)

	tx, err := db.GetDB().Begin(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Transaction failed"})
	}
	defer tx.Rollback(c.Request().Context())

	var postID, reportID string
	if err := tx.QueryRow(c.Request().Context(),
		`SELECT COALESCE(post_id::text, ''), COALESCE(report_id::text, '')
		 FROM appeals WHERE id = $1::uuid AND status = 'pending'`, appealID).Scan(&postID, &reportID); err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Appeal not found or already resolved"})
	}

	newStatus := map[string]string{"accept": "accepted", "reject": "rejected"}[req.Decision]
	if _, err := tx.Exec(c.Request().Context(),
		`UPDATE appeals SET status = $1, reviewer_id = $2, reviewer_note = $3, resolved_at = now() WHERE id = $4::uuid`,
		newStatus, adminID, req.Note, appealID); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update appeal"})
	}

	if req.Decision == "accept" && postID != "" {
		if _, err := tx.Exec(c.Request().Context(),
			`UPDATE posts SET deleted_at = NULL, hidden_by_ai = false, hidden_reason = NULL WHERE id = $1::uuid`, postID); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to restore post"})
		}
		if reportID != "" {
			_, _ = tx.Exec(c.Request().Context(), `UPDATE reports SET status = 'resolved' WHERE id = $1::uuid`, reportID)
		}
		h.logAdminAction(adminID, "appeal_accepted_restore", postID, req.Note)
	} else {
		if reportID != "" {
			_, _ = tx.Exec(c.Request().Context(), `UPDATE reports SET status = 'resolved' WHERE id = $1::uuid`, reportID)
		}
		h.logAdminAction(adminID, "appeal_rejected", postID, req.Note)
	}

	if err := tx.Commit(c.Request().Context()); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to commit"})
	}

	msg := map[string]string{
		"accept": "Appeal accepted. Content has been restored.",
		"reject": "Appeal rejected. Content remains removed.",
	}[req.Decision]
	return c.JSON(http.StatusOK, map[string]string{"message": msg})
}

// BanUser permanently bans a user (sets is_suspended = true).
// Route: POST /admin/users/:id/ban
func (h *AdminHandler) BanUser(c echo.Context) error {
	if err := h.requireAdmin(c); err != nil {
		return err
	}

	userID := c.Param("id")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "User ID required"})
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.Bind(&req)

	result, err := db.GetDB().Exec(c.Request().Context(),
		`UPDATE users SET is_suspended = true, updated_at = now() WHERE id = $1::uuid`, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to ban user: " + err.Error()})
	}
	if result.RowsAffected() == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	adminID := c.Get("user_id").(string)
	h.logAdminAction(adminID, "ban_user", userID, req.Reason)

	return c.JSON(http.StatusOK, map[string]string{"message": "User banned"})
}
