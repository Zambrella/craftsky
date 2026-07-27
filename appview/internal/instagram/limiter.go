package instagram

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	rateLimitKeyVersion  int16 = 1
	rateLimitKeyMinBytes       = 32
)

const rateLimitKeyDomain = "craftsky:instagram-rate-limit:v1\x00"

type RateLimitScope string

const (
	RateLimitChallengeDID           RateLimitScope = "challenge.did"
	RateLimitChallengeDevice        RateLimitScope = "challenge.device"
	RateLimitChallengeIP            RateLimitScope = "challenge.ip"
	RateLimitInvalidRedemptionIGSID RateLimitScope = "invalidRedemption.igsid"
	RateLimitInvalidRedemptionIP    RateLimitScope = "invalidRedemption.ip"
	RateLimitConfirmationDID        RateLimitScope = "confirmation.did"
	RateLimitConfirmationDevice     RateLimitScope = "confirmation.device"
	RateLimitImportDID              RateLimitScope = "import.did"
	RateLimitImportDevice           RateLimitScope = "import.device"
	RateLimitWebhookGlobal          RateLimitScope = "webhook.global"
	RateLimitWebhookIP              RateLimitScope = "webhook.ip"
	RateLimitMetaLookupIGSID        RateLimitScope = "metaLookup.igsid"
)

// RateLimitKey is the persistence-safe representation of one abuse identity.
// The source DID/device/IP/IGSID is never retained after Key returns.
type RateLimitKey struct {
	scope   RateLimitScope
	version int16
	digest  [sha256.Size]byte
}

func (k RateLimitKey) Scope() RateLimitScope {
	return k.scope
}

func (RateLimitKey) String() string {
	return "Instagram rate-limit key [REDACTED]"
}

func (k RateLimitKey) GoString() string {
	return k.String()
}

func (k RateLimitKey) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, k.String())
}

type RateLimitDecision struct {
	Allowed    bool
	Remaining  int
	RetryAfter time.Duration
}

func (d RateLimitDecision) String() string {
	return fmt.Sprintf(
		"RateLimitDecision{allowed:%t,remaining:%d,retryAfter:%s}",
		d.Allowed,
		d.Remaining,
		d.RetryAfter,
	)
}

func (d RateLimitDecision) GoString() string {
	return d.String()
}

// PostgresRateLimiter applies one atomic fixed-window increment against the
// shared instagram_rate_limit_buckets table.
type PostgresRateLimiter struct {
	pool  *pgxpool.Pool
	key   []byte
	clock func() time.Time
}

func NewPostgresRateLimiter(pool *pgxpool.Pool, key []byte, clock func() time.Time) (*PostgresRateLimiter, error) {
	if pool == nil {
		return nil, errors.New("Instagram rate-limit database is required")
	}
	if len(key) < rateLimitKeyMinBytes {
		return nil, fmt.Errorf("Instagram rate-limit HMAC key must contain at least %d bytes", rateLimitKeyMinBytes)
	}
	if clock == nil {
		return nil, errors.New("Instagram rate-limit clock is required")
	}
	return &PostgresRateLimiter{pool: pool, key: append([]byte(nil), key...), clock: clock}, nil
}

func (l *PostgresRateLimiter) Key(scope RateLimitScope, identifier []byte) (RateLimitKey, error) {
	if !validRateLimitScope(scope) {
		return RateLimitKey{}, errors.New("valid Instagram rate-limit scope is required")
	}
	if scope == RateLimitWebhookGlobal && len(identifier) != 0 {
		return RateLimitKey{}, errors.New("global Instagram rate-limit scope does not accept an identifier")
	}
	if scope != RateLimitWebhookGlobal && len(identifier) == 0 {
		return RateLimitKey{}, errors.New("Instagram rate-limit identifier is required")
	}
	mac := hmac.New(sha256.New, l.key)
	_, _ = mac.Write([]byte(rateLimitKeyDomain))
	_, _ = mac.Write([]byte(scope))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write(identifier)
	var digest [sha256.Size]byte
	copy(digest[:], mac.Sum(nil))
	return RateLimitKey{scope: scope, version: rateLimitKeyVersion, digest: digest}, nil
}

func (l *PostgresRateLimiter) Allow(ctx context.Context, key RateLimitKey, window time.Duration, limit int) (RateLimitDecision, error) {
	if key.scope == "" || key.version != rateLimitKeyVersion || key.digest == [sha256.Size]byte{} {
		return RateLimitDecision{}, errors.New("valid Instagram rate-limit key is required")
	}
	if window <= 0 {
		return RateLimitDecision{}, errors.New("Instagram rate-limit window must be positive")
	}
	if limit <= 0 || limit >= math.MaxInt32 {
		return RateLimitDecision{}, errors.New("Instagram rate-limit limit must be positive and bounded")
	}

	now := l.clock().UTC()
	windowStart := now.Truncate(window)
	windowEnd := windowStart.Add(window)

	var (
		count           int
		storedWindowEnd time.Time
	)
	err := l.pool.QueryRow(ctx, `
		INSERT INTO instagram_rate_limit_buckets (
			bucket_scope, key_version, key_digest,
			window_start, window_end, count, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, 1, $6, $6)
		ON CONFLICT (bucket_scope, key_version, key_digest, window_start)
		DO UPDATE SET
			count = CASE
				WHEN instagram_rate_limit_buckets.count < $7
				THEN instagram_rate_limit_buckets.count + 1
				ELSE instagram_rate_limit_buckets.count
			END,
			updated_at = EXCLUDED.updated_at
		RETURNING count, window_end
	`, string(key.scope), key.version, key.digest[:], windowStart, windowEnd, now, limit+1).Scan(&count, &storedWindowEnd)
	if err != nil {
		return RateLimitDecision{}, fmt.Errorf("apply Instagram rate limit: %w", err)
	}

	if count <= limit {
		return RateLimitDecision{Allowed: true, Remaining: limit - count}, nil
	}
	retryAfter := storedWindowEnd.Sub(now)
	if retryAfter < 0 {
		retryAfter = 0
	}
	return RateLimitDecision{Allowed: false, RetryAfter: retryAfter}, nil
}

// AllowIdentifier applies a scope without exposing the intermediate keyed
// value. It is used by background workers that receive a private provider ID.
func (l *PostgresRateLimiter) AllowIdentifier(ctx context.Context, scope RateLimitScope, identifier []byte, window time.Duration, limit int) (RateLimitDecision, error) {
	key, err := l.Key(scope, identifier)
	if err != nil {
		return RateLimitDecision{}, err
	}
	return l.Allow(ctx, key, window, limit)
}

func (PostgresRateLimiter) String() string {
	return "PostgresRateLimiter{database:configured,hmacKey:[REDACTED],clock:configured}"
}

func (l PostgresRateLimiter) GoString() string {
	return l.String()
}

func (l PostgresRateLimiter) Format(state fmt.State, _ rune) {
	_, _ = io.WriteString(state, l.String())
}

func validRateLimitScope(scope RateLimitScope) bool {
	switch scope {
	case RateLimitChallengeDID,
		RateLimitChallengeDevice,
		RateLimitChallengeIP,
		RateLimitInvalidRedemptionIGSID,
		RateLimitInvalidRedemptionIP,
		RateLimitConfirmationDID,
		RateLimitConfirmationDevice,
		RateLimitImportDID,
		RateLimitImportDevice,
		RateLimitWebhookGlobal,
		RateLimitWebhookIP,
		RateLimitMetaLookupIGSID:
		return true
	default:
		return false
	}
}
