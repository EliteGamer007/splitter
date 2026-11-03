package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"splitter/internal/db"
	"splitter/internal/models"

	"github.com/jackc/pgx/v5"
)

var ErrMessagePayload = errors.New("message exceeds 64 KiB text, 128 KiB ciphertext or 64 KiB wrapped-key limit")

var ErrMessageNotAllowed = errors.New("recipient does not accept this message")

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
		_ = reason
		return nil, ErrMessageNotAllowed
	}

	return r.getOrCreateThreadNoPolicy(ctx, userAID, userBID)
}

// GetOrCreateThreadForInbound applies the same recipient consent as local sends.
func (r *MessageRepository) GetOrCreateThreadForInbound(ctx context.Context, userAID, userBID string) (*models.MessageThread, error) {
	return r.GetOrCreateThread(ctx, userAID, userBID)
}

func (r *MessageRepository) getOrCreateThreadNoPolicy(ctx context.Context, userAID, userBID string) (*models.MessageThread, error) {

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

	tx, err := db.GetDB().Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err = db.CheckGrowthBudget(ctx, tx, 1024); err != nil {
		return nil, err
	}
	// Create new thread
	insertQuery := `
		INSERT INTO message_threads (participant_a, participant_b, participant_a_id, participant_b_id)
		SELECT ua.did, ub.did, ua.id, ub.id
		FROM users ua, users ub
		WHERE ua.id = $1 AND ub.id = $2
		RETURNING id, participant_a_id, participant_b_id, created_at, updated_at
	`

	err = tx.QueryRow(ctx, insertQuery, userAID, userBID).Scan(
		&thread.ID,
		&thread.ParticipantAID,
		&thread.ParticipantBID,
		&thread.CreatedAt,
		&thread.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create thread: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &thread, nil
}

// CanSendMessageToRecipient is shared with authenticated federation inbox processing.
func (r *MessageRepository) CanSendMessageToRecipient(ctx context.Context, senderID, recipientID string) (bool, string, error) {
	return r.canSendMessageToRecipient(ctx, senderID, recipientID)
}
func (r *MessageRepository) canSendMessageToRecipient(ctx context.Context, senderID, recipientID string) (bool, string, error) {
	if senderID == recipientID {
		return false, "cannot message yourself", nil
	}
	var allowed bool
	err := db.GetDB().QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users us,users ur WHERE us.id=$1 AND ur.id=$2 AND `+messageRecipientSQL()+`)`, senderID, recipientID).Scan(&allowed)
	if err != nil {
		return false, "unable to verify recipient", err
	}
	if !allowed {
		return false, "recipient does not accept this message", nil
	}
	return true, "", nil
}
func messageRecipientSQL() string {
	return `NOT us.is_suspended AND NOT ur.is_suspended AND (
 COALESCE(NULLIF(ur.message_privacy,''),'everyone')='everyone'
 OR (ur.message_privacy='followers' AND EXISTS(SELECT 1 FROM follows WHERE follower_did=us.did AND following_did=ur.did AND status='accepted')))`
}
func messageThreadSQL() string {
	return `EXISTS(SELECT 1 FROM message_threads mt WHERE mt.id=$1 AND ((mt.participant_a_id=us.id AND mt.participant_b_id=ur.id) OR (mt.participant_b_id=us.id AND mt.participant_a_id=ur.id)))`
}

// SendMessage sends a message in a thread
func (r *MessageRepository) SendMessage(ctx context.Context, threadID, senderID, recipientID, content, ciphertext, encryptedKeys string) (*models.Message, error) {
	if strings.TrimSpace(ciphertext) != "" {
		content = ""
	} else if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("message required")
	}
	if err := validateMessagePayload(content, ciphertext, encryptedKeys); err != nil {
		return nil, err
	}
	allowed, _, policyErr := r.canSendMessageToRecipient(ctx, senderID, recipientID)
	if policyErr != nil {
		return nil, policyErr
	}
	if !allowed {
		return nil, ErrMessageNotAllowed
	}

	tx, err := db.GetDB().Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err = db.CheckGrowthBudget(ctx, tx, int64(len(content)+len(ciphertext)+len(encryptedKeys)+1024)); err != nil {
		return nil, err
	}

	query := `
		INSERT INTO messages (thread_id, sender_did, recipient_did, sender_id, recipient_id, content, ciphertext, encrypted_keys)
		SELECT $1, us.did, ur.did, us.id, ur.id, $4, $5,
		       CASE WHEN NULLIF($6, '') IS NULL THEN NULL ELSE $6::jsonb END
		FROM users us, users ur
		WHERE us.id = $2 AND ur.id = $3 AND ` + messageRecipientSQL() + ` AND ` + messageThreadSQL() + `
		RETURNING id, thread_id, sender_id, recipient_id, COALESCE(client_message_id, ''), CASE WHEN COALESCE(ciphertext::text,'')<>'' THEN '' ELSE content END,
		          COALESCE(ciphertext::text, ''), COALESCE(encrypted_keys::text, '{}'),
		          is_read, created_at, client_created_at, delivered_at, deleted_at, edited_at
	`

	var msg models.Message
	err = tx.QueryRow(ctx, query, threadID, senderID, recipientID, content, ciphertext, encryptedKeys).Scan(
		&msg.ID,
		&msg.ThreadID,
		&msg.SenderID,
		&msg.RecipientID,
		&msg.ClientMessageID,
		&msg.Content,
		&msg.Ciphertext,
		&msg.EncryptedKeys,
		&msg.IsRead,
		&msg.CreatedAt,
		&msg.ClientCreatedAt,
		&msg.DeliveredAt,
		&msg.DeletedAt,
		&msg.EditedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Update thread timestamp
	updateQuery := `UPDATE message_threads SET updated_at = NOW() WHERE id = $1`
	_, _ = tx.Exec(ctx, updateQuery, threadID)

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &msg, nil
}

// SendMessageWithClientMetadata sends (or deduplicates) a client-queued message using client_message_id idempotency.
// Returns the message and a bool indicating whether a new row was created.
func (r *MessageRepository) SendMessageWithClientMetadata(ctx context.Context, threadID, senderID, recipientID, clientMessageID, content, ciphertext, encryptedKeys string, clientCreatedAt *time.Time) (*models.Message, bool, error) {
	if strings.TrimSpace(ciphertext) != "" {
		content = ""
	} else if strings.TrimSpace(content) == "" {
		return nil, false, fmt.Errorf("message required")
	}
	if err := validateMessagePayload(content, ciphertext, encryptedKeys); err != nil {
		return nil, false, err
	}
	allowed, _, policyErr := r.canSendMessageToRecipient(ctx, senderID, recipientID)
	if policyErr != nil {
		return nil, false, policyErr
	}
	if !allowed {
		return nil, false, ErrMessageNotAllowed
	}

	tx, err := db.GetDB().Begin(ctx)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback(ctx)
	if err = db.CheckGrowthBudget(ctx, tx, int64(len(content)+len(ciphertext)+len(encryptedKeys)+1024)); err != nil {
		return nil, false, err
	}

	insertQuery := `
		INSERT INTO messages (
			thread_id,
			sender_did,
			recipient_did,
			sender_id,
			recipient_id,
			client_message_id,
			content,
			ciphertext,
			encrypted_keys,
			client_created_at
		)
		SELECT $1, us.did, ur.did, us.id, ur.id, NULLIF($4, ''), $5, $6,
		       CASE WHEN NULLIF($7, '') IS NULL THEN NULL ELSE $7::jsonb END,
		       $8
		FROM users us, users ur
		WHERE us.id = $2 AND ur.id = $3 AND ` + messageRecipientSQL() + ` AND ` + messageThreadSQL() + `
		ON CONFLICT (sender_id, client_message_id)
		WHERE client_message_id IS NOT NULL
		DO NOTHING
		RETURNING id, thread_id, sender_id, recipient_id, COALESCE(client_message_id, ''), CASE WHEN COALESCE(ciphertext::text,'')<>'' THEN '' ELSE content END,
		          COALESCE(ciphertext::text, ''), COALESCE(encrypted_keys::text, '{}'),
		          is_read, created_at, client_created_at, delivered_at, deleted_at, edited_at
	`

	var msg models.Message
	err = tx.QueryRow(ctx, insertQuery, threadID, senderID, recipientID, clientMessageID, content, ciphertext, encryptedKeys, clientCreatedAt).Scan(
		&msg.ID,
		&msg.ThreadID,
		&msg.SenderID,
		&msg.RecipientID,
		&msg.ClientMessageID,
		&msg.Content,
		&msg.Ciphertext,
		&msg.EncryptedKeys,
		&msg.IsRead,
		&msg.CreatedAt,
		&msg.ClientCreatedAt,
		&msg.DeliveredAt,
		&msg.DeletedAt,
		&msg.EditedAt,
	)

	inserted := true
	if err == pgx.ErrNoRows {
		inserted = false
		existingQuery := `
			SELECT id, thread_id, sender_id, recipient_id, COALESCE(client_message_id, ''), CASE WHEN COALESCE(ciphertext::text,'')<>'' THEN '' ELSE content END,
			       COALESCE(ciphertext::text, ''), COALESCE(encrypted_keys::text, '{}'),
			       is_read, created_at, client_created_at, delivered_at, deleted_at, edited_at
			FROM messages
			WHERE sender_id = $1 AND client_message_id = NULLIF($2, '')
			LIMIT 1
		`
		if fetchErr := tx.QueryRow(ctx, existingQuery, senderID, clientMessageID).Scan(
			&msg.ID,
			&msg.ThreadID,
			&msg.SenderID,
			&msg.RecipientID,
			&msg.ClientMessageID,
			&msg.Content,
			&msg.Ciphertext,
			&msg.EncryptedKeys,
			&msg.IsRead,
			&msg.CreatedAt,
			&msg.ClientCreatedAt,
			&msg.DeliveredAt,
			&msg.DeletedAt,
			&msg.EditedAt,
		); fetchErr != nil {
			return nil, false, fmt.Errorf("failed to fetch deduplicated message: %w", fetchErr)
		}
	} else if err != nil {
		return nil, false, fmt.Errorf("failed to send/sync message: %w", err)
	}

	if inserted {
		updateQuery := `UPDATE message_threads SET updated_at = NOW() WHERE id = $1`
		_, _ = tx.Exec(ctx, updateQuery, threadID)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, false, err
	}
	return &msg, inserted, nil
}

// HasClientMessageID checks if a message already exists for sender+client_message_id.
func (r *MessageRepository) HasClientMessageID(ctx context.Context, senderID, clientMessageID string) (bool, error) {
	var exists bool
	err := db.GetDB().QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM messages WHERE sender_id = $1 AND client_message_id = NULLIF($2, ''))`,
		senderID, clientMessageID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check client_message_id existence: %w", err)
	}
	return exists, nil
}

