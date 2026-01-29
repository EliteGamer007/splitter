package models

import (
	"time"
)

// Post represents a post/content created by a user
type Post struct {
	ID         string     `json:"id"`                 // UUID
	AuthorDID  string     `json:"author_did"`         // DID of the author
	Username   string     `json:"username,omitempty"` // Joined from User table
	Content    string     `json:"content"`
	Visibility string     `json:"visibility,omitempty"` // public, followers, circle
	IsRemote   bool       `json:"is_remote"`
	LikeCount  int        `json:"like_count"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  *time.Time `json:"updated_at,omitempty"`
}

// PostCreate represents the data needed to create a new post
type PostCreate struct {
	Content    string `json:"content" validate:"required,max=500"`
	Visibility string `json:"visibility,omitempty"` // defaults to "public"
}

// PostUpdate represents the data that can be updated for a post
type PostUpdate struct {
	Content    *string `json:"content,omitempty" validate:"omitempty,max=500"`
	Visibility *string `json:"visibility,omitempty"`
}

// PostWithAuthor represents a post with author information
type PostWithAuthor struct {
	Post
	Author User `json:"author"`
}
