package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"splitter/internal/config"
	"splitter/internal/db"
	"splitter/internal/federation"
)

func main() {
	cfg := config.Load()

	if err := db.InitDB(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	federation.ConfigureDeliveryPolicy(
		cfg.Worker.MaxRetryCount,
		cfg.Worker.CircuitFailureThreshold,
		time.Duration(cfg.Worker.CircuitCooldownSeconds)*time.Second,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	retryTicker := time.NewTicker(time.Duration(cfg.Worker.RetryIntervalSeconds) * time.Second)
	reputationTicker := time.NewTicker(time.Duration(cfg.Worker.ReputationIntervalSeconds) * time.Second)
	defer retryTicker.Stop()
	defer reputationTicker.Stop()

	log.Printf("Worker started: retry every %ds, reputation every %ds", cfg.Worker.RetryIntervalSeconds, cfg.Worker.ReputationIntervalSeconds)

	if cfg.Federation.Enabled {
		if err := federation.RecalculateInstanceReputation(ctx); err != nil {
			log.Printf("[Worker] Initial reputation calc failed: %v", err)
		}
	}

	for {
		select {
		case <-sigCh:
			log.Println("Worker shutting down")
			return
		case <-ctx.Done():
			return
		case <-retryTicker.C:
			if !cfg.Federation.Enabled {
				continue
			}
			processed, failed, err := federation.RetryOutboxBatch(ctx, 50)
			if err != nil {
				log.Printf("[Worker] Retry batch failed: %v", err)
				continue
			}
			if processed > 0 {
				log.Printf("[Worker] Retry batch processed=%d failed=%d", processed, failed)
			}
		case <-reputationTicker.C:
			if !cfg.Federation.Enabled {
				continue
			}
			if err := federation.RecalculateInstanceReputation(ctx); err != nil {
				log.Printf("[Worker] Reputation recalculation failed: %v", err)
			}
		}
	}
}
