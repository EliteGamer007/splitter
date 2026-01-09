package main

import (
	"log"
	"splitter/Database"
)

func main() {
	// Initialize database connection
	err := Database.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer Database.Close()

	log.Println("Splitter application started successfully")

}
