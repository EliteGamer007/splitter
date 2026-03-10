// Package security_test — logger bridge for unit tests.
package security_test

import (
	"testing"
	"time"

	"splitter/internal/security"
	"splitter/tests/testlogger"
)

func TestMessagingGuardLocalRateLimitLogged(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "unit", start, testErr) }()

	guard := security.NewMessagingGuard()
	for i := 0; i < 20; i++ {
		allowed, reason := guard.AllowLocalSend("sender-logged", 1)
		if !allowed {
			t.Fatalf("request %d should be allowed; got throttled: %s", i+1, reason)
		}
	}
	allowed, reason := guard.AllowLocalSend("sender-logged", 1)
	if allowed {
		t.Fatal("21st request should be throttled")
	}
	if reason == "" {
		t.Fatal("throttle reason must not be empty")
	}
	snap := guard.Snapshot()
	if snap.Metrics.LocalSendThrottled < 1 {
		t.Fatal("LocalSendThrottled metric should increment")
	}
}

func TestMessagingGuardRemoteInboundLogged(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "unit", start, testErr) }()

	guard := security.NewMessagingGuard()
	for i := 0; i < 40; i++ {
		allowed, reason := guard.AllowRemoteInbound("https://remote.example/ap/users/alice", "remote.example")
		if !allowed {
			t.Fatalf("request %d should be allowed; got throttled: %s", i+1, reason)
		}
	}
	allowed, reason := guard.AllowRemoteInbound("https://remote.example/ap/users/alice", "remote.example")
	if allowed {
		t.Fatal("41st remote request should be throttled")
	}
	if reason == "" {
		t.Fatal("throttle reason must not be empty")
	}
}

func TestMessagingGuardSuspiciousLogged(t *testing.T) {
	start := time.Now()
	var testErr error
	defer func() { testlogger.LogTestResult(t, "unit", start, testErr) }()

	guard := security.NewMessagingGuard()
	guard.RecordSuspicious("source-a", "test suspicious event", map[string]interface{}{"key": "value"})
	guard.RecordInboxRejected("source-b", "signature invalid", map[string]interface{}{"activity": "Create"})

	snap := guard.Snapshot()
	if snap.Metrics.SuspiciousEventsLogged < 2 {
		t.Fatalf("expected >= 2 suspicious events, got %d", snap.Metrics.SuspiciousEventsLogged)
	}
	if snap.Metrics.InboxRejected < 1 {
		t.Fatal("InboxRejected metric should increment")
	}
	if len(snap.RecentEvents) < 2 {
		t.Fatal("recent events should include logged events")
	}
}
