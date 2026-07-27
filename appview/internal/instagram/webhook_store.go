package instagram

import (
	"context"
	"crypto/subtle"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/integrations/instagrammeta"
)

type WebhookWorkStatus string

const (
	WebhookWorkQueued     WebhookWorkStatus = "queued"
	WebhookWorkProcessing WebhookWorkStatus = "processing"
	WebhookWorkRetryable  WebhookWorkStatus = "retryable"
	WebhookWorkCompleted  WebhookWorkStatus = "completed"
	WebhookWorkIgnored    WebhookWorkStatus = "ignored"
	WebhookWorkFailed     WebhookWorkStatus = "failed"
)

var (
	ErrInvalidWebhookWork   = errors.New("invalid Instagram webhook work")
	ErrWebhookBatchTooLarge = errors.New("Instagram webhook work batch is too large")
	ErrWebhookLeaseLost     = errors.New("Instagram webhook work lease is no longer owned")
)

type WebhookTerminalReason string

const (
	WebhookReasonProcessed            WebhookTerminalReason = "processed"
	WebhookReasonChallengeUnavailable WebhookTerminalReason = "challengeUnavailable"
	WebhookReasonMembershipInactive   WebhookTerminalReason = "membershipInactive"
	WebhookReasonInvalidProfile       WebhookTerminalReason = "invalidProfileResponse"
	WebhookReasonProviderPermanent    WebhookTerminalReason = "providerPermanent"
	WebhookReasonMaxAttempts          WebhookTerminalReason = "maxAttempts"
	WebhookReasonMaxAge               WebhookTerminalReason = "maxAge"
	WebhookReasonRateLimited          WebhookTerminalReason = "rateLimited"
)

type WebhookWork struct {
	ID                  uuid.UUID                 `json:"-"`
	MessageIDDigest     instagrammeta.KeyedDigest `json:"-"`
	SenderIGSID         string                    `json:"-"`
	OfficialAccountID   string                    `json:"-"`
	ChallengeDigest     instagrammeta.KeyedDigest `json:"-"`
	EventAt             time.Time                 `json:"-"`
	Status              WebhookWorkStatus         `json:"-"`
	Attempts            int                       `json:"-"`
	NextAttemptAt       time.Time                 `json:"-"`
	ProcessingStartedAt time.Time                 `json:"-"`
	LeaseToken          uuid.UUID                 `json:"-"`
	LeaseExpiresAt      time.Time                 `json:"-"`
	CreatedAt           time.Time                 `json:"-"`
}

func (WebhookWork) String() string {
	return "Instagram webhook work [REDACTED]"
}

func (WebhookWork) GoString() string {
	return "Instagram webhook work [REDACTED]"
}

type WebhookStore struct {
	pool          *pgxpool.Pool
	leaseDuration time.Duration
	retryPolicy   WebhookRetryPolicy
}

type WebhookStoreOptions struct {
	LeaseDuration time.Duration
	RetryPolicy   WebhookRetryPolicy
}

var _ instagrammeta.WebhookWorkSink = (*WebhookStore)(nil)
var _ instagrammeta.GuardedWebhookWorkSink = (*WebhookStore)(nil)
var _ WebhookWorkQueue = (*WebhookStore)(nil)

func NewWebhookStore(pool *pgxpool.Pool) *WebhookStore {
	store, _ := NewWebhookStoreWithOptions(pool, WebhookStoreOptions{})
	return store
}

func NewWebhookStoreWithOptions(pool *pgxpool.Pool, options WebhookStoreOptions) (*WebhookStore, error) {
	if pool == nil {
		return nil, errors.New("Instagram webhook store database is unavailable")
	}
	if options.LeaseDuration == 0 {
		options.LeaseDuration = WebhookLeaseDuration
	}
	if options.RetryPolicy == (WebhookRetryPolicy{}) {
		options.RetryPolicy = DefaultWebhookRetryPolicy()
	}
	if options.LeaseDuration <= 0 || options.LeaseDuration > WebhookLeaseDuration || !options.RetryPolicy.valid() {
		return nil, errors.New("invalid Instagram webhook store limits")
	}
	return &WebhookStore{pool: pool, leaseDuration: options.LeaseDuration, retryPolicy: options.RetryPolicy}, nil
}

func (*WebhookStore) String() string {
	return "Instagram webhook store [REDACTED]"
}

