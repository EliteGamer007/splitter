package repository

import (
	"context"
	"fmt"

	"splitter/internal/db"
)

// InteractionRepository handles database operations for post interactions
type InteractionRepository struct{}

// NewInteractionRepository creates a new InteractionRepository
func NewInteractionRepository() *InteractionRepository {
	return &InteractionRepository{}
}

// CreateLike creates a like on a post
func (r *InteractionRepository) CreateLike(ctx context.Context, postID, actorDID string) error {
	query := `
		INSERT INTO interactions (post_id, actor_did, interaction_type)
		VALUES ($1, $2, 'like')
		ON CONFLICT (post_id, actor_did, interaction_type) DO NOTHING
	`

	_, err := db.GetDB().Exec(ctx, query, postID, actorDID)
	if err != nil {
		return fmt.Errorf("failed to create like: %w", err)
	}

	return nil
}

// DeleteLike removes a like from a post
func (r *InteractionRepository) DeleteLike(ctx context.Context, postID, actorDID string) error {
	query := `DELETE FROM interactions WHERE post_id = $1 AND actor_did = $2 AND interaction_type = 'like'`

	result, err := db.GetDB().Exec(ctx, query, postID, actorDID)
	if err != nil {
		return fmt.Errorf("failed to delete like: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("like not found")
	}

	return nil
}

// CreateRepost creates a repost/boost of a post
func (r *InteractionRepository) CreateRepost(ctx context.Context, postID, actorDID string) error {
	query := `
		INSERT INTO interactions (post_id, actor_did, interaction_type)
		VALUES ($1, $2, 'repost')
		ON CONFLICT (post_id, actor_did, interaction_type) DO NOTHING
	`

	_, err := db.GetDB().Exec(ctx, query, postID, actorDID)
	if err != nil {
		return fmt.Errorf("failed to create repost: %w", err)
	}

	return nil
}

// DeleteRepost removes a repost
func (r *InteractionRepository) DeleteRepost(ctx context.Context, postID, actorDID string) error {
	query := `DELETE FROM interactions WHERE post_id = $1 AND actor_did = $2 AND interaction_type = 'repost'`

	result, err := db.GetDB().Exec(ctx, query, postID, actorDID)
	if err != nil {
		return fmt.Errorf("failed to delete repost: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("repost not found")
	}

	return nil
}

// CreateBookmark creates a bookmark (private save) of a post
func (r *InteractionRepository) CreateBookmark(ctx context.Context, userID, postID string) error {
	query := `
		INSERT INTO bookmarks (user_id, post_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, post_id) DO NOTHING
	`

	_, err := db.GetDB().Exec(ctx, query, userID, postID)
	if err != nil {
		return fmt.Errorf("failed to create bookmark: %w", err)
	}

	return nil
}

// DeleteBookmark removes a bookmark
func (r *InteractionRepository) DeleteBookmark(ctx context.Context, userID, postID string) error {
	query := `DELETE FROM bookmarks WHERE user_id = $1 AND post_id = $2`

	result, err := db.GetDB().Exec(ctx, query, userID, postID)
	if err != nil {
		return fmt.Errorf("failed to delete bookmark: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("bookmark not found")
	}

	return nil
}

// GetBookmarks retrieves all bookmarked posts for a user
func (r *InteractionRepository) GetBookmarks(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	query := `
		SELECT p.id, p.author_did, p.content, p.visibility, p.created_at,
		       u.username, u.display_name, u.avatar_url
		FROM posts p
		INNER JOIN bookmarks b ON p.id = b.post_id
		INNER JOIN users u ON p.author_did = u.did
		WHERE b.user_id = $1
		ORDER BY b.created_at DESC
	`

	rows, err := db.GetDB().Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bookmarks: %w", err)
	}
	defer rows.Close()

	var posts []map[string]interface{}
	for rows.Next() {
		var (
			postID      string
			authorDID   string
			content     string
			visibility  string
			createdAt   string
			username    string
			displayName string
			avatarURL   *string
		)

		if err := rows.Scan(&postID, &authorDID, &content, &visibility, &createdAt,
			&username, &displayName, &avatarURL); err != nil {
			return nil, fmt.Errorf("failed to scan bookmark: %w", err)
		}

		post := map[string]interface{}{
			"id":         postID,
			"author_did": authorDID,
			"content":    content,
			"visibility": visibility,
			"created_at": createdAt,
			"author": map[string]interface{}{
				"username":     username,
				"display_name": displayName,
				"avatar_url":   avatarURL,
			},
		}
		posts = append(posts, post)
	}

	return posts, nil
}

// GetInteractionCounts retrieves like and repost counts for a post
func (r *InteractionRepository) GetInteractionCounts(ctx context.Context, postID string) (map[string]int, error) {
	query := `
		SELECT 
			COALESCE(SUM(CASE WHEN interaction_type = 'like' THEN 1 ELSE 0 END), 0) as likes,
			COALESCE(SUM(CASE WHEN interaction_type = 'repost' THEN 1 ELSE 0 END), 0) as reposts
		FROM interactions
		WHERE post_id = $1
	`

	var likes, reposts int
	err := db.GetDB().QueryRow(ctx, query, postID).Scan(&likes, &reposts)
	if err != nil {
		return nil, fmt.Errorf("failed to get interaction counts: %w", err)
	}

	return map[string]int{
		"likes":   likes,
		"reposts": reposts,
	}, nil
}

// HasUserLiked checks if a user has liked a post
func (r *InteractionRepository) HasUserLiked(ctx context.Context, postID, actorDID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM interactions WHERE post_id = $1 AND actor_did = $2 AND interaction_type = 'like')`

	var hasLiked bool
	err := db.GetDB().QueryRow(ctx, query, postID, actorDID).Scan(&hasLiked)
	if err != nil {
		return false, fmt.Errorf("failed to check like status: %w", err)
	}

	return hasLiked, nil
}

// HasUserReposted checks if a user has reposted a post
func (r *InteractionRepository) HasUserReposted(ctx context.Context, postID, actorDID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM interactions WHERE post_id = $1 AND actor_did = $2 AND interaction_type = 'repost')`

	var hasReposted bool
	err := db.GetDB().QueryRow(ctx, query, postID, actorDID).Scan(&hasReposted)
	if err != nil {
		return false, fmt.Errorf("failed to check repost status: %w", err)
	}

	return hasReposted, nil
}
