package ingestion_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
)

var errInjectedLifecycleParticipant = errors.New("injected lifecycle participant failure")

func TestProfileSourceWinnerAndLifecycleTransitionCommitAtomically(t *testing.T) {
	pool := lifecycleIngestionPool(t)
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatalf("new owner fencer: %v", err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatalf("new lifecycle store: %v", err)
	}
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	owner := syntax.DID("did:plc:profile-owner")
	if _, err := lifecycles.EnsureOnboardingOwner(context.Background(), owner); err != nil {
		t.Fatalf("ensure onboarding owner: %v", err)
	}
	profile := tap.Event{
		ID: 20, URI: "at://did:plc:profile-owner/social.craftsky.actor.profile/self",
		DID: owner, Collection: "social.craftsky.actor.profile", Rkey: "self",
		Rev: "3aaaaaaaaaaa4", CID: "bafy-profile", Action: "create",
		Record: json.RawMessage(`{"crafts":["sewing"]}`),
	}

	failingService := newLifecycleIngestionService(t, store, lifecycles,
		func(context.Context, pgx.Tx, ownerlifecycle.Lifecycle, ownerlifecycle.Lifecycle) error {
			return errInjectedLifecycleParticipant
		}, nil)
	if _, err := failingService.IngestRecord(context.Background(), profile); !errors.Is(err, errInjectedLifecycleParticipant) {
		t.Fatalf("failed activation error=%v", err)
	}
	assertLifecycleState(t, lifecycles, owner, ownerlifecycle.StateDeparted)
	assertNoTapSource(t, pool, profile.URI)

	service := newLifecycleIngestionService(t, store, lifecycles, nil, nil)
	if outcome, err := service.IngestRecord(context.Background(), profile); err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("activate profile outcome=%+v err=%v", outcome, err)
	}
	assertLifecycleState(t, lifecycles, owner, ownerlifecycle.StateActive)
	assertTapSourceAction(t, store, profile.URI, "create")
	assertRepositoryJobCount(t, pool, owner, 1)

	staleWhileActive := profile
	staleWhileActive.ID = 19
	staleWhileActive.Rev = "3aaaaaaaaaaa3"
	staleWhileActive.CID = "bafy-stale-profile"
	if outcome, err := service.IngestRecord(context.Background(), staleWhileActive); err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("active stale profile outcome=%+v err=%v", outcome, err)
	}
	assertSourceReceiptCount(t, pool, profile.URI, 2)

	conflictWhileActive := profile
	conflictWhileActive.CID = "bafy-conflicting-profile"
	conflictWhileActive.Record = json.RawMessage(`{"crafts":["quilting"]}`)
	if outcome, err := service.IngestRecord(context.Background(), conflictWhileActive); err != nil ||
		outcome.Kind != tap.OutcomeBlocked || outcome.Reason != tap.ReasonSourceOrderUncertain {
		t.Fatalf("active conflicting profile outcome=%+v err=%v", outcome, err)
	}
	source, err := store.Source(context.Background(), profile.URI)
	if err != nil || source.OrderingStatus != "uncertain" {
		t.Fatalf("uncertain profile source=%+v err=%v", source, err)
	}
	assertSourceReceiptCount(t, pool, profile.URI, 3)
	if outcome, err := service.ReconcileSource(context.Background(), ingestion.ReconciledSource{
		URI: profile.URI, DID: owner, ExpectedEventID: source.SourceEventID,
		ExpectedFingerprint: source.SourceFingerprint, Revision: profile.Rev,
		CID: conflictWhileActive.CID, Record: conflictWhileActive.Record, Present: true,
	}); err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("reconcile authoritative profile outcome=%+v err=%v", outcome, err)
	}
	source, err = store.Source(context.Background(), profile.URI)
	if err != nil || source.OrderingStatus != "authoritative" || source.CID != conflictWhileActive.CID {
		t.Fatalf("reconciled profile source=%+v err=%v", source, err)
	}
	job, err := store.ProjectionJob(context.Background(), profile.URI)
	if err != nil || job.State != "pending" {
		t.Fatalf("reconciled profile job=%+v err=%v", job, err)
	}
	assertLifecycleState(t, lifecycles, owner, ownerlifecycle.StateActive)

	deleted := profile
	deleted.ID = 21
	deleted.Rev = "3aaaaaaaaaaa5"
	deleted.CID = ""
	deleted.Action = "delete"
	deleted.Record = nil
	failingDeparture := newLifecycleIngestionService(t, store, lifecycles,
		func(context.Context, pgx.Tx, ownerlifecycle.Lifecycle, ownerlifecycle.Lifecycle) error {
			return errInjectedLifecycleParticipant
		}, nil)
	if _, err := failingDeparture.IngestRecord(context.Background(), deleted); !errors.Is(err, errInjectedLifecycleParticipant) {
		t.Fatalf("failed departure error=%v", err)
	}
	assertLifecycleState(t, lifecycles, owner, ownerlifecycle.StateActive)
	assertTapSourceAction(t, store, profile.URI, "update")

	if outcome, err := service.IngestRecord(context.Background(), deleted); err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("depart profile outcome=%+v err=%v", outcome, err)
	}
	assertLifecycleState(t, lifecycles, owner, ownerlifecycle.StateDeparted)
	assertTapSourceAction(t, store, profile.URI, "delete")

	// A same-token conflict while departed is classified under the owner fence.
	// It cannot commit the attempted activation before authoritative PDS
	// reconciliation chooses the source winner.
	conflictWhileDeparted := deleted
	conflictWhileDeparted.Action = "create"
	conflictWhileDeparted.CID = "bafy-departed-conflict"
	conflictWhileDeparted.Record = json.RawMessage(`{"crafts":["weaving"]}`)
	if outcome, err := service.IngestRecord(context.Background(), conflictWhileDeparted); err != nil ||
		outcome.Kind != tap.OutcomeBlocked || outcome.Reason != tap.ReasonSourceOrderUncertain {
		t.Fatalf("departed conflicting profile outcome=%+v err=%v", outcome, err)
	}
	assertLifecycleState(t, lifecycles, owner, ownerlifecycle.StateDeparted)
	source, err = store.Source(context.Background(), profile.URI)
	if err != nil || source.Action != "delete" || source.OrderingStatus != "uncertain" {
		t.Fatalf("departed conflict source=%+v err=%v", source, err)
	}

	// A redelivery older than the winning tombstone is durable but cannot
	// reactivate the lifecycle or resurrect source state.
	if outcome, err := service.IngestRecord(context.Background(), profile); err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("stale profile replay outcome=%+v err=%v", outcome, err)
	}
	assertLifecycleState(t, lifecycles, owner, ownerlifecycle.StateDeparted)
	assertTapSourceAction(t, store, profile.URI, "delete")

	// Tap's event ID belongs to Tap's local database and can restart from one
	// when that store is rebuilt. The repository revision remains authoritative
	// across that reset, so a newer profile recreation must still reactivate the
	// owner and replace the retained tombstone.
	recreatedAfterTapReset := profile
	recreatedAfterTapReset.ID = 1
	recreatedAfterTapReset.Rev = "3aaaaaaaaaaa6"
	recreatedAfterTapReset.CID = "bafy-profile-recreated"
	if outcome, err := service.IngestRecord(context.Background(), recreatedAfterTapReset); err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("profile recreation after Tap reset outcome=%+v err=%v", outcome, err)
	}
	assertLifecycleState(t, lifecycles, owner, ownerlifecycle.StateActive)
	assertTapSourceAction(t, store, profile.URI, "create")

	// Re-delivery after a Tap reset has a different database-local event ID
	// and therefore a different receipt fingerprint. The durable repository
	// source tuple is unchanged, so this is a duplicate rather than a conflict.
	replayedAfterTapReset := recreatedAfterTapReset
	replayedAfterTapReset.ID = 2
	if outcome, err := service.IngestRecord(context.Background(), replayedAfterTapReset); err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("profile replay after Tap reset outcome=%+v err=%v", outcome, err)
	}
	assertLifecycleState(t, lifecycles, owner, ownerlifecycle.StateActive)
	assertTapSourceAction(t, store, profile.URI, "create")

	// Conversely, a larger database-local Tap ID cannot let an older repository
	// revision replace the current profile after the reset.
	oldDeleteAfterTapReset := deleted
	oldDeleteAfterTapReset.ID = 100
	if outcome, err := service.IngestRecord(context.Background(), oldDeleteAfterTapReset); err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("old delete after Tap reset outcome=%+v err=%v", outcome, err)
	}
	assertLifecycleState(t, lifecycles, owner, ownerlifecycle.StateActive)
	assertTapSourceAction(t, store, profile.URI, "create")

	invalidRevision := deleted
	invalidRevision.ID = 101
	invalidRevision.Rev = "zzzzzzzzzzzzz"
	if outcome, err := service.IngestRecord(context.Background(), invalidRevision); err == nil || outcome.Kind != tap.OutcomeRetryable {
		t.Fatalf("invalid revision outcome=%+v err=%v", outcome, err)
	}
	assertLifecycleState(t, lifecycles, owner, ownerlifecycle.StateActive)
	assertTapSourceAction(t, store, profile.URI, "create")

	conflictingBody := recreatedAfterTapReset
	conflictingBody.ID = 102
	conflictingBody.Record = json.RawMessage(`{"crafts":["knitting"]}`)
	if outcome, err := service.IngestRecord(context.Background(), conflictingBody); err != nil ||
		outcome.Kind != tap.OutcomeBlocked || outcome.Reason != tap.ReasonSourceOrderUncertain {
		t.Fatalf("same tuple with different body outcome=%+v err=%v", outcome, err)
	}
	source, err = store.Source(context.Background(), profile.URI)
	if err != nil {
		t.Fatal(err)
	}
	job, err = store.ProjectionJob(context.Background(), profile.URI)
	if err != nil {
		t.Fatal(err)
	}
	if source.OrderingStatus != "uncertain" || job.State != "blocked" {
		t.Fatalf("conflicting source/job=%+v/%+v", source, job)
	}
	assertLifecycleState(t, lifecycles, owner, ownerlifecycle.StateActive)
}

