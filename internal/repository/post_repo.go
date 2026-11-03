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

type PostFederationMeta struct {
	AuthorDID       string
	OriginalPostURI string
	IsRemote        bool
}

// NewPostRepository creates a new PostRepository
func NewPostRepository() *PostRepository {
	return &PostRepository{}
}

// Create creates a new post in the database.
// Media bytes are stored durably with the post inside one quota-checked transaction.
func (r *PostRepository) Create(ctx context.Context, authorDID string, post *models.PostCreate, mediaURL, mediaType string) (*models.Post, error) {
	return r.CreateWithMedia(ctx, authorDID, post, mediaURL, mediaType, nil)
}
func (r *PostRepository) CreateWithMedia(ctx context.Context, authorDID string, post *models.PostCreate, mediaURL, mediaType string, mediaData []byte) (*models.Post, error) {
	if len(mediaData) > db.MaxStoredMedia {
		return nil, fmt.Errorf("stored image exceeds 256 KiB")
	}

	visibility := post.Visibility
	if visibility == "" {
		visibility = "public"
	}

	if visibility != "public" && visibility != "followers" && visibility != "circle" {
		return nil, fmt.Errorf("invalid visibility")
	}
	var expiresAt *time.Time
	if post.ExpiresInMinutes != nil && *post.ExpiresInMinutes > 0 {
		t := time.Now().UTC().Add(time.Duration(*post.ExpiresInMinutes) * time.Minute)
		expiresAt = &t
	}

	tx, err := db.GetDB().Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)
	if len(mediaData) > 0 {
		var ownerID string
		if err = tx.QueryRow(ctx, `SELECT id FROM users WHERE did=$1 AND NOT is_suspended`, authorDID).Scan(&ownerID); err != nil {
			return nil, fmt.Errorf("account unavailable")
		}
		if err = db.EnforceMediaBudget(ctx, tx, ownerID, int64(len(mediaData))); err != nil {
			return nil, err
		}
	} else if err = db.CheckGrowthBudget(ctx, tx, int64(len(post.Content)+1024)); err != nil {
		return nil, err
	}
	query := `
		INSERT INTO posts (author_did, content, visibility, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, author_did, content, visibility, is_remote, created_at, updated_at, expires_at
	`

	var newPost models.Post
	err = tx.QueryRow(ctx, query,
		authorDID,
		post.Content,
		visibility,
		expiresAt,
	).Scan(
		&newPost.ID,
		&newPost.AuthorDID,
		&newPost.Content,
		&newPost.Visibility,
		&newPost.IsRemote,
		&newPost.CreatedAt,
		&newPost.UpdatedAt,
		&newPost.ExpiresAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
	}

	// Insert media metadata if present
	if mediaURL != "" || len(mediaData) > 0 {
		mediaQuery := `
			INSERT INTO media (post_id, media_url, media_type,media_data)
			VALUES ($1, $2, $3,$4)
			RETURNING id, post_id, media_type, created_at
		`
		var media models.Media
		err = tx.QueryRow(ctx, mediaQuery, newPost.ID, mediaURL, mediaType, mediaData).Scan(
			&media.ID,
			&media.PostID,
			&media.MediaType,
			&media.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create media: %w", err)
		}

		if len(mediaData) > 0 {
			mediaURL = "/api/v1/media/" + media.ID
			if _, err = tx.Exec(ctx, `UPDATE media SET media_url=$1 WHERE id=$2`, mediaURL, media.ID); err != nil {
				return nil, err
			}
		}
		media.MediaURL = mediaURL
		newPost.Media = []models.Media{media}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &newPost, nil
}

// GetMediaContent retrieves media bytes and type by media ID.
func (r *PostRepository) GetMediaContent(ctx context.Context, mediaID string) ([]byte, string, string, error) {
	query := `
		SELECT COALESCE(media_data, ''::bytea), COALESCE(media_type, ''), COALESCE(media_url, '')
		FROM media
		WHERE id = $1
	`

	var mediaData []byte
	var mediaType, mediaURL string
	err := db.GetDB().QueryRow(ctx, query, mediaID).Scan(&mediaData, &mediaType, &mediaURL)
	if err == pgx.ErrNoRows {
		return nil, "", "", fmt.Errorf("media not found")
	}
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get media content: %w", err)
	}

	return mediaData, mediaType, mediaURL, nil
}

