package pdscommands

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

func TestPrepareOrReplaySerializesScopedCommandIdentity(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 23, 19, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:command-owner")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
		) VALUES($1,'active',4,1,'test',$2,$2,$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(StoreConfig{Pool: pool, Now: func() time.Time { return now }, NewUUID: uuid.New})
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := RequestFingerprint(RequestFingerprintInput{
		AlgorithmVersion: 1, Owner: owner, OperationKind: "like",
		Intent: json.RawMessage(`{"subject":"post"}`), Conditions: json.RawMessage(`{"generation":4}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	request := PrepareRequest{
		Owner: owner, OwnerGeneration: 4, OperationKind: "like",
		OperationKey:       uuid.MustParse("018f4d5c-7a61-7d40-a1a2-0123456789ab"),
		FingerprintVersion: 1, Fingerprint: fingerprint,
		ImmutableRequest:        json.RawMessage(`{"subject":"post"}`),
		SelectedURI:             "at://did:plc:command-owner/social.craftsky.feed.like/3aaaaaaaaaaa1",
		SelectedRkey:            "3aaaaaaaaaaa1",
		ExpectedOwner:           owner,
		ExpectedOwnerGeneration: 4,
		ExpectedTarget:          "did:plc:command-target",
	}

	const callers = 8
	results := make(chan PreparedCommand, callers)
	errorsCh := make(chan error, callers)
	var wait sync.WaitGroup
	for range callers {
		wait.Add(1)
		go func() {
			defer wait.Done()
			prepared, err := store.PrepareOrReplay(context.Background(), request)
			results <- prepared
			errorsCh <- err
		}()
	}
	wait.Wait()
	close(results)
	close(errorsCh)
	var commandID uuid.UUID
	for err := range errorsCh {
		if err != nil {
			t.Fatalf("concurrent prepare: %v", err)
		}
	}
	for result := range results {
		if commandID == uuid.Nil {
			commandID = result.ID
		}
		if result.ID != commandID || result.State != CommandPrepared || result.SelectedURI != request.SelectedURI ||
			result.SelectedRkey != request.SelectedRkey || !result.CreatedAt.Equal(now) || result.ExpectedOwner != owner ||
			result.ExpectedOwnerGeneration != 4 || result.ExpectedTarget != request.ExpectedTarget {
			t.Fatalf("prepared command = %+v, want shared id %s in prepared", result, commandID)
		}
	}
	var commandCount, dispatchCount int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM pds_commands`).Scan(&commandCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM pds_command_dispatches`).Scan(&dispatchCount); err != nil {
		t.Fatal(err)
	}
	if commandCount != 1 || dispatchCount != 0 {
		t.Fatalf("command/dispatch counts = %d/%d, want 1/0", commandCount, dispatchCount)
	}

	changed := request
	changed.OwnerGeneration++
	if _, err := store.PrepareOrReplay(context.Background(), changed); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed generation error = %v, want idempotency conflict", err)
	}
	changed = request
	changed.Fingerprint[0] ^= 0xff
	if _, err := store.PrepareOrReplay(context.Background(), changed); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed fingerprint error = %v, want idempotency conflict", err)
	}
}

func TestPrepareOrReplayRejectsTerminalOwnerWithoutCreatingState(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 25, 17, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:terminal-command-owner")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,terminal_at,created_at,updated_at
		) VALUES($1,'terminal',2,1,'account_deleted',$2,$2,$2,$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(StoreConfig{Pool: pool, Lifecycles: lifecycles, Now: func() time.Time { return now }, NewUUID: uuid.New})
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := RequestFingerprint(RequestFingerprintInput{
		AlgorithmVersion: 1,
		Owner:            owner,
		OperationKind:    "post_create",
		Intent:           json.RawMessage(`{"text":"must not persist"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.PrepareOrReplay(ctx, PrepareRequest{
		Owner: owner, OwnerGeneration: 2, OperationKind: "post_create", OperationKey: uuid.New(),
		FingerprintVersion: 1, Fingerprint: fingerprint, ImmutableRequest: json.RawMessage(`{"text":"must not persist"}`),
	})
	if !errors.Is(err, ownerlifecycle.ErrOwnerNotActive) {
		t.Fatalf("prepare terminal owner error = %v, want inactive owner", err)
	}
	var commandCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM pds_commands WHERE owner_did=$1`, owner).Scan(&commandCount); err != nil {
		t.Fatal(err)
	}
	if commandCount != 0 {
		t.Fatalf("terminal owner command count = %d, want 0", commandCount)
	}
}

func TestInvalidSwapDispatchCanPersistAReplacementPlan(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 23, 19, 30, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:swap-owner")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',$2,$2,$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(StoreConfig{Pool: pool, Now: func() time.Time { return now }, NewUUID: uuid.New})
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := RequestFingerprint(RequestFingerprintInput{
		AlgorithmVersion: 1, Owner: owner, OperationKind: "unlike",
		Intent: json.RawMessage(`{"subject":"at://did:plc:target/social.craftsky.feed.post/one"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	command, err := store.PrepareOrReplay(context.Background(), PrepareRequest{
		Owner: owner, OwnerGeneration: 1, OperationKind: "unlike", OperationKey: uuid.New(),
		FingerprintVersion: 1, Fingerprint: fingerprint, ImmutableRequest: json.RawMessage(`{"subject":"post"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	step := DispatchStep{Action: "delete", URI: "at://did:plc:swap-owner/social.craftsky.feed.like/one", ExpectedCID: "bafy-one"}
	first, err := store.BeginDispatch(context.Background(), command.ID, ExactPlan{
		Version: 1, FingerprintVersion: 1, Fingerprint: fingerprint, RepositoryCID: "bafy-head-one", Steps: []DispatchStep{step},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.CompleteDispatch(context.Background(), first.ID, DispatchCompletion{Outcome: DispatchInvalidSwap}); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkOpenDispatchAmbiguous(context.Background(), command.ID, 1, "recovery_required"); err != nil {
		t.Fatalf("recover completed invalid-swap dispatch: %v", err)
	}
	recovery, err := store.Result(context.Background(), command.ID)
	if err != nil || recovery.State != CommandAmbiguous {
		t.Fatalf("invalid-swap recovery = %+v err=%v", recovery, err)
	}
	step.ExpectedCID = "bafy-two"
	second, err := store.BeginDispatch(context.Background(), command.ID, ExactPlan{
		Version: 2, FingerprintVersion: 1, Fingerprint: fingerprint, RepositoryCID: "bafy-head-two", Steps: []DispatchStep{step},
	})
	if err != nil {
		t.Fatal(err)
	}
	if second.AttemptOrdinal != 2 || second.PlanVersion != 2 {
		t.Fatalf("replacement attempt = %+v", second)
	}
	var state, firstOutcome string
	if err := pool.QueryRow(context.Background(), `SELECT state FROM pds_commands WHERE id=$1`, command.ID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT outcome FROM pds_command_dispatches WHERE id=$1`, first.ID).Scan(&firstOutcome); err != nil {
		t.Fatal(err)
	}
	if state != "dispatching" || firstOutcome != "invalid_swap" {
		t.Fatalf("replacement durability state/outcome = %s/%s", state, firstOutcome)
	}
}

func TestDispatchIntentAndTerminalOutcomeAreDurableAndImmutable(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 23, 20, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:dispatch-owner")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
		) VALUES($1,'active',2,1,'test',$2,$2,$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(StoreConfig{Pool: pool, Now: func() time.Time { return now }, NewUUID: uuid.New})
	if err != nil {
		t.Fatal(err)
	}
	requestFingerprint, err := RequestFingerprint(RequestFingerprintInput{
		AlgorithmVersion: 1, Owner: owner, OperationKind: "remove_likes",
		Intent: json.RawMessage(`{"subject":"post"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	request := PrepareRequest{
		Owner: owner, OwnerGeneration: 2, OperationKind: "remove_likes",
		OperationKey:       uuid.MustParse("018f4d5c-7a61-7d40-a1a2-abcdefabcdef"),
		FingerprintVersion: 1, Fingerprint: requestFingerprint,
		ImmutableRequest: json.RawMessage(`{"subject":"post"}`),
	}
	command, err := store.PrepareOrReplay(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	steps := []DispatchStep{
		{Action: "delete", URI: "at://did:plc:dispatch-owner/social.craftsky.feed.like/3aaaaaaaaaaa2", ExpectedCID: "bafy-one"},
		{Action: "delete", URI: "at://did:plc:dispatch-owner/social.craftsky.feed.like/3aaaaaaaaaaa3", ExpectedCID: "bafy-two"},
	}
	dispatchFingerprint, err := DispatchFingerprint(DispatchFingerprintInput{
		AlgorithmVersion: 1, RequestFingerprint: requestFingerprint,
		RepositoryHead: "bafy-head", Steps: steps,
	})
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := store.BeginDispatch(context.Background(), command.ID, ExactPlan{
		Version: 1, FingerprintVersion: 1, Fingerprint: dispatchFingerprint,
		RepositoryCID: "bafy-head", RepositoryRevision: "3aaaaaaaaaaa4", Steps: steps,
	})
	if err != nil {
		t.Fatal(err)
	}
	var state string
	var stepCount, dispatchCount int
	if err := pool.QueryRow(context.Background(), `SELECT state FROM pds_commands WHERE id=$1`, command.ID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM pds_command_steps WHERE command_id=$1`, command.ID).Scan(&stepCount); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM pds_command_dispatches WHERE command_id=$1`, command.ID).Scan(&dispatchCount); err != nil {
		t.Fatal(err)
	}
	if state != "dispatching" || stepCount != 2 || dispatchCount != 1 {
		t.Fatalf("durable dispatch boundary = state:%s steps:%d dispatches:%d", state, stepCount, dispatchCount)
	}

	if err := store.CompleteDispatch(context.Background(), attempt.ID, DispatchCompletion{Outcome: CommandAmbiguous, RetryAfterSeconds: 3}); err != nil {
		t.Fatal(err)
	}
	replayed, err := store.PrepareOrReplay(context.Background(), request)
	if err != nil || replayed.State != CommandAmbiguous || !replayed.Replay {
		t.Fatalf("ambiguous replay = %+v err=%v", replayed, err)
	}

	now = now.Add(time.Minute)
	if err := store.CompleteCommand(context.Background(), command.ID, TerminalResult{
		State: CommandAccepted, HTTPStatus: 200,
		ResponseBody:    json.RawMessage(`{"status":"ok"}`),
		ResponseHeaders: json.RawMessage(`{"ETag":"bafy-result"}`),
	}); err != nil {
		t.Fatal(err)
	}
	var replayExpiresAt time.Time
	if err := pool.QueryRow(context.Background(), `SELECT state,replay_expires_at FROM pds_commands WHERE id=$1`, command.ID).Scan(&state, &replayExpiresAt); err != nil {
		t.Fatal(err)
	}
	if state != "accepted" || !replayExpiresAt.Equal(now.Add(ReplayRetention)) {
		t.Fatalf("terminal command state=%s replayExpiresAt=%s", state, replayExpiresAt)
	}
	result, err := store.Result(context.Background(), command.ID)
	if err != nil {
		t.Fatal(err)
	}
	var resultBody, resultHeaders map[string]string
	if err := json.Unmarshal(result.ResponseBody, &resultBody); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(result.ResponseHeaders, &resultHeaders); err != nil {
		t.Fatal(err)
	}
	if result.State != CommandAccepted || result.HTTPStatus != 200 || resultBody["status"] != "ok" || resultHeaders["ETag"] != "bafy-result" {
		t.Fatalf("stored command result = %+v", result)
	}
	if err := store.CompleteCommand(context.Background(), command.ID, TerminalResult{State: CommandRejected, HTTPStatus: 409}); !errors.Is(err, ErrIllegalTransition) {
		t.Fatalf("terminal rewrite error = %v, want illegal transition", err)
	}

	now = replayExpiresAt
	compactor, err := NewCompactionProcessor(store, 10)
	if err != nil {
		t.Fatal(err)
	}
	compacted, err := compactor.ProcessBatch(context.Background())
	if err != nil || compacted != 1 {
		t.Fatalf("compact expired commands = %d, err=%v", compacted, err)
	}
	var commands, stepsRemaining, dispatches, tombstones int
	for query, destination := range map[string]*int{
		`SELECT count(*) FROM pds_commands`:           &commands,
		`SELECT count(*) FROM pds_command_steps`:      &stepsRemaining,
		`SELECT count(*) FROM pds_command_dispatches`: &dispatches,
		`SELECT count(*) FROM pds_command_tombstones`: &tombstones,
	} {
		if err := pool.QueryRow(context.Background(), query).Scan(destination); err != nil {
			t.Fatal(err)
		}
	}
	if commands != 0 || stepsRemaining != 0 || dispatches != 0 || tombstones != 1 {
		t.Fatalf("post-compaction rows commands/steps/dispatches/tombstones=%d/%d/%d/%d", commands, stepsRemaining, dispatches, tombstones)
	}
	if _, err := store.PrepareOrReplay(context.Background(), request); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("tombstoned key reuse error = %v, want idempotency conflict", err)
	}
}

func TestDispatchAndTerminalOutcomeCompleteAtomically(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 25, 16, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:atomic-completion-owner")
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',$2,$2,$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(StoreConfig{Pool: pool, Now: func() time.Time { return now }, NewUUID: uuid.New})
	if err != nil {
		t.Fatal(err)
	}
	fingerprint, err := RequestFingerprint(RequestFingerprintInput{
		AlgorithmVersion: 1,
		Owner:            owner,
		OperationKind:    "post_create",
		Intent:           json.RawMessage(`{"text":"atomic"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	command, err := store.PrepareOrReplay(context.Background(), PrepareRequest{
		Owner: owner, OwnerGeneration: 1, OperationKind: "post_create", OperationKey: uuid.New(),
		FingerprintVersion: 1, Fingerprint: fingerprint, ImmutableRequest: json.RawMessage(`{"text":"atomic"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := store.BeginDispatch(context.Background(), command.ID, ExactPlan{
		Version: 1, FingerprintVersion: 1, Fingerprint: fingerprint,
		RepositoryCID: "bafy-head", Steps: []DispatchStep{{
			Action: "create", URI: "at://did:plc:atomic-completion-owner/social.craftsky.feed.post/3aaaaaaaaaaa1",
			Body: json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"atomic"}`),
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	terminal := TerminalResult{
		State: CommandAccepted, HTTPStatus: 201,
		ResponseBody: json.RawMessage(`{"uri":"at://did:plc:atomic-completion-owner/social.craftsky.feed.post/3aaaaaaaaaaa1"}`),
	}
	if err := store.CompleteDispatchAndCommand(
		context.Background(),
		attempt.ID,
		DispatchCompletion{Outcome: CommandAccepted, ResultCommitCID: "bafy-result"},
		terminal,
	); err != nil {
		t.Fatal(err)
	}

	result, err := store.Result(context.Background(), command.ID)
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(result.ResponseBody, &response); err != nil {
		t.Fatal(err)
	}
	if result.State != CommandAccepted || result.HTTPStatus != 201 || response.URI != "at://did:plc:atomic-completion-owner/social.craftsky.feed.post/3aaaaaaaaaaa1" {
		t.Fatalf("atomic terminal result = %+v", result)
	}
	var dispatchOutcome, commandState string
	if err := pool.QueryRow(context.Background(), `
		SELECT dispatch.outcome,command.state
		FROM pds_command_dispatches dispatch
		JOIN pds_commands command ON command.id=dispatch.command_id
		WHERE dispatch.id=$1
	`, attempt.ID).Scan(&dispatchOutcome, &commandState); err != nil {
		t.Fatal(err)
	}
	if dispatchOutcome != string(CommandAccepted) || commandState != string(CommandAccepted) {
		t.Fatalf("atomic states = dispatch:%s command:%s", dispatchOutcome, commandState)
	}
}
