package repository

import (
	"context"
	"fmt"

	"splitter/internal/db"
	"splitter/internal/models"

	"github.com/jackc/pgx/v5"
)

// ReplyRepository handles database operations for replies
type ReplyRepository struct{}

// NewReplyRepository creates a new ReplyRepository
func NewReplyRepository() *ReplyRepository {
	return &ReplyRepository{}
}

// Create creates a new reply and updates counters
func (r *ReplyRepository) Create(ctx context.Context, authorDID string, reply *models.ReplyCreate, depth int) (*models.Reply, error) {
	tx, err := db.GetDB().Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert Reply
	query := `
		INSERT INTO replies (post_id, parent_id, author_did, content, depth)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, post_id, parent_id, author_did, content, depth, likes_count, direct_reply_count, total_reply_count, created_at, updated_at
	`
	var newReply models.Reply
	err = tx.QueryRow(ctx, query,
		reply.PostID,
		reply.ParentID,
		authorDID,
		reply.Content,
		depth,
	).Scan(
		&newReply.ID,
		&newReply.PostID,
		&newReply.ParentID,
		&newReply.AuthorDID,
		&newReply.Content,
		&newReply.Depth,
		&newReply.LikesCount,
		&newReply.DirectReplyCount,
		&newReply.TotalReplyCount,
		&newReply.CreatedAt,
		&newReply.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create reply: %w", err)
	}

	// Update Root Post Counters (Always increment total_reply_count for the root post)
	// If depth is 1 (reply to post), we also increment direct_reply_count
	if depth == 1 {
		_, err = tx.Exec(ctx, `
			UPDATE posts 
			SET direct_reply_count = direct_reply_count + 1, 
			    total_reply_count = total_reply_count + 1 
			WHERE id = $1`, reply.PostID)
	} else {
		_, err = tx.Exec(ctx, `
			UPDATE posts 
			SET total_reply_count = total_reply_count + 1 
			WHERE id = $1`, reply.PostID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update post counters: %w", err)
	}

	// Update Parent Reply Counters (if parent is a reply)
	if reply.ParentID != nil {
		// Increment direct and total count for immediate parent
		_, err = tx.Exec(ctx, `
			UPDATE replies 
			SET direct_reply_count = direct_reply_count + 1, 
			    total_reply_count = total_reply_count + 1 
			WHERE id = $1`, *reply.ParentID)
		if err != nil {
			return nil, fmt.Errorf("failed to update parent reply counters: %w", err)
		}

		// Propagate total_reply_count update to all ancestors (excluding the immediate parent which is already done)
		// We can find ancestors by traversing up. Since max depth is 3, this loop is short.
		// Immediate parent is already updated. We need to update *its* parent, and so on, until we hit a top-level reply (parent_id is null).
		// Note: The root post is already updated above.

		currentParentID := *reply.ParentID
		for {
			var grandparentID *string
			err := tx.QueryRow(ctx, "SELECT parent_id FROM replies WHERE id = $1", currentParentID).Scan(&grandparentID)
			if err != nil {
				if err == pgx.ErrNoRows {
					break // Should not happen if foreign keys are correct
				}
				return nil, fmt.Errorf("failed to fetch ancestor: %w", err)
			}

			if grandparentID == nil {
				break // Reached top-level reply
			}

			// Update ancestor
			_, err = tx.Exec(ctx, `UPDATE replies SET total_reply_count = total_reply_count + 1 WHERE id = $1`, *grandparentID)
			if err != nil {
				return nil, fmt.Errorf("failed to update ancestor counters: %w", err)
			}

			currentParentID = *grandparentID
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &newReply, nil
}

// GetByPostID retrieves all replies for a post, sorted by popularity then time
func (r *ReplyRepository) GetByPostID(ctx context.Context, postID string, userDID string) ([]*models.Reply, error) {
	// If userDID is provided, check liked status
	var query string
	var args []interface{}

	if userDID != "" {
		query = `
			SELECT r.id, r.post_id, r.parent_id, r.author_did, r.content, r.depth,
			       r.likes_count, r.direct_reply_count, r.total_reply_count, r.created_at, r.updated_at,
			       u.username,
			       COALESCE((SELECT COUNT(*) > 0 FROM interactions WHERE post_id = r.id AND actor_did = $2 AND interaction_type = 'like'), false) as liked
			FROM replies r
			LEFT JOIN users u ON r.author_did = u.did
			WHERE r.post_id = $1 AND r.deleted_at IS NULL
			ORDER BY r.likes_count DESC, r.created_at ASC
		`
		args = []interface{}{postID, userDID}
	} else {
		query = `
			SELECT r.id, r.post_id, r.parent_id, r.author_did, r.content, r.depth,
			       r.likes_count, r.direct_reply_count, r.total_reply_count, r.created_at, r.updated_at,
			       u.username,
			       false as liked
			FROM replies r
			LEFT JOIN users u ON r.author_did = u.did
			WHERE r.post_id = $1 AND r.deleted_at IS NULL
			ORDER BY r.likes_count DESC, r.created_at ASC
		`
		args = []interface{}{postID}
	}

	rows, err := db.GetDB().Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get replies: %w", err)
	}
	defer rows.Close()

	var replies []*models.Reply
	for rows.Next() {
		var reply models.Reply
		err := rows.Scan(
			&reply.ID,
			&reply.PostID,
			&reply.ParentID,
			&reply.AuthorDID,
			&reply.Content,
			&reply.Depth,
			&reply.LikesCount,
			&reply.DirectReplyCount,
			&reply.TotalReplyCount,
			&reply.CreatedAt,
			&reply.UpdatedAt,
			&reply.Username,
			&reply.Liked,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reply: %w", err)
		}
		replies = append(replies, &reply)
	}

	return replies, nil
}

// GetByID retrieves a single reply by ID
func (r *ReplyRepository) GetByID(ctx context.Context, id string) (*models.Reply, error) {
	query := `
		SELECT r.id, r.post_id, r.parent_id, r.author_did, r.content, r.depth,
		       r.likes_count, r.direct_reply_count, r.total_reply_count, r.created_at, r.updated_at
		FROM replies r
		WHERE r.id = $1 AND r.deleted_at IS NULL
	`
	var reply models.Reply
	err := db.GetDB().QueryRow(ctx, query, id).Scan(
		&reply.ID,
		&reply.PostID,
		&reply.ParentID,
		&reply.AuthorDID,
		&reply.Content,
		&reply.Depth,
		&reply.LikesCount,
		&reply.DirectReplyCount,
		&reply.TotalReplyCount,
		&reply.CreatedAt,
		&reply.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("reply not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get reply: %w", err)
	}
	return &reply, nil
}
