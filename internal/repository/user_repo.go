package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"time"

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

// Create creates a new user in the database with password
func (r *UserRepository) Create(ctx context.Context, user *models.UserCreate, passwordHash string) (*models.User, error) {
	query := `
		INSERT INTO users (username, email, password_hash, instance_domain, did, display_name, bio, avatar_url, public_key, encryption_public_key, role)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'user')
		RETURNING id, username, COALESCE(email, ''), instance_domain, COALESCE(did, ''), display_name, COALESCE(bio, ''), COALESCE(avatar_url, ''), COALESCE(public_key, ''), COALESCE(encryption_public_key, ''), COALESCE(role, 'user'), COALESCE(moderation_requested, false), is_locked, is_suspended, created_at, updated_at
	`

	instanceDomain := user.InstanceDomain
	if instanceDomain == "" {
		instanceDomain = "localhost"
	}

	var newUser models.User
	err := db.GetDB().QueryRow(ctx, query,
		user.Username,
		user.Email,
		passwordHash,
		instanceDomain,
		user.DID,
		user.DisplayName,
		user.Bio,
		user.AvatarURL,
		user.PublicKey,
		user.EncryptionPublicKey,
	).Scan(
		&newUser.ID,
		&newUser.Username,
		&newUser.Email,
		&newUser.InstanceDomain,
		&newUser.DID,
		&newUser.DisplayName,
		&newUser.Bio,
		&newUser.AvatarURL,
		&newUser.PublicKey,
		&newUser.EncryptionPublicKey,
		&newUser.Role,
		&newUser.ModerationRequested,
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
		SELECT id, username, COALESCE(email, ''), instance_domain, COALESCE(did, ''), display_name, COALESCE(bio, ''), COALESCE(avatar_url, ''), COALESCE(public_key, ''), COALESCE(encryption_public_key, ''), COALESCE(role, 'user'), COALESCE(moderation_requested, false), is_locked, is_suspended, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := db.GetDB().QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.InstanceDomain,
		&user.DID,
		&user.DisplayName,
		&user.Bio,
		&user.AvatarURL,
		&user.PublicKey,
		&user.EncryptionPublicKey,
		&user.Role,
		&user.ModerationRequested,
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

// GetByUsername retrieves a user by username (for login)
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, string, error) {
	query := `
		SELECT id, username, COALESCE(email, ''), COALESCE(password_hash, ''), instance_domain, COALESCE(did, ''), display_name, COALESCE(bio, ''), COALESCE(avatar_url, ''), COALESCE(public_key, ''), COALESCE(encryption_public_key, ''), COALESCE(role, 'user'), COALESCE(moderation_requested, false), is_locked, is_suspended, created_at, updated_at
		FROM users
		WHERE username = $1 OR email = $1
	`

	var user models.User
	var passwordHash string
	err := db.GetDB().QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&passwordHash,
		&user.InstanceDomain,
		&user.DID,
		&user.DisplayName,
		&user.Bio,
		&user.AvatarURL,
		&user.PublicKey,
		&user.EncryptionPublicKey,
		&user.Role,
		&user.ModerationRequested,
		&user.IsLocked,
		&user.IsSuspended,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, "", fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user: %w", err)
	}

	return &user, passwordHash, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, username, COALESCE(email, ''), instance_domain, COALESCE(did, ''), display_name, COALESCE(bio, ''), COALESCE(avatar_url, ''), COALESCE(public_key, ''), COALESCE(encryption_public_key, ''), COALESCE(role, 'user'), COALESCE(moderation_requested, false), is_locked, is_suspended, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user models.User
	err := db.GetDB().QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.InstanceDomain,
		&user.DID,
		&user.DisplayName,
		&user.Bio,
		&user.AvatarURL,
		&user.PublicKey,
		&user.EncryptionPublicKey,
		&user.Role,
		&user.ModerationRequested,
		&user.IsLocked,
		&user.IsSuspended,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

// GetByDID retrieves a user by DID (Decentralized Identifier)
func (r *UserRepository) GetByDID(ctx context.Context, did string) (*models.User, error) {
	query := `
		SELECT id, username, COALESCE(email, ''), instance_domain, COALESCE(did, ''), display_name, COALESCE(bio, ''), COALESCE(avatar_url, ''), COALESCE(public_key, ''), COALESCE(encryption_public_key, ''), COALESCE(role, 'user'), COALESCE(moderation_requested, false), is_locked, is_suspended, created_at, updated_at
		FROM users
		WHERE did = $1
	`

	var user models.User
	err := db.GetDB().QueryRow(ctx, query, did).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.InstanceDomain,
		&user.DID,
		&user.DisplayName,
		&user.Bio,
		&user.AvatarURL,
		&user.PublicKey,
		&user.EncryptionPublicKey,
		&user.Role,
		&user.ModerationRequested,
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
		RETURNING id, username, COALESCE(email, ''), instance_domain, COALESCE(did, ''), display_name, COALESCE(bio, ''), COALESCE(avatar_url, ''), COALESCE(public_key, ''), COALESCE(encryption_public_key, ''), is_locked, is_suspended, created_at, updated_at
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
		&user.Email,
		&user.InstanceDomain,
		&user.DID,
		&user.DisplayName,
		&user.Bio,
		&user.AvatarURL,
		&user.PublicKey,
		&user.EncryptionPublicKey,
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
	tx, err := db.GetDB().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var did string
	err = tx.QueryRow(ctx, `SELECT did FROM users WHERE id = $1`, id).Scan(&did)
	if err == pgx.ErrNoRows {
		return fmt.Errorf("user not found")
	}
	if err != nil {
		return fmt.Errorf("failed to load user DID: %w", err)
	}

	if _, err := tx.Exec(ctx, `UPDATE posts SET deleted_at = NOW() WHERE author_did = $1 AND deleted_at IS NULL`, did); err != nil {
		return fmt.Errorf("failed to soft-delete user posts: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM interactions WHERE actor_did = $1`, did); err != nil {
		return fmt.Errorf("failed to remove user interactions: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM follows WHERE follower_did = $1 OR following_did = $1`, did); err != nil {
		return fmt.Errorf("failed to remove user follows: %w", err)
	}

	result, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit user deletion: %w", err)
	}

	return nil
}

// UsernameExists checks if a username is already taken
func (r *UserRepository) UsernameExists(ctx context.Context, username string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`
	var exists bool
	err := db.GetDB().QueryRow(ctx, query, username).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check username: %w", err)
	}
	return exists, nil
}

// EmailExists checks if an email is already taken
func (r *UserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	var exists bool
	err := db.GetDB().QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check email: %w", err)
	}
	return exists, nil
}

// SearchUsers searches for users by username or display name
func (r *UserRepository) SearchUsers(ctx context.Context, searchTerm string, limit, offset int) ([]*models.User, error) {
	query := `
		SELECT id, username, COALESCE(email, ''), instance_domain, COALESCE(did, ''), display_name, COALESCE(bio, ''), COALESCE(avatar_url, ''), COALESCE(public_key, ''), COALESCE(encryption_public_key, ''), COALESCE(role, 'user'), COALESCE(moderation_requested, false), is_locked, is_suspended, created_at, updated_at
		FROM users
		WHERE (username ILIKE $1 OR display_name ILIKE $1 OR instance_domain ILIKE $1)
		AND is_suspended = false
		ORDER BY username ASC
		LIMIT $2 OFFSET $3
	`

	searchPattern := "%" + searchTerm + "%"
	rows, err := db.GetDB().Query(ctx, query, searchPattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.InstanceDomain,
			&user.DID,
			&user.DisplayName,
			&user.Bio,
			&user.AvatarURL,
			&user.PublicKey,
			&user.EncryptionPublicKey,
			&user.Role,
			&user.ModerationRequested,
			&user.IsLocked,
			&user.IsSuspended,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	return users, nil
}

// GetAllUsers returns all users (admin only)
func (r *UserRepository) GetAllUsers(ctx context.Context, limit, offset int) ([]*models.User, int, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM users`
	var total int
	err := db.GetDB().QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	query := `
		SELECT id, username, COALESCE(email, ''), instance_domain, COALESCE(did, ''), display_name, COALESCE(bio, ''), COALESCE(avatar_url, ''), COALESCE(public_key, ''), COALESCE(encryption_public_key, ''), COALESCE(role, 'user'), COALESCE(moderation_requested, false), is_locked, is_suspended, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := db.GetDB().Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get all users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.InstanceDomain,
			&user.DID,
			&user.DisplayName,
			&user.Bio,
			&user.AvatarURL,
			&user.PublicKey,
			&user.EncryptionPublicKey,
			&user.Role,
			&user.ModerationRequested,
			&user.IsLocked,
			&user.IsSuspended,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	return users, total, nil
}

// UpdateUserRole updates a user's role (admin only)
func (r *UserRepository) UpdateUserRole(ctx context.Context, userID, role string) error {
	query := `UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2`
	result, err := db.GetDB().Exec(ctx, query, role, userID)
	if err != nil {
		return fmt.Errorf("failed to update user role: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// RequestModeration sets the moderation_requested flag for a user
func (r *UserRepository) RequestModeration(ctx context.Context, userID string) error {
	query := `UPDATE users SET moderation_requested = true, moderation_requested_at = NOW(), updated_at = NOW() WHERE id = $1`
	result, err := db.GetDB().Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to request moderation: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// GetModerationRequests gets all pending moderation requests (admin only)
func (r *UserRepository) GetModerationRequests(ctx context.Context) ([]*models.User, error) {
	query := `
		SELECT id, username, COALESCE(email, ''), instance_domain, COALESCE(did, ''), display_name, COALESCE(bio, ''), COALESCE(avatar_url, ''), COALESCE(public_key, ''), COALESCE(encryption_public_key, ''), COALESCE(role, 'user'), COALESCE(moderation_requested, false), is_locked, is_suspended, created_at, updated_at
		FROM users
		WHERE moderation_requested = true AND role = 'user'
		ORDER BY moderation_requested_at ASC
	`

	rows, err := db.GetDB().Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get moderation requests: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.InstanceDomain,
			&user.DID,
			&user.DisplayName,
			&user.Bio,
			&user.AvatarURL,
			&user.PublicKey,
			&user.EncryptionPublicKey,
			&user.Role,
			&user.ModerationRequested,
			&user.IsLocked,
			&user.IsSuspended,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	return users, nil
}

// ApproveModerationRequest approves a user's moderation request
func (r *UserRepository) ApproveModerationRequest(ctx context.Context, userID string) error {
	query := `UPDATE users SET role = 'moderator', moderation_requested = false, updated_at = NOW() WHERE id = $1`
	result, err := db.GetDB().Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to approve moderation: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// RejectModerationRequest rejects a user's moderation request
func (r *UserRepository) RejectModerationRequest(ctx context.Context, userID string) error {
	query := `UPDATE users SET moderation_requested = false, updated_at = NOW() WHERE id = $1`
	result, err := db.GetDB().Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to reject moderation: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// SuspendUser suspends a user (admin/moderator only)
func (r *UserRepository) SuspendUser(ctx context.Context, userID string) error {
	query := `UPDATE users SET is_suspended = true, updated_at = NOW() WHERE id = $1`
	result, err := db.GetDB().Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to suspend user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// UnsuspendUser unsuspends a user (admin/moderator only)
func (r *UserRepository) UnsuspendUser(ctx context.Context, userID string) error {
	query := `UPDATE users SET is_suspended = false, updated_at = NOW() WHERE id = $1`
	result, err := db.GetDB().Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to unsuspend user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// GetSuspendedUsers returns all suspended users
func (r *UserRepository) GetSuspendedUsers(ctx context.Context, limit, offset int) ([]*models.User, error) {
	query := `
		SELECT id, username, COALESCE(email, ''), instance_domain, COALESCE(did, ''), display_name, COALESCE(bio, ''), COALESCE(avatar_url, ''), 
		       COALESCE(public_key, ''), COALESCE(encryption_public_key, ''), COALESCE(role, 'user'), COALESCE(moderation_requested, false), 
		       is_locked, is_suspended, created_at, updated_at
		FROM users
		WHERE is_suspended = true
		ORDER BY updated_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := db.GetDB().Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get suspended users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.InstanceDomain,
			&user.DID,
			&user.DisplayName,
			&user.Bio,
			&user.AvatarURL,
			&user.PublicKey,
			&user.EncryptionPublicKey,
			&user.Role,
			&user.ModerationRequested,
			&user.IsLocked,
			&user.IsSuspended,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			continue
		}
		users = append(users, &user)
	}

	if users == nil {
		users = []*models.User{}
	}

	return users, nil
}

