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
func (r *PostRepository) Create(ctx context.Context, userID int, post *models.PostCreate) (*models.Post, error) {
	query := `
		INSERT INTO posts (user_id, content, image_url)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, content, image_url, like_count, created_at, updated_at
	`

	var newPost models.Post
	err := db.GetDB().QueryRow(ctx, query,
		userID,
		post.Content,
		post.ImageURL,
	).Scan(
		&newPost.ID,
		&newPost.UserID,
		&newPost.Content,
		&newPost.ImageURL,
		&newPost.LikeCount,
		&newPost.CreatedAt,
		&newPost.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
	}

	return &newPost, nil
}

// GetByID retrieves a post by ID
func (r *PostRepository) GetByID(ctx context.Context, id int) (*models.Post, error) {
	query := `
		SELECT p.id, p.user_id, p.content, p.image_url, p.like_count, 
		       p.created_at, p.updated_at, u.username
		FROM posts p
		JOIN users u ON p.user_id = u.id
		WHERE p.id = $1
	`

	var post models.Post
	err := db.GetDB().QueryRow(ctx, query, id).Scan(
		&post.ID,
		&post.UserID,
		&post.Content,
		&post.ImageURL,
		&post.LikeCount,
		&post.CreatedAt,
		&post.UpdatedAt,
		&post.Username,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("post not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get post: %w", err)
	}

	return &post, nil
}

// GetByUserID retrieves all posts by a specific user
func (r *PostRepository) GetByUserID(ctx context.Context, userID int, limit, offset int) ([]*models.Post, error) {
	query := `
		SELECT p.id, p.user_id, p.content, p.image_url, p.like_count, 
		       p.created_at, p.updated_at, u.username
		FROM posts p
		JOIN users u ON p.user_id = u.id
		WHERE p.user_id = $1
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.GetDB().Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get user posts: %w", err)
	}
	defer rows.Close()

	var posts []*models.Post
	for rows.Next() {
		var post models.Post
		if err := rows.Scan(
			&post.ID,
			&post.UserID,
			&post.Content,
			&post.ImageURL,
			&post.LikeCount,
			&post.CreatedAt,
			&post.UpdatedAt,
			&post.Username,
		); err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}
		posts = append(posts, &post)
	}

	return posts, nil
}

// GetFeed retrieves posts from users that the given user follows
func (r *PostRepository) GetFeed(ctx context.Context, userID int, limit, offset int) ([]*models.Post, error) {
	query := `
		SELECT p.id, p.user_id, p.content, p.image_url, p.like_count, 
		       p.created_at, p.updated_at, u.username
		FROM posts p
		JOIN users u ON p.user_id = u.id
		JOIN follows f ON p.user_id = f.following_id
		WHERE f.follower_id = $1
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.GetDB().Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get feed: %w", err)
	}
	defer rows.Close()

	var posts []*models.Post
	for rows.Next() {
		var post models.Post
		if err := rows.Scan(
			&post.ID,
			&post.UserID,
			&post.Content,
			&post.ImageURL,
			&post.LikeCount,
			&post.CreatedAt,
			&post.UpdatedAt,
			&post.Username,
		); err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}
		posts = append(posts, &post)
	}

	return posts, nil
}

// Update updates a post's content
func (r *PostRepository) Update(ctx context.Context, postID, userID int, update *models.PostUpdate) (*models.Post, error) {
	query := `
		UPDATE posts
		SET 
			content = COALESCE($1, content),
			image_url = COALESCE($2, image_url),
			updated_at = NOW()
		WHERE id = $3 AND user_id = $4
		RETURNING id, user_id, content, image_url, like_count, created_at, updated_at
	`

	var post models.Post
	err := db.GetDB().QueryRow(ctx, query,
		update.Content,
		update.ImageURL,
		postID,
		userID,
	).Scan(
		&post.ID,
		&post.UserID,
		&post.Content,
		&post.ImageURL,
		&post.LikeCount,
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

// Delete deletes a post
func (r *PostRepository) Delete(ctx context.Context, postID, userID int) error {
	query := `DELETE FROM posts WHERE id = $1 AND user_id = $2`

	result, err := db.GetDB().Exec(ctx, query, postID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete post: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("post not found or unauthorized")
	}

	return nil
}
