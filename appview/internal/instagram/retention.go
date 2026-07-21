package instagram

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const MaxRetentionBatch = 500

var ErrInvalidRetentionBatch = errors.New("Instagram retention batch must be between 1 and 500")

// RetentionStats contains counts only. It is safe for operator output and
// observability because it never contains an owner, username, IGSID, digest,
// or private graph fact.
type RetentionStats struct {
	AttemptsTerminalized   int
	AttemptsSensitiveClear int
	AttemptsPurged         int
	WebhookTerminalized    int
	WebhookSensitiveClear  int
	WebhookPurged          int
	LinksMembershipExpired int
	LinkIdentityCleared    int
	LinkTombstonesPurged   int
	ClaimsPurged           int
	ConflictsExpired       int
	ConflictsPurged        int
	ImportsPurged          int
	SuggestionsPurged      int
	DeliveriesPurged       int
	NotificationsPurged    int
	RateBucketsPurged      int
	AuditsPurged           int
}

// RetentionService applies the fixed privacy maxima from the Instagram
// migration requirements. Each record class is selected in a stable order and
// in a transaction bounded by the caller's batch size. Cascading dependants do
// not count against the primary-record batch.
type RetentionService struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewRetentionService(pool *pgxpool.Pool, now func() time.Time) *RetentionService {
	if now == nil {
		now = time.Now
	}
	return &RetentionService{pool: pool, now: now}
}

func (s *RetentionService) Run(ctx context.Context, batch int) (RetentionStats, error) {
	if err := s.validate(batch); err != nil {
		return RetentionStats{}, err
	}
	now := s.now().UTC()
	if now.IsZero() {
		return RetentionStats{}, errors.New("Instagram retention clock returned zero time")
	}
	stats := RetentionStats{}
	steps := []struct {
		destination *int
		run         func(context.Context, int, time.Time) (int, error)
	}{
		{&stats.AttemptsTerminalized, s.terminalizeAttempts},
		{&stats.AttemptsSensitiveClear, s.clearTerminalAttemptIdentity},
		{&stats.AttemptsPurged, s.purgeTerminalAttempts},
		{&stats.WebhookTerminalized, s.terminalizeStaleWebhookWork},
		{&stats.WebhookSensitiveClear, s.clearTerminalWebhookIdentity},
		{&stats.WebhookPurged, s.purgeTerminalWebhookWork},
		{&stats.ConflictsExpired, s.expireConflicts},
		{&stats.LinksMembershipExpired, s.expireInactiveLinks},
		{&stats.LinkIdentityCleared, s.clearTerminalLinkIdentity},
		{&stats.ClaimsPurged, s.purgeReleasedClaims},
		{&stats.LinkTombstonesPurged, s.purgeLinkTombstones},
		{&stats.ConflictsPurged, s.purgeResolvedConflicts},
	}
	for _, step := range steps {
		count, err := step.run(ctx, batch, now)
		if err != nil {
			return stats, err
		}
		*step.destination = count
	}
	if err := s.prepareNonconsentedImports(ctx, batch, now); err != nil {
		return stats, err
	}
	imports, err := s.purgeExpiredImportsAt(ctx, batch, now)
	if err != nil {
		return stats, err
	}
	stats.ImportsPurged = imports
	remaining := []struct {
		destination *int
		run         func(context.Context, int, time.Time) (int, error)
	}{
		{&stats.SuggestionsPurged, s.purgeTerminalSuggestions},
		{&stats.DeliveriesPurged, s.purgeRetractedDeliveries},
		{&stats.NotificationsPurged, s.purgeMatchNotifications},
		{&stats.RateBucketsPurged, s.purgeRateBuckets},
		{&stats.AuditsPurged, s.purgeAuditEvents},
	}
	for _, step := range remaining {
		count, err := step.run(ctx, batch, now)
		if err != nil {
			return stats, err
		}
		*step.destination = count
	}
	return stats, nil
}

