package federation

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"splitter/internal/db"
)

// InstanceHealth tracks the health status of a peer federation instance.
type InstanceHealth struct {
	Domain           string        `json:"domain"`
	URL              string        `json:"url"`
	IsHealthy        bool          `json:"is_healthy"`
	LastCheckedAt    time.Time     `json:"last_checked_at"`
	LastHealthyAt    time.Time     `json:"last_healthy_at"`
	FailingSince     time.Time     `json:"failing_since,omitempty"`
	ConsecutiveFails int           `json:"consecutive_fails"`
	Latency          time.Duration `json:"latency_ms"`
}

var (
	healthMu     sync.RWMutex
	healthStatus = make(map[string]*InstanceHealth)
)

// GetPeerHealth returns the current health status of a peer domain.
func GetPeerHealth(domain string) *InstanceHealth {
	healthMu.RLock()
	defer healthMu.RUnlock()
	if h, ok := healthStatus[domain]; ok {
		copied := *h
		return &copied
	}
	return nil
}

// GetAllPeerHealth returns health status for all tracked peers.
func GetAllPeerHealth() map[string]*InstanceHealth {
	healthMu.RLock()
	defer healthMu.RUnlock()
	result := make(map[string]*InstanceHealth, len(healthStatus))
	for k, v := range healthStatus {
		copied := *v
		result[k] = &copied
	}
	return result
}

// IsPeerHealthy returns true if the given domain's last health check succeeded.
func IsPeerHealthy(domain string) bool {
	healthMu.RLock()
	defer healthMu.RUnlock()
	if h, ok := healthStatus[domain]; ok {
		return h.IsHealthy
	}
	return true // optimistic default
}

// ProbePeerHealth performs a single health check against a peer instance.
func ProbePeerHealth(domain, baseURL string) *InstanceHealth {
	start := time.Now()
	client := &http.Client{Timeout: 10 * time.Second}

	healthy := false
	resp, err := client.Get(baseURL + "/api/v1/federation/public-users?limit=1")
	if err == nil {
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			healthy = true
		}
		resp.Body.Close()
	}

	latency := time.Since(start)

	healthMu.Lock()
	defer healthMu.Unlock()

	existing, ok := healthStatus[domain]
	if !ok {
		existing = &InstanceHealth{
			Domain:        domain,
			URL:           baseURL,
			LastHealthyAt: time.Now(),
		}
		healthStatus[domain] = existing
	}

	existing.LastCheckedAt = time.Now()
	existing.Latency = latency

	if healthy {
		existing.IsHealthy = true
		existing.LastHealthyAt = time.Now()
		existing.ConsecutiveFails = 0
		existing.FailingSince = time.Time{}
	} else {
		existing.IsHealthy = false
		existing.ConsecutiveFails++
		if existing.FailingSince.IsZero() {
			existing.FailingSince = time.Now()
		}
	}

	return existing
}

// RunHealthCheckLoop periodically probes all peer instances.
// Runs every 60 seconds. Should be called in a goroutine.
func RunHealthCheckLoop(ctx context.Context, selfDomain string) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	// Run immediately on start
	probeAllPeers(selfDomain)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			probeAllPeers(selfDomain)
		}
	}
}

func probeAllPeers(selfDomain string) {
	for domain, baseURL := range InstanceURLMap {
		if domain == selfDomain {
			continue
		}
		h := ProbePeerHealth(domain, baseURL)
		if h.IsHealthy {
			log.Printf("[HealthCheck] %s (%s) is HEALTHY (latency: %dms)", domain, baseURL, h.Latency.Milliseconds())
		} else {
			log.Printf("[HealthCheck] %s (%s) is DOWN (consecutive failures: %d, failing since: %s)",
				domain, baseURL, h.ConsecutiveFails, h.FailingSince.Format(time.RFC3339))
		}

		// Persist health state to DB for cross-restart tracking
		persistHealthState(domain, h)
	}
}

func persistHealthState(domain string, h *InstanceHealth) {
	ctx := context.Background()
	_, _ = db.GetDB().Exec(ctx, `
		INSERT INTO federation_failures (domain, failure_count, last_failure_at, circuit_open_until)
		VALUES ($1, $2, $3, NULL)
		ON CONFLICT (domain) DO UPDATE SET
			failure_count = CASE WHEN $4 THEN 0 ELSE $2 END,
			last_failure_at = CASE WHEN $4 THEN federation_failures.last_failure_at ELSE $3 END
	`, domain, h.ConsecutiveFails, h.LastCheckedAt, h.IsHealthy)
}

// CachedPeerData stores cached user lists and posts from peer instances
// so we can serve them when the peer is down.
type CachedPeerData struct {
	Users    []map[string]interface{} `json:"users"`
	Posts    []map[string]interface{} `json:"posts"`
	CachedAt time.Time                `json:"cached_at"`
}

var (
	peerCacheMu sync.RWMutex
	peerCache   = make(map[string]*CachedPeerData)
)

