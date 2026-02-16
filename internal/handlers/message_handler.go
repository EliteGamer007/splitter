package handlers

import (
	"net/http"
	"strconv"

	"splitter/internal/models"
	"splitter/internal/repository"

	"github.com/labstack/echo/v4"
)

// MessageHandler handles message-related requests
type MessageHandler struct {
	msgRepo  *repository.MessageRepository
	userRepo *repository.UserRepository
}

// NewMessageHandler creates a new MessageHandler
func NewMessageHandler(msgRepo *repository.MessageRepository, userRepo *repository.UserRepository) *MessageHandler {
	return &MessageHandler{
		msgRepo:  msgRepo,
		userRepo: userRepo,
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

	var req struct {
		RecipientID string `json:"recipient_id"`
		Content     string `json:"content"`
		Ciphertext  string `json:"ciphertext"`
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

	// Verify recipient exists
	recipient, err := h.userRepo.GetByID(c.Request().Context(), req.RecipientID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Recipient not found",
		})
	}

	// Get or create thread
	thread, err := h.msgRepo.GetOrCreateThread(c.Request().Context(), userID, req.RecipientID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to get/create thread: " + err.Error(),
		})
	}

	// Send message
	msg, err := h.msgRepo.SendMessage(c.Request().Context(), thread.ID, userID, req.RecipientID, req.Content, req.Ciphertext)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to send message: " + err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message":   msg,
		"thread":    thread,
		"recipient": recipient,
	})
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