// PurgeExpiredImports is the narrow operator primitive. It performs the same
// dependency-safe import cleanup as Run and never processes more than 500
// primary imports.
func (s *RetentionService) PurgeExpiredImports(ctx context.Context, batch int) (int, error) {
	if err := s.validate(batch); err != nil {
		return 0, err
	}
	now := s.now().UTC()
	if now.IsZero() {
		return 0, errors.New("Instagram retention clock returned zero time")
	}
	if err := s.prepareNonconsentedImports(ctx, batch, now); err != nil {
		return 0, err
	}
	return s.purgeExpiredImportsAt(ctx, batch, now)
}

func (s *RetentionService) validate(batch int) error {
	if s == nil || s.pool == nil {
		return errors.New("Instagram retention service is unavailable")
	}
	if batch < 1 || batch > MaxRetentionBatch {
		return ErrInvalidRetentionBatch
	}
	return nil
}

func (s *RetentionService) terminalizeAttempts(ctx context.Context, batch int, now time.Time) (int, error) {
	count := 0
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		ids, err := retentionUUIDs(ctx, tx, `
			SELECT id
			FROM instagram_verification_attempts
			WHERE state IN ('pendingDm','processing','pendingConfirmation')
			  AND (
				(state='pendingDm' AND expires_at <= $1)
				OR (state='processing' AND COALESCE(processing_started_at,updated_at,created_at) <= $1::timestamptz - interval '15 minutes')
				OR created_at <= $1::timestamptz - interval '24 hours'
			  )
			ORDER BY LEAST(
				expires_at,
				created_at + interval '24 hours',
				COALESCE(processing_started_at,updated_at,created_at) + interval '15 minutes'
			), id
			FOR UPDATE SKIP LOCKED
			LIMIT $2
		`, now, batch)
		if err != nil {
			return fmt.Errorf("select expired Instagram attempts: %w", err)
		}
		if len(ids) == 0 {
			return nil
		}
		if _, err := tx.Exec(ctx, `
			UPDATE instagram_webhook_work
			SET status='ignored',sender_igsid=NULL,official_account_id=NULL,
			    challenge_digest_version=NULL,challenge_digest=NULL,
			    lease_token=NULL,lease_expires_at=NULL,
			    terminal_at=COALESCE(terminal_at,$2),
			    terminal_reason=COALESCE(terminal_reason,'challengeUnavailable'),updated_at=$2
			WHERE verification_attempt_id=ANY($1::uuid[])
			  AND status IN ('queued','processing','retryable')
		`, ids, now); err != nil {
			return fmt.Errorf("terminalize expired-attempt webhook work: %w", err)
		}
		result, err := tx.Exec(ctx, `
			UPDATE instagram_verification_attempts
			SET state=CASE WHEN state='processing' THEN 'rejected' ELSE 'expired' END,
			    challenge_digest_version=NULL,challenge_digest=NULL,
			    candidate_igsid=NULL,candidate_username=NULL,retry_code=NULL,
			    terminal_at=COALESCE(terminal_at,$2),updated_at=$2
			WHERE id=ANY($1::uuid[])
		`, ids, now)
		if err != nil {
			return fmt.Errorf("terminalize expired Instagram attempts: %w", err)
		}
		count = int(result.RowsAffected())
		return nil
	})
	return count, err
}

func (s *RetentionService) clearTerminalAttemptIdentity(ctx context.Context, batch int, now time.Time) (int, error) {
	return s.updateUUIDBatch(ctx, batch, `
		SELECT id FROM instagram_verification_attempts
		WHERE state IN ('confirmed','expired','cancelled','superseded','rejected','conflicted')
		  AND $1::timestamptz IS NOT NULL
		  AND num_nonnulls(challenge_digest,candidate_igsid,candidate_username) > 0
		ORDER BY COALESCE(terminal_at,updated_at),id
		FOR UPDATE SKIP LOCKED LIMIT $2
	`, `
		UPDATE instagram_verification_attempts
		SET challenge_digest_version=NULL,challenge_digest=NULL,
		    candidate_igsid=NULL,candidate_username=NULL,updated_at=$2
		WHERE id=ANY($1::uuid[])
	`, now, "clear terminal Instagram attempt identity")
}

