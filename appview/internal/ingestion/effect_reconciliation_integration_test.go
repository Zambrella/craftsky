package ingestion_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"social.craftsky/appview/internal/ingestion"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/tap"
)

func TestPDSEffectSourceReconciliationMigrationDownUp(t *testing.T) {
	pool := lifecycleIngestionPool(t)
	renameDown, err := os.ReadFile("../../migrations/000058_tap_projection_generation_column.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(renameDown)); err != nil {
		t.Fatal(err)
	}
	assertRecordFingerprintColumn := func(want bool) {
		t.Helper()
		var exists bool
		if err := pool.QueryRow(context.Background(), `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema=current_schema()
				  AND table_name='owner_effect_attempts'
				  AND column_name='record_fingerprint'
			)
		`).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if exists != want {
			t.Fatalf("record_fingerprint exists=%t, want %t", exists, want)
		}
	}
	assertRecordFingerprintColumn(true)
	for _, step := range []struct {
		path string
		want bool
	}{
		{"../../migrations/000050_pds_effect_source_reconciliation.down.sql", false},
		{"../../migrations/000050_pds_effect_source_reconciliation.up.sql", true},
	} {
		migration, err := os.ReadFile(step.path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
			t.Fatalf("apply %s: %v", step.path, err)
		}
		assertRecordFingerprintColumn(step.want)
	}
	renameUp, err := os.ReadFile("../../migrations/000058_tap_projection_generation_column.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), string(renameUp)); err != nil {
		t.Fatal(err)
	}
}

func TestTapSourceProjectionGenerationMigrationBackfillsWakesAndRoundTrips(t *testing.T) {
	pool := lifecycleIngestionPool(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 21, 8, 0, 0, 0, time.UTC)
	const (
		owner       = "did:plc:migration-projection-generation"
		operationID = "migration-projection-generation:1"
		uri         = "at://did:plc:migration-projection-generation/social.craftsky.feed.post/post"
	)
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'active',2,2,'test',$2,$2,$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_effect_attempts(
			operation_id,owner_did,owner_generation,effect_kind,effect_action,
			mutation_key,deterministic_key,request_fingerprint,record_fingerprint,
			result_cid,remote_outcome,projection_disposition,repeat_forbidden,
			remote_deadline,dispatched_at,completed_at,created_at,updated_at
		) VALUES(
			$3,$1,1,'pds_record','put_record',$3,$4,
			decode(repeat('11',32),'hex'),decode(repeat('22',32),'hex'),
			'bafy-migration','accepted','eligible_current',true,
			$2::timestamptz+interval '1 hour',$2,$2,$2,$2
		)
	`, owner, now, operationID, uri); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO tap_source_records(
			uri,did,collection,rkey,source_event_id,source_fingerprint,
			revision,cid,action,record,record_bytes,live,ordering_status,
			projection_disposition,projection_generation,effect_operation_id,
			observed_at,updated_at
		) VALUES(
			$4,$1,'social.craftsky.feed.post','post',1,
			decode(repeat('33',32),'hex'),'3aaaaaaaaaaa2','bafy-migration',
			'create','{}'::json,2,false,'authoritative','eligible',2,$3,$2,$2
		)
	`, owner, now, operationID, uri); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO tap_projection_jobs(
			source_uri,projection_kind,source_event_id,state,dependency_kind,
			dependency_key,next_attempt_at,last_reason_code,created_at,updated_at
		) VALUES(
			$3,'social_craftsky_feed_post',1,'blocked','repository_did',$1,$2,
			'source_order_uncertain',$2,$2
		)
	`, owner, now, uri); err != nil {
		t.Fatal(err)
	}

	readMigration := func(path string) []byte {
		t.Helper()
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		return migration
	}
	apply := func(label string, migration []byte) {
		t.Helper()
		if _, err := pool.Exec(ctx, string(migration)); err != nil {
			t.Fatalf("apply %s: %v", label, err)
		}
	}
	down58 := readMigration("../../migrations/000058_tap_projection_generation_column.down.sql")
	up58 := readMigration("../../migrations/000058_tap_projection_generation_column.up.sql")
	down56 := readMigration("../../migrations/000056_tap_source_projection_generation.down.sql")
	up56 := readMigration("../../migrations/000056_tap_source_projection_generation.up.sql")
	apply("58 down", down58)
	apply("56 down", down56)

	assertState := func(wantGeneration int64, wantJob string) {
		t.Helper()
		var generation int64
		var state string
		if err := pool.QueryRow(ctx, `SELECT owner_generation FROM tap_source_records WHERE uri=$1`, uri).Scan(&generation); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT state FROM tap_projection_jobs WHERE source_uri=$1`, uri).Scan(&state); err != nil {
			t.Fatal(err)
		}
		if generation != wantGeneration || state != wantJob {
			t.Fatalf("source generation/job=%d/%s, want %d/%s", generation, state, wantGeneration, wantJob)
		}
	}
	assertState(1, "blocked")
	apply("56 up", up56)
	assertState(2, "pending")
	apply("56 second down", down56)
	assertState(1, "pending")
	if _, err := pool.Exec(ctx, `
		UPDATE tap_projection_jobs
		SET state='blocked',dependency_kind='repository_did',dependency_key=$2,
		    last_reason_code='source_order_uncertain',completed_at=NULL
		WHERE source_uri=$1
	`, uri, owner); err != nil {
		t.Fatal(err)
	}
	apply("56 second up", up56)
	assertState(2, "pending")
	apply("58 up", up58)
}

