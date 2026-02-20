package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	_ = godotenv.Load(envFile)

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer pool.Close()

	query := `
		DELETE FROM posts p
		USING users u
		WHERE p.author_did = u.did
		  AND (
			u.username ILIKE 'fed1_%'
			OR u.username ILIKE 'fed2_%'
			OR u.username ILIKE 'nofollow%'
			OR u.username ILIKE 'signupfixuser%'
			OR u.username ILIKE 'testuser_final-%'
			OR p.content ILIKE 'FED_POST_%'
			OR p.content ILIKE 'UNFOLLOWED_REMOTE_%'
			OR p.content ILIKE 'Federated Message Test %'
		  )
	`

	result, err := pool.Exec(ctx, query)
	if err != nil {
		log.Fatalf("failed to cleanup test posts: %v", err)
	}

	fmt.Printf("Deleted %d noisy test posts from %s\n", result.RowsAffected(), os.Getenv("DB_NAME"))
}