func (s *RetentionService) purgeTerminalAttempts(ctx context.Context, batch int, now time.Time) (int, error) {
	count := 0
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		ids, err := retentionUUIDs(ctx, tx, `
			SELECT id FROM instagram_verification_attempts
			WHERE state IN ('confirmed','expired','cancelled','superseded','rejected','conflicted')
			  AND COALESCE(terminal_at,updated_at) <= $1::timestamptz - interval '30 days'
			ORDER BY COALESCE(terminal_at,updated_at),id
			FOR UPDATE SKIP LOCKED LIMIT $2
		`, now, batch)
		if err != nil || len(ids) == 0 {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE instagram_webhook_work SET verification_attempt_id=NULL WHERE verification_attempt_id=ANY($1::uuid[])`, ids); err != nil {
			return fmt.Errorf("detach retained Instagram webhook replay rows: %w", err)
		}
		result, err := tx.Exec(ctx, `DELETE FROM instagram_verification_attempts WHERE id=ANY($1::uuid[])`, ids)
		if err != nil {
			return fmt.Errorf("purge terminal Instagram attempts: %w", err)
		}
		count = int(result.RowsAffected())
		return nil
	})
	return count, err
}

func (s *RetentionService) terminalizeStaleWebhookWork(ctx context.Context, batch int, now time.Time) (int, error) {
	return s.updateUUIDBatch(ctx, batch, `
		SELECT id FROM instagram_webhook_work
		WHERE status IN ('queued','processing','retryable')
		  AND COALESCE(processing_started_at,created_at) <= $1::timestamptz - interval '15 minutes'
		ORDER BY COALESCE(processing_started_at,created_at),id
		FOR UPDATE SKIP LOCKED LIMIT $2
	`, `
		UPDATE instagram_webhook_work
		SET status='failed',sender_igsid=NULL,official_account_id=NULL,
		    challenge_digest_version=NULL,challenge_digest=NULL,
		    lease_token=NULL,lease_expires_at=NULL,
		    terminal_at=COALESCE(terminal_at,$2),terminal_reason='maxAge',updated_at=$2
		WHERE id=ANY($1::uuid[])
	`, now, "terminalize stale Instagram webhook work")
}

func (s *RetentionService) clearTerminalWebhookIdentity(ctx context.Context, batch int, now time.Time) (int, error) {
	return s.updateUUIDBatch(ctx, batch, `
		SELECT id FROM instagram_webhook_work
		WHERE status IN ('completed','ignored','failed')
		  AND $1::timestamptz IS NOT NULL
		  AND num_nonnulls(sender_igsid,official_account_id,challenge_digest) > 0
		ORDER BY COALESCE(terminal_at,updated_at),id
		FOR UPDATE SKIP LOCKED LIMIT $2
	`, `
		UPDATE instagram_webhook_work
		SET sender_igsid=NULL,official_account_id=NULL,
		    challenge_digest_version=NULL,challenge_digest=NULL,updated_at=$2
		WHERE id=ANY($1::uuid[])
	`, now, "clear terminal Instagram webhook identity")
}

func (s *RetentionService) purgeTerminalWebhookWork(ctx context.Context, batch int, now time.Time) (int, error) {
	return s.deleteUUIDBatch(ctx, batch, `
		SELECT id FROM instagram_webhook_work
		WHERE status IN ('completed','ignored','failed')
		  AND COALESCE(terminal_at,updated_at) <= $1::timestamptz - interval '7 days'
		ORDER BY COALESCE(terminal_at,updated_at),id
		FOR UPDATE SKIP LOCKED LIMIT $2
	`, "instagram_webhook_work", now, "purge terminal Instagram webhook work")
}

func (s *RetentionService) expireInactiveLinks(ctx context.Context, batch int, now time.Time) (int, error) {
	count := 0
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id,owner_did FROM instagram_account_links
			WHERE state='membershipInactive'
			  AND membership_inactive_at <= $1::timestamptz - interval '1 year'
			ORDER BY membership_inactive_at,id
			FOR UPDATE SKIP LOCKED LIMIT $2
		`, now, batch)
		if err != nil {
			return fmt.Errorf("select membership-expired Instagram links: %w", err)
		}
		var ids []uuid.UUID
		var owners []string
		for rows.Next() {
			var id uuid.UUID
			var owner string
			if err := rows.Scan(&id, &owner); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
			owners = append(owners, owner)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		if len(ids) == 0 {
			return nil
		}
		result, err := tx.Exec(ctx, `
			UPDATE instagram_account_links
			SET state='revoked',igsid=NULL,username=NULL,username_normalized=NULL,
			    discoverable=false,conflict_pending=false,membership_inactive_at=NULL,
			    revoked_at=$2,raw_identity_purge_at=$2,updated_at=$2
			WHERE id=ANY($1::uuid[]) AND state='membershipInactive'
		`, ids, now)
		if err != nil {
			return fmt.Errorf("expire inactive Instagram links: %w", err)
		}
		count = int(result.RowsAffected())
		if _, err := tx.Exec(ctx, `
			UPDATE instagram_identity_claims
			SET state='revoked',released_at=COALESCE(released_at,$2),
			    anonymize_at=LEAST(COALESCE(anonymize_at,$2::timestamptz+interval '90 days'),$2::timestamptz+interval '90 days'),updated_at=$2
			WHERE link_id=ANY($1::uuid[]) AND state IN ('active','disputed')
		`, ids, now); err != nil {
			return fmt.Errorf("release membership-expired Instagram claims: %w", err)
		}
		suggestionIDs, err := retentionUUIDs(ctx, tx, `
			UPDATE instagram_follow_suggestions
			SET state='invalidated',accepting_since=NULL,
			    terminal_at=COALESCE(terminal_at,$2),updated_at=$2
			WHERE target_did=ANY($1::text[]) AND state IN ('pending','accepting')
			RETURNING id
		`, owners, now)
		if err != nil {
			return fmt.Errorf("invalidate membership-expired link suggestions: %w", err)
		}
		if err := failUnsentFollowOperations(ctx, tx, suggestionIDs, "linkExpired", now); err != nil {
			return err
		}
		if err := retractSuggestionNotifications(ctx, tx, suggestionIDs, "", "link_expired", now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE instagram_reconciliation_jobs
			SET status='ignored',lease_token=NULL,lease_expires_at=NULL,
			    terminal_at=COALESCE(terminal_at,$2),updated_at=$2
			WHERE link_id=ANY($1::uuid[]) AND status IN ('queued','processing','retryable')
		`, ids, now); err != nil {
			return fmt.Errorf("cancel membership-expired link jobs: %w", err)
		}
		for i, id := range ids {
			if err := insertRetentionAudit(ctx, tx, owners[i], "membership_inactive_link_expired", "link", id.String(), now); err != nil {
				return err
			}
		}
		return nil
	})
	return count, err
}

func (s *RetentionService) clearTerminalLinkIdentity(ctx context.Context, batch int, now time.Time) (int, error) {
	return s.updateUUIDBatch(ctx, batch, `
		SELECT id FROM instagram_account_links
		WHERE state IN ('revoked','superseded')
		  AND num_nonnulls(igsid,username,username_normalized) > 0
		  AND COALESCE(raw_identity_purge_at,revoked_at,superseded_at,updated_at) <= $1
		ORDER BY COALESCE(raw_identity_purge_at,revoked_at,superseded_at,updated_at),id
		FOR UPDATE SKIP LOCKED LIMIT $2
	`, `
		UPDATE instagram_account_links
		SET igsid=NULL,username=NULL,username_normalized=NULL,discoverable=false,updated_at=$2
		WHERE id=ANY($1::uuid[])
	`, now, "clear terminal Instagram link identity")
}

func (s *RetentionService) purgeReleasedClaims(ctx context.Context, batch int, now time.Time) (int, error) {
	return s.deleteUUIDBatch(ctx, batch, `
		SELECT id FROM instagram_identity_claims
		WHERE state='revoked' AND anonymize_at <= $1
		ORDER BY anonymize_at,id
		FOR UPDATE SKIP LOCKED LIMIT $2
	`, "instagram_identity_claims", now, "purge released Instagram claims")
}

func (s *RetentionService) purgeLinkTombstones(ctx context.Context, batch int, now time.Time) (int, error) {
	return s.deleteUUIDBatch(ctx, batch, `
		SELECT id FROM instagram_account_links
		WHERE state IN ('revoked','superseded')
		  AND COALESCE(revoked_at,superseded_at,updated_at) <= $1::timestamptz - interval '90 days'
		ORDER BY COALESCE(revoked_at,superseded_at,updated_at),id
		FOR UPDATE SKIP LOCKED LIMIT $2
	`, "instagram_account_links", now, "purge Instagram link tombstones")
}

func (s *RetentionService) expireConflicts(ctx context.Context, batch int, now time.Time) (int, error) {
	count := 0
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		ids, err := retentionUUIDs(ctx, tx, `
			SELECT id FROM instagram_link_conflicts
			WHERE state='open' AND expires_at <= $1
			ORDER BY expires_at,id
			FOR UPDATE SKIP LOCKED LIMIT $2
		`, now, batch)
		if err != nil || len(ids) == 0 {
			return err
		}
		result, err := tx.Exec(ctx, `
			UPDATE instagram_link_conflicts
			SET state='expired',existing_link_id=NULL,claimant_attempt_id=NULL,
			    claimant_link_id=NULL,igsid_digest_version=NULL,igsid_digest=NULL,
			    resolution_note_digest=NULL,resolved_at=$2,updated_at=$2
			WHERE id=ANY($1::uuid[]) AND state='open'
		`, ids, now)
		if err != nil {
			return fmt.Errorf("expire Instagram link conflicts: %w", err)
		}
		count = int(result.RowsAffected())
		if err := refreshConflictFlags(ctx, tx, now); err != nil {
			return err
		}
		for _, id := range ids {
			if err := insertRetentionAudit(ctx, tx, "", "link_conflict_expired", "conflict", id.String(), now); err != nil {
				return err
			}
		}
		return nil
	})
	return count, err
}

func (s *RetentionService) purgeResolvedConflicts(ctx context.Context, batch int, now time.Time) (int, error) {
	return s.deleteUUIDBatch(ctx, batch, `
		SELECT id FROM instagram_link_conflicts
		WHERE state IN ('resolvedKeepExisting','resolvedRevokeExisting','expired')
		  AND COALESCE(resolved_at,updated_at) <= $1::timestamptz - interval '365 days'
		ORDER BY COALESCE(resolved_at,updated_at),id
		FOR UPDATE SKIP LOCKED LIMIT $2
	`, "instagram_link_conflicts", now, "purge resolved Instagram conflicts")
}

func (s *RetentionService) prepareNonconsentedImports(ctx context.Context, batch int, now time.Time) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		// Snapshot the true final terminal time before support rows are removed.
		if _, err := tx.Exec(ctx, `
			WITH candidates AS (
				SELECT source.id,
				       COALESCE(MAX(suggestion.terminal_at),source.updated_at) AS final_at
				FROM instagram_graph_imports source
				LEFT JOIN instagram_suggestion_sources support ON support.import_id=source.id
				LEFT JOIN instagram_follow_suggestions suggestion ON suggestion.id=support.suggestion_id
				WHERE NOT source.retain_unmatched
				  AND source.final_terminal_at IS NULL
				GROUP BY source.id
				HAVING bool_and(suggestion.id IS NULL OR suggestion.state IN ('accepted','alreadyFollowing','dismissed','invalidated'))
				ORDER BY source.created_at,source.id
				LIMIT $1
			)
			UPDATE instagram_graph_imports source
			SET final_terminal_at=candidate.final_at,
			    aggregate_purge_at=LEAST(candidate.final_at+interval '90 days',source.created_at+interval '1 year'),
			    updated_at=GREATEST(source.updated_at,candidate.final_at)
			FROM candidates candidate WHERE source.id=candidate.id
		`, batch); err != nil {
			return fmt.Errorf("schedule non-consented Instagram aggregate purge: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			WITH candidates AS (
				SELECT support.suggestion_id,support.import_id
				FROM instagram_suggestion_sources support
				JOIN instagram_graph_imports source ON source.id=support.import_id AND NOT source.retain_unmatched
				JOIN instagram_follow_suggestions suggestion ON suggestion.id=support.suggestion_id AND suggestion.state IN ('accepted','alreadyFollowing','dismissed','invalidated')
				ORDER BY support.created_at,support.import_id,support.suggestion_id
				LIMIT $1
			)
			DELETE FROM instagram_suggestion_sources support
			USING candidates candidate
			WHERE support.suggestion_id=candidate.suggestion_id AND support.import_id=candidate.import_id
		`, batch); err != nil {
			return fmt.Errorf("remove terminal non-consented Instagram support: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			WITH candidates AS (
				SELECT handle.id
				FROM instagram_graph_handles handle
				JOIN instagram_graph_imports source ON source.id=handle.import_id
				WHERE handle.retain_until <= $2
				   OR (
					NOT source.retain_unmatched
					AND NOT EXISTS (
						SELECT 1
						FROM instagram_suggestion_sources support
						JOIN instagram_follow_suggestions suggestion
						  ON suggestion.id=support.suggestion_id
						 AND suggestion.state IN ('pending','accepting')
						JOIN instagram_account_links link
						  ON link.owner_did=suggestion.target_did
						 AND link.username_normalized=handle.username_normalized
						WHERE support.import_id=source.id
					)
				   )
				ORDER BY COALESCE(handle.retain_until,handle.created_at),handle.id
				LIMIT $1
			)
			DELETE FROM instagram_graph_handles handle
			USING candidates candidate WHERE handle.id=candidate.id
		`, batch, now); err != nil {
			return fmt.Errorf("purge expired Instagram graph handles: %w", err)
		}
		return nil
	})
}

func (s *RetentionService) purgeExpiredImportsAt(ctx context.Context, batch int, now time.Time) (int, error) {
	count := 0
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id,owner_did
			FROM instagram_graph_imports
			WHERE (retain_unmatched AND retention_expires_at <= $1)
			   OR aggregate_purge_at <= $1
			   OR created_at <= $1::timestamptz - interval '1 year'
			   OR (state='expired' AND COALESCE(aggregate_purge_at,retention_expires_at,created_at+interval '1 year') <= $1)
			ORDER BY LEAST(
				COALESCE(retention_expires_at,'infinity'::timestamptz),
				COALESCE(aggregate_purge_at,'infinity'::timestamptz),
				created_at+interval '1 year'
			),id
			FOR UPDATE SKIP LOCKED LIMIT $2
		`, now, batch)
		if err != nil {
			return fmt.Errorf("select expired Instagram imports: %w", err)
		}
		var ids []uuid.UUID
		var owners []string
		for rows.Next() {
			var id uuid.UUID
			var owner string
			if err := rows.Scan(&id, &owner); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
			owners = append(owners, owner)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return err
		}
		rows.Close()
		if len(ids) == 0 {
			return nil
		}
		impacted, err := retentionUUIDs(ctx, tx, `
			SELECT DISTINCT suggestion_id
			FROM instagram_suggestion_sources
			WHERE import_id=ANY($1::uuid[])
			ORDER BY suggestion_id
		`, ids)
		if err != nil {
			return fmt.Errorf("read expired-import suggestions: %w", err)
		}
		result, err := tx.Exec(ctx, `DELETE FROM instagram_graph_imports WHERE id=ANY($1::uuid[])`, ids)
		if err != nil {
			return fmt.Errorf("purge expired Instagram imports: %w", err)
		}
		count = int(result.RowsAffected())
		invalidated := make([]uuid.UUID, 0)
		if len(impacted) > 0 {
			invalidated, err = retentionUUIDs(ctx, tx, `
				UPDATE instagram_follow_suggestions suggestion
				SET state='invalidated',accepting_since=NULL,
				    terminal_at=COALESCE(terminal_at,$2),updated_at=$2
				WHERE suggestion.id=ANY($1::uuid[])
				  AND suggestion.state IN ('pending','accepting')
				  AND NOT EXISTS (
					SELECT 1 FROM instagram_suggestion_sources support
					JOIN instagram_graph_imports source ON source.id=support.import_id AND source.state='active'
					WHERE support.suggestion_id=suggestion.id
				  )
				RETURNING suggestion.id
			`, impacted, now)
			if err != nil {
				return fmt.Errorf("invalidate expired-import suggestions: %w", err)
			}
		}
		if err := failUnsentFollowOperations(ctx, tx, invalidated, "importExpired", now); err != nil {
			return err
		}
		if err := retractSuggestionNotifications(ctx, tx, invalidated, "", "import_expired", now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE instagram_reconciliation_jobs
			SET status='ignored',lease_token=NULL,lease_expires_at=NULL,
			    terminal_at=COALESCE(terminal_at,$2),updated_at=$2
			WHERE import_id=ANY($1::uuid[]) AND status IN ('queued','processing','retryable')
		`, ids, now); err != nil {
			return fmt.Errorf("cancel expired-import jobs: %w", err)
		}
		for i, id := range ids {
			if err := insertRetentionAudit(ctx, tx, owners[i], "graph_import_expired", "import", id.String(), now); err != nil {
				return err
			}
		}
		return nil
	})
	return count, err
}

func (s *RetentionService) purgeTerminalSuggestions(ctx context.Context, batch int, now time.Time) (int, error) {
	count := 0
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		ids, err := retentionUUIDs(ctx, tx, `
			SELECT id FROM instagram_follow_suggestions
			WHERE (
				state IN ('dismissed','invalidated')
				AND COALESCE(terminal_at,updated_at) <= $1::timestamptz - interval '90 days'
			) OR (
				state IN ('accepted','alreadyFollowing')
				AND COALESCE(terminal_at,updated_at) <= $1::timestamptz - interval '1 year'
			)
			ORDER BY COALESCE(terminal_at,updated_at),id
			FOR UPDATE SKIP LOCKED LIMIT $2
		`, now, batch)
		if err != nil || len(ids) == 0 {
			return err
		}
		if err := retractSuggestionNotifications(ctx, tx, ids, "", "suggestion_expired", now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM pds_follow_operations WHERE suggestion_id=ANY($1::uuid[])`, ids); err != nil {
			return fmt.Errorf("purge retained Instagram follow ledgers: %w", err)
		}
		result, err := tx.Exec(ctx, `DELETE FROM instagram_follow_suggestions WHERE id=ANY($1::uuid[])`, ids)
		if err != nil {
			return fmt.Errorf("purge terminal Instagram suggestions: %w", err)
		}
		count = int(result.RowsAffected())
		return nil
	})
	return count, err
}

