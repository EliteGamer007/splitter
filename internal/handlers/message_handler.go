package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"splitter/internal/config"
	"splitter/internal/federation"
	"splitter/internal/models"
	"splitter/internal/repository"
	"splitter/internal/security"

	"github.com/labstack/echo/v4"
)

// MessageHandler handles message-related requests
type MessageHandler struct {
	msgRepo  *repository.MessageRepository
	userRepo *repository.UserRepository
	cfg      *config.Config
}

// NewMessageHandler creates a new MessageHandler
func NewMessageHandler(msgRepo *repository.MessageRepository, userRepo *repository.UserRepository, cfg *config.Config) *MessageHandler {
	return &MessageHandler{
		msgRepo:  msgRepo,
		userRepo: userRepo,
		cfg:      cfg,
	}
}

// GetThreads gets all message threads for the current user
func (h *MessageHandler) GetThreads(c echo.Context) error {
	userID := c.Get("user_id").(string)

	threads, err := h.msgRepo.GetUserThreads(c.Request().Context(), userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get threads: " + err.Error(),
		})
	}

	if threads == nil {
		threads = []*models.MessageThread{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"threads": threads,
	})
}

// GetMessages gets messages in a thread
func (h *MessageHandler) GetMessages(c echo.Context) error {
	userID := c.Get("user_id").(string)
	threadID := c.Param("threadId")

	// Verify user is participant in thread
	thread, err := h.msgRepo.GetThread(c.Request().Context(), threadID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Thread not found",
		})
	}

	if thread.ParticipantAID != userID && thread.ParticipantBID != userID {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Not authorized to view this thread",
		})
	}

	limit := 50
	offset := 0

	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if o := c.QueryParam("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	messages, err := h.msgRepo.GetThreadMessages(c.Request().Context(), threadID, limit, offset)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get messages: " + err.Error(),
		})
	}

	// Mark messages as read
	_ = h.msgRepo.MarkMessagesAsRead(c.Request().Context(), threadID, userID)

	if messages == nil {
		messages = []*models.Message{}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"messages": messages,
		"thread":   thread,
	})
}

// SendMessage sends a message to another user
func (h *MessageHandler) SendMessage(c echo.Context) error {
	userID := c.Get("user_id").(string)
	ctx := c.Request().Context()
	guard := security.GetMessagingGuard()

	var req struct {
		RecipientID   string            `json:"recipient_id"`
		Content       string            `json:"content"`
		Ciphertext    string            `json:"ciphertext"`
		EncryptedKeys map[string]string `json:"encrypted_keys"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if req.RecipientID == "" || (req.Content == "" && req.Ciphertext == "") {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Recipient ID and message content (or ciphertext) are required",
		})
	}

	if req.RecipientID == userID {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Cannot send message to yourself",
		})
	}

	if allowed, reason := guard.AllowLocalSend(userID, 1); !allowed {
		return c.JSON(http.StatusTooManyRequests, map[string]string{
			"error": reason,
		})
	}

	// Verify recipient exists
	recipient, err := h.userRepo.GetByID(ctx, req.RecipientID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Recipient not found",
		})
	}

	// Get sender details
	sender, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get sender profile",
		})
	}

	// Get or create thread
	thread, err := h.msgRepo.GetOrCreateThread(ctx, userID, req.RecipientID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get/create thread: " + err.Error(),
		})
	}

	encryptedKeysJSON := "{}"
	if len(req.EncryptedKeys) > 0 {
		if raw, marshalErr := json.Marshal(req.EncryptedKeys); marshalErr == nil {
			encryptedKeysJSON = string(raw)
		} else {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "invalid encrypted_keys payload",
			})
		}
	}

	// Send message
	msg, err := h.msgRepo.SendMessage(ctx, thread.ID, userID, req.RecipientID, req.Content, req.Ciphertext, encryptedKeysJSON)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to send message: " + err.Error(),
		})
	}

	response := map[string]interface{}{
		"message":   msg,
		"thread":    thread,
		"recipient": recipient,
	}

	if err := h.deliverFederatedDM(sender, recipient, req.Content, req.Ciphertext, req.EncryptedKeys); err != nil {
		log.Printf("[DM] Federation delivery failed (message saved locally): %v", err)
		response["federation_error"] = err.Error()
	}

	return c.JSON(http.StatusCreated, response)
}

// StartConversation starts or gets a conversation with a user
func (h *MessageHandler) StartConversation(c echo.Context) error {
	userID := c.Get("user_id").(string)
	otherUserID := c.Param("userId")

	if otherUserID == userID {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Cannot start conversation with yourself",
		})
	}

	// Verify other user exists
	otherUser, err := h.userRepo.GetByID(c.Request().Context(), otherUserID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "User not found",
		})
	}

	// Get or create thread
	thread, err := h.msgRepo.GetOrCreateThread(c.Request().Context(), userID, otherUserID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to start conversation: " + err.Error(),
		})
	}

	thread.OtherUser = otherUser

	return c.JSON(http.StatusOK, map[string]interface{}{
		"thread": thread,
	})
}

// MarkAsRead marks all messages in a thread as read
func (h *MessageHandler) MarkAsRead(c echo.Context) error {
	userID := c.Get("user_id").(string)
	threadID := c.Param("threadId")

	// Verify user is participant
	thread, err := h.msgRepo.GetThread(c.Request().Context(), threadID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Thread not found",
		})
	}

	if thread.ParticipantAID != userID && thread.ParticipantBID != userID {
		return c.JSON(http.StatusForbidden, map[string]string{
			"error": "Not authorized",
		})
	}

	err = h.msgRepo.MarkMessagesAsRead(c.Request().Context(), threadID, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to mark as read: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Messages marked as read",
	})
}

// DeleteMessage soft-deletes a message (WhatsApp-style)
func (h *MessageHandler) DeleteMessage(c echo.Context) error {
	userID := c.Get("user_id").(string)
	messageID := c.Param("messageId")

	err := h.msgRepo.DeleteMessage(c.Request().Context(), messageID, userID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Message deleted",
	})
}

// EditMessage edits a message
func (h *MessageHandler) EditMessage(c echo.Context) error {
	userID := c.Get("user_id").(string)
	messageID := c.Param("messageId")

	var req struct {
		Content    string `json:"content"`
		Ciphertext string `json:"ciphertext"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	err := h.msgRepo.EditMessage(c.Request().Context(), messageID, userID, req.Content, req.Ciphertext)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Message edited",
	})
}

