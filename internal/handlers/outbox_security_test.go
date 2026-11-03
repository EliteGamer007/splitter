package handlers

import (
	"context"
	"encoding/json"
	"github.com/labstack/echo/v4"
	"net/http/httptest"
	"splitter/internal/config"
	"splitter/internal/repository"
	"strings"
	"testing"
)

func TestPublicOutboxOnlyContainsCurrentPublicNotesFromRequestedActor(t *testing.T) {
	pool := inboxTestDB(t)
	ctx := context.Background()
	_, err := pool.Exec(ctx, `INSERT INTO users(username,email,instance_domain,did,display_name,public_key) VALUES('alice','alice@outbox.test','test.local','did:outbox-alice','Alice',''),('bob','bob@outbox.test','test.local','did:outbox-bob','Bob','');
 INSERT INTO posts(id,author_did,content,visibility) VALUES('20000000-0000-4000-8000-000000000001','did:outbox-alice','current public content','public'),('20000000-0000-4000-8000-000000000002','did:outbox-alice','private content','followers'),('20000000-0000-4000-8000-000000000003','did:outbox-bob','other actor content','public');`)
	if err != nil {
		t.Fatal(err)
	}
	for _, payload := range []string{
		`{"type":"Create","actor":"https://local.test/ap/users/alice","to":["https://local.test/ap/users/bob"],"object":{"type":"Note","content":"private DM plaintext","ciphertext":"private ciphertext","encrypted_keys":{"bob":"private envelope"}}}`,
		`{"type":"Create","actor":"https://local.test/ap/users/bob","to":["https://www.w3.org/ns/activitystreams#Public"],"object":{"type":"Note","content":"other actor queued content"}}`,
		`{"type":"Create","actor":"https://local.test/ap/users/alice","to":["https://www.w3.org/ns/activitystreams#Public"],"object":{"id":"https://local.test/posts/20000000-0000-4000-8000-000000000001","type":"Note","content":"old public content"}}`,
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO outbox_activities(activity_type,payload,target_inbox,status) VALUES('Create',$1,'https://peer.test/inbox','sent')`, payload); err != nil {
			t.Fatal(err)
		}
	}
	h := NewOutboxHandler(repository.NewUserRepository(), &config.Config{Federation: config.FederationConfig{URL: "https://local.test"}})
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest("GET", "/ap/users/alice/outbox", nil), rec)
	c.SetParamNames("username")
	c.SetParamValues("alice")
	if err := h.GetOutbox(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 200 {
		t.Fatalf("outbox: %d %s", rec.Code, rec.Body.String())
	}
	for _, secret := range []string{"private DM", "private ciphertext", "private envelope", "other actor", "private content", "old public content"} {
		if strings.Contains(rec.Body.String(), secret) {
			t.Errorf("public outbox leaked %q", secret)
		}
	}
	var result struct {
		ID    string            `json:"id"`
		Items []json.RawMessage `json:"orderedItems"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.ID != "https://local.test/ap/users/alice/outbox" || len(result.Items) != 1 || !strings.Contains(string(result.Items[0]), "current public content") {
		t.Fatalf("outbox must expose one current public actor note: %s", rec.Body.String())
	}
}