func (s *RetentionService) purgeRetractedDeliveries(ctx context.Context, batch int, now time.Time) (int, error) {
	count := 0
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		ids, err := retentionUUIDs(ctx, tx, `
			SELECT delivery.id
			FROM push_deliveries delivery
			JOIN notification_events event ON event.id=delivery.notification_id
			WHERE event.kind='system' AND event.category='instagramMatch' AND event.state='retracted'
			  AND delivery.status='cancelled'
			  AND delivery.updated_at <= $1::timestamptz - interval '7 days'
			ORDER BY delivery.updated_at,delivery.id
			FOR UPDATE OF delivery SKIP LOCKED LIMIT $2
		`, now, batch)
		if err != nil || len(ids) == 0 {
			return err
		}
		result, err := tx.Exec(ctx, `DELETE FROM push_deliveries WHERE id=ANY($1::uuid[])`, ids)
		if err != nil {
			return fmt.Errorf("purge retracted Instagram deliveries: %w", err)
		}
		count = int(result.RowsAffected())
		return nil
	})
	return count, err
}

func (s *RetentionService) purgeMatchNotifications(ctx context.Context, batch int, now time.Time) (int, error) {
	return s.deleteUUIDBatch(ctx, batch, `
		SELECT id FROM notification_events
		WHERE kind='system' AND category='instagramMatch'
		  AND activity_at <= $1::timestamptz - interval '90 days'
		ORDER BY activity_at,id
		FOR UPDATE SKIP LOCKED LIMIT $2
	`, "notification_events", now, "purge Instagram match notifications")
}

