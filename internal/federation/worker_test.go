package federation

import (
	"testing"
	"time"
)

func TestCalculateRetryDelay(t *testing.T) {
	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{attempt: 1, expected: 15 * time.Second},
		{attempt: 2, expected: 30 * time.Second},
		{attempt: 3, expected: 60 * time.Second},
	}

	for _, tt := range tests {
		got := calculateRetryDelay(tt.attempt)
		if got != tt.expected {
			t.Fatalf("attempt=%d expected=%v got=%v", tt.attempt, tt.expected, got)
		}
	}
}

func TestCalculateRetryDelayCapped(t *testing.T) {
	got := calculateRetryDelay(20)
	if got != time.Hour {
		t.Fatalf("expected 1h cap, got %v", got)
	}
}

func TestFormatBackoffInterval(t *testing.T) {
	if got := formatBackoffInterval(30 * time.Second); got != "30 seconds" {
		t.Fatalf("expected '30 seconds', got %s", got)
	}
}
