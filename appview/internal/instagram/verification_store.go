package instagram

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VerificationAttempt struct {
	ID                uuid.UUID
	OwnerDID          syntax.DID
	State             VerificationAttemptState
	Digest            *ChallengeDigest
	CandidateIGSID    string
	CandidateUsername string
	RetryCode         AttemptRetryCode
	ExpiresAt         time.Time
	ProcessingStarted *time.Time
	TerminalAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (VerificationAttempt) String() string {
	return "Instagram verification attempt [REDACTED]"
}
func (a VerificationAttempt) GoString() string { return a.String() }

type CreateVerificationAttemptParams struct {
	ID        uuid.UUID
	OwnerDID  syntax.DID
	Digest    ChallengeDigest
	ExpiresAt time.Time
	Now       time.Time
}

type ConfirmVerificationAttemptParams struct {
	AttemptID         uuid.UUID
	OwnerDID          syntax.DID
	LinkID            uuid.UUID
	ClaimID           uuid.UUID
	ConflictID        uuid.UUID
	IGSID             string
	IGSIDDigest       ChallengeDigest
	Username          string
	Discoverable      bool
	Now               time.Time
	ConflictExpiresAt time.Time
}

type VerificationStore struct {
	pool *pgxpool.Pool
}

func NewVerificationStore(pool *pgxpool.Pool) *VerificationStore {
	return &VerificationStore{pool: pool}
}

func (s *VerificationStore) CreateVerificationAttempt(ctx context.Context, params CreateVerificationAttemptParams) (*VerificationAttempt, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("Instagram verification store is unavailable")
	}
	if params.ID == uuid.Nil || params.OwnerDID == "" || params.Digest.IsZero() || params.ExpiresAt.IsZero() || params.Now.IsZero() || !params.ExpiresAt.After(params.Now) {
		return nil, errors.New("invalid Instagram verification attempt parameters")
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, params.OwnerDID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_verification_attempts
		SET state = CASE WHEN expires_at <= $2 THEN 'expired' ELSE 'superseded' END,
		    challenge_digest_version = NULL,
		    challenge_digest = NULL,
		    candidate_igsid = NULL,
		    candidate_username = NULL,
		    terminal_at = $2,
		    updated_at = $2
		WHERE owner_did = $1
		  AND state IN ('pendingDm', 'processing', 'pendingConfirmation')
	`, params.OwnerDID, params.Now); err != nil {
		return nil, err
	}

	row := tx.QueryRow(ctx, `
		INSERT INTO instagram_verification_attempts (
			id, owner_did, state, challenge_digest_version,
			challenge_digest, expires_at, created_at, updated_at
		) VALUES ($1, $2, 'pendingDm', $3, $4, $5, $6, $6)
		RETURNING `+verificationAttemptColumns,
		params.ID, params.OwnerDID, params.Digest.Version, params.Digest.Value[:], params.ExpiresAt, params.Now)
	attempt, err := scanVerificationAttempt(row)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return attempt, nil
}

func (s *VerificationStore) GetVerificationAttempt(ctx context.Context, owner syntax.DID, id uuid.UUID, now time.Time) (*VerificationAttempt, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("Instagram verification store is unavailable")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_verification_attempts
		SET state = 'expired', challenge_digest_version = NULL,
		    challenge_digest = NULL, candidate_igsid = NULL,
		    candidate_username = NULL, terminal_at = $3, updated_at = $3
		WHERE id = $1 AND owner_did = $2
		  AND state IN ('pendingDm', 'processing', 'pendingConfirmation')
		  AND expires_at <= $3
	`, id, owner, now); err != nil {
		return nil, err
	}
	attempt, err := scanVerificationAttempt(tx.QueryRow(ctx, `
		SELECT `+verificationAttemptColumns+`
		FROM instagram_verification_attempts
		WHERE id = $1 AND owner_did = $2
	`, id, owner))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInstagramResourceNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return attempt, nil
}

