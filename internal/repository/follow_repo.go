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
	FollowerDID  string `json:"follower_did"`
	FollowingDID string `json:"following_did"`
	Status       string `json:"status"` // pending, accepted, rejected
	CreatedAt    string `json:"created_at"`
}

// Create creates a new follow relationship
func (r *FollowRepository) Create(ctx context.Context, followerDID, followingDID string) (*Follow, error) {
	// Check if already following
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM follows WHERE follower_did = $1 AND following_did = $2)`
	err := db.GetDB().QueryRow(ctx, checkQuery, followerDID, followingDID).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing follow: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("already following this user")
	}

	// Create follow relationship (auto-accepted for now, can add approval logic later)
	query := `
		INSERT INTO follows (follower_did, following_did, status)
		VALUES ($1, $2, 'accepted')
		RETURNING follower_did, following_did, status, created_at
	`

	var follow Follow
	err = db.GetDB().QueryRow(ctx, query, followerDID, followingDID).Scan(
		&follow.FollowerDID,
		&follow.FollowingDID,
		&follow.Status,
		&follow.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create follow: %w", err)
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
	query := `
		SELECT u.id, u.username, u.instance_domain, u.did, u.display_name, 
		       u.bio, u.avatar_url, u.public_key, u.is_locked, u.is_suspended,
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
	query := `
		SELECT u.id, u.username, u.instance_domain, u.did, u.display_name, 
		       u.bio, u.avatar_url, u.public_key, u.is_locked, u.is_suspended,
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
