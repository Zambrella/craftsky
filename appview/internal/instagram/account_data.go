package instagram

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PrivateDataService owns the reversible membership boundary and the explicit
// deletion/export boundary for private Instagram data. It deliberately has no
// PDS client: lifecycle cleanup must never delete an accepted public follow.
type PrivateDataService struct {
	pool        *pgxpool.Pool
	rateLimiter *PostgresRateLimiter
	now         func() time.Time
}

func NewPrivateDataService(pool *pgxpool.Pool, rateLimiter *PostgresRateLimiter, now func() time.Time) *PrivateDataService {
	if now == nil {
		now = time.Now
	}
	return &PrivateDataService{pool: pool, rateLimiter: rateLimiter, now: now}
}

// InactivateMembership applies the reversible side of profile membership
// loss. Calling it repeatedly, including concurrently, preserves the original
// inactivity timestamp and never re-enables state after a member rejoins.
func (s *PrivateDataService) InactivateMembership(ctx context.Context, owner syntax.DID) error {
	if err := s.validateOwner(owner); err != nil {
		return err
	}
	now := s.now().UTC()
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		return s.InactivateMembershipTx(ctx, tx, owner, now)
	})
}

// InactivateMembershipTx lets the craftsky profile indexer place private
// inactivation in the same transaction as removal of current membership.
func (s *PrivateDataService) InactivateMembershipTx(ctx context.Context, tx pgx.Tx, owner syntax.DID, now time.Time) error {
	if s == nil || s.pool == nil || tx == nil {
		return errors.New("Instagram private-data service is unavailable")
	}
	if owner == "" || now.IsZero() {
		return errors.New("Instagram membership inactivation requires an owner and time")
	}
	now = now.UTC()
	if err := lockInstagramOwner(ctx, tx, owner); err != nil {
		return fmt.Errorf("lock Instagram membership lifecycle: %w", err)
	}

	// Terminalize work before clearing attempt challenge/candidate fields. This
	// catches both an already-bound attempt and a still-queued digest match.
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_webhook_work work
		SET status='ignored', sender_igsid=NULL, official_account_id=NULL,
		    challenge_digest_version=NULL, challenge_digest=NULL,
		    lease_token=NULL, lease_expires_at=NULL,
		    terminal_at=$2, terminal_reason='membershipInactive', updated_at=$2
		WHERE work.status IN ('queued','processing','retryable')
		  AND EXISTS (
			SELECT 1
			FROM instagram_verification_attempts attempt
			WHERE attempt.owner_did=$1
			  AND (
				work.verification_attempt_id=attempt.id
				OR (
					work.verification_attempt_id IS NULL
					AND work.challenge_digest_version=attempt.challenge_digest_version
					AND work.challenge_digest=attempt.challenge_digest
				)
			  )
		  )
	`, owner, now); err != nil {
		return fmt.Errorf("cancel Instagram webhook work for inactive member: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_verification_attempts
		SET state=CASE WHEN state='pendingDm' THEN 'cancelled' ELSE 'rejected' END,
		    challenge_digest_version=NULL,
		    challenge_digest=NULL, candidate_igsid=NULL,
		    candidate_username=NULL,
		    retry_code=CASE WHEN state='pendingDm' THEN NULL ELSE 'membershipInactive' END,
		    terminal_at=$2, updated_at=$2
		WHERE owner_did=$1
		  AND state IN ('pendingDm','processing','pendingConfirmation')
	`, owner, now); err != nil {
		return fmt.Errorf("reject Instagram attempts for inactive member: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_account_links
		SET state='membershipInactive', discoverable=false,
		    membership_inactive_at=COALESCE(membership_inactive_at,$2),
		    updated_at=$2
		WHERE owner_did=$1 AND state IN ('active','disputed')
	`, owner, now); err != nil {
		return fmt.Errorf("inactivate Instagram link: %w", err)
	}

	// Expiry remains absolute across membership loss. At the exact boundary an
	// import expires; only still-unexpired imports become reversibly inactive.
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_graph_imports
		SET state='expired', final_terminal_at=COALESCE(final_terminal_at,$2),
		    aggregate_purge_at=LEAST(
			COALESCE(aggregate_purge_at,$2::timestamptz + interval '90 days'),
			created_at + interval '1 year'
		    ), updated_at=$2
		WHERE owner_did=$1
		  AND state IN ('active','membershipInactive')
		  AND (
			(retention_expires_at IS NOT NULL AND retention_expires_at <= $2)
			OR created_at + interval '1 year' <= $2
		  )
	`, owner, now); err != nil {
		return fmt.Errorf("expire Instagram imports during inactivation: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM instagram_graph_handles handle
		USING instagram_graph_imports source
		WHERE handle.import_id=source.id
		  AND source.owner_did=$1 AND source.state='expired'
	`, owner); err != nil {
		return fmt.Errorf("clear expired Instagram handles during inactivation: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_graph_imports
		SET state='membershipInactive',
		    membership_inactive_at=COALESCE(membership_inactive_at,$2),
		    updated_at=$2
		WHERE owner_did=$1 AND state='active'
	`, owner, now); err != nil {
		return fmt.Errorf("pause Instagram imports: %w", err)
	}

	suggestionIDs, err := updateSuggestionState(ctx, tx, `
		UPDATE instagram_follow_suggestions
		SET state='invalidated', accepting_since=NULL,
		    terminal_at=COALESCE(terminal_at,$2), updated_at=$2
		WHERE (importer_did=$1 OR target_did=$1)
		  AND state IN ('pending','accepting')
		RETURNING id
	`, owner, now)
	if err != nil {
		return fmt.Errorf("invalidate Instagram suggestions for inactive member: %w", err)
	}
	if err := failUnsentFollowOperations(ctx, tx, suggestionIDs, "membershipInactive", now); err != nil {
		return err
	}
	if err := retractSuggestionNotifications(ctx, tx, suggestionIDs, owner, "membership_inactive", now); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_reconciliation_jobs
		SET status='ignored', lease_token=NULL, lease_expires_at=NULL,
		    terminal_at=COALESCE(terminal_at,$2), updated_at=$2
		WHERE (owner_did=$1 OR target_did=$1)
		  AND status IN ('queued','processing','retryable')
	`, owner, now); err != nil {
		return fmt.Errorf("pause Instagram reconciliation jobs: %w", err)
	}
	return nil
}

// PurgeLink permanently removes one caller-owned link aggregate. Foreign,
// absent, and already-purged IDs are indistinguishable successful no-ops.
func (s *PrivateDataService) PurgeLink(ctx context.Context, owner syntax.DID, linkID uuid.UUID) error {
	if err := s.validateOwner(owner); err != nil {
		return err
	}
	if linkID == uuid.Nil {
		return nil
	}
	now := s.now().UTC()
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := lockInstagramOwner(ctx, tx, owner); err != nil {
			return err
		}
		var state InstagramLinkState
		var igsid sql.NullString
		err := tx.QueryRow(ctx, `
			SELECT state, igsid
			FROM instagram_account_links
			WHERE id=$1 AND owner_did=$2
			FOR UPDATE
		`, linkID, owner).Scan(&state, &igsid)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read scoped Instagram link: %w", err)
		}
		if !state.Valid() {
			return ErrInvalidInstagramState
		}

		if !state.Terminal() {
			suggestionIDs, err := updateSuggestionState(ctx, tx, `
				UPDATE instagram_follow_suggestions
				SET state='invalidated', accepting_since=NULL,
				    terminal_at=COALESCE(terminal_at,$2), updated_at=$2
				WHERE target_did=$1 AND state IN ('pending','accepting')
				RETURNING id
			`, owner, now)
			if err != nil {
				return fmt.Errorf("invalidate scoped-link suggestions: %w", err)
			}
			if err := failUnsentFollowOperations(ctx, tx, suggestionIDs, "linkDeleted", now); err != nil {
				return err
			}
			if err := retractSuggestionNotifications(ctx, tx, suggestionIDs, "", "link_revoked", now); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `DELETE FROM instagram_reconciliation_jobs WHERE link_id=$1`, linkID); err != nil {
			return fmt.Errorf("delete scoped-link jobs: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM instagram_link_conflicts
			WHERE existing_link_id=$1 OR claimant_link_id=$1
		`, linkID); err != nil {
			return fmt.Errorf("delete scoped-link conflicts: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM instagram_account_links WHERE id=$1 AND owner_did=$2`, linkID, owner); err != nil {
			return fmt.Errorf("delete scoped Instagram link: %w", err)
		}
		if err := refreshConflictFlags(ctx, tx, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE instagram_audit_events
			SET subject_id=NULL
			WHERE subject_id=$1
		`, linkID.String()); err != nil {
			return fmt.Errorf("anonymize scoped-link audit: %w", err)
		}
		if igsid.Valid {
			if err := s.purgeIGSIDRateBuckets(ctx, tx, igsid.String); err != nil {
				return err
			}
		}
		return nil
	})
}

// PurgeImport permanently removes one owner-scoped source and invalidates only
// pending suggestions that no other active source supports.
func (s *PrivateDataService) PurgeImport(ctx context.Context, owner syntax.DID, importID uuid.UUID) error {
	if err := s.validateOwner(owner); err != nil {
		return err
	}
	if importID == uuid.Nil {
		return nil
	}
	now := s.now().UTC()
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := lockInstagramOwner(ctx, tx, owner); err != nil {
			return err
		}
		var exists bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM instagram_graph_imports
				WHERE id=$1 AND owner_did=$2
			)
		`, importID, owner).Scan(&exists); err != nil {
			return fmt.Errorf("read scoped Instagram import: %w", err)
		}
		if !exists {
			return nil
		}
		suggestionIDs, err := queryUUIDs(ctx, tx, `
			SELECT suggestion.id
			FROM instagram_suggestion_sources selected
			JOIN instagram_follow_suggestions suggestion
			  ON suggestion.id=selected.suggestion_id
			WHERE selected.import_id=$1
			  AND suggestion.importer_did=$2
			  AND suggestion.state IN ('pending','accepting')
			  AND NOT EXISTS (
				SELECT 1
				FROM instagram_suggestion_sources other
				JOIN instagram_graph_imports source
				  ON source.id=other.import_id AND source.state='active'
				WHERE other.suggestion_id=suggestion.id
				  AND other.import_id<>$1
			  )
			FOR UPDATE OF suggestion
		`, importID, owner)
		if err != nil {
			return fmt.Errorf("find scoped-import dependents: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM instagram_graph_imports WHERE id=$1 AND owner_did=$2`, importID, owner); err != nil {
			return fmt.Errorf("delete scoped Instagram import: %w", err)
		}
		if err := invalidateSuggestionIDs(ctx, tx, suggestionIDs, now); err != nil {
			return err
		}
		if err := failUnsentFollowOperations(ctx, tx, suggestionIDs, "importDeleted", now); err != nil {
			return err
		}
		if err := retractSuggestionNotifications(ctx, tx, suggestionIDs, "", "import_deleted", now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM instagram_reconciliation_jobs WHERE import_id=$1`, importID); err != nil {
			return fmt.Errorf("delete scoped-import jobs: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE instagram_audit_events SET subject_id=NULL WHERE subject_id=$1
		`, importID.String()); err != nil {
			return fmt.Errorf("anonymize scoped-import audit: %w", err)
		}
		return nil
	})
}

