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

func TestCompoundPutRejectsUnsupportedAtomicWritesWithoutChangingRecords(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:compound-owner")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
		) VALUES($1,'active',3,1,'test',$2,$2,$2)
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
	bskyURI := syntax.ATURI("at://did:plc:compound-owner/app.bsky.actor.profile/self")
	craftskyURI := syntax.ATURI("at://did:plc:compound-owner/social.craftsky.actor.profile/self")
	pds.records[bskyURI] = setCommandRecord{cid: "bafy-old-bsky", value: map[string]any{"$type": "app.bsky.actor.profile", "displayName": "Before"}}
	pds.records[craftskyURI] = setCommandRecord{cid: "bafy-old-craftsky", value: map[string]any{"$type": "social.craftsky.actor.profile", "crafts": []any{"sewing"}}}
	pds.applyErr = auth.ErrApplyWritesUnsupported
	service, err := NewCompoundCommandService(CompoundCommandServiceConfig{
		Store: commandStore, Lifecycles: lifecycles,
		NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) {
			return &setCommandBoundary{client: pds}, nil
		},
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	desiredBsky := json.RawMessage(`{"$type":"app.bsky.actor.profile","displayName":"After"}`)
	desiredCraftsky := json.RawMessage(`{"$type":"social.craftsky.actor.profile","crafts":["quilting"]}`)
	request := CompoundPutCommandRequest{
		Owner: owner, OwnerGeneration: 3, SessionID: "session",
		OperationKind: "profile.update", OperationKey: uuid.MustParse("018f4d5c-7a61-7d40-a1a2-aaaaaaaaaaaa"),
		Intent: json.RawMessage(`{"displayName":"After","crafts":["quilting"]}`),
		Records: []CompoundPutRecordRequest{
			{URI: bskyURI, BuildRecord: func(AuthoritativeRecord) (json.RawMessage, error) { return desiredBsky, nil }},
			{URI: craftskyURI, BuildRecord: func(AuthoritativeRecord) (json.RawMessage, error) { return desiredCraftsky, nil }},
		},
		Accepted: func([]AuthoritativeRecord) (TerminalResult, error) {
			return TerminalResult{State: CommandAccepted, HTTPStatus: 200}, nil
		},
		Rejected: func(error) TerminalResult {
			return TerminalResult{State: CommandRejected, HTTPStatus: 502, ResponseBody: json.RawMessage(`{"error":"pds_write_failed"}`)}
		},
	}

	result, err := service.Put(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != CommandRejected || result.HTTPStatus != 502 || pds.applyCalls != 1 {
		t.Fatalf("result=%+v applyCalls=%d", result, pds.applyCalls)
	}
	if got := pds.records[bskyURI].value.(map[string]any)["displayName"]; got != "Before" {
		t.Fatalf("bluesky displayName = %v, want unchanged", got)
	}
	crafts := pds.records[craftskyURI].value.(map[string]any)["crafts"].([]any)
	if len(crafts) != 1 || crafts[0] != "sewing" {
		t.Fatalf("CraftSky crafts = %v, want unchanged", crafts)
	}
	if !errors.Is(pds.applyErr, auth.ErrApplyWritesUnsupported) {
		t.Fatal("test setup lost unsupported atomic-write result")
	}
}

func TestCompoundPutReconcilesBothRecordsAfterLostResponse(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 25, 12, 30, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:compound-reconcile")
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
	bskyURI := syntax.ATURI("at://did:plc:compound-reconcile/app.bsky.actor.profile/self")
	craftskyURI := syntax.ATURI("at://did:plc:compound-reconcile/social.craftsky.actor.profile/self")
	pds.records[bskyURI] = setCommandRecord{cid: "bafy-old-bsky", value: map[string]any{"$type": "app.bsky.actor.profile", "displayName": "Before"}}
	pds.records[craftskyURI] = setCommandRecord{cid: "bafy-old-craftsky", value: map[string]any{"$type": "social.craftsky.actor.profile", "crafts": []any{"sewing"}}}
	pds.loseNextResponse = true
	service, err := NewCompoundCommandService(CompoundCommandServiceConfig{
		Store: commandStore, Lifecycles: lifecycles,
		NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) {
			return &setCommandBoundary{client: pds}, nil
		},
		Now: func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	desiredBsky := json.RawMessage(`{"$type":"app.bsky.actor.profile","displayName":"After"}`)
	desiredCraftsky := json.RawMessage(`{"$type":"social.craftsky.actor.profile","crafts":["quilting"]}`)
	request := CompoundPutCommandRequest{
		Owner: owner, OwnerGeneration: 4, SessionID: "session",
		OperationKind: "profile.update", OperationKey: uuid.MustParse("018f4d5c-7a61-7d40-a1a2-aaaaaaaaaaac"),
		Intent: json.RawMessage(`{"displayName":"After","crafts":["quilting"]}`),
		Records: []CompoundPutRecordRequest{
			{URI: bskyURI, BuildRecord: func(AuthoritativeRecord) (json.RawMessage, error) { return desiredBsky, nil }},
			{URI: craftskyURI, BuildRecord: func(AuthoritativeRecord) (json.RawMessage, error) { return desiredCraftsky, nil }},
		},
		Accepted: func(records []AuthoritativeRecord) (TerminalResult, error) {
			body, err := json.Marshal(map[string]any{"records": len(records)})
			return TerminalResult{State: CommandAccepted, HTTPStatus: 200, ResponseBody: body}, err
		},
		Rejected: func(error) TerminalResult {
			return TerminalResult{State: CommandRejected, HTTPStatus: 502}
		},
	}

	first, err := service.Put(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if first.State != CommandAmbiguous || first.RetryAfterSeconds != 1 || pds.applyCalls != 1 {
		t.Fatalf("first result=%+v applyCalls=%d", first, pds.applyCalls)
	}
	second, err := service.Put(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if second.State != CommandAccepted || second.HTTPStatus != 200 || pds.applyCalls != 1 {
		t.Fatalf("reconciled result=%+v applyCalls=%d", second, pds.applyCalls)
	}
	if len(pds.lastWrites) != 2 || pds.lastWrites[0].URI != bskyURI || pds.lastWrites[1].URI != craftskyURI {
		t.Fatalf("ordered writes = %+v", pds.lastWrites)
	}
}
