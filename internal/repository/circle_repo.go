package repository

import (
	"context"
	"fmt"

	"splitter/internal/db"
	"splitter/internal/models"
)

// CircleRepository handles database operations for circle members.
type CircleRepository struct{}

// NewCircleRepository creates a new CircleRepository.
func NewCircleRepository() *CircleRepository {
	return &CircleRepository{}
}

// GetCircleMembers returns all circle members for a given owner (by UUID).
func (r *CircleRepository) GetCircleMembers(ctx context.Context, ownerID string) ([]*models.User, error) {
	query := `
		SELECT u.id, u.username, COALESCE(u.email, ''), u.instance_domain,
		       COALESCE(u.did, ''), COALESCE(u.display_name, ''), COALESCE(u.bio, ''),
		       COALESCE(u.avatar_url, ''), COALESCE(u.public_key, ''),
		       COALESCE(u.encryption_public_key, ''), COALESCE(u.role, 'user'),
		       COALESCE(u.moderation_requested, false), u.is_locked, u.is_suspended,
		       u.created_at, u.updated_at
		FROM circle_members cm
		JOIN users u ON cm.member_id = u.id
		WHERE cm.owner_id = $1
		ORDER BY cm.created_at DESC
	`

	rows, err := db.GetDB().Query(ctx, query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get circle members: %w", err)
	}
	defer rows.Close()

	var members []*models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(
			&u.ID,
			&u.Username,
			&u.Email,
			&u.InstanceDomain,
			&u.DID,
			&u.DisplayName,
			&u.Bio,
			&u.AvatarURL,
			&u.PublicKey,
			&u.EncryptionPublicKey,
			&u.Role,
			&u.ModerationRequested,
			&u.IsLocked,
			&u.IsSuspended,
			&u.CreatedAt,
			&u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan circle member: %w", err)
		}
		members = append(members, &u)
	}

	return members, nil
}

// AddCircleMember adds a user to the owner's circle.
// ownerID and memberID are both UUIDs.
// Returns an error if the member is already in the circle (UNIQUE constraint).
func (r *CircleRepository) AddCircleMember(ctx context.Context, ownerID, memberID string) error {
	_, err := db.GetDB().Exec(ctx,
		`INSERT INTO circle_members (owner_id, member_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		ownerID, memberID,
	)
	if err != nil {
		return fmt.Errorf("failed to add circle member: %w", err)
	}
	return nil
}

// RemoveCircleMember removes a user from the owner's circle.
func (r *CircleRepository) RemoveCircleMember(ctx context.Context, ownerID, memberID string) error {
	_, err := db.GetDB().Exec(ctx,
		`DELETE FROM circle_members WHERE owner_id = $1 AND member_id = $2`,
		ownerID, memberID,
	)
	if err != nil {
		return fmt.Errorf("failed to remove circle member: %w", err)
	}
	return nil
}

// IsInCircle returns true if memberID is in ownerID's circle.
func (r *CircleRepository) IsInCircle(ctx context.Context, ownerID, memberID string) (bool, error) {
	var exists bool
	err := db.GetDB().QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM circle_members WHERE owner_id = $1 AND member_id = $2)`,
		ownerID, memberID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check circle membership: %w", err)
	}
	return exists, nil
}
