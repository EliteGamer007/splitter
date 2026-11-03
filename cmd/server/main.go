package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"splitter/internal/config"
	"splitter/internal/db"
	"splitter/internal/federation"
	"splitter/internal/server"
)

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

// run starts the web process. Schema upgrades and administrator provisioning
// are explicit commands; starting the server never resets existing accounts.
func run() error {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid server configuration: %w", err)
	}
	if err := db.InitDB(cfg); err != nil {
		return errors.New("cannot initialize database; check configuration and connectivity")
	}
	defer db.Close()
	federation.ConfigureDeliveryPolicy(cfg.Worker.MaxRetryCount, cfg.Worker.CircuitFailureThreshold, time.Duration(cfg.Worker.CircuitCooldownSeconds)*time.Second)
	srv := server.NewServer(cfg)
	httpServer := srv.Echo().Server
	httpServer.ReadHeaderTimeout = 5 * time.Second
	httpServer.ReadTimeout = 15 * time.Second
	httpServer.WriteTimeout = 30 * time.Second
	httpServer.IdleTimeout = 60 * time.Second
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Echo().Shutdown(shutdownCtx); err != nil {
			log.Print("Server shutdown did not finish within its deadline")
		}
	}()
	log.Printf("Starting server on port %s", cfg.Server.Port)
	if err := srv.Start(":" + cfg.Server.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return errors.New("HTTP server failed to start or stopped unexpectedly")
	}
	return nil
}
