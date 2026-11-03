package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	"splitter/internal/security"

	"github.com/jackc/pgx/v5"
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
	if h.cfg == nil || !h.cfg.Federation.Enabled {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Federation is not enabled on this instance"})
	}
	guard := security.GetMessagingGuard()

	const maxInboxPayloadBytes int64 = 256 * 1024
	if c.Request().ContentLength > maxInboxPayloadBytes {
		guard.RecordInboxRejected("unknown", "payload too large", map[string]interface{}{
			"content_length": c.Request().ContentLength,
			"max":            maxInboxPayloadBytes,
		})
		return c.JSON(http.StatusRequestEntityTooLarge, map[string]string{
			"error": "payload too large",
		})
	}

	// Bound and preserve exactly the bytes authenticated by Digest.
	body, err := io.ReadAll(io.LimitReader(c.Request().Body, maxInboxPayloadBytes+1))
	if err != nil || int64(len(body)) > maxInboxPayloadBytes {
		return c.JSON(http.StatusRequestEntityTooLarge, map[string]string{"error": "payload too large"})
	}
	c.Request().Body = io.NopCloser(bytes.NewReader(body))
	var activity map[string]interface{}
	if err = json.Unmarshal(body, &activity); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid activity JSON"})
	}

	activityType, _ := activity["type"].(string)
	actorURI, _ := activity["actor"].(string)
	activityID, _ := activity["id"].(string)

	if strings.TrimSpace(actorURI) == "" {
		guard.RecordInboxRejected("unknown", "missing actor URI", map[string]interface{}{"activity_type": activityType})
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "missing actor URI",
		})
	}

	remoteActor, err := resolveActorFromURI(actorURI)
	if err != nil || remoteActor == nil {
		guard.RecordInboxRejected(actorURI, "unknown actor", map[string]interface{}{"error": fmt.Sprintf("%v", err)})
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "unknown actor",
		})
	}

	if allowed, reason := guard.AllowRemoteInbound(actorURI, remoteActor.Domain); !allowed {
		return c.JSON(http.StatusTooManyRequests, map[string]string{
			"error": reason,
		})
	}

	// A different actor's key must never authorize the claimed activity actor.
	sigKeyID := parseSignatureKeyID(c.Request().Header.Get("Signature"))
	if remoteActor.ActorURI != actorURI || stripFragment(sigKeyID) != actorURI || !strings.Contains(sigKeyID, "#") {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "signature key does not belong to activity actor"})
	}
	verificationKey := strings.TrimSpace(remoteActor.PublicKeyPEM)

	if verificationKey == "" {
		guard.RecordInboxRejected(actorURI, "missing verification key", map[string]interface{}{"domain": remoteActor.Domain})
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "missing remote signature key",
		})
	}

	if err := federation.VerifyRequest(c.Request(), verificationKey); err != nil {
		guard.RecordInboxRejected(actorURI, "invalid signature", map[string]interface{}{"error": err.Error()})
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "invalid signature",
		})
	}

	if strings.TrimSpace(activityType) == "Create" {
		if obj, ok := activity["object"].(map[string]interface{}); ok {
			if attributedTo, _ := obj["attributedTo"].(string); attributedTo != "" && attributedTo != actorURI {
				guard.RecordInboxRejected(actorURI, "actor DID mismatch", map[string]interface{}{"attributed_to": attributedTo})
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "actor mismatch",
				})
			}
		}
	}

	log.Printf("[Inbox] Received %s from %s (id: %s)", activityType, actorURI, activityID)

	// Check domain blocking
	actorDomain := extractDomainFromURI(actorURI)
	if federation.IsDomainBlocked(ctx, actorDomain) {
		log.Printf("[Inbox] Blocked domain: %s", actorDomain)
		guard.RecordInboxRejected(actorURI, "blocked domain", map[string]interface{}{"domain": actorDomain})
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
	_, err = db.GetDB().Exec(ctx,
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

func parseSignatureKeyID(signatureHeader string) string {
	if strings.TrimSpace(signatureHeader) == "" {
		return ""
	}
	parts := strings.Split(signatureHeader, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(part, "keyId=") {
			continue
		}
		value := strings.TrimPrefix(part, "keyId=")
		value = strings.Trim(value, `"`)
		return strings.TrimSpace(value)
	}
	return ""
}

func stripFragment(uri string) string {
	if idx := strings.Index(uri, "#"); idx >= 0 {
		return uri[:idx]
	}
	return uri
}

// handleAnnounce processes incoming Announce (repost/boost) activities
func (h *InboxHandler) handleAnnounce(c echo.Context, activity map[string]interface{}) error {
	ctx := c.Request().Context()
	actorURI, _ := activity["actor"].(string)
	objectURI, _ := activity["object"].(string)
	targetID := extractPostIDFromURI(objectURI)
	if targetID != "" {
		allowed, err := repository.NewPostRepository().CanViewTarget(ctx, targetID, actorURI)
		if err != nil || !allowed {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "post not accessible"})
		}
	}

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

	if objectURI != strings.TrimRight(h.cfg.Federation.URL, "/")+"/ap/users/"+localUser.Username {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid local follow target"})
	}
	if _, err = federation.EnsureRemoteUser(ctx, actorURI); err != nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Unable to resolve follower"})
	}
	// Store follow relationship
	// Use remote actor URI as follower_did for remote users
	followerDID := actorURI
	if remoteActor != nil {
		followerDID = remoteActor.ActorURI
	}

	_, err = db.GetDB().Exec(ctx,
		`INSERT INTO follows (follower_did, following_did, status)
		 VALUES ($1, $2, $3)
		 ON CONFLICT DO NOTHING`,
		followerDID, localUser.DID, func() string {
			if localUser.IsLocked {
				return "pending"
			}
			return "accepted"
		}(),
	)
	if err != nil {
		log.Printf("[Inbox] Failed to create follow: %v", err)
	}

	if localUser.IsLocked {
		if id, _ := activity["id"].(string); id != "" {
			federation.MarkActivityProcessed(ctx, id)
		}
		return c.JSON(http.StatusAccepted, map[string]string{"status": "pending"})
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
	signer, _ := activity["actor"].(string)
	object, ok := activity["object"].(map[string]interface{})
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing Follow object"})
	}
	follower, _ := object["actor"].(string)
	target, _ := object["object"].(string)
	if target != signer {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "accept actor mismatch"})
	}
	local, _, err := h.userRepo.GetByUsername(ctx, extractUsernameFromURI(follower))
	if err != nil || local == nil || follower != strings.TrimRight(h.cfg.Federation.URL, "/")+"/ap/users/"+local.Username {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "invalid local follow actor"})
	}
	_, err = db.GetDB().Exec(ctx, "UPDATE follows SET status='accepted' WHERE follower_did=$1 AND following_did=$2 AND status='pending'", local.DID, target)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to accept follow"})
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
	visibility, _ := object["splitter_visibility"].(string)
	ciphertext, _ := object["ciphertext"].(string)
	encryptedKeys := interfaceToStringMap(object["encrypted_keys"])
	noteID, _ := object["id"].(string)
	published, _ := object["published"].(string)
	inReplyTo, _ := object["inReplyTo"].(string)

	// Collect all possible recipients from ActivityPub fields
	recipients := []interface{}{}

	if v, ok := object["to"].([]interface{}); ok {
		recipients = append(recipients, v...)
	}
	if v, ok := object["cc"].([]interface{}); ok {
		recipients = append(recipients, v...)
	}

	if len(recipients) == 0 {
		if v, ok := activity["to"].([]interface{}); ok {
			recipients = append(recipients, v...)
		}
		if v, ok := activity["cc"].([]interface{}); ok {
			recipients = append(recipients, v...)
		}
	}

	isPublic := false
	var targetLocalUser *models.User
	var localRecipients []string

	for _, t := range recipients {
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
				// Resolve local user by username first (more robust than instance_domain matching),
				// while still preferring non-ghost local accounts when duplicates exist.
				var u models.User
				err := db.GetDB().QueryRow(ctx,
					`SELECT id, username, COALESCE(encryption_public_key, ''),COALESCE(did,'')
					 FROM users
					 WHERE username = $1
					 ORDER BY CASE
					   WHEN instance_domain = $2 OR instance_domain = 'localhost' OR COALESCE(instance_domain, '') = '' THEN 0
					   ELSE 1
					 END,
					 updated_at DESC
					 LIMIT 1`,
					username, h.cfg.Federation.Domain,
				).Scan(&u.ID, &u.Username, &u.EncryptionPublicKey, &u.DID)
				if err == nil {
					targetLocalUser = &u
					localRecipients = append(localRecipients, u.DID)
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

	if !isPublic && targetLocalUser != nil && visibility != "circle" && visibility != "followers" {
		// THIS IS A DM
		log.Printf("[Inbox] Handling DM from %s to %s", actorURI, targetLocalUser.Username)

		// 1. Ensure Sender exists in local DB (Ghost User)
		senderUser, err := federation.EnsureRemoteUser(ctx, actorURI)
		if err != nil {
			log.Printf("[Inbox] Failed to ensure remote user %s: %v", actorURI, err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to process sender"})
		}

		// 2. Get/Create Thread
		thread, err := h.msgRepo.GetOrCreateThreadForInbound(ctx, senderUser.ID, targetLocalUser.ID)
		if err != nil {
			log.Printf("[Inbox] Failed to get thread: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get thread"})
		}

		encryptedKeysJSON := ""
		if len(encryptedKeys) > 0 {
			if raw, marshalErr := json.Marshal(encryptedKeys); marshalErr == nil {
				encryptedKeysJSON = string(raw)
			}
		}

		// 3. Insert Message
		_, err = h.msgRepo.SendMessage(ctx, thread.ID, senderUser.ID, targetLocalUser.ID, content, ciphertext, encryptedKeysJSON)
		if err != nil {
			log.Printf("[Inbox] Failed to save message: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save message"})
		}

		if id, _ := activity["id"].(string); id != "" {
			federation.MarkActivityProcessed(ctx, id)
		}

		log.Printf("[Inbox] DM saved successfully")
		return c.JSON(http.StatusOK, map[string]string{"status": "created"})
	}

	if isPublic {
		visibility = "public"
	} else if (visibility != "circle" && visibility != "followers") || len(localRecipients) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Note must have an authorized audience"})
	}
	// Store explicit recipients and content together; never turn a restricted Note into a public feed item.
	tx, err := db.GetDB().Begin(ctx)
	if err != nil {
		return c.JSON(503, map[string]string{"error": "Database unavailable"})
	}
	defer tx.Rollback(ctx)
	// POST Handling
	log.Printf("[Inbox] Remote post received")

	// Store as remote post
	if len([]rune(content)) > 5000 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "remote content too large"})
	}

	var postID string
	err = tx.QueryRow(ctx,
		`INSERT INTO posts (author_did, content, visibility, is_remote, original_post_uri, in_reply_to_uri, created_at)
		 VALUES ($1, $2, $6, true, $3, NULLIF($4, ''), $5)
		 ON CONFLICT DO NOTHING
		 RETURNING id`,
		actorURI, content, noteID, inReplyTo, publishedTime, visibility,
	).Scan(&postID)
	if err != nil && err != pgx.ErrNoRows {
		log.Printf("[Inbox] Failed to store remote post: %v", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to store post",
		})
	}
	if postID != "" && visibility != "public" {
		for _, did := range localRecipients {
			if _, err = tx.Exec(ctx, "INSERT INTO remote_post_recipients(post_id,recipient_did) VALUES($1,$2) ON CONFLICT DO NOTHING", postID, did); err != nil {
				return c.JSON(503, map[string]string{"error": "Unable to persist audience"})
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return c.JSON(503, map[string]string{"error": "Unable to persist post"})
	}

	// Store media attachments from the Note
	if postID != "" {
		if attachments, ok := object["attachment"].([]interface{}); ok {
			for _, att := range attachments {
				attMap, ok := att.(map[string]interface{})
				if !ok {
					continue
				}
				mediaURL, _ := attMap["url"].(string)
				mediaType, _ := attMap["mediaType"].(string)
				if mediaURL == "" {
					continue
				}
				if mediaType == "" {
					mediaType = "image/jpeg"
				}
				_, mErr := db.GetDB().Exec(ctx,
					`INSERT INTO media (post_id, media_url, media_type) VALUES ($1, $2, $3)`,
					postID, mediaURL, mediaType,
				)
				if mErr != nil {
					log.Printf("[Inbox] Failed to store media attachment: %v", mErr)
				}
			}
		}
	}

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
	targetID := extractPostIDFromURI(objectURI)
	if targetID != "" {
		allowed, err := repository.NewPostRepository().CanViewTarget(ctx, targetID, actorURI)
		if err != nil || !allowed {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "post not accessible"})
		}
	}

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
			if objectURI != actorURI {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "actor delete owner mismatch"})
			}
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
				`UPDATE posts SET deleted_at = now() WHERE author_did = $2 AND is_remote = true AND (original_post_uri = $1 OR id::text = $3)`,
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

	claimedActor, _ := object["actor"].(string)
	signer, _ := activity["actor"].(string)
	if claimedActor != "" && claimedActor != signer {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "undo actor mismatch"})
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

	return parsed.Host
}

func extractUsernameFromURI(uri string) string {
	// Extract username from http://localhost:8000/users/alice Ã¢â€ â€™ alice
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

func interfaceToStringMap(value interface{}) map[string]string {
	if value == nil {
		return nil
	}
	input, ok := value.(map[string]interface{})
	if !ok {
		return nil
	}
	result := make(map[string]string, len(input))
	for key, raw := range input {
		text, castOK := raw.(string)
		if !castOK {
			continue
		}
		result[key] = text
	}
	return result
}
