package federation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"splitter/internal/db"
)

// RemoteActor represents a cached remote user from another instance
type RemoteActor struct {
	ID            string    `json:"id"`
	ActorURI      string    `json:"actor_uri"`
	Username      string    `json:"username"`
	Domain        string    `json:"domain"`
	InboxURL      string    `json:"inbox_url"`
	OutboxURL     string    `json:"outbox_url"`
	PublicKeyPEM  string    `json:"public_key_pem"`
	DisplayName   string    `json:"display_name"`
	AvatarURL     string    `json:"avatar_url"`
	LastFetchedAt time.Time `json:"last_fetched_at"`
	CreatedAt     time.Time `json:"created_at"`
}

// InstanceURLMap maps domain names to actual URLs (for local testing)
var InstanceURLMap = map[string]string{
	"splitter-1": "http://localhost:8000",
	"splitter-2": "http://localhost:8001",
}

// ResolveRemoteUser resolves @username@domain to a RemoteActor
// 1. Check local cache (remote_actors table)
// 2. WebFinger lookup
// 3. Fetch Actor JSON
// 4. Cache and return
func ResolveRemoteUser(handle string) (*RemoteActor, error) {
	// Parse @username@domain or username@domain
	handle = strings.TrimPrefix(handle, "@")
	parts := strings.SplitN(handle, "@", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid handle format: expected user@domain, got %s", handle)
	}
	username := parts[0]
	domain := parts[1]

	ctx := context.Background()

	// 1. Check cache
	actor, err := getRemoteActorFromCache(ctx, username, domain)
	if err == nil && actor != nil {
		// Refresh if stale (older than 1 hour)
		if time.Since(actor.LastFetchedAt) < time.Hour {
			return actor, nil
		}
	}

	// 2. WebFinger lookup
	baseURL := resolveInstanceURL(domain)
	webfingerURL := fmt.Sprintf("%s/.well-known/webfinger?resource=acct:%s@%s", baseURL, username, domain)

	log.Printf("[Federation] WebFinger lookup: %s", webfingerURL)
	resp, err := httpGet(webfingerURL)
	if err != nil {
		return nil, fmt.Errorf("webfinger lookup failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("webfinger returned status %d", resp.StatusCode)
	}

	var jrd struct {
		Subject string `json:"subject"`
		Links   []struct {
			Rel  string `json:"rel"`
			Type string `json:"type"`
			Href string `json:"href"`
		} `json:"links"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jrd); err != nil {
		return nil, fmt.Errorf("failed to decode webfinger response: %w", err)
	}

	// Find actor link
	var actorURI string
	for _, link := range jrd.Links {
		if link.Rel == "self" && link.Type == "application/activity+json" {
			actorURI = link.Href
			break
		}
	}
	if actorURI == "" {
		return nil, fmt.Errorf("no actor link found in webfinger response")
	}

	// 3. Fetch Actor JSON
	log.Printf("[Federation] Fetching actor: %s", actorURI)
	actor, err = fetchActor(actorURI)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch actor: %w", err)
	}
	actor.Domain = domain

	// 4. Cache
	err = upsertRemoteActor(ctx, actor)
	if err != nil {
		log.Printf("[Federation] Warning: failed to cache actor: %v", err)
	}

	return actor, nil
}

// fetchActor fetches and parses an ActivityPub Actor document
func fetchActor(actorURI string) (*RemoteActor, error) {
	req, err := http.NewRequest("GET", actorURI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/activity+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("actor fetch returned %d: %s", resp.StatusCode, string(body))
	}

	var actorJSON struct {
		ID                string `json:"id"`
		Type              string `json:"type"`
		PreferredUsername string `json:"preferredUsername"`
		Name              string `json:"name"`
		Inbox             string `json:"inbox"`
		Outbox            string `json:"outbox"`
		Icon              *struct {
			URL string `json:"url"`
		} `json:"icon"`
		PublicKey *struct {
			ID           string `json:"id"`
			Owner        string `json:"owner"`
			PublicKeyPEM string `json:"publicKeyPem"`
		} `json:"publicKey"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&actorJSON); err != nil {
		return nil, fmt.Errorf("failed to decode actor: %w", err)
	}

	actor := &RemoteActor{
		ActorURI:    actorJSON.ID,
		Username:    actorJSON.PreferredUsername,
		InboxURL:    actorJSON.Inbox,
		OutboxURL:   actorJSON.Outbox,
		DisplayName: actorJSON.Name,
	}
	if actorJSON.PublicKey != nil {
		actor.PublicKeyPEM = actorJSON.PublicKey.PublicKeyPEM
	}
	if actorJSON.Icon != nil {
		actor.AvatarURL = actorJSON.Icon.URL
	}

	return actor, nil
}

// resolveInstanceURL maps a domain name to its actual URL
func resolveInstanceURL(domain string) string {
	if url, ok := InstanceURLMap[domain]; ok {
		return url
	}
	// If not in map, assume HTTPS
	return "https://" + domain
}

// getRemoteActorFromCache fetches a remote actor from the local cache
func getRemoteActorFromCache(ctx context.Context, username, domain string) (*RemoteActor, error) {
	var actor RemoteActor
	err := db.GetDB().QueryRow(ctx,
		`SELECT id, actor_uri, username, domain, inbox_url, COALESCE(outbox_url,''),
		        COALESCE(public_key_pem,''), COALESCE(display_name,''), COALESCE(avatar_url,''),
		        last_fetched_at, created_at
		 FROM remote_actors WHERE username = $1 AND domain = $2`,
		username, domain,
	).Scan(&actor.ID, &actor.ActorURI, &actor.Username, &actor.Domain,
		&actor.InboxURL, &actor.OutboxURL, &actor.PublicKeyPEM,
		&actor.DisplayName, &actor.AvatarURL, &actor.LastFetchedAt, &actor.CreatedAt)

	if err != nil {
		return nil, err
	}
	return &actor, nil
}

// upsertRemoteActor inserts or updates a remote actor in the cache
func upsertRemoteActor(ctx context.Context, actor *RemoteActor) error {
	_, err := db.GetDB().Exec(ctx,
		`INSERT INTO remote_actors (actor_uri, username, domain, inbox_url, outbox_url, public_key_pem, display_name, avatar_url, last_fetched_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
		 ON CONFLICT (actor_uri) DO UPDATE SET
		   inbox_url = $4, outbox_url = $5, public_key_pem = $6,
		   display_name = $7, avatar_url = $8, last_fetched_at = now()`,
		actor.ActorURI, actor.Username, actor.Domain, actor.InboxURL,
		actor.OutboxURL, actor.PublicKeyPEM, actor.DisplayName, actor.AvatarURL,
	)
	return err
}

// GetAllRemoteActors returns all cached remote actors
func GetAllRemoteActors(ctx context.Context) ([]*RemoteActor, error) {
	rows, err := db.GetDB().Query(ctx,
		`SELECT id, actor_uri, username, domain, inbox_url, COALESCE(outbox_url,''),
		        COALESCE(public_key_pem,''), COALESCE(display_name,''), COALESCE(avatar_url,''),
		        last_fetched_at, created_at
		 FROM remote_actors ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actors []*RemoteActor
	for rows.Next() {
		var a RemoteActor
		if err := rows.Scan(&a.ID, &a.ActorURI, &a.Username, &a.Domain,
			&a.InboxURL, &a.OutboxURL, &a.PublicKeyPEM,
			&a.DisplayName, &a.AvatarURL, &a.LastFetchedAt, &a.CreatedAt); err != nil {
			return nil, err
		}
		actors = append(actors, &a)
	}
	return actors, nil
}

// httpGet is a helper for simple GET requests with timeout
func httpGet(url string) (*http.Response, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	return client.Get(url)
}
