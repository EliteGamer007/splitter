package models

import (
	"time"
)

// Reply represents a threaded reply
type Reply struct {
	ID               string     `json:"id"`
	PostID           string     `json:"post_id"`
	ParentID         *string    `json:"parent_id,omitempty"`
	AuthorDID        string     `json:"author_did"`
	Username         string     `json:"username,omitempty"` // populated from join
	Content          string     `json:"content"`
	Depth            int        `json:"depth"`
	LikesCount       int        `json:"likes_count"`
	Liked            bool       `json:"liked"` // Whether current user has liked this reply
	DirectReplyCount int        `json:"direct_reply_count"`
	TotalReplyCount  int        `json:"total_reply_count"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        *time.Time `json:"updated_at,omitempty"`
}

// ReplyCreate represents the data needed to create a new reply
type ReplyCreate struct {
	PostID   string  `json:"post_id" validate:"required"`
	ParentID *string `json:"parent_id,omitempty"`
	Content  string  `json:"content" validate:"required,max=500"`
}
