package federation

import (
	"testing"
	"time"
)

func TestRestrictedActivityDoesNotAddressPublic(t *testing.T) {
	a := BuildCreateNoteActivity("https://peer.test/ap/users/alice", "post", "secret", time.Now(), "", "")
	applyAudience(a, []string{"https://other.test/ap/users/bob"})
	if len(a.To) != 1 || a.To[0] != "https://other.test/ap/users/bob" {
		t.Fatal("activity audience not restricted")
	}
	n, ok := a.Object.(Note)
	if !ok || len(n.To) != 1 || n.To[0] != a.To[0] || len(n.CC) > 0 {
		t.Fatal("nested Note leaked public audience")
	}
}
