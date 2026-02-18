package federation

import (
	"context"
	"fmt"
	"log"
	"time"

	"splitter/internal/db"
	"splitter/internal/models"
)

// EnsureRemoteUser ensures a remote actor exists in the local users table
// This is required for foreign key constraints in the messages table
func EnsureRemoteUser(ctx context.Context, actorURI string) (*models.User, error) {
	// 1. Check if user already exists in users table by DID
	// Remote users should have DID set to their actor URI (or a did:web)
	// For now, we use actorURI as DID for remote users if they don't have a did:key
	var user models.User
	err := db.GetDB().QueryRow(ctx,
		`SELECT id, username, instance_domain, did, public_key, encryption_public_key 
		 FROM users WHERE did = $1 OR (did IS NULL AND username = $2 AND instance_domain = $3)`,
		actorURI, extractUsernameFromURI(actorURI), extractDomainFromURI(actorURI)).Scan(
		&user.ID, &user.Username, &user.InstanceDomain, &user.DID, &user.PublicKey, &user.EncryptionPublicKey,
	)

	if err == nil {
		return &user, nil
	}

	// 2. User not found, need to create "ghost" user
	// Resolve remote actor details first
	remoteActor, err := resolveActorFromURI(actorURI)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve remote actor %s: %w", actorURI, err)
	}

	// 3. Insert into users table
	// Password hash is empty/null which effectively disables password login
	// Role is 'user'
	log.Printf("[Federation] Creating ghost user for %s@%s", remoteActor.Username, remoteActor.Domain)

	err = db.GetDB().QueryRow(ctx,
		`INSERT INTO users (username, instance_domain, did, display_name, avatar_url, public_key, encryption_public_key, role)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, 'user')
		 RETURNING id, username, instance_domain, did`,
		remoteActor.Username,
		remoteActor.Domain,
		remoteActor.ActorURI,
		remoteActor.DisplayName,
		remoteActor.AvatarURL,
		remoteActor.PublicKeyPEM,
		"", // Encryption key - might need to fetch this separately if supported
	).Scan(&user.ID, &user.Username, &user.InstanceDomain, &user.DID)

	if err != nil {
		return nil, fmt.Errorf("failed to create ghost user: %w", err)
	}

	return &user, nil
}

// BuildCreateDMActivity creates a Create activity wrapping a Note (DM)
func BuildCreateDMActivity(actorURI, recipientURI, content string) *Activity {
	domain := GetInstanceDomain()
	baseURL := resolveInstanceURL(domain)

	// Generate IDs
	postID := fmt.Sprintf("%d", time.Now().UnixNano())

	note := Note{
		ID:           fmt.Sprintf("%s/posts/%s", baseURL, postID),
		Type:         "Note",
		AttributedTo: actorURI,
		Content:      content,
		Published:    time.Now().UTC().Format(time.RFC3339),
		To:           []string{recipientURI}, // Addressed to specific user only
	}

	return &Activity{
		Context: "https://www.w3.org/ns/activitystreams",
		ID:      fmt.Sprintf("%s/activities/create-%s", baseURL, postID),
		Type:    "Create",
		Actor:   actorURI,
		Object:  note,
		To:      []string{recipientURI},
	}
}

// resolveActorFromURI resolves a remote actor from their URI
func resolveActorFromURI(actorURI string) (*RemoteActor, error) {
	username := extractUsernameFromURI(actorURI)
	domain := extractDomainFromURI(actorURI)

	if username == "" || domain == "" {
		// Fallback: try to fetch actor directly if parsing fails
		// This handles cases where URI structure might differ (e.g. Mastodon/Misskey)
		// but for now we rely on our known patterns or fetchActor
		actor, err := fetchActor(actorURI)
		if err != nil {
			return nil, fmt.Errorf("could not parse or fetch actor URI: %s", actorURI)
		}
		actor.Domain = domain // Might be empty if extraction failed
		return actor, nil
	}
	return ResolveRemoteUser(fmt.Sprintf("%s@%s", username, domain))
}

// ExtractUsernameFromURI extracts the username from an actor URI
func extractUsernameFromURI(uri string) string {
	// Support http://domain/users/username format
	parts := splitURI(uri)
	for i, p := range parts {
		if p == "users" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

// ExtractDomainFromURI extracts the domain from an actor URI
func extractDomainFromURI(uri string) string {
	// Check against known instances first (for local testing)
	for domain, baseURL := range InstanceURLMap {
		if len(uri) > len(baseURL) && uri[:len(baseURL)] == baseURL {
			return domain
		}
	}

	// Otherwise parse from URL
	// http://example.com/... -> example.com
	// This is sophisticated enough for now
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
	current := ""
	for _, ch := range path {
		if ch == '/' {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
