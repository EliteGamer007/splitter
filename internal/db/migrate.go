package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RunMigrations is explicit: web startup never runs migrations.
func RunMigrations(dir string) error {
	if DB == nil {
		return fmt.Errorf("database is not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	return ApplyMigrations(ctx, DB, dir)
}

// ApplyMigrations serializes upgrades and records only committed SQL. The master
// baseline supersedes 001; 002 and later upgrades support existing databases.
func ApplyMigrations(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	files := []string{}
	for _, e := range entries {
		n := e.Name()
		if !e.IsDir() && strings.HasSuffix(n, ".sql") && n != "001_initial_schema.sql" && n != "verify_migration.sql" {
			files = append(files, n)
		}
	}
	sort.Strings(files)
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	const lockID int64 = 827416015
	if _, err = conn.Exec(ctx, "SELECT pg_advisory_lock($1)", lockID); err != nil {
		return err
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		conn.Exec(unlockCtx, "SELECT pg_advisory_unlock($1)", lockID)
	}()
	var schema string
	if err = conn.QueryRow(ctx, "SELECT current_schema()").Scan(&schema); err != nil || schema == "" {
		return fmt.Errorf("migration target schema is not available")
	}
	ledger := pgx.Identifier{schema, "schema_migrations"}.Sanitize()
	if _, err = conn.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS "uuid-ossp" WITH SCHEMA public`); err != nil {
		return fmt.Errorf("initialize UUID extension: %w", err)
	}
	// Never resolve a sibling/public ledger through search_path.
	if _, err = conn.Exec(ctx, "CREATE TABLE IF NOT EXISTS "+ledger+"(version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ DEFAULT now()); ALTER TABLE "+ledger+" ADD COLUMN IF NOT EXISTS checksum TEXT"); err != nil {
		return err
	}

	for _, name := range files {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return err
		}
		sum := sha256.Sum256(content)
		checksum := hex.EncodeToString(sum[:])
		var previous *string
		err = conn.QueryRow(ctx, "SELECT checksum FROM "+ledger+" WHERE version=$1", name).Scan(&previous)
		if err == nil {
			if previous != nil && *previous != checksum {
				return fmt.Errorf("migration %s checksum changed; create an additive migration instead", name)
			}
			continue
		}
		if err != pgx.ErrNoRows {
			return err
		}
		if name == "000_master_schema.sql" {
			var exists bool
			if err = conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema=current_schema() AND table_name='users')").Scan(&exists); err != nil {
				return err
			}
			if exists {
				continue
			} // never replay CREATE TABLE over a legacy application's data
		}
		tx, err := conn.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, string(content)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("migration %s failed (rolled back): %w", name, err)
		}
		if _, err = tx.Exec(ctx, "INSERT INTO "+ledger+"(version,checksum) VALUES($1,$2)", name, checksum); err != nil {
			tx.Rollback(ctx)
			return err
		}
		if err = tx.Commit(ctx); err != nil {
			return fmt.Errorf("commit %s: %w", name, err)
		}
	}
	return nil
}
