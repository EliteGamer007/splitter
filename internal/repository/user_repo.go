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

// Create creates a new user in the database
func (r *UserRepository) Create(ctx context.Context, user *models.UserCreate) (*models.User, error) {
	query := `
		INSERT INTO users (username, email, password, full_name)
		VALUES ($1, $2, $3, $4)
		RETURNING id, username, email, full_name, bio, avatar_url, created_at, updated_at
	`

	var newUser models.User
	err := db.GetDB().QueryRow(ctx, query,
		user.Username,
		user.Email,
		user.Password, // Should be hashed before calling this
		user.FullName,
	).Scan(
		&newUser.ID,
		&newUser.Username,
		&newUser.Email,
		&newUser.FullName,
		&newUser.Bio,
		&newUser.AvatarURL,
		&newUser.CreatedAt,
		&newUser.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &newUser, nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id int) (*models.User, error) {
	query := `
		SELECT id, username, email, full_name, bio, avatar_url, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := db.GetDB().QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.FullName,
		&user.Bio,
		&user.AvatarURL,
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

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, username, email, password, full_name, bio, avatar_url, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user models.User
	err := db.GetDB().QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password, // Include password for authentication
		&user.FullName,
		&user.Bio,
		&user.AvatarURL,
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

// Update updates a user's information
func (r *UserRepository) Update(ctx context.Context, id int, update *models.UserUpdate) (*models.User, error) {
	query := `
		UPDATE users
		SET 
			full_name = COALESCE($1, full_name),
			bio = COALESCE($2, bio),
			avatar_url = COALESCE($3, avatar_url),
			updated_at = NOW()
		WHERE id = $4
		RETURNING id, username, email, full_name, bio, avatar_url, created_at, updated_at
	`

	var user models.User
	err := db.GetDB().QueryRow(ctx, query,
		update.FullName,
		update.Bio,
		update.AvatarURL,
		id,
	).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.FullName,
		&user.Bio,
		&user.AvatarURL,
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

// Delete deletes a user by ID
func (r *UserRepository) Delete(ctx context.Context, id int) error {
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
