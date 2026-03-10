package service

import (
	"context"
	"strings"
	"time"

	"splitter/internal/models"
	"splitter/internal/repository"

	"github.com/google/uuid"
)

type StoryService struct {
	repo *repository.StoryRepository
}

func NewStoryService(repo *repository.StoryRepository) *StoryService {
	return &StoryService{repo: repo}
}

func (s *StoryService) CreateStory(ctx context.Context, userID uuid.UUID, mediaURL string, mediaData []byte, mediaType string) error {
	now := time.Now()
	story := &models.Story{
		ID:        uuid.New(),
		UserID:    userID,
		MediaData: mediaData,
		MediaType: mediaType,
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}
	
	if mediaURL != "" && !strings.HasPrefix(mediaURL, "/media/") {
		story.MediaURL = mediaURL
	} else {
		story.MediaURL = "/api/v1/stories/" + story.ID.String() + "/media"
	}
	
	return s.repo.CreateStory(ctx, story)
}

func (s *StoryService) GetStories(ctx context.Context) ([]models.Story, error) {
	return s.repo.GetActiveStories(ctx)
}

func (s *StoryService) DeleteStory(ctx context.Context, storyID uuid.UUID, userID uuid.UUID) error {
	return s.repo.DeleteStory(ctx, storyID, userID)
}

func (s *StoryService) RecordStoryView(ctx context.Context, storyID uuid.UUID, viewerID uuid.UUID) error {
	return s.repo.RecordStoryView(ctx, storyID, viewerID)
}

func (s *StoryService) GetStoryFeed(ctx context.Context) ([]models.StoryUser, error) {
	return s.repo.GetStoryFeed(ctx)
}

func (s *StoryService) GetStoryMedia(ctx context.Context, id string) ([]byte, string, error) {
	return s.repo.GetStoryMedia(ctx, id)
}