// GetByID retrieves a post by ID
func (r *PostRepository) GetByID(ctx context.Context, id string) (*models.Post, error) {
	return r.getByID(ctx, id, "", false)
}
func (r *PostRepository) getByID(ctx context.Context, id, viewerDID string, checkAudience bool) (*models.Post, error) {
	query := `
		SELECT p.id, p.author_did, p.content, p.visibility, p.is_remote,
		       COALESCE(p.original_post_uri, ''), COALESCE(p.in_reply_to_uri, ''),
		       p.created_at, p.updated_at, p.expires_at,
		       COALESCE(u.username, '') as username, COALESCE(u.avatar_url, '') as avatar_url,
		       COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'like'), 0) as like_count,
		       p.direct_reply_count, p.total_reply_count,
		       m.id, m.media_url, m.media_type, m.created_at
		FROM posts p
		LEFT JOIN users u ON p.author_did = u.did
		LEFT JOIN media m ON p.id = m.post_id
		WHERE p.id = $1 AND p.deleted_at IS NULL
		  AND (p.expires_at IS NULL OR p.expires_at > NOW())
	`

	args := []interface{}{id}
	if checkAudience {
		query += " AND " + postVisibilitySQL("p", "$2")
		args = append(args, viewerDID)
	}
	var post models.Post
	var mediaID, mediaURL, mediaType *string
	var mediaCreatedAt *time.Time

	err := db.GetDB().QueryRow(ctx, query, args...).Scan(
		&post.ID,
		&post.AuthorDID,
		&post.Content,
		&post.Visibility,
		&post.IsRemote,
		&post.OriginalPostURI,
		&post.InReplyToURI,
		&post.CreatedAt,
		&post.UpdatedAt,
		&post.ExpiresAt,
		&post.Username,
		&post.AvatarURL,
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

// GetByOriginalURI retrieves a locally cached post by its original remote URI.
func (r *PostRepository) GetByOriginalURI(ctx context.Context, originalURI string) (*models.Post, error) {
	query := `
		SELECT p.id, p.author_did, p.content, p.visibility, p.is_remote,
		       COALESCE(p.original_post_uri, ''), COALESCE(p.in_reply_to_uri, ''),
		       p.created_at, p.updated_at, p.expires_at,
		       COALESCE(u.username, '') as username, COALESCE(u.avatar_url, '') as avatar_url,
		       COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'like'), 0) as like_count,
		       p.direct_reply_count, p.total_reply_count,
		       m.id, m.media_url, m.media_type, m.created_at
		FROM posts p
		LEFT JOIN users u ON p.author_did = u.did
		LEFT JOIN media m ON p.id = m.post_id
		WHERE p.original_post_uri = $1 AND p.deleted_at IS NULL
		  AND (p.expires_at IS NULL OR p.expires_at > NOW())
		ORDER BY p.created_at DESC
		LIMIT 1
	`

	var post models.Post
	var mediaID, mediaURL, mediaType *string
	var mediaCreatedAt *time.Time

	err := db.GetDB().QueryRow(ctx, query, originalURI).Scan(
		&post.ID,
		&post.AuthorDID,
		&post.Content,
		&post.Visibility,
		&post.IsRemote,
		&post.OriginalPostURI,
		&post.InReplyToURI,
		&post.CreatedAt,
		&post.UpdatedAt,
		&post.ExpiresAt,
		&post.Username,
		&post.AvatarURL,
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
		return nil, fmt.Errorf("failed to get post by original URI: %w", err)
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

// CreateRemoteCachedPost inserts a remote post fetched on demand for thread context.
func (r *PostRepository) CreateRemoteCachedPost(ctx context.Context, authorDID, content, originalURI, inReplyToURI string, createdAt time.Time) (*models.Post, error) {
	if originalURI == "" {
		return nil, fmt.Errorf("original URI is required")
	}

	var id string
	err := db.GetDB().QueryRow(ctx, `
		WITH existing AS (
			SELECT id::text AS id
			FROM posts
			WHERE original_post_uri = $1 AND deleted_at IS NULL
			ORDER BY created_at DESC
			LIMIT 1
		), inserted AS (
			INSERT INTO posts (author_did, content, visibility, is_remote, original_post_uri, in_reply_to_uri, created_at)
			SELECT $2, $3, 'public', true, $1, NULLIF($4, ''), $5
			WHERE NOT EXISTS (SELECT 1 FROM existing)
			RETURNING id::text AS id
		)
		SELECT id FROM inserted
		UNION ALL
		SELECT id FROM existing
		LIMIT 1
	`, originalURI, authorDID, content, inReplyToURI, createdAt.UTC()).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("failed to cache remote post: %w", err)
	}

	return r.GetByID(ctx, id)
}

// GetByAuthorDID retrieves all posts by a specific author
func (r *PostRepository) GetByAuthorDID(ctx context.Context, authorDID string, limit, offset int) ([]*models.Post, error) {
	limit, offset = boundedPage(limit, offset, 20)
	return r.GetByAuthorDIDForViewer(ctx, authorDID, "", limit, offset)
}
func (r *PostRepository) GetByAuthorDIDForViewer(ctx context.Context, authorDID, viewerDID string, limit, offset int) ([]*models.Post, error) {
	limit, offset = boundedPage(limit, offset, 20)
	query := `
		SELECT p.id, p.author_did, p.content, p.visibility, p.is_remote,
		       COALESCE(p.original_post_uri,''),COALESCE(p.in_reply_to_uri,''),p.created_at, p.updated_at, p.expires_at,
		       COALESCE(u.username, '') as username, COALESCE(u.avatar_url, '') as avatar_url,
		       COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'like'), 0) as like_count,
		       p.direct_reply_count, p.total_reply_count
		FROM posts p
		LEFT JOIN users u ON p.author_did = u.did
		WHERE p.author_did = $1 AND p.deleted_at IS NULL
		  AND (p.expires_at IS NULL OR p.expires_at > NOW())
		AND ` + postVisibilitySQL("p", "$4") + `
 ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.GetDB().Query(ctx, query, authorDID, limit, offset, viewerDID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user posts: %w", err)
	}
	defer rows.Close()

	posts := make([]*models.Post, 0)
	for rows.Next() {
		var post models.Post
		if err := rows.Scan(
			&post.ID,
			&post.AuthorDID,
			&post.Content,
			&post.Visibility,
			&post.IsRemote,
			&post.OriginalPostURI,
			&post.InReplyToURI,
			&post.CreatedAt,
			&post.UpdatedAt,
			&post.ExpiresAt,
			&post.Username,
			&post.AvatarURL,
			&post.LikeCount,
			&post.DirectReplyCount,
			&post.TotalReplyCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan post: %w", err)
		}
		posts = append(posts, &post)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	if len(posts) > 0 {
		ids := make([]string, 0, len(posts))
		byID := make(map[string]*models.Post, len(posts))
		for _, post := range posts {
			ids = append(ids, post.ID)
			byID[post.ID] = post
		}
		mediaRows, err := db.GetDB().Query(ctx, `SELECT m.id,m.post_id,COALESCE(m.media_url,''),COALESCE(m.media_type,''),m.created_at FROM media m JOIN posts p ON p.id=m.post_id WHERE m.post_id=ANY($1::uuid[]) AND `+postVisibilitySQL("p", "$2")+` ORDER BY m.created_at,m.id LIMIT 2500`, ids, viewerDID)
		if err != nil {
			return nil, err
		}
		defer mediaRows.Close()
		for mediaRows.Next() {
			var media models.Media
			if err = mediaRows.Scan(&media.ID, &media.PostID, &media.MediaURL, &media.MediaType, &media.CreatedAt); err != nil {
				return nil, err
			}
			if post := byID[media.PostID]; post != nil {
				post.Media = append(post.Media, media)
			}
		}
		if err = mediaRows.Err(); err != nil {
			return nil, err
		}
	}
	return posts, nil
}

// GetFeed retrieves posts from users that the given user follows
func (r *PostRepository) GetFeed(ctx context.Context, userDID string, limit, offset int) ([]*models.Post, error) {
	limit, offset = boundedPage(limit, offset, 20)
	query := `
		SELECT p.id, p.author_did, p.content, p.visibility, p.is_remote,
		       p.created_at, p.updated_at, p.expires_at,
		       COALESCE(u.username, '') as username, COALESCE(u.avatar_url, '') as avatar_url,
		       COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'like'), 0) as like_count,
		       COALESCE((SELECT COUNT(*) > 0 FROM interactions WHERE post_id = p.id AND actor_did = $1 AND interaction_type = 'like'), false) as liked_by_user,
		       COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'repost'), 0) as repost_count,
		       COALESCE((SELECT COUNT(*) > 0 FROM interactions WHERE post_id = p.id AND actor_did = $1 AND interaction_type = 'repost'), false) as reposted_by_user,
		       p.direct_reply_count, p.total_reply_count,
		       m.id, m.media_url, m.media_type, m.created_at
		FROM posts p
		LEFT JOIN users u ON p.author_did = u.did
		LEFT JOIN follows f ON p.author_did = f.following_did AND f.follower_did = $1 AND f.status = 'accepted'
		LEFT JOIN media m ON p.id = m.post_id
