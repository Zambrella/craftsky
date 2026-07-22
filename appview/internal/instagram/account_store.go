package instagram

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrInstagramLinkNotFound         = errors.New("Instagram link not found")
	ErrInstagramReactivationRequired = errors.New("Instagram link reactivation required")
	ErrInvalidInstagramSettings      = errors.New("invalid Instagram account settings")
)

type AccountSettingsPatch struct {
	Discoverable *bool
	Reactivate   *bool
}

type AccountStore struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewAccountStore(pool *pgxpool.Pool, now func() time.Time) *AccountStore {
	if now == nil {
		now = time.Now
	}
	return &AccountStore{pool: pool, now: now}
}

func (s *AccountStore) GetAccount(ctx context.Context, owner syntax.DID) (*AccountView, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("Instagram account store is unavailable")
	}
	if owner == "" {
		return nil, errors.New("Instagram account owner is required")
	}
	account, err := scanAccountView(s.pool.QueryRow(ctx, currentAccountViewQuery, owner))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read Instagram account: %w", err)
	}
	return &account, nil
}

func (s *AccountStore) UpdateSettings(ctx context.Context, owner syntax.DID, patch AccountSettingsPatch) (*AccountView, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("Instagram account store is unavailable")
	}
	if owner == "" || validateAccountSettingsPatch(patch) != nil {
		return nil, ErrInvalidInstagramSettings
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, fmt.Errorf("begin Instagram account settings: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, owner); err != nil {
		return nil, fmt.Errorf("lock Instagram account settings: %w", err)
	}

	var current struct {
		ID      uuid.UUID
		Account AccountView
	}
	err = tx.QueryRow(ctx, `
		SELECT id, state, username, discoverable, conflict_pending,
		       state = 'membershipInactive' AS reactivation_required,
		       verified_at
		FROM instagram_account_links
		WHERE owner_did = $1
		  AND state IN ('active', 'membershipInactive', 'disputed')
		ORDER BY updated_at DESC, id DESC
		LIMIT 1
		FOR UPDATE
	`, owner).Scan(
		&current.ID,
		&current.Account.State,
		&current.Account.Username,
		&current.Account.Discoverable,
		&current.Account.ConflictPending,
		&current.Account.ReactivationRequired,
		&current.Account.VerifiedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInstagramLinkNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read Instagram account settings: %w", err)
	}
	if !current.Account.State.Valid() {
		return nil, ErrInvalidInstagramState
	}

	privacyDisable := patch.Discoverable != nil && !*patch.Discoverable && patch.Reactivate == nil
	if current.Account.State == LinkDisputed || current.Account.ConflictPending {
		if !privacyDisable {
			return nil, ErrInstagramLinkConflict
		}
	}

	nextState := current.Account.State
	nextDiscoverable := current.Account.Discoverable
	if patch.Discoverable != nil {
		nextDiscoverable = *patch.Discoverable
	}
	if current.Account.State == LinkMembershipInactive {
		if patch.Reactivate == nil || !*patch.Reactivate {
			return nil, ErrInstagramReactivationRequired
		}
		nextState = LinkActive
	}

	now := s.now().UTC()
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_account_links
		SET state = $2,
		    discoverable = $3,
		    membership_inactive_at = CASE WHEN $2 = 'active' THEN NULL ELSE membership_inactive_at END,
		    updated_at = $4
		WHERE id = $1
	`, current.ID, nextState, nextDiscoverable, now); err != nil {
		return nil, fmt.Errorf("update Instagram account settings: %w", err)
	}

	if !nextDiscoverable {
		suggestionIDs, err := updateSuggestionState(ctx, tx, `
			UPDATE instagram_follow_suggestions
			SET state = 'invalidated', accepting_since = NULL,
			    terminal_at = COALESCE(terminal_at, $2), updated_at = $2
			WHERE target_did = $1 AND state IN ('pending', 'accepting')
			RETURNING id
		`, owner, now)
		if err != nil {
			return nil, fmt.Errorf("invalidate Instagram account suggestions: %w", err)
		}
		if err := failUnsentFollowOperations(ctx, tx, suggestionIDs, "linkDiscoveryDisabled", now); err != nil {
			return nil, err
		}
		if err := retractSuggestionNotifications(ctx, tx, suggestionIDs, "", "eligibility_changed", now); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE instagram_reconciliation_jobs
			SET status='ignored', terminal_at=COALESCE(terminal_at,$3),
			    lease_token=NULL, lease_expires_at=NULL, updated_at=$3
			WHERE (target_did=$1 OR link_id=$2)
			  AND status IN ('queued','processing','retryable')
		`, owner, current.ID, now); err != nil {
			return nil, fmt.Errorf("cancel Instagram account reconciliation: %w", err)
		}
	}
	if nextState == LinkActive && nextDiscoverable &&
		(current.Account.State == LinkMembershipInactive || !current.Account.Discoverable) {
		reason := "instagramLinkDiscoveryEnabled"
		if current.Account.State == LinkMembershipInactive {
			reason = "instagramLinkReactivated"
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO instagram_reconciliation_jobs (
				id, owner_did, link_id, reason, status, next_attempt_at,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, 'queued', $5, $5, $5)
		`, uuid.New(), owner, current.ID, reason, now); err != nil {
			return nil, fmt.Errorf("queue Instagram account reconciliation: %w", err)
		}
	}

	account, err := scanAccountView(tx.QueryRow(ctx, currentAccountViewQuery, owner))
	if err != nil {
		return nil, fmt.Errorf("read updated Instagram account: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit Instagram account settings: %w", err)
	}
	return &account, nil
}

// RevokeAccount is deliberately privacy-preserving and idempotent: absent,
// already-revoked, and purged links are the same successful no-op to callers.
func (s *AccountStore) RevokeAccount(ctx context.Context, owner syntax.DID) error {
	if s == nil || s.pool == nil {
		return errors.New("Instagram account store is unavailable")
	}
	if owner == "" {
		return errors.New("Instagram account owner is required")
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return fmt.Errorf("begin Instagram account revocation: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, owner); err != nil {
		return fmt.Errorf("lock Instagram account revocation: %w", err)
	}

	var linkID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id
		FROM instagram_account_links
		WHERE owner_did = $1
		  AND state IN ('active', 'membershipInactive', 'disputed')
		ORDER BY updated_at DESC, id DESC
		LIMIT 1
		FOR UPDATE
	`, owner).Scan(&linkID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read Instagram account for revocation: %w", err)
	}

	now := s.now().UTC()
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_account_links
		SET state = 'revoked',
		    igsid = NULL,
		    username = NULL,
		    username_normalized = NULL,
		    discoverable = false,
		    conflict_pending = false,
		    membership_inactive_at = NULL,
		    revoked_at = $2,
		    raw_identity_purge_at = $2::timestamptz + interval '90 days',
		    updated_at = $2
		WHERE id = $1
	`, linkID, now); err != nil {
		return fmt.Errorf("revoke Instagram account link: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_identity_claims
		SET state = 'revoked',
		    released_at = $2,
		    anonymize_at = $2::timestamptz + interval '90 days',
		    updated_at = $2
		WHERE link_id = $1 AND state IN ('active', 'disputed')
	`, linkID, now); err != nil {
		return fmt.Errorf("revoke Instagram identity claim: %w", err)
	}
	suggestionIDs, err := updateSuggestionState(ctx, tx, `
		UPDATE instagram_follow_suggestions
		SET state = 'invalidated', accepting_since = NULL,
		    terminal_at = COALESCE(terminal_at, $2), updated_at = $2
		WHERE target_did = $1 AND state IN ('pending', 'accepting')
		RETURNING id
	`, owner, now)
	if err != nil {
		return fmt.Errorf("invalidate revoked Instagram suggestions: %w", err)
	}
	if err := failUnsentFollowOperations(ctx, tx, suggestionIDs, "linkRevoked", now); err != nil {
		return err
	}
	if err := retractSuggestionNotifications(ctx, tx, suggestionIDs, "", "link_revoked", now); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_reconciliation_jobs
		SET status = 'ignored', terminal_at = $3,
		    lease_token = NULL, lease_expires_at = NULL, updated_at = $3
		WHERE owner_did = $1 AND link_id = $2
		  AND status IN ('queued', 'processing', 'retryable')
	`, owner, linkID, now); err != nil {
		return fmt.Errorf("cancel revoked Instagram reconciliation: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit Instagram account revocation: %w", err)
	}
	return nil
}

func (AccountStore) String() string {
	return "Instagram AccountStore{database:configured,clock:configured}"
}

func (s AccountStore) GoString() string {
	return s.String()
}

func (s AccountStore) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, s.String())
}

func validateAccountSettingsPatch(patch AccountSettingsPatch) error {
	if patch.Discoverable == nil && patch.Reactivate == nil {
		return ErrInvalidInstagramSettings
	}
	if patch.Reactivate != nil {
		if !*patch.Reactivate || patch.Discoverable == nil {
			return ErrInvalidInstagramSettings
		}
	}
	return nil
}
