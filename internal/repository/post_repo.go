package repository

import (
	"context"
	"fmt"
	"time"

	"splitter/internal/db"
	"splitter/internal/models"

	"github.com/jackc/pgx/v5"
)

// PostRepository handles database operations for posts
type PostRepository struct{}

// NewPostRepository creates a new PostRepository
func NewPostRepository() *PostRepository {
	return &PostRepository{}
}

// Create creates a new post in the database
func (r *PostRepository) Create(ctx context.Context, authorDID string, post *models.PostCreate, mediaURL, mediaType string) (*models.Post, error) {
	visibility := post.Visibility
	if visibility == "" {
		visibility = "public"
	}

	tx, err := db.GetDB().Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO posts (author_did, content, visibility)
		VALUES ($1, $2, $3)
		RETURNING id, author_did, content, visibility, is_remote, created_at, updated_at
	`

	var newPost models.Post
	err = tx.QueryRow(ctx, query,
		authorDID,
		post.Content,
		visibility,
	).Scan(
		&newPost.ID,
		&newPost.AuthorDID,
		&newPost.Content,
		&newPost.Visibility,
		&newPost.IsRemote,
		&newPost.CreatedAt,
		&newPost.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
	}

	// Insert media if present
	if mediaURL != "" {
		mediaQuery := `
			INSERT INTO media (post_id, media_url, media_type)
			VALUES ($1, $2, $3)
			RETURNING id, post_id, media_url, media_type, created_at
		`
		var media models.Media
		err = tx.QueryRow(ctx, mediaQuery, newPost.ID, mediaURL, mediaType).Scan(
			&media.ID,
			&media.PostID,
			&media.MediaURL,
			&media.MediaType,
			&media.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create media: %w", err)
		}
		newPost.Media = []models.Media{media}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &newPost, nil
}

// GetByID retrieves a post by ID
func (r *PostRepository) GetByID(ctx context.Context, id string) (*models.Post, error) {
	query := `
		SELECT p.id, p.author_did, p.content, p.visibility, p.is_remote, 
		       p.created_at, p.updated_at, COALESCE(u.username, '') as username,
		       COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'like'), 0) as like_count,
		       p.direct_reply_count, p.total_reply_count,
		       m.id, m.media_url, m.media_type, m.created_at
		FROM posts p
		LEFT JOIN users u ON p.author_did = u.did
		LEFT JOIN media m ON p.id = m.post_id
		WHERE p.id = $1 AND p.deleted_at IS NULL
	`

	var post models.Post
	var mediaID, mediaURL, mediaType *string
	var mediaCreatedAt *time.Time

	err := db.GetDB().QueryRow(ctx, query, id).Scan(
		&post.ID,
		&post.AuthorDID,
		&post.Content,
		&post.Visibility,
		&post.IsRemote,
		&post.CreatedAt,
		&post.UpdatedAt,
		&post.Username,
		&post.LikeCount,
		&post.DirectReplyCount,
		&post.TotalReplyCount,
		&mediaID,
		&mediaURL,
		&mediaType,
		&mediaCreatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("post not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	if mediaID != nil {
		post.Media = []models.Media{{
			ID:        *mediaID,
			PostID:    post.ID,
			MediaURL:  *mediaURL,
			MediaType: *mediaType,
			CreatedAt: *mediaCreatedAt,
		}}
	}

	return &post, nil
}

// GetByAuthorDID retrieves all posts by a specific author
func (r *PostRepository) GetByAuthorDID(ctx context.Context, authorDID string, limit, offset int) ([]*models.Post, error) {
	query := `
		SELECT p.id, p.author_did, p.content, p.visibility, p.is_remote, 
		       p.created_at, p.updated_at, COALESCE(u.username, '') as username,
		       COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'like'), 0) as like_count,
		       p.direct_reply_count, p.total_reply_count
		FROM posts p
		LEFT JOIN users u ON p.author_did = u.did
		WHERE p.author_did = $1 AND p.deleted_at IS NULL
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.GetDB().Query(ctx, query, authorDID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get user posts: %w", err)
	}
	defer rows.Close()

	var posts []*models.Post
	for rows.Next() {
		var post models.Post
		if err := rows.Scan(
			&post.ID,
			&post.AuthorDID,
			&post.Content,
			&post.Visibility,
			&post.IsRemote,
			&post.CreatedAt,
			&post.UpdatedAt,
			&post.Username,
			&post.LikeCount,
			&post.DirectReplyCount,
			&post.TotalReplyCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}
		posts = append(posts, &post)
	}

	return posts, nil
}

