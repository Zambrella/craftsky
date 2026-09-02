package auth_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atcrypto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
)

type testCleanupClock struct {
	mu sync.RWMutex
	at time.Time
}

func (clock *testCleanupClock) Now() time.Time {
	clock.mu.RLock()
	defer clock.mu.RUnlock()
	return clock.at
}

func (clock *testCleanupClock) Set(at time.Time) {
	clock.mu.Lock()
	clock.at = at
	clock.mu.Unlock()
}

type blockingCredentialRevoker struct {
	calls        atomic.Int32
	firstStarted chan struct{}
	releaseFirst chan error
	otherError   error
}

type cleanupRoundTripFunc func(*http.Request) (*http.Response, error)

func (roundTrip cleanupRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func TestIndigoOAuthCredentialRevokerUsesValidatedHardenedClientForBothTokens(t *testing.T) {
	privateKey, err := atcrypto.GeneratePrivateKeyP256()
	if err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	client := &http.Client{Transport: cleanupRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != "https://auth.example.com/oauth/revoke" {
			t.Fatalf("revocation destination=%s", request.URL)
		}
		calls.Add(1)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
			Request:    request,
		}, nil
	})}
	config := oauth.NewPublicConfig(
		"https://appview.example/oauth/client-metadata.json",
		"https://appview.example/oauth/callback",
		[]string{"atproto"},
	)
	store := auth.NewPostgresAuthStore(nil, auth.StoreConfig{EndpointValidator: testOAuthEndpointValidator{}})
	revoker, err := auth.NewIndigoOAuthCredentialRevoker(
		&oauth.ClientApp{Client: client, Config: &config}, store,
	)
	if err != nil {
		t.Fatal(err)
	}
	data := validOAuthSession("did:plc:cleanup-revoker", "revoker-parent")
	data.AccessToken = "access-token"
	data.RefreshToken = "refresh-token"
	data.DPoPPrivateKeyMultibase = privateKey.Multibase()
	if err := revoker.RevokeSession(context.Background(), data); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("revocation calls=%d, want access and refresh", calls.Load())
	}

	data.AuthServerRevocationEndpoint = "https://attacker.example/oauth/revoke"
	if err := revoker.RevokeSession(context.Background(), data); !errors.Is(err, auth.ErrOAuthSessionEndpointInvalid) {
		t.Fatalf("unvalidated endpoint error=%v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("unvalidated endpoint reached client, calls=%d", calls.Load())
	}
}

func (revoker *blockingCredentialRevoker) RevokeSession(context.Context, oauth.ClientSessionData) error {
	call := revoker.calls.Add(1)
	if call == 1 && revoker.firstStarted != nil {
		close(revoker.firstStarted)
		return <-revoker.releaseFirst
	}
	return revoker.otherError
}

