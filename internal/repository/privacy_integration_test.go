package repository_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"image"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"net/url"
	"os"
	"splitter/internal/config"
	"splitter/internal/db"
	"splitter/internal/handlers"
	"splitter/internal/models"
	"splitter/internal/repository"
	"splitter/internal/service"
	"strings"
	"sync"
	"testing"
	"time"
)

// Explicit local disposable database only. Never load .env or connect to a remote host.
func privacyDB(t *testing.T) context.Context {
	t.Helper()
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("TEST_DATABASE_URL not set; local disposable database required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if (parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1") || !strings.HasPrefix(strings.TrimPrefix(parsed.Path, "/"), "splitter_repair") {
		t.Fatal("privacy tests require local splitter_repair disposable database")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatal(err)
	}
	previous := db.DB
	db.DB = pool
	t.Cleanup(func() { db.DB = previous; pool.Close() })
	return ctx
}
func TestPrivacyRepositoryBoundaries(t *testing.T) {
	ctx := privacyDB(t)
	pool := db.GetDB()
	// Per-test schema keeps repeated runs and other repair suites isolated.
	schema := "privacy_" + strings.ReplaceAll(t.Name(), "/", "_")
	_, err := pool.Exec(ctx, "CREATE SCHEMA "+schema)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE") })
	poolCfg := pool.Config()
	poolCfg.ConnConfig.RuntimeParams["search_path"] = schema
	scoped, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		t.Fatal(err)
	}
	db.DB = scoped
	t.Cleanup(func() { scoped.Close(); db.DB = pool })
	_, err = scoped.Exec(ctx, `CREATE TABLE users(id uuid PRIMARY KEY, did text UNIQUE, username text, display_name text, avatar_url text,is_locked boolean DEFAULT false,is_suspended boolean DEFAULT false,message_privacy text DEFAULT 'everyone',email text DEFAULT '',password_hash text DEFAULT '',instance_domain text DEFAULT '',bio text DEFAULT '',public_key text DEFAULT '',encryption_public_key text DEFAULT '',role text DEFAULT 'user',moderation_requested boolean DEFAULT false,avatar_data bytea,avatar_media_type text,default_visibility text DEFAULT 'public',created_at timestamptz DEFAULT now(),updated_at timestamptz DEFAULT now());
 CREATE TABLE posts(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),author_did text,content text,visibility text,is_remote boolean DEFAULT false,original_post_uri text,in_reply_to_uri text,created_at timestamptz DEFAULT now(),updated_at timestamptz DEFAULT now(),expires_at timestamptz,deleted_at timestamptz,direct_reply_count int DEFAULT 0,total_reply_count int DEFAULT 0);
 CREATE TABLE follows(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),follower_did text,following_did text,status text,created_at timestamptz DEFAULT now());
 CREATE TABLE circle_members(owner_id uuid,member_id uuid,created_at timestamptz DEFAULT now());
 CREATE TABLE bookmarks(user_id uuid,post_id uuid,created_at timestamptz DEFAULT now(),UNIQUE(user_id,post_id));
 CREATE TABLE interactions(post_id uuid,actor_did text,interaction_type text,UNIQUE(post_id,actor_did,interaction_type));
 CREATE TABLE media(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),post_id uuid,media_data bytea,media_type text,media_url text DEFAULT '',created_at timestamptz DEFAULT now());
 CREATE TABLE replies(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),post_id uuid,parent_id uuid,author_did text,content text,depth int,likes_count int DEFAULT 0,direct_reply_count int DEFAULT 0,total_reply_count int DEFAULT 0,created_at timestamptz DEFAULT now(),updated_at timestamptz DEFAULT now(),deleted_at timestamptz);
 CREATE TABLE reply_interactions(reply_id uuid REFERENCES replies(id),actor_did text,created_at timestamptz DEFAULT now(),PRIMARY KEY(reply_id,actor_did));
 CREATE TABLE remote_post_recipients(post_id uuid,recipient_did text,PRIMARY KEY(post_id,recipient_did));
 CREATE TABLE stories(id uuid PRIMARY KEY,user_id uuid,media_url text,media_type text,media_data bytea,created_at timestamptz,expires_at timestamptz);
 CREATE TABLE story_views(story_id uuid REFERENCES stories(id)ON DELETE CASCADE,viewer_id uuid,PRIMARY KEY(story_id,viewer_id));
 CREATE TABLE message_threads(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),participant_a text,participant_b text,participant_a_id uuid,participant_b_id uuid,created_at timestamptz DEFAULT now(),updated_at timestamptz DEFAULT now());
 CREATE TABLE messages(id uuid PRIMARY KEY DEFAULT gen_random_uuid(),thread_id uuid,sender_did text,recipient_did text,sender_id uuid,recipient_id uuid,client_message_id text,content text,ciphertext text,encrypted_keys jsonb,is_read boolean DEFAULT false,created_at timestamptz DEFAULT now(),client_created_at timestamptz,delivered_at timestamptz,deleted_at timestamptz,edited_at timestamptz);
 CREATE UNIQUE INDEX messages_client_id ON messages(sender_id,client_message_id) WHERE client_message_id IS NOT NULL;
 INSERT INTO users(id,did,username,display_name,avatar_url,is_locked) VALUES ('00000000-0000-0000-0000-000000000001','owner','Owner','','',true),('00000000-0000-0000-0000-000000000002','viewer','Viewer','','',false),('00000000-0000-0000-0000-000000000003','stranger','Stranger','','',false);
 INSERT INTO posts(id,author_did,content,visibility) VALUES ('00000000-0000-0000-0000-000000000010','owner','secret','followers');
 INSERT INTO media(id,post_id,media_data,media_type) VALUES ('00000000-0000-0000-0000-000000000020','00000000-0000-0000-0000-000000000010','secret bytes','image/png');
 UPDATE media SET media_url='/uploads/posts/private.png';
 INSERT INTO posts(id,author_did,content,visibility,in_reply_to_uri)VALUES('00000000-0000-0000-0000-000000000011','stranger','public child','public','https://local.test/posts/00000000-0000-0000-0000-000000000010');`)
	if err != nil {
		t.Fatal(err)
	}
	const postID = "00000000-0000-0000-0000-000000000010"
	const mediaID = "00000000-0000-0000-0000-000000000020"
	posts := repository.NewPostRepository()
	follows := repository.NewFollowRepository()
	// Existing Create incorrectly auto-accepts a restricted follow.
	follow, err := follows.Create(ctx, "viewer", "owner")
	if err != nil {
		t.Fatal(err)
	}
	if follow.Status != "pending" {
		t.Errorf("locked follow status = %s, want pending", follow.Status)
	}
	// Remaining assertions exercise every alternate read/write surface.
	cfg := &config.Config{}
	cfg.Federation.URL = "https://local.test"
	postHandler := handlers.NewPostHandler(posts, repository.NewUserRepository(), cfg)
	mediaHandler := handlers.NewMediaHandler(posts)
	replyHandler := handlers.NewReplyHandler(nil, repository.NewUserRepository())
	interactionHandler := handlers.NewInteractionHandler(repository.NewInteractionRepository(), repository.NewUserRepository(), posts, nil)
	for _, surface := range []struct {
		name, param, value string
		handler            echo.HandlerFunc
	}{
		{"direct", "id", postID, postHandler.GetPost}, {"replies", "id", postID, replyHandler.GetReplies}, {"media", "id", mediaID, mediaHandler.GetMediaContent}, {"like", "id", postID, interactionHandler.LikePost},
	} {
		t.Run(surface.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(httptest.NewRequest("GET", "/", nil), rec)
			c.SetParamNames(surface.param)
			c.SetParamValues(surface.value)
			c.Set("did", "viewer")
			if err := surface.handler(c); err != nil {
				t.Fatal(err)
			}
			if rec.Code != 404 {
				t.Errorf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}

	if _, err = posts.GetByIDForViewer(ctx, postID, "viewer"); err == nil {
		t.Error("pending follower read private post")
	}
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(httptest.NewRequest("GET", "/", nil), rec)
	c.SetParamNames("id")
	c.SetParamValues("00000000-0000-0000-0000-000000000011")
	if err = postHandler.GetPost(c); err != nil {
		t.Fatal(err)
	}
	var child models.Post
	if err = json.Unmarshal(rec.Body.Bytes(), &child); err != nil {
		t.Fatal(err)
	}
	if child.ParentContext == nil || child.ParentContext.Status != "missing" || child.ParentContext.Post != nil {
		t.Errorf("public child exposed restricted parent: %s", rec.Body.String())
	}
	legacyRec := httptest.NewRecorder()
	legacyCtx := echo.New().NewContext(httptest.NewRequest("GET", "/uploads/posts/private.png", nil), legacyRec)
	legacyCtx.Set("did", "viewer")
	if err = mediaHandler.GetLegacyPostMedia(legacyCtx); err != nil {
		t.Fatal(err)
	}
	if legacyRec.Code != 404 {
		t.Errorf("legacy media leaked: %d", legacyRec.Code)
	}
	list, err := posts.GetByAuthorDIDForViewer(ctx, "owner", "viewer", 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Error("private profile list leaked")
	}
	if _, _, _, err = posts.GetMediaContentForViewer(ctx, mediaID, "viewer"); err == nil {
		t.Error("private media leaked")
	}
	if err = repository.NewInteractionRepository().CreateLike(ctx, postID, "viewer"); err == nil {
		t.Error("unauthorized like accepted")
	}
	if _, err = repository.NewReplyRepository().Create(ctx, "viewer", &models.ReplyCreate{PostID: postID, Content: "intrusion"}, 1); err == nil {
		t.Error("unauthorized reply accepted")
	}
	var mutations int
	_ = scoped.QueryRow(ctx, `SELECT (SELECT count(*) FROM interactions)+(SELECT count(*) FROM replies)`).Scan(&mutations)
	if mutations != 0 {
		t.Errorf("denied operations caused %d writes", mutations)
	}
	messages := repository.NewMessageRepository()
	const ownerID = "00000000-0000-0000-0000-000000000001"
	const viewerID = "00000000-0000-0000-0000-000000000002"
	_, _ = scoped.Exec(ctx, `UPDATE users SET message_privacy='followers' WHERE id=$1`, ownerID)
	if _, err = messages.GetOrCreateThread(ctx, viewerID, ownerID); err == nil {
		t.Error("pending follower started DM")
	}
	if _, err = messages.GetOrCreateThreadForInbound(ctx, viewerID, ownerID); err == nil {
		t.Error("inbound bypassed DM policy")
	}
	if err = follows.AcceptRequest(ctx, "owner", "viewer"); err != nil {
		t.Fatal(err)
	}
	thread, err := messages.GetOrCreateThread(ctx, viewerID, ownerID)
	if err != nil {
		t.Fatal("accepted follower DM denied", err)
	}
	msg, err := messages.SendMessage(ctx, thread.ID, viewerID, ownerID, "plaintext leak", "encrypted", "{}")
	if err != nil {
		t.Fatal(err)
	}
	if msg.Content != "" {
		t.Error("encrypted send retained plaintext")
	}
	syncMsg, created, err := messages.SendMessageWithClientMetadata(ctx, thread.ID, viewerID, ownerID, "queued-1", "queue plaintext", "queue ciphertext", `{"device":"wrapped"}`, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !created || syncMsg.Content != "" {
		t.Error("offline encrypted send retained plaintext or failed creation")
	}
	_, created, err = messages.SendMessageWithClientMetadata(ctx, thread.ID, viewerID, ownerID, "queued-1", "queue plaintext", "queue ciphertext", `{"device":"wrapped"}`, nil)
	if err != nil || created {
		t.Error("offline replay created another message", err)
	}
	msgHandler := handlers.NewMessageHandler(messages, repository.NewUserRepository(), cfg)
	forbiddenRec := httptest.NewRecorder()
	forbiddenCtx := echo.New().NewContext(httptest.NewRequest("GET", "/", nil), forbiddenRec)
	forbiddenCtx.Set("user_id", "00000000-0000-0000-0000-000000000003")
	forbiddenCtx.SetParamNames("threadId")
	forbiddenCtx.SetParamValues(thread.ID)
	if err = msgHandler.GetMessages(forbiddenCtx); err != nil {
		t.Fatal(err)
	}
	if forbiddenRec.Code != 403 {
		t.Errorf("nonparticipant thread read status=%d", forbiddenRec.Code)
	}
	pageRec := httptest.NewRecorder()
	pageCtx := echo.New().NewContext(httptest.NewRequest("GET", "/?limit=1&offset=1", nil), pageRec)
	pageCtx.Set("user_id", viewerID)
	if err = msgHandler.GetThreads(pageCtx); err != nil {
		t.Fatal(err)
	}
	var threadPage struct {
		Threads []models.MessageThread `json:"threads"`
	}
	if err = json.Unmarshal(pageRec.Body.Bytes(), &threadPage); err != nil {
		t.Fatal(err)
	}
	if len(threadPage.Threads) != 0 {
		t.Error("thread offset ignored")
	}
	var read int
	_ = scoped.QueryRow(ctx, `SELECT count(*) FROM messages WHERE is_read`).Scan(&read)
	if read != 0 {
		t.Error("denied thread read mutated read receipts")
	}

	before := 0
	_ = scoped.QueryRow(ctx, `SELECT count(*) FROM messages`).Scan(&before)
	if _, err := messages.SendMessage(ctx, thread.ID, viewerID, ownerID, strings.Repeat("x", (64<<10)+1), "", "{}"); err == nil {
		t.Error("oversized message text stored")
	}
	if _, err := messages.SendMessage(ctx, thread.ID, viewerID, ownerID, "", strings.Repeat("x", (128<<10)+1), "{}"); err == nil {
		t.Error("oversized ciphertext stored")
	}
	after := 0
	_ = scoped.QueryRow(ctx, `SELECT count(*) FROM messages`).Scan(&after)
	if before != after {
		t.Error("rejected payload caused message writes")
	}
	if err = messages.EditMessage(ctx, msg.ID, viewerID, "edited plaintext leak", "edited ciphertext"); err != nil {
		t.Fatal(err)
	}
	var stored string
	_ = scoped.QueryRow(ctx, `SELECT content FROM messages WHERE id=$1`, msg.ID).Scan(&stored)
	if stored != "" {
		t.Error("encrypted edit stored plaintext")
	}
	if err = messages.EditMessage(ctx, msg.ID, viewerID, "downgrade", ""); err == nil {
		t.Error("encrypted message downgraded to plaintext")
	}
	_, _ = scoped.Exec(ctx, `UPDATE users SET message_privacy='none' WHERE id=$1`, ownerID)
	if _, err = messages.SendMessage(ctx, thread.ID, viewerID, ownerID, "blocked", "", "{}"); err == nil {
		t.Error("existing thread bypassed changed DM policy")
	}
	_, _ = scoped.Exec(ctx, `UPDATE users SET message_privacy='everyone',is_suspended=true WHERE id=$1`, ownerID)
	if _, err = messages.GetOrCreateThread(ctx, viewerID, ownerID); err == nil {
		t.Error("suspended account accepted new DM")
	}

	if _, err = posts.GetByIDForViewer(ctx, postID, "viewer"); err != nil {
		t.Fatal("accepted follower denied", err)
	}
	profilePosts, err := posts.GetByAuthorDIDForViewer(ctx, "owner", "viewer", 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(profilePosts) != 1 || len(profilePosts[0].Media) != 1 || profilePosts[0].Media[0].ID != mediaID {
		t.Error("profile list dropped existing post media")
	}
	reply, err := repository.NewReplyRepository().Create(ctx, "viewer", &models.ReplyCreate{PostID: postID, Content: "permitted"}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err = repository.NewInteractionRepository().CreateLike(ctx, reply.ID, "viewer"); err != nil {
		t.Fatal(err)
	}
	if err = repository.NewInteractionRepository().CreateLike(ctx, reply.ID, "viewer"); err != nil {
		t.Fatal(err)
	}
	replies, err := repository.NewReplyRepository().GetByPostID(ctx, postID, "viewer")
	if err != nil {
		t.Fatal(err)
	}
	if len(replies) != 1 || replies[0].LikesCount != 1 || !replies[0].Liked {
		t.Errorf("reply like not idempotent/visible: %+v", replies)
	}

	if _, _, _, err = posts.GetMediaContentForViewer(ctx, mediaID, "viewer"); err != nil {
		t.Fatal("accepted media denied", err)
	}
	_, _ = scoped.Exec(ctx, `UPDATE posts SET visibility='circle' WHERE id=$1`, postID)
	if _, err = posts.GetByIDForViewer(ctx, postID, "viewer"); err == nil {
		t.Error("follower read circle post")
	}
	_, _ = scoped.Exec(ctx, `INSERT INTO circle_members VALUES('00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000002')`)
	if _, err = posts.GetByIDForViewer(ctx, postID, "viewer"); err != nil {
		t.Fatal("circle member denied", err)
	}
	if _, err = posts.GetByIDForViewer(ctx, postID, "stranger"); err == nil {
		t.Error("stranger read circle post")
	}
	_, _ = scoped.Exec(ctx, `INSERT INTO remote_post_recipients VALUES($1,'stranger')`, postID)
	if _, err = posts.GetByIDForViewer(ctx, postID, "stranger"); err == nil {
		t.Error("remote mapping overrode local audience")
	}
	_, _ = scoped.Exec(ctx, `UPDATE posts SET is_remote=true WHERE id=$1`, postID)
	if _, err = posts.GetByIDForViewer(ctx, postID, "stranger"); err != nil {
		t.Fatal("explicit remote recipient denied", err)
	}
	_, _ = scoped.Exec(ctx, `UPDATE posts SET is_remote=false WHERE id=$1`, postID)
	t.Run("reply_pagination", func(t *testing.T) {
		_, err := scoped.Exec(ctx, `INSERT INTO replies(post_id,author_did,content,depth)SELECT $1,'owner','page reply',1 FROM generate_series(1,60)`, postID)
		if err != nil {
			t.Fatal(err)
		}
		for _, page := range []struct {
			query string
			want  int
		}{{"?limit=1000", 50}, {"?limit=20&offset=40", 20}, {"?limit=20&offset=60", 1}} {
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(httptest.NewRequest("GET", "/"+page.query, nil), rec)
			c.Set("did", "viewer")
			c.SetParamNames("id")
			c.SetParamValues(postID)
			if err := replyHandler.GetReplies(c); err != nil {
				t.Fatal(err)
			}
			var result []models.Reply
			if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if len(result) != page.want {
				t.Errorf("%s returned %d, want %d", page.query, len(result), page.want)
			}
		}
	})
	_, _ = scoped.Exec(ctx, `UPDATE users SET is_suspended=false WHERE id=$1`, ownerID)
	t.Run("durable_media_budget", func(t *testing.T) {
		t.Setenv("MEDIA_ACCOUNT_BUDGET_BYTES", "130000")
		t.Setenv("MEDIA_TOTAL_BUDGET_BYTES", "150000")
		data := bytes.Repeat([]byte{42}, 64<<10)
		createdPost, err := posts.CreateWithMedia(ctx, "owner", &models.PostCreate{Content: "durable", Visibility: "public"}, "", "image/png", data)
		if err != nil {
			t.Fatal(err)
		}
		if len(createdPost.Media) != 1 {
			t.Fatal("upload discarded media bytes")
		}
		persisted, _, _, err := repository.NewPostRepository().GetMediaContentForViewer(ctx, createdPost.Media[0].ID, "")
		if err != nil || !bytes.Equal(persisted, data) {
			t.Fatal("media did not survive repository reload", err)
		}
		users := repository.NewUserRepository()
		if _, err = users.UpdateAvatar(ctx, ownerID, bytes.Repeat([]byte{1}, 32<<10), "image/png", ""); err != nil {
			t.Fatal(err)
		}
		stories := repository.NewStoryRepository()
		story := &models.Story{ID: uuid.New(), UserID: uuid.MustParse(ownerID), MediaURL: "/api/v1/stories/pending/media", MediaData: data, MediaType: "image/png", CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)}
		if err = stories.CreateStory(ctx, story); err == nil {
			t.Error("story upload bypassed combined account media quota")
		}
		var rejected int
		_ = scoped.QueryRow(ctx, `SELECT count(*) FROM stories WHERE id=$1`, story.ID).Scan(&rejected)
		if rejected != 0 {
			t.Error("overquota story persisted")
		}
		var wg sync.WaitGroup
		results := make(chan error, 2)
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := posts.CreateWithMedia(ctx, "viewer", &models.PostCreate{Content: "concurrent", Visibility: "public"}, "", "image/png", bytes.Repeat([]byte{2}, 32<<10))
				results <- err
			}()
		}
		wg.Wait()
		close(results)
		successes := 0
		for result := range results {
			if result == nil {
				successes++
			}
		}
		if successes != 1 {
			t.Errorf("concurrent successfuluploads=%d want1", successes)
		}
		if _, err = users.UpdateAvatar(ctx, ownerID, bytes.Repeat([]byte{3}, 16<<10), "image/png", ""); err != nil {
			t.Fatal("avatar replacement did not reclaim its own usage", err)
		}
		story.MediaData = bytes.Repeat([]byte{4}, 16<<10)
		if err = stories.CreateStory(ctx, story); err != nil {
			t.Fatal(err)
		}
		viewedCtx := context.WithValue(ctx, "viewer_id", uuid.MustParse(viewerID))
		loaded, _, err := stories.GetStoryMedia(viewedCtx, story.ID.String())
		if err != nil || !bytes.Equal(loaded, story.MediaData) {
			t.Fatal("story bytes lost", err)
		}
		if err = stories.RecordStoryView(ctx, story.ID, uuid.MustParse(viewerID)); err != nil {
			t.Fatal(err)
		}
		viewedCtx = context.WithValue(ctx, "viewer_id", uuid.MustParse(viewerID))
		active, err := stories.GetActiveStories(viewedCtx)
		if err != nil || len(active) != 1 || !active[0].Seen {
			t.Fatal("story readstate lost", err)
		}
		_, _ = scoped.Exec(ctx, `UPDATE stories SET expires_at=now()-interval '1 minute' WHERE id=$1`, story.ID)
		if _, _, err = stories.GetStoryMedia(ctx, story.ID.String()); err == nil {
			t.Error("expired story bytes remain readable")
		}
		if err = stories.RecordStoryView(ctx, story.ID, uuid.MustParse(viewerID)); err == nil {
			t.Error("expired story view recorded")
		}
	})
	t.Run("encryption_key_first_registration", func(t *testing.T) {
		users := repository.NewUserRepository()
		key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		other, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		der, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
		otherDER, _ := x509.MarshalPKIXPublicKey(&other.PublicKey)
		first := base64.StdEncoding.EncodeToString(der)
		replacement := base64.StdEncoding.EncodeToString(otherDER)
		if err := users.UpdateEncryptionKey(ctx, viewerID, first); err != nil {
			t.Fatal(err)
		}
		if err := users.UpdateEncryptionKey(ctx, viewerID, first); err != nil {
			t.Fatal("idempotent registration rejected", err)
		}
		if err := users.UpdateEncryptionKey(ctx, viewerID, replacement); err == nil {
			t.Error("fresh device overwrote existing account encryption key")
		}
		var stored string
		_ = scoped.QueryRow(ctx, `SELECT encryption_public_key FROM users WHERE id=$1`, viewerID).Scan(&stored)
		if stored != first {
			t.Error("account encryption key changed after rejected replacement")
		}
		if err := users.UpdateEncryptionKey(ctx, ownerID, "invalid key"); err == nil {
			t.Error("invalid encryption public key accepted")
		}
	})
	t.Run("media_http_roundtrip", func(t *testing.T) {
		var original bytes.Buffer
		if err := png.Encode(&original, image.NewRGBA(image.Rect(0, 0, 24, 24))); err != nil {
			t.Fatal(err)
		}
		padded := append(original.Bytes(), make([]byte, 2<<20)...)
		callUpload := func(field string, handler echo.HandlerFunc) *httptest.ResponseRecorder {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile(field, "large-photo.png")
			if err != nil {
				t.Fatal(err)
			}
			_, _ = part.Write(padded)
			_ = writer.WriteField("content", "photo")
			_ = writer.WriteField("visibility", "circle")
			_ = writer.Close()
			req := httptest.NewRequest("POST", "/", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)
			c.Set("did", "viewer")
			c.Set("user_id", viewerID)
			if err = handler(c); err != nil {
				t.Fatal(err)
			}
			if req.MultipartForm != nil {
				_ = req.MultipartForm.RemoveAll()
			}
			return rec
		}
		rec := callUpload("file", postHandler.CreatePost)
		if rec.Code != 201 {
			t.Fatalf("post upload %d %s", rec.Code, rec.Body.String())
		}
		var post models.Post
		if err := json.Unmarshal(rec.Body.Bytes(), &post); err != nil {
			t.Fatal(err)
		}
		if len(post.Media) != 1 || post.Media[0].MediaURL != "/api/v1/media/"+post.Media[0].ID {
			t.Fatalf("missing durable canonical media: %s", rec.Body.String())
		}
		data, _, _, err := posts.GetMediaContentForViewer(ctx, post.Media[0].ID, "viewer")
		if err != nil || len(data) == 0 || len(data) > 256<<10 {
			t.Fatal("post media bytes missing or oversized", err)
		}
		if _, _, _, err = posts.GetMediaContentForViewer(ctx, post.Media[0].ID, "owner"); err == nil {
			t.Error("private upload leaked through media")
		}
		userHandler := handlers.NewUserHandler(repository.NewUserRepository(), cfg)
		rec = callUpload("avatar", userHandler.UploadAvatar)
		if rec.Code != 200 {
			t.Fatalf("avatar upload %d %s", rec.Code, rec.Body.String())
		}
		data, _, _, err = repository.NewUserRepository().GetAvatarContentByUserID(ctx, viewerID)
		if err != nil || len(data) == 0 || len(data) > 128<<10 {
			t.Fatal("avatar bytes missing or oversized", err)
		}
		storyHandler := handlers.NewStoryHandler(service.NewStoryService(repository.NewStoryRepository()))
		rec = callUpload("file", storyHandler.CreateStory)
		if rec.Code != 201 {
			t.Fatalf("story upload %d %s", rec.Code, rec.Body.String())
		}
		var storyID string
		_ = scoped.QueryRow(ctx, `SELECT id FROM stories WHERE user_id=$1 ORDER BY created_at DESC LIMIT 1`, viewerID).Scan(&storyID)
		data, _, err = repository.NewStoryRepository().GetStoryMedia(ctx, storyID)
		if err != nil || len(data) == 0 || len(data) > 256<<10 {
			t.Fatal("story bytes missing or oversized", err)
		}
	})
	t.Run("all_list_pages_accessible", func(t *testing.T) {
		if _, err := follows.Create(ctx, "stranger", "owner"); err != nil {
			t.Fatal(err)
		}
		var publicPost string
		_ = scoped.QueryRow(ctx, `SELECT id FROM posts WHERE content='durable' LIMIT 1`).Scan(&publicPost)
		if err := repository.NewInteractionRepository().CreateBookmark(ctx, viewerID, publicPost); err != nil {
			t.Fatal(err)
		}
		userHandler := handlers.NewUserHandler(repository.NewUserRepository(), cfg)
		followHandler := handlers.NewFollowHandler(follows, repository.NewUserRepository())
		storyHandler := handlers.NewStoryHandler(service.NewStoryService(repository.NewStoryRepository()))
		for _, surface := range []struct {
			name    string
			handler echo.HandlerFunc
			did, id string
		}{
			{"circle", userHandler.GetCircle, "owner", ownerID}, {"requests", followHandler.GetFollowRequests, "owner", ownerID}, {"bookmarks", interactionHandler.GetBookmarks, "viewer", viewerID}, {"stories", storyHandler.GetStories, "viewer", viewerID},
		} {
			t.Run(surface.name, func(t *testing.T) {
				rec := httptest.NewRecorder()
				c := echo.New().NewContext(httptest.NewRequest("GET", "/?limit=1&offset=1", nil), rec)
				c.Set("did", surface.did)
				c.Set("user_id", surface.id)
				if err := surface.handler(c); err != nil {
					t.Fatal(err)
				}
				if rec.Code != 200 {
					t.Fatalf("status=%d %s", rec.Code, rec.Body.String())
				}
				var page []interface{}
				if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
					t.Fatal(err)
				}
				if len(page) != 0 {
					t.Errorf("offset ignored: %s", rec.Body.String())
				}
			})
		}
		threads, err := messages.GetUserThreadsPage(ctx, ownerID, 30, 0)
		if err != nil || len(threads) != 1 || threads[0].LastMessage == nil || threads[0].LastMessage.Content != "" || threads[0].OtherUser.ID != viewerID {
			t.Fatalf("thread hydration failed %+v %v", threads, err)
		}
	})
	t.Run("remote_username_cannot_shadow_local", func(t *testing.T) {
		_, err := scoped.Exec(ctx, `INSERT INTO users(id,did,username,display_name,instance_domain,password_hash)VALUES('00000000-0000-0000-0000-000000000004','https://peer.test/users/shadow','shadow','','peer.test',''),('00000000-0000-0000-0000-000000000005','did:key:shadow','shadow','','local.test','local-password-hash')`)
		if err != nil {
			t.Fatal(err)
		}
		found, hash, err := repository.NewUserRepository().GetByUsername(ctx, "shadow")
		if err != nil {
			t.Fatal(err)
		}
		if found.DID != "did:key:shadow" || hash != "local-password-hash" {
			t.Errorf("remote actor shadowed local login/discovery: %s", found.DID)
		}
	})
	_, _ = scoped.Exec(ctx, `UPDATE posts SET expires_at=now()-interval '1 minute' WHERE id=$1`, postID)
	if _, err = posts.GetByIDForViewer(ctx, postID, "owner"); err == nil {
		t.Error("owner read expired post")
	}
}
