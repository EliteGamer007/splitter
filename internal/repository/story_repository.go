package repository

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"splitter/internal/db"
	"splitter/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type StoryRepository struct{}

func NewStoryRepository() *StoryRepository {
	return &StoryRepository{}
}

func (r *StoryRepository) CreateStory(ctx context.Context, story *models.Story) error {
	if len(story.MediaData) > db.MaxStoredMedia {
		return fmt.Errorf("stored image exceeds 256 KiB")
	}
	r.maybeCleanup(ctx)
	tx, err := db.GetDB().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = db.EnforceMediaBudget(ctx, tx, story.UserID.String(), int64(len(story.MediaData))); err != nil {
		return err
	}
	if err = db.CheckGrowthBudget(ctx, tx, int64(len(story.MediaData)+1024)); err != nil {
		return err
	}
	if len(story.MediaData) > 0 {
		story.MediaURL = "/api/v1/stories/" + story.ID.String() + "/media"
	}
	result, err := tx.Exec(ctx, `INSERT INTO stories(id,user_id,media_url,media_type,media_data,created_at,expires_at) SELECT $1,$2,$3,$4,$5,$6,$7 FROM users WHERE id=$2 AND NOT is_suspended`, story.ID, story.UserID, story.MediaURL, story.MediaType, story.MediaData, story.CreatedAt, story.ExpiresAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("account unavailable")
	}
	return tx.Commit(ctx)
}
func storyViewer(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value("viewer_id").(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}
func storyVisibilitySQL(viewer string) string {
	return `s.expires_at>NOW() AND NOT u.is_suspended AND (NOT u.is_locked OR s.user_id=` + viewer + `::uuid OR EXISTS(SELECT 1 FROM follows f JOIN users viewer ON viewer.did=f.follower_did WHERE viewer.id=` + viewer + `::uuid AND NOT viewer.is_suspended AND f.following_did=u.did AND f.status='accepted'))`
}

var storyCleanupState struct {
	sync.Mutex
	last time.Time
	pool interface{}
}

func (r *StoryRepository) maybeCleanup(ctx context.Context) {
	storyCleanupState.Lock()
	if storyCleanupState.pool == db.GetDB() && time.Since(storyCleanupState.last) < 5*time.Minute {
		storyCleanupState.Unlock()
		return
	}
	storyCleanupState.pool = db.GetDB()
	storyCleanupState.last = time.Now()
	storyCleanupState.Unlock()
	_ = r.DeleteExpiredStories(ctx)
}

func (r *StoryRepository) GetActiveStories(ctx context.Context) ([]models.Story, error) {
	return r.GetActiveStoriesPage(ctx, 20, 0)
}
func (r *StoryRepository) GetActiveStoriesPage(ctx context.Context, limit, offset int) ([]models.Story, error) {
	limit, offset = boundedPage(limit, offset, 20)

	r.maybeCleanup(ctx)
	viewerID := storyViewer(ctx)
	ok := viewerID != uuid.Nil
	var query string
	var rows pgx.Rows
	var err error

	if ok && viewerID != uuid.Nil {
		query = `
			SELECT s.id, s.user_id, s.media_url, s.created_at, s.expires_at, 
			       EXISTS (
			           SELECT 1 FROM story_views sv 
			           WHERE sv.story_id = s.id AND sv.viewer_id = $1
			       ) AS seen,
			       u.id AS author_id, u.username, COALESCE(u.avatar_url, '') AS avatar
			FROM stories s
			JOIN users u ON u.id = s.user_id
			WHERE ` + storyVisibilitySQL("$1") + `
			ORDER BY s.created_at DESC,s.id DESC LIMIT $2 OFFSET $3
		`
		rows, err = db.GetDB().Query(ctx, query, viewerID, limit, offset)
	} else {
		query = `
			SELECT s.id, s.user_id, s.media_url, s.created_at, s.expires_at, false as seen,
			       u.id AS author_id, u.username, COALESCE(u.avatar_url, '') AS avatar
			FROM stories s
			JOIN users u ON u.id = s.user_id
			WHERE ` + storyVisibilitySQL("$1") + `
			ORDER BY s.created_at DESC,s.id DESC LIMIT $2 OFFSET $3
		`
		rows, err = db.GetDB().Query(ctx, query, uuid.Nil, limit, offset)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stories := make([]models.Story, 0)
	for rows.Next() {
		var s models.Story
		if err := rows.Scan(&s.ID, &s.UserID, &s.MediaURL, &s.CreatedAt, &s.ExpiresAt, &s.Seen,
			&s.Author.ID, &s.Author.Username, &s.Author.Avatar); err != nil {
			return nil, err
		}
		stories = append(stories, s)
	}
	return stories, nil
}

func (r *StoryRepository) DeleteExpiredStories(ctx context.Context) error {
	query := `DELETE FROM stories WHERE id IN(SELECT id FROM stories WHERE expires_at<=NOW() ORDER BY expires_at LIMIT 20 FOR UPDATE SKIP LOCKED)`
	_, err := db.GetDB().Exec(ctx, query)
	return err
}

func (r *StoryRepository) DeleteStory(ctx context.Context, storyID uuid.UUID, userID uuid.UUID) error {
	query := `
		DELETE FROM stories
		WHERE id = $1 AND user_id = $2
	`
	result, err := db.GetDB().Exec(ctx, query, storyID, userID)
	if err == nil && result.RowsAffected() == 0 {
		return fmt.Errorf("story not found")
	}
	return err
}

func (r *StoryRepository) RecordStoryView(ctx context.Context, storyID, viewerID uuid.UUID) error {
	var visible bool
	err := db.GetDB().QueryRow(ctx, `WITH visible AS(SELECT s.id FROM stories s JOIN users u ON u.id=s.user_id WHERE s.id=$1 AND `+storyVisibilitySQL("$2")+`),inserted AS(INSERT INTO story_views(story_id,viewer_id)SELECT id,$2 FROM visible ON CONFLICT DO NOTHING RETURNING story_id) SELECT EXISTS(SELECT 1 FROM visible)`, storyID, viewerID).Scan(&visible)
	if err != nil {
		return err
	}
	if !visible {
		return fmt.Errorf("story not found")
	}
	return nil
}

func (r *StoryRepository) GetStoryFeed(ctx context.Context) ([]models.StoryUser, error) {
	return r.GetStoryFeedPage(ctx, 20, 0)
}
func (r *StoryRepository) GetStoryFeedPage(ctx context.Context, limit, offset int) ([]models.StoryUser, error) {
	limit, offset = boundedPage(limit, offset, 20)

	r.maybeCleanup(ctx)
	viewerID := storyViewer(ctx)
	ok := viewerID != uuid.Nil
	var query string
	var rows pgx.Rows
	var err error

	if ok && viewerID != uuid.Nil {
		query = `
			SELECT s.id, s.user_id, s.media_url, s.created_at, s.expires_at, 
			       EXISTS (
			           SELECT 1 FROM story_views sv 
			           WHERE sv.story_id = s.id AND sv.viewer_id = $1
			       ) AS seen,
			       u.username, COALESCE(u.avatar_url, '') AS avatar
			FROM stories s
			JOIN users u ON u.id = s.user_id
			WHERE ` + storyVisibilitySQL("$1") + `
			ORDER BY s.created_at DESC,s.id DESC LIMIT $2 OFFSET $3
		`
		rows, err = db.GetDB().Query(ctx, query, viewerID, limit, offset)
	} else {
		query = `
			SELECT s.id, s.user_id, s.media_url, s.created_at, s.expires_at, false as seen,
			       u.username, COALESCE(u.avatar_url, '') AS avatar
			FROM stories s
			JOIN users u ON u.id = s.user_id
			WHERE ` + storyVisibilitySQL("$1") + `
			ORDER BY s.created_at DESC,s.id DESC LIMIT $2 OFFSET $3
		`
		rows, err = db.GetDB().Query(ctx, query, uuid.Nil, limit, offset)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userMap := make(map[uuid.UUID]*models.StoryUser)
	var userOrder []uuid.UUID

	for rows.Next() {
		var s models.Story
		var username, avatar string
		if err := rows.Scan(&s.ID, &s.UserID, &s.MediaURL, &s.CreatedAt, &s.ExpiresAt, &s.Seen,
			&username, &avatar); err != nil {
			return nil, err
		}

		s.Author = models.Author{
			ID:       s.UserID,
			Username: username,
			Avatar:   avatar,
		}

		if _, exists := userMap[s.UserID]; !exists {
			userMap[s.UserID] = &models.StoryUser{
				UserID:   s.UserID,
				Username: username,
				Avatar:   avatar,
				Stories:  []models.Story{},
			}
			userOrder = append(userOrder, s.UserID)
		}
		userMap[s.UserID].Stories = append(userMap[s.UserID].Stories, s)
	}

	feed := make([]models.StoryUser, 0)
	for _, uid := range userOrder {
		feed = append(feed, *userMap[uid])
	}

	return feed, nil
}

func (r *StoryRepository) GetStoryMedia(ctx context.Context, id string) ([]byte, string, error) {
	var legacyURL, kind string
	var data []byte
	err := db.GetDB().QueryRow(ctx, `SELECT s.media_url,COALESCE(s.media_data,''::bytea),COALESCE(s.media_type,'') FROM stories s JOIN users u ON u.id=s.user_id WHERE s.id=$1 AND `+storyVisibilitySQL("$2"), id, storyViewer(ctx)).Scan(&legacyURL, &data, &kind)
	if err != nil {
		return nil, "", fmt.Errorf("story not found")
	}
	if len(data) > 0 {
		if kind == "" {
			kind = http.DetectContentType(data)
		}
		return data, kind, nil
	}
	if !strings.HasPrefix(legacyURL, "/uploads/") && !strings.HasPrefix(legacyURL, "/media/") {
		return nil, "", fmt.Errorf("story media unavailable")
	}
	if path.Clean(legacyURL) != legacyURL || strings.Contains(legacyURL, "\\") {
		return nil, "", fmt.Errorf("story media unavailable")
	}
	filename := strings.TrimPrefix(legacyURL, "/")
	if strings.HasPrefix(filename, "media/") {
		filename = "uploads/" + strings.TrimPrefix(filename, "media/")
	}
	root, err := filepath.Abs("uploads")
	if err != nil {
		return nil, "", err
	}
	target, err := filepath.Abs(filepath.FromSlash(filename))
	if err != nil {
		return nil, "", err
	}
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
		return nil, "", fmt.Errorf("story media unavailable")
	}
	data, err = os.ReadFile(target)
	if err != nil {
		return nil, "", fmt.Errorf("story media unavailable")
	}
	return data, http.DetectContentType(data), nil
}

// GetLegacyMedia resolves a saved URL before applying expiry and audience checks.
func (r *StoryRepository) GetLegacyMedia(ctx context.Context, legacyURL string) ([]byte, string, error) {
	var id string
	err := db.GetDB().QueryRow(ctx, `SELECT s.id FROM stories s JOIN users u ON u.id=s.user_id WHERE s.media_url=$1 AND `+storyVisibilitySQL("$2")+` LIMIT 1`, legacyURL, storyViewer(ctx)).Scan(&id)
	if err != nil {
		return nil, "", fmt.Errorf("story not found")
	}
	return r.GetStoryMedia(ctx, id)
}
