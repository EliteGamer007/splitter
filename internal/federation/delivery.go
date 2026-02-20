package federation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"splitter/internal/db"
)

// Activity represents an ActivityPub activity
type Activity struct {
	Context interface{} `json:"@context"`
	ID      string      `json:"id"`
	Type    string      `json:"type"`
	Actor   string      `json:"actor"`
	Object  interface{} `json:"object"`
	To      []string    `json:"to,omitempty"`
	CC      []string    `json:"cc,omitempty"`
}

// Note represents an ActivityPub Note (post)
type Note struct {
	Context      interface{} `json:"@context,omitempty"`
	ID           string      `json:"id"`
	Type         string      `json:"type"`
	AttributedTo string      `json:"attributedTo"`
	Content      string      `json:"content"`
	Published    string      `json:"published"`
	To           []string    `json:"to,omitempty"`
	CC           []string    `json:"cc,omitempty"`
}

// DeliverActivity sends an activity to a remote inbox with HTTP Signature
func DeliverActivity(activity *Activity, targetInbox string) error {
	ctx := context.Background()

	if targetDomain := extractDomainFromURI(targetInbox); targetDomain != "" && IsDomainBlocked(ctx, targetDomain) {
		return fmt.Errorf("target domain %s is blocked", targetDomain)
	}

	// Serialize activity
	body, err := json.Marshal(activity)
	if err != nil {
		return fmt.Errorf("failed to marshal activity: %w", err)
	}

	// Store in outbox
	outboxID, err := storeOutboxActivity(ctx, activity.Type, body, targetInbox)
	if err != nil {
		log.Printf("[Federation] Warning: failed to store outbox activity: %v", err)
	}

	// Create request
	req, err := http.NewRequest("POST", targetInbox, bytes.NewReader(body))
	if err != nil {
		updateOutboxStatus(ctx, outboxID, "failed")
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/activity+json")
	req.Header.Set("Accept", "application/activity+json")

	// Sign request
	privKey := GetInstancePrivateKey()
	domain := GetInstanceDomain()
	if privKey != nil {
		keyID := fmt.Sprintf("%s/ap/users/%s#main-key", resolveInstanceURL(domain), "admin")
		if err := SignRequest(req, privKey, keyID); err != nil {
			log.Printf("[Federation] Warning: failed to sign request: %v", err)
		}
	}

	// Send
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		updateOutboxStatus(ctx, outboxID, "failed")
		return fmt.Errorf("delivery failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		updateOutboxStatus(ctx, outboxID, "sent")
		log.Printf("[Federation] Delivered %s to %s (status: %d)", activity.Type, targetInbox, resp.StatusCode)
		return nil
	}

	updateOutboxStatus(ctx, outboxID, "failed")
	return fmt.Errorf("delivery returned status %d", resp.StatusCode)
}

// DeliverToActor resolves a remote actor and delivers an activity to their inbox
func DeliverToActor(activity *Activity, actorURI string) error {
	if actorURI == "" {
		return fmt.Errorf("actor URI is required")
	}

	actor, err := resolveActorFromURI(actorURI)
	if err != nil {
		return fmt.Errorf("failed to resolve actor %s: %w", actorURI, err)
	}

	if actor == nil || actor.InboxURL == "" {
		return fmt.Errorf("actor inbox not found for %s", actorURI)
	}

	return DeliverActivity(activity, actor.InboxURL)
}

// DeliverToFollowers delivers an activity to all remote followers of a user
// For public posts, also delivers to all known instance shared inboxes
func DeliverToFollowers(activity *Activity, authorDID string) {
	ctx := context.Background()
	domain := GetInstanceDomain()
	log.Printf("[Federation] DeliverToFollowers: authorDID=%s, domain=%s", authorDID, domain)

	var inboxes []string

	// Get remote followers (followers from other instances)
	rows, err := db.GetDB().Query(ctx,
		`SELECT DISTINCT ra.inbox_url
		 FROM follows f
		 JOIN remote_actors ra ON ra.actor_uri = (
			SELECT actor_uri FROM remote_actors WHERE username || '@' || domain = 
			  SUBSTRING(f.follower_did FROM 'did:key:(.+)')
		 )
		 WHERE f.following_did = $1 AND f.status = 'accepted'`, authorDID)

	if err != nil {
		log.Printf("[Federation] Complex follower query failed: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var inbox string
			if err := rows.Scan(&inbox); err == nil {
				inboxes = append(inboxes, inbox)
			}
		}
	}

	// For public posts (addressed to Public), also deliver to all known instance shared inboxes
	isPublic := false
	if activity.To != nil {
		for _, to := range activity.To {
			if to == "https://www.w3.org/ns/activitystreams#Public" {
				isPublic = true
				break
			}
		}
	}

	if isPublic {
		for otherDomain, otherURL := range InstanceURLMap {
			if otherDomain != domain {
				sharedInbox := otherURL + "/ap/shared-inbox"
				log.Printf("[Federation] Adding shared inbox for %s: %s", otherDomain, sharedInbox)
				inboxes = append(inboxes, sharedInbox)
			}
		}
	}

	// Also get inboxes from remote_actors table
	actorRows, err := db.GetDB().Query(ctx,
		`SELECT DISTINCT inbox_url FROM remote_actors WHERE domain != $1 AND domain != '' AND inbox_url != ''`, domain)
	if err == nil {
		defer actorRows.Close()
		for actorRows.Next() {
			var inbox string
			if err := actorRows.Scan(&inbox); err == nil {
				inboxes = append(inboxes, inbox)
			}
		}
	}

	// Deduplicate and deliver
	seen := make(map[string]bool)
	deliveryCount := 0
	for _, inbox := range inboxes {
		if seen[inbox] {
			continue
		}
		seen[inbox] = true
		deliveryCount++
		go func(inboxURL string) {
			if err := DeliverActivity(activity, inboxURL); err != nil {
				log.Printf("[Federation] Failed to deliver to %s: %v", inboxURL, err)
			}
		}(inbox)
	}

	log.Printf("[Federation] Delivering to %d unique inboxes", deliveryCount)
}

// deliverToAllKnownInboxes delivers to all unique remote inboxes
// Simpler approach for local testing with 2 instances
func deliverToAllKnownInboxes(activity *Activity, authorDID string) {
	ctx := context.Background()
	domain := GetInstanceDomain()
	log.Printf("[Federation] deliverToAllKnownInboxes: domain=%s, authorDID=%s", domain, authorDID)

	// Get all remote actors from OTHER instances
	rows, err := db.GetDB().Query(ctx,
		`SELECT DISTINCT inbox_url FROM remote_actors WHERE domain != $1 AND domain != ''`, domain)
	if err != nil {
		log.Printf("[Federation] Failed to get remote inboxes: %v", err)
		return
	}
	defer rows.Close()

	var inboxes []string
	for rows.Next() {
		var inbox string
		if err := rows.Scan(&inbox); err == nil {
			inboxes = append(inboxes, inbox)
		}
	}

	// Also try delivering to the other instance's shared inbox
	for otherDomain, otherURL := range InstanceURLMap {
		if otherDomain != domain {
			sharedInbox := otherURL + "/ap/shared-inbox"
			log.Printf("[Federation] Adding shared inbox for %s: %s", otherDomain, sharedInbox)
			inboxes = append(inboxes, sharedInbox)
		}
	}

	// Deduplicate
	log.Printf("[Federation] Total inboxes to deliver to: %d", len(inboxes))
	seen := make(map[string]bool)
	for _, inbox := range inboxes {
		if seen[inbox] {
			continue
		}
		seen[inbox] = true
		go func(inboxURL string) {
			if err := DeliverActivity(activity, inboxURL); err != nil {
				log.Printf("[Federation] Failed to deliver to %s: %v", inboxURL, err)
			}
		}(inbox)
	}
}

// SendFollow sends a Follow activity to a remote actor
func SendFollow(localActorURI string, remoteActor *RemoteActor) error {
	activity := &Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		ID:      fmt.Sprintf("%s/activities/follow-%d", localActorURI, time.Now().UnixNano()),
		Type:    "Follow",
		Actor:   localActorURI,
		Object:  remoteActor.ActorURI,
	}

	return DeliverActivity(activity, remoteActor.InboxURL)
}