// SyncOfflineMessages safely syncs client-queued encrypted messages after reconnect.
// Idempotency is guaranteed per sender using client_message_id.
func (h *MessageHandler) SyncOfflineMessages(c echo.Context) error {
	userID := c.Get("user_id").(string)
	ctx := c.Request().Context()
	guard := security.GetMessagingGuard()

	var req struct {
		QueuedMessages []struct {
			ClientMessageID string            `json:"client_message_id"`
			RecipientID     string            `json:"recipient_id"`
			Content         string            `json:"content"`
			Ciphertext      string            `json:"ciphertext"`
			EncryptedKeys   map[string]string `json:"encrypted_keys"`
			ClientCreatedAt string            `json:"client_created_at"`
		} `json:"queued_messages"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request body",
		})
	}

	if len(req.QueuedMessages) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "queued_messages is required",
		})
	}

	if len(req.QueuedMessages) > 100 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Batch too large. Maximum 100 queued messages per sync.",
		})
	}

	if allowed, reason := guard.AllowLocalSend(userID, len(req.QueuedMessages)); !allowed {
		guard.RecordSuspicious(userID, "offline sync rate-limited", map[string]interface{}{
			"queued_messages": len(req.QueuedMessages),
		})
		return c.JSON(http.StatusTooManyRequests, map[string]string{
			"error": reason,
		})
	}

	type syncResult struct {
		ClientMessageID string          `json:"client_message_id"`
		Message         *models.Message `json:"message,omitempty"`
		ThreadID        string          `json:"thread_id,omitempty"`
		Created         bool            `json:"created"`
		Error           string          `json:"error,omitempty"`
	}

	sender, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to load sender profile",
		})
	}

	results := make([]syncResult, 0, len(req.QueuedMessages))
	createdCount := 0
	deduplicatedCount := 0
	failureCount := 0

	for _, queued := range req.QueuedMessages {
		if queued.RecipientID == "" || queued.ClientMessageID == "" || (queued.Content == "" && queued.Ciphertext == "") {
			failureCount++
			results = append(results, syncResult{
				ClientMessageID: queued.ClientMessageID,
				Created:         false,
				Error:           "recipient_id, client_message_id and content/ciphertext are required",
			})
			continue
		}

		if queued.RecipientID == userID {
			failureCount++
			results = append(results, syncResult{
				ClientMessageID: queued.ClientMessageID,
				Created:         false,
				Error:           "cannot send message to yourself",
			})
			continue
		}

		recipient, err := h.userRepo.GetByID(ctx, queued.RecipientID)
		if err != nil {
			failureCount++
			results = append(results, syncResult{
				ClientMessageID: queued.ClientMessageID,
				Created:         false,
				Error:           "recipient not found",
			})
			continue
		}

		thread, err := h.msgRepo.GetOrCreateThread(ctx, userID, queued.RecipientID)
		if err != nil {
			failureCount++
			results = append(results, syncResult{
				ClientMessageID: queued.ClientMessageID,
				Created:         false,
				Error:           "failed to get/create thread: " + err.Error(),
			})
			continue
		}

		var clientCreatedAt *time.Time
		if queued.ClientCreatedAt != "" {
			if parsed, parseErr := time.Parse(time.RFC3339, queued.ClientCreatedAt); parseErr == nil {
				clientCreatedAt = &parsed
			}
		}

		existing, existsErr := h.msgRepo.HasClientMessageID(ctx, userID, queued.ClientMessageID)
		if existsErr != nil {
			failureCount++
			results = append(results, syncResult{
				ClientMessageID: queued.ClientMessageID,
				Created:         false,
				Error:           "failed to check duplicate message id: " + existsErr.Error(),
			})
			continue
		}

		msg, created, err := h.msgRepo.SendMessageWithClientMetadata(
			ctx,
			thread.ID,
			userID,
			queued.RecipientID,
			queued.ClientMessageID,
			queued.Content,
			queued.Ciphertext,
			mapToJSONString(queued.EncryptedKeys),
			clientCreatedAt,
		)
		if err != nil {
			failureCount++
			results = append(results, syncResult{
				ClientMessageID: queued.ClientMessageID,
				Created:         false,
				Error:           "failed to sync message: " + err.Error(),
			})
			continue
		}

		if existing {
			created = false
		} else {
			created = true
		}

		if created {
			createdCount++
			sr := syncResult{
				ClientMessageID: queued.ClientMessageID,
				Message:         msg,
				ThreadID:        thread.ID,
				Created:         true,
			}
			if fedErr := h.deliverFederatedDM(sender, recipient, queued.Content, queued.Ciphertext, queued.EncryptedKeys); fedErr != nil {
				log.Printf("[DM] Sync federation delivery failed (message saved): %v", fedErr)
				sr.Error = "federation delivery pending: " + fedErr.Error()
			}
			results = append(results, sr)
		} else {
			deduplicatedCount++
			results = append(results, syncResult{
				ClientMessageID: queued.ClientMessageID,
				Message:         msg,
				ThreadID:        thread.ID,
				Created:         created,
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"results":            results,
		"created_count":      createdCount,
		"deduplicated_count": deduplicatedCount,
		"failed_count":       failureCount,
	})
}

func mapToJSONString(value map[string]string) string {
	if len(value) == 0 {
		return "{}"
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(raw)
}

func (h *MessageHandler) deliverFederatedDM(sender, recipient *models.User, content, ciphertext string, encryptedKeys map[string]string) error {
	if !h.cfg.Federation.Enabled || recipient == nil || sender == nil {
		return nil
	}

	if recipient.InstanceDomain == h.cfg.Federation.Domain || recipient.InstanceDomain == "localhost" || recipient.InstanceDomain == "" {
		return nil
	}

	resolveInputs := []string{}
	if recipient.Username != "" && recipient.InstanceDomain != "" {
		resolveInputs = append(resolveInputs, recipient.Username+"@"+recipient.InstanceDomain)
	}
	if recipient.DID != "" {
		resolveInputs = append(resolveInputs, recipient.DID)
	}

	var remoteActor *federation.RemoteActor
	var resolveErr error
	for _, candidate := range resolveInputs {
		remoteActor, resolveErr = federation.ResolveRemoteUser(candidate)
		if resolveErr == nil && remoteActor != nil && strings.TrimSpace(remoteActor.InboxURL) != "" {
			break
		}
	}

	if remoteActor == nil || strings.TrimSpace(remoteActor.InboxURL) == "" {
		if resolveErr == nil {
			resolveErr = fmt.Errorf("missing remote inbox")
		}
		return fmt.Errorf("resolve recipient failed: %w", resolveErr)
	}

	actorURI := fmt.Sprintf("%s/ap/users/%s", h.cfg.Federation.URL, sender.Username)
	recipientURI := recipient.DID
	if strings.TrimSpace(recipientURI) == "" {
		recipientURI = remoteActor.ActorURI
	}

	activity := federation.BuildCreateDMActivity(actorURI, recipientURI, content, ciphertext, encryptedKeys)
	return federation.DeliverActivity(activity, remoteActor.InboxURL)
}