// CachePeerUsers stores a snapshot of a peer's user list.
func CachePeerUsers(domain string, users []map[string]interface{}) {
	peerCacheMu.Lock()
	defer peerCacheMu.Unlock()
	if _, ok := peerCache[domain]; !ok {
		peerCache[domain] = &CachedPeerData{}
	}
	peerCache[domain].Users = users
	peerCache[domain].CachedAt = time.Now()
}

// CachePeerPosts stores a snapshot of a peer's public posts.
func CachePeerPosts(domain string, posts []map[string]interface{}) {
	peerCacheMu.Lock()
	defer peerCacheMu.Unlock()
	if _, ok := peerCache[domain]; !ok {
		peerCache[domain] = &CachedPeerData{}
	}
	peerCache[domain].Posts = posts
	peerCache[domain].CachedAt = time.Now()
}

// GetCachedPeerUsers returns cached user list for a down peer.
func GetCachedPeerUsers(domain string) []map[string]interface{} {
	peerCacheMu.RLock()
	defer peerCacheMu.RUnlock()
	if data, ok := peerCache[domain]; ok {
		return data.Users
	}
	return nil
}

// GetCachedPeerPosts returns cached posts for a down peer.
func GetCachedPeerPosts(domain string) []map[string]interface{} {
	peerCacheMu.RLock()
	defer peerCacheMu.RUnlock()
	if data, ok := peerCache[domain]; ok {
		return data.Posts
	}
	return nil
}

// FetchAndCachePeerUsers fetches and caches users from a healthy peer.
func FetchAndCachePeerUsers(domain, baseURL string) []map[string]interface{} {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(baseURL + "/api/v1/federation/public-users?limit=100")
	if err != nil {
		log.Printf("[FederationCache] Failed to fetch users from %s: %v", domain, err)
		return GetCachedPeerUsers(domain)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("[FederationCache] Non-200 from %s: %d", domain, resp.StatusCode)
		return GetCachedPeerUsers(domain)
	}

	var data struct {
		Users []map[string]interface{} `json:"users"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("[FederationCache] Decode error from %s: %v", domain, err)
		return GetCachedPeerUsers(domain)
	}

	CachePeerUsers(domain, data.Users)
	return data.Users
}

// FetchAndCachePeerPosts fetches and caches posts from a healthy peer.
func FetchAndCachePeerPosts(domain, baseURL string) []map[string]interface{} {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(baseURL + "/api/v1/posts/public?limit=50&offset=0")
	if err != nil {
		log.Printf("[FederationCache] Failed to fetch posts from %s: %v", domain, err)
		return GetCachedPeerPosts(domain)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("[FederationCache] Non-200 posts from %s: %d", domain, resp.StatusCode)
		return GetCachedPeerPosts(domain)
	}

	var remotePosts []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&remotePosts); err != nil {
		log.Printf("[FederationCache] Decode error posts from %s: %v", domain, err)
		return GetCachedPeerPosts(domain)
	}

	CachePeerPosts(domain, remotePosts)
	return remotePosts
}

// GetPeerDownDuration returns how long a peer has been continuously failing.
// Returns 0 if peer is healthy or not tracked.
func GetPeerDownDuration(domain string) time.Duration {
	healthMu.RLock()
	defer healthMu.RUnlock()
	if h, ok := healthStatus[domain]; ok && !h.IsHealthy && !h.FailingSince.IsZero() {
		return time.Since(h.FailingSince)
	}
	return 0
}

// GetPeerDownDurationFromDB checks the DB for persistent failure tracking.
// Returns the duration since the first continuous failure.
func GetPeerDownDurationFromDB(ctx context.Context, domain string) time.Duration {
	var lastFailure *time.Time
	var failCount int
	err := db.GetDB().QueryRow(ctx,
		`SELECT failure_count, last_failure_at FROM federation_failures WHERE domain = $1`,
		domain,
	).Scan(&failCount, &lastFailure)
	if err != nil || lastFailure == nil || failCount == 0 {
		return 0
	}
	return time.Since(*lastFailure)
}

// HealthStatusJSON returns a JSON-friendly summary for the health API endpoint.
func HealthStatusJSON() []map[string]interface{} {
	healthMu.RLock()
	defer healthMu.RUnlock()

	var result []map[string]interface{}
	for _, h := range healthStatus {
		entry := map[string]interface{}{
			"domain":            h.Domain,
			"url":               h.URL,
			"is_healthy":        h.IsHealthy,
			"last_checked_at":   h.LastCheckedAt,
			"last_healthy_at":   h.LastHealthyAt,
			"consecutive_fails": h.ConsecutiveFails,
			"latency_ms":        h.Latency.Milliseconds(),
		}
		if !h.FailingSince.IsZero() {
			entry["failing_since"] = h.FailingSince
			entry["down_duration"] = fmt.Sprintf("%.0f hours", time.Since(h.FailingSince).Hours())
		}
		result = append(result, entry)
	}
	return result
}
