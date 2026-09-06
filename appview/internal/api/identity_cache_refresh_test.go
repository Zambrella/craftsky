package api_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/testdb"
)

const identityRefreshDDL = `
CREATE TABLE owner_lifecycles (
	owner_did TEXT PRIMARY KEY,
	state TEXT NOT NULL,
	generation BIGINT NOT NULL,
	auth_epoch BIGINT NOT NULL DEFAULT 1,
	transition_reason TEXT NOT NULL DEFAULT 'test',
	transitioned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	terminal_at TIMESTAMPTZ,
	purge_completed_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE FUNCTION appview_owner_is_terminal(candidate_did TEXT)
RETURNS BOOLEAN LANGUAGE SQL STABLE AS $$
	SELECT EXISTS (SELECT 1 FROM owner_lifecycles WHERE owner_did=candidate_did AND state='terminal')
$$;
CREATE TABLE craftsky_profiles (
	did TEXT PRIMARY KEY,
	crafts TEXT[] NOT NULL DEFAULT '{}',
	record_cid TEXT NOT NULL,
	indexed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE atproto_identity_cache (
	did TEXT PRIMARY KEY,
	handle TEXT NOT NULL,
	handle_lower TEXT NOT NULL,
	resolved_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX atproto_identity_cache_handle_lower_unique
	ON atproto_identity_cache(handle_lower)
	WHERE handle_lower <> 'handle.invalid';
CREATE TABLE atproto_identity_refresh_state (
	did TEXT PRIMARY KEY,
	next_attempt_at TIMESTAMPTZ NOT NULL,
	attempt_count INTEGER NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
	last_result TEXT NOT NULL CHECK (last_result IN ('pending','retry')),
	tap_event_id BIGINT CHECK (tap_event_id > 0),
	refresh_version BIGINT NOT NULL DEFAULT 1 CHECK (refresh_version > 0),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
`

type refreshResolver struct {
	handles map[syntax.DID]syntax.Handle
	dids    map[syntax.Handle]syntax.DID
	errors  map[syntax.DID]error
	calls   []syntax.DID
}

type refreshInvalidation struct {
	did     syntax.DID
	handles []syntax.Handle
}

type refreshInvalidator struct{ calls []refreshInvalidation }

func (invalidator *refreshInvalidator) InvalidateIdentity(_ context.Context, did syntax.DID, handles ...syntax.Handle) {
	invalidator.calls = append(invalidator.calls, refreshInvalidation{did: did, handles: append([]syntax.Handle(nil), handles...)})
}

func (resolver *refreshResolver) ResolveHandle(_ context.Context, did syntax.DID) (syntax.Handle, error) {
	resolver.calls = append(resolver.calls, did)
	if err := resolver.errors[did]; err != nil {
		return "", err
	}
	return resolver.handles[did], nil
}

func (resolver *refreshResolver) ResolveDID(_ context.Context, handle syntax.Handle) (syntax.DID, error) {
	if did := resolver.dids[handle]; did != "" {
		return did, nil
	}
	return "", errors.New("handle does not resolve")
}