// BuildCreateNoteActivity creates a Create activity wrapping a Note
func BuildCreateNoteActivity(actorURI, postID, content string, createdAt time.Time) *Activity {
	domain := GetInstanceDomain()
	baseURL := resolveInstanceURL(domain)

	note := Note{
		ID:           fmt.Sprintf("%s/posts/%s", baseURL, postID),
		Type:         "Note",
		AttributedTo: actorURI,
		Content:      content,
		Published:    createdAt.UTC().Format(time.RFC3339),
		To:           []string{"https://www.w3.org/ns/activitystreams#Public"},
	}

	return &Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		ID:      fmt.Sprintf("%s/activities/create-%s", baseURL, postID),
		Type:    "Create",
		Actor:   actorURI,
		Object:  note,
		To:      []string{"https://www.w3.org/ns/activitystreams#Public"},
	}
}

func BuildLikeActivity(actorURI, objectURI string) *Activity {
	domain := GetInstanceDomain()
	baseURL := resolveInstanceURL(domain)
	activityID := fmt.Sprintf("%s/activities/like-%d", baseURL, time.Now().UnixNano())

	return &Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		ID:      activityID,
		Type:    "Like",
		Actor:   actorURI,
		Object:  objectURI,
	}
}

func BuildAnnounceActivity(actorURI, objectURI string) *Activity {
	domain := GetInstanceDomain()
	baseURL := resolveInstanceURL(domain)
	activityID := fmt.Sprintf("%s/activities/announce-%d", baseURL, time.Now().UnixNano())

	return &Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		ID:      activityID,
		Type:    "Announce",
		Actor:   actorURI,
		Object:  objectURI,
		To:      []string{"https://www.w3.org/ns/activitystreams#Public"},
	}
}

