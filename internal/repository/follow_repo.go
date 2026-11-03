package repository

import (
	"context"
	"fmt"

	"splitter/internal/db"
	"splitter/internal/models"

	"github.com/jackc/pgx/v5"
)

// FollowRepository handles database operations for follow relationships
type FollowRepository struct{}

// NewFollowRepository creates a new FollowRepository
func NewFollowRepository() *FollowRepository {
	return &FollowRepository{}
}

// Follow represents a follow relationship with DID-based identification
type Follow struct {
	ID           string `json:"id"`
	FollowerDID  string `json:"follower_did"`
	FollowingDID string `json:"following_did"`
	Status       string `json:"status"` // pending, accepted, rejected
	CreatedAt    string `json:"created_at"`
}

// Create creates a new follow relationship
func (r *FollowRepository) Create(ctx context.Context, followerDID, followingDID string) (*Follow, error) {
	if followerDID == followingDID {
		return nil, fmt.Errorf("cannot follow yourself")
	}
	tx, err := db.GetDB().Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	var locked, suspended bool
	// Serialize requests with account locking changes and concurrent duplicate follows.
	err = tx.QueryRow(ctx, `SELECT is_locked,is_suspended FROM users WHERE did=$1 FOR UPDATE`, followingDID).Scan(&locked, &suspended)
	if err != nil || suspended {
		return nil, fmt.Errorf("target unavailable")
	}
	var senderActive bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE did=$1 AND NOT is_suspended)`, followerDID).Scan(&senderActive); err != nil || !senderActive {
		return nil, fmt.Errorf("sender unavailable")
	}
	var follow Follow
	err = tx.QueryRow(ctx, `SELECT id,follower_did,following_did,status,created_at::text FROM follows WHERE follower_did=$1 AND following_did=$2 ORDER BY created_at DESC LIMIT 1`, followerDID, followingDID).Scan(&follow.ID, &follow.FollowerDID, &follow.FollowingDID, &follow.Status, &follow.CreatedAt)
	if err != nil && err != pgx.ErrNoRows {
		return nil, err
	}
	if err == nil && follow.Status == "rejected" {
		if _, err = tx.Exec(ctx, `DELETE FROM follows WHERE follower_did=$1 AND following_did=$2 AND status='rejected'`, followerDID, followingDID); err != nil {
			return nil, err
		}
		err = pgx.ErrNoRows
	}
	if err == nil {
		if err = tx.Commit(ctx); err != nil {
			return nil, err
		}
		return &follow, nil
	}
	status := "accepted"
	if locked {
		status = "pending"
	}
	err = tx.QueryRow(ctx, `INSERT INTO follows(follower_did,following_did,status) VALUES($1,$2,$3) RETURNING id,follower_did,following_did,status,created_at::text`, followerDID, followingDID, status).Scan(&follow.ID, &follow.FollowerDID, &follow.FollowingDID, &follow.Status, &follow.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &follow, nil
}

// Delete removes a follow relationship
func (r *FollowRepository) Delete(ctx context.Context, followerDID, followingDID string) error {
	query := `DELETE FROM follows WHERE follower_did = $1 AND following_did = $2`

	result, err := db.GetDB().Exec(ctx, query, followerDID, followingDID)
	if err != nil {
		return fmt.Errorf("failed to delete follow: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("follow relationship not found")
	}

	return nil
}

// GetFollowers retrieves users following a specific user
func (r *FollowRepository) GetFollowers(ctx context.Context, userDID string, limit, offset int) ([]*models.User, error) {
	limit, offset = boundedPage(limit, offset, 20)
	query := `
		SELECT u.id, u.username, COALESCE(u.instance_domain, ''), COALESCE(u.did, ''), u.display_name,
		       COALESCE(u.bio, ''), COALESCE(u.avatar_url, ''), COALESCE(u.public_key, ''), u.is_locked, u.is_suspended,
		       u.created_at, u.updated_at
		FROM users u
		INNER JOIN follows f ON u.did = f.follower_did
		WHERE f.following_did = $1 AND f.status = 'accepted'
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.GetDB().Query(ctx, query, userDID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get followers: %w", err)
	}
	defer rows.Close()

	var followers []*models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.InstanceDomain,
			&user.DID,
			&user.DisplayName,
			&user.Bio,
			&user.AvatarURL,
			&user.PublicKey,
			&user.IsLocked,
			&user.IsSuspended,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan follower: %w", err)
		}
		followers = append(followers, &user)
	}

	return followers, nil
}