// HandleIdentityDeleted satisfies Tap's terminal identity-deletion seam.
func (s *PrivateDataService) HandleIdentityDeleted(ctx context.Context, owner syntax.DID) error {
	return s.PurgeOwner(ctx, owner)
}

// PurgeOwner permanently removes all private Instagram facts identifying the
// owner. It deletes only AppView state and never performs a PDS operation.
func (s *PrivateDataService) PurgeOwner(ctx context.Context, owner syntax.DID) error {
	if err := s.validateOwner(owner); err != nil {
		return err
	}
	now := s.now().UTC()
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if err := lockInstagramOwner(ctx, tx, owner); err != nil {
			return fmt.Errorf("lock terminal Instagram purge: %w", err)
		}

		igsids, err := queryStrings(ctx, tx, `
			SELECT igsid FROM instagram_account_links
			WHERE owner_did=$1 AND igsid IS NOT NULL
			UNION
			SELECT candidate_igsid FROM instagram_verification_attempts
			WHERE owner_did=$1 AND candidate_igsid IS NOT NULL
		`, owner)
		if err != nil {
			return fmt.Errorf("read terminal Instagram rate identities: %w", err)
		}
		suggestionIDs, err := queryUUIDs(ctx, tx, `
			SELECT id FROM instagram_follow_suggestions
			WHERE importer_did=$1 OR target_did=$1
			FOR UPDATE
		`, owner)
		if err != nil {
			return fmt.Errorf("read terminal Instagram suggestions: %w", err)
		}
		if err := retractSuggestionNotifications(ctx, tx, suggestionIDs, owner, "membership_inactive", now); err != nil {
			return err
		}

		// Delete the private operation ledger, not the public PDS records it
		// described. The absence of a PDS dependency makes that boundary hard.
		if _, err := tx.Exec(ctx, `
			DELETE FROM pds_follow_operations
			WHERE owner_did=$1 OR target_did=$1
			   OR suggestion_id IN (
				SELECT id FROM instagram_follow_suggestions
				WHERE importer_did=$1 OR target_did=$1
			   )
		`, owner); err != nil {
			return fmt.Errorf("delete terminal Instagram follow ledger: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM instagram_follow_suggestions WHERE importer_did=$1 OR target_did=$1`, owner); err != nil {
			return fmt.Errorf("delete terminal Instagram suggestions: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM instagram_reconciliation_jobs WHERE owner_did=$1 OR target_did=$1`, owner); err != nil {
			return fmt.Errorf("delete terminal Instagram reconciliation: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM instagram_graph_imports WHERE owner_did=$1`, owner); err != nil {
			return fmt.Errorf("delete terminal Instagram imports: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM instagram_webhook_work work
			WHERE EXISTS (
				SELECT 1 FROM instagram_verification_attempts attempt
				WHERE attempt.owner_did=$1
				  AND (
					work.verification_attempt_id=attempt.id
					OR (
						work.verification_attempt_id IS NULL
						AND work.challenge_digest_version=attempt.challenge_digest_version
						AND work.challenge_digest=attempt.challenge_digest
					)
				  )
			)
		`, owner); err != nil {
			return fmt.Errorf("delete terminal Instagram webhook work: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM instagram_link_conflicts conflict
			WHERE EXISTS (
				SELECT 1 FROM instagram_account_links link
				WHERE link.owner_did=$1
				  AND link.id IN (conflict.existing_link_id, conflict.claimant_link_id)
			)
			OR EXISTS (
				SELECT 1 FROM instagram_verification_attempts attempt
				WHERE attempt.owner_did=$1
				  AND attempt.id=conflict.claimant_attempt_id
			)
		`, owner); err != nil {
			return fmt.Errorf("delete terminal Instagram conflicts: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM instagram_identity_claims WHERE owner_did=$1`, owner); err != nil {
			return fmt.Errorf("delete terminal Instagram claims: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM instagram_account_links WHERE owner_did=$1`, owner); err != nil {
			return fmt.Errorf("delete terminal Instagram links: %w", err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM instagram_verification_attempts WHERE owner_did=$1`, owner); err != nil {
			return fmt.Errorf("delete terminal Instagram attempts: %w", err)
		}
		if err := refreshConflictFlags(ctx, tx, now); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM notification_preferences
			WHERE account_did=$1 AND category='instagramMatch'
		`, owner); err != nil {
			return fmt.Errorf("delete terminal Instagram notification preference: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM notification_events
			WHERE recipient_did=$1 AND kind='system' AND category='instagramMatch'
		`, owner); err != nil {
			return fmt.Errorf("delete terminal Instagram notifications: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE instagram_audit_events
			SET owner_did=NULL, subject_id=NULL
			WHERE owner_did=$1
		`, owner); err != nil {
			return fmt.Errorf("anonymize terminal Instagram audit: %w", err)
		}
		if err := s.purgeOwnerRateBuckets(ctx, tx, owner, igsids); err != nil {
			return err
		}
		return nil
	})
}

func (s *PrivateDataService) validateOwner(owner syntax.DID) error {
	if s == nil || s.pool == nil {
		return errors.New("Instagram private-data service is unavailable")
	}
	if owner == "" {
		return errors.New("Instagram private-data owner is required")
	}
	return nil
}

func (PrivateDataService) String() string {
	return "Instagram PrivateDataService{database:configured,rateLimiter:[REDACTED],clock:configured}"
}

func (s PrivateDataService) GoString() string { return s.String() }

func (s PrivateDataService) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, s.String())
}

func lockInstagramOwner(ctx context.Context, tx pgx.Tx, owner syntax.DID) error {
	// Match the link/attempt and import stores' existing lock domains. Always
	// acquire them in this order so lifecycle operations cannot deadlock.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, owner); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 2))`, owner); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 3))`, owner); err != nil {
		return err
	}
	return nil
}

func updateSuggestionState(ctx context.Context, tx pgx.Tx, query string, args ...any) ([]uuid.UUID, error) {
	return queryUUIDs(ctx, tx, query, args...)
}

func invalidateSuggestionIDs(ctx context.Context, tx pgx.Tx, ids []uuid.UUID, now time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_follow_suggestions
		SET state='invalidated', accepting_since=NULL,
		    terminal_at=COALESCE(terminal_at,$2), updated_at=$2
		WHERE id=ANY($1::uuid[]) AND state IN ('pending','accepting')
	`, ids, now); err != nil {
		return fmt.Errorf("invalidate Instagram suggestion IDs: %w", err)
	}
	return nil
}

func failUnsentFollowOperations(ctx context.Context, tx pgx.Tx, ids []uuid.UUID, code string, now time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	if _, err := tx.Exec(ctx, `
		UPDATE pds_follow_operations
		SET status='failed', last_error_code=$2, updated_at=$3
		WHERE suggestion_id=ANY($1::uuid[])
		  AND status IN ('pending','writing','failed')
	`, ids, code, now); err != nil {
		return fmt.Errorf("cancel unsent Instagram follow operations: %w", err)
	}
	return nil
}

func retractSuggestionNotifications(ctx context.Context, tx pgx.Tx, suggestionIDs []uuid.UUID, recipient syntax.DID, reason string, now time.Time) error {
	recipients := make(map[string]struct{})
	if recipient != "" {
		recipients[recipient.String()] = struct{}{}
	}
	if len(suggestionIDs) > 0 {
		rows, err := tx.Query(ctx, `
			SELECT DISTINCT event.recipient_did
			FROM instagram_notification_suggestions support
			JOIN notification_events event ON event.id=support.notification_id
			WHERE support.suggestion_id=ANY($1::uuid[])
		`, suggestionIDs)
		if err != nil {
			return fmt.Errorf("read Instagram notification recipients: %w", err)
		}
		for rows.Next() {
			var did string
			if err := rows.Scan(&did); err != nil {
				rows.Close()
				return fmt.Errorf("scan Instagram notification recipient: %w", err)
			}
			recipients[did] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return fmt.Errorf("iterate Instagram notification recipients: %w", err)
		}
		rows.Close()
	}
	orderedRecipients := make([]string, 0, len(recipients))
	for did := range recipients {
		orderedRecipients = append(orderedRecipients, did)
	}
	sort.Strings(orderedRecipients)
	for _, did := range orderedRecipients {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 7))`, did); err != nil {
			return fmt.Errorf("lock Instagram notification lifecycle: %w", err)
		}
	}

	eventSet := make(map[uuid.UUID]struct{})
	if len(suggestionIDs) > 0 {
		ids, err := queryUUIDs(ctx, tx, `
			SELECT notification_id
			FROM instagram_notification_suggestions
			WHERE suggestion_id=ANY($1::uuid[])
			FOR UPDATE
		`, suggestionIDs)
		if err != nil {
			return fmt.Errorf("read Instagram suggestion notifications: %w", err)
		}
		for _, id := range ids {
			eventSet[id] = struct{}{}
		}
		if _, err := tx.Exec(ctx, `
			DELETE FROM instagram_notification_suggestions
			WHERE suggestion_id=ANY($1::uuid[])
		`, suggestionIDs); err != nil {
			return fmt.Errorf("detach Instagram suggestion notifications: %w", err)
		}
	}
	if recipient != "" {
		ids, err := queryUUIDs(ctx, tx, `
			SELECT id FROM notification_events
			WHERE recipient_did=$1 AND kind='system' AND category='instagramMatch'
			FOR UPDATE
		`, recipient)
		if err != nil {
			return fmt.Errorf("read member Instagram notifications: %w", err)
		}
		for _, id := range ids {
			eventSet[id] = struct{}{}
		}
		if len(ids) > 0 {
			if _, err := tx.Exec(ctx, `
				DELETE FROM instagram_notification_suggestions
				WHERE notification_id=ANY($1::uuid[])
			`, ids); err != nil {
				return fmt.Errorf("detach inactive member notifications: %w", err)
			}
		}
	}

	for eventID := range eventSet {
		var supportCount int
		if err := tx.QueryRow(ctx, `
			SELECT count(*) FROM instagram_notification_suggestions
			WHERE notification_id=$1
		`, eventID).Scan(&supportCount); err != nil {
			return fmt.Errorf("recount Instagram notification support: %w", err)
		}
		if supportCount > 0 {
			if _, err := tx.Exec(ctx, `
				UPDATE notification_events
				SET system_count=LEAST($2,99), system_count_capped=$2>99,
				    indexed_at=$3
				WHERE id=$1 AND kind='system' AND state='active'
			`, eventID, supportCount, now); err != nil {
				return fmt.Errorf("update Instagram notification support: %w", err)
			}
			continue
		}
		if _, err := tx.Exec(ctx, `
			UPDATE notification_events
			SET state='retracted', retracted_at=COALESCE(retracted_at,$2),
			    retraction_reason=COALESCE(retraction_reason,$3), indexed_at=$2
			WHERE id=$1 AND kind='system' AND state='active'
		`, eventID, now, reason); err != nil {
			return fmt.Errorf("retract Instagram notification: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE push_deliveries
			SET status='cancelled', lease_owner=NULL, lease_expires_at=NULL,
			    updated_at=$2
			WHERE notification_id=$1 AND status IN ('pending','retry','leased')
		`, eventID, now); err != nil {
			return fmt.Errorf("cancel Instagram notification delivery: %w", err)
		}
	}
	return nil
}

func refreshConflictFlags(ctx context.Context, tx pgx.Tx, now time.Time) error {
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_account_links link
		SET conflict_pending=EXISTS (
			SELECT 1 FROM instagram_link_conflicts conflict
			WHERE conflict.state='open'
			  AND link.id IN (conflict.existing_link_id, conflict.claimant_link_id)
		), updated_at=$1
		WHERE link.conflict_pending
		  AND NOT EXISTS (
			SELECT 1 FROM instagram_link_conflicts conflict
			WHERE conflict.state='open'
			  AND link.id IN (conflict.existing_link_id, conflict.claimant_link_id)
		  )
	`, now); err != nil {
		return fmt.Errorf("refresh Instagram conflict flags: %w", err)
	}
	return nil
}

func (s *PrivateDataService) purgeOwnerRateBuckets(ctx context.Context, tx pgx.Tx, owner syntax.DID, igsids []string) error {
	if s.rateLimiter == nil {
		return nil
	}
	for _, scope := range []RateLimitScope{
		RateLimitChallengeDID,
		RateLimitConfirmationDID,
		RateLimitImportDID,
	} {
		key, err := s.rateLimiter.Key(scope, []byte(owner))
		if err != nil {
			return err
		}
		if err := deleteRateLimitKey(ctx, tx, key); err != nil {
			return err
		}
	}
	for _, igsid := range igsids {
		if err := s.purgeIGSIDRateBuckets(ctx, tx, igsid); err != nil {
			return err
		}
	}
	return nil
}

func (s *PrivateDataService) purgeIGSIDRateBuckets(ctx context.Context, tx pgx.Tx, igsid string) error {
	if s.rateLimiter == nil || igsid == "" {
		return nil
	}
	for _, scope := range []RateLimitScope{RateLimitInvalidRedemptionIGSID, RateLimitMetaLookupIGSID} {
		key, err := s.rateLimiter.Key(scope, []byte(igsid))
		if err != nil {
			return err
		}
		if err := deleteRateLimitKey(ctx, tx, key); err != nil {
			return err
		}
	}
	return nil
}

func deleteRateLimitKey(ctx context.Context, tx pgx.Tx, key RateLimitKey) error {
	if _, err := tx.Exec(ctx, `
		DELETE FROM instagram_rate_limit_buckets
		WHERE bucket_scope=$1 AND key_version=$2 AND key_digest=$3
	`, key.scope, key.version, key.digest[:]); err != nil {
		return fmt.Errorf("delete Instagram rate-limit bucket: %w", err)
	}
	return nil
}

func queryUUIDs(ctx context.Context, tx pgx.Tx, query string, args ...any) ([]uuid.UUID, error) {
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

func queryStrings(ctx context.Context, tx pgx.Tx, query string, args ...any) ([]string, error) {
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := make([]string, 0)
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}