func TestIdentityCacheAuthoritativeRefreshNewestWinsAndOwnsAliases(t *testing.T) {
	pool := testdb.WithSchema(t, identityRefreshDDL)
	ctx := context.Background()
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	did := syntax.DID("did:plc:migrating")
	previousOwner := syntax.DID("did:plc:previous-owner")
	for _, statement := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO owner_lifecycles(owner_did,state,generation) VALUES ($1,'active',1),($2,'active',1)`, []any{did, previousOwner}},
		{`INSERT INTO craftsky_profiles(did,record_cid) VALUES ($1,'migrating-profile'),($2,'previous-profile')`, []any{did, previousOwner}},
		{`INSERT INTO atproto_identity_cache(did,handle,handle_lower,resolved_at) VALUES ($1,'old.example','old.example',$3),($2,'reassigned.example','reassigned.example',$3)`, []any{did, previousOwner, now}},
		{`INSERT INTO atproto_identity_refresh_state(did,next_attempt_at,attempt_count,last_result,tap_event_id,refresh_version,updated_at) VALUES($1,$2,0,'pending',100,1,$2)`, []any{did, now}},
	} {
		if _, err := pool.Exec(ctx, statement.sql, statement.args...); err != nil {
			t.Fatal(err)
		}
	}

	releaseOld := make(chan struct{})
	oldResolver := &blockingHandleResolver{
		did: did, handle: "stale.example", handleStarted: make(chan struct{}), releaseHandle: releaseOld,
	}
	oldInvalidator := &refreshInvalidator{}
	oldProcessor, err := api.NewIdentityCacheRefreshProcessor(api.IdentityCacheRefreshProcessorOptions{
		Store: api.NewIdentityCacheStore(pool), Resolver: oldResolver,
		BatchSize: 1, OperationTimeout: 5 * time.Second, RetryDelay: time.Minute,
		Now: func() time.Time { return now }, IdentityInvalidator: oldInvalidator,
	})
	if err != nil {
		t.Fatal(err)
	}
	type processResult struct {
		processed int
		err       error
	}
	oldResult := make(chan processResult, 1)
	go func() {
		processed, processErr := oldProcessor.ProcessBatch(ctx)
		oldResult <- processResult{processed: processed, err: processErr}
	}()
	select {
	case <-oldResolver.handleStarted:
	case <-time.After(time.Second):
		t.Fatal("old authoritative refresh did not start")
	}

	if _, err := pool.Exec(ctx, `
		UPDATE atproto_identity_refresh_state
		SET refresh_version=refresh_version+1,tap_event_id=101,updated_at=$2
		WHERE did=$1
	`, did, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	newInvalidator := &refreshInvalidator{}
	newResolver := &refreshResolver{
		handles: map[syntax.DID]syntax.Handle{did: "new.example"},
		dids:    map[syntax.Handle]syntax.DID{"new.example": did},
		errors:  map[syntax.DID]error{},
	}
	newProcessor, err := api.NewIdentityCacheRefreshProcessor(api.IdentityCacheRefreshProcessorOptions{
		Store: api.NewIdentityCacheStore(pool), Resolver: newResolver,
		BatchSize: 1, OperationTimeout: time.Second, RetryDelay: time.Minute,
		Now: func() time.Time { return now.Add(time.Second) }, IdentityInvalidator: newInvalidator,
	})
	if err != nil {
		t.Fatal(err)
	}
	if processed, err := newProcessor.ProcessBatch(ctx); err != nil || processed != 1 {
		t.Fatalf("new authoritative refresh processed=%d err=%v", processed, err)
	}
	close(releaseOld)
	if result := <-oldResult; result.err != nil || result.processed != 1 {
		t.Fatalf("old authoritative refresh processed=%d err=%v", result.processed, result.err)
	}
	assertIdentityHandle(t, pool, did, "new.example")
	if len(oldInvalidator.calls) != 0 {
		t.Fatalf("stale completion invalidations=%+v, want none", oldInvalidator.calls)
	}
	if len(newInvalidator.calls) != 1 ||
		!containsRefreshHandle(newInvalidator.calls[0].handles, "old.example") ||
		!containsRefreshHandle(newInvalidator.calls[0].handles, "new.example") {
		t.Fatalf("committed invalidations=%+v, want DID plus old/new handles", newInvalidator.calls)
	}
	for _, staleAlias := range []syntax.Handle{"old.example", "stale.example"} {
		row, err := api.NewIdentityCacheStore(pool).FreshByHandle(ctx, staleAlias, now.Add(time.Second))
		if err != nil || row != nil {
			t.Fatalf("stale alias %s row=%+v err=%v, want removed", staleAlias, row, err)
		}
	}
	newAlias, err := api.NewIdentityCacheStore(pool).FreshByHandle(ctx, "new.example", now.Add(time.Second))
	if err != nil || newAlias == nil || newAlias.DID != did {
		t.Fatalf("new alias row=%+v err=%v, want verified DID %s", newAlias, err, did)
	}

	for _, testCase := range []struct {
		name       string
		handle     syntax.Handle
		reverseDID syntax.DID
		want       syntax.Handle
	}{
		{name: "reassigned alias", handle: "reassigned.example", reverseDID: did, want: "reassigned.example"},
		{name: "failed bidirectional proof", handle: "stolen.example", reverseDID: previousOwner, want: syntax.HandleInvalid},
		{name: "no valid handle sentinel", handle: syntax.HandleInvalid, want: syntax.HandleInvalid},
		{name: "duplicate sentinel converges", handle: syntax.HandleInvalid, want: syntax.HandleInvalid},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := pool.Exec(ctx, `
				INSERT INTO atproto_identity_refresh_state(
					did,next_attempt_at,attempt_count,last_result,tap_event_id,refresh_version,updated_at
				) VALUES($1,$2,0,'pending',200,1,$2)
				ON CONFLICT(did) DO UPDATE SET
					next_attempt_at=EXCLUDED.next_attempt_at,last_result='pending',
					refresh_version=atproto_identity_refresh_state.refresh_version+1,updated_at=EXCLUDED.updated_at
			`, did, now.Add(2*time.Second)); err != nil {
				t.Fatal(err)
			}
			resolver := &refreshResolver{
				handles: map[syntax.DID]syntax.Handle{did: testCase.handle},
				dids: map[syntax.Handle]syntax.DID{
					testCase.handle: testCase.reverseDID,
				},
				errors: map[syntax.DID]error{},
			}
			invalidator := &refreshInvalidator{}
			processor, err := api.NewIdentityCacheRefreshProcessor(api.IdentityCacheRefreshProcessorOptions{
				Store: api.NewIdentityCacheStore(pool), Resolver: resolver,
				BatchSize: 1, OperationTimeout: time.Second, RetryDelay: time.Minute,
				Now: func() time.Time { return now.Add(2 * time.Second) }, IdentityInvalidator: invalidator,
			})
			if err != nil {
				t.Fatal(err)
			}
			if processed, err := processor.ProcessBatch(ctx); err != nil || processed != 1 {
				t.Fatalf("ProcessBatch processed=%d err=%v", processed, err)
			}
			assertIdentityHandle(t, pool, did, testCase.want)
			wantInvalidations := 1
			if testCase.name == "reassigned alias" {
				wantInvalidations = 2
			}
			if len(invalidator.calls) != wantInvalidations {
				t.Fatalf("committed invalidations=%+v, want %d", invalidator.calls, wantInvalidations)
			}
		})
	}

	assertIdentityHandle(t, pool, previousOwner, syntax.HandleInvalid)
	var profiles int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM craftsky_profiles WHERE did=ANY($1)`, []string{did.String(), previousOwner.String()}).Scan(&profiles); err != nil || profiles != 2 {
		t.Fatalf("preserved DID profiles=%d err=%v, want 2", profiles, err)
	}
	row, err := api.NewIdentityCacheStore(pool).FreshByHandle(ctx, syntax.HandleInvalid, now.Add(2*time.Second))
	if err != nil || row != nil {
		t.Fatalf("sentinel alias lookup row=%+v err=%v, want no alias", row, err)
	}
}