// UpdateEncryptionKey updates a user's encryption public key
// This allows existing users without keys to add them
func (r *UserRepository) UpdateEncryptionKey(ctx context.Context, userID, encryptionPublicKey string) error {
	query := `UPDATE users SET encryption_public_key = $1, updated_at = NOW() WHERE id = $2`
	result, err := db.GetDB().Exec(ctx, query, encryptionPublicKey, userID)
	if err != nil {
		return fmt.Errorf("failed to update encryption key: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// UpdateAvatar stores avatar binary data in DB and updates avatar URL.
func (r *UserRepository) UpdateAvatar(ctx context.Context, userID string, avatarData []byte, mediaType string) (*models.User, error) {
	avatarURL := fmt.Sprintf("/api/v1/users/%s/avatar", userID)
	query := `
		UPDATE users
		SET avatar_data = $1,
		    avatar_media_type = $2,
		    avatar_url = $3,
		    updated_at = NOW()
		WHERE id = $4
		RETURNING id, username, COALESCE(email, ''), instance_domain, COALESCE(did, ''), display_name, COALESCE(bio, ''), COALESCE(avatar_url, ''), COALESCE(public_key, ''), COALESCE(encryption_public_key, ''), COALESCE(role, 'user'), COALESCE(moderation_requested, false), is_locked, is_suspended, created_at, updated_at
	`

	var user models.User
	err := db.GetDB().QueryRow(ctx, query, avatarData, mediaType, avatarURL, userID).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.InstanceDomain,
		&user.DID,
		&user.DisplayName,
		&user.Bio,
		&user.AvatarURL,
		&user.PublicKey,
		&user.EncryptionPublicKey,
		&user.Role,
		&user.ModerationRequested,
		&user.IsLocked,
		&user.IsSuspended,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update avatar: %w", err)
	}

	return &user, nil
}

// GetAvatarContentByUserID retrieves avatar bytes, type and URL for a user.
func (r *UserRepository) GetAvatarContentByUserID(ctx context.Context, userID string) ([]byte, string, string, error) {
	query := `
		SELECT COALESCE(avatar_data, ''::bytea), COALESCE(avatar_media_type, ''), COALESCE(avatar_url, '')
		FROM users
		WHERE id = $1
	`

	var avatarData []byte
	var mediaType, avatarURL string
	err := db.GetDB().QueryRow(ctx, query, userID).Scan(&avatarData, &mediaType, &avatarURL)
	if err == pgx.ErrNoRows {
		return nil, "", "", fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to get avatar: %w", err)
	}

	return avatarData, mediaType, avatarURL, nil
}

// RotateKey records a key rotation: archives the old key into key_rotations and
// updates users.public_key to the new key. Both writes happen in a single transaction.
func (r *UserRepository) RotateKey(ctx context.Context, userID, oldPublicKey, newPublicKey, nonce, reason string) error {
	if reason == "" {
		reason = "rotated"
	}
	tx, err := db.GetDB().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert into revocation list
	insertQuery := `
		INSERT INTO key_rotations (user_id, old_public_key, new_public_key, nonce, reason)
		VALUES ($1, $2, $3, $4, $5)
	`
	if _, err := tx.Exec(ctx, insertQuery, userID, oldPublicKey, newPublicKey, nonce, reason); err != nil {
		return fmt.Errorf("failed to record key rotation: %w", err)
	}

	// Update active key on user
	updateQuery := `UPDATE users SET public_key = $1, updated_at = NOW() WHERE id = $2`
	result, err := tx.Exec(ctx, updateQuery, newPublicKey, userID)
	if err != nil {
		return fmt.Errorf("failed to update user public key: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return tx.Commit(ctx)
}

// GetKeyHistory returns all key rotation records for a user, newest first.
func (r *UserRepository) GetKeyHistory(ctx context.Context, userID string) ([]*models.KeyRotation, error) {
	query := `
		SELECT id, user_id, old_public_key, new_public_key, rotated_at, nonce,
		       COALESCE(reason, 'rotated')
		FROM key_rotations
		WHERE user_id = $1
		ORDER BY rotated_at DESC
	`
	rows, err := db.GetDB().Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get key history: %w", err)
	}
	defer rows.Close()

	var rotations []*models.KeyRotation
	for rows.Next() {
		var kr models.KeyRotation
		if err := rows.Scan(&kr.ID, &kr.UserID, &kr.OldPublicKey, &kr.NewPublicKey, &kr.RotatedAt, &kr.Nonce, &kr.Reason); err != nil {
			return nil, fmt.Errorf("failed to scan key rotation: %w", err)
		}
		kr.IsRevoked = true
		rotations = append(rotations, &kr)
	}
	if rotations == nil {
		rotations = []*models.KeyRotation{}
	}
	return rotations, nil
}

// GetKeyHistoryByDID returns key rotation records for a given DID (public endpoint).
func (r *UserRepository) GetKeyHistoryByDID(ctx context.Context, did string) ([]*models.KeyRotation, error) {
	query := `
		SELECT kr.id, kr.user_id, kr.old_public_key, kr.new_public_key, kr.rotated_at, kr.nonce,
		       COALESCE(kr.reason, 'rotated')
		FROM key_rotations kr
		JOIN users u ON u.id = kr.user_id
		WHERE u.did = $1
		ORDER BY kr.rotated_at DESC
	`
	rows, err := db.GetDB().Query(ctx, query, did)
	if err != nil {
		return nil, fmt.Errorf("failed to get key history by DID: %w", err)
	}
	defer rows.Close()

	var rotations []*models.KeyRotation
	for rows.Next() {
		var kr models.KeyRotation
		if err := rows.Scan(&kr.ID, &kr.UserID, &kr.OldPublicKey, &kr.NewPublicKey, &kr.RotatedAt, &kr.Nonce, &kr.Reason); err != nil {
			return nil, fmt.Errorf("failed to scan key rotation: %w", err)
		}
		kr.IsRevoked = true
		rotations = append(rotations, &kr)
	}
	if rotations == nil {
		rotations = []*models.KeyRotation{}
	}
	return rotations, nil
}

// IsKeyRevoked returns true if the given public key appears in the key_rotations revocation list.
func (r *UserRepository) IsKeyRevoked(ctx context.Context, publicKey string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM key_rotations WHERE old_public_key = $1)`
	var revoked bool
	if err := db.GetDB().QueryRow(ctx, query, publicKey).Scan(&revoked); err != nil {
		return false, fmt.Errorf("failed to check key revocation: %w", err)
	}
	return revoked, nil
}

// IsNonceUsed returns true if the given nonce has already been recorded in key_rotations.
// Used to prevent rotation request replay attacks.
func (r *UserRepository) IsNonceUsed(ctx context.Context, nonce string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM key_rotations WHERE nonce = $1)`
	var used bool
	if err := db.GetDB().QueryRow(ctx, query, nonce).Scan(&used); err != nil {
		return false, fmt.Errorf("failed to check nonce: %w", err)
	}
	return used, nil
}

// RevokeCurrentKey archives the user's active key into key_rotations and
// removes it from the users profile. Both happen in a single transaction.
func (r *UserRepository) RevokeCurrentKey(ctx context.Context, userID, oldKey string) error {
	tx, err := db.GetDB().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Move to revocation list
	// Generate a unique nonce to avoid DB constraint violations
	nonceBytes := make([]byte, 8)
	rand.Read(nonceBytes)
	nonce := fmt.Sprintf("revoke_%d_%s", time.Now().UnixNano(), hex.EncodeToString(nonceBytes))

	insertQuery := `
		INSERT INTO key_rotations (user_id, old_public_key, new_public_key, nonce, reason)
		VALUES ($1, $2, $3, $4, $5)
	`
	// new_public_key is empty for a manual revocation with no replacement
	if _, err := tx.Exec(ctx, insertQuery, userID, oldKey, "", nonce, "manually_revoked"); err != nil {
		log.Printf("[UserRepository] RevokeCurrentKey: failed to archive key for user %s: %v", userID, err)
		return fmt.Errorf("failed to archive key: %w", err)
	}

	// 2. Clear from user profile
	updateQuery := `UPDATE users SET public_key = NULL, updated_at = NOW() WHERE id = $1`
	if _, err := tx.Exec(ctx, updateQuery, userID); err != nil {
		log.Printf("[UserRepository] RevokeCurrentKey: failed to clear public_key for user %s: %v", userID, err)
		return fmt.Errorf("failed to clear user key: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		log.Printf("[UserRepository] RevokeCurrentKey: failed to commit transaction for user %s: %v", userID, err)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