func BuildDeleteActivity(actorURI, objectURI string) *Activity {
	domain := GetInstanceDomain()
	baseURL := resolveInstanceURL(domain)
	activityID := fmt.Sprintf("%s/activities/delete-%d", baseURL, time.Now().UnixNano())

	return &Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		ID:      activityID,
		Type:    "Delete",
		Actor:   actorURI,
		Object:  objectURI,
		To:      []string{"https://www.w3.org/ns/activitystreams#Public"},
	}
}

func BuildUpdateActorActivity(actorURI, username, displayName, summary, publicKeyPEM, encryptionPublicKey string) *Activity {
	domain := GetInstanceDomain()
	baseURL := resolveInstanceURL(domain)
	activityID := fmt.Sprintf("%s/activities/update-%d", baseURL, time.Now().UnixNano())

	object := map[string]interface{}{
		"id":                actorURI,
		"type":              "Person",
		"preferredUsername": username,
		"name":              displayName,
		"summary":           summary,
		"inbox":             fmt.Sprintf("%s/ap/users/%s/inbox", baseURL, username),
		"outbox":            fmt.Sprintf("%s/ap/users/%s/outbox", baseURL, username),
		"publicKey": map[string]interface{}{
			"id":           actorURI + "#main-key",
			"owner":        actorURI,
			"publicKeyPem": publicKeyPEM,
		},
	}

	if strings.TrimSpace(encryptionPublicKey) != "" {
		object["encryption_public_key"] = encryptionPublicKey
	}

	return &Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		ID:      activityID,
		Type:    "Update",
		Actor:   actorURI,
		Object:  object,
		To:      []string{"https://www.w3.org/ns/activitystreams#Public"},
	}
}

// storeOutboxActivity stores an activity in the outbox table
func storeOutboxActivity(ctx context.Context, activityType string, payload []byte, targetInbox string) (string, error) {
	var id string
	err := db.GetDB().QueryRow(ctx,
		`INSERT INTO outbox_activities (activity_type, payload, target_inbox, status)
		 VALUES ($1, $2, $3, 'pending') RETURNING id::text`,
		activityType, payload, targetInbox,
	).Scan(&id)
	return id, err
}

// updateOutboxStatus updates the delivery status of an outbox activity
func updateOutboxStatus(ctx context.Context, id string, status string) {
	if id == "" {
		return
	}
	db.GetDB().Exec(ctx,
		`UPDATE outbox_activities SET status = $1, retry_count = retry_count + 1 WHERE id = $2`,
		status, id)
}

// IsDomainBlocked checks if a domain is blocked for federation
func IsDomainBlocked(ctx context.Context, domain string) bool {
	var exists bool
	err := db.GetDB().QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM blocked_domains WHERE domain = $1)`, domain,
	).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

// IsActivityProcessed checks if an activity has already been processed (deduplication)
func IsActivityProcessed(ctx context.Context, activityID string) bool {
	var exists bool
	err := db.GetDB().QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM activity_deduplication WHERE activity_id = $1)`, activityID,
	).Scan(&exists)
	if err != nil {
		return false
	}
	return exists
}

// MarkActivityProcessed records an activity as processed
func MarkActivityProcessed(ctx context.Context, activityID string) error {
	_, err := db.GetDB().Exec(ctx,
		`INSERT INTO activity_deduplication (activity_id, expires_at)
		 VALUES ($1, now() + interval '7 days')
		 ON CONFLICT DO NOTHING`, activityID)
	return err
}
