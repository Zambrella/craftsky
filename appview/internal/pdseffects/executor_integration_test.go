package pdseffects

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

func TestExecutorPutPersistsBeforeIOAndReadReconcilesWithoutRepeating(t *testing.T) {
	pool, lifecycles, now := newEffectExecutorStore(t)
	owner := syntax.DID("did:plc:durable-put")
	seedEffectOwner(t, pool, owner, 2, now)
	pds := newEffectPDS()
	pds.putError = errors.New("lost put response")
	pds.acceptPutBeforeError = true
	boundary := &storeEffectBoundary{lifecycles: lifecycles, client: pds}
	executor, err := NewExecutor(lifecycles, boundary, owner, 10*time.Second, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	request := PutRecordRequest{
		OperationID: "post:durable-put:v1",
		MutationKey: "post:durable-put:v1",
		Owner:       owner, OwnerGeneration: 2,
		ExpectedOwners: []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 2}},
		Collection:     syntax.NSID("social.craftsky.feed.post"),
		Rkey:           syntax.RecordKey("3ldurableput"),
		Record: map[string]any{
			"$type": "social.craftsky.feed.post",
			"text":  "durable",
		},
	}

	_, err = executor.PutRecord(context.Background(), request)
	if !errors.Is(err, ErrOutcomeAmbiguous) {
		t.Fatalf("lost response error = %v, want ErrOutcomeAmbiguous", err)
	}
	if pds.putCalls != 1 || pds.getCalls != 0 {
		t.Fatalf("first PDS calls = put %d get %d", pds.putCalls, pds.getCalls)
	}
	var outcome ownerlifecycle.EffectOutcome
	var action ownerlifecycle.EffectAction
	var exactKey string
	if err := pool.QueryRow(context.Background(), `
		SELECT remote_outcome,effect_action,deterministic_key
		FROM owner_effect_attempts WHERE operation_id=$1
	`, request.OperationID).Scan(&outcome, &action, &exactKey); err != nil {
		t.Fatal(err)
	}
	if outcome != ownerlifecycle.OutcomeDispatched ||
		action != ownerlifecycle.EffectActionPutRecord ||
		exactKey != "at://did:plc:durable-put/social.craftsky.feed.post/3ldurableput" {
		t.Fatalf("dispatched attempt = outcome %q action %q key %q", outcome, action, exactKey)
	}

	pds.putError = nil
	result, err := executor.PutRecord(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.URI.String() != exactKey || result.CID == "" {
		t.Fatalf("reconciled result = %+v", result)
	}
	if pds.putCalls != 1 || pds.getCalls != 1 {
		t.Fatalf("reconciled PDS calls = put %d get %d, want 1/1", pds.putCalls, pds.getCalls)
	}
	if _, err := executor.PutRecord(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if pds.putCalls != 1 || pds.getCalls != 1 {
		t.Fatalf("accepted replay crossed PDS boundary: put %d get %d", pds.putCalls, pds.getCalls)
	}
}

func TestExecutorReturnsTypedConflictButAllowsNewSameURIRecordVersion(t *testing.T) {
	pool, lifecycles, now := newEffectExecutorStore(t)
	owner := syntax.DID("did:plc:versioned-profile")
	seedEffectOwner(t, pool, owner, 2, now)
	pds := newEffectPDS()
	executor, err := NewExecutor(
		lifecycles,
		&storeEffectBoundary{lifecycles: lifecycles, client: pds},
		owner,
		10*time.Second,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	request := PutRecordRequest{
		OperationID: "profile:update:v1",
		MutationKey: "profile:update:v1",
		Owner:       owner, OwnerGeneration: 2,
		ExpectedOwners: []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 2}},
		Collection:     syntax.NSID("social.craftsky.actor.profile"),
		Rkey:           syntax.RecordKey("self"),
		Record:         map[string]any{"$type": "social.craftsky.actor.profile", "displayName": "first"},
	}
	if _, err := executor.PutRecord(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	sameMutation := request
	sameMutation.OperationID = "profile:update:v1-retry"
	if _, err := executor.PutRecord(context.Background(), sameMutation); err != nil {
		t.Fatal(err)
	}
	if pds.putCalls != 1 {
		t.Fatalf("same mutation key repeated Put: calls=%d", pds.putCalls)
	}
	conflicting := sameMutation
	conflicting.OperationID = "profile:update:v1-conflict"
	conflicting.Record = map[string]any{"$type": "social.craftsky.actor.profile", "displayName": "conflict"}
	_, err = executor.PutRecord(context.Background(), conflicting)
	var conflict *ConflictError
	if !errors.Is(err, ErrEffectConflict) || !errors.As(err, &conflict) || pds.putCalls != 1 {
		t.Fatalf("same-operation conflict=%v typed=%t puts=%d", err, conflict != nil, pds.putCalls)
	}
	versioned := conflicting
	versioned.OperationID = "profile:update:v2"
	versioned.MutationKey = "profile:update:v2"
	if _, err := executor.PutRecord(context.Background(), versioned); err != nil {
		t.Fatal(err)
	}
	if pds.putCalls != 2 {
		t.Fatalf("same-URI new version puts=%d, want 2", pds.putCalls)
	}
	returned := request
	returned.OperationID = "profile:update:v3"
	returned.MutationKey = "profile:update:v3"
	if _, err := executor.PutRecord(context.Background(), returned); err != nil {
		t.Fatal(err)
	}
	if pds.putCalls != 3 {
		t.Fatalf("A-B-A same-URI puts=%d, want 3", pds.putCalls)
	}
	if _, err := executor.DeleteRecord(context.Background(), DeleteRecordRequest{
		OperationID: "profile:delete:v4",
		MutationKey: "profile:delete:v4",
		Owner:       owner, OwnerGeneration: 2,
		ExpectedOwners: request.ExpectedOwners,
		Collection:     request.Collection,
		Rkey:           request.Rkey,
	}); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(context.Background(), `
		SELECT count(*) FROM owner_effect_attempts
		WHERE owner_did=$1 AND deterministic_key=$2
	`, owner, "at://did:plc:versioned-profile/social.craftsky.actor.profile/self").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 4 {
		t.Fatalf("versioned exact-URI attempts=%d, want 4", count)
	}
}

func TestExecutorPutVerifiesAuthoritativeRecordBeforeAccepting(t *testing.T) {
	pool, lifecycles, now := newEffectExecutorStore(t)
	owner := syntax.DID("did:plc:verified-put")
	seedEffectOwner(t, pool, owner, 2, now)
	pds := newEffectPDS()
	executor, err := NewExecutor(
		lifecycles,
		&storeEffectBoundary{lifecycles: lifecycles, client: pds},
		owner,
		10*time.Second,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	request := PutRecordRequest{
		OperationID:     "post:verified-put:v1",
		MutationKey:     "post:verified-put:v1",
		Owner:           owner,
		OwnerGeneration: 2,
		ExpectedOwners:  []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 2}},
		Collection:      syntax.NSID("social.craftsky.feed.post"),
		Rkey:            syntax.RecordKey("3lverifiedput"),
		Record: map[string]any{
			"$type": "social.craftsky.feed.post",
			"text":  "verify me",
		},
	}
	pds.getError = errors.New("lost authoritative readback")
	if _, err := executor.PutRecord(context.Background(), request); !errors.Is(err, ErrOutcomeAmbiguous) {
		t.Fatalf("readback failure = %v, want ErrOutcomeAmbiguous", err)
	}
	assertEffectOutcome(t, pool, request.OperationID, ownerlifecycle.OutcomeDispatched)
	if pds.putCalls != 1 || pds.getCalls != 1 {
		t.Fatalf("first verified Put calls = put %d get %d", pds.putCalls, pds.getCalls)
	}

	pds.getError = nil
	pds.getOverride = &storedEffectRecord{
		cid: "bafy-wrong-record",
		value: map[string]any{
			"$type": "social.craftsky.feed.post",
			"text":  "different",
		},
	}
	if _, err := executor.PutRecord(context.Background(), request); !errors.Is(err, ErrEffectConflict) {
		t.Fatalf("mismatching readback = %v, want ErrEffectConflict", err)
	}
	assertEffectOutcome(t, pool, request.OperationID, ownerlifecycle.OutcomeDispatched)
	if pds.putCalls != 1 || pds.getCalls != 2 {
		t.Fatalf("mismatch repeated Put: put %d get %d", pds.putCalls, pds.getCalls)
	}

	pds.getOverride = nil
	result, err := executor.PutRecord(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.CID != "bafy-effect-record" {
		t.Fatalf("verified result CID = %q", result.CID)
	}
	assertEffectOutcome(t, pool, request.OperationID, ownerlifecycle.OutcomeAccepted)
	if pds.putCalls != 1 || pds.getCalls != 3 {
		t.Fatalf("verified retry calls = put %d get %d", pds.putCalls, pds.getCalls)
	}
}

func TestExecutorConditionalPutReconcilesNewCIDWithoutRepeating(t *testing.T) {
	pool, lifecycles, now := newEffectExecutorStore(t)
	owner := syntax.DID("did:plc:conditional-put")
	seedEffectOwner(t, pool, owner, 2, now)
	pds := newEffectPDS()
	key := effectRecordKey(owner, "social.craftsky.actor.profile", "self")
	pds.records[key] = storedEffectRecord{
		cid: "bafy-prior-profile", value: map[string]any{"displayName": "before"},
	}
	pds.putError = errors.New("lost conditional Put response")
	pds.acceptPutBeforeError = true
	executor, err := NewExecutor(
		lifecycles,
		&storeEffectBoundary{lifecycles: lifecycles, client: pds},
		owner,
		10*time.Second,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	request := PutRecordRequest{
		OperationID:     "profile:conditional-put:v1",
		MutationKey:     "profile:conditional-put:v1",
		Owner:           owner,
		OwnerGeneration: 2,
		ExpectedOwners:  []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 2}},
		Collection:      syntax.NSID("social.craftsky.actor.profile"),
		Rkey:            syntax.RecordKey("self"),
		Record:          map[string]any{"displayName": "after"},
		ExpectedCID:     syntax.CID("bafy-prior-profile"),
	}
	if _, err := executor.PutRecord(context.Background(), request); !errors.Is(
		err, ErrOutcomeAmbiguous,
	) {
		t.Fatalf("lost conditional Put = %v", err)
	}
	assertEffectOutcome(t, pool, request.OperationID, ownerlifecycle.OutcomeDispatched)
	if pds.conditionalPutCalls != 1 || pds.unconditionalPutCalls != 0 || pds.getCalls != 0 {
		t.Fatalf("conditional/unconditional/read calls = %d/%d/%d", pds.conditionalPutCalls, pds.unconditionalPutCalls, pds.getCalls)
	}
	pds.putError = nil
	result, err := executor.PutRecord(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.CID != "bafy-effect-record" || result.CID == request.ExpectedCID {
		t.Fatalf("conditional Put authoritative result = %+v", result)
	}
	if pds.conditionalPutCalls != 1 || pds.getCalls != 1 {
		t.Fatalf("conditional Put retry repeated remote write: puts=%d reads=%d", pds.conditionalPutCalls, pds.getCalls)
	}
	_, wantFingerprint, err := canonicalPutBody(
		request.Owner, request.Collection, request.Rkey, request.Record, request.ExpectedCID,
	)
	if err != nil {
		t.Fatal(err)
	}
	var storedCID string
	var storedFingerprint []byte
	if err := pool.QueryRow(context.Background(), `
		SELECT expected_cid,request_fingerprint FROM owner_effect_attempts WHERE operation_id=$1
	`, request.OperationID).Scan(&storedCID, &storedFingerprint); err != nil {
		t.Fatal(err)
	}
	if storedCID != request.ExpectedCID.String() || !bytes.Equal(storedFingerprint, wantFingerprint[:]) {
		t.Fatalf("stored conditional Put identity = %q/%x", storedCID, storedFingerprint)
	}
}

func TestExecutorConditionalPutRejectsStaleCIDAndMissingCapability(t *testing.T) {
	pool, lifecycles, now := newEffectExecutorStore(t)
	owner := syntax.DID("did:plc:stale-conditional-put")
	seedEffectOwner(t, pool, owner, 2, now)
	pds := newEffectPDS()
	key := effectRecordKey(owner, "social.craftsky.actor.profile", "self")
	pds.records[key] = storedEffectRecord{
		cid: "bafy-current-profile", value: map[string]any{"displayName": "current"},
	}
	executor, err := NewExecutor(
		lifecycles,
		&storeEffectBoundary{lifecycles: lifecycles, client: pds},
		owner,
		10*time.Second,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	request := PutRecordRequest{
		OperationID:     "profile:stale-conditional-put:v1",
		MutationKey:     "profile:stale-conditional-put:v1",
		Owner:           owner,
		OwnerGeneration: 2,
		ExpectedOwners:  []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 2}},
		Collection:      syntax.NSID("social.craftsky.actor.profile"),
		Rkey:            syntax.RecordKey("self"),
		Record:          map[string]any{"displayName": "stale update"},
		ExpectedCID:     syntax.CID("bafy-stale-profile"),
	}
	if _, err := executor.PutRecord(context.Background(), request); !errors.Is(
		err, ErrEffectConflict,
	) {
		t.Fatalf("stale conditional Put = %v", err)
	}
	assertEffectOutcome(t, pool, request.OperationID, ownerlifecycle.OutcomeRejected)
	if pds.records[key].cid != "bafy-current-profile" || pds.conditionalPutCalls != 1 {
		t.Fatalf("stale conditional Put changed current record: %+v", pds.records[key])
	}
	if _, err := executor.PutRecord(context.Background(), request); !errors.Is(err, ErrEffectConflict) {
		t.Fatalf("replayed stale conditional Put = %v", err)
	}
	if pds.conditionalPutCalls != 1 {
		t.Fatal("replayed stale conditional Put crossed remote boundary")
	}

	unsupportedRequest := request
	unsupportedRequest.OperationID = "profile:unsupported-conditional-put:v1"
	unsupportedRequest.MutationKey = unsupportedRequest.OperationID
	unsupported, err := NewExecutor(
		lifecycles,
		&storeEffectBoundary{
			lifecycles: lifecycles,
			client:     baseOnlyPDSClient{PDSClient: pds},
		},
		owner,
		10*time.Second,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := unsupported.PutRecord(
		context.Background(), unsupportedRequest,
	); !errors.Is(err, auth.ErrConditionalPutUnsupported) {
		t.Fatalf("conditional Put without capability = %v", err)
	}
	assertEffectOutcome(t, pool, unsupportedRequest.OperationID, ownerlifecycle.OutcomePrepared)
}

func TestExecutorDeletePersistsAndReadReconcilesAbsenceWithoutRepeating(t *testing.T) {
	pool, lifecycles, now := newEffectExecutorStore(t)
	owner := syntax.DID("did:plc:durable-delete")
	seedEffectOwner(t, pool, owner, 3, now)
	pds := newEffectPDS()
	key := effectRecordKey(owner, "social.craftsky.feed.post", "3ldurabledelete")
	pds.records[key] = storedEffectRecord{
		cid: "bafy-before-delete",
		value: map[string]any{
			"$type": "social.craftsky.feed.post",
			"text":  "delete me",
		},
	}
	pds.deleteError = errors.New("lost delete response")
	pds.acceptDeleteBeforeError = true
	executor, err := NewExecutor(
		lifecycles,
		&storeEffectBoundary{lifecycles: lifecycles, client: pds},
		owner,
		10*time.Second,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	request := DeleteRecordRequest{
		OperationID: "post:durable-delete:v1",
		MutationKey: "post:durable-delete:v1",
		Owner:       owner, OwnerGeneration: 3,
		ExpectedOwners: []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 3}},
		Collection:     syntax.NSID("social.craftsky.feed.post"),
		Rkey:           syntax.RecordKey("3ldurabledelete"),
		ExpectedCID:    syntax.CID("bafy-before-delete"),
	}

	_, err = executor.DeleteRecord(context.Background(), request)
	if !errors.Is(err, ErrOutcomeAmbiguous) {
		t.Fatalf("lost delete response = %v, want ErrOutcomeAmbiguous", err)
	}
	if pds.deleteCalls != 1 || pds.getCalls != 0 {
		t.Fatalf("first delete calls = delete %d get %d", pds.deleteCalls, pds.getCalls)
	}
	pds.deleteError = nil
	result, err := executor.DeleteRecord(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.URI.String() != key || result.CID != "" {
		t.Fatalf("delete result = %+v", result)
	}
	if pds.deleteCalls != 1 || pds.getCalls != 1 {
		t.Fatalf("reconciled delete calls = delete %d get %d, want 1/1", pds.deleteCalls, pds.getCalls)
	}
	var action ownerlifecycle.EffectAction
	if err := pool.QueryRow(context.Background(), `
		SELECT effect_action FROM owner_effect_attempts WHERE operation_id=$1
	`, request.OperationID).Scan(&action); err != nil {
		t.Fatal(err)
	}
	if action != ownerlifecycle.EffectActionDeleteRecord {
		t.Fatalf("delete effect action = %q", action)
	}
}

func TestExecutorDeleteRejectsStaleCIDAndPreservesCurrentRecord(t *testing.T) {
	pool, lifecycles, now := newEffectExecutorStore(t)
	owner := syntax.DID("did:plc:stale-delete")
	seedEffectOwner(t, pool, owner, 3, now)
	pds := newEffectPDS()
	key := effectRecordKey(owner, "social.craftsky.feed.post", "3lstaledelete")
	pds.records[key] = storedEffectRecord{
		cid:   "bafy-current-replacement",
		value: map[string]any{"text": "current replacement"},
	}
	executor, err := NewExecutor(
		lifecycles,
		&storeEffectBoundary{lifecycles: lifecycles, client: pds},
		owner,
		10*time.Second,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	request := DeleteRecordRequest{
		OperationID:     "post:stale-delete:v1",
		MutationKey:     "post:stale-delete:v1",
		Owner:           owner,
		OwnerGeneration: 3,
		ExpectedOwners:  []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 3}},
		Collection:      syntax.NSID("social.craftsky.feed.post"),
		Rkey:            syntax.RecordKey("3lstaledelete"),
		ExpectedCID:     syntax.CID("bafy-stale-record"),
	}
	if _, err := executor.DeleteRecord(context.Background(), request); !errors.Is(err, ErrEffectConflict) {
		t.Fatalf("stale delete = %v, want ErrEffectConflict", err)
	}
	assertEffectOutcome(t, pool, request.OperationID, ownerlifecycle.OutcomeRejected)
	if got := pds.records[key].cid; got != "bafy-current-replacement" {
		t.Fatalf("current record CID = %q after stale delete", got)
	}
	if pds.conditionalDeleteCalls != 1 || pds.unconditionalDeleteCalls != 0 {
		t.Fatalf("conditional/unconditional deletes = %d/%d", pds.conditionalDeleteCalls, pds.unconditionalDeleteCalls)
	}
	if _, err := executor.DeleteRecord(context.Background(), request); !errors.Is(err, ErrEffectConflict) {
		t.Fatalf("replayed stale delete = %v, want ErrEffectConflict", err)
	}
	if pds.conditionalDeleteCalls != 1 {
		t.Fatalf("replayed stale delete crossed remote boundary: calls=%d", pds.conditionalDeleteCalls)
	}
	var storedCID string
	var storedFingerprint []byte
	if err := pool.QueryRow(context.Background(), `
		SELECT expected_cid,request_fingerprint FROM owner_effect_attempts WHERE operation_id=$1
	`, request.OperationID).Scan(&storedCID, &storedFingerprint); err != nil {
		t.Fatal(err)
	}
	_, wantFingerprint, _, err := canonicalDeleteBody(
		request.Owner, request.Collection, request.Rkey, request.ExpectedCID,
	)
	if err != nil {
		t.Fatal(err)
	}
	if storedCID != request.ExpectedCID.String() || !bytes.Equal(storedFingerprint, wantFingerprint[:]) {
		t.Fatalf("stored delete CAS identity = cid %q fingerprint %x", storedCID, storedFingerprint)
	}
}