WHERE (f.follower_did=$1 OR p.author_did=$1) AND ` + postVisibilitySQL("p", "$1") + `
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
			&post.ExpiresAt,
			&post.Username,
			&post.AvatarURL,
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
func (r *PostRepository) GetPublicFeedWithUser(ctx context.Context, userDID string, limit, offset int, localOnly bool) ([]*models.Post, error) {
	limit, offset = boundedPage(limit, offset, 20)
	var query string
	var args []interface{}
	localFilterClause := ""
	if localOnly {
		localFilterClause = " AND p.is_remote = false"
	}

	if userDID != "" {
		// Authenticated user - include liked and reposted status
		query = `
			SELECT p.id, p.author_did, p.content, p.visibility, p.is_remote,
			       p.created_at, p.updated_at, p.expires_at,
			       COALESCE(u.username, '') as username, COALESCE(u.avatar_url, '') as avatar_url,
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
			  AND (p.expires_at IS NULL OR p.expires_at > NOW())` + localFilterClause + `
			ORDER BY p.created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{userDID, limit, offset}
	} else {
		// Unauthenticated user - liked and reposted are always false
		query = `
			SELECT p.id, p.author_did, p.content, p.visibility, p.is_remote,
			       p.created_at, p.updated_at, p.expires_at,
			       COALESCE(u.username, '') as username, COALESCE(u.avatar_url, '') as avatar_url,
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
			  AND (p.expires_at IS NULL OR p.expires_at > NOW())` + localFilterClause + `
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
			&post.ExpiresAt,
			&post.Username,
			&post.AvatarURL,
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
	limit, offset = boundedPage(limit, offset, 20)
	return r.GetPublicFeedWithUser(ctx, "", limit, offset, false)
}

// Update updates a post's content
func (r *PostRepository) Update(ctx context.Context, postID, authorDID string, update *models.PostUpdate) (*models.Post, error) {
	if update.Visibility != nil && *update.Visibility != "public" && *update.Visibility != "followers" && *update.Visibility != "circle" {
		return nil, fmt.Errorf("invalid visibility")
	}
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

func (r *PostRepository) GetFederationMeta(ctx context.Context, postID string) (*PostFederationMeta, error) {
	query := `
		SELECT author_did, COALESCE(original_post_uri, ''), is_remote
		FROM posts
		WHERE id = $1
	`

	var meta PostFederationMeta
	err := db.GetDB().QueryRow(ctx, query, postID).Scan(&meta.AuthorDID, &meta.OriginalPostURI, &meta.IsRemote)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("post not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get post federation metadata: %w", err)
	}

	return &meta, nil
}

// TrendingHashtag represents a hashtag with its post count
type TrendingHashtag struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

// GetTrendingHashtags returns the most used hashtags from the last 24 hours
func (r *PostRepository) GetTrendingHashtags(ctx context.Context, limit int) ([]TrendingHashtag, error) {
	query := `
		SELECT tag, COUNT(*) as cnt
		FROM (
			SELECT LOWER(match[1]) as tag
			FROM posts,
			     LATERAL regexp_matches(content, '#([A-Za-z0-9_]+)', 'g') AS match
			WHERE deleted_at IS NULL
			  AND visibility = 'public'
			  AND (expires_at IS NULL OR expires_at > NOW())
			  AND created_at > NOW() - INTERVAL '24 hours'
		) hashtags
		GROUP BY tag
		ORDER BY cnt DESC, tag ASC
		LIMIT $1
	`

	rows, err := db.GetDB().Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get trending hashtags: %w", err)
	}
	defer rows.Close()

	var trending []TrendingHashtag
	for rows.Next() {
		var h TrendingHashtag
		if err := rows.Scan(&h.Tag, &h.Count); err != nil {
			return nil, fmt.Errorf("failed to scan trending hashtag: %w", err)
		}
		trending = append(trending, h)
	}

	if trending == nil {
		trending = []TrendingHashtag{}
	}

	return trending, nil
}