func (s *RetentionService) purgeRateBuckets(ctx context.Context, batch int, now time.Time) (int, error) {
	count := 0
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, `
			WITH candidates AS (
				SELECT bucket_scope,key_version,key_digest,window_start
				FROM instagram_rate_limit_buckets
				WHERE window_end <= $1::timestamptz - interval '24 hours'
				ORDER BY window_end,bucket_scope,key_version,key_digest,window_start
				FOR UPDATE SKIP LOCKED LIMIT $2
			)
			DELETE FROM instagram_rate_limit_buckets bucket
			USING candidates candidate
			WHERE bucket.bucket_scope=candidate.bucket_scope
			  AND bucket.key_version=candidate.key_version
			  AND bucket.key_digest=candidate.key_digest
			  AND bucket.window_start=candidate.window_start
		`, now, batch)
		if err != nil {
			return fmt.Errorf("purge Instagram rate buckets: %w", err)
		}
		count = int(result.RowsAffected())
		return nil
	})
	return count, err
}

func (s *RetentionService) purgeAuditEvents(ctx context.Context, batch int, now time.Time) (int, error) {
	count := 0
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		result, err := tx.Exec(ctx, `
			WITH candidates AS (
				SELECT id FROM instagram_audit_events
				WHERE created_at <= $1::timestamptz - interval '365 days'
				ORDER BY created_at,id
				FOR UPDATE SKIP LOCKED LIMIT $2
			)
			DELETE FROM instagram_audit_events audit
			USING candidates candidate WHERE audit.id=candidate.id
		`, now, batch)
		if err != nil {
			return fmt.Errorf("purge Instagram audit events: %w", err)
		}
		count = int(result.RowsAffected())
		return nil
	})
	return count, err
}

