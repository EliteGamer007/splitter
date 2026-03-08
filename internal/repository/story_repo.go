package repository

import (
	"context"
	"fmt"

	"splitter/internal/db"
	"splitter/internal/models"

	"github.com/jackc/pgx/v5"
)

// StoryRepository handles database operations for stories
type StoryRepository struct{}

// NewStoryRepository creates a new StoryRepository
func NewStoryRepository() *StoryRepository {
	return &StoryRepository{}
}

// Create inserts a new story with binary media data
func (r *StoryRepository) Create(ctx context.Context, authorDID string, mediaData []byte, mediaType string) (*models.Story, error) {
	query := `
		INSERT INTO stories (author_did, media_data, media_type)
		VALUES ($1, $2, $3)
		RETURNING id, author_did, media_type, created_at, expires_at
	`

	var story models.Story
	err := db.GetDB().QueryRow(ctx, query, authorDID, mediaData, mediaType).Scan(
		&story.ID,
		&story.AuthorDID,
		&story.MediaType,
		&story.CreatedAt,
		&story.ExpiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create story: %w", err)
	}

	// Set the media URL for API access
	story.MediaURL = fmt.Sprintf("/api/v1/stories/%s/media", story.ID)

	return &story, nil
}

// GetFeedForUser returns stories grouped by author, only from users the requester follows + own stories.
// Only non-expired stories are returned.
func (r *StoryRepository) GetFeedForUser(ctx context.Context, userDID string) ([]models.StoryUser, error) {
	// Get all non-expired stories from followed users + own stories
	query := `
		SELECT s.id, s.author_did, s.media_type, s.created_at, s.expires_at,
		       COALESCE(u.username, '') AS username,
		       COALESCE(u.display_name, u.username, '') AS display_name,
		       COALESCE(u.avatar_url, '') AS avatar_url
		FROM stories s
		LEFT JOIN users u ON s.author_did = u.did
		WHERE s.expires_at > NOW()
		  AND (
		    s.author_did = $1
		    OR s.author_did IN (
		      SELECT f.following_did FROM follows f
		      WHERE f.follower_did = $1 AND f.status = 'accepted'
		    )
		  )
		ORDER BY s.author_did, s.created_at ASC
	`

	rows, err := db.GetDB().Query(ctx, query, userDID)
	if err != nil {
		return nil, fmt.Errorf("failed to get story feed: %w", err)
	}
	defer rows.Close()

	// Group stories by author
	userMap := make(map[string]*models.StoryUser)
	var orderedDIDs []string

	for rows.Next() {
		var story models.Story
		var username, displayName, avatarURL string

		if err := rows.Scan(
			&story.ID,
			&story.AuthorDID,
			&story.MediaType,
			&story.CreatedAt,
			&story.ExpiresAt,
			&username,
			&displayName,
			&avatarURL,
		); err != nil {
			return nil, fmt.Errorf("failed to scan story: %w", err)
		}

		story.MediaURL = fmt.Sprintf("/api/v1/stories/%s/media", story.ID)

		if _, exists := userMap[story.AuthorDID]; !exists {
			userMap[story.AuthorDID] = &models.StoryUser{
				AuthorDID:   story.AuthorDID,
				Username:    username,
				DisplayName: displayName,
				AvatarURL:   avatarURL,
				Stories:     []models.Story{},
			}
			orderedDIDs = append(orderedDIDs, story.AuthorDID)
		}

		userMap[story.AuthorDID].Stories = append(userMap[story.AuthorDID].Stories, story)
	}

	// Build result with own stories first
	var result []models.StoryUser
	if own, ok := userMap[userDID]; ok {
		result = append(result, *own)
	}
	for _, did := range orderedDIDs {
		if did == userDID {
			continue
		}
		result = append(result, *userMap[did])
	}

	if result == nil {
		result = []models.StoryUser{}
	}

	return result, nil
}

// GetMediaContent retrieves the binary media data for a story
func (r *StoryRepository) GetMediaContent(ctx context.Context, storyID string) ([]byte, string, error) {
	query := `
		SELECT media_data, media_type
		FROM stories
		WHERE id = $1 AND expires_at > NOW()
	`

	var mediaData []byte
	var mediaType string
	err := db.GetDB().QueryRow(ctx, query, storyID).Scan(&mediaData, &mediaType)
	if err == pgx.ErrNoRows {
		return nil, "", fmt.Errorf("story not found or expired")
	}
	if err != nil {
		return nil, "", fmt.Errorf("failed to get story media: %w", err)
	}

	return mediaData, mediaType, nil
}

// Delete removes a story (only the author can delete)
func (r *StoryRepository) Delete(ctx context.Context, storyID, authorDID string) error {
	query := `DELETE FROM stories WHERE id = $1 AND author_did = $2`
	result, err := db.GetDB().Exec(ctx, query, storyID, authorDID)
	if err != nil {
		return fmt.Errorf("failed to delete story: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("story not found or unauthorized")
	}
	return nil
}
