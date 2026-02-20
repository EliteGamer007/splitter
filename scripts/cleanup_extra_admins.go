package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type adminRow struct {
	ID             string
	Username       string
	InstanceDomain string
	CreatedAt      string
}

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

	rows, err := pool.Query(ctx, `
		SELECT id, username, COALESCE(instance_domain, ''), created_at::text
		FROM users
		WHERE role = 'admin'
		ORDER BY instance_domain ASC, created_at ASC
	`)
	if err != nil {
		log.Fatalf("failed to query admins: %v", err)
	}
	defer rows.Close()

	adminsByDomain := map[string][]adminRow{}
	for rows.Next() {
		var row adminRow
		if scanErr := rows.Scan(&row.ID, &row.Username, &row.InstanceDomain, &row.CreatedAt); scanErr != nil {
			log.Fatalf("scan error: %v", scanErr)
		}
		domain := row.InstanceDomain
		if domain == "" {
			domain = "localhost"
		}
		adminsByDomain[domain] = append(adminsByDomain[domain], row)
	}

	if len(adminsByDomain) == 0 {
		fmt.Println("No admin users found; nothing to do.")
		return
	}

	for domain, admins := range adminsByDomain {
		if len(admins) <= 1 {
			fmt.Printf("Domain %s: keeping %s (only admin)\n", domain, admins[0].Username)
			continue
		}

		keeper := admins[0]
		fmt.Printf("Domain %s: keeping main admin %s (%s)\n", domain, keeper.Username, keeper.ID)
		for _, admin := range admins[1:] {
			_, execErr := pool.Exec(ctx, `UPDATE users SET role = 'user', updated_at = NOW() WHERE id = $1`, admin.ID)
			if execErr != nil {
				log.Fatalf("failed to demote admin %s: %v", admin.Username, execErr)
			}
			fmt.Printf("  demoted %s (%s)\n", admin.Username, admin.ID)
		}
	}

	fmt.Println("Admin cleanup completed.")
}