// GetThreadMessages gets all messages in a thread
func (r *MessageRepository) GetThreadMessages(ctx context.Context, threadID string, limit, offset int) ([]*models.Message, error) {
	limit, offset = boundedPage(limit, offset, 20)
	query := `
		SELECT id, thread_id, sender_id, recipient_id, COALESCE(client_message_id, ''), CASE WHEN COALESCE(ciphertext::text,'')<>'' THEN '' ELSE content END,
		       COALESCE(ciphertext::text, ''), COALESCE(encrypted_keys::text, '{}'),
		       is_read, created_at, client_created_at, delivered_at, deleted_at, edited_at
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
			&msg.ClientMessageID,
			&msg.Content,
			&msg.Ciphertext,
			&msg.EncryptedKeys,
			&msg.IsRead,
			&msg.CreatedAt,
			&msg.ClientCreatedAt,
			&msg.DeliveredAt,
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
	return r.GetUserThreadsPage(ctx, userID, 30, 0)
}
func (r *MessageRepository) GetUserThreadsPage(ctx context.Context, userID string, limit, offset int) ([]*models.MessageThread, error) {
	limit, offset = boundedPage(limit, offset, 30)
	rows, err := db.GetDB().Query(ctx, `SELECT t.id,t.participant_a_id,t.participant_b_id,t.created_at,t.updated_at,
 u.id,u.username,COALESCE(u.display_name,''),COALESCE(u.avatar_url,''),COALESCE(u.instance_domain,''),COALESCE(u.encryption_public_key,''),COALESCE(u.did,''),
 (SELECT count(*)FROM messages unread WHERE unread.thread_id=t.id AND unread.recipient_id=$1 AND NOT unread.is_read),
 COALESCE(last.id::text,''),COALESCE(last.sender_id::text,''),COALESCE(last.recipient_id::text,''),COALESCE(last.client_message_id,''),
 CASE WHEN COALESCE(last.ciphertext::text,'')<>'' THEN '' ELSE COALESCE(last.content,'') END,
 COALESCE(last.ciphertext::text,''),COALESCE(last.encrypted_keys::text,'{}'),COALESCE(last.is_read,false),COALESCE(last.created_at,t.created_at),last.client_created_at,last.delivered_at,last.deleted_at,last.edited_at
 FROM message_threads t JOIN users u ON u.id=CASE WHEN t.participant_a_id=$1 THEN t.participant_b_id ELSE t.participant_a_id END
 LEFT JOIN LATERAL(SELECT *FROM messages m WHERE m.thread_id=t.id AND m.deleted_at IS NULL ORDER BY m.created_at DESC,m.id DESC LIMIT 1)last ON true
 WHERE t.participant_a_id=$1 OR t.participant_b_id=$1 ORDER BY t.updated_at DESC,t.id DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get threads: %w", err)
	}
	defer rows.Close()
	threads := make([]*models.MessageThread, 0)
	for rows.Next() {
		var thread models.MessageThread
		var other models.User
		var last models.Message
		if err = rows.Scan(&thread.ID, &thread.ParticipantAID, &thread.ParticipantBID, &thread.CreatedAt, &thread.UpdatedAt, &other.ID, &other.Username, &other.DisplayName, &other.AvatarURL, &other.InstanceDomain, &other.EncryptionPublicKey, &other.DID, &thread.UnreadCount, &last.ID, &last.SenderID, &last.RecipientID, &last.ClientMessageID, &last.Content, &last.Ciphertext, &last.EncryptedKeys, &last.IsRead, &last.CreatedAt, &last.ClientCreatedAt, &last.DeliveredAt, &last.DeletedAt, &last.EditedAt); err != nil {
			return nil, err
		}
		thread.OtherUser = models.ToPublicUser(&other)
		if last.ID != "" {
			last.ThreadID = thread.ID
			thread.LastMessage = &last
		}
		threads = append(threads, &thread)
	}
	return threads, rows.Err()
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
		return fmt.Errorf("message not found")
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
		return fmt.Errorf("failed to delete message")
	}

	return nil
}