func (s *RetentionService) updateUUIDBatch(ctx context.Context, batch int, selectSQL, updateSQL string, now time.Time, operation string) (int, error) {
	count := 0
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		ids, err := retentionUUIDs(ctx, tx, selectSQL, now, batch)
		if err != nil || len(ids) == 0 {
			return err
		}
		result, err := tx.Exec(ctx, updateSQL, ids, now)
		if err != nil {
			return fmt.Errorf("%s: %w", operation, err)
		}
		count = int(result.RowsAffected())
		return nil
	})
	return count, err
}

func (s *RetentionService) deleteUUIDBatch(ctx context.Context, batch int, selectSQL, table string, now time.Time, operation string) (int, error) {
	count := 0
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		ids, err := retentionUUIDs(ctx, tx, selectSQL, now, batch)
		if err != nil || len(ids) == 0 {
			return err
		}
		// Table names are constants owned by the caller methods above, never
		// operator or request input.
		result, err := tx.Exec(ctx, "DELETE FROM "+table+" WHERE id=ANY($1::uuid[])", ids)
		if err != nil {
			return fmt.Errorf("%s: %w", operation, err)
		}
		count = int(result.RowsAffected())
		return nil
	})
	return count, err
}

func retentionUUIDs(ctx context.Context, tx pgx.Tx, query string, args ...any) ([]uuid.UUID, error) {
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func insertRetentionAudit(ctx context.Context, tx pgx.Tx, owner, action, subjectKind, subjectID string, now time.Time) error {
	var ownerValue any
	if owner != "" {
		ownerValue = owner
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO instagram_audit_events(owner_did,action,subject_kind,subject_id,outcome,created_at)
		VALUES($1,$2,$3,$4,'completed',$5)
	`, ownerValue, action, subjectKind, subjectID, now); err != nil {
		return fmt.Errorf("write Instagram retention audit: %w", err)
	}
	return nil
}
