package security

import (
	"sync"
	"time"
)

type MessagingSecurityEvent struct {
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Action    string                 `json:"action"`
	Reason    string                 `json:"reason,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type MessagingSecurityMetrics struct {
	LocalSendAllowed       int `json:"local_send_allowed"`
	LocalSendThrottled     int `json:"local_send_throttled"`
	OfflineSyncAllowed     int `json:"offline_sync_allowed"`
	OfflineSyncThrottled   int `json:"offline_sync_throttled"`
	InboxAllowed           int `json:"inbox_allowed"`
	InboxThrottled         int `json:"inbox_throttled"`
	InboxRejected          int `json:"inbox_rejected"`
	SuspiciousEventsLogged int `json:"suspicious_events_logged"`
}

type MessagingSecuritySnapshot struct {
	Limits struct {
		LocalPerMinute        int `json:"local_per_minute"`
		LocalPerHour          int `json:"local_per_hour"`
		RemoteActorPerMinute  int `json:"remote_actor_per_minute"`
		RemoteDomainPerMinute int `json:"remote_domain_per_minute"`
	} `json:"limits"`
	Metrics      MessagingSecurityMetrics `json:"metrics"`
	RecentEvents []MessagingSecurityEvent `json:"recent_events"`
}

type MessagingGuard struct {
	mu sync.Mutex

	localSenderHits  map[string][]time.Time
	remoteActorHits  map[string][]time.Time
	remoteDomainHits map[string][]time.Time

	metrics MessagingSecurityMetrics
	events  []MessagingSecurityEvent
}

const (
	localPerMinuteLimit        = 20
	localPerHourLimit          = 120
	remoteActorPerMinuteLimit  = 40
	remoteDomainPerMinuteLimit = 200
	maxRecentEvents            = 200
)

var globalMessagingGuard = NewMessagingGuard()

func GetMessagingGuard() *MessagingGuard {
	return globalMessagingGuard
}

func NewMessagingGuard() *MessagingGuard {
	return &MessagingGuard{
		localSenderHits:  make(map[string][]time.Time),
		remoteActorHits:  make(map[string][]time.Time),
		remoteDomainHits: make(map[string][]time.Time),
		events:           make([]MessagingSecurityEvent, 0, maxRecentEvents),
	}
}

func trimWindow(hits []time.Time, windowStart time.Time) []time.Time {
	kept := hits[:0]
	for _, ts := range hits {
		if ts.After(windowStart) {
			kept = append(kept, ts)
		}
	}
	return kept
}

func (g *MessagingGuard) appendEvents(events []MessagingSecurityEvent, event MessagingSecurityEvent) []MessagingSecurityEvent {
	events = append(events, event)
	if len(events) > maxRecentEvents {
		events = events[len(events)-maxRecentEvents:]
	}
	return events
}

func (g *MessagingGuard) AllowLocalSend(senderID string, units int) (bool, string) {
	if units < 1 {
		units = 1
	}
	now := time.Now().UTC()

	g.mu.Lock()
	defer g.mu.Unlock()

	hits := g.localSenderHits[senderID]
	hits = trimWindow(hits, now.Add(-time.Hour))

	countLastHour := len(hits)
	countLastMinute := 0
	minuteStart := now.Add(-time.Minute)
	for _, ts := range hits {
		if ts.After(minuteStart) {
			countLastMinute++
		}
	}

	if countLastMinute+units > localPerMinuteLimit {
		g.metrics.LocalSendThrottled++
		g.events = g.appendEvents(g.events, MessagingSecurityEvent{
			Type:      "rate_limit",
			Source:    senderID,
			Action:    "throttled",
			Reason:    "local sender per-minute limit exceeded",
			Timestamp: now,
			Metadata: map[string]interface{}{
				"window":          "1m",
				"limit":           localPerMinuteLimit,
				"attempted_units": units,
			},
		})
		return false, "per-minute messaging rate limit exceeded"
	}

	if countLastHour+units > localPerHourLimit {
		g.metrics.LocalSendThrottled++
		g.events = g.appendEvents(g.events, MessagingSecurityEvent{
			Type:      "rate_limit",
			Source:    senderID,
			Action:    "throttled",
			Reason:    "local sender per-hour limit exceeded",
			Timestamp: now,
			Metadata: map[string]interface{}{
				"window":          "1h",
				"limit":           localPerHourLimit,
				"attempted_units": units,
			},
		})
		return false, "per-hour messaging rate limit exceeded"
	}

	for i := 0; i < units; i++ {
		hits = append(hits, now)
	}
	g.localSenderHits[senderID] = hits
	if units == 1 {
		g.metrics.LocalSendAllowed++
	} else {
		g.metrics.OfflineSyncAllowed++
	}
	return true, ""
}

func (g *MessagingGuard) AllowRemoteInbound(remoteActorURI, remoteDomain string) (bool, string) {
	now := time.Now().UTC()

	g.mu.Lock()
	defer g.mu.Unlock()

	actorHits := trimWindow(g.remoteActorHits[remoteActorURI], now.Add(-time.Minute))
	domainHits := trimWindow(g.remoteDomainHits[remoteDomain], now.Add(-time.Minute))

	if len(actorHits)+1 > remoteActorPerMinuteLimit {
		g.metrics.InboxThrottled++
		g.events = g.appendEvents(g.events, MessagingSecurityEvent{
			Type:      "rate_limit",
			Source:    remoteActorURI,
			Action:    "throttled",
			Reason:    "remote actor per-minute inbox limit exceeded",
			Timestamp: now,
			Metadata: map[string]interface{}{
				"window": "1m",
				"limit":  remoteActorPerMinuteLimit,
				"domain": remoteDomain,
			},
		})
		return false, "remote actor inbox rate limit exceeded"
	}

	if len(domainHits)+1 > remoteDomainPerMinuteLimit {
		g.metrics.InboxThrottled++
		g.events = g.appendEvents(g.events, MessagingSecurityEvent{
			Type:      "rate_limit",
			Source:    remoteDomain,
			Action:    "throttled",
			Reason:    "remote domain per-minute inbox limit exceeded",
			Timestamp: now,
			Metadata: map[string]interface{}{
				"window": "1m",
				"limit":  remoteDomainPerMinuteLimit,
				"actor":  remoteActorURI,
			},
		})
		return false, "remote domain inbox rate limit exceeded"
	}

	actorHits = append(actorHits, now)
	domainHits = append(domainHits, now)
	g.remoteActorHits[remoteActorURI] = actorHits
	g.remoteDomainHits[remoteDomain] = domainHits
	g.metrics.InboxAllowed++

	return true, ""
}

func (g *MessagingGuard) RecordSuspicious(source, reason string, metadata map[string]interface{}) {
	now := time.Now().UTC()
	g.mu.Lock()
	defer g.mu.Unlock()

	g.metrics.SuspiciousEventsLogged++
	g.events = g.appendEvents(g.events, MessagingSecurityEvent{
		Type:      "suspicious",
		Source:    source,
		Action:    "flagged",
		Reason:    reason,
		Timestamp: now,
		Metadata:  metadata,
	})
}

func (g *MessagingGuard) RecordInboxRejected(source, reason string, metadata map[string]interface{}) {
	now := time.Now().UTC()
	g.mu.Lock()
	defer g.mu.Unlock()

	g.metrics.InboxRejected++
	g.metrics.SuspiciousEventsLogged++
	g.events = g.appendEvents(g.events, MessagingSecurityEvent{
		Type:      "inbox_rejected",
		Source:    source,
		Action:    "rejected",
		Reason:    reason,
		Timestamp: now,
		Metadata:  metadata,
	})
}

func (g *MessagingGuard) Snapshot() MessagingSecuritySnapshot {
	g.mu.Lock()
	defer g.mu.Unlock()

	eventsCopy := make([]MessagingSecurityEvent, len(g.events))
	copy(eventsCopy, g.events)

	snapshot := MessagingSecuritySnapshot{
		Metrics:      g.metrics,
		RecentEvents: eventsCopy,
	}
	snapshot.Limits.LocalPerMinute = localPerMinuteLimit
	snapshot.Limits.LocalPerHour = localPerHourLimit
	snapshot.Limits.RemoteActorPerMinute = remoteActorPerMinuteLimit
	snapshot.Limits.RemoteDomainPerMinute = remoteDomainPerMinuteLimit
	return snapshot
}
