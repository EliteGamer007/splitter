package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	dbs := map[string]string{
		"neondb":   "postgres://neondb_owner:npg_doQ6W7BuhytJ@ep-falling-mode-a1k832j8-pooler.ap-southeast-1.aws.neon.tech:5432/neondb?sslmode=require",
		"neondb_2": "postgres://neondb_owner:npg_doQ6W7BuhytJ@ep-falling-mode-a1k832j8-pooler.ap-southeast-1.aws.neon.tech:5432/neondb_2?sslmode=require",
	}

	for name, conn := range dbs {
		fmt.Printf("\n=== %s ===\n", name)
		pool, err := pgxpool.New(context.Background(), conn)
		if err != nil {
			fmt.Printf("connect error: %v\n", err)
			continue
		}

		var total int
		_ = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM users").Scan(&total)
		fmt.Printf("total users: %d\n", total)

		fmt.Println("by instance_domain:")
		rows, _ := pool.Query(context.Background(), `
			SELECT COALESCE(instance_domain, '<NULL>') as d, COUNT(*)
			FROM users GROUP BY d ORDER BY COUNT(*) DESC, d ASC`)
		for rows.Next() {
			var d string
			var c int
			_ = rows.Scan(&d, &c)
			fmt.Printf("  %s -> %d\n", d, c)
		}
		rows.Close()

		var testCount int
		_ = pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM users WHERE username ILIKE '%test%'").Scan(&testCount)
		fmt.Printf("test users: %d\n", testCount)

		fmt.Println("sample test users:")
		rows2, _ := pool.Query(context.Background(), `
			SELECT username, COALESCE(instance_domain,''), COALESCE(did,'')
			FROM users
			WHERE username ILIKE '%test%'
			ORDER BY created_at DESC
			LIMIT 20`)
		for rows2.Next() {
			var u, d, did string
			_ = rows2.Scan(&u, &d, &did)
			fmt.Printf("  %s | domain=%s | did=%s\n", u, d, did)
		}
		rows2.Close()

		pool.Close()
	}
}
