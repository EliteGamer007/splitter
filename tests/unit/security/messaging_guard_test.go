package security_test

import (
	"testing"

	"splitter/internal/security"
)

func TestMessagingGuard_LocalSenderRateLimit(t *testing.T) {
	guard := security.NewMessagingGuard()

	for i := 0; i < 20; i++ {
		allowed, reason := guard.AllowLocalSend("sender-1", 1)
		if !allowed {
			t.Fatalf("expected request %d to be allowed, got throttled: %s", i+1, reason)
		}
	}

	allowed, reason := guard.AllowLocalSend("sender-1", 1)
	if allowed {
		t.Fatalf("expected 21st request in minute to be throttled")
	}
	if reason == "" {
		t.Fatalf("expected throttle reason to be populated")
	}

	snapshot := guard.Snapshot()
	if snapshot.Metrics.LocalSendThrottled < 1 {
		t.Fatalf("expected LocalSendThrottled metric to increment")
	}
}

func TestMessagingGuard_RemoteInboundRateLimit(t *testing.T) {
	guard := security.NewMessagingGuard()

	for i := 0; i < 40; i++ {
		allowed, reason := guard.AllowRemoteInbound("https://remote.example/ap/users/alice", "remote.example")
		if !allowed {
			t.Fatalf("expected remote request %d to be allowed, got throttled: %s", i+1, reason)
		}
	}

	allowed, reason := guard.AllowRemoteInbound("https://remote.example/ap/users/alice", "remote.example")
	if allowed {
		t.Fatalf("expected 41st remote actor request to be throttled")
	}
	if reason == "" {
		t.Fatalf("expected remote throttle reason to be populated")
	}

	snapshot := guard.Snapshot()
	if snapshot.Metrics.InboxThrottled < 1 {
		t.Fatalf("expected InboxThrottled metric to increment")
	}
}

func TestMessagingGuard_SuspiciousEventRecording(t *testing.T) {
	guard := security.NewMessagingGuard()

	guard.RecordSuspicious("source-a", "test suspicious event", map[string]interface{}{"key": "value"})
	guard.RecordInboxRejected("source-b", "signature invalid", map[string]interface{}{"activity": "Create"})

	snapshot := guard.Snapshot()
	if snapshot.Metrics.SuspiciousEventsLogged < 2 {
		t.Fatalf("expected at least 2 suspicious events logged, got %d", snapshot.Metrics.SuspiciousEventsLogged)
	}
	if snapshot.Metrics.InboxRejected < 1 {
		t.Fatalf("expected InboxRejected metric to increment")
	}
	if len(snapshot.RecentEvents) < 2 {
		t.Fatalf("expected recent events to include logged security events")
	}
}
