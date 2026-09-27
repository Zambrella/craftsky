package pdscommands

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

func TestAddressedDeleteReconcilesAbsenceAndRejectsStaleCID(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 24, 14, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:addressed-owner")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
		) VALUES($1,'active',4,1,'test',$2,$2,$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	commandStore, err := NewStore(StoreConfig{Pool: pool, Now: func() time.Time { return now }, NewUUID: uuid.New})
	if err != nil {
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
	pds := newSetCommandPDS(owner)
	uri := syntax.ATURI("at://did:plc:addressed-owner/social.craftsky.feed.post/3deletepost")
	pds.records[uri] = setCommandRecord{cid: "bafy-delete-current", value: map[string]any{"$type": "social.craftsky.feed.post", "text": "delete me"}}
	service, err := NewAddressedCommandService(AddressedCommandServiceConfig{
		Store: commandStore, Lifecycles: lifecycles,
		NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) {
			return &setCommandBoundary{client: pds}, nil
		},
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	var rejectedCause error
	request := AddressedDeleteCommandRequest{
		Owner: owner, OwnerGeneration: 4, SessionID: "session",
		OperationKind: "post.delete", OperationKey: uuid.MustParse("018f4d5c-7a61-7d40-a1a2-888888888888"),
		URI: uri, ExpectedCID: "bafy-delete-current", Intent: json.RawMessage(`{"uri":"` + uri.String() + `"}`),
		AcceptedAbsent: func() TerminalResult { return TerminalResult{State: CommandAccepted, HTTPStatus: 204} },
		Rejected: func(err error) TerminalResult {
			rejectedCause = err
			return setCommandRejectedResult(err)
		},
	}

	pds.loseNextResponse = true
	first, err := service.Delete(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if first.State != CommandAmbiguous || pds.applyCalls != 1 {
		t.Fatalf("first result=%+v applyCalls=%d", first, pds.applyCalls)
	}
	second, err := service.Delete(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if second.State != CommandAccepted || second.HTTPStatus != 204 || len(second.ResponseBody) != 0 || pds.applyCalls != 1 {
		t.Fatalf("reconciled result=%+v applyCalls=%d", second, pds.applyCalls)
	}

	pds.records[uri] = setCommandRecord{cid: "bafy-new-version", value: map[string]any{"$type": "social.craftsky.feed.post", "text": "changed"}}
	stale := request
	stale.OperationKey = uuid.MustParse("018f4d5c-7a61-7d40-a1a2-888888888889")
	result, err := service.Delete(ctx, stale)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != CommandRejected || pds.applyCalls != 1 {
		t.Fatalf("stale result=%+v applyCalls=%d", result, pds.applyCalls)
	}
	if !errors.Is(rejectedCause, ErrRecordConflict) {
		t.Fatalf("rejected cause = %v, want record conflict", rejectedCause)
	}
}

func TestAddressedPutReconcilesDesiredContentAndRejectsStaleCID(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 24, 14, 30, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:addressed-put-owner")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
		) VALUES($1,'active',5,1,'test',$2,$2,$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	commandStore, err := NewStore(StoreConfig{Pool: pool, Now: func() time.Time { return now }, NewUUID: uuid.New})
	if err != nil {
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
	pds := newSetCommandPDS(owner)
	uri := syntax.ATURI("at://did:plc:addressed-put-owner/social.craftsky.business.event/3updateevent")
	pds.records[uri] = setCommandRecord{cid: "bafy-put-current", value: map[string]any{
		"$type": "social.craftsky.business.event", "name": "Before", "createdAt": "2026-09-01T12:34:56Z",
	}}
	service, err := NewAddressedCommandService(AddressedCommandServiceConfig{
		Store: commandStore, Lifecycles: lifecycles,
		NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) {
			return &setCommandBoundary{client: pds}, nil
		},
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	desired := json.RawMessage(`{"$type":"social.craftsky.business.event","createdAt":"2026-09-01T12:34:56Z","name":"After"}`)
	var rejectedCause error
	request := AddressedPutCommandRequest{
		Owner: owner, OwnerGeneration: 5, SessionID: "session",
		OperationKind: "business_event.update", OperationKey: uuid.MustParse("018f4d5c-7a61-7d40-a1a2-999999999991"),
		URI: uri, ExpectedCID: "bafy-put-current", Intent: json.RawMessage(`{"name":"After"}`),
		BuildRecord: func(AuthoritativeRecord) (json.RawMessage, error) { return desired, nil },
		Accepted: func(record AuthoritativeRecord) (TerminalResult, error) {
			body, err := json.Marshal(map[string]string{"uri": record.URI.String(), "cid": record.CID.String()})
			return TerminalResult{State: CommandAccepted, HTTPStatus: 200, ResponseBody: body}, err
		},
		Rejected: func(err error) TerminalResult {
			rejectedCause = err
			return setCommandRejectedResult(err)
		},
	}

	pds.loseNextResponse = true
	first, err := service.Put(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if first.State != CommandAmbiguous || pds.applyCalls != 1 {
		t.Fatalf("first result=%+v applyCalls=%d", first, pds.applyCalls)
	}
	second, err := service.Put(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if second.State != CommandAccepted || second.HTTPStatus != 200 || pds.applyCalls != 1 {
		t.Fatalf("reconciled result=%+v applyCalls=%d", second, pds.applyCalls)
	}

	pds.records[uri] = setCommandRecord{cid: "bafy-put-newer", value: map[string]any{
		"$type": "social.craftsky.business.event", "name": "Someone Else", "createdAt": "2026-09-01T12:34:56Z",
	}}
	stale := request
	stale.OperationKey = uuid.MustParse("018f4d5c-7a61-7d40-a1a2-999999999992")
	result, err := service.Put(ctx, stale)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != CommandRejected || pds.applyCalls != 1 {
		t.Fatalf("stale result=%+v applyCalls=%d", result, pds.applyCalls)
	}
	if !errors.Is(rejectedCause, ErrRecordConflict) {
		t.Fatalf("rejected cause = %v, want record conflict", rejectedCause)
	}

	profileURI := syntax.ATURI("at://did:plc:addressed-put-owner/social.craftsky.business.profile/self")
	profile := json.RawMessage(`{"$type":"social.craftsky.business.profile","tagline":"Made locally"}`)
	create := request
	create.OperationKind = "business_profile.put"
	create.OperationKey = uuid.MustParse("018f4d5c-7a61-7d40-a1a2-999999999995")
	create.URI = profileURI
	create.ExpectedCID = "*"
	create.Intent = json.RawMessage(`{"tagline":"Made locally"}`)
	create.BuildRecord = func(current AuthoritativeRecord) (json.RawMessage, error) {
		if current.URI != profileURI || current.CID != "" || len(current.Record) != 0 {
			t.Fatalf("absent fixed-key input = %+v", current)
		}
		return profile, nil
	}
	created, err := service.Put(ctx, create)
	if err != nil {
		t.Fatal(err)
	}
	if created.State != CommandAccepted || pds.applyCalls != 2 || len(pds.lastWrites) != 1 ||
		pds.lastWrites[0].Action != "create" || pds.lastWrites[0].URI != profileURI {
		t.Fatalf("fixed-key create result=%+v applyCalls=%d writes=%+v", created, pds.applyCalls, pds.lastWrites)
	}
}