// IT-008: the existing durable revocation worker also owns cleanup-pending
// ownerless registration credentials and converges them across restarts.
func TestProviderRegistrationCredentialCleanupConverges(t *testing.T) {
	pool := withAuthSchema(t)
	now := time.Now().UTC().Truncate(time.Second)
	clock := &testCleanupClock{at: now}
	store := auth.NewPostgresAuthStore(pool, testStoreConfig())
	registrationContext := auth.WithRegistrationAuthRequest(
		context.Background(), "https://pds.example.com", "https://auth.example.com",
		auth.HandoffVerifiedLink, "cleanup-registration-device", "",
	)
	reservation, err := store.ReserveAuthRequestCapacity(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	state := "registration-cleanup-worker"
	if err := store.SaveRegistrationAuthRequest(registrationContext, reservation.ID, oauth.AuthRequestData{
		State: state, RequestURI: "urn:request:" + state,
	}); err != nil {
		t.Fatal(err)
	}
	attemptID, err := store.BeginRegistrationExchange(context.Background(), state)
	if err != nil {
		t.Fatal(err)
	}
	session := validOAuthSession("did:plc:registration-cleanup", state)
	if err := store.QuarantineRegistrationCredential(context.Background(), state, attemptID, session, now); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkRegistrationCredentialForCleanup(context.Background(), state, attemptID); err != nil {
		t.Fatal(err)
	}
	futureState := "registration-cleanup-held"
	reservation, err = store.ReserveAuthRequestCapacity(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.SaveRegistrationAuthRequest(registrationContext, reservation.ID, oauth.AuthRequestData{
		State: futureState, RequestURI: "urn:request:" + futureState,
	}); err != nil {
		t.Fatal(err)
	}
	futureAttemptID, err := store.BeginRegistrationExchange(context.Background(), futureState)
	if err != nil {
		t.Fatal(err)
	}
	futureSession := validOAuthSession("did:plc:registration-cleanup-held", futureState)
	if err := store.QuarantineRegistrationCredential(
		context.Background(), futureState, futureAttemptID, futureSession, now.Add(time.Hour),
	); err != nil {
		t.Fatal(err)
	}

	revoker := &blockingCredentialRevoker{}
	processor := newOAuthRevocationProcessor(t, pool, revoker, clock.Now, 3, time.Minute)
	processed, err := processor.ProcessBatch(context.Background())
	if err != nil || processed != 1 {
		t.Fatalf("registration cleanup processed=%d err=%v", processed, err)
	}
	var requestState string
	var credentials int
	if err := pool.QueryRow(context.Background(), `
		SELECT request_state FROM oauth_auth_requests WHERE state=$1
	`, state).Scan(&requestState); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM oauth_unverified_credentials WHERE request_state=$1
	`, state).Scan(&credentials); err != nil {
		t.Fatal(err)
	}
	if requestState != string(auth.AuthRequestRevoked) || credentials != 0 || revoker.calls.Load() != 1 {
		t.Fatalf("registration cleanup state=%s credentials=%d revoke calls=%d", requestState, credentials, revoker.calls.Load())
	}
	var heldStatus, heldRequestState string
	if err := pool.QueryRow(context.Background(), `
		SELECT request.request_state,credential.status
		FROM oauth_auth_requests request
		JOIN oauth_unverified_credentials credential ON credential.request_state=request.state
		WHERE request.state=$1
	`, futureState).Scan(&heldRequestState, &heldStatus); err != nil {
		t.Fatal(err)
	}
	if heldRequestState != string(auth.AuthRequestExchangeStarted) || heldStatus != "held" {
		t.Fatalf("pre-eligibility cleanup changed held callback to %s/%s", heldRequestState, heldStatus)
	}

	clock.Set(now.Add(time.Hour))
	restarted := newOAuthRevocationProcessor(t, pool, revoker, clock.Now, 3, time.Minute)
	if processed, err := restarted.ProcessBatch(context.Background()); err != nil || processed != 1 {
		t.Fatalf("restarted held cleanup processed=%d err=%v", processed, err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT request_state FROM oauth_auth_requests WHERE state=$1
	`, futureState).Scan(&heldRequestState); err != nil {
		t.Fatal(err)
	}
	if heldRequestState != string(auth.AuthRequestRevoked) || revoker.calls.Load() != 2 {
		t.Fatalf("eligible held cleanup state=%s revoke calls=%d", heldRequestState, revoker.calls.Load())
	}
}

func TestOAuthRevocationProcessorReclaimsExpiredLeaseAndFencesStaleWorker(t *testing.T) {
	pool := withAuthSchema(t)
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	clock := &testCleanupClock{at: now}
	seedRevocationPendingSession(t, pool, "did:plc:revocation-reclaim", "session-one", now)
	revoker := &blockingCredentialRevoker{
		firstStarted: make(chan struct{}),
		releaseFirst: make(chan error, 1),
	}
	first := newOAuthRevocationProcessor(t, pool, revoker, clock.Now, 2, time.Minute)
	firstResult := make(chan error, 1)
	go func() {
		_, err := first.ProcessBatch(context.Background())
		firstResult <- err
	}()
	<-revoker.firstStarted

	// A new process instance starts after the durable lease expires. Its fresh
	// token owns completion; the original worker may no longer mutate the row.
	clock.Set(now.Add(2 * time.Minute))
	second := newOAuthRevocationProcessor(t, pool, revoker, clock.Now, 2, time.Minute)
	processed, err := second.ProcessBatch(context.Background())
	if err != nil || processed != 1 {
		t.Fatalf("restart processor = %d, %v", processed, err)
	}
	revoker.releaseFirst <- errors.New("stale worker failed after reclaim")
	if err := <-firstResult; err != nil {
		t.Fatalf("stale processor: %v", err)
	}
	var rows int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM oauth_sessions
		WHERE account_did='did:plc:revocation-reclaim' AND session_id='session-one'
	`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 0 || revoker.calls.Load() != 2 {
		t.Fatalf("retained rows=%d revoke calls=%d, want 0 rows/2 idempotent calls", rows, revoker.calls.Load())
	}
}

func TestOAuthRevocationProcessorRetriesThenDeletesAtAttemptBound(t *testing.T) {
	pool := withAuthSchema(t)
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	clock := &testCleanupClock{at: now}
	seedRevocationPendingSession(t, pool, "did:plc:revocation-retry", "session-one", now)
	revoker := &blockingCredentialRevoker{otherError: errors.New("authorization server unavailable")}
	processor := newOAuthRevocationProcessor(t, pool, revoker, clock.Now, 2, time.Minute)

	if processed, err := processor.ProcessBatch(context.Background()); err != nil || processed != 1 {
		t.Fatalf("first attempt = %d, %v", processed, err)
	}
	var attempts int
	var next time.Time
	var lease *uuid.UUID
	if err := pool.QueryRow(context.Background(), `
		SELECT cleanup_attempts,cleanup_next_attempt_at,cleanup_lease_token
		FROM oauth_sessions WHERE account_did='did:plc:revocation-retry'
	`).Scan(&attempts, &next, &lease); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 || !next.Equal(now.Add(time.Minute)) || lease != nil {
		t.Fatalf("retry state attempts=%d next=%s lease=%v", attempts, next, lease)
	}
	if processed, err := processor.ProcessBatch(context.Background()); err != nil || processed != 0 {
		t.Fatalf("early retry = %d, %v", processed, err)
	}
	clock.Set(now.Add(time.Minute))
	if processed, err := processor.ProcessBatch(context.Background()); err != nil || processed != 1 {
		t.Fatalf("bounded attempt = %d, %v", processed, err)
	}
	var rows int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM oauth_sessions WHERE account_did='did:plc:revocation-retry'`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 0 || revoker.calls.Load() != 2 {
		t.Fatalf("rows=%d calls=%d, want local deletion after 2 attempts", rows, revoker.calls.Load())
	}
}

func TestOAuthRevocationProcessorEnforcesCredentialRetentionWithoutNetwork(t *testing.T) {
	pool := withAuthSchema(t)
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	clock := &testCleanupClock{at: now}
	seedRevocationPendingSession(t, pool, "did:plc:revocation-retention", "session-one", now.Add(-25*time.Hour))
	revoker := &blockingCredentialRevoker{}
	processor := newOAuthRevocationProcessor(t, pool, revoker, clock.Now, 3, time.Minute)
	if processed, err := processor.ProcessBatch(context.Background()); err != nil || processed != 1 {
		t.Fatalf("retention batch = %d, %v", processed, err)
	}
	var rows int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM oauth_sessions WHERE account_did='did:plc:revocation-retention'`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 0 || revoker.calls.Load() != 0 {
		t.Fatalf("rows=%d calls=%d, want bounded local discard without network", rows, revoker.calls.Load())
	}
}

func TestCleanupProcessorsRespectBatchBoundsAcrossRestart(t *testing.T) {
	pool := withAuthSchema(t)
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	clock := &testCleanupClock{at: now}
	for index, owner := range []string{"did:plc:bounded-revocation-one", "did:plc:bounded-revocation-two"} {
		seedRevocationPendingSession(t, pool, owner, fmt.Sprintf("parent-%d", index), now)
	}
	revoker := &blockingCredentialRevoker{}
	for run := 0; run < 2; run++ {
		processor := newOAuthRevocationProcessorWithBatch(t, pool, revoker, clock.Now, 3, time.Minute, 1)
		if processed, err := processor.ProcessBatch(context.Background()); err != nil || processed != 1 {
			t.Fatalf("revocation restart %d = %d, %v", run, processed, err)
		}
	}
	if processed, err := newOAuthRevocationProcessorWithBatch(
		t, pool, revoker, clock.Now, 3, time.Minute, 1,
	).ProcessBatch(context.Background()); err != nil || processed != 0 {
		t.Fatalf("revocation drained = %d, %v", processed, err)
	}

	owner := syntax.DID("did:plc:bounded-auxiliary")
	seedActiveAuthOwner(t, pool, owner)
	seedAuxiliaryCleanupJob(t, pool, owner, 1, "installation_push", "device-one", now)
	seedAuxiliaryCleanupJob(t, pool, owner, 1, "installation_push", "device-two", now)
	cleaner := &blockingNotificationCleaner{}
	for run := 0; run < 2; run++ {
		processor := newAuxiliaryCleanupProcessorWithBatch(t, pool, cleaner, clock.Now, 3, 1)
		if processed, err := processor.ProcessBatch(context.Background()); err != nil || processed != 1 {
			t.Fatalf("auxiliary restart %d = %d, %v", run, processed, err)
		}
	}
	if processed, err := newAuxiliaryCleanupProcessorWithBatch(
		t, pool, cleaner, clock.Now, 3, 1,
	).ProcessBatch(context.Background()); err != nil || processed != 0 {
		t.Fatalf("auxiliary drained = %d, %v", processed, err)
	}
}

func newOAuthRevocationProcessor(
	t *testing.T,
	pool *pgxpool.Pool,
	revoker auth.OAuthCredentialRevoker,
	now func() time.Time,
	maxAttempts int,
	baseBackoff time.Duration,
) *auth.OAuthRevocationProcessor {
	return newOAuthRevocationProcessorWithBatch(t, pool, revoker, now, maxAttempts, baseBackoff, 5)
}

func newOAuthRevocationProcessorWithBatch(
	t *testing.T,
	pool *pgxpool.Pool,
	revoker auth.OAuthCredentialRevoker,
	now func() time.Time,
	maxAttempts int,
	baseBackoff time.Duration,
	batch int,
) *auth.OAuthRevocationProcessor {
	t.Helper()
	processor, err := auth.NewOAuthRevocationProcessor(auth.OAuthRevocationProcessorOptions{
		Pool: pool, Revoker: revoker, Now: now, NewLeaseToken: uuid.New,
		BatchSize: batch, LeaseDuration: time.Minute, OperationTimeout: 30 * time.Second,
		MaxAttempts: maxAttempts, BaseBackoff: baseBackoff, MaxBackoff: 10 * time.Minute,
		MaxCredentialRetention: 24 * time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	return processor
}

func seedRevocationPendingSession(t *testing.T, pool *pgxpool.Pool, owner, sessionID string, requestedAt time.Time) {
	t.Helper()
	seedActiveOAuthSession(t, pool, owner, sessionID)
	if _, err := pool.Exec(context.Background(), `
		UPDATE oauth_sessions
		SET lifecycle_state='revocation_pending',revocation_requested_at=$3,
		    cleanup_next_attempt_at=$3,updated_at=$3
		WHERE account_did=$1 AND session_id=$2
	`, owner, sessionID, requestedAt); err != nil {
		t.Fatal(err)
	}
}

type blockingNotificationCleaner struct {
	calls        atomic.Int32
	firstStarted chan struct{}
	releaseFirst chan error
	otherError   error
	mu           sync.Mutex
	requests     []string
	cutoffs      []time.Time
}

func (cleaner *blockingNotificationCleaner) DeactivateForInstallationBefore(
	_ context.Context,
	owner, installation string,
	cutoff time.Time,
) error {
	cleaner.mu.Lock()
	cleaner.requests = append(cleaner.requests, "installation:"+owner+":"+installation)
	cleaner.cutoffs = append(cleaner.cutoffs, cutoff)
	cleaner.mu.Unlock()
	return cleaner.result()
}

func (cleaner *blockingNotificationCleaner) DeactivateForAccountBefore(
	_ context.Context,
	owner string,
	cutoff time.Time,
) error {
	cleaner.mu.Lock()
	cleaner.requests = append(cleaner.requests, "account:"+owner)
	cleaner.cutoffs = append(cleaner.cutoffs, cutoff)
	cleaner.mu.Unlock()
	return cleaner.result()
}

func (cleaner *blockingNotificationCleaner) result() error {
	call := cleaner.calls.Add(1)
	if call == 1 && cleaner.firstStarted != nil {
		close(cleaner.firstStarted)
		return <-cleaner.releaseFirst
	}
	return cleaner.otherError
}

func TestAuxiliaryCleanupProcessorReclaimsLeaseAndFencesStaleWorker(t *testing.T) {
	pool := withAuthSchema(t)
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	clock := &testCleanupClock{at: now}
	owner := syntax.DID("did:plc:aux-reclaim")
	seedActiveAuthOwner(t, pool, owner)
	jobID := seedAuxiliaryCleanupJob(t, pool, owner, 1, "installation_push", "device-one", now)
	cleaner := &blockingNotificationCleaner{
		firstStarted: make(chan struct{}), releaseFirst: make(chan error, 1),
	}
	first := newAuxiliaryCleanupProcessor(t, pool, cleaner, clock.Now, 3)
	firstResult := make(chan error, 1)
	go func() {
		_, err := first.ProcessBatch(context.Background())
		firstResult <- err
	}()
	<-cleaner.firstStarted
	clock.Set(now.Add(2 * time.Minute))
	second := newAuxiliaryCleanupProcessor(t, pool, cleaner, clock.Now, 3)
	if processed, err := second.ProcessBatch(context.Background()); err != nil || processed != 1 {
		t.Fatalf("restart cleanup = %d, %v", processed, err)
	}
	cleaner.releaseFirst <- errors.New("stale cleanup failure")
	if err := <-firstResult; err != nil {
		t.Fatalf("stale cleanup processor: %v", err)
	}
	var state string
	var attempts int
	if err := pool.QueryRow(context.Background(), `
		SELECT state,attempt_count FROM auth_auxiliary_cleanup_jobs WHERE id=$1
	`, jobID).Scan(&state, &attempts); err != nil {
		t.Fatal(err)
	}
	if state != "complete" || attempts != 0 || cleaner.calls.Load() != 2 {
		t.Fatalf("state=%s attempts=%d calls=%d", state, attempts, cleaner.calls.Load())
	}
	cleaner.mu.Lock()
	defer cleaner.mu.Unlock()
	for _, cutoff := range cleaner.cutoffs {
		if !cutoff.Equal(now) {
			t.Fatalf("cleanup cutoff=%s, want durable job creation %s", cutoff, now)
		}
	}
}

func TestAuxiliaryCleanupProcessorRetriesAndExhaustsSafely(t *testing.T) {
	pool := withAuthSchema(t)
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	clock := &testCleanupClock{at: now}
	owner := syntax.DID("did:plc:aux-exhaust")
	seedActiveAuthOwner(t, pool, owner)
	jobID := seedAuxiliaryCleanupJob(t, pool, owner, 1, "account_push", "", now)
	cleaner := &blockingNotificationCleaner{otherError: errors.New("push store unavailable")}
	processor := newAuxiliaryCleanupProcessor(t, pool, cleaner, clock.Now, 1)
	if processed, err := processor.ProcessBatch(context.Background()); err != nil || processed != 1 {
		t.Fatalf("cleanup failure = %d, %v", processed, err)
	}
	var state, category string
	var attempts int
	var lease *uuid.UUID
	if err := pool.QueryRow(context.Background(), `
		SELECT state,attempt_count,last_category,lease_token
		FROM auth_auxiliary_cleanup_jobs WHERE id=$1
	`, jobID).Scan(&state, &attempts, &category, &lease); err != nil {
		t.Fatal(err)
	}
	if state != "exhausted" || attempts != 1 || category != "dependency_unavailable" || lease != nil {
		t.Fatalf("state=%s attempts=%d category=%s lease=%v", state, attempts, category, lease)
	}
}

func newAuxiliaryCleanupProcessor(
	t *testing.T,
	pool *pgxpool.Pool,
	cleaner auth.FencedNotificationSubscriptionCleaner,
	now func() time.Time,
	maxAttempts int,
) *auth.AuxiliaryCleanupProcessor {
	return newAuxiliaryCleanupProcessorWithBatch(t, pool, cleaner, now, maxAttempts, 5)
}

func newAuxiliaryCleanupProcessorWithBatch(
	t *testing.T,
	pool *pgxpool.Pool,
	cleaner auth.FencedNotificationSubscriptionCleaner,
	now func() time.Time,
	maxAttempts int,
	batch int,
) *auth.AuxiliaryCleanupProcessor {
	t.Helper()
	processor, err := auth.NewAuxiliaryCleanupProcessor(auth.AuxiliaryCleanupProcessorOptions{
		Pool: pool, Cleaner: cleaner, Now: now, NewLeaseToken: uuid.New,
		BatchSize: batch, LeaseDuration: time.Minute, OperationTimeout: 30 * time.Second,
		MaxAttempts: maxAttempts, BaseBackoff: time.Minute, MaxBackoff: 10 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	return processor
}

func seedAuxiliaryCleanupJob(
	t *testing.T,
	pool *pgxpool.Pool,
	owner syntax.DID,
	authEpoch int64,
	kind string,
	installationID string,
	now time.Time,
) uuid.UUID {
	t.Helper()
	id := uuid.New()
	var installation any
	if installationID != "" {
		installation = installationID
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO auth_auxiliary_cleanup_jobs(
			id,owner_did,auth_epoch,kind,installation_id,state,next_attempt_at,created_at,updated_at
		) VALUES($1,$2,$3,$4,$5,'pending',$6,$6,$6)
	`, id, owner, authEpoch, kind, installation, now); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestSessionExpiryProcessorIsBoundedAndRestartSafe(t *testing.T) {
	pool := withAuthSchema(t)
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	clock := &testCleanupClock{at: now}
	seedExpiredParentFamily(t, pool, "did:plc:expired-parent-worker", "expired-parent", now)
	seedExpiredChildFamily(t, pool, "did:plc:expired-child-worker", "active-parent", now)
	processor := newSessionExpiryProcessor(t, pool, clock.Now, 1)
	if processed, err := processor.ProcessBatch(context.Background()); err != nil || processed != 1 {
		t.Fatalf("first bounded expiry = %d, %v", processed, err)
	}
	restarted := newSessionExpiryProcessor(t, pool, clock.Now, 1)
	if processed, err := restarted.ProcessBatch(context.Background()); err != nil || processed != 1 {
		t.Fatalf("restart bounded expiry = %d, %v", processed, err)
	}
	if processed, err := restarted.ProcessBatch(context.Background()); err != nil || processed != 0 {
		t.Fatalf("drained expiry = %d, %v", processed, err)
	}

	var parentState, parentChildState, childParentState, childState string
	if err := pool.QueryRow(context.Background(), `
		SELECT lifecycle_state FROM oauth_sessions
		WHERE account_did='did:plc:expired-parent-worker'
	`).Scan(&parentState); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT lifecycle_state FROM craftsky_sessions
		WHERE account_did='did:plc:expired-parent-worker'
	`).Scan(&parentChildState); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT lifecycle_state FROM oauth_sessions
		WHERE account_did='did:plc:expired-child-worker'
	`).Scan(&childParentState); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT lifecycle_state FROM craftsky_sessions
		WHERE account_did='did:plc:expired-child-worker'
	`).Scan(&childState); err != nil {
		t.Fatal(err)
	}
	if parentState != "revocation_pending" || parentChildState != "revoked" ||
		childParentState != "active" || childState != "revoked" {
		t.Fatalf("states parent=%s/%s child=%s/%s", parentState, parentChildState, childParentState, childState)
	}
}

func TestSessionExpiryProcessorExpiresPendingHandoffSecrets(t *testing.T) {
	pool := withAuthSchema(t)
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	clock := &testCleanupClock{at: now}
	owner := syntax.DID("did:plc:expired-handoff-worker")
	seedExpiredPendingHandoff(t, pool, owner, "pending-parent", now)
	processor := newSessionExpiryProcessor(t, pool, clock.Now, 5)
	if processed, err := processor.ProcessBatch(context.Background()); err != nil || processed != 1 {
		t.Fatalf("pending expiry = %d, %v", processed, err)
	}
	var parentState, childState, exchangeState, receiptState, requestState string
	var codeHash, ciphertext, nonce []byte
	if err := pool.QueryRow(context.Background(), `SELECT lifecycle_state FROM oauth_sessions WHERE account_did=$1`, owner).Scan(&parentState); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT lifecycle_state FROM craftsky_sessions WHERE account_did=$1`, owner).Scan(&childState); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT state,code_hash FROM oauth_handoff_exchanges WHERE owner_did=$1`, owner).Scan(&exchangeState, &codeHash); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT state,ciphertext,nonce FROM oauth_handoff_receipts`).Scan(&receiptState, &ciphertext, &nonce); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT request_state FROM oauth_auth_requests WHERE owner_did=$1`, owner).Scan(&requestState); err != nil {
		t.Fatal(err)
	}
	if parentState != "revocation_pending" || childState != "revoked" || exchangeState != "expired" ||
		receiptState != "expired" || requestState != "revoked" || codeHash != nil || ciphertext != nil || nonce != nil {
		t.Fatalf("expired handoff parent=%s child=%s exchange=%s receipt=%s request=%s secrets=%v/%v/%v",
			parentState, childState, exchangeState, receiptState, requestState, codeHash, ciphertext, nonce)
	}
}

func TestSessionExpiryProcessorDefersUnacceptedDeletionCredentialToIntentExpiry(t *testing.T) {
	pool := withAuthSchema(t)
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:unaccepted-expiry-owner")
	operationID := uuid.MustParse("10000000-0000-4000-8000-000000000101")
	seedDeletionExpiryOwner(t, pool, owner, "deletion_pending", 2, 1, now)
	seedDeletionOnlyExpiryParent(t, pool, owner, operationID, "intent-parent", 2, 1, now)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO account_deletion_operations(
			id,owner_did,owner_generation,state,reauth_oauth_session_id,
			deletion_credential_generation,intent_expires_at,updated_at
		) VALUES($1,$2,2,'intent','intent-parent',1,
		         $3::timestamptz+interval '1 hour',$3)
	`, operationID, owner, now); err != nil {
		t.Fatal(err)
	}
	processor := newSessionExpiryProcessor(t, pool, func() time.Time { return now }, 5)
	if processed, err := processor.ProcessBatch(context.Background()); err != nil || processed != 0 {
		t.Fatalf("unaccepted credential expiry=%d, %v", processed, err)
	}
	var parentState, operationState string
	if err := pool.QueryRow(context.Background(), `
		SELECT lifecycle_state FROM oauth_sessions WHERE account_did=$1 AND session_id='intent-parent'
	`, owner).Scan(&parentState); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT state FROM account_deletion_operations WHERE id=$1
	`, operationID).Scan(&operationState); err != nil {
		t.Fatal(err)
	}
	if parentState != "deletion_only" || operationState != "intent" {
		t.Fatalf("unaccepted credential states=%s/%s", parentState, operationState)
	}
}

func TestSessionExpiryProcessorExpiresAcceptedDeletionCredentialToReauthRequired(t *testing.T) {
	pool := withAuthSchema(t)
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:accepted-expiry-owner")
	operationID := uuid.MustParse("10000000-0000-4000-8000-000000000102")
	seedDeletionExpiryOwner(t, pool, owner, "deleting", 3, 2, now)
	seedDeletionOnlyExpiryParent(t, pool, owner, operationID, "accepted-parent", 2, 2, now)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO account_deletion_operations(
			id,owner_did,owner_generation,state,accepted_at,deletion_oauth_session_id,
			deletion_credential_generation,next_attempt_at,updated_at
		) VALUES($1,$2,3,'active',$3,'accepted-parent',1,$3,$3)
	`, operationID, owner, now); err != nil {
		t.Fatal(err)
	}
	processor := newSessionExpiryProcessor(t, pool, func() time.Time { return now }, 5)
	if processed, err := processor.ProcessBatch(context.Background()); err != nil || processed != 1 {
		t.Fatalf("accepted credential expiry=%d, %v", processed, err)
	}
	var parentState, operationState string
	var boundSession *string
	if err := pool.QueryRow(context.Background(), `
		SELECT lifecycle_state FROM oauth_sessions WHERE account_did=$1 AND session_id='accepted-parent'
	`, owner).Scan(&parentState); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `
		SELECT state,deletion_oauth_session_id FROM account_deletion_operations WHERE id=$1
	`, operationID).Scan(&operationState, &boundSession); err != nil {
		t.Fatal(err)
	}
	if parentState != "revocation_pending" || operationState != "reauth_required" || boundSession != nil {
		t.Fatalf("accepted credential states=%s/%s bound=%v", parentState, operationState, boundSession)
	}
}

