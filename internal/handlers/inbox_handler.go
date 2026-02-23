package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"splitter/internal/config"
	"splitter/internal/db"
	"splitter/internal/federation"
	"splitter/internal/models"
	"splitter/internal/repository"

	"github.com/labstack/echo/v4"
)

// InboxHandler handles incoming ActivityPub activities
type InboxHandler struct {
	userRepo *repository.UserRepository
	msgRepo  *repository.MessageRepository
	cfg      *config.Config
}

// NewInboxHandler creates a new InboxHandler
func NewInboxHandler(userRepo *repository.UserRepository, msgRepo *repository.MessageRepository, cfg *config.Config) *InboxHandler {
	return &InboxHandler{
		userRepo: userRepo,
		msgRepo:  msgRepo,
		cfg:      cfg,
	}
}

// Handle processes incoming ActivityPub activities
// POST /users/:username/inbox
func (h *InboxHandler) Handle(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse the activity
	var activity map[string]interface{}
	if err := json.NewDecoder(c.Request().Body).Decode(&activity); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid activity JSON",
		})
	}

	activityType, _ := activity["type"].(string)
	actorURI, _ := activity["actor"].(string)
	activityID, _ := activity["id"].(string)

	log.Printf("[Inbox] Received %s from %s (id: %s)", activityType, actorURI, activityID)

	// Check domain blocking
	actorDomain := extractDomainFromURI(actorURI)
	if federation.IsDomainBlocked(ctx, actorDomain) {
		log.Printf("[Inbox] Blocked domain: %s", actorDomain)
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "domain blocked",
		})
	}

	// Deduplication
	if activityID != "" && federation.IsActivityProcessed(ctx, activityID) {
		log.Printf("[Inbox] Duplicate activity: %s", activityID)
		return c.JSON(http.StatusOK, map[string]string{
			"status": "already processed",
		})
	}

	// Store in inbox_activities
	payload, _ := json.Marshal(activity)
	_, err := db.GetDB().Exec(ctx,
		`INSERT INTO inbox_activities (activity_id, actor_uri, activity_type, payload)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (activity_id) DO NOTHING`,
		activityID, actorURI, activityType, payload,
	)
	if err != nil {
		log.Printf("[Inbox] Warning: failed to store activity: %v", err)
	}

	// Process by type
	switch activityType {
	case "Follow":
		return h.handleFollow(c, activity)
	case "Accept":
		return h.handleAccept(c, activity)
	case "Create":
		return h.handleCreate(c, activity)
	case "Like":
		return h.handleLike(c, activity)
	case "Announce":
		return h.handleAnnounce(c, activity)
	case "Update":
		return h.handleUpdate(c, activity)
	case "Delete":
		return h.handleDelete(c, activity)
	case "Undo":
		return h.handleUndo(c, activity)
	default:
		log.Printf("[Inbox] Unhandled activity type: %s", activityType)
		return c.JSON(http.StatusOK, map[string]string{
			"status": "accepted",
		})
	}
}

// handleAnnounce processes incoming Announce (repost/boost) activities
func (h *InboxHandler) handleAnnounce(c echo.Context, activity map[string]interface{}) error {
	ctx := c.Request().Context()
	actorURI, _ := activity["actor"].(string)
	objectURI, _ := activity["object"].(string)

	if strings.TrimSpace(objectURI) == "" {
		return c.JSON(http.StatusOK, map[string]string{"status": "ignored"})
	}

	_, err := db.GetDB().Exec(ctx,
		`INSERT INTO interactions (post_id, actor_did, interaction_type)
		 SELECT id, $1, 'repost' FROM posts WHERE original_post_uri = $2 OR id::text = $3
		 ON CONFLICT DO NOTHING`,
		actorURI, objectURI, extractPostIDFromURI(objectURI),
	)
	if err != nil {
		log.Printf("[Inbox] Failed to process announce: %v", err)
	}

	if id, _ := activity["id"].(string); id != "" {
		federation.MarkActivityProcessed(ctx, id)
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "announced"})
}

