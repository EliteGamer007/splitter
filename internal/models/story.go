package models

import (
	"time"
)

// Story represents a single story image uploaded by a user
type Story struct {
	ID        string    `json:"id"`
	AuthorDID string    `json:"author_did"`
	MediaURL  string    `json:"media_url"`
	MediaType string    `json:"media_type"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// StoryUser represents a user who has active stories, with their stories grouped
type StoryUser struct {
	AuthorDID   string  `json:"author_did"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	AvatarURL   string  `json:"avatar_url"`
	Stories     []Story `json:"stories"`
}
