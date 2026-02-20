package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

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

		// Also search known remote instances directly (not just cache)
		for remoteDomain, baseURL := range federation.InstanceURLMap {
			if remoteDomain == h.cfg.Federation.Domain {
				continue
			}

			remoteUsers := fetchRemoteUserList(baseURL)
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

	// Store follow locally (as pending until Accept received)
	// Use actor URI so it matches remote post author_did values in inbox processing
	remoteDID := remoteActor.ActorURI
	_, err = db.GetDB().Exec(c.Request().Context(),
		`INSERT INTO follows (follower_did, following_did, status)
		 VALUES ($1, $2, 'pending')
		 ON CONFLICT DO NOTHING`,
		localUser.DID, remoteDID,
	)
	if err != nil {
		log.Printf("[Federation] Failed to store pending follow: %v", err)
	}

	// Send Follow activity
	localActorURI := fmt.Sprintf("%s/ap/users/%s", h.cfg.Federation.URL, localUser.Username)
	err = federation.SendFollow(localActorURI, remoteActor)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("failed to send follow: %v", err),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "follow_sent",
		"target":  req.Handle,
		"message": "Follow request sent. Waiting for acceptance.",
	})
}

// GetFederatedTimeline returns posts from all instances (local + remote)
// GET /api/v1/federation/timeline
func (h *FederationHandler) GetFederatedTimeline(c echo.Context) error {
	ctx := c.Request().Context()

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
		 WHERE p.deleted_at IS NULL AND p.visibility = 'public'
		 ORDER BY p.created_at DESC
		 LIMIT 50`)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to get timeline",
		})
	}
	defer rows.Close()

	var posts []map[string]interface{}
	for rows.Next() {
		var id, authorDID, content, visibility, originalURI, username, displayName, avatarURL string
		var isRemote bool
		var likeCount, repostCount int
		var createdAt interface{}

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
			"created_at":        createdAt,
			"username":          username,
			"display_name":      displayName,
			"avatar_url":        avatarURL,
		}

		// For remote posts, try to get info from remote_actors
		if isRemote && username == "" {
			post["username"] = extractUsernameFromDID(authorDID)
			post["domain"] = extractDomainFromDID(authorDID)
		}

		posts = append(posts, post)
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"posts": posts,
		"total": len(posts),
	})
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

	// Remote users from other known instances
	for domain, baseURL := range federation.InstanceURLMap {
		if domain == h.cfg.Federation.Domain {
			continue
		}
		remoteUsers := fetchRemoteUserList(baseURL)
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
	parts := strings.Split(did, ":")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return did
}

// Helper: extract domain from DID like did:web:splitter-1:alice → splitter-1
func extractDomainFromDID(did string) string {
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

	users, _, err := h.userRepo.GetAllUsers(ctx, limit, 0)
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
		"total":  len(publicUsers),
		"domain": h.cfg.Federation.Domain,
	})
}