func (s *VerificationStore) GetCurrentVerificationAttempt(ctx context.Context, owner syntax.DID, now time.Time) (*VerificationAttempt, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("Instagram verification store is unavailable")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_verification_attempts
		SET state = 'expired', challenge_digest_version = NULL,
		    challenge_digest = NULL, candidate_igsid = NULL,
		    candidate_username = NULL, terminal_at = $2, updated_at = $2
		WHERE owner_did = $1
		  AND state IN ('pendingDm', 'processing', 'pendingConfirmation')
		  AND expires_at <= $2
	`, owner, now); err != nil {
		return nil, err
	}
	attempt, err := scanVerificationAttempt(tx.QueryRow(ctx, `
		SELECT `+verificationAttemptColumns+`
		FROM instagram_verification_attempts
		WHERE owner_did = $1
		  AND state IN ('pendingDm', 'processing', 'pendingConfirmation')
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, owner))
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return attempt, nil
}

func (s *VerificationStore) CancelVerificationAttempt(ctx context.Context, owner syntax.DID, id uuid.UUID, now time.Time) error {
	if s == nil || s.pool == nil {
		return errors.New("Instagram verification store is unavailable")
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE instagram_verification_attempts
		SET state = CASE WHEN expires_at <= $3 THEN 'expired' ELSE 'cancelled' END,
		    challenge_digest_version = NULL,
		    challenge_digest = NULL,
		    candidate_igsid = NULL,
		    candidate_username = NULL,
		    terminal_at = $3,
		    updated_at = $3
		WHERE id = $1 AND owner_did = $2
		  AND state IN ('pendingDm', 'processing', 'pendingConfirmation')
	`, id, owner, now)
	return err
}

func (s *VerificationStore) RedeemVerificationChallenge(ctx context.Context, digest ChallengeDigest, senderIGSID string, now time.Time) (*VerificationAttempt, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("Instagram verification store is unavailable")
	}
	if digest.IsZero() || senderIGSID == "" {
		return nil, ErrInstagramResourceNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_verification_attempts
		SET state = 'expired', challenge_digest_version = NULL,
		    challenge_digest = NULL, candidate_igsid = NULL,
		    candidate_username = NULL, terminal_at = $3, updated_at = $3
		WHERE challenge_digest_version = $1 AND challenge_digest = $2
		  AND state = 'pendingDm' AND expires_at <= $3
	`, digest.Version, digest.Value[:], now); err != nil {
		return nil, err
	}
	attempt, err := scanVerificationAttempt(tx.QueryRow(ctx, `
		UPDATE instagram_verification_attempts
		SET state = 'processing', challenge_digest_version = NULL,
		    challenge_digest = NULL, candidate_igsid = $3,
		    processing_started_at = $4, updated_at = $4
		WHERE challenge_digest_version = $1 AND challenge_digest = $2
		  AND state = 'pendingDm' AND expires_at > $4
		RETURNING `+verificationAttemptColumns,
		digest.Version, digest.Value[:], senderIGSID, now))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInstagramResourceNotFound
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return attempt, nil
}

func (s *VerificationStore) SetVerificationCandidate(ctx context.Context, id uuid.UUID, username string, now time.Time) error {
	if s == nil || s.pool == nil {
		return errors.New("Instagram verification store is unavailable")
	}
	normalized, err := NormalizeInstagramUsername(username)
	if err != nil {
		return err
	}
	result, err := s.pool.Exec(ctx, `
		UPDATE instagram_verification_attempts
		SET state = 'pendingConfirmation', candidate_username = $2, updated_at = $3
		WHERE id = $1 AND state = 'processing'
	`, id, normalized, now)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrInstagramStateTransition
	}
	return nil
}

