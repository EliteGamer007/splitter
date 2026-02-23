package federation

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchRemoteNoteFromNoteObject(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/activity+json")
		_, _ = w.Write([]byte(`{
			"id":"https://remote.test/posts/abc",
			"type":"Note",
			"attributedTo":"https://remote.test/ap/users/alice",
			"content":"hello from remote",
			"inReplyTo":"https://remote.test/posts/parent-1",
			"published":"2026-02-23T10:11:12Z"
		}`))
	}))
	defer ts.Close()

	note, err := FetchRemoteNote(ts.URL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if note == nil {
		t.Fatalf("expected note, got nil")
	}
	if note.Deleted {
		t.Fatalf("expected non-deleted note")
	}
	if note.ID != "https://remote.test/posts/abc" {
		t.Fatalf("expected note id to be parsed")
	}
	if note.AttributedTo != "https://remote.test/ap/users/alice" {
		t.Fatalf("expected attributedTo to be parsed")
	}
	if note.InReplyTo != "https://remote.test/posts/parent-1" {
		t.Fatalf("expected inReplyTo to be parsed")
	}
}

func TestFetchRemoteNoteFromCreateActivity(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/activity+json")
		_, _ = w.Write([]byte(`{
			"id":"https://remote.test/activities/create-1",
			"type":"Create",
			"actor":"https://remote.test/ap/users/bob",
			"object":{
				"id":"https://remote.test/posts/reply-2",
				"type":"Note",
				"content":"reply payload",
				"inReplyTo":"https://remote.test/posts/parent-2"
			}
		}`))
	}))
	defer ts.Close()

	note, err := FetchRemoteNote(ts.URL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if note == nil {
		t.Fatalf("expected note, got nil")
	}
	if note.AttributedTo != "https://remote.test/ap/users/bob" {
		t.Fatalf("expected actor fallback for attributedTo")
	}
	if note.InReplyTo != "https://remote.test/posts/parent-2" {
		t.Fatalf("expected nested inReplyTo to be parsed")
	}
}

func TestFetchRemoteNoteHandlesMissingOrDeleted(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer ts.Close()

	note, err := FetchRemoteNote(ts.URL)
	if err != nil {
		t.Fatalf("expected no error for 404 deleted parent, got %v", err)
	}
	if note == nil || !note.Deleted {
		t.Fatalf("expected deleted marker note for 404")
	}
}
