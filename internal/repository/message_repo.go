package repository

import (
	"context"
	"fmt"
	"time"

	"splitter/internal/db"
	"splitter/internal/models"

	"github.com/jackc/pgx/v5"
)

// MessageRepository handles database operations for messages
type MessageRepository struct{}

// NewMessageRepository creates a new MessageRepository
func NewMessageRepository() *MessageRepository {
	return &MessageRepository{}
}

// GetOrCreateThread gets an existing thread between two users or creates one
func (r *MessageRepository) GetOrCreateThread(ctx context.Context, userAID, userBID string) (*models.MessageThread, error) {
	allowed, reason, err := r.canSendMessageToRecipient(ctx, userAID, userBID)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, fmt.Errorf("message request blocked: %s", reason)
	}

	// Check if thread exists (either direction)
	query := `
		SELECT id, participant_a_id, participant_b_id, created_at, updated_at
		FROM message_threads
		WHERE (participant_a_id = $1 AND participant_b_id = $2)
		   OR (participant_a_id = $2 AND participant_b_id = $1)
	`

	var thread models.MessageThread
	err = db.GetDB().QueryRow(ctx, query, userAID, userBID).Scan(
		&thread.ID,
		&thread.ParticipantAID,
		&thread.ParticipantBID,
		&thread.CreatedAt,
		&thread.UpdatedAt,
	)

	if err == nil {
		return &thread, nil
	}

	if err != pgx.ErrNoRows {
		return nil, fmt.Errorf("failed to check thread: %w", err)
	}

	// Create new thread
	insertQuery := `
		INSERT INTO message_threads (participant_a_id, participant_b_id)
		VALUES ($1, $2)
		RETURNING id, participant_a_id, participant_b_id, created_at, updated_at
	`

	err = db.GetDB().QueryRow(ctx, insertQuery, userAID, userBID).Scan(
		&thread.ID,
		&thread.ParticipantAID,
		&thread.ParticipantBID,
		&thread.CreatedAt,
		&thread.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create thread: %w", err)
	}

	return &thread, nil
}

func (r *MessageRepository) canSendMessageToRecipient(ctx context.Context, senderID, recipientID string) (bool, string, error) {
	var senderDID string
	if err := db.GetDB().QueryRow(ctx, `SELECT did FROM users WHERE id = $1`, senderID).Scan(&senderDID); err != nil {
		if err == pgx.ErrNoRows {
			return false, "sender not found", nil
		}
		return false, "failed to load sender", fmt.Errorf("failed to load sender DID: %w", err)
	}

	var recipientDID string
	var messagePrivacy string
	if err := db.GetDB().QueryRow(ctx,
		`SELECT did, COALESCE(NULLIF(message_privacy, ''), 'everyone')
		 FROM users
		 WHERE id = $1`,
		recipientID,
	).Scan(&recipientDID, &messagePrivacy); err != nil {
		if err == pgx.ErrNoRows {
			return false, "recipient not found", nil
		}
		return false, "failed to load recipient", fmt.Errorf("failed to load recipient privacy settings: %w", err)
	}

	switch messagePrivacy {
	case "none":
		return false, "recipient does not accept direct messages", nil
	case "followers":
		var follows bool
		err := db.GetDB().QueryRow(ctx,
			`SELECT EXISTS(
				SELECT 1 FROM follows
				WHERE follower_did = $1 AND following_did = $2 AND status = 'accepted'
			)`,
			senderDID, recipientDID,
		).Scan(&follows)
		if err != nil {
			return false, "failed to verify follower relationship", fmt.Errorf("failed to check follower relationship: %w", err)
		}
		if !follows {
			return false, "only followers can message this user", nil
		}
		return true, "", nil
	default:
		return true, "", nil
	}
}