// GetFeed retrieves posts from users that the given user follows
func (r *PostRepository) GetFeed(ctx context.Context, userDID string, limit, offset int) ([]*models.Post, error) {
	query := `
		SELECT p.id, p.author_did, p.content, p.visibility, p.is_remote, 
		       p.created_at, p.updated_at, COALESCE(u.username, '') as username,
		       COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'like'), 0) as like_count,
		       COALESCE((SELECT COUNT(*) > 0 FROM interactions WHERE post_id = p.id AND actor_did = $1 AND interaction_type = 'like'), false) as liked_by_user,
		       COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'repost'), 0) as repost_count,
		       COALESCE((SELECT COUNT(*) > 0 FROM interactions WHERE post_id = p.id AND actor_did = $1 AND interaction_type = 'repost'), false) as reposted_by_user,
		       p.direct_reply_count, p.total_reply_count,
		       m.id, m.media_url, m.media_type, m.created_at
		FROM posts p
		LEFT JOIN users u ON p.author_did = u.did
		LEFT JOIN follows f ON p.author_did = f.following_did
		LEFT JOIN media m ON p.id = m.post_id
		WHERE (f.follower_did = $1 OR p.author_did = $1) AND p.deleted_at IS NULL
		  AND (p.visibility = 'public' OR (p.visibility = 'followers' AND f.follower_did = $1))
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.GetDB().Query(ctx, query, userDID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get feed: %w", err)
	}
	defer rows.Close()

	var posts []*models.Post
	for rows.Next() {
		var post models.Post
		var mediaID, mediaURL, mediaType *string
		var mediaCreatedAt *time.Time

		if err := rows.Scan(
			&post.ID,
			&post.AuthorDID,
			&post.Content,
			&post.Visibility,
			&post.IsRemote,
			&post.CreatedAt,
			&post.UpdatedAt,
			&post.Username,
			&post.LikeCount,
			&post.Liked,
			&post.RepostCount,
			&post.Reposted,
			&post.DirectReplyCount,
			&post.TotalReplyCount,
			&mediaID,
			&mediaURL,
			&mediaType,
			&mediaCreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}
		if mediaID != nil {
			post.Media = []models.Media{{
				ID:        *mediaID,
				PostID:    post.ID,
				MediaURL:  *mediaURL,
				MediaType: *mediaType,
				CreatedAt: *mediaCreatedAt,
			}}
		}
		posts = append(posts, &post)
	}

	return posts, nil
}

// GetPublicFeed retrieves public posts for unauthenticated users or with optional user context
func (r *PostRepository) GetPublicFeedWithUser(ctx context.Context, userDID string, limit, offset int) ([]*models.Post, error) {
	var query string
	var args []interface{}

	if userDID != "" {
		// Authenticated user - include liked and reposted status
		query = `
			SELECT p.id, p.author_did, p.content, p.visibility, p.is_remote, 
			       p.created_at, p.updated_at, COALESCE(u.username, '') as username,
			       COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'like'), 0) as like_count,
			       COALESCE((SELECT COUNT(*) > 0 FROM interactions WHERE post_id = p.id AND actor_did = $1 AND interaction_type = 'like'), false) as liked_by_user,
			       COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'repost'), 0) as repost_count,
			       COALESCE((SELECT COUNT(*) > 0 FROM interactions WHERE post_id = p.id AND actor_did = $1 AND interaction_type = 'repost'), false) as reposted_by_user,
			       p.direct_reply_count, p.total_reply_count,
			       m.id, m.media_url, m.media_type, m.created_at
			FROM posts p
			LEFT JOIN users u ON p.author_did = u.did
			LEFT JOIN media m ON p.id = m.post_id
			WHERE p.visibility = 'public' AND p.deleted_at IS NULL
			ORDER BY p.created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{userDID, limit, offset}
	} else {
		// Unauthenticated user - liked and reposted are always false
		query = `
			SELECT p.id, p.author_did, p.content, p.visibility, p.is_remote, 
			       p.created_at, p.updated_at, COALESCE(u.username, '') as username,
			       COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'like'), 0) as like_count,
			       false as liked_by_user,
			       COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'repost'), 0) as repost_count,
			       false as reposted_by_user,
			       p.direct_reply_count, p.total_reply_count,
			       m.id, m.media_url, m.media_type, m.created_at
			FROM posts p
			LEFT JOIN users u ON p.author_did = u.did
			LEFT JOIN media m ON p.id = m.post_id
			WHERE p.visibility = 'public' AND p.deleted_at IS NULL
			ORDER BY p.created_at DESC
			LIMIT $1 OFFSET $2
		`
		args = []interface{}{limit, offset}
	}

	rows, err := db.GetDB().Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get public feed: %w", err)
	}
	defer rows.Close()

	var posts []*models.Post
	for rows.Next() {
		var post models.Post
		var mediaID, mediaURL, mediaType *string
		var mediaCreatedAt *time.Time

		if err := rows.Scan(
			&post.ID,
			&post.AuthorDID,
			&post.Content,
			&post.Visibility,
			&post.IsRemote,
			&post.CreatedAt,
			&post.UpdatedAt,
			&post.Username,
			&post.LikeCount,
			&post.Liked,
			&post.RepostCount,
			&post.Reposted,
			&post.DirectReplyCount,
			&post.TotalReplyCount,
			&mediaID,
			&mediaURL,
			&mediaType,
			&mediaCreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}
		if mediaID != nil {
			post.Media = []models.Media{{
				ID:        *mediaID,
				PostID:    post.ID,
				MediaURL:  *mediaURL,
				MediaType: *mediaType,
				CreatedAt: *mediaCreatedAt,
			}}
		}
		posts = append(posts, &post)
	}

	return posts, nil
}