// GetPostsByHashtag returns posts containing a specific hashtag
func (r *PostRepository) GetPostsByHashtag(ctx context.Context, tag string, userDID string, limit, offset int) ([]*models.Post, error) {
	limit, offset = boundedPage(limit, offset, 20)
	// In PostgreSQL POSIX regex, \y is the word boundary marker, not \b
	pattern := `#` + tag + `\y`

	var query string
	var args []interface{}

	if userDID != "" {
		query = `
			SELECT p.id, p.author_did, p.content, p.visibility, p.is_remote,
			       p.created_at, p.updated_at, p.expires_at,
			       COALESCE(u.username, '') as username, COALESCE(u.avatar_url, '') as avatar_url,
			       COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'like'), 0) as like_count,
			       COALESCE((SELECT COUNT(*) > 0 FROM interactions WHERE post_id = p.id AND actor_did = $1 AND interaction_type = 'like'), false) as liked_by_user,
			       COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'repost'), 0) as repost_count,
			       COALESCE((SELECT COUNT(*) > 0 FROM interactions WHERE post_id = p.id AND actor_did = $1 AND interaction_type = 'repost'), false) as reposted_by_user,
			       p.direct_reply_count, p.total_reply_count,
			       m.id, m.media_url, m.media_type, m.created_at
			FROM posts p
			LEFT JOIN users u ON p.author_did = u.did
			LEFT JOIN media m ON p.id = m.post_id
			WHERE p.content ~* $2
			  AND p.deleted_at IS NULL
			  AND p.visibility = 'public'
			  AND (p.expires_at IS NULL OR p.expires_at > NOW())
			ORDER BY p.created_at DESC
			LIMIT $3 OFFSET $4
		`
		args = []interface{}{userDID, pattern, limit, offset}
	} else {
		query = `
			SELECT p.id, p.author_did, p.content, p.visibility, p.is_remote,
			       p.created_at, p.updated_at, p.expires_at,
			       COALESCE(u.username, '') as username, COALESCE(u.avatar_url, '') as avatar_url,
			       COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'like'), 0) as like_count,
			       false as liked_by_user,
			       COALESCE((SELECT COUNT(*) FROM interactions WHERE post_id = p.id AND interaction_type = 'repost'), 0) as repost_count,
			       false as reposted_by_user,
			       p.direct_reply_count, p.total_reply_count,
			       m.id, m.media_url, m.media_type, m.created_at
			FROM posts p
			LEFT JOIN users u ON p.author_did = u.did
			LEFT JOIN media m ON p.id = m.post_id
			WHERE p.content ~* $1
			  AND p.deleted_at IS NULL
			  AND p.visibility = 'public'
			  AND (p.expires_at IS NULL OR p.expires_at > NOW())
			ORDER BY p.created_at DESC
			LIMIT $2 OFFSET $3
		`
		args = []interface{}{pattern, limit, offset}
	}

	rows, err := db.GetDB().Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get posts by hashtag: %w", err)
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
			&post.ExpiresAt,
			&post.Username,
			&post.AvatarURL,
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

	if posts == nil {
		posts = []*models.Post{}
	}

	return posts, nil
}