func newSessionExpiryProcessor(
	t *testing.T,
	pool *pgxpool.Pool,
	now func() time.Time,
	batch int,
) *auth.SessionExpiryProcessor {
	t.Helper()
	children := auth.NewCraftskySessionStore(pool, time.Minute)
	lifecycle, err := auth.NewSessionLifecycleService(auth.SessionLifecycleOptions{
		Pool: pool, Owners: newAuthOwnerStore(t, pool), Sessions: children, Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	processor, err := auth.NewSessionExpiryProcessor(auth.SessionExpiryProcessorOptions{
		Lifecycle: lifecycle, BatchSize: batch,
	})
	if err != nil {
		t.Fatal(err)
	}
	return processor
}

func seedExpiredParentFamily(t *testing.T, pool *pgxpool.Pool, owner, sessionID string, now time.Time) {
	t.Helper()
	seedActiveOAuthSession(t, pool, owner, sessionID)
	store := auth.NewCraftskySessionStore(pool, time.Minute)
	if _, err := store.Create(context.Background(), owner, sessionID, "device-parent"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE oauth_sessions
		SET created_at=$3::timestamptz-interval '2 hours',
		    updated_at=$3::timestamptz-interval '2 hours',
		    absolute_expires_at=$3::timestamptz-interval '1 hour'
		WHERE account_did=$1 AND session_id=$2
	`, owner, sessionID, now); err != nil {
		t.Fatal(err)
	}
}

func seedExpiredChildFamily(t *testing.T, pool *pgxpool.Pool, owner, sessionID string, now time.Time) {
	t.Helper()
	seedActiveOAuthSession(t, pool, owner, sessionID)
	store := auth.NewCraftskySessionStore(pool, time.Minute)
	if _, err := store.Create(context.Background(), owner, sessionID, "device-child"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		UPDATE craftsky_sessions
		SET created_at=$2::timestamptz-interval '2 hours',
		    last_seen_at=$2::timestamptz-interval '2 hours',
		    idle_expires_at=$2::timestamptz-interval '1 hour'
		WHERE account_did=$1
	`, owner, now); err != nil {
		t.Fatalf("expire child: %v", err)
	}
}

func seedExpiredPendingHandoff(t *testing.T, pool *pgxpool.Pool, owner syntax.DID, sessionID string, now time.Time) {
	t.Helper()
	seedActiveStoredOAuthSession(t, pool, validOAuthSession(owner, sessionID))
	if _, err := pool.Exec(context.Background(), `
		UPDATE oauth_sessions SET lifecycle_state='pending_handoff' WHERE account_did=$1 AND session_id=$2
	`, owner, sessionID); err != nil {
		t.Fatal(err)
	}
	childHash := []byte("pending-child-hash-000000000000000")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO craftsky_sessions(
			token_hash,account_did,oauth_session_id,lifecycle_state,auth_epoch,
			created_at,last_seen_at,idle_expires_at
		) VALUES($1,$2,$3,'pending_confirmation',1,
		         $4::timestamptz-interval '2 hours',
		         $4::timestamptz-interval '2 hours',
		         $4::timestamptz+interval '1 hour')
	`, childHash, owner, sessionID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO oauth_auth_requests(
			state,data,handoff_mode,device_id,purpose,owner_did,owner_generation,
			auth_epoch,request_uri,request_state,consumed_at
		) VALUES($1,'{}','verified_link','device-handoff','login',$2,1,1,
		         'urn:request:pending-worker','consumed',
		         $3::timestamptz-interval '2 hours')
	`, sessionID, owner, now); err != nil {
		t.Fatal(err)
	}
	exchangeID := uuid.New()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO oauth_handoff_exchanges(
			id,code_hash,owner_did,owner_generation,auth_epoch,oauth_session_id,
			device_id,canonical_handle,state,expires_at,created_at,updated_at
		) VALUES($1,$2,$3,1,1,$4,'device-handoff','alice.test','redeemed',
		         $5::timestamptz-interval '1 hour',
		         $5::timestamptz-interval '2 hours',
		         $5::timestamptz-interval '2 hours')
	`, exchangeID, make([]byte, 32), owner, sessionID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO oauth_handoff_receipts(
			id,exchange_id,child_token_hash,ciphertext,nonce,key_version,state,
			confirm_by,created_at,updated_at
		) VALUES($1,$2,$3,$4,$5,1,'pending',
		         $6::timestamptz-interval '1 hour',
		         $6::timestamptz-interval '2 hours',
		         $6::timestamptz-interval '2 hours')
	`, uuid.New(), exchangeID, childHash, []byte("sealed"), []byte("nonce"), now); err != nil {
		t.Fatal(err)
	}
}

