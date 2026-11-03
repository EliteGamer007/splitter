package main

import (
	"context"
	"flag"
	"log"
	"splitter/internal/config"
	"splitter/internal/db"
	"time"
)

func main() {
	dir := flag.String("dir", "migrations", "directory of ordered SQL migrations")
	flag.Parse()
	cfg := config.Load()
	if err := db.InitDB(cfg); err != nil {
		log.Fatal("Cannot connect to migration database: ", err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := db.ApplyMigrations(ctx, db.GetDB(), *dir); err != nil {
		log.Fatal("Migration failed; deployment must stop: ", err)
	}
	log.Println("All migrations committed successfully")
}