func TestDepartedUnresolvedPutUsesLeasedReadOnlyReconciliation(t *testing.T) {
	pool := lifecycleIngestionPool(t)
	now := time.Date(2026, 8, 20, 9, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, clock)
	if err != nil {
		t.Fatal(err)
	}
	store, err := ingestion.NewStore(pool, clock)
	if err != nil {
		t.Fatal(err)
	}
	service := newLifecycleIngestionService(t, store, lifecycles, nil, nil)
	owner := syntax.DID("did:plc:effect-read-worker")
	departed, err := lifecycles.EnsureOnboardingOwner(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	active, err := lifecycles.Transition(context.Background(), ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: departed.Generation,
		To: ownerlifecycle.StateActive, Reason: "testActivate",
	})
	if err != nil {
		t.Fatal(err)
	}
	collection := syntax.NSID("social.craftsky.feed.post")
	rkey := syntax.RecordKey("3lreadworker")
	uri := syntax.ATURI("at://did:plc:effect-read-worker/social.craftsky.feed.post/3lreadworker")
	record := json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"accepted before crash"}`)
	seedDispatchedPutAttempt(
		t, lifecycles, now, active, uri, collection, rkey, "read-worker-put", record,
	)
	if _, err := lifecycles.TransitionWith(context.Background(), ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: active.Generation,
		To: ownerlifecycle.StateDeparted, Reason: "testDepart",
	}, store.PDSAttemptDepartureParticipant()); err != nil {
		t.Fatal(err)
	}
	job, err := store.RepositoryJob(
		context.Background(), owner, ingestion.RepositoryJobPDSReconcile,
	)
	if err != nil || job.State != "pending" {
		t.Fatalf("durable effect reconciliation job=%+v err=%v", job, err)
	}

	// The remote deadline has passed before a read-only worker may classify the
	// unknown result. The worker has no Put/Delete capability by construction.
	now = now.Add(2 * time.Minute)
	claims, err := store.ClaimRepositoryJobs(context.Background(), ingestion.RepositoryClaimRequest{
		Worker: "effect-read-worker", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 1,
	})
	if err != nil || len(claims) != 1 {
		t.Fatalf("effect reconciliation claims=%+v err=%v", claims, err)
	}
	reader := &effectAttemptRecordReader{
		cid: "bafy-read-worker-accepted",
		record: map[string]any{
			"$type": "social.craftsky.feed.post",
			"text":  "accepted before crash",
		},
	}
	if err := store.RunRepositoryJob(context.Background(), claims[0], func(
		ctx context.Context,
		claim ingestion.RepositoryClaim,
	) (string, error) {
		remaining, err := service.ReconcileUnresolvedPDSAttempts(
			ctx, claim.DID, reader, 10,
		)
		if err != nil {
			return "", err
		}
		if remaining {
			return "", errors.New("effect reconciliation page remains")
		}
		return "3m-effect-read", nil
	}); err != nil {
		t.Fatal(err)
	}
	if reader.calls != 1 {
		t.Fatalf("read-only reconciliation calls=%d, want 1", reader.calls)
	}
	attempt, err := lifecycles.GetEffectAttempt(context.Background(), "read-worker-put")
	if err != nil {
		t.Fatal(err)
	}
	if attempt.Outcome != ownerlifecycle.OutcomeReconciledAccepted ||
		attempt.ProjectionDisposition != ownerlifecycle.ProjectionHiddenNonActive ||
		attempt.ResultCID != reader.cid || !attempt.RepeatForbidden {
		t.Fatalf("read-reconciled departed attempt=%+v", attempt)
	}
	job, err = store.RepositoryJob(
		context.Background(), owner, ingestion.RepositoryJobPDSReconcile,
	)
	if err != nil || job.State != "complete" || job.AuthoritativeRevision != "3m-effect-read" {
		t.Fatalf("completed effect reconciliation job=%+v err=%v", job, err)
	}
}

func TestReadOnlyWorkerDoesNotExchangeAtoBtoAAttempts(t *testing.T) {
	pool := lifecycleIngestionPool(t)
	now := time.Date(2026, 8, 20, 9, 15, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, clock)
	if err != nil {
		t.Fatal(err)
	}
	store, err := ingestion.NewStore(pool, clock)
	if err != nil {
		t.Fatal(err)
	}
	service := newLifecycleIngestionService(t, store, lifecycles, nil, nil)
	owner := syntax.DID("did:plc:effect-read-aba")
	departed, err := lifecycles.EnsureOnboardingOwner(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	active, err := lifecycles.Transition(context.Background(), ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: departed.Generation,
		To: ownerlifecycle.StateActive, Reason: "testActivate",
	})
	if err != nil {
		t.Fatal(err)
	}
	collection := syntax.NSID("social.craftsky.feed.post")
	rkey := syntax.RecordKey("3lreadaba")
	uri := syntax.ATURI("at://did:plc:effect-read-aba/social.craftsky.feed.post/3lreadaba")
	recordA := json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"A"}`)
	recordB := json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"B"}`)
	seedDispatchedPutAttempt(t, lifecycles, now, active, uri, collection, rkey, "read-aba-a1", recordA)
	seedDispatchedPutAttempt(t, lifecycles, now, active, uri, collection, rkey, "read-aba-b1", recordB)
	seedDispatchedPutAttempt(t, lifecycles, now, active, uri, collection, rkey, "read-aba-a2", recordA)
	if _, err := lifecycles.Transition(context.Background(), ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: active.Generation,
		To: ownerlifecycle.StateDeparted, Reason: "testDepart",
	}); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Minute)
	reader := &effectAttemptRecordReader{
		cid: "bafy-read-aba-current",
		record: map[string]any{
			"$type": "social.craftsky.feed.post",
			"text":  "A",
		},
	}
	remaining, err := service.ReconcileUnresolvedPDSAttempts(
		context.Background(), owner, reader, 10,
	)
	if err != nil || !remaining {
		t.Fatalf("A/B/A reconciliation remaining=%t err=%v", remaining, err)
	}
	if reader.calls != 1 {
		t.Fatalf("A/B/A authoritative reads=%d, want one per URI", reader.calls)
	}
	for operationID := range map[string]struct{}{
		"read-aba-a1": {},
		"read-aba-b1": {},
		"read-aba-a2": {},
	} {
		attempt, err := lifecycles.GetEffectAttempt(context.Background(), operationID)
		if err != nil {
			t.Fatal(err)
		}
		if attempt.Outcome != ownerlifecycle.OutcomeUnknownPreTransition ||
			attempt.ProjectionDisposition != ownerlifecycle.ProjectionHiddenNonActive {
			t.Fatalf("ambiguous attempt %s was classified: %+v", operationID, attempt)
		}
	}
}

type effectAttemptRecordReader struct {
	cid    string
	record map[string]any
	calls  int
}

func (reader *effectAttemptRecordReader) GetRecord(
	_ context.Context,
	_ syntax.DID,
	_ string,
	_ string,
	destination any,
) (string, error) {
	reader.calls++
	target, ok := destination.(*map[string]any)
	if !ok {
		return "", errors.New("unexpected reconciliation destination")
	}
	*target = reader.record
	return reader.cid, nil
}

func TestProvedNotAcceptedAndTerminalEffectSourcesNeverProject(t *testing.T) {
	for _, test := range []struct {
		name        string
		terminal    bool
		wantReason  tap.ReasonCode
		wantSource  string
		wantAttempt ownerlifecycle.ProjectionDisposition
	}{
		{
			name: "proved not accepted", wantReason: tap.ReasonStaleSource,
			wantSource: "not_accepted", wantAttempt: ownerlifecycle.ProjectionNotApplicable,
		},
		{
			name: "terminal overrides proved not accepted", terminal: true,
			wantReason: tap.ReasonOwnerTerminal, wantSource: "denied_terminal",
			wantAttempt: ownerlifecycle.ProjectionDeniedTerminal,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			pool := lifecycleIngestionPool(t)
			now := time.Date(2026, 8, 20, 9, 30, 0, 0, time.UTC)
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
			service := newLifecycleIngestionService(t, store, lifecycles, nil, nil)
			owner := syntax.DID("did:plc:effect-never-project")
			departed, err := lifecycles.EnsureOnboardingOwner(context.Background(), owner)
			if err != nil {
				t.Fatal(err)
			}
			active, err := lifecycles.Transition(context.Background(), ownerlifecycle.TransitionRequest{
				Owner: owner, ExpectedGeneration: departed.Generation,
				To: ownerlifecycle.StateActive, Reason: "testActivate",
			})
			if err != nil {
				t.Fatal(err)
			}
			collection := syntax.NSID("social.craftsky.feed.post")
			rkey := syntax.RecordKey("3lneverproject")
			uri := syntax.ATURI("at://did:plc:effect-never-project/social.craftsky.feed.post/3lneverproject")
			record := json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"must not project"}`)
			seedDispatchedPutAttempt(
				t, lifecycles, now, active, uri, collection, rkey, "never-project-put", record,
			)
			if _, err := lifecycles.ReconcileEffectAttempt(
				context.Background(),
				ownerlifecycle.ReconcileEffectRequest{
					OperationID: "never-project-put", Owner: owner,
					Outcome: ownerlifecycle.OutcomeReconciledNotAccepted,
				},
			); err != nil {
				t.Fatal(err)
			}
			if test.terminal {
				if _, err := lifecycles.Terminalize(context.Background(), ownerlifecycle.TerminalizeRequest{
					Owner: owner, Reason: "testTerminal",
				}); err != nil {
					t.Fatal(err)
				}
			}

			event := tap.Event{
				ID: 45, URI: uri, DID: owner, Collection: collection, Rkey: rkey,
				Rev: "3aaaaaaaaaaa2", CID: "bafy-never-project",
				Action: "create", Record: record,
			}
			outcome, err := service.IngestRecord(context.Background(), event)
			if err != nil || outcome.Kind != tap.OutcomePermanentInvalid || outcome.Reason != test.wantReason {
				t.Fatalf("denied source outcome=%+v err=%v", outcome, err)
			}
			source, err := store.Source(context.Background(), uri)
			if err != nil {
				t.Fatal(err)
			}
			attempt, err := lifecycles.GetEffectAttempt(context.Background(), "never-project-put")
			if err != nil {
				t.Fatal(err)
			}
			job, err := store.ProjectionJob(context.Background(), uri)
			if err != nil {
				t.Fatal(err)
			}
			if source.EffectOperationID != attempt.OperationID ||
				source.ProjectionDisposition != test.wantSource ||
				attempt.ProjectionDisposition != test.wantAttempt || job.State != "permanent_denied" {
				t.Fatalf("denied source/attempt/job=%+v/%+v/%+v", source, attempt, job)
			}
			claims, err := store.ClaimProjectionJobs(context.Background(), ingestion.ProjectionClaimRequest{
				Worker: "never-project", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 10,
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(claims) != 0 {
				t.Fatalf("denied effect source became claimable: %+v", claims)
			}
		})
	}
}

