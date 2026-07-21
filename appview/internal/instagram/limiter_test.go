package instagram

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/testdb"
)

func TestPostgresRateLimiterFixedWindowBoundariesAndExpiry(t *testing.T) {
	pool := instagramLimiterTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 12, 7, 0, 0, time.UTC)
	secret := []byte("synthetic-limiter-hmac-key-32bytes")
	rawIdentifier := []byte("did:plc:synthetic-limiter-member")

	limiter, err := NewPostgresRateLimiter(pool, secret, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewPostgresRateLimiter: %v", err)
	}
	key, err := limiter.Key(RateLimitChallengeDID, rawIdentifier)
	if err != nil {
		t.Fatalf("Key: %v", err)
	}

	first, err := limiter.Allow(ctx, key, 15*time.Minute, 2)
	if err != nil {
		t.Fatalf("first Allow: %v", err)
	}
	assertRateLimitDecision(t, first, true, 1, 0)

	atLimit, err := limiter.Allow(ctx, key, 15*time.Minute, 2)
	if err != nil {
		t.Fatalf("at-limit Allow: %v", err)
	}
	assertRateLimitDecision(t, atLimit, true, 0, 0)

	overLimit, err := limiter.Allow(ctx, key, 15*time.Minute, 2)
	if err != nil {
		t.Fatalf("over-limit Allow: %v", err)
	}
	assertRateLimitDecision(t, overLimit, false, 0, 8*time.Minute)

	var (
		storedScope  string
		storedDigest []byte
		storedCount  int
	)
	if err := pool.QueryRow(ctx, `
		SELECT bucket_scope, key_digest, count
		FROM instagram_rate_limit_buckets
		WHERE bucket_scope = $1
	`, string(RateLimitChallengeDID)).Scan(&storedScope, &storedDigest, &storedCount); err != nil {
		t.Fatalf("inspect bucket: %v", err)
	}
	if storedScope != string(RateLimitChallengeDID) || storedCount != 3 {
		t.Fatalf("stored bucket = scope %q count %d, want %q/3", storedScope, storedCount, RateLimitChallengeDID)
	}
	if len(storedDigest) != 32 || bytes.Equal(storedDigest, rawIdentifier) || bytes.Contains(storedDigest, rawIdentifier) {
		t.Fatalf("stored key is not a 32-byte keyed digest: length=%d", len(storedDigest))
	}

	diagnostic := fmt.Sprintf("limiter=%+v key=%+v decision=%+v", limiter, key, overLimit)
	for _, private := range []string{string(secret), string(rawIdentifier)} {
		if strings.Contains(diagnostic, private) {
			t.Fatalf("formatted limiter value leaked private input %q: %s", private, diagnostic)
		}
	}

	now = time.Date(2026, 7, 19, 12, 14, 59, 0, time.UTC)
	nearlyExpired, err := limiter.Allow(ctx, key, 15*time.Minute, 2)
	if err != nil {
		t.Fatalf("nearly-expired Allow: %v", err)
	}
	assertRateLimitDecision(t, nearlyExpired, false, 0, time.Second)

	now = time.Date(2026, 7, 19, 12, 15, 0, 0, time.UTC)
	newWindow, err := limiter.Allow(ctx, key, 15*time.Minute, 2)
	if err != nil {
		t.Fatalf("new-window Allow: %v", err)
	}
	assertRateLimitDecision(t, newWindow, true, 1, 0)

	var bucketCount int
	if err := pool.QueryRow(ctx, `
		SELECT count(*)
		FROM instagram_rate_limit_buckets
		WHERE bucket_scope = $1
	`, string(RateLimitChallengeDID)).Scan(&bucketCount); err != nil {
		t.Fatalf("count buckets: %v", err)
	}
	if bucketCount != 2 {
		t.Fatalf("bucket rows after exact window boundary = %d, want 2", bucketCount)
	}
}

