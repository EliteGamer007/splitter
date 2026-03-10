package federation

import (
	"context"
	"fmt"
	"log"
	"time"

	"splitter/internal/db"
)

const (
	// PermanentFailureThreshold is how long a peer must be down before triggering
	// user migration (30 days).
	PermanentFailureThreshold = 30 * 24 * time.Hour
)

// MigratedUser records a user who was migrated from a failed instance.
type MigratedUser struct {
	OriginalDomain string    `json:"original_domain"`
	Username       string    `json:"username"`
	MigratedTo     string    `json:"migrated_to"`
	MigratedAt     time.Time `json:"migrated_at"`
	Notified       bool      `json:"notified"`
}

// EnsureMigrationTable creates the user_migrations table if it doesn't exist.
func EnsureMigrationTable(ctx context.Context) error {
	_, err := db.GetDB().Exec(ctx, `
		CREATE TABLE IF NOT EXISTS user_migrations (
			id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
			original_domain TEXT NOT NULL,
			username TEXT NOT NULL,
			display_name TEXT DEFAULT '',
			avatar_url TEXT DEFAULT '',
			original_actor_uri TEXT DEFAULT '',
			migrated_to_domain TEXT NOT NULL,
			migrated_at TIMESTAMPTZ DEFAULT now(),
			notified BOOLEAN DEFAULT false,
			notification_message TEXT DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT now(),
			UNIQUE(original_domain, username)
		)
	`)
	return err
}

// CheckAndMigrateUsers checks all peer domains for permanent failure and migrates
// their users to this instance. Should be called periodically from the worker loop.
func CheckAndMigrateUsers(ctx context.Context, selfDomain string) {
	for domain := range InstanceURLMap {
		if domain == selfDomain {
			continue
		}

		downDuration := GetPeerDownDuration(domain)
		if downDuration == 0 {
			// Also check DB for cross-restart persistence
			downDuration = GetPeerDownDurationFromDB(ctx, domain)
		}

		if downDuration < PermanentFailureThreshold {
			continue
		}

		log.Printf("[Migration] Peer %s has been down for %s (>30 days), checking for user migration...",
			domain, downDuration.Round(time.Hour))

		migrateUsersFromDomain(ctx, domain, selfDomain)
	}
}

// migrateUsersFromDomain migrates all ghost users from a failed domain to the local instance
// and sends them a notification message.
func migrateUsersFromDomain(ctx context.Context, failedDomain, selfDomain string) {
	// Find all remote actors (ghost users) from the failed domain
	rows, err := db.GetDB().Query(ctx, `
		SELECT ra.username, ra.display_name, ra.avatar_url, ra.actor_uri
		FROM remote_actors ra
		WHERE ra.domain = $1
		  AND NOT EXISTS (
			SELECT 1 FROM user_migrations um 
			WHERE um.original_domain = $1 AND um.username = ra.username
		  )
	`, failedDomain)
	if err != nil {
		log.Printf("[Migration] Failed to query remote actors from %s: %v", failedDomain, err)
		return
	}
	defer rows.Close()

	migratedCount := 0
	for rows.Next() {
		var username, displayName, avatarURL, actorURI string
		if err := rows.Scan(&username, &displayName, &avatarURL, &actorURI); err != nil {
			continue
		}

		notifMsg := fmt.Sprintf(
			"Your home instance %s has been unreachable for over 30 days. "+
				"Your account has been migrated to %s to preserve your connections and messages. "+
				"You can continue using Splitter through this instance.",
			failedDomain, selfDomain,
		)

		// Record migration
		_, err := db.GetDB().Exec(ctx, `
			INSERT INTO user_migrations (original_domain, username, display_name, avatar_url, original_actor_uri, migrated_to_domain, notification_message)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (original_domain, username) DO NOTHING
		`, failedDomain, username, displayName, avatarURL, actorURI, selfDomain, notifMsg)
		if err != nil {
			log.Printf("[Migration] Failed to record migration for %s@%s: %v", username, failedDomain, err)
			continue
		}

		// Update the ghost user's instance_domain to mark as migrated
		_, _ = db.GetDB().Exec(ctx, `
			UPDATE users SET instance_domain = $1
			WHERE instance_domain = $2 AND username = $3
		`, selfDomain+":migrated:"+failedDomain, failedDomain, username)

		// Create a system notification via messages
		sendMigrationNotification(ctx, username, failedDomain, selfDomain, notifMsg)

		migratedCount++
	}

	if migratedCount > 0 {
		log.Printf("[Migration] Migrated %d users from failed instance %s to %s", migratedCount, failedDomain, selfDomain)
	}
}

// sendMigrationNotification sends a DM from the system bot to notify about migration.
func sendMigrationNotification(ctx context.Context, username, fromDomain, toDomain, message string) {
	// Find the ghost user's local ID
	var userID string
	err := db.GetDB().QueryRow(ctx,
		`SELECT id::text FROM users WHERE username = $1 AND instance_domain LIKE '%' || $2 || '%' LIMIT 1`,
		username, fromDomain,
	).Scan(&userID)
	if err != nil || userID == "" {
		return
	}

	// Find the split bot user ID
	var botID string
	err = db.GetDB().QueryRow(ctx,
		`SELECT id::text FROM users WHERE username = 'split' LIMIT 1`,
	).Scan(&botID)
	if err != nil || botID == "" {
		return
	}

	// Create or get thread between bot and user
	var threadID string
	err = db.GetDB().QueryRow(ctx, `
		SELECT id::text FROM message_threads 
		WHERE (participant_a_id = $1 AND participant_b_id = $2)
		   OR (participant_a_id = $2 AND participant_b_id = $1)
		LIMIT 1
	`, botID, userID).Scan(&threadID)

	if err != nil || threadID == "" {
		err = db.GetDB().QueryRow(ctx, `
			INSERT INTO message_threads (participant_a_id, participant_b_id) 
			VALUES ($1, $2) RETURNING id::text
		`, botID, userID).Scan(&threadID)
		if err != nil {
			return
		}
	}

	// Send the notification message
	_, _ = db.GetDB().Exec(ctx, `
		INSERT INTO messages (thread_id, sender_id, recipient_id, content)
		VALUES ($1, $2, $3, $4)
	`, threadID, botID, userID, message)

	// Mark as notified
	_, _ = db.GetDB().Exec(ctx, `
		UPDATE user_migrations SET notified = true 
		WHERE original_domain = $1 AND username = $2
	`, fromDomain, username)
}

// GetMigrationStatus returns the list of migrated users for admin visibility.
func GetMigrationStatus(ctx context.Context) ([]MigratedUser, error) {
	rows, err := db.GetDB().Query(ctx, `
		SELECT original_domain, username, migrated_to_domain, migrated_at, notified
		FROM user_migrations
		ORDER BY migrated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []MigratedUser
	for rows.Next() {
		var m MigratedUser
		if err := rows.Scan(&m.OriginalDomain, &m.Username, &m.MigratedTo, &m.MigratedAt, &m.Notified); err != nil {
			continue
		}
		migrations = append(migrations, m)
	}
	return migrations, nil
}