// handleUpdate processes incoming Update activities for actor metadata
func (h *InboxHandler) handleUpdate(c echo.Context, activity map[string]interface{}) error {
	ctx := c.Request().Context()
	actorURI, _ := activity["actor"].(string)
	obj, ok := activity["object"].(map[string]interface{})
	if !ok {
		return c.JSON(http.StatusOK, map[string]string{"status": "ignored"})
	}

	objType, _ := obj["type"].(string)
	if objType != "Person" {
		if id, _ := activity["id"].(string); id != "" {
			federation.MarkActivityProcessed(ctx, id)
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ignored"})
	}

	name, _ := obj["name"].(string)
	summary, _ := obj["summary"].(string)
	encryptionKey, _ := obj["encryption_public_key"].(string)
	avatarURL := ""
	if icon, ok := obj["icon"].(map[string]interface{}); ok {
		avatarURL, _ = icon["url"].(string)
	}
	publicKeyPEM := ""
	if pkObj, ok := obj["publicKey"].(map[string]interface{}); ok {
		publicKeyPEM, _ = pkObj["publicKeyPem"].(string)
	}

	if _, err := federation.EnsureRemoteUser(ctx, actorURI); err != nil {
		log.Printf("[Inbox] Failed to ensure remote user during update: %v", err)
	}

	_, _ = db.GetDB().Exec(ctx,
		`UPDATE users
		 SET display_name = COALESCE(NULLIF($1, ''), display_name),
		     bio = COALESCE(NULLIF($2, ''), bio),
		     avatar_url = COALESCE(NULLIF($3, ''), avatar_url),
		     public_key = COALESCE(NULLIF($4, ''), public_key),
		     encryption_public_key = COALESCE(NULLIF($5, ''), encryption_public_key),
		     updated_at = NOW()
		 WHERE did = $6`,
		name, summary, avatarURL, publicKeyPEM, encryptionKey, actorURI,
	)

	_, _ = db.GetDB().Exec(ctx,
		`UPDATE remote_actors
		 SET display_name = COALESCE(NULLIF($1, ''), display_name),
		     avatar_url = COALESCE(NULLIF($2, ''), avatar_url),
		     public_key_pem = COALESCE(NULLIF($3, ''), public_key_pem),
		     last_fetched_at = NOW()
		 WHERE actor_uri = $4`,
		name, avatarURL, publicKeyPEM, actorURI,
	)

	if id, _ := activity["id"].(string); id != "" {
		federation.MarkActivityProcessed(ctx, id)
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

// handleFollow processes incoming Follow requests
func (h *InboxHandler) handleFollow(c echo.Context, activity map[string]interface{}) error {
	ctx := c.Request().Context()
	actorURI, _ := activity["actor"].(string)
	objectURI, _ := activity["object"].(string)

	log.Printf("[Inbox] Follow request: %s wants to follow %s", actorURI, objectURI)

	// Resolve the remote actor
	// Extract username@domain from actor URI
	remoteActor, err := resolveActorFromURI(actorURI)
	if err != nil {
		log.Printf("[Inbox] Failed to resolve actor %s: %v", actorURI, err)
	}

	// Extract local username from object URI
	localUsername := extractUsernameFromURI(objectURI)
	if localUsername == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid follow target"})
	}

	// Look up local user
	localUser, _, err := h.userRepo.GetByUsername(ctx, localUsername)
	if err != nil || localUser == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "user not found"})
	}

	// Store follow relationship
	// Use remote actor URI as follower_did for remote users
	followerDID := actorURI
	if remoteActor != nil {
		followerDID = fmt.Sprintf("did:web:%s:%s", remoteActor.Domain, remoteActor.Username)
	}

	_, err = db.GetDB().Exec(ctx,
		`INSERT INTO follows (follower_did, following_did, status)
		 VALUES ($1, $2, 'accepted')
		 ON CONFLICT DO NOTHING`,
		followerDID, localUser.DID,
	)
	if err != nil {
		log.Printf("[Inbox] Failed to create follow: %v", err)
	}

	// Auto-accept: send Accept back
	acceptActivity := &federation.Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		ID:      fmt.Sprintf("%s/activities/accept-%d", h.cfg.Federation.URL, time.Now().UnixNano()),
		Type:    "Accept",
		Actor:   objectURI,
		Object:  activity,
	}

	if remoteActor != nil {
		go func() {
			if err := federation.DeliverActivity(acceptActivity, remoteActor.InboxURL); err != nil {
				log.Printf("[Inbox] Failed to send Accept: %v", err)
			}
		}()
	}

	// Mark as processed
	if id, _ := activity["id"].(string); id != "" {
		federation.MarkActivityProcessed(ctx, id)
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "accepted"})
}