func TestOutcomeUnknownPutStaysHiddenUntilAuthoritativeRejoinReconciliation(t *testing.T) {
	pool := lifecycleIngestionPool(t)
	now := time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC)
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
	service := newLifecycleIngestionService(t, store, lifecycles, nil, nil)
	owner := syntax.DID("did:plc:effect-rejoin")
	initial, err := lifecycles.EnsureOnboardingOwner(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	active, err := lifecycles.Transition(context.Background(), ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: initial.Generation,
		To: ownerlifecycle.StateActive, Reason: "testActivate",
	})
	if err != nil {
		t.Fatal(err)
	}

	collection := syntax.NSID("social.craftsky.feed.post")
	rkey := syntax.RecordKey("3lunknownrejoin")
	uri := syntax.ATURI("at://did:plc:effect-rejoin/social.craftsky.feed.post/3lunknownrejoin")
	record := json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"accepted before crash"}`)
	recordFingerprint, err := pdseffects.RecordContentFingerprint(owner, collection, rkey, record)
	if err != nil {
		t.Fatal(err)
	}
	requestFingerprint := sha256.Sum256([]byte("exact durable Put request"))
	const operationID = "post:effect-rejoin:v1"
	if err := lifecycles.WithActiveEffects(
		context.Background(),
		[]ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: active.Generation}},
		func(effectCtx context.Context) error {
			attempt, err := lifecycles.CreateEffectAttempt(effectCtx, ownerlifecycle.NewEffectAttempt{
				OperationID: operationID, MutationKey: "post:effect-rejoin:v1",
				Owner: owner, OwnerGeneration: active.Generation,
				Kind: ownerlifecycle.EffectPDSRecord, Action: ownerlifecycle.EffectActionPutRecord,
				DeterministicKey: uri.String(), RequestFingerprint: requestFingerprint,
				RecordFingerprint: recordFingerprint, RemoteDeadline: now.Add(time.Minute),
			})
			if err != nil {
				return err
			}
			_, err = lifecycles.MarkAttemptDispatched(
				effectCtx, attempt.OperationID, owner, active.Generation,
			)
			return err
		},
	); err != nil {
		t.Fatal(err)
	}
	departed, err := lifecycles.Transition(context.Background(), ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: active.Generation,
		To: ownerlifecycle.StateDeparted, Reason: "testDepart",
	})
	if err != nil {
		t.Fatal(err)
	}

	event := tap.Event{
		ID: 50, URI: uri, DID: owner, Collection: collection, Rkey: rkey,
		Rev: "3aaaaaaaaaaa3", CID: "bafy-accepted-before-crash",
		Action: "create", Record: record,
	}
	outcome, err := service.IngestRecord(context.Background(), event)
	if err != nil || outcome.Kind != tap.OutcomeBlocked || outcome.Reason != tap.ReasonOwnerDeparted {
		t.Fatalf("departed ingestion outcome=%+v err=%v", outcome, err)
	}
	source, err := store.Source(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	if source.EffectOperationID != operationID || source.ProjectionGeneration == nil ||
		*source.ProjectionGeneration != departed.Generation || source.ProjectionDisposition != "blocked_departed" {
		t.Fatalf("departed linked source=%+v", source)
	}
	attempt, err := lifecycles.GetEffectAttempt(context.Background(), operationID)
	if err != nil {
		t.Fatal(err)
	}
	if attempt.Outcome != ownerlifecycle.OutcomeReconciledAccepted ||
		attempt.ProjectionDisposition != ownerlifecycle.ProjectionHiddenNonActive ||
		attempt.ResultCID != event.CID.String() || !attempt.RepeatForbidden {
		t.Fatalf("Tap-reconciled attempt=%+v", attempt)
	}

	profile := tap.Event{
		ID:  51,
		URI: "at://did:plc:effect-rejoin/social.craftsky.actor.profile/self",
		DID: owner, Collection: "social.craftsky.actor.profile", Rkey: "self",
		Rev: "3aaaaaaaaaaa4", CID: "bafy-rejoin-profile", Action: "create",
		Record: json.RawMessage(`{"$type":"social.craftsky.actor.profile","displayName":"Rejoined"}`),
	}
	if outcome, err := service.IngestRecord(context.Background(), profile); err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("rejoin profile outcome=%+v err=%v", outcome, err)
	}
	rejoined, err := lifecycles.Get(context.Background(), owner)
	if err != nil || rejoined.State != ownerlifecycle.StateActive || rejoined.Generation != departed.Generation+1 {
		t.Fatalf("rejoined lifecycle=%+v err=%v", rejoined, err)
	}
	source, err = store.Source(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	job, err := store.ProjectionJob(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	if source.OrderingStatus != "uncertain" || source.ProjectionDisposition != "pending" ||
		job.State != "blocked" || job.Dependency.Kind != "repository_did" {
		t.Fatalf("pre-read rejoin source/job=%+v/%+v", source, job)
	}

	outcome, err = service.ReconcileSource(context.Background(), ingestion.ReconciledSource{
		URI: uri, DID: owner, ExpectedEventID: source.SourceEventID,
		ExpectedFingerprint: source.SourceFingerprint, Revision: source.Revision,
		CID: event.CID, Record: record, Present: true,
	})
	if err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("authoritative rejoin reconciliation outcome=%+v err=%v", outcome, err)
	}
	source, err = store.Source(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err = lifecycles.GetEffectAttempt(context.Background(), operationID)
	if err != nil {
		t.Fatal(err)
	}
	if source.EffectOperationID != operationID || source.ProjectionDisposition != "eligible" ||
		source.OrderingStatus != "authoritative" ||
		attempt.ProjectionDisposition != ownerlifecycle.ProjectionEligibleCurrent {
		t.Fatalf("authoritatively eligible source/attempt=%+v/%+v", source, attempt)
	}

	claims, err := store.ClaimProjectionJobs(context.Background(), ingestion.ProjectionClaimRequest{
		Worker: "effect-rejoin-test", LeaseToken: uuid.New(), LeaseDuration: time.Minute, Limit: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	var effectClaim *ingestion.ProjectionClaim
	for index := range claims {
		if claims[index].SourceURI == uri {
			effectClaim = &claims[index]
			break
		}
	}
	if effectClaim == nil {
		t.Fatalf("eligible effect source was not claimable: %+v", claims)
	}
	projected := false
	if err := store.Project(context.Background(), *effectClaim, func(
		context.Context, pgx.Tx, ingestion.SourceRecord,
	) (tap.Outcome, error) {
		projected = true
		return tap.Applied(), nil
	}); err != nil {
		t.Fatal(err)
	}
	if !projected {
		t.Fatal("locked eligible attempt did not reach projector")
	}
}

func TestOutcomeUnknownOnboardingProfileSourceRetainsAttemptProvenance(t *testing.T) {
	pool := lifecycleIngestionPool(t)
	now := time.Date(2026, 8, 20, 12, 0, 0, 0, time.UTC)
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
	service := newLifecycleIngestionService(t, store, lifecycles, nil, nil)
	owner := syntax.DID("did:plc:onboarding-effect")
	departed, err := lifecycles.EnsureOnboardingOwner(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	record := json.RawMessage(`{"$type":"social.craftsky.actor.profile","displayName":"New member"}`)
	collection := syntax.NSID("social.craftsky.actor.profile")
	rkey := syntax.RecordKey("self")
	uri := syntax.ATURI("at://did:plc:onboarding-effect/social.craftsky.actor.profile/self")
	recordFingerprint, err := pdseffects.RecordContentFingerprint(owner, collection, rkey, record)
	if err != nil {
		t.Fatal(err)
	}
	const operationID = "oauth-onboarding-profile:did:plc:onboarding-effect:1"
	if err := lifecycles.WithExistingAuth(context.Background(), owner, func(
		authCtx context.Context,
		authority ownerlifecycle.Lifecycle,
	) error {
		return lifecycles.WithOnboardingEffect(
			authCtx, owner, authority.Generation, func(effectCtx context.Context) error {
				attempt, err := lifecycles.CreateEffectAttempt(effectCtx, ownerlifecycle.NewEffectAttempt{
					OperationID: operationID, MutationKey: operationID,
					Owner: owner, OwnerGeneration: authority.Generation,
					Kind: ownerlifecycle.EffectPDSRecord, Action: ownerlifecycle.EffectActionPutRecord,
					DeterministicKey:   uri.String(),
					RequestFingerprint: sha256.Sum256([]byte("onboarding profile request")),
					RecordFingerprint:  recordFingerprint, RemoteDeadline: now.Add(time.Minute),
				})
				if err != nil {
					return err
				}
				_, err = lifecycles.MarkAttemptDispatched(
					effectCtx, attempt.OperationID, owner, authority.Generation,
				)
				return err
			},
		)
	}); err != nil {
		t.Fatal(err)
	}

	event := tap.Event{
		ID: 60, URI: uri, DID: owner, Collection: collection, Rkey: rkey,
		Rev: "3aaaaaaaaaaa5", CID: "bafy-onboarding-profile",
		Action: "create", Record: record,
	}
	if outcome, err := service.IngestRecord(context.Background(), event); err != nil ||
		outcome.Kind != tap.OutcomeBlocked || outcome.Reason != tap.ReasonSourceOrderUncertain {
		t.Fatalf("onboarding profile ingestion outcome=%+v err=%v", outcome, err)
	}
	active, err := lifecycles.Get(context.Background(), owner)
	if err != nil || active.State != ownerlifecycle.StateActive || active.Generation != departed.Generation+1 {
		t.Fatalf("onboarding lifecycle=%+v err=%v", active, err)
	}
	source, err := store.Source(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	job, err := store.ProjectionJob(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	if source.EffectOperationID != operationID || source.ProjectionGeneration == nil ||
		*source.ProjectionGeneration != active.Generation || source.OrderingStatus != "uncertain" ||
		source.ProjectionDisposition != "pending" || job.State != "blocked" ||
		job.Dependency.Kind != "repository_did" {
		t.Fatalf("read-gated onboarding source/job=%+v/%+v", source, job)
	}
	attempt, err := lifecycles.GetEffectAttempt(context.Background(), operationID)
	if err != nil {
		t.Fatal(err)
	}
	if attempt.Outcome != ownerlifecycle.OutcomeReconciledAccepted ||
		attempt.ProjectionDisposition != ownerlifecycle.ProjectionHiddenNonActive ||
		attempt.ResultCID != event.CID.String() || !attempt.RepeatForbidden {
		t.Fatalf("onboarding source attempt=%+v", attempt)
	}

	outcome, err := service.ReconcileSource(context.Background(), ingestion.ReconciledSource{
		URI: uri, DID: owner, ExpectedEventID: source.SourceEventID,
		ExpectedFingerprint: source.SourceFingerprint, Revision: event.Rev,
		CID: event.CID, Record: record, Present: true,
	})
	if err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("authoritative onboarding reconciliation outcome=%+v err=%v", outcome, err)
	}
	source, err = store.Source(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	job, err = store.ProjectionJob(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	if source.ProjectionGeneration == nil || *source.ProjectionGeneration != active.Generation ||
		source.OrderingStatus != "authoritative" || source.ProjectionDisposition != "eligible" ||
		job.State != "pending" {
		t.Fatalf("authoritative onboarding source/job=%+v/%+v", source, job)
	}
}

func TestLiveCIDMismatchCannotBorrowAcceptedPutWithoutAuthoritativeRead(t *testing.T) {
	pool := lifecycleIngestionPool(t)
	now := time.Date(2026, 8, 20, 13, 0, 0, 0, time.UTC)
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
	service := newLifecycleIngestionService(t, store, lifecycles, nil, nil)
	owner := syntax.DID("did:plc:effect-cid-mismatch")
	departed, err := lifecycles.EnsureOnboardingOwner(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	active, err := lifecycles.Transition(context.Background(), ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: departed.Generation,
		To: ownerlifecycle.StateActive, Reason: "testActivate",
	})
	if err != nil {
		t.Fatal(err)
	}
	collection := syntax.NSID("social.craftsky.feed.post")
	rkey := syntax.RecordKey("3lcidmismatch")
	uri := syntax.ATURI("at://did:plc:effect-cid-mismatch/social.craftsky.feed.post/3lcidmismatch")
	record := json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"same content rewritten"}`)
	seedDispatchedPutAttempt(
		t, lifecycles, now, active, uri, collection, rkey, "cid-mismatch-put", record,
	)
	if err := lifecycles.WithActiveEffects(
		context.Background(),
		[]ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: active.Generation}},
		func(effectCtx context.Context) error {
			_, err := lifecycles.CompleteEffectAttempt(
				effectCtx, "cid-mismatch-put", owner, active.Generation,
				ownerlifecycle.OutcomeAccepted, "bafy-original-result",
			)
			return err
		},
	); err != nil {
		t.Fatal(err)
	}

	event := tap.Event{
		ID: 65, URI: uri, DID: owner, Collection: collection, Rkey: rkey,
		Rev: "3aaaaaaaaaaa6", CID: "bafy-current-rewrite",
		Action: "create", Record: record,
	}
	outcome, err := service.IngestRecord(context.Background(), event)
	if err != nil || outcome.Kind != tap.OutcomeBlocked || outcome.Reason != tap.ReasonSourceOrderUncertain {
		t.Fatalf("live CID mismatch outcome=%+v err=%v", outcome, err)
	}
	source, err := store.Source(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	if source.EffectOperationID != "" || source.OrderingStatus != "uncertain" ||
		source.ProjectionDisposition != "pending" {
		t.Fatalf("read-gated CID mismatch source=%+v", source)
	}
	outcome, err = service.ReconcileSource(context.Background(), ingestion.ReconciledSource{
		URI: uri, DID: owner, ExpectedEventID: source.SourceEventID,
		ExpectedFingerprint: source.SourceFingerprint, Revision: source.Revision,
		CID: event.CID, Record: record, Present: true,
	})
	if err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("authoritative CID mismatch outcome=%+v err=%v", outcome, err)
	}
	source, err = store.Source(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := lifecycles.GetEffectAttempt(context.Background(), "cid-mismatch-put")
	if err != nil {
		t.Fatal(err)
	}
	if source.EffectOperationID != "" || source.OrderingStatus != "authoritative" ||
		source.ProjectionDisposition != "eligible" ||
		attempt.Outcome != ownerlifecycle.OutcomeReconciliationMismatch ||
		attempt.ProjectionDisposition != ownerlifecycle.ProjectionNotApplicable {
		t.Fatalf("authoritative CID mismatch source/attempt=%+v/%+v", source, attempt)
	}
}

