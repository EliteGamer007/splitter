package repository

import (
	"context"
	"fmt"

	"splitter/internal/db"
	"splitter/internal/models"

	"github.com/jackc/pgx/v5"
)

// UserRepository handles database operations for users
type UserRepository struct{}

// NewUserRepository creates a new UserRepository
func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

// Create creates a new user in the database with DID
func (r *UserRepository) Create(ctx context.Context, user *models.UserCreate) (*models.User, error) {
	query := `
		INSERT INTO users (username, instance_domain, did, display_name, bio, avatar_url, public_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, username, instance_domain, did, display_name, bio, avatar_url, public_key, is_locked, is_suspended, created_at, updated_at
	`

	var newUser models.User
	err := db.GetDB().QueryRow(ctx, query,
		user.Username,
		user.InstanceDomain,
		user.DID,
		user.DisplayName,
		user.Bio,
		user.AvatarURL,
		user.PublicKey,
	).Scan(
		&newUser.ID,
		&newUser.Username,
		&newUser.InstanceDomain,
		&newUser.DID,
		&newUser.DisplayName,
		&newUser.Bio,
		&newUser.AvatarURL,
		&newUser.PublicKey,
		&newUser.IsLocked,
		&newUser.IsSuspended,
		&newUser.CreatedAt,
		&newUser.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &newUser, nil
}

// GetByID retrieves a user by UUID
func (r *UserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	query := `
		SELECT id, username, instance_domain, did, display_name, bio, avatar_url, public_key, is_locked, is_suspended, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := db.GetDB().QueryRow(ctx, query, id).Scan(
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
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// GetByDID retrieves a user by DID (Decentralized Identifier)
func (r *UserRepository) GetByDID(ctx context.Context, did string) (*models.User, error) {
	query := `
		SELECT id, username, instance_domain, did, display_name, bio, avatar_url, public_key, is_locked, is_suspended, created_at, updated_at
		FROM users
		WHERE did = $1
	`

	var user models.User
	err := db.GetDB().QueryRow(ctx, query, did).Scan(
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
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by DID: %w", err)
	}

	return &user, nil
}

// Update updates a user's information
func (r *UserRepository) Update(ctx context.Context, id string, update *models.UserUpdate) (*models.User, error) {
	query := `
		UPDATE users
		SET 
			display_name = COALESCE($1, display_name),
			bio = COALESCE($2, bio),
			avatar_url = COALESCE($3, avatar_url),
			updated_at = NOW()
		WHERE id = $4
		RETURNING id, username, instance_domain, did, display_name, bio, avatar_url, public_key, is_locked, is_suspended, created_at, updated_at
	`

	var user models.User
	err := db.GetDB().QueryRow(ctx, query,
		update.DisplayName,
		update.Bio,
		update.AvatarURL,
		id,
	).Scan(
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
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return &user, nil
}

// Delete deletes a user by UUID
func (r *UserRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = $1`

	result, err := db.GetDB().Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}