func TestExecutorConditionalDeletePreservesConcurrentReplacement(t *testing.T) {
	pool, lifecycles, now := newEffectExecutorStore(t)
	owner := syntax.DID("did:plc:racing-delete")
	seedEffectOwner(t, pool, owner, 3, now)
	pds := newEffectPDS()
	key := effectRecordKey(owner, "social.craftsky.feed.post", "3lracingdelete")
	pds.records[key] = storedEffectRecord{cid: "bafy-original", value: map[string]any{"text": "original"}}
	pds.replaceBeforeConditionalDelete = &storedEffectRecord{
		cid: "bafy-racing-replacement", value: map[string]any{"text": "replacement"},
	}
	executor, err := NewExecutor(
		lifecycles,
		&storeEffectBoundary{lifecycles: lifecycles, client: pds},
		owner,
		10*time.Second,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	request := DeleteRecordRequest{
		OperationID:     "post:racing-delete:v1",
		MutationKey:     "post:racing-delete:v1",
		Owner:           owner,
		OwnerGeneration: 3,
		ExpectedOwners:  []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 3}},
		Collection:      syntax.NSID("social.craftsky.feed.post"),
		Rkey:            syntax.RecordKey("3lracingdelete"),
		ExpectedCID:     syntax.CID("bafy-original"),
	}
	if _, err := executor.DeleteRecord(context.Background(), request); !errors.Is(err, ErrEffectConflict) {
		t.Fatalf("racing replacement delete = %v, want ErrEffectConflict", err)
	}
	if got := pds.records[key].cid; got != "bafy-racing-replacement" {
		t.Fatalf("racing replacement CID = %q", got)
	}
	if pds.getCalls != 0 || pds.conditionalDeleteCalls != 1 || pds.unconditionalDeleteCalls != 0 {
		t.Fatalf("read/conditional/unconditional calls = %d/%d/%d", pds.getCalls, pds.conditionalDeleteCalls, pds.unconditionalDeleteCalls)
	}
}

