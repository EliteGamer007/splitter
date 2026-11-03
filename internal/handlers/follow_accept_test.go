package handlers

import (
	"splitter/internal/config"
	"splitter/internal/federation"
	"splitter/internal/models"
	"testing"
	"time"
)

func TestAcceptedRemoteFollowTargetsOnlyRequester(t *testing.T) {
	cfg := &config.Config{}
	cfg.Federation.Enabled = true
	cfg.Federation.URL = "https://local.test"
	type delivered struct {
		activity *federation.Activity
		target   string
	}
	received := make(chan delivered, 2)
	h := NewFollowHandler(nil, nil, cfg)
	h.deliverAccepted = func(a *federation.Activity, target string) error { received <- delivered{a, target}; return nil }
	owner := &models.User{Username: "alice", DID: "did:key:alice"}
	remote := &models.User{DID: "https://peer.test/ap/users/bob"}
	h.notifyFollowAccepted(owner, remote)
	select {
	case call := <-received:
		if call.target != remote.DID || call.activity.Type != "Accept" || call.activity.Actor != "https://local.test/ap/users/alice" {
			t.Fatalf("wrong accept: %+v", call)
		}
		object, ok := call.activity.Object.(map[string]interface{})
		if !ok || object["type"] != "Follow" || object["actor"] != remote.DID || object["object"] != call.activity.Actor {
			t.Fatalf("incorrect original Follow: %+v", call.activity.Object)
		}
	case <-time.After(time.Second):
		t.Fatal("accepted remote request did not receive Accept")
	}
	cfg.Federation.Enabled = false
	h.notifyFollowAccepted(owner, remote)
	h.notifyFollowAccepted(owner, &models.User{DID: "did:key:local"})
	select {
	case <-received:
		t.Fatal("disabled or local follow triggered federation")
	case <-time.After(30 * time.Millisecond):
	}
}