func TestInvalidProfileSourceDoesNotActivateMembership(t *testing.T) {
	pool := lifecycleIngestionPool(t)
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatalf("new owner fencer: %v", err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatalf("new lifecycle store: %v", err)
	}
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	owner := syntax.DID("did:plc:invalid-profile-owner")
	if _, err := lifecycles.EnsureOnboardingOwner(context.Background(), owner); err != nil {
		t.Fatalf("ensure onboarding owner: %v", err)
	}
	event := tap.Event{
		ID: 120, URI: "at://did:plc:invalid-profile-owner/social.craftsky.actor.profile/self",
		DID: owner, Collection: "social.craftsky.actor.profile", Rkey: "self",
		Rev: "3aaaaaaaaaab2", CID: "bafy-invalid-profile", Action: "create",
		Record: json.RawMessage(`{"crafts":["1","2","3","4","5","6","7","8","9","10","11"]}`),
	}
	service := newLifecycleIngestionService(t, store, lifecycles, nil, nil)
	if outcome, err := service.IngestRecord(context.Background(), event); err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("ingest invalid profile outcome=%+v err=%v", outcome, err)
	}
	assertLifecycleState(t, lifecycles, owner, ownerlifecycle.StateDeparted)
	assertTapSourceAction(t, store, event.URI, "create")
	source, err := store.Source(context.Background(), event.URI)
	if err != nil {
		t.Fatalf("read invalid profile source: %v", err)
	}
	if source.StructuralValidationStatus != "invalid" || source.SemanticValidationStatus != "invalid" || source.ValidationReason != "invalid_lexicon" {
		t.Fatalf("invalid profile validation = %q/%q %q, want invalid/invalid invalid_lexicon",
			source.StructuralValidationStatus, source.SemanticValidationStatus, source.ValidationReason)
	}
}