func (*WebhookStore) GoString() string {
	return "Instagram webhook store [REDACTED]"
}

// EnqueueWebhookWork persists an entire signed webhook delivery atomically.
// Duplicate message digests are successful no-ops.
func (s *WebhookStore) EnqueueWebhookWork(ctx context.Context, items []instagrammeta.WorkItem, now time.Time) (int, error) {
	return s.enqueueWebhookWork(ctx, items, now, nil)
}

// EnqueueWebhookWorkGuarded persists the complete signed delivery in one
// transaction. Only newly inserted events whose challenge is not currently
// redeemable consume invalid-redemption source-IP quota. Excess events remain
// durably deduplicated by message digest, but are terminal on insertion and
// retain no sender, official-account, or challenge fields for a worker to use.
func (s *WebhookStore) EnqueueWebhookWorkGuarded(ctx context.Context, items []instagrammeta.WorkItem, now time.Time, limiter instagrammeta.WebhookInvalidRedemptionLimiter) (int, error) {
	if limiter == nil {
		return 0, errors.New("Instagram invalid-redemption source limiter is unavailable")
	}
	return s.enqueueWebhookWork(ctx, items, now, limiter)
}

func (s *WebhookStore) enqueueWebhookWork(ctx context.Context, items []instagrammeta.WorkItem, now time.Time, limiter instagrammeta.WebhookInvalidRedemptionLimiter) (int, error) {
	if s == nil || s.pool == nil {
		return 0, errors.New("Instagram webhook store is unavailable")
	}
	if len(items) > instagrammeta.MaxSupportedEvents {
		return 0, ErrWebhookBatchTooLarge
	}
	for _, item := range items {
		if !validReducedWorkItem(item) {
			return 0, ErrInvalidWebhookWork
		}
	}
	if len(items) == 0 {
		return 0, nil
	}
	if now.IsZero() {
		return 0, ErrInvalidWebhookWork
	}
	now = now.UTC()

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	inserted := 0
	for _, item := range items {
		workID := uuid.New()
		result, err := tx.Exec(ctx, `
			INSERT INTO instagram_webhook_work (
				id, message_digest_version, message_digest,
				sender_igsid, official_account_id,
				challenge_digest_version, challenge_digest,
				event_at, status, attempts, next_attempt_at,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'queued', 0, $9, $9, $9)
			ON CONFLICT (message_digest_version, message_digest) DO NOTHING
		`, workID, item.MessageIDDigest.Version, item.MessageIDDigest.Value[:],
			item.SenderIGSID, item.OfficialAccountID,
			item.ChallengeDigest.Version, item.ChallengeDigest.Value[:],
			item.EventAt.UTC(), now)
		if err != nil {
			return 0, err
		}
		if result.RowsAffected() == 0 {
			// A duplicate delivery is already durable and does not consume the
			// invalid-redemption source-IP quota.
			continue
		}
		inserted++
		if limiter == nil {
			continue
		}

		var redeemable bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM instagram_verification_attempts attempt
				JOIN craftsky_profiles member ON member.did = attempt.owner_did
				WHERE attempt.challenge_digest_version = $1
				  AND attempt.challenge_digest = $2
				  AND attempt.state = 'pendingDm'
				  AND attempt.expires_at > $3
			)
		`, item.ChallengeDigest.Version, item.ChallengeDigest.Value[:], now).Scan(&redeemable); err != nil {
			return 0, err
		}
		if redeemable {
			continue
		}

		decision, err := limiter.AllowInvalidRedemption(ctx)
		if err != nil {
			return 0, err
		}
		if decision.Allowed {
			continue
		}
		result, err = tx.Exec(ctx, `
			UPDATE instagram_webhook_work
			SET status = 'ignored', sender_igsid = NULL,
			    official_account_id = NULL,
			    challenge_digest_version = NULL, challenge_digest = NULL,
			    terminal_at = $2, terminal_reason = $3, updated_at = $2
			WHERE id = $1 AND status = 'queued'
		`, workID, now, WebhookReasonRateLimited)
		if err != nil {
			return 0, err
		}
		if result.RowsAffected() != 1 {
			return 0, ErrInvalidWebhookWork
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return inserted, nil
}

// ClaimWebhookWork leases due rows without waiting on rows selected by another
// transaction. Expired leases are recovered before new work is selected.
func (s *WebhookStore) ClaimWebhookWork(ctx context.Context, limit int, now time.Time) ([]WebhookWork, error) {
	if s == nil || s.pool == nil {
		return nil, errors.New("Instagram webhook store is unavailable")
	}
	if limit <= 0 || limit > instagrammeta.MaxSupportedEvents || now.IsZero() {
		return nil, ErrInvalidWebhookWork
	}
	now = now.UTC()
	maxAgeCutoff := now.Add(-s.retryPolicy.MaxProcessingAge)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		UPDATE instagram_webhook_work
		SET status = 'failed', sender_igsid = NULL,
		    challenge_digest_version = NULL, challenge_digest = NULL,
		    lease_token = NULL, lease_expires_at = NULL,
		    terminal_at = $1,
		    terminal_reason = CASE
		        WHEN attempts >= $2 THEN 'maxAttempts'
		        ELSE 'maxAge'
		    END,
		    updated_at = $1
		WHERE ((status = 'processing' AND lease_expires_at <= $1)
		       OR status = 'retryable')
		  AND (attempts >= $2
		       OR (processing_started_at IS NOT NULL AND processing_started_at <= $3))
	`, now, s.retryPolicy.MaxAttempts, maxAgeCutoff); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE instagram_webhook_work
		SET status = 'retryable', next_attempt_at = $1,
		    lease_token = NULL, lease_expires_at = NULL, updated_at = $1
		WHERE status = 'processing' AND lease_expires_at <= $1
	`, now); err != nil {
		return nil, err
	}

	rows, err := tx.Query(ctx, `
		SELECT id, message_digest_version, message_digest,
		       sender_igsid, official_account_id,
		       challenge_digest_version, challenge_digest,
		       event_at, status, attempts, next_attempt_at,
		       processing_started_at, created_at
		FROM instagram_webhook_work
		WHERE status IN ('queued', 'retryable') AND next_attempt_at <= $1
		ORDER BY next_attempt_at, id
		FOR UPDATE SKIP LOCKED
		LIMIT $2
	`, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	claimed := make([]WebhookWork, 0, limit)
	for rows.Next() {
		item, err := scanWebhookWork(rows)
		if err != nil {
			return nil, err
		}
		item.Attempts++
		item.Status = WebhookWorkProcessing
		item.LeaseToken = uuid.New()
		item.LeaseExpiresAt = now.Add(s.leaseDuration)
		if item.ProcessingStartedAt.IsZero() {
			item.ProcessingStartedAt = now
		}
		claimed = append(claimed, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	for _, item := range claimed {
		if _, err := tx.Exec(ctx, `
			UPDATE instagram_webhook_work
			SET status = 'processing', attempts = $2,
			    processing_started_at = COALESCE(processing_started_at, $3),
			    lease_token = $4, lease_expires_at = $5, updated_at = $3
			WHERE id = $1
		`, item.ID, item.Attempts, now, item.LeaseToken, item.LeaseExpiresAt); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return claimed, nil
}

func (s *WebhookStore) CompleteWebhookWork(ctx context.Context, id, leaseToken uuid.UUID, now time.Time, reason WebhookTerminalReason) error {
	if reason != WebhookReasonProcessed {
		return ErrInvalidWebhookWork
	}
	return s.finishWebhookWork(ctx, id, leaseToken, WebhookWorkCompleted, reason, now)
}

func (s *WebhookStore) IgnoreWebhookWork(ctx context.Context, id, leaseToken uuid.UUID, now time.Time, reason WebhookTerminalReason) error {
	if reason != WebhookReasonChallengeUnavailable &&
		reason != WebhookReasonMembershipInactive &&
		reason != WebhookReasonRateLimited {
		return ErrInvalidWebhookWork
	}
	return s.finishWebhookWork(ctx, id, leaseToken, WebhookWorkIgnored, reason, now)
}

func (s *WebhookStore) FailWebhookWork(ctx context.Context, id, leaseToken uuid.UUID, now time.Time, reason WebhookTerminalReason) error {
	if reason != WebhookReasonInvalidProfile && reason != WebhookReasonProviderPermanent &&
		reason != WebhookReasonMaxAttempts && reason != WebhookReasonMaxAge {
		return ErrInvalidWebhookWork
	}
	return s.finishWebhookWork(ctx, id, leaseToken, WebhookWorkFailed, reason, now)
}

func (s *WebhookStore) RetryWebhookWork(ctx context.Context, id, leaseToken uuid.UUID, nextAttemptAt, now time.Time) error {
	if s == nil || s.pool == nil {
		return errors.New("Instagram webhook store is unavailable")
	}
	if id == uuid.Nil || leaseToken == uuid.Nil || now.IsZero() || !nextAttemptAt.After(now) {
		return ErrInvalidWebhookWork
	}
	now = now.UTC()
	nextAttemptAt = nextAttemptAt.UTC()
	result, err := s.pool.Exec(ctx, `
		UPDATE instagram_webhook_work
		SET status = 'retryable', next_attempt_at = $3,
		    lease_token = NULL, lease_expires_at = NULL,
		    terminal_reason = NULL, updated_at = $4
		WHERE id = $1 AND status = 'processing' AND lease_token = $2
		  AND lease_expires_at > $4 AND attempts < $5
		  AND processing_started_at > $6
	`, id, leaseToken, nextAttemptAt, now, s.retryPolicy.MaxAttempts, nextAttemptAt.Add(-s.retryPolicy.MaxProcessingAge))
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrWebhookLeaseLost
	}
	return nil
}

func (s *WebhookStore) finishWebhookWork(ctx context.Context, id, leaseToken uuid.UUID, status WebhookWorkStatus, reason WebhookTerminalReason, now time.Time) error {
	if s == nil || s.pool == nil {
		return errors.New("Instagram webhook store is unavailable")
	}
	if id == uuid.Nil || leaseToken == uuid.Nil || now.IsZero() {
		return ErrInvalidWebhookWork
	}
	result, err := s.pool.Exec(ctx, `
		UPDATE instagram_webhook_work
		SET status = $3, sender_igsid = NULL,
		    challenge_digest_version = NULL, challenge_digest = NULL,
		    lease_token = NULL, lease_expires_at = NULL,
		    terminal_at = $4, terminal_reason = $5, updated_at = $4
		WHERE id = $1 AND status = 'processing' AND lease_token = $2
		  AND lease_expires_at > $4
	`, id, leaseToken, status, now.UTC(), reason)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrWebhookLeaseLost
	}
	return nil
}

type webhookWorkRow interface {
	Scan(...any) error
}

func scanWebhookWork(row webhookWorkRow) (WebhookWork, error) {
	var (
		item              WebhookWork
		messageVersion    int16
		messageDigest     []byte
		challengeVersion  int16
		challengeDigest   []byte
		processingStarted sql.NullTime
	)
	if err := row.Scan(
		&item.ID, &messageVersion, &messageDigest,
		&item.SenderIGSID, &item.OfficialAccountID,
		&challengeVersion, &challengeDigest,
		&item.EventAt, &item.Status, &item.Attempts,
		&item.NextAttemptAt, &processingStarted, &item.CreatedAt,
	); err != nil {
		return WebhookWork{}, err
	}
	if len(messageDigest) != 32 || len(challengeDigest) != 32 ||
		item.SenderIGSID == "" || item.OfficialAccountID == "" {
		return WebhookWork{}, ErrInvalidWebhookWork
	}
	item.MessageIDDigest.Version = int(messageVersion)
	copy(item.MessageIDDigest.Value[:], messageDigest)
	item.ChallengeDigest.Version = int(challengeVersion)
	copy(item.ChallengeDigest.Value[:], challengeDigest)
	if processingStarted.Valid {
		item.ProcessingStartedAt = processingStarted.Time
	}
	return item, nil
}

func validReducedWorkItem(item instagrammeta.WorkItem) bool {
	return item.MessageIDDigest.Version > 0 &&
		item.ChallengeDigest.Version > 0 &&
		!zeroDigest(item.MessageIDDigest.Value[:]) &&
		!zeroDigest(item.ChallengeDigest.Value[:]) &&
		item.SenderIGSID != "" && len(item.SenderIGSID) <= 128 &&
		item.OfficialAccountID != "" && len(item.OfficialAccountID) <= 128 &&
		!item.EventAt.IsZero()
}

func zeroDigest(value []byte) bool {
	var zero [32]byte
	return len(value) != len(zero) || subtle.ConstantTimeCompare(value, zero[:]) == 1
}
