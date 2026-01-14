package models

import (
	"time"
)

// Post represents a post/content created by a user
type Post struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username,omitempty"` // Joined from User table
	Content   string    `json:"content"`
	ImageURL  string    `json:"image_url,omitempty"`
	LikeCount int       `json:"like_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PostCreate represents the data needed to create a new post
type PostCreate struct {
	Content  string `json:"content" validate:"required,max=500"`
	ImageURL string `json:"image_url,omitempty"`
}

// PostUpdate represents the data that can be updated for a post
type PostUpdate struct {
	Content  *string `json:"content,omitempty" validate:"omitempty,max=500"`
	ImageURL *string `json:"image_url,omitempty"`
}

// PostWithAuthor represents a post with author information
type PostWithAuthor struct {
	Post
	Author User `json:"author"`
}
