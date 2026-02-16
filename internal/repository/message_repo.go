package repository

import (
	"context"
	"fmt"

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
	// Check if thread exists (either direction)
	query := `
		SELECT id, participant_a_id, participant_b_id, created_at, updated_at
		FROM message_threads
		WHERE (participant_a_id = $1 AND participant_b_id = $2)
		   OR (participant_a_id = $2 AND participant_b_id = $1)
	`

	var thread models.MessageThread
	err := db.GetDB().QueryRow(ctx, query, userAID, userBID).Scan(
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
		SELECT id, thread_id, sender_id, recipient_id, content, COALESCE(ciphertext, ''), is_read, created_at
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
			SELECT id, thread_id, sender_id, recipient_id, content, COALESCE(ciphertext, ''), is_read, created_at
			FROM messages
			WHERE thread_id = $1
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
		)
		if err == nil {
			thread.LastMessage = &lastMsg
		}

		threads = append(threads, &thread)
	}

	return threads, nil
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
