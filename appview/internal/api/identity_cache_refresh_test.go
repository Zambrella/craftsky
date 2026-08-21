package api_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

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
	handle_lower TEXT NOT NULL UNIQUE,
	resolved_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
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
	resolver := &refreshResolver{handles: map[syntax.DID]syntax.Handle{did: "new.example"}}
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