func TestForeignClientRecordIsNotGatedByHistoricalEffectAttempt(t *testing.T) {
	pool := lifecycleIngestionPool(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatalf("new owner fencer: %v", err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatalf("new lifecycle store: %v", err)
	}
	store, err := ingestion.NewStore(pool, func() time.Time { return now })
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	owner := syntax.DID("did:plc:foreign-client-owner")
	lifecycle, err := lifecycles.EnsureOnboardingOwner(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycles.Transition(ctx, ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: lifecycle.Generation,
		To: ownerlifecycle.StateActive, Reason: "testActive",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO craftsky_profiles(did,record_cid) VALUES($1,'bafy-profile')`, owner); err != nil {
		t.Fatalf("seed active profile: %v", err)
	}
	uri := syntax.ATURI("at://did:plc:foreign-client-owner/social.craftsky.feed.post/3aaaaaaaaaab2")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_effect_attempts(
			operation_id,owner_did,owner_generation,effect_kind,effect_action,
			mutation_key,deterministic_key,request_fingerprint,record_fingerprint,
			remote_outcome,projection_disposition,repeat_forbidden,remote_deadline,
			dispatched_at,created_at,updated_at
		) VALUES(
			'legacy-foreign-client-collision',$1,$2,'pds_record','put_record',
			'legacy-foreign-client-collision',$3,decode(repeat('04',32),'hex'),
			decode(repeat('04',32),'hex'),'outcome_unknown_pre_transition',
			'hidden_non_active',true,$4,$5,$5,$5
		)
	`, owner, lifecycle.Generation+1, uri, now.Add(time.Hour), now); err != nil {
		t.Fatalf("seed historical effect attempt: %v", err)
	}

	event := tap.Event{
		ID: 121, URI: uri, DID: owner, Collection: "social.craftsky.feed.post", Rkey: "3aaaaaaaaaab2",
		Rev: "3aaaaaaaaaab2", CID: "bafy-foreign-client-post", Action: "create", Live: true,
		Record: json.RawMessage(`{"text":"written by another client","createdAt":"2026-09-25T12:00:00Z"}`),
	}
	service := newLifecycleIngestionService(t, store, lifecycles, nil, nil)
	outcome, err := service.IngestRecord(ctx, event)
	if err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("foreign-client record outcome=%+v err=%v", outcome, err)
	}
	assertTapSourceAction(t, store, uri, "create")
}

func TestTapDeletedIdentityIsOnlyARefreshHint(t *testing.T) {
	pool := lifecycleIngestionPool(t)
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatalf("new owner fencer: %v", err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, time.Now)
	if err != nil {
		t.Fatalf("new lifecycle store: %v", err)
	}
	store, err := ingestion.NewStore(pool, time.Now)
	if err != nil {
		t.Fatalf("new ingestion store: %v", err)
	}
	owner := syntax.DID("did:plc:status-hint-owner")
	identity := tap.IdentityEvent{ID: 90, DID: owner, Status: "deleted"}
	active, err := lifecycles.EnsureOnboardingOwner(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycles.Transition(context.Background(), ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: active.Generation,
		To: ownerlifecycle.StateActive, Reason: "testActive",
	}); err != nil {
		t.Fatal(err)
	}

	service := newLifecycleIngestionService(t, store, lifecycles, nil, nil)
	if outcome, err := service.IngestIdentity(context.Background(), identity); err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("deleted identity hint outcome=%+v err=%v", outcome, err)
	}
	assertLifecycleState(t, lifecycles, owner, ownerlifecycle.StateActive)
	assertIdentityReceiptCount(t, pool, 1)
	var components int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM owner_purge_components WHERE owner_did=$1
	`, owner).Scan(&components); err != nil || components != 0 {
		t.Fatalf("status hint purge components=%d err=%v", components, err)
	}
}

func newLifecycleIngestionService(
	t *testing.T,
	store *ingestion.Store,
	lifecycles *ownerlifecycle.Store,
	profileParticipant ownerlifecycle.TransitionParticipant,
	_ ownerlifecycle.TerminalParticipant,
) *ingestion.Service {
	t.Helper()
	if profileParticipant == nil {
		profileParticipant = func(context.Context, pgx.Tx, ownerlifecycle.Lifecycle, ownerlifecycle.Lifecycle) error {
			return nil
		}
	}
	service, err := ingestion.NewService(ingestion.ServiceConfig{
		Store: store, Lifecycles: lifecycles,
		ProfileParticipant: profileParticipant,
	})
	if err != nil {
		t.Fatalf("new ingestion service: %v", err)
	}
	return service
}

func lifecycleIngestionPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := testdb.WithSchema(t, ingestionProjectionFixtureDDL)
	for _, path := range []string{
		"000015_identity_handle_cache.up.sql",
		"000002_oauth_tables.up.sql",
		"000003_oauth_auth_requests_handoff.up.sql",
		"000006_craftsky_sessions_device_id.up.sql",
		"000037_account_deletion.up.sql",
		"000038_owner_auth_lifecycle.up.sql",
		"000039_owner_effects_terminal_purge.up.sql",
		"000045_tap_ingestion_durability.up.sql",
		"000049_pds_effect_action.up.sql",
		"000050_pds_effect_source_reconciliation.up.sql",
		"000051_tap_quarantine_replay_payload.up.sql",
		"000053_identity_cache_refresh.up.sql",
		"000054_tap_identity_refresh_trigger.up.sql",
		"000056_tap_source_projection_generation.up.sql",
		"000057_tap_identity_refresh_version.up.sql",
		"000058_tap_projection_generation_column.up.sql",
	} {
		sql, err := testdb.ReadMigration(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", path, err)
		}
		if _, err := pool.Exec(context.Background(), string(sql)); err != nil {
			t.Fatalf("apply migration %s: %v", path, err)
		}
	}
	if _, err := pool.Exec(context.Background(), `
		ALTER TABLE tap_source_records
			ADD COLUMN validation_version INTEGER NOT NULL DEFAULT 1,
			ADD COLUMN structural_validation_status TEXT NOT NULL DEFAULT 'pending',
			ADD COLUMN semantic_validation_status TEXT NOT NULL DEFAULT 'pending',
			ADD COLUMN validation_reason TEXT
	`); err != nil {
		t.Fatalf("add source validation fixture columns: %v", err)
	}
	return pool
}

func assertLifecycleState(t *testing.T, store *ownerlifecycle.Store, owner syntax.DID, state ownerlifecycle.State) {
	t.Helper()
	lifecycle, err := store.Get(context.Background(), owner)
	if err != nil {
		t.Fatalf("read lifecycle %s: %v", owner, err)
	}
	if lifecycle.State != state {
		t.Fatalf("lifecycle state=%s, want %s", lifecycle.State, state)
	}
}

func assertNoTapSource(t *testing.T, pool *pgxpool.Pool, uri syntax.ATURI) {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM tap_source_records WHERE uri=$1`, uri).Scan(&count); err != nil || count != 0 {
		t.Fatalf("Tap source count=%d err=%v", count, err)
	}
}

func assertTapSourceAction(t *testing.T, store *ingestion.Store, uri syntax.ATURI, action string) {
	t.Helper()
	source, err := store.Source(context.Background(), uri)
	if err != nil {
		t.Fatalf("read Tap source: %v", err)
	}
	if source.Action != action {
		t.Fatalf("Tap source action=%s, want %s", source.Action, action)
	}
}

func assertIdentityReceiptCount(t *testing.T, pool *pgxpool.Pool, want int) {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM tap_ingestion_receipts WHERE event_type='identity'
	`).Scan(&count); err != nil || count != want {
		t.Fatalf("identity receipts=%d want=%d err=%v", count, want, err)
	}
}

func assertSourceReceiptCount(t *testing.T, pool *pgxpool.Pool, uri syntax.ATURI, want int) {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM tap_ingestion_receipts WHERE source_uri=$1
	`, uri).Scan(&count); err != nil || count != want {
		t.Fatalf("source receipts=%d want=%d err=%v", count, want, err)
	}
}
