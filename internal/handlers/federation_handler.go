package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"splitter/internal/config"
	"splitter/internal/db"
	"splitter/internal/federation"
	"splitter/internal/repository"

	"github.com/labstack/echo/v4"
)

// FederationHandler handles federation-specific API endpoints
type FederationHandler struct {
	userRepo *repository.UserRepository
	cfg      *config.Config
}

// NewFederationHandler creates a new FederationHandler
func NewFederationHandler(userRepo *repository.UserRepository, cfg *config.Config) *FederationHandler {
	return &FederationHandler{userRepo: userRepo, cfg: cfg}
}

// SearchRemoteUsers searches for users across all known instances
// GET /api/v1/federation/users?q=@alice@splitter-1
func (h *FederationHandler) SearchRemoteUsers(c echo.Context) error {
	query := c.QueryParam("q")
	if query == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "query parameter 'q' required",
		})
	}

	var results []map[string]interface{}
	localDomains := map[string]bool{
		h.cfg.Federation.Domain: true,
		"localhost":             true,
		"":                      true,
	}

	// If handle format (@user@domain), resolve specific user
	if strings.Contains(query, "@") {
		handle := strings.TrimPrefix(query, "@")
		parts := strings.SplitN(handle, "@", 2)

		if len(parts) == 2 {
			username := parts[0]
			domain := parts[1]

			// If it's our domain, search locally
			if domain == h.cfg.Federation.Domain {
				users, err := h.userRepo.SearchUsers(c.Request().Context(), username, 10, 0)
				if err == nil {
					for _, u := range users {
						if !localDomains[u.InstanceDomain] {
							continue
						}
						results = append(results, map[string]interface{}{
							"id":                    u.ID,
							"username":              u.Username,
							"display_name":          u.DisplayName,
							"domain":                h.cfg.Federation.Domain,
							"avatar_url":            u.AvatarURL,
							"did":                   u.DID,
							"encryption_public_key": u.EncryptionPublicKey,
							"is_remote":             false,
						})
					}
				}
			} else {
				// Resolve remote user via WebFinger
				actor, err := federation.ResolveRemoteUser(handle)
				if err != nil {
					log.Printf("[Federation] Failed to resolve %s: %v", handle, err)
				} else if actor != nil {
					// Ensure ghost user to get local ID for messaging
					var userID string
					var encryptionPublicKey string
					if user, err := federation.EnsureRemoteUser(c.Request().Context(), actor.ActorURI); err == nil {
						userID = user.ID
						encryptionPublicKey = user.EncryptionPublicKey
					} else {
						log.Printf("[Federation] Failed to ensure user: %v", err)
					}

					if encryptionPublicKey == "" {
						encryptionPublicKey = actor.EncryptionPublicKey
					}

					results = append(results, map[string]interface{}{
						"id":                    userID,
						"username":              actor.Username,
						"display_name":          actor.DisplayName,
						"domain":                actor.Domain,
						"avatar_url":            actor.AvatarURL,
						"actor_uri":             actor.ActorURI,
						"encryption_public_key": encryptionPublicKey,
						"is_remote":             true,
					})
				}
			}
		}
	} else {
		// Plain search — search local users and cached remote actors
		localUsers, err := h.userRepo.SearchUsers(c.Request().Context(), query, 10, 0)
		if err == nil {
			for _, u := range localUsers {
				if !localDomains[u.InstanceDomain] {
					continue
				}
				results = append(results, map[string]interface{}{
					"id":                    u.ID,
					"username":              u.Username,
					"display_name":          u.DisplayName,
					"domain":                h.cfg.Federation.Domain,
					"avatar_url":            u.AvatarURL,
					"did":                   u.DID,
					"encryption_public_key": u.EncryptionPublicKey,
					"is_remote":             false,
				})
			}
		}

		// Also search known remote instances directly (with failover)
		for remoteDomain, baseURL := range federation.InstanceURLMap {
			if remoteDomain == h.cfg.Federation.Domain {
				continue
			}

			var remoteUsers []map[string]interface{}
			if federation.IsPeerHealthy(remoteDomain) {
				remoteUsers = federation.FetchAndCachePeerUsers(remoteDomain, baseURL)
			} else {
				remoteUsers = federation.GetCachedPeerUsers(remoteDomain)
			}
			for _, u := range remoteUsers {
				username, _ := u["username"].(string)
				displayName, _ := u["display_name"].(string)
				avatarURL, _ := u["avatar_url"].(string)

				queryLower := strings.ToLower(query)
				if !strings.Contains(strings.ToLower(username), queryLower) &&
					!strings.Contains(strings.ToLower(displayName), queryLower) {
					continue
				}

				// Ensure ghost user for local thread creation and DM support
				actorURI := fmt.Sprintf("%s/ap/users/%s", baseURL, username)
				var userID string
				if ghostUser, err := federation.EnsureRemoteUser(c.Request().Context(), actorURI); err == nil {
					userID = ghostUser.ID
				}

				results = append(results, map[string]interface{}{
					"id":                    userID,
					"username":              username,
					"display_name":          displayName,
					"domain":                remoteDomain,
					"avatar_url":            avatarURL,
					"encryption_public_key": u["encryption_public_key"],
					"actor_uri":             actorURI,
					"is_remote":             true,
				})
			}
		}

		// Also search cached remote actors
		remoteActors, err := federation.GetAllRemoteActors(c.Request().Context())
		if err == nil {
			for _, a := range remoteActors {
				if strings.Contains(strings.ToLower(a.Username), strings.ToLower(query)) ||
					strings.Contains(strings.ToLower(a.DisplayName), strings.ToLower(query)) {
					// Ensure ghost user to get local ID
					var userID string
					if user, err := federation.EnsureRemoteUser(c.Request().Context(), a.ActorURI); err == nil {
						userID = user.ID
					} else {
						log.Printf("[Federation] Failed to ensure user: %v", err)
					}

					results = append(results, map[string]interface{}{
						"id":                    userID,
						"username":              a.Username,
						"display_name":          a.DisplayName,
						"domain":                a.Domain,
						"avatar_url":            a.AvatarURL,
						"actor_uri":             a.ActorURI,
						"encryption_public_key": a.EncryptionPublicKey,
						"is_remote":             true,
					})
				}
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"users": results,
		"total": len(results),
	})
}

