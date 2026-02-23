package federation

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"splitter/internal/db"
)

var (
	deliveryPolicyMu        sync.RWMutex
	maxRetryCount           = 6
	circuitFailureThreshold = 5
	circuitCooldown         = 5 * time.Minute
)

// ConfigureDeliveryPolicy updates runtime retry/circuit breaker policy.
func ConfigureDeliveryPolicy(maxRetries, failureThreshold int, cooldown time.Duration) {
	deliveryPolicyMu.Lock()
	defer deliveryPolicyMu.Unlock()

	if maxRetries > 0 {
		maxRetryCount = maxRetries
	}
	if failureThreshold > 0 {
		circuitFailureThreshold = failureThreshold
	}
	if cooldown > 0 {
		circuitCooldown = cooldown
	}
}

func currentDeliveryPolicy() (int, int, time.Duration) {
	deliveryPolicyMu.RLock()
	defer deliveryPolicyMu.RUnlock()
	return maxRetryCount, circuitFailureThreshold, circuitCooldown
}

func calculateRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}

	base := 15 * time.Second
	multiplier := math.Pow(2, float64(attempt-1))
	delay := time.Duration(multiplier) * base
	if delay > time.Hour {
		return time.Hour
	}
	return delay
}

func formatBackoffInterval(duration time.Duration) string {
	seconds := int(duration.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	return fmt.Sprintf("%d seconds", seconds)
}

func recordDeliveryFailure(ctx context.Context, domain string) {
	if domain == "" {
		return
	}

	_, threshold, cooldown := currentDeliveryPolicy()

	_, _ = db.GetDB().Exec(ctx, `
		INSERT INTO federation_failures (domain, failure_count, last_failure_at, circuit_open_until)
		VALUES ($1, 1, now(), NULL)
		ON CONFLICT (domain) DO UPDATE SET
			failure_count = federation_failures.failure_count + 1,
			last_failure_at = now(),
			circuit_open_until = CASE
				WHEN federation_failures.failure_count + 1 >= $2 THEN now() + $3::interval
				ELSE federation_failures.circuit_open_until
			END
	`, domain, threshold, formatBackoffInterval(cooldown))
}

func recordDeliverySuccess(ctx context.Context, domain string) {
	if domain == "" {
		return
	}

	_, _ = db.GetDB().Exec(ctx, `
		INSERT INTO federation_failures (domain, failure_count, last_failure_at, circuit_open_until)
		VALUES ($1, 0, NULL, NULL)
		ON CONFLICT (domain) DO UPDATE SET
			failure_count = 0,
			last_failure_at = NULL,
			circuit_open_until = NULL
	`, domain)
}

func recordFederationConnection(ctx context.Context, sourceDomain, targetDomain, status string) {
	if sourceDomain == "" || targetDomain == "" || sourceDomain == targetDomain {
		return
	}

	_, _ = db.GetDB().Exec(ctx, `
		INSERT INTO federation_connections (
			source_domain, target_domain, success_count, failure_count, last_status, last_seen, updated_at
		) VALUES (
			$1,
			$2,
			CASE WHEN $3 = 'sent' THEN 1 ELSE 0 END,
			CASE WHEN $3 = 'failed' THEN 1 ELSE 0 END,
			$3,
			now(),
			now()
		)
		ON CONFLICT (source_domain, target_domain) DO UPDATE SET
			success_count = federation_connections.success_count + CASE WHEN EXCLUDED.last_status = 'sent' THEN 1 ELSE 0 END,
			failure_count = federation_connections.failure_count + CASE WHEN EXCLUDED.last_status = 'failed' THEN 1 ELSE 0 END,
			last_status = EXCLUDED.last_status,
			last_seen = now(),
			updated_at = now()
	`, sourceDomain, targetDomain, status)
}

// IsCircuitOpen returns true when the domain is temporarily disabled for delivery.
func IsCircuitOpen(ctx context.Context, domain string) bool {
	if domain == "" {
		return false
	}

	var openUntil *time.Time
	err := db.GetDB().QueryRow(ctx,
		`SELECT circuit_open_until FROM federation_failures WHERE domain = $1`,
		domain,
	).Scan(&openUntil)
	if err != nil {
		return false
	}

	if openUntil == nil {
		return false
	}

	return openUntil.After(time.Now().UTC())
}

// RetryOutboxBatch retries due outbox entries (pending/failed) with exponential backoff.
func RetryOutboxBatch(ctx context.Context, batchSize int) (int, int, error) {
	if batchSize <= 0 {
		batchSize = 25
	}
	maxRetries, _, _ := currentDeliveryPolicy()

	rows, err := db.GetDB().Query(ctx, `
		SELECT id::text
		FROM outbox_activities
		WHERE status IN ('pending','failed')
		  AND retry_count < $1
		  AND COALESCE(next_retry_at, now()) <= now()
		ORDER BY created_at ASC
		LIMIT $2
	`, maxRetries, batchSize)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	processed := 0
	failed := 0
	for rows.Next() {
		var outboxID string
		if scanErr := rows.Scan(&outboxID); scanErr != nil {
			continue
		}

		processed++
		if retryErr := RetryOutboxActivity(ctx, outboxID); retryErr != nil {
			failed++
			log.Printf("[FederationWorker] Retry failed for outbox=%s: %v", outboxID, retryErr)
		}
	}

	return processed, failed, nil
}

// RecalculateInstanceReputation recomputes domain reputation using spam + failure signals.
func RecalculateInstanceReputation(ctx context.Context) error {
	domainsRows, err := db.GetDB().Query(ctx, `
		WITH outbox_domains AS (
			SELECT DISTINCT regexp_replace(target_inbox, '^https?://([^/]+)/?.*$', '\\1') AS domain
			FROM outbox_activities
			WHERE target_inbox ILIKE 'http%'
		)
		SELECT DISTINCT domain FROM (
			SELECT domain FROM remote_actors WHERE COALESCE(domain, '') <> ''
			UNION SELECT domain FROM blocked_domains
			UNION SELECT domain FROM federation_failures
			UNION SELECT domain FROM outbox_domains WHERE COALESCE(domain, '') <> ''
		) d
		WHERE COALESCE(domain, '') <> ''
	`)
	if err != nil {
		return err
	}
	defer domainsRows.Close()

	for domainsRows.Next() {
		var domain string
		if scanErr := domainsRows.Scan(&domain); scanErr != nil {
			continue
		}

		var spamSignals int
		var failureSignals int
		var successSignals int

		_ = db.GetDB().QueryRow(ctx, `
			SELECT COUNT(*)
			FROM admin_actions
			WHERE action_type = 'block_domain'
			  AND target = $1
			  AND (LOWER(COALESCE(reason, '')) LIKE '%spam%' OR LOWER(COALESCE(reason, '')) LIKE '%abuse%')
			  AND created_at > now() - interval '30 days'
		`, domain).Scan(&spamSignals)

		_ = db.GetDB().QueryRow(ctx, `
			SELECT COUNT(*)
			FROM outbox_activities
			WHERE target_inbox ILIKE '%' || $1 || '%'
			  AND status = 'failed'
			  AND created_at > now() - interval '24 hours'
		`, domain).Scan(&failureSignals)

		_ = db.GetDB().QueryRow(ctx, `
			SELECT COUNT(*)
			FROM outbox_activities
			WHERE target_inbox ILIKE '%' || $1 || '%'
			  AND status = 'sent'
			  AND created_at > now() - interval '24 hours'
		`, domain).Scan(&successSignals)

		score := 100 - (spamSignals * 20) - (failureSignals * 5) + (successSignals * 2)
		if score < 0 {
			score = 0
		}
		if score > 100 {
			score = 100
		}

		_, _ = db.GetDB().Exec(ctx, `
			INSERT INTO instance_reputation (domain, reputation_score, spam_count, failure_count, updated_at)
			VALUES ($1, $2, $3, $4, now())
			ON CONFLICT (domain) DO UPDATE SET
				reputation_score = EXCLUDED.reputation_score,
				spam_count = EXCLUDED.spam_count,
				failure_count = EXCLUDED.failure_count,
				updated_at = now()
		`, domain, score, spamSignals, failureSignals)
	}

	return nil
}
