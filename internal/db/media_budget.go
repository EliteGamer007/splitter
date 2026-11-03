package db

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"strconv"
	"sync"
	"time"
)

const MaxStoredMedia = 256 << 10
const MaxStoredAvatar = 128 << 10
const mediaBudgetLock int64 = 734171933103

var ErrStorageQuota = errors.New("storage budget reached; existing content remains readable")
var sizeSnapshot struct {
	sync.Mutex
	pool *pgxpool.Pool
	size int64
	at   time.Time
}

func budgetLimit(name string, fallback int64) int64 {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}
func LockMediaBudget(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, mediaBudgetLock)
	return err
}

// CheckGrowthBudget uses a request-triggered, five-minute database size snapshot.
// It is a soft guard: other services/project history remain provider-level usage.
func CheckGrowthBudget(ctx context.Context, tx pgx.Tx, estimate int64) error {
	if estimate <= 0 {
		return nil
	}
	sizeSnapshot.Lock()
	defer sizeSnapshot.Unlock()
	if sizeSnapshot.pool != GetDB() || time.Since(sizeSnapshot.at) > 5*time.Minute {
		var size int64
		if err := tx.QueryRow(ctx, `SELECT pg_database_size(current_database())`).Scan(&size); err != nil {
			return err
		}
		sizeSnapshot.pool = GetDB()
		sizeSnapshot.size = size
		sizeSnapshot.at = time.Now()
	}
	if sizeSnapshot.size+estimate > budgetLimit("DB_WRITE_BUDGET_BYTES", 256<<20) {
		return ErrStorageQuota
	}
	return nil
}

// EnforceMediaBudget must run in the same transaction as the byte write. The
// database-wide lock makes post/avatar/story usage atomic across all processes.
func EnforceMediaBudget(ctx context.Context, tx pgx.Tx, ownerID string, delta int64) error {
	if err := LockMediaBudget(ctx, tx); err != nil {
		return err
	}
	if delta <= 0 {
		return nil
	}
	var total, owned int64
	err := tx.QueryRow(ctx, `SELECT
 COALESCE((SELECT sum(octet_length(media_data)) FROM media),0)+COALESCE((SELECT sum(octet_length(avatar_data))FROM users),0)+COALESCE((SELECT sum(octet_length(media_data))FROM stories),0),
 COALESCE((SELECT sum(octet_length(m.media_data))FROM media m JOIN posts p ON p.id=m.post_id JOIN users u ON u.did=p.author_did WHERE u.id=$1),0)+COALESCE((SELECT octet_length(avatar_data) FROM users WHERE id=$1),0)+COALESCE((SELECT sum(octet_length(media_data))FROM stories WHERE user_id=$1),0)`, ownerID).Scan(&total, &owned)
	if err != nil {
		return err
	}
	if total+delta > budgetLimit("MEDIA_TOTAL_BUDGET_BYTES", 32<<20) || owned+delta > budgetLimit("MEDIA_ACCOUNT_BUDGET_BYTES", 2<<20) {
		return ErrStorageQuota
	}
	return CheckGrowthBudget(ctx, tx, delta)
}
