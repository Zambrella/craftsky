package ingestion_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
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
		Rev: "3m00000000020", CID: "bafy-profile", Action: "create",
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
	staleWhileActive.Rev = "3m00000000019"
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
	deleted.Rev = "3m00000000021"
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
}

func TestTerminalIdentityReceiptAndLifecycleDenialCommitAtomically(t *testing.T) {
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
	owner := syntax.DID("did:plc:terminal-owner")
	identity := tap.IdentityEvent{ID: 90, DID: owner, Status: "deleted"}

	failing := newLifecycleIngestionService(t, store, lifecycles, nil,
		func(context.Context, pgx.Tx, *ownerlifecycle.Lifecycle, ownerlifecycle.Lifecycle) error {
			return errInjectedLifecycleParticipant
		})
	if _, err := failing.IngestIdentity(context.Background(), identity); !errors.Is(err, errInjectedLifecycleParticipant) {
		t.Fatalf("failed terminal identity error=%v", err)
	}
	if _, err := lifecycles.Get(context.Background(), owner); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("lifecycle after failed terminal identity error=%v", err)
	}
	assertIdentityReceiptCount(t, pool, 0)

	service := newLifecycleIngestionService(t, store, lifecycles, nil, nil)
	if outcome, err := service.IngestIdentity(context.Background(), identity); err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("terminal identity outcome=%+v err=%v", outcome, err)
	}
	assertLifecycleState(t, lifecycles, owner, ownerlifecycle.StateTerminal)
	assertIdentityReceiptCount(t, pool, 1)
	var components int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM owner_purge_components WHERE owner_did=$1
	`, owner).Scan(&components); err != nil || components != 2 {
		t.Fatalf("terminal purge components=%d err=%v", components, err)
	}

	profile := tap.Event{
		ID: 91, URI: "at://did:plc:terminal-owner/social.craftsky.actor.profile/self",
		DID: owner, Collection: "social.craftsky.actor.profile", Rkey: "self",
		Rev: "3m00000000091", CID: "bafy-terminal-profile", Action: "create",
		Record: json.RawMessage(`{"crafts":["sewing"]}`),
	}
	if outcome, err := service.IngestRecord(context.Background(), profile); err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("terminal profile denial outcome=%+v err=%v", outcome, err)
	}
	var addRepoJobs int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM tap_repository_jobs
		WHERE did=$1 AND job_kind='tap_add_repo'
	`, owner).Scan(&addRepoJobs); err != nil || addRepoJobs != 0 {
		t.Fatalf("terminal profile enqueued Tap AddRepo jobs=%d err=%v", addRepoJobs, err)
	}
}

func newLifecycleIngestionService(
	t *testing.T,
	store *ingestion.Store,
	lifecycles *ownerlifecycle.Store,
	profileParticipant ownerlifecycle.TransitionParticipant,
	terminalParticipant ownerlifecycle.TerminalParticipant,
) *ingestion.Service {
	t.Helper()
	if profileParticipant == nil {
		profileParticipant = func(context.Context, pgx.Tx, ownerlifecycle.Lifecycle, ownerlifecycle.Lifecycle) error {
			return nil
		}
	}
	if terminalParticipant == nil {
		terminalParticipant = func(context.Context, pgx.Tx, *ownerlifecycle.Lifecycle, ownerlifecycle.Lifecycle) error {
			return nil
		}
	}
	service, err := ingestion.NewService(ingestion.ServiceConfig{
		Store: store, Lifecycles: lifecycles,
		ProfileParticipant:  profileParticipant,
		TerminalParticipant: terminalParticipant,
		TerminalComponents: []ownerlifecycle.PurgeComponent{
			{Component: "public_records", DIDRole: "owner"},
			{Component: "sessions", DIDRole: "owner"},
		},
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
		"../../migrations/000002_oauth_tables.up.sql",
		"../../migrations/000003_oauth_auth_requests_handoff.up.sql",
		"../../migrations/000006_craftsky_sessions_device_id.up.sql",
		"../../migrations/000037_account_deletion.up.sql",
		"../../migrations/000038_owner_auth_lifecycle.up.sql",
		"../../migrations/000039_owner_effects_terminal_purge.up.sql",
		"../../migrations/000045_tap_ingestion_durability.up.sql",
	} {
		sql, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", path, err)
		}
		if _, err := pool.Exec(context.Background(), string(sql)); err != nil {
			t.Fatalf("apply migration %s: %v", path, err)
		}
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
