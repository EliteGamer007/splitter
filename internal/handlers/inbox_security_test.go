package handlers

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"splitter/internal/config"
	"splitter/internal/db"
	"splitter/internal/repository"
	"splitter/tests/testdb"
	"strings"
	"testing"
	"time"
)

func inboxTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("explicit disposable database required")
	}
	if _, err := testdb.ParseLocalURL(raw); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	base, err := pgxpool.New(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("inbox_test_%d", time.Now().UnixNano())
	if _, err = base.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err = db.ApplyMigrations(ctx, pool, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	old := db.DB
	db.DB = pool
	t.Cleanup(func() { db.DB = old; pool.Close(); base.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); base.Close() })
	return pool
}

func TestDisabledInboxRejectsWithoutSideEffects(t *testing.T) {
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodPost, "/ap/shared-inbox", strings.NewReader(`{}`)), rec)
	h := &InboxHandler{cfg: &config.Config{}}
	if err := h.Handle(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("disabled inbox status=%d", rec.Code)
	}
}

func TestInboxDeleteRequiresObjectOwnership(t *testing.T) {
	p := inboxTestDB(t)
	ctx := context.Background()
	id := "10000000-0000-4000-8000-000000000001"
	victim := "https://victim.test/ap/users/alice"
	uri := "https://victim.test/posts/secret"
	_, err := p.Exec(ctx, "INSERT INTO posts(id,author_did,content,visibility,is_remote,original_post_uri) VALUES($1,$2,'secret','public',true,$3)", id, victim, uri)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(httptest.NewRequest(http.MethodPost, "/inbox", nil), rec)
	h := &InboxHandler{}
	h.handleDelete(c, map[string]interface{}{"actor": "https://attacker.test/ap/users/eve", "object": uri})
	var intact bool
	if err = p.QueryRow(ctx, "SELECT deleted_at IS NULL FROM posts WHERE id=$1", id).Scan(&intact); err != nil || !intact {
		t.Fatalf("other actor's post was deleted: %v", err)
	}
}

func TestInboxLikeCannotAccessPrivatePost(t *testing.T) {
	p := inboxTestDB(t)
	ctx := context.Background()
	id := "10000000-0000-4000-8000-000000000002"
	_, err := p.Exec(ctx, "INSERT INTO posts(id,author_did,content,visibility) VALUES($1,'did:local:owner','secret','circle')", id)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(httptest.NewRequest(http.MethodPost, "/inbox", nil), rec)
	h := &InboxHandler{}
	h.handleLike(c, map[string]interface{}{"actor": "https://attacker.test/ap/users/eve", "object": "https://local.test/posts/" + id})
	var count int
	if err = p.QueryRow(ctx, "SELECT count(*) FROM interactions WHERE post_id=$1", id).Scan(&count); err != nil || count != 0 {
		t.Fatalf("private post accepted unauthorized like: %d, %v", count, err)
	}
}

func TestUnaddressedNoteCannotBecomePublic(t *testing.T) {
	p := inboxTestDB(t)
	ctx := context.Background()
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(httptest.NewRequest(http.MethodPost, "/inbox", nil), rec)
	h := &InboxHandler{cfg: &config.Config{}}
	h.handleCreate(c, map[string]interface{}{"actor": "https://remote.test/ap/users/alice", "object": map[string]interface{}{"type": "Note", "id": "https://remote.test/posts/private", "content": "private sentinel"}})
	var count int
	if err := p.QueryRow(ctx, "SELECT count(*) FROM posts WHERE original_post_uri='https://remote.test/posts/private'").Scan(&count); err != nil || count != 0 {
		t.Fatalf("unaddressed Note became public: %d %v", count, err)
	}
}

func TestActivityPubPostIsDiscoverableAndPrivateContentIsProtected(t *testing.T) {
	p := inboxTestDB(t)
	ctx := context.Background()
	_, err := p.Exec(ctx, `INSERT INTO users(username,did,instance_domain,display_name) VALUES('apauthor','did:local:author','local.test','AP Author'); INSERT INTO posts(id,author_did,content,visibility) VALUES('20000000-0000-4000-8000-000000000001','did:local:author','public note','public'),('20000000-0000-4000-8000-000000000002','did:local:author','private note','circle')`)
	if err != nil {
		t.Fatal(err)
	}
	h := NewPostHandler(repository.NewPostRepository(), repository.NewUserRepository(), &config.Config{Federation: config.FederationConfig{URL: "https://local.test"}})
	for _, tt := range []struct {
		id     string
		status int
	}{{"20000000-0000-4000-8000-000000000001", 200}, {"20000000-0000-4000-8000-000000000002", 404}} {
		rec := httptest.NewRecorder()
		c := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/posts/"+tt.id, nil), rec)
		c.SetParamNames("id")
		c.SetParamValues(tt.id)
		if err = h.GetActivityPubPost(c); err != nil {
			t.Fatal(err)
		}
		if rec.Code != tt.status {
			t.Fatalf("AP Note status=%d want%d", rec.Code, tt.status)
		}
		if tt.status == 200 && !strings.Contains(rec.Body.String(), `"type":"Note"`) {
			t.Fatal("canonical published URI did not serve Note")
		}
	}
}