// SearchHashtags searches for hashtags matching a query
func (r *PostRepository) SearchHashtags(ctx context.Context, query string, limit int) ([]TrendingHashtag, error) {
	sqlQuery := `
		SELECT tag, COUNT(*) as cnt
		FROM (
			SELECT LOWER(match[1]) as tag
			FROM posts,
			     LATERAL regexp_matches(content, '#([A-Za-z0-9_]+)', 'g') AS match
			WHERE deleted_at IS NULL
			  AND visibility = 'public'
			  AND (expires_at IS NULL OR expires_at > NOW())
		) hashtags
		WHERE tag ILIKE $1
		GROUP BY tag
		ORDER BY cnt DESC, tag ASC
		LIMIT $2
	`

	searchPattern := "%" + query + "%"
	rows, err := db.GetDB().Query(ctx, sqlQuery, searchPattern, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search hashtags: %w", err)
	}
	defer rows.Close()

	var results []TrendingHashtag
	for rows.Next() {
		var h TrendingHashtag
		if err := rows.Scan(&h.Tag, &h.Count); err != nil {
			return nil, fmt.Errorf("failed to scan hashtag: %w", err)
		}
		results = append(results, h)
	}

	if results == nil {
		results = []TrendingHashtag{}
	}

	return results, nil
}

