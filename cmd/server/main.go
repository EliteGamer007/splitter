package main

import (
	"log"
	"os"

	"splitter/internal/config"
	"splitter/internal/db"
	"splitter/internal/server"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database connection
	if err := db.InitDB(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize and start server
	srv := server.NewServer(cfg)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	log.Printf("Starting server on port %s", port)
	if err := srv.Start(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
