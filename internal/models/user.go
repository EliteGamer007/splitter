package models

import (
	"time"
)

// User represents a user in the system with Decentralized Identity (DID)
type User struct {
	ID                    string     `json:"id"` // UUID
	Username              string     `json:"username"`
	Email                 string     `json:"email,omitempty"`
	PasswordHash          string     `json:"-"`               // Never expose in JSON
	InstanceDomain        string     `json:"instance_domain"` // Home server domain
	DID                   string     `json:"did"`             // Decentralized Identifier (did:key:...)
	DisplayName           string     `json:"display_name"`
	Bio                   string     `json:"bio,omitempty"`
	AvatarURL             string     `json:"avatar_url,omitempty"`
	PublicKey             string     `json:"public_key"` // Base64 encoded public key
	Role                  string     `json:"role"`       // user, moderator, admin
	ModerationRequested   bool       `json:"moderation_requested"`
	ModerationRequestedAt *time.Time `json:"moderation_requested_at,omitempty"`
	IsLocked              bool       `json:"is_locked"`
	IsSuspended           bool       `json:"is_suspended"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// Message represents a direct message between two users
type Message struct {
	ID          string     `json:"id"`
	ThreadID    string     `json:"thread_id"`
	SenderID    string     `json:"sender_id"`
	RecipientID string     `json:"recipient_id"`
	Content     string     `json:"content"`
	IsRead      bool       `json:"is_read"`
	CreatedAt   time.Time  `json:"created_at"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
}

// MessageThread represents a conversation between two users
type MessageThread struct {
	ID             string    `json:"id"`
	ParticipantAID string    `json:"participant_a_id"`
	ParticipantBID string    `json:"participant_b_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	// Populated fields
	OtherUser   *User    `json:"other_user,omitempty"`
	LastMessage *Message `json:"last_message,omitempty"`
	UnreadCount int      `json:"unread_count"`
}

// ModerationRequest represents a request for moderation privileges
type ModerationRequest struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	Status     string     `json:"status"` // pending, approved, rejected
	Reason     string     `json:"reason,omitempty"`
	ReviewedBy *string    `json:"reviewed_by,omitempty"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	// Populated fields
	User *User `json:"user,omitempty"`
}

// UserCreate represents the data needed to create a new user
type UserCreate struct {
	Username       string `json:"username" validate:"required,min=3,max=50"`
	Email          string `json:"email" validate:"required,email"`
	Password       string `json:"password" validate:"required,min=8"`
	DisplayName    string `json:"display_name"`
	InstanceDomain string `json:"instance_domain"`
	DID            string `json:"did"`
	PublicKey      string `json:"public_key"`
	Bio            string `json:"bio,omitempty"`
	AvatarURL      string `json:"avatar_url,omitempty"`
}

// LoginRequest represents a login request with username/email and password
type LoginRequest struct {
	Username string `json:"username"` // Can be username or email
	Password string `json:"password" validate:"required"`
}

// UserUpdate represents the data that can be updated for a user
type UserUpdate struct {
	DisplayName *string `json:"display_name,omitempty"`
	Bio         *string `json:"bio,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

// ChallengeRequest represents a request to get an auth challenge
type ChallengeRequest struct {
	DID string `json:"did" validate:"required"`
}

// ChallengeResponse represents the server's challenge for authentication
type ChallengeResponse struct {
	Challenge string `json:"challenge"` // Nonce to be signed by client
	ExpiresAt int64  `json:"expires_at"`
}

// VerifyChallengeRequest represents a signed challenge for verification
type VerifyChallengeRequest struct {
	DID       string `json:"did" validate:"required"`
	Challenge string `json:"challenge" validate:"required"`
	Signature string `json:"signature" validate:"required"` // Base64 encoded signature
}

// AuthChallenge represents a stored challenge for verification
type AuthChallenge struct {
	DID       string
	Challenge string
	ExpiresAt time.Time
}