// SendMessage sends a message in a thread
func (r *MessageRepository) SendMessage(ctx context.Context, threadID, senderID, recipientID, content, ciphertext string) (*models.Message, error) {
	query := `
		INSERT INTO messages (thread_id, sender_id, recipient_id, content, ciphertext)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, thread_id, sender_id, recipient_id, content, ciphertext, is_read, created_at
	`

	var msg models.Message
	err := db.GetDB().QueryRow(ctx, query, threadID, senderID, recipientID, content, ciphertext).Scan(
		&msg.ID,
		&msg.ThreadID,
		&msg.SenderID,
		&msg.RecipientID,
		&msg.Content,
		&msg.Ciphertext,
		&msg.IsRead,
		&msg.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Update thread timestamp
	updateQuery := `UPDATE message_threads SET updated_at = NOW() WHERE id = $1`
	_, _ = db.GetDB().Exec(ctx, updateQuery, threadID)

	return &msg, nil
}

// GetThreadMessages gets all messages in a thread
func (r *MessageRepository) GetThreadMessages(ctx context.Context, threadID string, limit, offset int) ([]*models.Message, error) {
	query := `
		SELECT id, thread_id, sender_id, recipient_id, content, COALESCE(ciphertext, ''), is_read, created_at, deleted_at, edited_at
		FROM messages
		WHERE thread_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.GetDB().Query(ctx, query, threadID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		var msg models.Message
		err := rows.Scan(
			&msg.ID,
			&msg.ThreadID,
			&msg.SenderID,
			&msg.RecipientID,
			&msg.Content,
			&msg.Ciphertext,
			&msg.IsRead,
			&msg.CreatedAt,
			&msg.DeletedAt,
			&msg.EditedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, &msg)
	}

	return messages, nil
}

// GetUserThreads gets all message threads for a user
func (r *MessageRepository) GetUserThreads(ctx context.Context, userID string) ([]*models.MessageThread, error) {
	query := `
		SELECT t.id, t.participant_a_id, t.participant_b_id, t.created_at, t.updated_at,
		       u.id, u.username, COALESCE(u.display_name, ''), COALESCE(u.avatar_url, ''), u.instance_domain, COALESCE(u.encryption_public_key, '')
		FROM message_threads t
		JOIN users u ON (
			CASE WHEN t.participant_a_id = $1 THEN t.participant_b_id ELSE t.participant_a_id END = u.id
		)
		WHERE t.participant_a_id = $1 OR t.participant_b_id = $1
		ORDER BY t.updated_at DESC
	`

	rows, err := db.GetDB().Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get threads: %w", err)
	}
	defer rows.Close()

	var threads []*models.MessageThread
	for rows.Next() {
		var thread models.MessageThread
		var otherUser models.User
		err := rows.Scan(
			&thread.ID,
			&thread.ParticipantAID,
			&thread.ParticipantBID,
			&thread.CreatedAt,
			&thread.UpdatedAt,
			&otherUser.ID,
			&otherUser.Username,
			&otherUser.DisplayName,
			&otherUser.AvatarURL,
			&otherUser.InstanceDomain,
			&otherUser.EncryptionPublicKey,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan thread: %w", err)
		}
		thread.OtherUser = &otherUser

		// Get unread count
		unreadQuery := `SELECT COUNT(*) FROM messages WHERE thread_id = $1 AND recipient_id = $2 AND is_read = false`
		_ = db.GetDB().QueryRow(ctx, unreadQuery, thread.ID, userID).Scan(&thread.UnreadCount)

		// Get last message
		lastMsgQuery := `
			SELECT id, thread_id, sender_id, recipient_id, content, COALESCE(ciphertext, ''), is_read, created_at, deleted_at, edited_at
			FROM messages
			WHERE thread_id = $1 AND deleted_at IS NULL
			ORDER BY created_at DESC
			LIMIT 1
		`
		var lastMsg models.Message
		err = db.GetDB().QueryRow(ctx, lastMsgQuery, thread.ID).Scan(
			&lastMsg.ID,
			&lastMsg.ThreadID,
			&lastMsg.SenderID,
			&lastMsg.RecipientID,
			&lastMsg.Content,
			&lastMsg.Ciphertext,
			&lastMsg.IsRead,
			&lastMsg.CreatedAt,
			&lastMsg.DeletedAt,
			&lastMsg.EditedAt,
		)
		if err == nil {
			thread.LastMessage = &lastMsg
		}

		threads = append(threads, &thread)
	}

	return threads, nil
}

// DeleteMessage soft-deletes a message (WhatsApp-style)
// Only allows deletion within 3 hours of sending
func (r *MessageRepository) DeleteMessage(ctx context.Context, messageID, userID string) error {
	// Check if message exists, user owns it, and it's within 3 hour window
	var senderID string
	var createdAt time.Time
	var alreadyDeleted bool

	checkQuery := `
		SELECT sender_id, created_at, deleted_at IS NOT NULL
		FROM messages
		WHERE id = $1
	`
	err := db.GetDB().QueryRow(ctx, checkQuery, messageID).Scan(&senderID, &createdAt, &alreadyDeleted)
	if err != nil {
		return fmt.Errorf("message not found: %w", err)
	}

	if senderID != userID {
		return fmt.Errorf("unauthorized: you can only delete your own messages")
	}

	if alreadyDeleted {
		return fmt.Errorf("message already deleted")
	}

	// Check 3-hour window
	if time.Since(createdAt) > 3*time.Hour {
		return fmt.Errorf("cannot delete messages older than 3 hours")
	}

	// Soft delete by setting deleted_at timestamp
	deleteQuery := `UPDATE messages SET deleted_at = NOW() WHERE id = $1`
	_, err = db.GetDB().Exec(ctx, deleteQuery, messageID)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	return nil
}

// EditMessage updates message content (encrypted or plain)
// Only allows editing within 3 hours of sending
func (r *MessageRepository) EditMessage(ctx context.Context, messageID, userID, newContent, newCiphertext string) error {
	// Check if message exists, user owns it, and it's within 3 hour window
	var senderID string
	var createdAt time.Time
	var deletedAt *time.Time

	checkQuery := `
		SELECT sender_id, created_at, deleted_at
		FROM messages
		WHERE id = $1
	`
	err := db.GetDB().QueryRow(ctx, checkQuery, messageID).Scan(&senderID, &createdAt, &deletedAt)
	if err != nil {
		return fmt.Errorf("message not found: %w", err)
	}

	if senderID != userID {
		return fmt.Errorf("unauthorized: you can only edit your own messages")
	}

	if deletedAt != nil {
		return fmt.Errorf("cannot edit deleted message")
	}

	// Check 3-hour window
	if time.Since(createdAt) > 3*time.Hour {
		return fmt.Errorf("cannot edit messages older than 3 hours")
	}

	// Update message content and set edited_at timestamp
	updateQuery := `
		UPDATE messages 
		SET content = $1, ciphertext = $2, edited_at = NOW() 
		WHERE id = $3
	`
	_, err = db.GetDB().Exec(ctx, updateQuery, newContent, newCiphertext, messageID)
	if err != nil {
		return fmt.Errorf("failed to edit message: %w", err)
	}

	return nil
}

// MarkMessagesAsRead marks all messages in a thread as read for a user
func (r *MessageRepository) MarkMessagesAsRead(ctx context.Context, threadID, userID string) error {
	query := `UPDATE messages SET is_read = true WHERE thread_id = $1 AND recipient_id = $2 AND is_read = false`
	_, err := db.GetDB().Exec(ctx, query, threadID, userID)
	if err != nil {
		return fmt.Errorf("failed to mark messages as read: %w", err)
	}
	return nil
}

// GetThread gets a thread by ID
func (r *MessageRepository) GetThread(ctx context.Context, threadID string) (*models.MessageThread, error) {
	query := `
		SELECT id, participant_a_id, participant_b_id, created_at, updated_at
		FROM message_threads
		WHERE id = $1
	`

	var thread models.MessageThread
	err := db.GetDB().QueryRow(ctx, query, threadID).Scan(
		&thread.ID,
		&thread.ParticipantAID,
		&thread.ParticipantBID,
		&thread.CreatedAt,
		&thread.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("thread not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	return &thread, nil
}