func assertIdentityHandle(t *testing.T, pool interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, did syntax.DID, want syntax.Handle) {
	t.Helper()
	var got syntax.Handle
	if err := pool.QueryRow(context.Background(), `SELECT handle FROM atproto_identity_cache WHERE did=$1`, did).Scan(&got); err != nil {
		t.Fatalf("read identity handle for %s: %v", did, err)
	}
	if got != want {
		t.Fatalf("identity handle for %s=%s, want %s", did, got, want)
	}
}

func TestIdentityCacheRefreshProcessorIsBoundedAndDefersFailuresWithoutStarvation(t *testing.T) {
	t.Parallel()
	pool := testdb.WithSchema(t, identityRefreshDDL)
	ctx := context.Background()
	now := time.Date(2026, 8, 20, 15, 0, 0, 0, time.UTC)
	failed := syntax.DID("did:plc:aaa-failed")
	missing := syntax.DID("did:plc:bbb-missing")
	terminal := syntax.DID("did:plc:ccc-terminal")

	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(owner_did,state,generation,terminal_at) VALUES
			($1,'active',1,NULL),($2,'active',1,NULL),($3,'terminal',2,$4)
	`, failed, missing, terminal, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO craftsky_profiles(did,record_cid) VALUES
			($1,'cid-failed'),($2,'cid-missing'),($3,'cid-terminal')
	`, failed, missing, terminal); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_identity_cache(did,handle,handle_lower,resolved_at)
		VALUES($1,'old.example','old.example',$2)
	`, failed, now.Add(-25*time.Hour)); err != nil {
		t.Fatal(err)
	}
	resolver := &refreshResolver{
		handles: map[syntax.DID]syntax.Handle{missing: "missing.example", terminal: "terminal.example"},
		dids:    map[syntax.Handle]syntax.DID{"missing.example": missing, "terminal.example": terminal},
		errors:  map[syntax.DID]error{failed: errors.New("temporary directory outage")},
	}
	processor, err := api.NewIdentityCacheRefreshProcessor(api.IdentityCacheRefreshProcessorOptions{
		Store: api.NewIdentityCacheStore(pool), Resolver: resolver,
		BatchSize: 2, OperationTimeout: time.Second, RetryDelay: 5 * time.Minute,
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("NewIdentityCacheRefreshProcessor: %v", err)
	}

	processed, err := processor.ProcessBatch(ctx)
	if err != nil {
		t.Fatalf("ProcessBatch: %v", err)
	}
	if processed != 2 || len(resolver.calls) != 2 || !containsRefreshDID(resolver.calls, failed) || !containsRefreshDID(resolver.calls, missing) {
		t.Fatalf("processed=%d calls=%v, want both failed and missing candidates", processed, resolver.calls)
	}
	var handle syntax.Handle
	if err := pool.QueryRow(ctx, `SELECT handle FROM atproto_identity_cache WHERE did=$1`, missing).Scan(&handle); err != nil {
		t.Fatalf("read refreshed missing identity: %v", err)
	}
	if handle != "missing.example" {
		t.Fatalf("refreshed handle=%s, want missing.example", handle)
	}
	var nextAttempt time.Time
	if err := pool.QueryRow(ctx, `SELECT next_attempt_at FROM atproto_identity_refresh_state WHERE did=$1`, failed).Scan(&nextAttempt); err != nil {
		t.Fatalf("read failure deferral: %v", err)
	}
	if !nextAttempt.Equal(now.Add(5 * time.Minute)) {
		t.Fatalf("next attempt=%s, want %s", nextAttempt, now.Add(5*time.Minute))
	}

	processed, err = processor.ProcessBatch(ctx)
	if err != nil {
		t.Fatalf("second ProcessBatch: %v", err)
	}
	if processed != 0 || len(resolver.calls) != 2 {
		t.Fatalf("second batch processed=%d calls=%v, want deferred failure and terminal skipped", processed, resolver.calls)
	}
}

func TestIdentityCacheRefreshInvalidatesOldAndVerifiedMappingsAfterWrite(t *testing.T) {
	pool := testdb.WithSchema(t, identityRefreshDDL)
	ctx := context.Background()
	now := time.Date(2026, 8, 20, 18, 0, 0, 0, time.UTC)
	did := syntax.DID("did:plc:refresh-invalidation")
	for _, statement := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO owner_lifecycles(owner_did,state,generation) VALUES($1,'active',1)`, []any{did}},
		{`INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'cid-profile')`, []any{did}},
		{`INSERT INTO atproto_identity_cache(did,handle,handle_lower,resolved_at) VALUES($1,'old.example','old.example',$2)`, []any{did, now.Add(-time.Hour)}},
		{`INSERT INTO atproto_identity_refresh_state(did,next_attempt_at,attempt_count,last_result,updated_at,tap_event_id) VALUES($1,$2,0,'pending',$2,44)`, []any{did, now}},
	} {
		if _, err := pool.Exec(ctx, statement.sql, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	resolver := &refreshResolver{
		handles: map[syntax.DID]syntax.Handle{did: "new.example"},
		dids:    map[syntax.Handle]syntax.DID{"new.example": did},
	}
	invalidator := &refreshInvalidator{}
	processor, err := api.NewIdentityCacheRefreshProcessor(api.IdentityCacheRefreshProcessorOptions{
		Store: api.NewIdentityCacheStore(pool), Resolver: resolver,
		BatchSize: 1, OperationTimeout: time.Second, RetryDelay: 5 * time.Minute,
		Now: func() time.Time { return now }, IdentityInvalidator: invalidator,
	})
	if err != nil {
		t.Fatal(err)
	}

	processed, err := processor.ProcessBatch(ctx)
	if err != nil || processed != 1 {
		t.Fatalf("ProcessBatch processed=%d err=%v", processed, err)
	}
	var handle syntax.Handle
	if err := pool.QueryRow(ctx, `SELECT handle FROM atproto_identity_cache WHERE did=$1`, did).Scan(&handle); err != nil || handle != "new.example" {
		t.Fatalf("stored handle=%s err=%v", handle, err)
	}
	if len(invalidator.calls) != 1 || invalidator.calls[0].did != did ||
		!containsRefreshHandle(invalidator.calls[0].handles, "old.example") ||
		!containsRefreshHandle(invalidator.calls[0].handles, "new.example") {
		t.Fatalf("invalidations=%+v, want old and verified handles", invalidator.calls)
	}
	var states int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM atproto_identity_refresh_state WHERE did=$1`, did).Scan(&states); err != nil || states != 0 {
		t.Fatalf("refresh state rows=%d err=%v", states, err)
	}
}

func containsRefreshHandle(handles []syntax.Handle, expected syntax.Handle) bool {
	for _, handle := range handles {
		if handle == expected {
			return true
		}
	}
	return false
}

func containsRefreshDID(values []syntax.DID, expected syntax.DID) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
