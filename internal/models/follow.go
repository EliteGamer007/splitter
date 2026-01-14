package models

import (
	"time"
)

// Follow represents a follow relationship between users
type Follow struct {
	ID          int       `json:"id"`
	FollowerID  int       `json:"follower_id"`  // User who is following
	FollowingID int       `json:"following_id"` // User being followed
	CreatedAt   time.Time `json:"created_at"`
}

// FollowRequest represents a follow/unfollow action
type FollowRequest struct {
	FollowingID int `json:"following_id" validate:"required"`
}

// FollowStats represents follower/following statistics for a user
type FollowStats struct {
	UserID         int `json:"user_id"`
	FollowerCount  int `json:"follower_count"`
	FollowingCount int `json:"following_count"`
}
