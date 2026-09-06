package ingestion_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/tap"
)

type recordedIdentityInvalidation struct {
	did     syntax.DID
	handles []syntax.Handle
}

type recordingIdentityInvalidator struct {
	calls []recordedIdentityInvalidation
}

type blockingIdentityRefreshResolver struct {
	started chan struct{}
	release chan struct{}
	did     syntax.DID
	handle  syntax.Handle
	err     error
}

func (resolver *blockingIdentityRefreshResolver) ResolveHandle(ctx context.Context, _ syntax.DID) (syntax.Handle, error) {
	close(resolver.started)
	select {
	case <-resolver.release:
		return resolver.handle, resolver.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (resolver *blockingIdentityRefreshResolver) ResolveDID(context.Context, syntax.Handle) (syntax.DID, error) {
	return resolver.did, nil
}

type staticIdentityRefreshResolver struct {
	did    syntax.DID
	handle syntax.Handle
}

func (resolver staticIdentityRefreshResolver) ResolveHandle(context.Context, syntax.DID) (syntax.Handle, error) {
	return resolver.handle, nil
}

func (resolver staticIdentityRefreshResolver) ResolveDID(context.Context, syntax.Handle) (syntax.DID, error) {
	return resolver.did, nil
}

func (invalidator *recordingIdentityInvalidator) InvalidateIdentity(_ context.Context, did syntax.DID, handles ...syntax.Handle) {
	invalidator.calls = append(invalidator.calls, recordedIdentityInvalidation{did: did, handles: append([]syntax.Handle(nil), handles...)})
}

func TestOrdinaryIdentityReceiptAndImmediateRefreshTriggerCommitAtomicallyAndRedeliverIdempotently(t *testing.T) {
	now := time.Date(2026, 8, 20, 17, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:identity-refresh-owner")
	event := tap.IdentityEvent{
		ID: 700, DID: owner, Handle: "new.example", IsActive: true, Status: "active",
	}

	t.Run("refresh scheduling failure rolls back receipt", func(t *testing.T) {
		pool := identityRefreshTriggerPool(t)
		if _, err := pool.Exec(context.Background(), `DROP TABLE atproto_identity_refresh_state`); err != nil {
			t.Fatal(err)
		}
		service := identityRefreshTriggerService(t, pool, now)

		if _, err := service.IngestIdentity(context.Background(), event); err == nil {
			t.Fatal("identity ingestion succeeded without durable refresh state")
		}
		assertIdentityReceiptCount(t, pool, 0)
	})

	t.Run("same event cannot reset retry while a newer event schedules immediately", func(t *testing.T) {
		pool := identityRefreshTriggerPool(t)
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO atproto_identity_cache(did,handle,handle_lower,resolved_at)
			VALUES($1,'old.example','old.example',$2)
		`, owner, now.Add(-time.Hour)); err != nil {
			t.Fatal(err)
		}
		invalidator := &recordingIdentityInvalidator{}
		service := identityRefreshTriggerService(t, pool, now, invalidator)
		ctx := context.Background()

		outcome, err := service.IngestIdentity(ctx, event)
		if err != nil || outcome.Kind != tap.OutcomeApplied {
			t.Fatalf("ingest identity outcome=%+v err=%v", outcome, err)
		}
		assertIdentityRefreshState(t, pool, owner, 700, "pending", 0, now)
		assertIdentityReceiptCount(t, pool, 1)
		var indexedHandle syntax.Handle
		if err := pool.QueryRow(ctx, `SELECT handle FROM atproto_identity_cache WHERE did=$1`, owner).Scan(&indexedHandle); err != nil || indexedHandle != "old.example" {
			t.Fatalf("Tap hint changed verified index handle=%s err=%v", indexedHandle, err)
		}
		if len(invalidator.calls) != 1 || invalidator.calls[0].did != owner ||
			!containsHandle(invalidator.calls[0].handles, "old.example") ||
			!containsHandle(invalidator.calls[0].handles, "new.example") {
			t.Fatalf("identity invalidations=%+v, want owner with old and event handles", invalidator.calls)
		}

		retryAt := now.Add(15 * time.Minute)
		if _, err := pool.Exec(ctx, `
			UPDATE atproto_identity_refresh_state
			SET next_attempt_at=$2,attempt_count=1,last_result='retry',updated_at=$1
			WHERE did=$3
		`, now, retryAt, owner); err != nil {
			t.Fatal(err)
		}
		if outcome, err := service.IngestIdentity(ctx, event); err != nil || outcome.Kind != tap.OutcomeApplied {
			t.Fatalf("redeliver identity outcome=%+v err=%v", outcome, err)
		}
		assertIdentityRefreshState(t, pool, owner, 700, "retry", 1, retryAt)
		assertIdentityReceiptCount(t, pool, 1)
		if len(invalidator.calls) != 1 {
			t.Fatalf("duplicate identity invalidations=%d, want one", len(invalidator.calls))
		}

		newer := event
		newer.ID = 701
		newer.Handle = "newer.example"
		if outcome, err := service.IngestIdentity(ctx, newer); err != nil || outcome.Kind != tap.OutcomeApplied {
			t.Fatalf("new identity outcome=%+v err=%v", outcome, err)
		}
		assertIdentityRefreshState(t, pool, owner, 701, "pending", 0, now)
		assertIdentityReceiptCount(t, pool, 2)

		reset := newer
		reset.ID = 1
		reset.Handle = "after-reset.example"
		if outcome, err := service.IngestIdentity(ctx, reset); err != nil || outcome.Kind != tap.OutcomeApplied {
			t.Fatalf("identity after Tap reset outcome=%+v err=%v", outcome, err)
		}
		assertIdentityRefreshState(t, pool, owner, 1, "pending", 0, now)
		assertIdentityReceiptCount(t, pool, 3)
	})
}

func TestAllTapIdentityStatusesCoalesceAcrossRestartWithoutChangingPresentation(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 3, 10, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:identity-status-hints")
	pool := identityRefreshTriggerPool(t)
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'active',7,3,'test',$2,$2,$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'profile-cid')
	`, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_identity_cache(did,handle,handle_lower,resolved_at)
		VALUES($1,'verified.example','verified.example',$2)
	`, owner, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	statuses := []string{"active", "deactivated", "suspended", "takendown", "deleted"}
	service := identityRefreshTriggerService(t, pool, now)
	for index, status := range statuses {
		event := tap.IdentityEvent{
			ID: uint64(800 + index), DID: owner, Handle: status + ".example",
			IsActive: status == "active", Status: status,
		}
		if outcome, err := service.IngestIdentity(ctx, event); err != nil || outcome.Kind != tap.OutcomeApplied {
			t.Fatalf("ingest %s identity outcome=%+v err=%v", status, outcome, err)
		}
	}

	// Recreate the service to model an AppView restart, then redeliver a
	// duplicate and an older hint. Tap IDs are local to Tap and may reset, so
	// they coalesce work but never establish identity authority.
	service = identityRefreshTriggerService(t, pool, now.Add(time.Second))
	duplicate := tap.IdentityEvent{ID: 804, DID: owner, Handle: "deleted.example", Status: "deleted"}
	if outcome, err := service.IngestIdentity(ctx, duplicate); err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("redeliver duplicate identity outcome=%+v err=%v", outcome, err)
	}
	outOfOrder := tap.IdentityEvent{ID: 12, DID: owner, Handle: "older.example", Status: "deactivated"}
	if outcome, err := service.IngestIdentity(ctx, outOfOrder); err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("ingest out-of-order identity outcome=%+v err=%v", outcome, err)
	}

	assertIdentityReceiptCount(t, pool, len(statuses)+1)
	assertIdentityRefreshState(t, pool, owner, 12, "pending", 0, now.Add(time.Second))
	var refreshRows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM atproto_identity_refresh_state WHERE did=$1`, owner).Scan(&refreshRows); err != nil || refreshRows != 1 {
		t.Fatalf("coalesced refresh rows=%d err=%v, want one", refreshRows, err)
	}
	var handle syntax.Handle
	if err := pool.QueryRow(ctx, `SELECT handle FROM atproto_identity_cache WHERE did=$1`, owner).Scan(&handle); err != nil || handle != "verified.example" {
		t.Fatalf("verified presentation handle=%s err=%v, want unchanged", handle, err)
	}
	lifecycle, err := ownerLifecycleAt(t, pool, owner)
	if err != nil || lifecycle.State != ownerlifecycle.StateActive || lifecycle.Generation != 7 || lifecycle.AuthEpoch != 3 {
		t.Fatalf("owner lifecycle=%+v err=%v, want unchanged active owner", lifecycle, err)
	}
	var profileRows, purgeRows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM craftsky_profiles WHERE did=$1`, owner).Scan(&profileRows); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM owner_purge_components WHERE owner_did=$1`, owner).Scan(&purgeRows); err != nil {
		t.Fatal(err)
	}
	if profileRows != 1 || purgeRows != 0 {
		t.Fatalf("owner data profileRows=%d purgeRows=%d, want 1/0", profileRows, purgeRows)
	}
}

func ownerLifecycleAt(t *testing.T, pool *pgxpool.Pool, owner syntax.DID) (ownerlifecycle.Lifecycle, error) {
	t.Helper()
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		return ownerlifecycle.Lifecycle{}, err
	}
	store, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		return ownerlifecycle.Lifecycle{}, err
	}
	return store.Get(context.Background(), owner)
}

func TestNewerTapIdentityEventSupersedesInFlightRefreshFinalization(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		resolved   syntax.Handle
		resolveErr error
	}{
		{name: "older success cannot overwrite or clear newer trigger", resolved: "event-700.example"},
		{name: "older failure cannot defer newer trigger", resolveErr: errors.New("temporary identity authority outage")},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			now := time.Date(2026, 8, 21, 9, 0, 0, 0, time.UTC)
			owner := syntax.DID("did:plc:identity-refresh-race-" + testCase.name[:5])
			pool := identityRefreshTriggerPool(t)
			if _, err := pool.Exec(ctx, `
				INSERT INTO owner_lifecycles(
					owner_did,state,generation,auth_epoch,transition_reason,
					transitioned_at,created_at,updated_at
				) VALUES($1,'active',1,1,'test',$2,$2,$2)
			`, owner, now); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'profile-cid')
			`, owner); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO atproto_identity_cache(did,handle,handle_lower,resolved_at)
				VALUES($1,'before.example','before.example',$2)
			`, owner, now.Add(-time.Hour)); err != nil {
				t.Fatal(err)
			}

			service := identityRefreshTriggerService(t, pool, now)
			if outcome, err := service.IngestIdentity(ctx, tap.IdentityEvent{
				ID: 700, DID: owner, Handle: "event-700.example", IsActive: true, Status: "active",
			}); err != nil || outcome.Kind != tap.OutcomeApplied {
				t.Fatalf("ingest event 700 outcome=%+v err=%v", outcome, err)
			}

			resolver := &blockingIdentityRefreshResolver{
				started: make(chan struct{}), release: make(chan struct{}),
				did: owner, handle: testCase.resolved, err: testCase.resolveErr,
			}
			processor, err := api.NewIdentityCacheRefreshProcessor(api.IdentityCacheRefreshProcessorOptions{
				Store: api.NewIdentityCacheStore(pool), Resolver: resolver,
				BatchSize: 1, OperationTimeout: 5 * time.Second, RetryDelay: 15 * time.Minute,
				Now: func() time.Time { return now },
			})
			if err != nil {
				t.Fatal(err)
			}
			type processResult struct {
				processed int
				err       error
			}
			result := make(chan processResult, 1)
			go func() {
				processed, processErr := processor.ProcessBatch(ctx)
				result <- processResult{processed: processed, err: processErr}
			}()
			select {
			case <-resolver.started:
			case <-time.After(time.Second):
				t.Fatal("older refresh did not reach authoritative resolution")
			}

			if outcome, err := service.IngestIdentity(ctx, tap.IdentityEvent{
				ID: 701, DID: owner, Handle: "event-701.example", IsActive: true, Status: "active",
			}); err != nil || outcome.Kind != tap.OutcomeApplied {
				t.Fatalf("ingest event 701 outcome=%+v err=%v", outcome, err)
			}
			if err := api.NewIdentityCacheService(
				pool,
				staticIdentityRefreshResolver{did: owner, handle: "verified-701.example"},
				func() time.Time { return now.Add(time.Second) },
				nil,
			).RefreshCurrentHandle(ctx, owner); err != nil {
				t.Fatalf("write newer verified mapping: %v", err)
			}

			close(resolver.release)
			select {
			case got := <-result:
				if got.err != nil || got.processed != 1 {
					t.Fatalf("older ProcessBatch processed=%d err=%v", got.processed, got.err)
				}
			case <-time.After(time.Second):
				t.Fatal("older refresh did not finish")
			}

			var refreshStates int
			if err := pool.QueryRow(ctx, `SELECT count(*) FROM atproto_identity_refresh_state WHERE did=$1`, owner).Scan(&refreshStates); err != nil || refreshStates != 0 {
				t.Fatalf("refresh state rows=%d err=%v, want committed authoritative refresh", refreshStates, err)
			}
			var verified syntax.Handle
			if err := pool.QueryRow(ctx, `SELECT handle FROM atproto_identity_cache WHERE did=$1`, owner).Scan(&verified); err != nil {
				t.Fatal(err)
			}
			if verified != "verified-701.example" {
				t.Fatalf("verified mapping=%s, want verified-701.example", verified)
			}
		})
	}
}

func identityRefreshTriggerPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return lifecycleIngestionPool(t)
}

func identityRefreshTriggerService(
	t *testing.T,
	pool *pgxpool.Pool,
	now time.Time,
	invalidators ...*recordingIdentityInvalidator,
) *ingestion.Service {
	t.Helper()
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	var invalidator ingestion.IdentityInvalidator
	if len(invalidators) > 0 {
		invalidator = invalidators[0]
	}
	service, err := ingestion.NewService(ingestion.ServiceConfig{
		Store: store, Lifecycles: lifecycles,
		ProfileParticipant: func(context.Context, pgx.Tx, ownerlifecycle.Lifecycle, ownerlifecycle.Lifecycle) error {
			return nil
		},
		IdentityInvalidator: invalidator,
	})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func containsHandle(handles []syntax.Handle, expected syntax.Handle) bool {
	for _, handle := range handles {
		if handle == expected {
			return true
		}
	}
	return false
}

func assertIdentityRefreshState(
	t *testing.T,
	pool *pgxpool.Pool,
	did syntax.DID,
	eventID int64,
	result string,
	attempts int,
	nextAttempt time.Time,
) {
	t.Helper()
	var gotEventID *int64
	var gotResult string
	var gotAttempts int
	var gotNext time.Time
	if err := pool.QueryRow(context.Background(), `
		SELECT tap_event_id,last_result,attempt_count,next_attempt_at
		FROM atproto_identity_refresh_state WHERE did=$1
	`, did).Scan(&gotEventID, &gotResult, &gotAttempts, &gotNext); err != nil {
		t.Fatalf("read identity refresh state: %v", err)
	}
	if gotEventID == nil || *gotEventID != eventID || gotResult != result ||
		gotAttempts != attempts || !gotNext.Equal(nextAttempt) {
		t.Fatalf("refresh state event=%v result=%q attempts=%d next=%s; want %d/%q/%d/%s",
			gotEventID, gotResult, gotAttempts, gotNext, eventID, result, attempts, nextAttempt)
	}
}
