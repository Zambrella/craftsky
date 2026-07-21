package instagram

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// VerificationWebhookRedeemer atomically binds durable webhook work to a
// verification attempt. The binding is retained after the challenge digest is
// cleared so a crash or provider retry cannot redeem the work against a
// different attempt.
type VerificationWebhookRedeemer struct {
	store *VerificationStore
}

var _ WebhookRedeemer = (*VerificationWebhookRedeemer)(nil)

func NewVerificationWebhookRedeemer(store *VerificationStore) (*VerificationWebhookRedeemer, error) {
	if store == nil || store.pool == nil {
		return nil, errors.New("Instagram verification webhook redeemer requires a store")
	}
	return &VerificationWebhookRedeemer{store: store}, nil
}

func (*VerificationWebhookRedeemer) String() string {
	return "Instagram verification webhook redeemer [REDACTED]"
}

func (*VerificationWebhookRedeemer) GoString() string {
	return "Instagram verification webhook redeemer [REDACTED]"
}

func (r *VerificationWebhookRedeemer) RedeemWebhookChallenge(ctx context.Context, request WebhookRedemptionRequest) (WebhookRedemption, error) {
	if r == nil || r.store == nil || r.store.pool == nil {
		return WebhookRedemption{}, errors.New("Instagram verification webhook redeemer is unavailable")
	}
	if request.WorkID == uuid.Nil || request.LeaseToken == uuid.Nil || request.ChallengeDigest.Version <= 0 ||
		request.ChallengeDigest.IsZero() || request.SenderIGSID == "" || request.Now.IsZero() {
		return WebhookRedemption{}, ErrInstagramResourceNotFound
	}
	now := request.Now.UTC()
	tx, err := r.store.pool.Begin(ctx)
	if err != nil {
		return WebhookRedemption{}, err
	}
	defer tx.Rollback(ctx)

	var (
		mappedAttempt uuid.NullUUID
		storedSender  sql.NullString
		digestVersion sql.NullInt16
		digestValue   []byte
	)
	err = tx.QueryRow(ctx, `
		SELECT verification_attempt_id, sender_igsid,
		       challenge_digest_version, challenge_digest
		FROM instagram_webhook_work
		WHERE id = $1 AND status = 'processing' AND lease_token = $2
		  AND lease_expires_at > $3
		FOR UPDATE
	`, request.WorkID, request.LeaseToken, now).Scan(
		&mappedAttempt, &storedSender, &digestVersion, &digestValue,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return WebhookRedemption{}, ErrInstagramResourceNotFound
	}
	if err != nil {
		return WebhookRedemption{}, err
	}
	if !storedSender.Valid || storedSender.String != request.SenderIGSID ||
		!digestVersion.Valid || len(digestValue) != len(request.ChallengeDigest.Value) {
		return WebhookRedemption{}, ErrInstagramResourceNotFound
	}
	storedDigest := ChallengeDigest{Version: int(digestVersion.Int16)}
	copy(storedDigest.Value[:], digestValue)
	if !storedDigest.Equal(request.ChallengeDigest) {
		return WebhookRedemption{}, ErrInstagramResourceNotFound
	}

	if mappedAttempt.Valid {
		attempt, err := scanVerificationAttempt(tx.QueryRow(ctx, `
			SELECT `+verificationAttemptColumns+`
			FROM instagram_verification_attempts
			WHERE id = $1
			FOR UPDATE
		`, mappedAttempt.UUID))
		if errors.Is(err, pgx.ErrNoRows) {
			return WebhookRedemption{}, ErrInstagramResourceNotFound
		}
		if err != nil {
			return WebhookRedemption{}, err
		}
		if (attempt.State == AttemptProcessing || attempt.State == AttemptPendingConfirmation) &&
			!now.Before(attempt.ExpiresAt) {
			if _, err := tx.Exec(ctx, `
				UPDATE instagram_verification_attempts
				SET state = 'expired', challenge_digest_version = NULL,
				    challenge_digest = NULL, candidate_igsid = NULL,
				    candidate_username = NULL, terminal_at = $2, updated_at = $2
				WHERE id = $1
			`, attempt.ID, now); err != nil {
				return WebhookRedemption{}, err
			}
			if err := tx.Commit(ctx); err != nil {
				return WebhookRedemption{}, err
			}
			return WebhookRedemption{}, ErrInstagramResourceNotFound
		}
		if (attempt.State != AttemptProcessing && attempt.State != AttemptPendingConfirmation) ||
			attempt.CandidateIGSID != request.SenderIGSID {
			return WebhookRedemption{}, ErrInstagramResourceNotFound
		}
		if err := tx.Commit(ctx); err != nil {
			return WebhookRedemption{}, err
		}
		return WebhookRedemption{AttemptID: attempt.ID, OwnerDID: attempt.OwnerDID}, nil
	}

	expired, err := tx.Exec(ctx, `
		UPDATE instagram_verification_attempts
		SET state = 'expired', challenge_digest_version = NULL,
		    challenge_digest = NULL, candidate_igsid = NULL,
		    candidate_username = NULL, terminal_at = $3, updated_at = $3
		WHERE challenge_digest_version = $1 AND challenge_digest = $2
		  AND state = 'pendingDm' AND expires_at <= $3
	`, storedDigest.Version, storedDigest.Value[:], now)
	if err != nil {
		return WebhookRedemption{}, err
	}
	attempt, err := scanVerificationAttempt(tx.QueryRow(ctx, `
		UPDATE instagram_verification_attempts
		SET state = 'processing', challenge_digest_version = NULL,
		    challenge_digest = NULL, candidate_igsid = $3,
		    processing_started_at = $4, updated_at = $4
		WHERE challenge_digest_version = $1 AND challenge_digest = $2
		  AND state = 'pendingDm' AND expires_at > $4
		RETURNING `+verificationAttemptColumns,
		storedDigest.Version, storedDigest.Value[:], request.SenderIGSID, now))
	if errors.Is(err, pgx.ErrNoRows) {
		if expired.RowsAffected() > 0 {
			if err := tx.Commit(ctx); err != nil {
				return WebhookRedemption{}, err
			}
		}
		return WebhookRedemption{}, ErrInstagramResourceNotFound
	}
	if err != nil {
		return WebhookRedemption{}, err
	}
	result, err := tx.Exec(ctx, `
		UPDATE instagram_webhook_work
		SET verification_attempt_id = $2, updated_at = $3
		WHERE id = $1 AND verification_attempt_id IS NULL
	`, request.WorkID, attempt.ID, now)
	if err != nil {
		return WebhookRedemption{}, err
	}
	if result.RowsAffected() != 1 {
		return WebhookRedemption{}, ErrInstagramStateTransition
	}
	if err := tx.Commit(ctx); err != nil {
		return WebhookRedemption{}, err
	}
	return WebhookRedemption{AttemptID: attempt.ID, OwnerDID: attempt.OwnerDID}, nil
}