// GetPublicFeed retrieves public posts for unauthenticated users
func (r *PostRepository) GetPublicFeed(ctx context.Context, limit, offset int) ([]*models.Post, error) {
	return r.GetPublicFeedWithUser(ctx, "", limit, offset)
}

// Update updates a post's content
func (r *PostRepository) Update(ctx context.Context, postID, authorDID string, update *models.PostUpdate) (*models.Post, error) {
	query := `
		UPDATE posts
		SET 
			content = COALESCE($1, content),
			visibility = COALESCE($2, visibility),
			updated_at = NOW()
		WHERE id = $3 AND author_did = $4 AND deleted_at IS NULL
		RETURNING id, author_did, content, visibility, is_remote, created_at, updated_at, direct_reply_count, total_reply_count
	`

	var post models.Post
	err := db.GetDB().QueryRow(ctx, query,
		update.Content,
		update.Visibility,
		postID,
		authorDID,
	).Scan(
		&post.ID,
		&post.AuthorDID,
		&post.Content,
		&post.Visibility,
		&post.IsRemote,
		&post.CreatedAt,
		&post.UpdatedAt,
		&post.DirectReplyCount,
		&post.TotalReplyCount,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("post not found or unauthorized")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update post: %w", err)
	}

	return &post, nil
}

// Delete soft-deletes a post
func (r *PostRepository) Delete(ctx context.Context, postID, authorDID string, isAdmin bool) error {
	var query string

	if isAdmin {
		// Admins can delete any post
		query = `UPDATE posts SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
		result, err := db.GetDB().Exec(ctx, query, postID)
		if err != nil {
			return fmt.Errorf("failed to delete post: %w", err)
		}
		if result.RowsAffected() == 0 {
			return fmt.Errorf("post not found")
		}
	} else {
		// Regular users can only delete their own posts
		query = `UPDATE posts SET deleted_at = NOW() WHERE id = $1 AND author_did = $2 AND deleted_at IS NULL`
		result, err := db.GetDB().Exec(ctx, query, postID, authorDID)
		if err != nil {
			return fmt.Errorf("failed to delete post: %w", err)
		}
		if result.RowsAffected() == 0 {
			return fmt.Errorf("post not found or unauthorized")
		}
	}

	return nil
}