// FollowRemoteUser initiates a follow to a remote user
// POST /api/v1/federation/follow {"handle": "@alice@splitter-1"}
func (h *FederationHandler) FollowRemoteUser(c echo.Context) error {
	// Get current user
	userID := c.Get("user_id").(string)
	localUser, err := h.userRepo.GetByID(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "user not found"})
	}

	var req struct {
		Handle string `json:"handle"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	// Resolve remote actor
	remoteActor, err := federation.ResolveRemoteUser(req.Handle)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("failed to resolve user: %v", err),
		})
	}

	// Ensure ghost user exists locally so GetFollowing JOIN works
	_, err = federation.EnsureRemoteUser(c.Request().Context(), remoteActor.ActorURI)
	if err != nil {
		log.Printf("[Federation] Failed to ensure remote user: %v", err)
	}

	// Store follow locally as accepted (both instances auto-accept)
	remoteDID := remoteActor.ActorURI
	_, err = db.GetDB().Exec(c.Request().Context(),
		`INSERT INTO follows (follower_did, following_did, status)
		 VALUES ($1, $2, 'accepted')
		 ON CONFLICT (follower_did, following_did) DO UPDATE SET status = 'accepted'`,
		localUser.DID, remoteDID,
	)
	if err != nil {
		log.Printf("[Federation] Failed to store follow: %v", err)
	}

	// Send Follow activity to remote instance
	localActorURI := fmt.Sprintf("%s/ap/users/%s", h.cfg.Federation.URL, localUser.Username)
	err = federation.SendFollow(localActorURI, remoteActor)
	if err != nil {
		log.Printf("[Federation] Failed to send follow activity: %v", err)
		// Don't fail — local follow is already stored
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "follow_accepted",
		"target":  req.Handle,
		"message": "Follow successful.",
	})
}

// GetFederatedTimeline returns posts from all instances (local + remote)
// GET /api/v1/federation/timeline
func (h *FederationHandler) GetFederatedTimeline(c echo.Context) error {
	ctx := c.Request().Context()

	// Phase 1: Get LOCAL posts from DB only (remote posts come from live fetch in Phase 2)
	rows, err := db.GetDB().Query(ctx,
		`SELECT p.id, p.author_did, p.content, p.visibility, p.is_remote,
		        COALESCE(p.original_post_uri, ''),
		        COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'like'), 0) as like_count,
		        COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'repost'), 0) as repost_count,
		        p.created_at,
		        COALESCE(u.username, ''),
		        COALESCE(u.display_name, ''),
		        COALESCE(u.avatar_url, '')
		 FROM posts p
		 LEFT JOIN users u ON u.did = p.author_did
		 WHERE p.deleted_at IS NULL AND p.visibility = 'public' AND p.is_remote = false
		 ORDER BY p.created_at DESC
		 LIMIT 50`)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to get timeline",
		})
	}
	defer rows.Close()

	// Dedup map: key = author_did|content, value = post (prefer version with username)
	seen := make(map[string]map[string]interface{})
	addPost := func(post map[string]interface{}) {
		author, _ := post["author_did"].(string)
		content, _ := post["content"].(string)
		key := author + "|" + content
		if existing, ok := seen[key]; ok {
			// Prefer the version that has a username
			existingUser, _ := existing["username"].(string)
			newUser, _ := post["username"].(string)
			if existingUser == "" && newUser != "" {
				seen[key] = post
			}
			return
		}
		seen[key] = post
	}

	for rows.Next() {
		var id, authorDID, content, visibility, originalURI, username, displayName, avatarURL string
		var isRemote bool
		var likeCount, repostCount int
		var createdAt time.Time

		if err := rows.Scan(&id, &authorDID, &content, &visibility, &isRemote,
			&originalURI, &likeCount, &repostCount, &createdAt,
			&username, &displayName, &avatarURL); err != nil {
			continue
		}

		post := map[string]interface{}{
			"id":                id,
			"author_did":        authorDID,
			"content":           content,
			"visibility":        visibility,
			"is_remote":         isRemote,
			"original_post_uri": originalURI,
			"like_count":        likeCount,
			"repost_count":      repostCount,
			"created_at":        createdAt.Format(time.RFC3339Nano),
			"username":          username,
			"display_name":      displayName,
			"avatar_url":        avatarURL,
		}

		// For remote posts, try to get info from remote_actors
		if isRemote && username == "" {
			post["username"] = extractUsernameFromDID(authorDID)
			post["domain"] = extractDomainFromDID(authorDID)
		}

		addPost(post)
	}

	// Phase 2: Live fetch from healthy peers (with cache fallback)
	for domain, baseURL := range federation.InstanceURLMap {
		if domain == h.cfg.Federation.Domain {
			continue
		}

		var remotePosts []map[string]interface{}

		if federation.IsPeerHealthy(domain) {
			remotePosts = federation.FetchAndCachePeerPosts(domain, baseURL)
		} else {
			remotePosts = federation.GetCachedPeerPosts(domain)
			if remotePosts != nil {
				log.Printf("[Federation] Serving %d cached posts for down peer %s", len(remotePosts), domain)
			}
		}

		for _, remotePost := range remotePosts {
			authorDID, _ := remotePost["author_did"].(string)
			content, _ := remotePost["content"].(string)
			visibility, _ := remotePost["visibility"].(string)
			if visibility == "" {
				visibility = "public"
			}
			if visibility != "public" {
				continue
			}

			username, _ := remotePost["username"].(string)
			displayName, _ := remotePost["display_name"].(string)
			avatarURL, _ := remotePost["avatar_url"].(string)
			createdAt, _ := remotePost["created_at"].(string)
			id, _ := remotePost["id"].(string)
			likeCount, _ := remotePost["like_count"].(float64)
			repostCount, _ := remotePost["repost_count"].(float64)
			originalURI, _ := remotePost["original_post_uri"].(string)

			if authorDID == "" && username != "" {
				authorDID = fmt.Sprintf("%s/ap/users/%s", baseURL, username)
			}

			post := map[string]interface{}{
				"id":                id,
				"author_did":        authorDID,
				"content":           content,
				"visibility":        visibility,
				"is_remote":         true,
				"original_post_uri": originalURI,
				"like_count":        int(likeCount),
				"repost_count":      int(repostCount),
				"created_at":        createdAt,
				"username":          username,
				"display_name":      displayName,
				"avatar_url":        avatarURL,
				"domain":            domain,
				"instance_url":      baseURL,
			}

			addPost(post)
		}
	}

	// Collect deduped posts, sort by created_at DESC, limit to 50
	posts := make([]map[string]interface{}, 0, len(seen))
	for _, post := range seen {
		posts = append(posts, post)
	}

	sort.Slice(posts, func(i, j int) bool {
		ti := parseCreatedAt(posts[i]["created_at"])
		tj := parseCreatedAt(posts[j]["created_at"])
		return ti.After(tj)
	})

	if len(posts) > 50 {
		posts = posts[:50]
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"posts": posts,
		"total": len(posts),
	})
}

// parseCreatedAt parses a created_at value (string or time.Time) into time.Time for sorting.
func parseCreatedAt(v interface{}) time.Time {
	switch val := v.(type) {
	case time.Time:
		return val
	case string:
		for _, layout := range []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05.999999Z",
		} {
			if t, err := time.Parse(layout, val); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}

// GetAllFederatedUsers returns users from all instances
// GET /api/v1/federation/all-users
func (h *FederationHandler) GetAllFederatedUsers(c echo.Context) error {
	ctx := c.Request().Context()

	var allUsers []map[string]interface{}

	// Local users (filter out ghost users from remote instances)
	localUsers, _, err := h.userRepo.GetAllUsers(ctx, 100, 0)
	localDomains := map[string]bool{
		h.cfg.Federation.Domain: true,
		"localhost":             true,
		"":                      true,
	}
	if err == nil {
		for _, u := range localUsers {
			if !localDomains[u.InstanceDomain] {
				continue // skip ghost users
			}
			allUsers = append(allUsers, map[string]interface{}{
				"id":                    u.ID,
				"username":              u.Username,
				"display_name":          u.DisplayName,
				"domain":                h.cfg.Federation.Domain,
				"avatar_url":            u.AvatarURL,
				"did":                   u.DID,
				"encryption_public_key": u.EncryptionPublicKey,
				"is_remote":             false,
				"bio":                   u.Bio,
			})
		}
	}

	// Remote users from other known instances (with failover)
	for domain, baseURL := range federation.InstanceURLMap {
		if domain == h.cfg.Federation.Domain {
			continue
		}

		var remoteUsers []map[string]interface{}
		if federation.IsPeerHealthy(domain) {
			remoteUsers = federation.FetchAndCachePeerUsers(domain, baseURL)
		} else {
			// Peer is down — serve from cache
			remoteUsers = federation.GetCachedPeerUsers(domain)
		}

		for _, u := range remoteUsers {
			allUsers = append(allUsers, map[string]interface{}{
				"id":                    u["id"],
				"username":              u["username"],
				"display_name":          u["display_name"],
				"domain":                domain,
				"avatar_url":            u["avatar_url"],
				"did":                   u["did"],
				"encryption_public_key": u["encryption_public_key"],
				"is_remote":             true,
				"bio":                   u["bio"],
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"users": allUsers,
		"total": len(allUsers),
	})
}

// fetchRemoteUserList fetches users from a remote instance's public federation API
func fetchRemoteUserList(baseURL string) []map[string]interface{} {
	client := &http.Client{}
	resp, err := client.Get(baseURL + "/api/v1/federation/public-users?limit=100")
	if err != nil {
		log.Printf("[Federation] Failed to fetch users from %s: %v", baseURL, err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("[Federation] Non-200 status fetching users from %s: %d", baseURL, resp.StatusCode)
		return nil
	}

	var data struct {
		Users []map[string]interface{} `json:"users"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("[Federation] Failed to decode user list from %s: %v", baseURL, err)
		return nil
	}

	return data.Users
}