// GetFollowing retrieves users followed by a specific user
func (r *FollowRepository) GetFollowing(ctx context.Context, userDID string, limit, offset int) ([]*models.User, error) {
	limit, offset = boundedPage(limit, offset, 20)
	query := `
		SELECT u.id, u.username, COALESCE(u.instance_domain, ''), COALESCE(u.did, ''), u.display_name,
		       COALESCE(u.bio, ''), COALESCE(u.avatar_url, ''), COALESCE(u.public_key, ''), u.is_locked, u.is_suspended,
		       u.created_at, u.updated_at
		FROM users u
		INNER JOIN follows f ON u.did = f.following_did
		WHERE f.follower_did = $1 AND f.status = 'accepted'
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.GetDB().Query(ctx, query, userDID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get following: %w", err)
	}
	defer rows.Close()

	var following []*models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.InstanceDomain,
			&user.DID,
			&user.DisplayName,
			&user.Bio,
			&user.AvatarURL,
			&user.PublicKey,
			&user.IsLocked,
			&user.IsSuspended,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan following: %w", err)
		}
		following = append(following, &user)
	}

	return following, nil
}

// GetStats retrieves follow statistics for a user
func (r *FollowRepository) GetStats(ctx context.Context, userDID string) (map[string]int, error) {
	query := `
		SELECT 
			(SELECT COUNT(*) FROM follows WHERE following_did = $1 AND status = 'accepted') as followers,
			(SELECT COUNT(*) FROM follows WHERE follower_did = $1 AND status = 'accepted') as following
	`

	var followers, following int
	err := db.GetDB().QueryRow(ctx, query, userDID).Scan(&followers, &following)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return map[string]int{
		"followers": followers,
		"following": following,
	}, nil
}

// IsFollowing checks if one user follows another
func (r *FollowRepository) IsFollowing(ctx context.Context, followerDID, followingDID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM follows WHERE follower_did = $1 AND following_did = $2 AND status = 'accepted')`

	var isFollowing bool
	err := db.GetDB().QueryRow(ctx, query, followerDID, followingDID).Scan(&isFollowing)
	if err != nil && err != pgx.ErrNoRows {
		return false, fmt.Errorf("failed to check following status: %w", err)
	}

	return isFollowing, nil
}

// AcceptRequest can only approve a pending request addressed to the current owner.
func (r *FollowRepository) AcceptRequest(ctx context.Context, ownerDID, followerDID string) error {
	result, err := db.GetDB().Exec(ctx, `UPDATE follows SET status='accepted' WHERE following_did=$1 AND follower_did=$2 AND status='pending' AND EXISTS(SELECT 1 FROM users WHERE did=$1 AND NOT is_suspended)`, ownerDID, followerDID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("pending request not found")
	}
	return nil
}
func (r *FollowRepository) RejectRequest(ctx context.Context, ownerDID, followerDID string) error {
	result, err := db.GetDB().Exec(ctx, `DELETE FROM follows WHERE following_did=$1 AND follower_did=$2 AND status='pending'`, ownerDID, followerDID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("pending request not found")
	}
	return nil
}
func (r *FollowRepository) GetRequests(ctx context.Context, ownerDID string, limit, offset int) ([]*models.PublicUser, error) {
	limit, offset = boundedPage(limit, offset, 20)
	if limit < 1 || limit > 50 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := db.GetDB().Query(ctx, `SELECT u.id,u.did,u.username,COALESCE(u.display_name,''),COALESCE(u.avatar_url,'') FROM follows f JOIN users u ON u.did=f.follower_did WHERE f.following_did=$1 AND f.status='pending' AND NOT u.is_suspended ORDER BY f.created_at DESC LIMIT $2 OFFSET $3`, ownerDID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]*models.PublicUser, 0)
	for rows.Next() {
		var u models.PublicUser
		if err = rows.Scan(&u.ID, &u.DID, &u.Username, &u.DisplayName, &u.AvatarURL); err != nil {
			return nil, err
		}
		result = append(result, &u)
	}
	return result, rows.Err()
}
func (r *FollowRepository) GetStatus(ctx context.Context, followerDID, followingDID string) (string, error) {
	var status string
	err := db.GetDB().QueryRow(ctx, `SELECT status FROM follows WHERE follower_did=$1 AND following_did=$2 ORDER BY created_at DESC LIMIT 1`, followerDID, followingDID).Scan(&status)
	if err == pgx.ErrNoRows {
		return "none", nil
	}
	return status, err
}
