// Command bootstrap-admin provisions the first administrator explicitly.
// Credentials are read from BOOTSTRAP_ADMIN_USERNAME, BOOTSTRAP_ADMIN_EMAIL and
// BOOTSTRAP_ADMIN_PASSWORD; they are never printed or passed as process flags.
package main

import (
	"context"
	"errors"
	"log"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"splitter/internal/auth"
	"splitter/internal/config"
	"splitter/internal/db"
)

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
	log.Print("Administrator bootstrap completed")
}
func run() error {
	username, email, password := os.Getenv("BOOTSTRAP_ADMIN_USERNAME"), os.Getenv("BOOTSTRAP_ADMIN_EMAIL"), os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	// Validate supplied credentials before initializing any database connection.
	if err := validateCredentials(username, email, password); err != nil {
		return err
	}
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		return errors.New("invalid deployment configuration")
	}
	if err := db.InitDB(cfg); err != nil {
		return errors.New("cannot initialize database; check configuration and connectivity")
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return bootstrapAdmin(ctx, db.GetDB(), username, email, password, cfg.Federation.Domain)
}
func validateCredentials(username, email, password string) error {
	if username != strings.TrimSpace(username) || len(username) < 3 || len(username) > 50 {
		return errors.New("supply BOOTSTRAP_ADMIN_USERNAME with 3-50 characters and no surrounding spaces")
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return errors.New("supply a valid BOOTSTRAP_ADMIN_EMAIL")
	}
	if len(password) < 12 || len(password) > 72 {
		return errors.New("supply BOOTSTRAP_ADMIN_PASSWORD with 12-72 bytes")
	}
	return nil
}
func bootstrapAdmin(ctx context.Context, pool *pgxpool.Pool, username, email, password, domain string) error {
	if err := validateCredentials(username, email, password); err != nil {
		return err
	}
	if pool == nil || strings.TrimSpace(domain) == "" {
		return errors.New("administrator bootstrap requires a database and instance domain")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return errors.New("cannot prepare administrator credentials")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return errors.New("cannot begin administrator bootstrap")
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", int64(827416016)); err != nil {
		return errors.New("cannot lock administrator bootstrap")
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE role='admin' OR username=$1 OR lower(email)=lower($2))`, username, email).Scan(&exists); err != nil {
		return errors.New("cannot inspect administrator state; apply explicit migrations first")
	}
	if exists {
		return errors.New("administrator or supplied account already exists; no accounts were changed")
	}
	_, err = tx.Exec(ctx, `INSERT INTO users(username,email,password_hash,instance_domain,did,display_name,public_key,role) VALUES($1,$2,$3,$4,$5,$1,'','admin')`, username, email, hash, domain, auth.GenerateSimpleDID(username))
	if err != nil {
		return errors.New("cannot create administrator; no accounts were changed")
	}
	if err := tx.Commit(ctx); err != nil {
		return errors.New("administrator bootstrap did not commit")
	}
	return nil
}