func TestExecutorConditionalDeleteFailsClosedWithoutSwapCapability(t *testing.T) {
	pool, lifecycles, now := newEffectExecutorStore(t)
	owner := syntax.DID("did:plc:conditional-delete-capability")
	seedEffectOwner(t, pool, owner, 3, now)
	pds := newEffectPDS()
	pds.records[effectRecordKey(owner, "social.craftsky.feed.post", "3lcapability")] =
		storedEffectRecord{cid: "bafy-current", value: map[string]any{"text": "keep"}}
	baseOnly := baseOnlyPDSClient{PDSClient: pds}
	executor, err := NewExecutor(
		lifecycles,
		&storeEffectBoundary{lifecycles: lifecycles, client: baseOnly},
		owner,
		10*time.Second,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	request := DeleteRecordRequest{
		OperationID:     "post:conditional-delete-capability:v1",
		MutationKey:     "post:conditional-delete-capability:v1",
		Owner:           owner,
		OwnerGeneration: 3,
		ExpectedOwners:  []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 3}},
		Collection:      syntax.NSID("social.craftsky.feed.post"),
		Rkey:            syntax.RecordKey("3lcapability"),
		ExpectedCID:     syntax.CID("bafy-current"),
	}
	if _, err := executor.DeleteRecord(context.Background(), request); !errors.Is(
		err, auth.ErrConditionalDeleteUnsupported,
	) {
		t.Fatalf("conditional delete without capability = %v", err)
	}
	assertEffectOutcome(t, pool, request.OperationID, ownerlifecycle.OutcomePrepared)
	if pds.deleteCalls != 0 {
		t.Fatalf("unsupported conditional delete crossed dispatch: calls=%d", pds.deleteCalls)
	}
}