// CountPostsByHashtag returns the total number of public posts containing a given hashtag
func (r *PostRepository) CountPostsByHashtag(ctx context.Context, tag string) (int, error) {
	pattern := `#` + tag + `\y`
	var count int
	err := db.GetDB().QueryRow(ctx, `
		SELECT COUNT(*)
		FROM posts
		WHERE content ~* $1
		  AND deleted_at IS NULL
		  AND visibility = 'public'
		  AND (expires_at IS NULL OR expires_at > NOW())
	`, pattern).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count posts by hashtag: %w", err)
	}
	return count, nil
}

// postVisibilitySQL is the single audience predicate for reads and writes.
// Both arguments are fixed SQL fragments supplied only by this package.
func postVisibilitySQL(alias, viewer string) string {
	return fmt.Sprintf(`(%[1]s.deleted_at IS NULL AND (%[1]s.expires_at IS NULL OR %[1]s.expires_at > NOW()) AND (
 %[1]s.visibility = 'public' OR (%[2]s <> '' AND EXISTS(SELECT 1 FROM users active_viewer WHERE active_viewer.did=%[2]s AND NOT active_viewer.is_suspended) AND (
 %[1]s.author_did = %[2]s
 OR (%[1]s.visibility = 'followers' AND EXISTS (SELECT 1 FROM follows vf WHERE vf.follower_did = %[2]s AND vf.following_did = %[1]s.author_did AND vf.status = 'accepted'))
 OR (%[1]s.is_remote AND EXISTS(SELECT 1 FROM remote_post_recipients rp WHERE rp.post_id=%[1]s.id AND rp.recipient_did=%[2]s))
 OR (%[1]s.visibility = 'circle' AND EXISTS (SELECT 1 FROM circle_members vc JOIN users vo ON vo.id=vc.owner_id JOIN users vv ON vv.id=vc.member_id WHERE vo.did=%[1]s.author_did AND vv.did=%[2]s))
 ))))`, alias, viewer)
}

// CanView checks an existing, live post using the current accepted audience.
func (r *PostRepository) CanView(ctx context.Context, id, viewerDID string) (bool, error) {
	var allowed bool
	err := db.GetDB().QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM posts p WHERE p.id=$1 AND `+postVisibilitySQL("p", "$2")+`)`, id, viewerDID).Scan(&allowed)
	return allowed, err
}

// CanViewTarget also resolves a reply to its root post; replies inherit that audience.
func (r *PostRepository) CanViewTarget(ctx context.Context, id, viewerDID string) (bool, error) {
	var allowed bool
	err := db.GetDB().QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM posts p WHERE (p.id=$1 OR p.id IN (SELECT post_id FROM replies WHERE id=$1 AND deleted_at IS NULL)) AND `+postVisibilitySQL("p", "$2")+`)`, id, viewerDID).Scan(&allowed)
	return allowed, err
}
func (r *PostRepository) GetByIDForViewer(ctx context.Context, id, viewerDID string) (*models.Post, error) {
	return r.getByID(ctx, id, viewerDID, true)
}
func (r *PostRepository) GetMediaContentForViewer(ctx context.Context, id, viewerDID string) ([]byte, string, string, error) {
	var data []byte
	var kind, legacyURL string
	err := db.GetDB().QueryRow(ctx, `SELECT COALESCE(m.media_data,''::bytea),COALESCE(m.media_type,''),COALESCE(m.media_url,'') FROM media m JOIN posts p ON p.id=m.post_id WHERE m.id=$1 AND `+postVisibilitySQL("p", "$2"), id, viewerDID).Scan(&data, &kind, &legacyURL)
	if err != nil {
		return nil, "", "", fmt.Errorf("media not found")
	}
	return data, kind, legacyURL, nil
}

func (r *PostRepository) GetLegacyMediaURL(ctx context.Context, path, viewerDID string) ([]byte, string, string, error) {
	var id string
	err := db.GetDB().QueryRow(ctx, `SELECT m.id FROM media m JOIN posts p ON p.id=m.post_id WHERE m.media_url=$1 AND `+postVisibilitySQL("p", "$2")+` LIMIT 1`, path, viewerDID).Scan(&id)
	if err != nil {
		return nil, "", "", fmt.Errorf("media not found")
	}
	return r.GetMediaContentForViewer(ctx, id, viewerDID)
}
