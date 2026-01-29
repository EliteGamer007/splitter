package repository

import (
	"context"
	"fmt"

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
func (r *PostRepository) Create(ctx context.Context, authorDID string, post *models.PostCreate) (*models.Post, error) {
	visibility := post.Visibility
	if visibility == "" {
		visibility = "public"
	}

	query := `
		INSERT INTO posts (author_did, content, visibility)
		VALUES ($1, $2, $3)
		RETURNING id, author_did, content, visibility, is_remote, created_at, updated_at
	`

	var newPost models.Post
	err := db.GetDB().QueryRow(ctx, query,
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

	return &newPost, nil
}

// GetByID retrieves a post by ID
func (r *PostRepository) GetByID(ctx context.Context, id string) (*models.Post, error) {
	query := `
		SELECT p.id, p.author_did, p.content, p.visibility, p.is_remote, 
		       p.created_at, p.updated_at, u.username,
		       COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'like'), 0) as like_count
		FROM posts p
		LEFT JOIN users u ON p.author_did = u.did
		WHERE p.id = $1 AND p.deleted_at IS NULL
	`

	var post models.Post
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
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("post not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	return &post, nil
}

// GetByAuthorDID retrieves all posts by a specific author
func (r *PostRepository) GetByAuthorDID(ctx context.Context, authorDID string, limit, offset int) ([]*models.Post, error) {
	query := `
		SELECT p.id, p.author_did, p.content, p.visibility, p.is_remote, 
		       p.created_at, p.updated_at, u.username,
		       COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'like'), 0) as like_count
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
		       p.created_at, p.updated_at, u.username,
		       COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'like'), 0) as like_count
		FROM posts p
		LEFT JOIN users u ON p.author_did = u.did
		LEFT JOIN follows f ON p.author_did = f.following_did
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
		); err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}
		posts = append(posts, &post)
	}

	return posts, nil
}

// GetPublicFeed retrieves public posts for unauthenticated users
func (r *PostRepository) GetPublicFeed(ctx context.Context, limit, offset int) ([]*models.Post, error) {
	query := `
		SELECT p.id, p.author_did, p.content, p.visibility, p.is_remote, 
		       p.created_at, p.updated_at, u.username,
		       COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'like'), 0) as like_count
		FROM posts p
		LEFT JOIN users u ON p.author_did = u.did
		WHERE p.visibility = 'public' AND p.deleted_at IS NULL
		ORDER BY p.created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := db.GetDB().Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get public feed: %w", err)
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
		); err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}
		posts = append(posts, &post)
	}

	return posts, nil
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
		RETURNING id, author_did, content, visibility, is_remote, created_at, updated_at
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
func (r *PostRepository) Delete(ctx context.Context, postID, authorDID string) error {
	query := `UPDATE posts SET deleted_at = NOW() WHERE id = $1 AND author_did = $2 AND deleted_at IS NULL`

	result, err := db.GetDB().Exec(ctx, query, postID, authorDID)
	if err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("post not found or unauthorized")
	}

	return nil
}
