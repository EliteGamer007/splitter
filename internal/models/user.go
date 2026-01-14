package models

import (
	"time"
)

// User represents a user in the system with Decentralized Identity (DID)
type User struct {
	ID             string    `json:"id"` // UUID
	Username       string    `json:"username"`
	InstanceDomain string    `json:"instance_domain"` // Home server domain
	DID            string    `json:"did"`             // Decentralized Identifier (did:key:...)
	DisplayName    string    `json:"display_name"`
	Bio            string    `json:"bio,omitempty"`
	AvatarURL      string    `json:"avatar_url,omitempty"`
	PublicKey      string    `json:"public_key"` // Base64 encoded public key
	IsLocked       bool      `json:"is_locked"`
	IsSuspended    bool      `json:"is_suspended"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// UserCreate represents the data needed to create a new user with DID
type UserCreate struct {
	Username       string `json:"username" validate:"required,min=3,max=50"`
	InstanceDomain string `json:"instance_domain" validate:"required"`
	DID            string `json:"did" validate:"required"`
	DisplayName    string `json:"display_name" validate:"required"`
	PublicKey      string `json:"public_key" validate:"required"` // Base64 encoded public key
	Bio            string `json:"bio,omitempty"`
	AvatarURL      string `json:"avatar_url,omitempty"`
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
