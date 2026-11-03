package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"splitter/internal/config"
	"strings"
	"testing"
)

func TestFollowApprovalRoutesRequireAuthentication(t *testing.T) {
	s := NewServer(&config.Config{JWT: config.JWTConfig{Secret: "test-route-secret"}})
	for _, r := range []struct{ method, path string }{{"GET", "/api/v1/users/me/follow-requests"}, {"POST", "/api/v1/users/me/follow-requests/person/accept"}, {"DELETE", "/api/v1/users/me/follow-requests/person"}, {"GET", "/api/v1/users/person/follow-status"}} {
		rec := httptest.NewRecorder()
		s.Echo().ServeHTTP(rec, httptest.NewRequest(r.method, r.path, nil))
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s returned%d, want401", r.method, r.path, rec.Code)
		}
	}
}

func TestLegacyUploadsCannotBypassAuthorization(t *testing.T) {
	root := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	os.MkdirAll(filepath.Join("uploads", "posts"), 0700)
	os.WriteFile(filepath.Join("uploads", "posts", "private.txt"), []byte("private sentinel"), 0600)
	s := NewServer(&config.Config{JWT: config.JWTConfig{Secret: "test-route-secret"}})
	rec := httptest.NewRecorder()
	s.Echo().ServeHTTP(rec, httptest.NewRequest("GET", "/uploads/posts/private.txt", nil))
	if rec.Code == 200 || strings.Contains(rec.Body.String(), "private sentinel") {
		t.Fatal("legacy static upload bypasses post visibility")
	}
}