// handleAccept processes Accept responses to our Follow requests
func (h *InboxHandler) handleAccept(c echo.Context, activity map[string]interface{}) error {
	ctx := c.Request().Context()
	log.Printf("[Inbox] Follow accepted")

	// Mark the original follow as accepted
	if object, ok := activity["object"].(map[string]interface{}); ok {
		if followActor, ok := object["actor"].(string); ok {
			if followTarget, ok := object["object"].(string); ok {
				_, err := db.GetDB().Exec(ctx,
					`UPDATE follows SET status = 'accepted'
					 WHERE follower_did LIKE '%' || $1 || '%' AND following_did LIKE '%' || $2 || '%'`,
					extractUsernameFromURI(followActor),
					extractUsernameFromURI(followTarget),
				)
				if err != nil {
					log.Printf("[Inbox] Failed to update follow status: %v", err)
				}
			}
		}
	}

	if id, _ := activity["id"].(string); id != "" {
		federation.MarkActivityProcessed(ctx, id)
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "accepted"})
}

// handleCreate processes incoming Create activities (new posts)
func (h *InboxHandler) handleCreate(c echo.Context, activity map[string]interface{}) error {
	ctx := c.Request().Context()
	actorURI, _ := activity["actor"].(string)

	// Parse the Note object
	object, ok := activity["object"].(map[string]interface{})
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing object"})
	}

	objType, _ := object["type"].(string)
	if objType != "Note" {
		log.Printf("[Inbox] Ignoring Create of type %s", objType)
		return c.JSON(http.StatusOK, map[string]string{"status": "ignored"})
	}

	content, _ := object["content"].(string)
	noteID, _ := object["id"].(string)
	published, _ := object["published"].(string)
	inReplyTo, _ := object["inReplyTo"].(string)

	// Check TO/CC to distinguish Public Post vs DM
	to, _ := activity["to"].([]interface{})
	isPublic := false
	var targetLocalUser *models.User

	for _, t := range to {
		recipient, ok := t.(string)
		if !ok {
			continue
		}
		if recipient == "https://www.w3.org/ns/activitystreams#Public" || recipient == "as:Public" {
			isPublic = true
			break
		}
		// Check if recipient is local user
		if domain := extractDomainFromURI(recipient); domain == h.cfg.Federation.Domain || domain == "localhost" {
			username := extractUsernameFromURI(recipient)
			if username != "" {
				// Verify local user exists (support localhost/legacy local domain values)
				var u models.User
				err := db.GetDB().QueryRow(ctx,
					"SELECT id, username FROM users WHERE username = $1 AND (instance_domain = $2 OR instance_domain = 'localhost' OR COALESCE(instance_domain,'') = '') LIMIT 1",
					username, h.cfg.Federation.Domain,
				).Scan(&u.ID, &u.Username)
				if err == nil {
					targetLocalUser = &u
				}
			}
		}
	}

	publishedTime := time.Now()
	if published != "" {
		if t, err := time.Parse(time.RFC3339, published); err == nil {
			publishedTime = t
		}
	}

	if !isPublic && targetLocalUser != nil {
		// THIS IS A DM
		log.Printf("[Inbox] Handling DM from %s to %s", actorURI, targetLocalUser.Username)

		// 1. Ensure Sender exists in local DB (Ghost User)
		senderUser, err := federation.EnsureRemoteUser(ctx, actorURI)
		if err != nil {
			log.Printf("[Inbox] Failed to ensure remote user %s: %v", actorURI, err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to process sender"})
		}

		// 2. Get/Create Thread
		thread, err := h.msgRepo.GetOrCreateThread(ctx, senderUser.ID, targetLocalUser.ID)
		if err != nil {
			log.Printf("[Inbox] Failed to get thread: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get thread"})
		}

		// 3. Insert Message
		// Store content as plaintext.
		_, err = h.msgRepo.SendMessage(ctx, thread.ID, senderUser.ID, targetLocalUser.ID, content, "")
		if err != nil {
			log.Printf("[Inbox] Failed to save message: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save message"})
		}

		log.Printf("[Inbox] DM saved successfully")
		return c.JSON(http.StatusOK, map[string]string{"status": "created"})
	}

	// PUBLIC POST Handling
	log.Printf("[Inbox] Remote post from %s: %s", actorURI, truncate(content, 50))

	// Store as remote post
	log.Printf("[Inbox] DEBUG: Inserting remote post values: author_did=%s, content=%s, original_post_uri=%s, published=%v", actorURI, content, noteID, publishedTime)

	_, err := db.GetDB().Exec(ctx,
		`INSERT INTO posts (author_did, content, visibility, is_remote, original_post_uri, in_reply_to_uri, created_at)
		 VALUES ($1, $2, 'public', true, $3, NULLIF($4, ''), $5)
		 ON CONFLICT DO NOTHING`,
		actorURI, content, noteID, inReplyTo, publishedTime,
	)
	if err != nil {
		log.Printf("[Inbox] Failed to store remote post: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to store post",
		})
	}
	log.Printf("[Inbox] DEBUG: Successfully inserted remote post")

	// Mark as processed
	if id, _ := activity["id"].(string); id != "" {
		federation.MarkActivityProcessed(ctx, id)
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "created"})
}

