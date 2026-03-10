package models

import (
	"time"

	"github.com/google/uuid"
)

type Author struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Avatar   string    `json:"avatar"`
}

type Story struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	MediaURL  string    `json:"media_url"`
	MediaData []byte    `json:"-"`
	MediaType string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Seen      bool      `json:"seen"`
	Author    Author    `json:"author"`
}

type StoryUser struct {
	UserID   uuid.UUID `json:"user_id"`
	Username string    `json:"username"`
	Avatar   string    `json:"avatar"`
	Stories  []Story   `json:"stories"`
}