// EditMessage updates message content (encrypted or plain)
// Only allows editing within 3 hours of sending
func (r *MessageRepository) EditMessage(ctx context.Context, messageID, userID, newContent, newCiphertext string) error {
	return r.EditMessageWithKeys(ctx, messageID, userID, newContent, newCiphertext, nil)
}
func (r *MessageRepository) EditMessageWithKeys(ctx context.Context, messageID, userID, newContent, newCiphertext string, encryptedKeys *string) error {
	if strings.TrimSpace(newCiphertext) != "" {
		newContent = ""
	} else if strings.TrimSpace(newContent) == "" {
		return fmt.Errorf("message required")
	}
	keys := ""
	if encryptedKeys != nil {
		keys = *encryptedKeys
	}
	if err := validateMessagePayload(newContent, newCiphertext, keys); err != nil {
		return err
	}
	tx, err := db.GetDB().Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = db.CheckGrowthBudget(ctx, tx, int64(len(newContent)+len(newCiphertext)+len(keys)+1024)); err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `UPDATE messages SET content=$1,ciphertext=$2,
 encrypted_keys=CASE WHEN $3::text IS NULL THEN encrypted_keys ELSE $3::jsonb END,edited_at=NOW()
 WHERE id=$4 AND sender_id=$5 AND deleted_at IS NULL AND created_at>NOW()-INTERVAL '3 hours'
 AND (COALESCE(ciphertext::text,'')='' OR $2<>'')
 AND EXISTS(SELECT 1 FROM users WHERE id=$5 AND NOT is_suspended)`, newContent, newCiphertext, encryptedKeys, messageID, userID)
	if err != nil {
		return fmt.Errorf("failed to edit message")
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("message unavailable, edit window expired or encrypted message requires ciphertext")
	}
	return tx.Commit(ctx)
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

func validateMessagePayload(content, ciphertext, encryptedKeys string) error {
	if len(content) > 64<<10 || len(ciphertext) > 128<<10 || len(encryptedKeys) > 64<<10 {
		return ErrMessagePayload
	}
	return nil
}
