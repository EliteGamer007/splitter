package models

import (
	"time"
)

// Post represents a post/content created by a user
type Post struct {
	ID          string     `json:"id"`
	AuthorDID   string     `json:"author_did"`
	Username    string     `json:"username,omitempty"`
	Content     string     `json:"content"`
	Visibility  string     `json:"visibility,omitempty"`
	IsRemote    bool       `json:"is_remote"`
	LikeCount   int        `json:"like_count"`
	Liked       bool       `json:"liked"` // Whether current user has liked this post
	RepostCount int        `json:"repost_count"`
	Reposted    bool       `json:"reposted"`        // Whether current user has reposted this post
	Media       []Media    `json:"media,omitempty"` // Attached media
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

// Media represents a media attachment
type Media struct {
	ID        string    `json:"id"`
	PostID    string    `json:"post_id"`
	MediaURL  string    `json:"media_url"`
	MediaType string    `json:"media_type"` // image/jpeg, image/png, etc.
	CreatedAt time.Time `json:"created_at"`
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
