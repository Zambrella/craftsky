// Package ops provides aggregate, read-only operator snapshots. It never builds
// application dependencies, starts workers, or selects private row payloads.
package ops

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Snapshot struct {
	CollectedAt    time.Time `json:"collectedAt"`
	RetryThreshold int       `json:"retryThreshold"`
	Queues         []Queue   `json:"queues"`
}

type Queue struct {
	Name             string           `json:"name"`
	Pending          int64            `json:"pending"`
	Due              int64            `json:"due"`
	Overdue          int64            `json:"overdue"`
	RepeatedAttempts int64            `json:"repeatedAttempts"`
	MaxAttempts      int              `json:"maxAttempts"`
	OldestPendingAt  *time.Time       `json:"oldestPendingAt"`
	OldestDueAt      *time.Time       `json:"oldestDueAt"`
	States           map[string]int64 `json:"states"`
}

// ReadSnapshot uses one bounded, repeatable read-only transaction. The retry
// threshold counts unfinished rows with repeated attempts, not proven failures.
// Due means ready for dispatch or lease recovery; blocked/manual work is not due.
func ReadSnapshot(ctx context.Context, pool *pgxpool.Pool, retryThreshold int) (Snapshot, error) {
	if pool == nil || retryThreshold < 1 || retryThreshold > 1000 {
		return Snapshot{}, fmt.Errorf("ops snapshot requires a database and retry threshold between 1 and 1000")
	}
	var snapshot Snapshot
	snapshot.RetryThreshold = retryThreshold
	err := pgx.BeginTxFunc(ctx, pool, pgx.TxOptions{
		IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly,
	}, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `SELECT CURRENT_TIMESTAMP`).Scan(&snapshot.CollectedAt); err != nil {
			return fmt.Errorf("read snapshot time: %w", err)
		}
		for _, source := range queueSources {
			queue := Queue{Name: source.name, States: map[string]int64{}}
			// Every fragment below is a fixed application-owned SQL definition.
			// No CLI or database value can become an SQL identifier/expression.
			query := `SELECT state, count(*), min(created_at),
				count(*) FILTER (WHERE due_at <= $1),
				count(*) FILTER (WHERE due_at <= $1 - interval '60 seconds'),
				min(due_at) FILTER (WHERE due_at <= $1),
				count(*) FILTER (WHERE attempts >= $2), COALESCE(max(attempts),0)
				FROM (` + source.sql + `) AS work GROUP BY state ORDER BY state`
			rows, err := tx.Query(ctx, query, snapshot.CollectedAt, retryThreshold)
			if err != nil {
				return fmt.Errorf("read %s queue: %w", source.name, err)
			}
			for rows.Next() {
				var state string
				var count, due, overdue, repeated int64
				var attempts int
				var oldest, oldestDue *time.Time
				if err := rows.Scan(&state, &count, &oldest, &due, &overdue, &oldestDue, &repeated, &attempts); err != nil {
					rows.Close()
					return fmt.Errorf("scan %s queue: %w", source.name, err)
				}
				queue.States[state] = count
				queue.Pending += count
				queue.Due += due
				queue.Overdue += overdue
				queue.RepeatedAttempts += repeated
				queue.MaxAttempts = max(queue.MaxAttempts, attempts)
				queue.OldestPendingAt = earlier(queue.OldestPendingAt, oldest)
				queue.OldestDueAt = earlier(queue.OldestDueAt, oldestDue)
			}
			if err := rows.Err(); err != nil {
				rows.Close()
				return fmt.Errorf("iterate %s queue: %w", source.name, err)
			}
			rows.Close()
			snapshot.Queues = append(snapshot.Queues, queue)
		}
		return nil
	})
	if err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

func earlier(current, candidate *time.Time) *time.Time {
	if candidate != nil && (current == nil || candidate.Before(*current)) {
		return candidate
	}
	return current
}

var queueSources = []struct{ name, sql string }{
	{"tapProjection", `SELECT state, attempts, created_at,
		CASE WHEN state='pending' THEN next_attempt_at WHEN state='processing' THEN lease_expires_at END AS due_at
		FROM tap_projection_jobs WHERE state IN ('pending','processing','blocked')`},
	{"tapRepository", `SELECT state, attempts, created_at,
		CASE WHEN state='pending' THEN next_attempt_at ELSE lease_expires_at END AS due_at
		FROM tap_repository_jobs WHERE state IN ('pending','processing')`},
	{"tapQuarantine", `SELECT replay_state AS state, 0 AS attempts, first_seen_at AS created_at,
		CASE WHEN replay_state='pending' THEN last_seen_at WHEN replay_state='processing' THEN lease_expires_at END AS due_at
		FROM tap_quarantined_events WHERE replay_state <> 'resolved'`},
	{"imageScanning", `SELECT state, attempts, created_at,
		CASE WHEN state='queued' THEN next_attempt_at WHEN state='leased' THEN lease_expires_at END AS due_at
		FROM image_scan_jobs`},
	{"pushDelivery", `SELECT status AS state, attempts, created_at,
		CASE WHEN status='leased' THEN lease_expires_at ELSE LEAST(next_attempt_at,deadline_at) END AS due_at
		FROM push_deliveries WHERE status IN ('pending','retry','leased')`},
	{"scheduledPublication", `SELECT posts.status AS state, posts.attempt_count AS attempts, posts.created_at,
		CASE WHEN posts.status IN ('scheduled','retrying') THEN posts.next_attempt_at
		WHEN posts.status='publishing' THEN posts.lease_expires_at END AS due_at
		FROM scheduled_posts posts JOIN owner_lifecycles lifecycle
		ON lifecycle.owner_did=posts.owner_did AND lifecycle.generation=posts.owner_generation AND lifecycle.state='active'`},
	{"scheduledMediaCleanup", `SELECT state, attempt_count AS attempts, created_at,
		CASE WHEN state='pending' THEN next_attempt_at ELSE lease_expires_at END AS due_at
		FROM scheduled_post_cleanup_jobs`},
	{"accountDeletion", `SELECT state, attempt_count AS attempts, created_at,
		CASE WHEN state <> 'intent' THEN COALESCE(lease_expires_at,next_attempt_at) END AS due_at
		FROM account_deletion_operations`},
	{"terminalPurge", `SELECT state, attempts, created_at,
		CASE WHEN state='pending' THEN next_attempt_at ELSE lease_expires_at END AS due_at
		FROM owner_purge_components WHERE state <> 'complete'`},
	{"authCleanup", `SELECT state, attempt_count AS attempts, created_at,
		CASE WHEN state='pending' THEN next_attempt_at WHEN state='leased' THEN lease_expires_at END AS due_at
		FROM auth_auxiliary_cleanup_jobs WHERE state <> 'complete'`},
	{"moderationCases", `SELECT state, 0 AS attempts, created_at, NULL::timestamptz AS due_at
		FROM moderation_cases WHERE state='open'`},
	{"moderationAppeals", `SELECT status AS state, 0 AS attempts, confirmed_at AS created_at, NULL::timestamptz AS due_at
		FROM moderation_appeals WHERE status='pending'`},
	{"moderationRestoration", `SELECT status AS state, 0 AS attempts, created_at, created_at AS due_at
		FROM moderation_restoration_outbox WHERE status='pending'`},
}
