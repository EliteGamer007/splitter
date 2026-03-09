package worker

import (
	"context"
	"log"
	"time"

	"splitter/internal/repository"
)

func StartStoryCleanup(repo *repository.StoryRepository) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for {
			<-ticker.C
			if err := repo.DeleteExpiredStories(context.Background()); err != nil {
				log.Printf("[StoryCleanup] Failed to delete expired stories: %v", err)
			} else {
				log.Println("[StoryCleanup] Successfully cleaned up expired stories")
			}
		}
	}()
}