func seedDeletionExpiryOwner(
	t *testing.T,
	pool *pgxpool.Pool,
	owner syntax.DID,
	state string,
	generation int64,
	authEpoch int64,
	now time.Time,
) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,$2,$3,$4,'test',$5,$5,$5)
	`, owner, state, generation, authEpoch, now); err != nil {
		t.Fatal(err)
	}
}

func seedDeletionOnlyExpiryParent(
	t *testing.T,
	pool *pgxpool.Pool,
	owner syntax.DID,
	operationID uuid.UUID,
	sessionID string,
	ownerGeneration int64,
	authEpoch int64,
	now time.Time,
) {
	t.Helper()
	data := validOAuthSession(owner, sessionID)
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO oauth_sessions(
			account_did,session_id,data,lifecycle_state,owner_generation,auth_epoch,
			absolute_expires_at,deletion_operation_id,deletion_credential_generation,
			created_at,updated_at
		) VALUES($1,$2,$3,'deletion_only',$4,$5,
		         $7::timestamptz-interval '1 hour',$6,1,
		         $7::timestamptz-interval '2 hours',$7::timestamptz-interval '2 hours')
	`, owner, sessionID, data, ownerGeneration, authEpoch, operationID, now); err != nil {
		t.Fatal(err)
	}
}