// Helper: extract username from DID like did:web:splitter-1:alice → alice
func extractUsernameFromDID(did string) string {
	if strings.HasPrefix(did, "http://") || strings.HasPrefix(did, "https://") {
		parsed, err := url.Parse(did)
		if err == nil {
			parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
			if len(parts) >= 2 {
				return parts[len(parts)-1]
			}
		}
	}

	parts := strings.Split(did, ":")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return did
}

// Helper: extract domain from DID like did:web:splitter-1:alice → splitter-1
func extractDomainFromDID(did string) string {
	if strings.HasPrefix(did, "http://") || strings.HasPrefix(did, "https://") {
		parsed, err := url.Parse(did)
		if err == nil {
			host := parsed.Host
			if strings.Contains(host, "localhost:8001") || strings.Contains(host, "splitter-2.onrender.com") {
				return "splitter-2"
			}
			if strings.Contains(host, "localhost:8000") || strings.Contains(host, "splitter-m0kv.onrender.com") {
				return "splitter-1"
			}
			if host != "" {
				return parsed.Hostname()
			}
		}
	}

	parts := strings.Split(did, ":")
	if len(parts) >= 3 {
		return parts[2]
	}
	return ""
}

// GetPublicUserList returns a public list of local users for federation discovery
// GET /api/v1/federation/public-users
func (h *FederationHandler) GetPublicUserList(c echo.Context) error {
	ctx := c.Request().Context()

	limit := 100
	if l := c.QueryParam("limit"); l != "" {
		var parsed int
		fmt.Sscanf(l, "%d", &parsed)
		if parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	users, total, err := h.userRepo.GetAllUsers(ctx, limit, 0)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to get users",
		})
	}

	var publicUsers []map[string]interface{}
	localDomains := map[string]bool{
		h.cfg.Federation.Domain: true,
		"localhost":             true,
		"":                      true,
	}
	for _, u := range users {
		// Only expose local users (skip ghost users from remote instances)
		if !localDomains[u.InstanceDomain] {
			continue
		}
		publicUsers = append(publicUsers, map[string]interface{}{
			"id":                    u.ID,
			"username":              u.Username,
			"display_name":          u.DisplayName,
			"avatar_url":            u.AvatarURL,
			"bio":                   u.Bio,
			"did":                   u.DID,
			"encryption_public_key": u.EncryptionPublicKey,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"users":  publicUsers,
		"total":  total,
		"domain": h.cfg.Federation.Domain,
	})
}

// GetFederationHealth returns health status of all peer instances
// GET /api/v1/federation/health
func (h *FederationHandler) GetFederationHealth(c echo.Context) error {
	peers := federation.HealthStatusJSON()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"self_domain": h.cfg.Federation.Domain,
		"peers":       peers,
	})
}

// GetMigrationStatus returns user migration status
// GET /api/v1/federation/migrations
func (h *FederationHandler) GetMigrationStatus(c echo.Context) error {
	migrations, err := federation.GetMigrationStatus(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to get migration status",
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"migrations": migrations,
		"total":      len(migrations),
	})
}