func TestAuthoritativeUserUpdateCannotExchangeAtoBtoAAttempts(t *testing.T) {
	pool := lifecycleIngestionPool(t)
	now := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
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
	service := newLifecycleIngestionService(t, store, lifecycles, nil, nil)
	owner := syntax.DID("did:plc:effect-aba")
	departed, err := lifecycles.EnsureOnboardingOwner(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	active, err := lifecycles.Transition(context.Background(), ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: departed.Generation,
		To: ownerlifecycle.StateActive, Reason: "testActivate",
	})
	if err != nil {
		t.Fatal(err)
	}
	collection := syntax.NSID("social.craftsky.feed.post")
	rkey := syntax.RecordKey("3lprovenanceaba")
	uri := syntax.ATURI("at://did:plc:effect-aba/social.craftsky.feed.post/3lprovenanceaba")
	recordA := json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"A"}`)
	recordB := json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"B"}`)
	seedDispatchedPutAttempt(t, lifecycles, now, active, uri, collection, rkey, "aba-a1", recordA)
	seedDispatchedPutAttempt(t, lifecycles, now, active, uri, collection, rkey, "aba-b1", recordB)
	seedDispatchedPutAttempt(t, lifecycles, now, active, uri, collection, rkey, "aba-a2", recordA)

	eventA := tap.Event{
		ID: 70, URI: uri, DID: owner, Collection: collection, Rkey: rkey,
		Rev: "3aaaaaaaaaaa7", CID: "bafy-a", Action: "update", Record: recordA,
	}
	if outcome, err := service.IngestRecord(context.Background(), eventA); err != nil ||
		outcome.Kind != tap.OutcomeBlocked || outcome.Reason != tap.ReasonSourceOrderUncertain {
		t.Fatalf("ambiguous A ingestion outcome=%+v err=%v", outcome, err)
	}
	sourceA, err := store.Source(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := service.ReconcileSource(context.Background(), ingestion.ReconciledSource{
		URI: uri, DID: owner, ExpectedEventID: sourceA.SourceEventID,
		ExpectedFingerprint: sourceA.SourceFingerprint, Revision: sourceA.Revision,
		CID: eventA.CID, Record: recordA, Present: true,
	})
	if !errors.Is(err, ownerlifecycle.ErrEffectSourceAmbiguous) ||
		outcome.Kind != tap.OutcomeRetryable || outcome.Reason != tap.ReasonSourceOrderUncertain {
		t.Fatalf("authoritative A reconciliation outcome=%+v err=%v", outcome, err)
	}
	sourceA, err = store.Source(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	if sourceA.EffectOperationID != "" || sourceA.OrderingStatus != "uncertain" ||
		sourceA.ProjectionDisposition != "pending" {
		t.Fatalf("ambiguous authoritative A source=%+v", sourceA)
	}

	eventB := tap.Event{
		ID: 71, URI: uri, DID: owner, Collection: collection, Rkey: rkey,
		Rev: "3aaaaaaaaaaae", CID: "bafy-user-b", Action: "update", Record: recordB,
	}
	if outcome, err := service.IngestRecord(context.Background(), eventB); err != nil ||
		outcome.Kind != tap.OutcomeBlocked || outcome.Reason != tap.ReasonSourceOrderUncertain {
		t.Fatalf("user B ingestion outcome=%+v err=%v", outcome, err)
	}
	sourceB, err := store.Source(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	if sourceB.EffectOperationID != "" {
		t.Fatalf("unreconciled user B borrowed operation %q", sourceB.EffectOperationID)
	}
	now = now.Add(2 * time.Minute)
	outcome, err = service.ReconcileSource(context.Background(), ingestion.ReconciledSource{
		URI: uri, DID: owner, ExpectedEventID: sourceB.SourceEventID,
		ExpectedFingerprint: sourceB.SourceFingerprint, Revision: sourceB.Revision,
		CID: eventB.CID, Record: recordB, Present: true,
	})
	if err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("authoritative user B reconciliation outcome=%+v err=%v", outcome, err)
	}
	sourceB, err = store.Source(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	if sourceB.EffectOperationID != "" || sourceB.ProjectionDisposition != "eligible" ||
		sourceB.OrderingStatus != "authoritative" || sourceB.ProjectionGeneration == nil ||
		*sourceB.ProjectionGeneration != active.Generation {
		t.Fatalf("authoritative user B source=%+v", sourceB)
	}
	for _, operationID := range []string{"aba-a1", "aba-b1", "aba-a2"} {
		attempt, err := lifecycles.GetEffectAttempt(context.Background(), operationID)
		if err != nil {
			t.Fatal(err)
		}
		if attempt.Outcome != ownerlifecycle.OutcomeReconciliationMismatch ||
			attempt.ProjectionDisposition != ownerlifecycle.ProjectionNotApplicable {
			t.Fatalf("conflicting attempt %s=%+v", operationID, attempt)
		}
	}

	if outcome, err := service.IngestRecord(context.Background(), eventA); err != nil || outcome.Kind != tap.OutcomeApplied {
		t.Fatalf("stale A replay outcome=%+v err=%v", outcome, err)
	}
	afterStale, err := store.Source(context.Background(), uri)
	if err != nil {
		t.Fatal(err)
	}
	if afterStale.SourceEventID != eventB.ID || afterStale.EffectOperationID != "" || afterStale.CID != eventB.CID {
		t.Fatalf("stale A exchanged authoritative user source=%+v", afterStale)
	}
}

func seedDispatchedPutAttempt(
	t *testing.T,
	lifecycles *ownerlifecycle.Store,
	now time.Time,
	authority ownerlifecycle.Lifecycle,
	uri syntax.ATURI,
	collection syntax.NSID,
	rkey syntax.RecordKey,
	operationID string,
	record json.RawMessage,
) {
	t.Helper()
	fingerprint, err := pdseffects.RecordContentFingerprint(
		authority.Owner, collection, rkey, record,
	)
	if err != nil {
		t.Fatal(err)
	}
	err = lifecycles.WithActiveEffects(
		context.Background(),
		[]ownerlifecycle.ExpectedOwner{{Owner: authority.Owner, Generation: authority.Generation}},
		func(effectCtx context.Context) error {
			attempt, err := lifecycles.CreateEffectAttempt(effectCtx, ownerlifecycle.NewEffectAttempt{
				OperationID: operationID, MutationKey: operationID,
				Owner: authority.Owner, OwnerGeneration: authority.Generation,
				Kind: ownerlifecycle.EffectPDSRecord, Action: ownerlifecycle.EffectActionPutRecord,
				DeterministicKey:   uri.String(),
				RequestFingerprint: sha256.Sum256([]byte("request:" + operationID)),
				RecordFingerprint:  fingerprint, RemoteDeadline: now.Add(time.Minute),
			})
			if err != nil {
				return err
			}
			_, err = lifecycles.MarkAttemptDispatched(
				effectCtx, attempt.OperationID, authority.Owner, authority.Generation,
			)
			return err
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}