func TestGuardedExecutorUsesOneOwnerSessionBoundaryForDurableEffect(t *testing.T) {
	pool, lifecycles, now := newEffectExecutorStore(t)
	owner := syntax.DID("did:plc:single-guarded-boundary")
	seedEffectOwner(t, pool, owner, 2, now)
	pds := newEffectPDS()
	boundary := &storeEffectBoundary{lifecycles: lifecycles, client: pds}
	executor, err := NewExecutor(
		lifecycles, boundary, owner, 10*time.Second, func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	expected := []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 2}}
	err = executor.WithGuardedEffects(
		context.Background(), expected,
		func(effectCtx context.Context, scoped EffectExecutor) error {
			_, putErr := scoped.PutRecord(effectCtx, PutRecordRequest{
				OperationID: "guarded-post:v1", MutationKey: "guarded-post:v1",
				Owner: owner, OwnerGeneration: 2, ExpectedOwners: expected,
				Collection: syntax.NSID("social.craftsky.feed.post"),
				Rkey:       syntax.RecordKey("3lguarded"),
				Record: map[string]any{
					"$type": "social.craftsky.feed.post", "text": "guarded",
				},
			})
			return putErr
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if boundary.calls != 1 || pds.putCalls != 1 {
		t.Fatalf("owner-session boundary/PDS calls = %d/%d, want 1/1", boundary.calls, pds.putCalls)
	}
}

func TestExecutorResolvesExactActiveTargetScopeAndRejectsKnownInactiveTargets(t *testing.T) {
	pool, lifecycles, now := newEffectExecutorStore(t)
	owner := syntax.DID("did:plc:scope-owner")
	activeTarget := syntax.DID("did:plc:scope-active-target")
	departedTarget := syntax.DID("did:plc:scope-departed-target")
	terminalTarget := syntax.DID("did:plc:scope-terminal-target")
	unknownTarget := syntax.DID("did:plc:scope-external-target")
	seedEffectOwner(t, pool, owner, 2, now)
	seedEffectOwner(t, pool, activeTarget, 4, now)
	seedEffectOwnerState(t, pool, departedTarget, ownerlifecycle.StateDeparted, 3, now)
	seedEffectOwnerState(t, pool, terminalTarget, ownerlifecycle.StateTerminal, 5, now)
	pds := newEffectPDS()
	executor, err := NewExecutor(
		lifecycles,
		&storeEffectBoundary{lifecycles: lifecycles, client: pds},
		owner,
		10*time.Second,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := executor.ResolveExpectedOwners(
		context.Background(), 2,
		[]syntax.DID{unknownTarget, activeTarget, owner, activeTarget},
	)
	if err != nil {
		t.Fatal(err)
	}
	want := []ownerlifecycle.ExpectedOwner{
		{Owner: activeTarget, Generation: 4},
		{Owner: unknownTarget, AllowMissing: true},
		{Owner: owner, Generation: 2},
	}
	if fmt.Sprint(resolved) != fmt.Sprint(want) {
		t.Fatalf("resolved owners = %+v, want %+v", resolved, want)
	}
	seedEffectOwnerState(t, pool, unknownTarget, ownerlifecycle.StateTerminal, 1, now)
	if _, err := executor.PutRecord(context.Background(), PutRecordRequest{
		OperationID: "target-scope-race:v1", MutationKey: "target-scope-race:v1",
		Owner: owner, OwnerGeneration: 2, ExpectedOwners: resolved,
		Collection: syntax.NSID("social.craftsky.feed.post"),
		Rkey:       syntax.RecordKey("3ltargetscope"),
		Record:     map[string]any{"$type": "social.craftsky.feed.post"},
	}); !errors.Is(err, ownerlifecycle.ErrTerminalOwner) {
		t.Fatalf("missing target terminalized before boundary = %v", err)
	}
	if pds.putCalls != 0 {
		t.Fatalf("target terminalization race crossed PDS boundary: puts=%d", pds.putCalls)
	}
	if _, err := executor.ResolveExpectedOwners(
		context.Background(), 1, nil,
	); !errors.Is(err, ownerlifecycle.ErrGenerationChanged) {
		t.Fatalf("stale bound owner generation = %v", err)
	}
	if _, err := executor.ResolveExpectedOwners(
		context.Background(), 2, []syntax.DID{departedTarget},
	); !errors.Is(err, ownerlifecycle.ErrOwnerNotActive) {
		t.Fatalf("departed target = %v", err)
	}
	if _, err := executor.ResolveExpectedOwners(
		context.Background(), 2, []syntax.DID{terminalTarget},
	); !errors.Is(err, ownerlifecycle.ErrTerminalOwner) {
		t.Fatalf("terminal target = %v", err)
	}
}

func TestExecutorUploadBlobUsesContentIdentityAndNeverRepeatsAfterDispatch(t *testing.T) {
	pool, lifecycles, now := newEffectExecutorStore(t)
	owner := syntax.DID("did:plc:durable-blob")
	seedEffectOwner(t, pool, owner, 4, now)
	pds := newEffectPDS()
	body := []byte("durable blob bytes")
	predictedCID, digest, err := PredictBlobCID(body)
	if err != nil {
		t.Fatal(err)
	}
	pds.uploadResponse = &auth.UploadedBlob{
		CID: predictedCID.String(), MIME: "image/png", Size: int64(len(body)),
		Raw: map[string]any{
			"$type":    "blob",
			"ref":      map[string]any{"$link": predictedCID.String()},
			"mimeType": "image/png",
			"size":     int64(len(body)),
		},
	}
	executor, err := NewExecutor(
		lifecycles,
		&storeEffectBoundary{lifecycles: lifecycles, client: pds},
		owner,
		10*time.Second,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	request := UploadBlobRequest{
		OperationID: "blob:durable:v1",
		MutationKey: "blob:durable:v1",
		Owner:       owner, OwnerGeneration: 4,
		ExpectedOwners: []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 4}},
		MIME:           "image/png", Bytes: body,
	}
	uploaded, err := executor.UploadBlob(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if uploaded == nil || uploaded.CID != predictedCID.String() || uploaded.MIME != request.MIME ||
		uploaded.Size != int64(len(body)) || uploaded.Raw == nil {
		t.Fatalf("uploaded blob = %+v", uploaded)
	}
	if pds.uploadCalls != 1 {
		t.Fatalf("upload calls = %d", pds.uploadCalls)
	}
	var kind ownerlifecycle.EffectKind
	var action ownerlifecycle.EffectAction
	var exactKey string
	if err := pool.QueryRow(context.Background(), `
		SELECT effect_kind,effect_action,deterministic_key
		FROM owner_effect_attempts WHERE operation_id=$1
	`, request.OperationID).Scan(&kind, &action, &exactKey); err != nil {
		t.Fatal(err)
	}
	if kind != ownerlifecycle.EffectObjectPut || action != ownerlifecycle.EffectActionUploadBlob ||
		!strings.Contains(exactKey, predictedCID.String()) ||
		!strings.Contains(exactKey, hex.EncodeToString(digest[:])) ||
		!strings.Contains(exactKey, "image%2Fpng") ||
		!strings.Contains(exactKey, fmt.Sprintf("size=%d", len(body))) {
		t.Fatalf("blob effect identity = kind %q action %q key %q", kind, action, exactKey)
	}
	replayed, err := executor.UploadBlob(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if replayed == nil || replayed.CID != uploaded.CID || replayed.Raw == nil || pds.uploadCalls != 1 {
		t.Fatalf("blob replay = %+v calls %d", replayed, pds.uploadCalls)
	}

	ambiguous := request
	ambiguous.OperationID = "blob:durable:v2"
	ambiguous.MutationKey = "blob:durable:v2"
	ambiguous.Bytes = []byte("second durable blob")
	pds.uploadError = errors.New("lost upload response")
	if _, err := executor.UploadBlob(context.Background(), ambiguous); !errors.Is(err, ErrOutcomeAmbiguous) {
		t.Fatalf("ambiguous upload = %v", err)
	}
	if _, err := executor.UploadBlob(context.Background(), ambiguous); !errors.Is(err, ErrOutcomeAmbiguous) {
		t.Fatalf("ambiguous upload replay = %v", err)
	}
	if pds.uploadCalls != 2 {
		t.Fatalf("ambiguous upload was repeated: calls %d", pds.uploadCalls)
	}
}

func TestExecutorBlobMetadataMismatchIsNeverAcceptedOrRepeated(t *testing.T) {
	pool, lifecycles, now := newEffectExecutorStore(t)
	owner := syntax.DID("did:plc:blob-metadata-mismatch")
	seedEffectOwner(t, pool, owner, 4, now)
	pds := newEffectPDS()
	body := []byte("metadata mismatch")
	predictedCID, _, err := PredictBlobCID(body)
	if err != nil {
		t.Fatal(err)
	}
	pds.uploadResponse = &auth.UploadedBlob{
		CID:  predictedCID.String(),
		MIME: "application/octet-stream",
		Size: int64(len(body)),
		Raw:  map[string]any{"$type": "blob"},
	}
	executor, err := NewExecutor(
		lifecycles,
		&storeEffectBoundary{lifecycles: lifecycles, client: pds},
		owner,
		10*time.Second,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	request := UploadBlobRequest{
		OperationID:     "blob:metadata-mismatch:v1",
		MutationKey:     "blob:metadata-mismatch:v1",
		Owner:           owner,
		OwnerGeneration: 4,
		ExpectedOwners:  []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 4}},
		MIME:            "image/png",
		Bytes:           body,
	}
	if _, err := executor.UploadBlob(context.Background(), request); !errors.Is(err, ErrEffectConflict) {
		t.Fatalf("metadata mismatch = %v, want ErrEffectConflict", err)
	}
	assertEffectOutcome(t, pool, request.OperationID, ownerlifecycle.OutcomeDispatched)
	if _, err := executor.UploadBlob(context.Background(), request); !errors.Is(err, ErrOutcomeAmbiguous) {
		t.Fatalf("metadata mismatch replay = %v, want ErrOutcomeAmbiguous", err)
	}
	if pds.uploadCalls != 1 {
		t.Fatalf("metadata mismatch upload repeated: calls=%d", pds.uploadCalls)
	}
}

func TestExecutorReadOnlyReconcilesUnknownPutAfterDeparture(t *testing.T) {
	pool, lifecycles, now := newEffectExecutorStore(t)
	owner := syntax.DID("did:plc:departed-reconcile-put")
	seedEffectOwner(t, pool, owner, 6, now)
	pds := newEffectPDS()
	executor, err := NewExecutor(
		lifecycles,
		&storeEffectBoundary{lifecycles: lifecycles, client: pds},
		owner,
		10*time.Second,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	request := PutRecordRequest{
		OperationID: "post:departed-reconcile:v1",
		MutationKey: "post:departed-reconcile:v1",
		Owner:       owner, OwnerGeneration: 6,
		ExpectedOwners: []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 6}},
		Collection:     syntax.NSID("social.craftsky.feed.post"),
		Rkey:           syntax.RecordKey("3ldepartedreconcile"),
		Record: map[string]any{
			"$type": "social.craftsky.feed.post",
			"text":  "accepted before departure",
		},
	}
	_, fingerprint, err := canonicalPutBody(
		request.Owner, request.Collection, request.Rkey, request.Record, request.ExpectedCID,
	)
	if err != nil {
		t.Fatal(err)
	}
	uri, err := deterministicRecordURI(request.Owner, request.Collection, request.Rkey)
	if err != nil {
		t.Fatal(err)
	}
	err = lifecycles.WithActiveEffects(
		context.Background(), request.ExpectedOwners,
		func(effectCtx context.Context) error {
			attempt, err := lifecycles.CreateEffectAttempt(effectCtx, ownerlifecycle.NewEffectAttempt{
				OperationID: request.OperationID, MutationKey: request.MutationKey,
				Owner: owner, OwnerGeneration: 6,
				Kind: ownerlifecycle.EffectPDSRecord, Action: ownerlifecycle.EffectActionPutRecord,
				DeterministicKey: uri.String(), RequestFingerprint: fingerprint,
				RemoteDeadline: now.Add(time.Minute),
			})
			if err != nil {
				return err
			}
			_, err = lifecycles.MarkAttemptDispatched(effectCtx, attempt.OperationID, owner, 6)
			return err
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	pds.records[uri.String()] = storedEffectRecord{
		cid:   "bafy-departed-accepted",
		value: request.Record.(map[string]any),
	}
	if _, err := lifecycles.Transition(context.Background(), ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: 6,
		To: ownerlifecycle.StateDeparted, Reason: "profileDeleted",
	}); err != nil {
		t.Fatal(err)
	}

	result, err := executor.ReconcilePutRecord(context.Background(), request, pds)
	if err != nil {
		t.Fatal(err)
	}
	if result.URI != uri || result.CID != "bafy-departed-accepted" || pds.putCalls != 0 || pds.getCalls != 1 {
		t.Fatalf("read-only reconcile result=%+v puts=%d gets=%d", result, pds.putCalls, pds.getCalls)
	}
	attempt, err := lifecycles.GetEffectAttempt(context.Background(), request.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if attempt.Outcome != ownerlifecycle.OutcomeReconciledAccepted ||
		attempt.ProjectionDisposition != ownerlifecycle.ProjectionHiddenNonActive {
		t.Fatalf("departed reconciled attempt = %+v", attempt)
	}
}

func TestExecutorReadOnlyReconcilesUnknownDeleteToAbsent(t *testing.T) {
	pool, lifecycles, now := newEffectExecutorStore(t)
	owner := syntax.DID("did:plc:departed-reconcile-delete")
	seedEffectOwner(t, pool, owner, 7, now)
	pds := newEffectPDS()
	executor, err := NewExecutor(
		lifecycles,
		&storeEffectBoundary{lifecycles: lifecycles, client: pds},
		owner,
		10*time.Second,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	request := DeleteRecordRequest{
		OperationID: "post:departed-delete:v1",
		MutationKey: "post:departed-delete:v1",
		Owner:       owner, OwnerGeneration: 7,
		ExpectedOwners: []ownerlifecycle.ExpectedOwner{{Owner: owner, Generation: 7}},
		Collection:     syntax.NSID("social.craftsky.feed.post"),
		Rkey:           syntax.RecordKey("3ldeparteddelete"),
	}
	_, fingerprint, uri, err := canonicalDeleteBody(
		request.Owner, request.Collection, request.Rkey, request.ExpectedCID,
	)
	if err != nil {
		t.Fatal(err)
	}
	err = lifecycles.WithActiveEffects(
		context.Background(), request.ExpectedOwners,
		func(effectCtx context.Context) error {
			attempt, err := lifecycles.CreateEffectAttempt(effectCtx, ownerlifecycle.NewEffectAttempt{
				OperationID: request.OperationID, MutationKey: request.MutationKey,
				Owner: owner, OwnerGeneration: 7,
				Kind: ownerlifecycle.EffectPDSRecord, Action: ownerlifecycle.EffectActionDeleteRecord,
				DeterministicKey: uri.String(), RequestFingerprint: fingerprint,
				RemoteDeadline: now.Add(time.Minute),
			})
			if err != nil {
				return err
			}
			_, err = lifecycles.MarkAttemptDispatched(effectCtx, attempt.OperationID, owner, 7)
			return err
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lifecycles.Transition(context.Background(), ownerlifecycle.TransitionRequest{
		Owner: owner, ExpectedGeneration: 7,
		To: ownerlifecycle.StateDeparted, Reason: "profileDeleted",
	}); err != nil {
		t.Fatal(err)
	}
	result, err := executor.ReconcileDeleteRecord(context.Background(), request, pds)
	if err != nil {
		t.Fatal(err)
	}
	if result.URI != uri || result.CID != "" || pds.deleteCalls != 0 || pds.getCalls != 1 {
		t.Fatalf("delete reconcile result=%+v deletes=%d gets=%d", result, pds.deleteCalls, pds.getCalls)
	}
	attempt, err := lifecycles.GetEffectAttempt(context.Background(), request.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if attempt.Outcome != ownerlifecycle.OutcomeReconciledAccepted || attempt.ResultCID != "" {
		t.Fatalf("reconciled delete attempt = %+v", attempt)
	}
}

func TestOnboardingExecutorDurablyPutsOnlyDepartedProfileWithoutRepeating(t *testing.T) {
	_, lifecycles, now := newEffectExecutorStore(t)
	owner := syntax.DID("did:plc:onboarding-durable-profile")
	authority, err := lifecycles.EnsureOnboardingOwner(context.Background(), owner)
	if err != nil {
		t.Fatal(err)
	}
	pds := newEffectPDS()
	pds.putError = errors.New("lost onboarding profile response")
	pds.acceptPutBeforeError = true
	executor, err := NewOnboardingExecutor(
		lifecycles,
		10*time.Second,
		func() time.Time { return now },
	)
	if err != nil {
		t.Fatal(err)
	}
	request := OnboardingProfileRequest{
		OperationID: "onboarding-profile:attempt:v1",
		MutationKey: "onboarding-profile:attempt:v1",
		Owner:       owner, OwnerGeneration: authority.Generation,
		Record: map[string]any{
			"$type":  "social.craftsky.actor.profile",
			"crafts": []string{},
		},
	}
	err = lifecycles.WithOnboardingAuth(
		context.Background(), owner,
		func(authCtx context.Context, _ ownerlifecycle.Lifecycle) error {
			_, executeErr := executor.PutProfile(authCtx, pds, request)
			return executeErr
		},
	)
	if !errors.Is(err, ErrOutcomeAmbiguous) || pds.putCalls != 1 {
		t.Fatalf("first onboarding put error=%v calls=%d", err, pds.putCalls)
	}
	pds.putError = nil
	var result RecordResult
	err = lifecycles.WithOnboardingAuth(
		context.Background(), owner,
		func(authCtx context.Context, _ ownerlifecycle.Lifecycle) error {
			var executeErr error
			result, executeErr = executor.PutProfile(authCtx, pds, request)
			return executeErr
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.URI.String() != "at://did:plc:onboarding-durable-profile/social.craftsky.actor.profile/self" ||
		result.CID == "" || pds.putCalls != 1 || pds.getCalls != 1 {
		t.Fatalf("onboarding reconciliation result=%+v puts=%d gets=%d", result, pds.putCalls, pds.getCalls)
	}
	if _, err := lifecycles.Terminalize(context.Background(), ownerlifecycle.TerminalizeRequest{
		Owner: owner, Reason: "identityDeleted",
		Components: []ownerlifecycle.PurgeComponent{{Component: "profiles", DIDRole: "owner"}},
	}); err != nil {
		t.Fatal(err)
	}
	called := false
	err = lifecycles.WithOnboardingAuth(
		context.Background(), owner,
		func(authCtx context.Context, _ ownerlifecycle.Lifecycle) error {
			called = true
			_, executeErr := executor.PutProfile(authCtx, pds, request)
			return executeErr
		},
	)
	if !errors.Is(err, ownerlifecycle.ErrTerminalOwner) || called || pds.putCalls != 1 {
		t.Fatalf("terminal onboarding put error=%v called=%t puts=%d", err, called, pds.putCalls)
	}
}

type storeEffectBoundary struct {
	lifecycles *ownerlifecycle.Store
	client     auth.PDSClient
	calls      int
}

func (boundary *storeEffectBoundary) WithActiveEffects(
	ctx context.Context,
	expected []ownerlifecycle.ExpectedOwner,
	operation auth.ActiveEffectPDSOperation,
) error {
	boundary.calls++
	return boundary.lifecycles.WithActiveEffects(ctx, expected, func(effectCtx context.Context) error {
		return operation(effectCtx, boundary.client)
	})
}

type baseOnlyPDSClient struct {
	auth.PDSClient
}

type storedEffectRecord struct {
	cid   string
	value map[string]any
}

type effectPDS struct {
	records                        map[string]storedEffectRecord
	putCalls                       int
	conditionalPutCalls            int
	unconditionalPutCalls          int
	getCalls                       int
	deleteCalls                    int
	conditionalDeleteCalls         int
	unconditionalDeleteCalls       int
	uploadCalls                    int
	putError                       error
	acceptPutBeforeError           bool
	deleteError                    error
	acceptDeleteBeforeError        bool
	uploadResponse                 *auth.UploadedBlob
	uploadError                    error
	getError                       error
	getOverride                    *storedEffectRecord
	replaceBeforeConditionalDelete *storedEffectRecord
}

func newEffectPDS() *effectPDS {
	return &effectPDS{records: make(map[string]storedEffectRecord)}
}

func effectRecordKey(repo syntax.DID, collection, rkey string) string {
	return "at://" + repo.String() + "/" + collection + "/" + rkey
}

func (pds *effectPDS) GetRecord(
	_ context.Context,
	repo syntax.DID,
	collection string,
	rkey string,
	out any,
) (string, error) {
	pds.getCalls++
	if pds.getError != nil {
		return "", pds.getError
	}
	record, ok := pds.records[effectRecordKey(repo, collection, rkey)]
	if pds.getOverride != nil {
		record = *pds.getOverride
		ok = true
	}
	if !ok {
		return "", auth.ErrRecordNotFound
	}
	target, ok := out.(*map[string]any)
	if !ok {
		return "", errors.New("unsupported effect test output")
	}
	*target = record.value
	return record.cid, nil
}

func assertEffectOutcome(
	t *testing.T,
	pool *pgxpool.Pool,
	operationID string,
	want ownerlifecycle.EffectOutcome,
) {
	t.Helper()
	var got ownerlifecycle.EffectOutcome
	if err := pool.QueryRow(context.Background(), `
		SELECT remote_outcome FROM owner_effect_attempts WHERE operation_id=$1
	`, operationID).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("effect outcome = %q, want %q", got, want)
	}
}

func (pds *effectPDS) PutRecord(
	_ context.Context,
	repo syntax.DID,
	collection string,
	rkey string,
	record any,
) error {
	pds.putCalls++
	pds.unconditionalPutCalls++
	value, ok := record.(map[string]any)
	if !ok {
		return errors.New("effect test requires map record")
	}
	if pds.putError == nil || pds.acceptPutBeforeError {
		pds.records[effectRecordKey(repo, collection, rkey)] = storedEffectRecord{
			cid: "bafy-effect-record", value: value,
		}
	}
	return pds.putError
}

func (pds *effectPDS) PutRecordWithSwap(
	_ context.Context,
	repo syntax.DID,
	collection string,
	rkey string,
	record any,
	expectedCID syntax.CID,
) error {
	pds.putCalls++
	pds.conditionalPutCalls++
	key := effectRecordKey(repo, collection, rkey)
	current, ok := pds.records[key]
	if !ok || current.cid != expectedCID.String() {
		return auth.ErrRecordSwapConflict
	}
	value, ok := record.(map[string]any)
	if !ok {
		return errors.New("effect test requires map record")
	}
	if pds.putError == nil || pds.acceptPutBeforeError {
		pds.records[key] = storedEffectRecord{cid: "bafy-effect-record", value: value}
	}
	return pds.putError
}

func (pds *effectPDS) CreateRecord(
	context.Context, syntax.DID, string, any,
) (syntax.ATURI, syntax.CID, error) {
	return "", "", errors.New("CreateRecord is forbidden at the deterministic effect boundary")
}

func (pds *effectPDS) DeleteRecord(
	_ context.Context,
	repo syntax.DID,
	collection string,
	rkey string,
) error {
	pds.deleteCalls++
	pds.unconditionalDeleteCalls++
	if pds.deleteError == nil || pds.acceptDeleteBeforeError {
		delete(pds.records, effectRecordKey(repo, collection, rkey))
	}
	return pds.deleteError
}

func (pds *effectPDS) DeleteRecordWithSwap(
	_ context.Context,
	repo syntax.DID,
	collection string,
	rkey string,
	expectedCID syntax.CID,
) error {
	pds.deleteCalls++
	pds.conditionalDeleteCalls++
	key := effectRecordKey(repo, collection, rkey)
	if pds.replaceBeforeConditionalDelete != nil {
		pds.records[key] = *pds.replaceBeforeConditionalDelete
		pds.replaceBeforeConditionalDelete = nil
	}
	record, ok := pds.records[key]
	if !ok {
		return auth.ErrRecordNotFound
	}
	if record.cid != expectedCID.String() {
		return auth.ErrRecordSwapConflict
	}
	if pds.deleteError == nil || pds.acceptDeleteBeforeError {
		delete(pds.records, key)
	}
	return pds.deleteError
}

func (pds *effectPDS) UploadBlob(
	context.Context, string, []byte,
) (*auth.UploadedBlob, error) {
	pds.uploadCalls++
	return pds.uploadResponse, pds.uploadError
}

func newEffectExecutorStore(t *testing.T) (*pgxpool.Pool, *ownerlifecycle.Store, time.Time) {
	t.Helper()
	pool := testdb.WithSchema(t, `
		CREATE TABLE craftsky_profiles(did TEXT PRIMARY KEY, created_at TIMESTAMPTZ NOT NULL DEFAULT now());
		CREATE TABLE oauth_sessions(
			account_did TEXT NOT NULL, session_id TEXT NOT NULL, data JSONB NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			PRIMARY KEY(account_did,session_id)
		);
		CREATE TABLE oauth_auth_requests(
			state TEXT PRIMARY KEY, data JSONB NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			handoff_mode TEXT NOT NULL DEFAULT 'deep_link', loopback_redirect_uri TEXT,
			purpose TEXT NOT NULL DEFAULT 'login', device_id TEXT,
			account_deletion_owner_did TEXT, account_deletion_job_id UUID
		);
		CREATE TABLE craftsky_sessions(
			token_hash BYTEA PRIMARY KEY, account_did TEXT NOT NULL, oauth_session_id TEXT NOT NULL,
			device_label TEXT, last_device_id TEXT, created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(), revoked_at TIMESTAMPTZ,
			FOREIGN KEY(account_did,oauth_session_id)
				REFERENCES oauth_sessions(account_did,session_id) ON DELETE CASCADE
		);
		CREATE TABLE account_deletion_operations(
			id UUID PRIMARY KEY, owner_did TEXT NOT NULL UNIQUE, state TEXT NOT NULL,
			reauth_oauth_session_id TEXT, deletion_oauth_session_id TEXT,
			FOREIGN KEY(owner_did,deletion_oauth_session_id)
				REFERENCES oauth_sessions(account_did,session_id),
			FOREIGN KEY(owner_did,reauth_oauth_session_id)
				REFERENCES oauth_sessions(account_did,session_id)
		);
	`)
	for _, path := range []string{
		"../../migrations/000038_owner_auth_lifecycle.up.sql",
		"../../migrations/000039_owner_effects_terminal_purge.up.sql",
		"../../migrations/000049_pds_effect_action.up.sql",
	} {
		migration, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(context.Background(), string(migration)); err != nil {
			t.Fatalf("apply %s: %v", path, err)
		}
	}
	now := time.Date(2026, 8, 14, 19, 0, 0, 0, time.UTC)
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return pool, lifecycles, now
}

func seedEffectOwner(
	t *testing.T,
	pool *pgxpool.Pool,
	owner syntax.DID,
	generation int64,
	now time.Time,
) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,created_at,updated_at
		) VALUES($1,'active',$2,1,'test',$3,$3,$3)
	`, owner, generation, now); err != nil {
		t.Fatal(err)
	}
}

func seedEffectOwnerState(
	t *testing.T,
	pool *pgxpool.Pool,
	owner syntax.DID,
	state ownerlifecycle.State,
	generation int64,
	now time.Time,
) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,
			transitioned_at,terminal_at,created_at,updated_at
		) VALUES($1,$2,$3,1,'test',$4,CASE WHEN $2='terminal' THEN $4 ELSE NULL END,$4,$4)
	`, owner, state, generation, now); err != nil {
		t.Fatal(err)
	}
}

var _ auth.ConditionalPDSRecordDeleter = (*effectPDS)(nil)
var _ auth.ConditionalPDSRecordPutter = (*effectPDS)(nil)
