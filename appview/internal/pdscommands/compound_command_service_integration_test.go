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

func TestCompoundPutDispatchesOnlyChangedRecords(t *testing.T) {
	for _, tc := range []struct {
		name            string
		displayName     string
		crafts          string
		missingCraftsky bool
		loseResponse    bool
		wantWrites      []string
	}{
		{name: "crafts only", displayName: "Before", crafts: ` ["sewing", "embroidery"] `, wantWrites: []string{"social.craftsky.actor.profile"}},
		{name: "bluesky only", displayName: "After", crafts: `["sewing"]`, wantWrites: []string{"app.bsky.actor.profile"}},
		{name: "both changed", displayName: "After", crafts: `["embroidery"]`, wantWrites: []string{"app.bsky.actor.profile", "social.craftsky.actor.profile"}},
		{name: "neither changed", displayName: "Before", crafts: `["sewing"]`},
		{name: "missing craftsky", displayName: "Before", crafts: `["sewing"]`, missingCraftsky: true, wantWrites: []string{"social.craftsky.actor.profile"}},
		{name: "crafts only lost response", displayName: "Before", crafts: `["embroidery"]`, loseResponse: true, wantWrites: []string{"social.craftsky.actor.profile"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			pool := testdb.WithMigratedSchema(t)
			now := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
			owner := syntax.DID("did:plc:compound-changed")
			if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(
				owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
			) VALUES($1,'active',3,1,'test',$2,$2,$2)`, owner, now); err != nil {
				t.Fatal(err)
			}
			store, err := NewStore(StoreConfig{Pool: pool, Now: func() time.Time { return now }, NewUUID: uuid.New})
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
			bskyURI := syntax.ATURI("at://" + owner.String() + "/app.bsky.actor.profile/self")
			craftskyURI := syntax.ATURI("at://" + owner.String() + "/social.craftsky.actor.profile/self")
			pds.records[bskyURI] = setCommandRecord{cid: "bafy-old-bsky", value: map[string]any{"$type": "app.bsky.actor.profile", "displayName": "Before"}}
			if !tc.missingCraftsky {
				pds.records[craftskyURI] = setCommandRecord{cid: "bafy-old-craftsky", value: map[string]any{"$type": "social.craftsky.actor.profile", "crafts": []any{"sewing"}}}
			}
			pds.loseNextResponse = tc.loseResponse
			service, err := NewCompoundCommandService(CompoundCommandServiceConfig{
				Store: store, Lifecycles: lifecycles,
				NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) {
					return &setCommandBoundary{client: pds}, nil
				},
				Now: func() time.Time { return now },
			})
			if err != nil {
				t.Fatal(err)
			}
			// Different key order and whitespace must not turn unchanged content into a write.
			desiredBsky := json.RawMessage(`{"displayName":"` + tc.displayName + `", "$type":"app.bsky.actor.profile"}`)
			desiredCraftsky := json.RawMessage(`{"crafts":` + tc.crafts + `, "$type":"social.craftsky.actor.profile"}`)
			var accepted []AuthoritativeRecord
			request := CompoundPutCommandRequest{
				Owner: owner, OwnerGeneration: 3, SessionID: "session", OperationKind: "profile.update", OperationKey: uuid.New(),
				Intent: json.RawMessage(`{"displayName":"` + tc.displayName + `", "crafts":` + tc.crafts + `}`),
				Records: []CompoundPutRecordRequest{
					{URI: bskyURI, BuildRecord: func(AuthoritativeRecord) (json.RawMessage, error) { return desiredBsky, nil }},
					{URI: craftskyURI, BuildRecord: func(AuthoritativeRecord) (json.RawMessage, error) { return desiredCraftsky, nil }},
				},
				Accepted: func(records []AuthoritativeRecord) (TerminalResult, error) {
					accepted = records
					return TerminalResult{State: CommandAccepted, HTTPStatus: 200}, nil
				},
				Rejected: func(error) TerminalResult { return TerminalResult{State: CommandRejected, HTTPStatus: 502} },
			}
			result, err := service.Put(ctx, request)
			if err != nil {
				t.Fatal(err)
			}
			if tc.loseResponse {
				if result.State != CommandAmbiguous {
					t.Fatalf("lost response state = %s", result.State)
				}
				result, err = service.Put(ctx, request)
				if err != nil {
					t.Fatal(err)
				}
			}
			if result.State != CommandAccepted || len(accepted) != 2 || accepted[0].URI != bskyURI || accepted[1].URI != craftskyURI {
				t.Fatalf("result=%+v accepted records=%+v", result, accepted)
			}
			if !compoundRecordsMatch(accepted, []json.RawMessage{desiredBsky, desiredCraftsky}) {
				t.Fatal("accepted records do not match desired profile")
			}
			if len(pds.lastWrites) != len(tc.wantWrites) {
				t.Fatalf("writes=%+v, want collections=%v", pds.lastWrites, tc.wantWrites)
			}
			for i, collection := range tc.wantWrites {
				wantAction := "update"
				if tc.missingCraftsky && collection == "social.craftsky.actor.profile" {
					wantAction = "create"
				}
				if pds.lastWrites[i].URI.Collection().String() != collection || pds.lastWrites[i].Action != wantAction {
					t.Fatalf("write=%+v, want %s %s", pds.lastWrites[i], wantAction, collection)
				}
			}
			if tc.displayName == "Before" && accepted[0].CID != "bafy-old-bsky" {
				t.Fatal("unchanged Bluesky record was rewritten")
			}
			wantCalls := 1
			if len(tc.wantWrites) == 0 {
				wantCalls = 0
			}
			if pds.applyCalls != wantCalls {
				t.Fatalf("apply calls=%d, want %d", pds.applyCalls, wantCalls)
			}
			if replay, err := service.Put(ctx, request); err != nil || replay.State != CommandAccepted || pds.applyCalls != wantCalls {
				t.Fatalf("replay=%+v err=%v apply calls=%d", replay, err, pds.applyCalls)
			}
		})
	}
}