func (r *VerificationWebhookRedeemer) SetWebhookCandidate(ctx context.Context, attemptID uuid.UUID, username string, now time.Time) error {
	if r == nil || r.store == nil || r.store.pool == nil {
		return errors.New("Instagram verification webhook redeemer is unavailable")
	}
	if attemptID == uuid.Nil || now.IsZero() {
		return ErrInstagramStateTransition
	}
	normalized, err := NormalizeInstagramUsername(username)
	if err != nil {
		return err
	}
	tx, err := r.store.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var (
		state     VerificationAttemptState
		candidate sql.NullString
		expiresAt time.Time
	)
	err = tx.QueryRow(ctx, `
		SELECT state, candidate_username, expires_at
		FROM instagram_verification_attempts
		WHERE id = $1
		FOR UPDATE
	`, attemptID).Scan(&state, &candidate, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInstagramResourceNotFound
	}
	if err != nil {
		return err
	}
	if (state == AttemptProcessing || state == AttemptPendingConfirmation) && !now.Before(expiresAt) {
		if _, err := tx.Exec(ctx, `
			UPDATE instagram_verification_attempts
			SET state = 'expired', candidate_igsid = NULL,
			    candidate_username = NULL, terminal_at = $2, updated_at = $2
			WHERE id = $1
		`, attemptID, now.UTC()); err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		return ErrInstagramStateTransition
	}
	switch state {
	case AttemptProcessing:
		if _, err := tx.Exec(ctx, `
			UPDATE instagram_verification_attempts
			SET state = 'pendingConfirmation', candidate_username = $2,
			    updated_at = $3
			WHERE id = $1
		`, attemptID, normalized, now.UTC()); err != nil {
			return err
		}
	case AttemptPendingConfirmation:
		if !candidate.Valid || candidate.String != normalized {
			return ErrInstagramStateTransition
		}
	default:
		return ErrInstagramStateTransition
	}
	return tx.Commit(ctx)
}

func (r *VerificationWebhookRedeemer) InactivateWebhookOwner(ctx context.Context, attemptID uuid.UUID, owner syntax.DID, now time.Time) error {
	if owner == "" {
		return ErrInstagramStateTransition
	}
	return r.rejectAttempt(ctx, attemptID, owner, RetryMembershipInactive, now)
}

func (r *VerificationWebhookRedeemer) RejectWebhookAttempt(ctx context.Context, attemptID uuid.UUID, retryCode AttemptRetryCode, now time.Time) error {
	return r.rejectAttempt(ctx, attemptID, "", retryCode, now)
}

func (r *VerificationWebhookRedeemer) rejectAttempt(ctx context.Context, attemptID uuid.UUID, expectedOwner syntax.DID, retryCode AttemptRetryCode, now time.Time) error {
	if r == nil || r.store == nil || r.store.pool == nil {
		return errors.New("Instagram verification webhook redeemer is unavailable")
	}
	if attemptID == uuid.Nil || !retryCode.Valid() || now.IsZero() {
		return ErrInstagramStateTransition
	}
	tx, err := r.store.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var (
		owner      syntax.DID
		state      VerificationAttemptState
		storedCode sql.NullString
	)
	err = tx.QueryRow(ctx, `
		SELECT owner_did, state, retry_code
		FROM instagram_verification_attempts
		WHERE id = $1
		FOR UPDATE
	`, attemptID).Scan(&owner, &state, &storedCode)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInstagramResourceNotFound
	}
	if err != nil {
		return err
	}
	if expectedOwner != "" && owner != expectedOwner {
		return ErrInstagramResourceNotFound
	}
	if state == AttemptRejected {
		if storedCode.Valid && AttemptRetryCode(storedCode.String) == retryCode {
			return tx.Commit(ctx)
		}
		return ErrInstagramStateTransition
	}
	if state != AttemptProcessing && state != AttemptPendingConfirmation {
		return ErrInstagramStateTransition
	}
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_verification_attempts
		SET state = 'rejected', challenge_digest_version = NULL,
		    challenge_digest = NULL, candidate_igsid = NULL,
		    candidate_username = NULL, retry_code = $2,
		    terminal_at = $3, updated_at = $3
		WHERE id = $1
	`, attemptID, retryCode, now.UTC()); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
