package safetyalerts

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/observability"
)

type Sender interface {
	Send(context.Context, uuid.UUID) error
}

type Worker struct {
	pool     *pgxpool.Pool
	sender   Sender
	observer *observability.Observer
	now      func() time.Time
}

func NewWorker(pool *pgxpool.Pool, sender Sender, observer *observability.Observer) *Worker {
	return &Worker{pool: pool, sender: sender, observer: observer, now: time.Now}
}

// Source discovery reads only IDs and classifications. One source alert is
// persisted per open case/incident/intake/workflow, including the existing backlog.
// A digest contains at most 500 sources. It never depends on a moderator visit.
const discoverSQL = `
WITH candidates AS (
 SELECT DISTINCT 'case'::text kind,c.id FROM moderation_cases c
 JOIN moderation_case_reports cr ON cr.case_id=c.id
 JOIN moderation_reports r ON r.id=cr.report_id
 WHERE c.state='open' AND r.reason_type IN (
 'childSexualExploitation','grooming','childAbuseMaterial','childSexualAbuse','childGrooming','childEndangerment',
 'intimateImageAbuse','nonConsensualIntimateImage','sexualExploitation',
 'immediateDanger','credibleThreat','terrorism','violence','selfHarm','stalking','doxxing')
 UNION ALL SELECT 'incident',id FROM safety_incidents WHERE state IN ('detected','confirmed','caseCreated','reporting')
 UNION ALL SELECT 'intake',id FROM external_safety_intakes WHERE urgency IN ('priority','urgent','immediate')
 UNION ALL SELECT 'workflow',id FROM safety_workflows WHERE state IN ('open','inProgress') AND workflow_class IN ('intimateImage','credibleThreat')
), fresh AS (
 SELECT kind,id FROM candidates c WHERE NOT EXISTS (
 SELECT 1 FROM safety_email_sources s WHERE s.source_kind=c.kind AND s.source_id=c.id
 ) ORDER BY kind,id LIMIT 500
)
INSERT INTO safety_email_sources(source_kind,source_id,delivery_id)
SELECT kind,id,$1 FROM fresh ON CONFLICT DO NOTHING`

// ProcessBatch uses a short transaction to lease one durable digest. The global
// transaction lock serializes discovery/claims across processes, never HTTP.
func (w *Worker) ProcessBatch(ctx context.Context) (int, error) {
	id, token, attempt, err := w.claim(ctx)
	if err != nil || id == uuid.Nil {
		return 0, err
	}
	sendCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	sendErr := w.sender.Send(sendCtx, id)
	cancel()
	now := w.now().UTC()
	if sendErr != nil {
		w.observer.ObservePrivateFailure(ctx, sendErr, "", id.String(), "safety_email.send", "provider", "retry", attempt)
		delay := retryDelay(attempt)
		_, err = w.pool.Exec(ctx, `UPDATE safety_email_deliveries SET next_attempt_at=$3,lease_token=NULL,lease_until=NULL WHERE id=$1 AND lease_token=$2 AND sent_at IS NULL`, id, token, now.Add(delay))
	} else {
		_, err = w.pool.Exec(ctx, `UPDATE safety_email_deliveries SET sent_at=$3,lease_token=NULL,lease_until=NULL WHERE id=$1 AND lease_token=$2 AND sent_at IS NULL`, id, token, now)
	}
	if err != nil {
		return 0, fmt.Errorf("finalize safety email delivery: %w", err)
	}
	// The next call observes the one-minute delivery spacing below.
	return 1, nil
}

func retryDelay(attempt int) time.Duration {
	delay := time.Minute
	for i := 1; i < attempt && delay < time.Hour; i++ {
		delay *= 2
	}
	return min(delay, time.Hour)
}

func (w *Worker) claim(ctx context.Context) (id, token uuid.UUID, attempt int, err error) {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var locked bool
	if err = tx.QueryRow(ctx, `SELECT pg_try_advisory_xact_lock(17482696,121)`).Scan(&locked); err != nil || !locked {
		return
	}
	now := w.now().UTC()
	// Remove orphaned references when their private source is deleted. Delivery
	// timing metadata is retained for 30 days once no source needs its dedupe key.
	if _, err = tx.Exec(ctx, `DELETE FROM safety_email_sources s WHERE
        (source_kind='case' AND NOT EXISTS(SELECT 1 FROM moderation_cases c WHERE c.id=s.source_id)) OR
        (source_kind='incident' AND NOT EXISTS(SELECT 1 FROM safety_incidents i WHERE i.id=s.source_id)) OR
        (source_kind='intake' AND NOT EXISTS(SELECT 1 FROM external_safety_intakes i WHERE i.id=s.source_id)) OR
        (source_kind='workflow' AND NOT EXISTS(SELECT 1 FROM safety_workflows f WHERE f.id=s.source_id))`); err != nil {
		return
	}
	if _, err = tx.Exec(ctx, `DELETE FROM safety_email_deliveries d WHERE d.sent_at < $1 AND NOT EXISTS(SELECT 1 FROM safety_email_sources s WHERE s.delivery_id=d.id)`, now.Add(-30*24*time.Hour)); err != nil {
		return
	}
	// Bound outgoing volume to one digest/minute, including after process restart.
	var recentlySent bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM safety_email_deliveries WHERE sent_at > $1)`, now.Add(-time.Minute)).Scan(&recentlySent); err != nil {
		return
	}
	if recentlySent {
		err = tx.Commit(ctx)
		return
	}
	// One outstanding digest at a time. Expired leases are retried after crashes.
	var next time.Time
	var until *time.Time
	err = tx.QueryRow(ctx, `SELECT id,attempts,next_attempt_at,lease_until FROM safety_email_deliveries WHERE sent_at IS NULL ORDER BY created_at,id LIMIT 1 FOR UPDATE`).Scan(&id, &attempt, &next, &until)
	if errors.Is(err, pgx.ErrNoRows) {
		err = nil
		id = uuid.New()
		if _, err = tx.Exec(ctx, `INSERT INTO safety_email_deliveries(id,created_at,next_attempt_at) VALUES($1,$2,$2)`, id, now); err != nil {
			return
		}
		var result int64
		tag, e := tx.Exec(ctx, discoverSQL, id)
		err = e
		if err != nil {
			return
		}
		result = tag.RowsAffected()
		if result == 0 {
			if _, err = tx.Exec(ctx, `DELETE FROM safety_email_deliveries WHERE id=$1`, id); err != nil {
				return
			}
			id = uuid.Nil
			err = tx.Commit(ctx)
			return
		}
	} else if err != nil {
		return
	} else if next.After(now) || until != nil && until.After(now) {
		id = uuid.Nil
		err = tx.Commit(ctx)
		return
	}
	token = uuid.New()
	attempt++
	_, err = tx.Exec(ctx, `UPDATE safety_email_deliveries SET attempts=$2,lease_token=$3,lease_until=$4 WHERE id=$1`, id, attempt, token, now.Add(time.Minute))
	if err != nil {
		return
	}
	err = tx.Commit(ctx)
	return
}
