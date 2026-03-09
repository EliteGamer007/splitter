package repository

import (
	"context"

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
	query := `
		INSERT INTO stories (id, user_id, media_url, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := db.GetDB().Exec(ctx, query, story.ID, story.UserID, story.MediaURL, story.CreatedAt, story.ExpiresAt)
	return err
}

func (r *StoryRepository) GetActiveStories(ctx context.Context) ([]models.Story, error) {
	viewerID, ok := ctx.Value("viewer_id").(uuid.UUID)
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
			WHERE s.expires_at > NOW()
			ORDER BY s.created_at DESC
		`
		rows, err = db.GetDB().Query(ctx, query, viewerID)
	} else {
		query = `
			SELECT s.id, s.user_id, s.media_url, s.created_at, s.expires_at, false as seen,
			       u.id AS author_id, u.username, COALESCE(u.avatar_url, '') AS avatar
			FROM stories s
			JOIN users u ON u.id = s.user_id
			WHERE s.expires_at > NOW()
			ORDER BY s.created_at DESC
		`
		rows, err = db.GetDB().Query(ctx, query)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stories []models.Story
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
	query := `DELETE FROM stories WHERE expires_at <= NOW()`
	_, err := db.GetDB().Exec(ctx, query)
	return err
}

func (r *StoryRepository) DeleteStory(ctx context.Context, storyID uuid.UUID, userID uuid.UUID) error {
	query := `
		DELETE FROM stories
		WHERE id = $1 AND user_id = $2
	`
	_, err := db.GetDB().Exec(ctx, query, storyID, userID)
	return err
}

func (r *StoryRepository) RecordStoryView(ctx context.Context, storyID uuid.UUID, viewerID uuid.UUID) error {
	query := `
		INSERT INTO story_views (story_id, viewer_id)
		VALUES ($1, $2)
		ON CONFLICT (story_id, viewer_id) DO NOTHING
	`
	_, err := db.GetDB().Exec(ctx, query, storyID, viewerID)
	return err
}