func TestPostgresRateLimiterExactScopesAndApprovedDefaults(t *testing.T) {
	pool := instagramLimiterTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	limiter, err := NewPostgresRateLimiter(
		pool,
		[]byte("synthetic-scope-hmac-key-32-bytes"),
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatalf("NewPostgresRateLimiter: %v", err)
	}

	tests := []struct {
		scope      RateLimitScope
		identifier []byte
		window     time.Duration
		limit      int
	}{
		{RateLimitChallengeDID, []byte("shared-synthetic-identifier"), 15 * time.Minute, 5},
		{RateLimitChallengeDevice, []byte("shared-synthetic-identifier"), 15 * time.Minute, 10},
		{RateLimitChallengeIP, []byte("shared-synthetic-identifier"), 15 * time.Minute, 30},
		{RateLimitInvalidRedemptionIGSID, []byte("shared-synthetic-identifier"), 15 * time.Minute, 10},
		{RateLimitInvalidRedemptionIP, []byte("shared-synthetic-identifier"), 15 * time.Minute, 30},
		{RateLimitConfirmationDID, []byte("shared-synthetic-identifier"), time.Hour, 20},
		{RateLimitConfirmationDevice, []byte("shared-synthetic-identifier"), time.Hour, 30},
		{RateLimitImportDID, []byte("shared-synthetic-identifier"), time.Hour, 10},
		{RateLimitImportDevice, []byte("shared-synthetic-identifier"), time.Hour, 20},
		{RateLimitWebhookGlobal, nil, time.Minute, 1000},
		{RateLimitWebhookIP, []byte("shared-synthetic-identifier"), time.Minute, 300},
		{RateLimitMetaLookupIGSID, []byte("shared-synthetic-identifier"), time.Hour, 5},
	}

	seenDigests := make(map[[32]byte]RateLimitScope, len(tests))
	for _, tt := range tests {
		t.Run(string(tt.scope), func(t *testing.T) {
			key, err := limiter.Key(tt.scope, tt.identifier)
			if err != nil {
				t.Fatalf("Key: %v", err)
			}
			if prior, exists := seenDigests[key.digest]; exists {
				t.Fatalf("scope %q reused digest from %q", tt.scope, prior)
			}
			seenDigests[key.digest] = tt.scope

			for request := 1; request <= tt.limit+1; request++ {
				decision, err := limiter.Allow(ctx, key, tt.window, tt.limit)
				if err != nil {
					t.Fatalf("Allow request %d: %v", request, err)
				}
				if request <= tt.limit {
					assertRateLimitDecision(t, decision, true, tt.limit-request, 0)
				} else {
					assertRateLimitDecision(t, decision, false, 0, tt.window)
				}
			}
		})
	}

	var bucketCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM instagram_rate_limit_buckets`).Scan(&bucketCount); err != nil {
		t.Fatalf("count buckets: %v", err)
	}
	if bucketCount != len(tests) {
		t.Fatalf("bucket rows = %d, want %d isolated scopes", bucketCount, len(tests))
	}
}

func TestPostgresRateLimiterConcurrentInstancesIncrementAtomically(t *testing.T) {
	pool := instagramLimiterTestPool(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	secret := []byte("synthetic-concurrent-hmac-key-32bytes")
	clock := func() time.Time { return now }

	first, err := NewPostgresRateLimiter(pool, secret, clock)
	if err != nil {
		t.Fatalf("first limiter: %v", err)
	}
	second, err := NewPostgresRateLimiter(pool, secret, clock)
	if err != nil {
		t.Fatalf("second limiter: %v", err)
	}
	firstKey, err := first.Key(RateLimitWebhookIP, []byte("192.0.2.10"))
	if err != nil {
		t.Fatalf("first key: %v", err)
	}
	secondKey, err := second.Key(RateLimitWebhookIP, []byte("192.0.2.10"))
	if err != nil {
		t.Fatalf("second key: %v", err)
	}
	if firstKey != secondKey {
		t.Fatal("same keyed identifier produced different cross-instance buckets")
	}

	const (
		requests = 100
		limit    = 25
	)
	start := make(chan struct{})
	errCh := make(chan error, requests)
	var (
		wg      sync.WaitGroup
		allowed atomic.Int64
		denied  atomic.Int64
	)
	for request := 0; request < requests; request++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			limiter := first
			key := firstKey
			if index%2 == 1 {
				limiter = second
				key = secondKey
			}
			decision, err := limiter.Allow(ctx, key, time.Minute, limit)
			if err != nil {
				errCh <- err
				return
			}
			if decision.Allowed {
				allowed.Add(1)
				return
			}
			if decision.RetryAfter != time.Minute || decision.Remaining != 0 {
				errCh <- fmt.Errorf("denied decision = %v", decision)
				return
			}
			denied.Add(1)
		}(request)
	}
	close(start)
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Errorf("concurrent Allow: %v", err)
	}
	if got := allowed.Load(); got != limit {
		t.Fatalf("allowed requests = %d, want exactly %d", got, limit)
	}
	if got := denied.Load(); got != requests-limit {
		t.Fatalf("denied requests = %d, want %d", got, requests-limit)
	}

	var storedCount int
	if err := pool.QueryRow(ctx, `
		SELECT count
		FROM instagram_rate_limit_buckets
		WHERE bucket_scope = $1
	`, string(RateLimitWebhookIP)).Scan(&storedCount); err != nil {
		t.Fatalf("inspect concurrent bucket: %v", err)
	}
	if storedCount != limit+1 {
		t.Fatalf("stored capped count = %d, want %d", storedCount, limit+1)
	}
}

func TestPostgresRateLimiterFailsClosedWithoutLeakingKeys(t *testing.T) {
	pool := instagramLimiterTestPool(t)
	now := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)
	secret := []byte("synthetic-private-limiter-key-32bytes")
	rawIdentifier := []byte("private-device-identifier-canary")

	if _, err := NewPostgresRateLimiter(nil, secret, func() time.Time { return now }); err == nil {
		t.Fatal("nil database accepted")
	}
	if _, err := NewPostgresRateLimiter(pool, []byte("short"), func() time.Time { return now }); err == nil {
		t.Fatal("short HMAC key accepted")
	}
	if _, err := NewPostgresRateLimiter(pool, secret, nil); err == nil {
		t.Fatal("nil clock accepted")
	}

	limiter, err := NewPostgresRateLimiter(pool, secret, func() time.Time { return now })
	if err != nil {
		t.Fatalf("NewPostgresRateLimiter: %v", err)
	}
	for _, input := range []struct {
		scope      RateLimitScope
		identifier []byte
	}{
		{RateLimitScope("unknown.private.scope"), rawIdentifier},
		{RateLimitChallengeDevice, nil},
		{RateLimitWebhookGlobal, rawIdentifier},
	} {
		if _, err := limiter.Key(input.scope, input.identifier); err == nil {
			t.Fatalf("Key(%q) accepted invalid scope/identifier shape", input.scope)
		} else if strings.Contains(err.Error(), string(rawIdentifier)) {
			t.Fatalf("Key error leaked identifier: %v", err)
		}
	}

	key, err := limiter.Key(RateLimitChallengeDevice, rawIdentifier)
	if err != nil {
		t.Fatalf("Key: %v", err)
	}
	otherKey, err := limiter.Key(RateLimitChallengeDevice, []byte("another-private-device"))
	if err != nil {
		t.Fatalf("other Key: %v", err)
	}
	if key.digest == otherKey.digest {
		t.Fatal("different identifiers produced the same keyed bucket")
	}
	for _, invalid := range []struct {
		key    RateLimitKey
		window time.Duration
		limit  int
	}{
		{RateLimitKey{}, time.Minute, 1},
		{key, 0, 1},
		{key, time.Minute, 0},
	} {
		if _, err := limiter.Allow(context.Background(), invalid.key, invalid.window, invalid.limit); err == nil {
			t.Fatalf("Allow accepted invalid key/window/limit: %+v", invalid)
		} else if strings.Contains(err.Error(), string(rawIdentifier)) || strings.Contains(err.Error(), string(secret)) {
			t.Fatalf("Allow error leaked private limiter input: %v", err)
		}
	}

	diagnostic := fmt.Sprintf("%v %+v %#v dereferenced=%+v", limiter, key, key, *limiter)
	for _, private := range []string{string(secret), string(rawIdentifier)} {
		if strings.Contains(diagnostic, private) {
			t.Fatalf("formatted limiter leaked %q: %s", private, diagnostic)
		}
	}
	if numericSecret := fmt.Sprint(secret); strings.Contains(diagnostic, numericSecret) {
		t.Fatalf("formatted limiter leaked HMAC key bytes: %s", diagnostic)
	}

	pool.Close()
	_, err = limiter.Allow(context.Background(), key, time.Minute, 1)
	if err == nil {
		t.Fatal("closed database Allow unexpectedly succeeded")
	}
	for _, private := range []string{string(secret), string(rawIdentifier)} {
		if strings.Contains(err.Error(), private) {
			t.Fatalf("database error leaked %q: %v", private, err)
		}
	}
}

func assertRateLimitDecision(t *testing.T, got RateLimitDecision, allowed bool, remaining int, retryAfter time.Duration) {
	t.Helper()
	if got.Allowed != allowed || got.Remaining != remaining || got.RetryAfter != retryAfter {
		t.Fatalf("decision = %+v, want allowed=%t remaining=%d retryAfter=%s", got, allowed, remaining, retryAfter)
	}
}

func instagramLimiterTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	migration, err := os.ReadFile("../../migrations/000023_instagram_migration.up.sql")
	if err != nil {
		t.Fatalf("read Instagram migration: %v", err)
	}
	return testdb.WithSchema(t, string(migration))
}
