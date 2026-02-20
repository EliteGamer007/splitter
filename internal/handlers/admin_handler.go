package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
		Domain    string
		Blocked   bool
		LastSeen  *time.Time
		IncomingM int
		OutgoingM int
		FailedH   int
	}

	serverRows, err := db.GetDB().Query(ctx, `
		WITH domains AS (
			SELECT DISTINCT domain FROM remote_actors WHERE domain IS NOT NULL AND domain != ''
			UNION
			SELECT DISTINCT domain FROM blocked_domains
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
			COALESCE((SELECT COUNT(*) FROM outbox_activities o WHERE o.target_inbox ILIKE '%' || d.domain || '%' AND o.status = 'failed' AND o.created_at > now() - interval '1 hour'), 0) AS failed_h
		FROM domains d
		ORDER BY d.domain
	`)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch federation server stats: " + err.Error()})
	}
	defer serverRows.Close()

	servers := make([]map[string]interface{}, 0)
	for serverRows.Next() {
		row := serverRow{}
		if scanErr := serverRows.Scan(&row.Domain, &row.Blocked, &row.LastSeen, &row.IncomingM, &row.OutgoingM, &row.FailedH); scanErr != nil {
			continue
		}

		status := "healthy"
		if row.Blocked {
			status = "blocked"
		} else if row.FailedH > 2 {
			status = "degraded"
		}

		lastSeen := "—"
		if row.LastSeen != nil && !row.LastSeen.IsZero() && row.LastSeen.Year() > 1970 {
			lastSeen = row.LastSeen.UTC().Format(time.RFC3339)
		}

		servers = append(servers, map[string]interface{}{
			"domain":       row.Domain,
			"status":       status,
			"reputation":   map[bool]string{true: "Blocked", false: "Trusted"}[row.Blocked],
			"last_seen":    lastSeen,
			"incoming_m":   row.IncomingM,
			"outgoing_m":   row.OutgoingM,
			"activities_m": row.IncomingM + row.OutgoingM,
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
		},
		"servers":         servers,
		"recent_incoming": recentIncoming,
		"recent_outgoing": recentOutgoing,
	})
}
