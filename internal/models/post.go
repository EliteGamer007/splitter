package models

import (
	"fmt"
	"time"
)

// Post represents a post/content created by a user
type Post struct {
	ID               string     `json:"id"`
	AuthorDID        string     `json:"author_did"`
	Username         string     `json:"username,omitempty"`
	Content          string     `json:"content"`
	Visibility       string     `json:"visibility,omitempty"`
	IsRemote         bool       `json:"is_remote"`
	LikeCount        int        `json:"like_count"`
	Liked            bool       `json:"liked"` // Whether current user has liked this post
	RepostCount      int        `json:"repost_count"`
	Reposted         bool       `json:"reposted"` // Whether current user has reposted this post
	DirectReplyCount int        `json:"direct_reply_count"`
	TotalReplyCount  int        `json:"total_reply_count"`
	Media            []Media    `json:"media,omitempty"` // Attached media
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        *time.Time `json:"updated_at,omitempty"`
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

// Validate checks if the PostCreate struct is valid
func (p *PostCreate) Validate(hasMedia bool) error {
	if len(p.Content) > 500 {
		return fmt.Errorf("content too long (max 500 characters)")
	}
	if p.Content == "" && !hasMedia {
		return fmt.Errorf("either content or media is required")
	}
	if p.Visibility != "" && p.Visibility != "public" && p.Visibility != "followers" && p.Visibility != "private" {
		return fmt.Errorf("invalid visibility setting")
	}
	return nil
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