// handleLike processes incoming Like activities
func (h *InboxHandler) handleLike(c echo.Context, activity map[string]interface{}) error {
	ctx := c.Request().Context()
	actorURI, _ := activity["actor"].(string)
	objectURI, _ := activity["object"].(string)

	log.Printf("[Inbox] Like from %s on %s", actorURI, objectURI)

	// Record the like as an interaction
	postID := extractPostIDFromURI(objectURI)
	if postID != "" {
		_, err := db.GetDB().Exec(ctx,
			`INSERT INTO interactions (post_id, actor_did, interaction_type)
			 SELECT id, $1, 'like' FROM posts WHERE id::text = $2 OR original_post_uri = $3
			 ON CONFLICT DO NOTHING`,
			actorURI, postID, objectURI,
		)
		if err != nil {
			log.Printf("[Inbox] Failed to process like: %v", err)
		}
	}

	if id, _ := activity["id"].(string); id != "" {
		federation.MarkActivityProcessed(ctx, id)
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "liked"})
}

// handleDelete processes incoming Delete activities
func (h *InboxHandler) handleDelete(c echo.Context, activity map[string]interface{}) error {
	ctx := c.Request().Context()
	actorURI, _ := activity["actor"].(string)
	objectURI := ""

	if obj, ok := activity["object"].(string); ok {
		objectURI = obj
	} else if obj, ok := activity["object"].(map[string]interface{}); ok {
		objectURI, _ = obj["id"].(string)
	}

	if objectURI != "" {
		isActorDelete := strings.Contains(objectURI, "/ap/users/") && !strings.Contains(objectURI, "/posts/")

		if isActorDelete {
			if _, err := db.GetDB().Exec(ctx, `UPDATE posts SET deleted_at = now() WHERE author_did = $1`, objectURI); err != nil {
				log.Printf("[Inbox] Failed to delete remote actor posts: %v", err)
			}
			if _, err := db.GetDB().Exec(ctx, `DELETE FROM interactions WHERE actor_did = $1`, objectURI); err != nil {
				log.Printf("[Inbox] Failed to delete remote actor interactions: %v", err)
			}
			if _, err := db.GetDB().Exec(ctx, `DELETE FROM follows WHERE follower_did = $1 OR following_did = $1`, objectURI); err != nil {
				log.Printf("[Inbox] Failed to delete remote actor follows: %v", err)
			}
			if _, err := db.GetDB().Exec(ctx, `DELETE FROM users WHERE did = $1`, objectURI); err != nil {
				log.Printf("[Inbox] Failed to delete remote actor user: %v", err)
			}
			if _, err := db.GetDB().Exec(ctx, `DELETE FROM remote_actors WHERE actor_uri = $1`, objectURI); err != nil {
				log.Printf("[Inbox] Failed to delete remote actor cache: %v", err)
			}
		} else {
			_, err := db.GetDB().Exec(ctx,
				`UPDATE posts SET deleted_at = now() WHERE original_post_uri = $1 OR (author_did = $2 AND id::text = $3)`,
				objectURI, actorURI, extractPostIDFromURI(objectURI),
			)
			if err != nil {
				log.Printf("[Inbox] Failed to delete remote post: %v", err)
			}
		}
	}

	if id, _ := activity["id"].(string); id != "" {
		federation.MarkActivityProcessed(ctx, id)
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// handleUndo processes Undo activities (unfollow, unlike)
func (h *InboxHandler) handleUndo(c echo.Context, activity map[string]interface{}) error {
	ctx := c.Request().Context()

	object, ok := activity["object"].(map[string]interface{})
	if !ok {
		return c.JSON(http.StatusOK, map[string]string{"status": "accepted"})
	}

	undoType, _ := object["type"].(string)
	switch undoType {
	case "Follow":
		actorURI, _ := object["actor"].(string)
		targetURI, _ := object["object"].(string)
		_, err := db.GetDB().Exec(ctx,
			`DELETE FROM follows WHERE follower_did LIKE '%' || $1 || '%' AND following_did LIKE '%' || $2 || '%'`,
			extractUsernameFromURI(actorURI),
			extractUsernameFromURI(targetURI),
		)
		if err != nil {
			log.Printf("[Inbox] Failed to undo follow: %v", err)
		}
	case "Like":
		objectURI, _ := object["object"].(string)
		actorURI, _ := object["actor"].(string)
		postID := extractPostIDFromURI(objectURI)
		if postID != "" {
			_, err := db.GetDB().Exec(ctx,
				`DELETE FROM interactions WHERE actor_did = $1 AND interaction_type = 'like'
				 AND post_id IN (SELECT id FROM posts WHERE id::text = $2 OR original_post_uri = $3)`,
				actorURI, postID, objectURI)
			if err != nil {
				log.Printf("[Inbox] Failed to undo like: %v", err)
			}
		}
	}

	if id, _ := activity["id"].(string); id != "" {
		federation.MarkActivityProcessed(ctx, id)
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "undone"})
}

// Helper functions

func extractDomainFromURI(uri string) string {
	// Map known local instance base URLs first
	for domain, baseURL := range federation.InstanceURLMap {
		if len(uri) > len(baseURL) && uri[:len(baseURL)] == baseURL {
			return domain
		}
	}

	parsed, err := url.Parse(strings.TrimSpace(uri))
	if err != nil {
		return ""
	}

	host := parsed.Hostname()
	if host == "" {
		return ""
	}

	if strings.Contains(host, "localhost") || strings.Contains(host, "127.0.0.1") {
		if strings.Contains(parsed.Host, ":8000") {
			return "splitter-1"
		}
		if strings.Contains(parsed.Host, ":8001") {
			return "splitter-2"
		}
	}

	return host
}

func extractUsernameFromURI(uri string) string {
	// Extract username from http://localhost:8000/users/alice → alice
	parts := splitURI(uri)
	for i, p := range parts {
		if p == "users" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func extractPostIDFromURI(uri string) string {
	parts := splitURI(uri)
	for i, p := range parts {
		if p == "posts" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func splitURI(uri string) []string {
	// Remove protocol
	idx := 0
	if i := len("http://"); len(uri) > i && uri[:i] == "http://" {
		idx = i
	} else if i := len("https://"); len(uri) > i && uri[:i] == "https://" {
		idx = i
	}
	path := uri[idx:]
	// Split by /
	var parts []string
	for _, p := range splitPath(path) {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func splitPath(path string) []string {
	result := []string{}
	current := ""
	for _, ch := range path {
		if ch == '/' {
			if current != "" {
				result = append(result, current)
			}
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func resolveActorFromURI(actorURI string) (*federation.RemoteActor, error) {
	username := extractUsernameFromURI(actorURI)
	domain := extractDomainFromURI(actorURI)
	if username == "" || domain == "" {
		return nil, fmt.Errorf("could not parse actor URI: %s", actorURI)
	}
	return federation.ResolveRemoteUser(fmt.Sprintf("%s@%s", username, domain))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