func (s *VerificationStore) ConfirmVerificationAttempt(ctx context.Context, params ConfirmVerificationAttemptParams) (ConfirmationResult, error) {
	if s == nil || s.pool == nil {
		return ConfirmationResult{}, errors.New("Instagram verification store is unavailable")
	}
	// The owner, stable IGSID, and normalized username advisory locks below are
	// the serialization boundary for confirmation. Read committed lets the
	// transaction that acquires the username lock second observe the first
	// transaction's committed claim and turn it into a private conflict instead
	// of surfacing a serialization failure to the member.
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return ConfirmationResult{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, params.OwnerDID); err != nil {
		return ConfirmationResult{}, err
	}
	attempt, err := scanVerificationAttempt(tx.QueryRow(ctx, `
		SELECT `+verificationAttemptColumns+`
		FROM instagram_verification_attempts
		WHERE id = $1 AND owner_did = $2
		FOR UPDATE
	`, params.AttemptID, params.OwnerDID))
	if errors.Is(err, pgx.ErrNoRows) {
		return ConfirmationResult{}, ErrInstagramResourceNotFound
	}
	if err != nil {
		return ConfirmationResult{}, err
	}
	if attempt.State == AttemptConfirmed {
		account, err := scanAccountView(tx.QueryRow(ctx, currentAccountViewQuery, params.OwnerDID))
		if errors.Is(err, pgx.ErrNoRows) {
			return ConfirmationResult{}, ErrInstagramStateTransition
		}
		if err != nil {
			return ConfirmationResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ConfirmationResult{}, err
		}
		return ConfirmationResult{State: AttemptConfirmed, Account: account}, nil
	}
	if attempt.State != AttemptPendingConfirmation || attempt.CandidateIGSID == "" || attempt.CandidateUsername == "" {
		return ConfirmationResult{}, ErrInstagramStateTransition
	}
	if params.IGSID != attempt.CandidateIGSID || params.Username != attempt.CandidateUsername || params.IGSIDDigest.IsZero() {
		return ConfirmationResult{}, ErrInstagramStateTransition
	}
	normalized, err := NormalizeInstagramUsername(params.Username)
	if err != nil {
		return ConfirmationResult{}, err
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended(encode($1::bytea, 'hex'), 1))`, params.IGSIDDigest.Value[:]); err != nil {
		return ConfirmationResult{}, err
	}
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 2))`, normalized); err != nil {
		return ConfirmationResult{}, err
	}

	var existingIGSIDLinkID uuid.UUID
	var existingIGSIDOwner string
	var existingIGSIDUsername string
	err = tx.QueryRow(ctx, `
		SELECT l.id, l.owner_did, COALESCE(l.username_normalized,'')
		FROM instagram_identity_claims c
		JOIN instagram_account_links l ON l.id = c.link_id
		WHERE c.igsid_digest_version = $1 AND c.igsid_digest = $2
		  AND c.state = 'active'
		FOR UPDATE OF c, l
	`, params.IGSIDDigest.Version, params.IGSIDDigest.Value[:]).Scan(&existingIGSIDLinkID, &existingIGSIDOwner, &existingIGSIDUsername)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return ConfirmationResult{}, err
	}
	hasExistingIGSID := err == nil

	var existingUsernameLinkID uuid.UUID
	var existingUsernameOwner string
	err = tx.QueryRow(ctx, `
		SELECT id, owner_did
		FROM instagram_account_links
		WHERE username_normalized = $1
		  AND state IN ('active', 'membershipInactive', 'disputed')
		FOR UPDATE
	`, normalized).Scan(&existingUsernameLinkID, &existingUsernameOwner)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return ConfirmationResult{}, err
	}
	hasExistingUsername := err == nil

	var conflictingLinkID uuid.UUID
	var conflictingOwner syntax.DID
	switch {
	case hasExistingIGSID && existingIGSIDOwner != params.OwnerDID.String():
		conflictingLinkID = existingIGSIDLinkID
		conflictingOwner = syntax.DID(existingIGSIDOwner)
	case hasExistingUsername && existingUsernameOwner != params.OwnerDID.String():
		conflictingLinkID = existingUsernameLinkID
		conflictingOwner = syntax.DID(existingUsernameOwner)
	}
	if conflictingLinkID != uuid.Nil {
		if _, err := tx.Exec(ctx, `
			INSERT INTO instagram_link_conflicts (
				id, state, existing_link_id, claimant_attempt_id,
				igsid_digest_version, igsid_digest, opened_at, expires_at,
				created_at, updated_at
			) VALUES ($1, 'open', $2, $3, $4, $5, $6, $7, $6, $6)
		`, params.ConflictID, conflictingLinkID, params.AttemptID, params.IGSIDDigest.Version, params.IGSIDDigest.Value[:], params.Now, params.ConflictExpiresAt); err != nil {
			return ConfirmationResult{}, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE instagram_account_links
			SET conflict_pending = true, discoverable = false, updated_at = $2
			WHERE id = $1
		`, conflictingLinkID, params.Now); err != nil {
			return ConfirmationResult{}, err
		}
		suggestionIDs, err := updateSuggestionState(ctx, tx, `
			UPDATE instagram_follow_suggestions
			SET state='invalidated', accepting_since=NULL,
			    terminal_at=COALESCE(terminal_at,$2), updated_at=$2
			WHERE target_did=$1 AND state IN ('pending','accepting')
			RETURNING id
		`, conflictingOwner, params.Now)
		if err != nil {
			return ConfirmationResult{}, fmt.Errorf("invalidate conflicted Instagram suggestions: %w", err)
		}
		if err := failUnsentFollowOperations(ctx, tx, suggestionIDs, "linkConflict", params.Now); err != nil {
			return ConfirmationResult{}, err
		}
		if err := retractSuggestionNotifications(ctx, tx, suggestionIDs, "", "eligibility_changed", params.Now); err != nil {
			return ConfirmationResult{}, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE instagram_reconciliation_jobs
			SET status='ignored', terminal_at=COALESCE(terminal_at,$3),
			    lease_token=NULL, lease_expires_at=NULL, updated_at=$3
			WHERE (target_did=$1 OR link_id=$2)
			  AND status IN ('queued','processing','retryable')
		`, conflictingOwner, conflictingLinkID, params.Now); err != nil {
			return ConfirmationResult{}, fmt.Errorf("cancel conflicted Instagram reconciliation: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE instagram_verification_attempts
			SET state = 'conflicted', candidate_igsid = NULL,
			    candidate_username = NULL, terminal_at = $2, updated_at = $2
			WHERE id = $1
		`, params.AttemptID, params.Now); err != nil {
			return ConfirmationResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ConfirmationResult{}, err
		}
		return ConfirmationResult{}, ErrInstagramLinkConflict
	}
	if hasExistingIGSID && existingIGSIDOwner == params.OwnerDID.String() {
		if existingIGSIDUsername != normalized {
			suggestionIDs, err := updateSuggestionState(ctx, tx, `
				UPDATE instagram_follow_suggestions
				SET state='invalidated', accepting_since=NULL,
				    terminal_at=COALESCE(terminal_at,$2), updated_at=$2
				WHERE target_did=$1 AND state IN ('pending','accepting')
				RETURNING id
			`, params.OwnerDID, params.Now)
			if err != nil {
				return ConfirmationResult{}, fmt.Errorf("invalidate refreshed Instagram suggestions: %w", err)
			}
			if err := failUnsentFollowOperations(ctx, tx, suggestionIDs, "usernameRefreshed", params.Now); err != nil {
				return ConfirmationResult{}, err
			}
			if err := retractSuggestionNotifications(ctx, tx, suggestionIDs, "", "eligibility_changed", params.Now); err != nil {
				return ConfirmationResult{}, err
			}
			if _, err := tx.Exec(ctx, `
				UPDATE instagram_reconciliation_jobs
				SET status='ignored', terminal_at=COALESCE(terminal_at,$3),
				    lease_token=NULL, lease_expires_at=NULL, updated_at=$3
				WHERE (target_did=$1 OR link_id=$2)
				  AND status IN ('queued','processing','retryable')
			`, params.OwnerDID, existingIGSIDLinkID, params.Now); err != nil {
				return ConfirmationResult{}, fmt.Errorf("cancel stale username reconciliation: %w", err)
			}
			var discoverable bool
			if err := tx.QueryRow(ctx, `
				UPDATE instagram_account_links
				SET username=$2, username_normalized=$3, updated_at=$4
				WHERE id=$1 AND owner_did=$5
				RETURNING discoverable
			`, existingIGSIDLinkID, params.Username, normalized, params.Now, params.OwnerDID).Scan(&discoverable); err != nil {
				return ConfirmationResult{}, fmt.Errorf("refresh validated Instagram username: %w", err)
			}
			if discoverable {
				if _, err := tx.Exec(ctx, `
					INSERT INTO instagram_reconciliation_jobs (
						id,owner_did,link_id,reason,status,next_attempt_at,created_at,updated_at
					) VALUES ($1,$2,$3,'instagramUsernameRefreshed','queued',$4,$4,$4)
				`, uuid.New(), params.OwnerDID, existingIGSIDLinkID, params.Now); err != nil {
					return ConfirmationResult{}, fmt.Errorf("queue refreshed Instagram username reconciliation: %w", err)
				}
			}
		}
		if _, err := tx.Exec(ctx, `
			UPDATE instagram_verification_attempts
			SET state = 'confirmed', candidate_igsid = NULL,
			    candidate_username = NULL, terminal_at = $2, updated_at = $2
			WHERE id = $1
		`, params.AttemptID, params.Now); err != nil {
			return ConfirmationResult{}, err
		}
		account, err := scanAccountView(tx.QueryRow(ctx, currentAccountViewQuery, params.OwnerDID))
		if err != nil {
			return ConfirmationResult{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return ConfirmationResult{}, err
		}
		return ConfirmationResult{State: AttemptConfirmed, Account: account}, nil
	}

	suggestionIDs, err := updateSuggestionState(ctx, tx, `
		UPDATE instagram_follow_suggestions
		SET state='invalidated', accepting_since=NULL,
		    terminal_at=COALESCE(terminal_at,$2), updated_at=$2
		WHERE target_did=$1 AND state IN ('pending','accepting')
		RETURNING id
	`, params.OwnerDID, params.Now)
	if err != nil {
		return ConfirmationResult{}, fmt.Errorf("invalidate superseded Instagram suggestions: %w", err)
	}
	if err := failUnsentFollowOperations(ctx, tx, suggestionIDs, "linkSuperseded", params.Now); err != nil {
		return ConfirmationResult{}, err
	}
	if err := retractSuggestionNotifications(ctx, tx, suggestionIDs, "", "eligibility_changed", params.Now); err != nil {
		return ConfirmationResult{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_reconciliation_jobs job
		SET status='ignored', terminal_at=COALESCE(terminal_at,$2),
		    lease_token=NULL, lease_expires_at=NULL, updated_at=$2
		WHERE job.status IN ('queued','processing','retryable')
		  AND (
			job.target_did=$1
			OR EXISTS (
				SELECT 1 FROM instagram_account_links link
				WHERE link.id=job.link_id AND link.owner_did=$1
				  AND link.state IN ('active','membershipInactive','disputed')
			)
		  )
	`, params.OwnerDID, params.Now); err != nil {
		return ConfirmationResult{}, fmt.Errorf("cancel superseded Instagram reconciliation: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_account_links
		SET state = 'superseded', discoverable = false, conflict_pending = false,
		    igsid = NULL, username = NULL, username_normalized = NULL,
		    superseded_at = $2, raw_identity_purge_at = $2, updated_at = $2
		WHERE owner_did = $1
		  AND state IN ('active', 'membershipInactive', 'disputed')
	`, params.OwnerDID, params.Now); err != nil {
		return ConfirmationResult{}, fmt.Errorf("supersede prior Instagram link: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_identity_claims
		SET state = 'revoked', released_at = $2::timestamptz,
		    anonymize_at = $2::timestamptz + interval '90 days', updated_at = $2::timestamptz
		WHERE owner_did = $1 AND state IN ('active', 'disputed')
	`, params.OwnerDID, params.Now); err != nil {
		return ConfirmationResult{}, fmt.Errorf("release prior Instagram claim: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO instagram_account_links (
			id, owner_did, state, igsid, igsid_digest_version,
			igsid_digest, username, username_normalized, discoverable,
			conflict_pending, verified_at, created_at, updated_at
		) VALUES ($1, $2, 'active', $3, $4, $5, $6, $7, $8, false, $9, $9, $9)
	`, params.LinkID, params.OwnerDID, params.IGSID, params.IGSIDDigest.Version, params.IGSIDDigest.Value[:], params.Username, normalized, params.Discoverable, params.Now); err != nil {
		return ConfirmationResult{}, fmt.Errorf("insert Instagram account link: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO instagram_identity_claims (
			id, link_id, owner_did, state, igsid_digest_version,
			igsid_digest, claimed_at, created_at, updated_at
		) VALUES ($1, $2, $3, 'active', $4, $5, $6, $6, $6)
	`, params.ClaimID, params.LinkID, params.OwnerDID, params.IGSIDDigest.Version, params.IGSIDDigest.Value[:], params.Now); err != nil {
		return ConfirmationResult{}, fmt.Errorf("insert Instagram identity claim: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_verification_attempts
		SET state = 'confirmed', candidate_igsid = NULL,
		    candidate_username = NULL, terminal_at = $2, updated_at = $2
		WHERE id = $1
	`, params.AttemptID, params.Now); err != nil {
		return ConfirmationResult{}, fmt.Errorf("complete Instagram verification attempt: %w", err)
	}
	if params.Discoverable {
		if _, err := tx.Exec(ctx, `
			INSERT INTO instagram_reconciliation_jobs (
				id, owner_did, link_id, reason, status, next_attempt_at,
				created_at, updated_at
			) VALUES ($1, $2, $3, 'instagramLinkConfirmed', 'queued', $4, $4, $4)
		`, uuid.New(), params.OwnerDID, params.LinkID, params.Now); err != nil {
			return ConfirmationResult{}, fmt.Errorf("queue confirmed Instagram link reconciliation: %w", err)
		}
	}
	account, err := scanAccountView(tx.QueryRow(ctx, currentAccountViewQuery, params.OwnerDID))
	if err != nil {
		return ConfirmationResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return ConfirmationResult{}, err
	}
	return ConfirmationResult{State: AttemptConfirmed, Account: account}, nil
}

const currentAccountViewQuery = `
	SELECT state, username, discoverable, conflict_pending,
	       state = 'membershipInactive' AS reactivation_required, verified_at
	FROM instagram_account_links
	WHERE owner_did = $1
	  AND state IN ('active', 'membershipInactive', 'disputed')
	ORDER BY updated_at DESC, id DESC
	LIMIT 1`

func scanAccountView(row verificationAttemptRow) (AccountView, error) {
	var account AccountView
	if err := row.Scan(
		&account.State,
		&account.Username,
		&account.Discoverable,
		&account.ConflictPending,
		&account.ReactivationRequired,
		&account.VerifiedAt,
	); err != nil {
		return AccountView{}, err
	}
	if !account.State.Valid() {
		return AccountView{}, ErrInvalidInstagramState
	}
	return account, nil
}

const verificationAttemptColumns = `
	id, owner_did, state, challenge_digest_version, challenge_digest,
	candidate_igsid, candidate_username, retry_code, expires_at,
	processing_started_at, terminal_at, created_at, updated_at`

type verificationAttemptRow interface {
	Scan(...any) error
}

func scanVerificationAttempt(row verificationAttemptRow) (*VerificationAttempt, error) {
	var (
		attempt           VerificationAttempt
		owner             string
		digestVersion     sql.NullInt16
		digestValue       []byte
		candidateIGSID    sql.NullString
		candidateUsername sql.NullString
		retryCode         sql.NullString
		processingStarted sql.NullTime
		terminalAt        sql.NullTime
	)
	if err := row.Scan(
		&attempt.ID,
		&owner,
		&attempt.State,
		&digestVersion,
		&digestValue,
		&candidateIGSID,
		&candidateUsername,
		&retryCode,
		&attempt.ExpiresAt,
		&processingStarted,
		&terminalAt,
		&attempt.CreatedAt,
		&attempt.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if !attempt.State.Valid() {
		return nil, fmt.Errorf("%w: verification attempt", ErrInvalidInstagramState)
	}
	attempt.OwnerDID = syntax.DID(owner)
	attempt.CandidateIGSID = candidateIGSID.String
	attempt.CandidateUsername = candidateUsername.String
	if retryCode.Valid {
		attempt.RetryCode = AttemptRetryCode(retryCode.String)
		if !attempt.RetryCode.Valid() {
			return nil, errors.New("invalid stored Instagram retry code")
		}
	}
	if digestVersion.Valid {
		if len(digestValue) != 32 {
			return nil, errors.New("invalid stored Instagram challenge digest")
		}
		digest := ChallengeDigest{Version: int(digestVersion.Int16)}
		copy(digest.Value[:], digestValue)
		attempt.Digest = &digest
	}
	if processingStarted.Valid {
		attempt.ProcessingStarted = &processingStarted.Time
	}
	if terminalAt.Valid {
		attempt.TerminalAt = &terminalAt.Time
	}
	return &attempt, nil
}
