package pdscommands

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

func TestCompoundProfileLinksCreateAndRemoveConvergeAfterLostResponse(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	now := time.Now().UTC()
	owner := syntax.DID("did:plc:link-command")
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at)VALUES($1,'active',1,1,'test',$2,$2,$2)`, owner, now); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(StoreConfig{Pool: pool, Now: func() time.Time { return now }, NewUUID: uuid.New})
	if err != nil {
		t.Fatal(err)
	}
	fence, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	owners, err := ownerlifecycle.NewStore(pool, fence, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	pds := newSetCommandPDS(owner)
	profileURI := syntax.ATURI("at://" + owner.String() + "/app.bsky.actor.profile/self")
	craftURI := syntax.ATURI("at://" + owner.String() + "/social.craftsky.actor.profile/self")
	newURI := syntax.ATURI("at://" + owner.String() + "/app.bsky.actor.link/3m5abc2345672")
	oldURI := syntax.ATURI("at://" + owner.String() + "/app.bsky.actor.link/3m5abc2345673")
	pds.records[oldURI] = setCommandRecord{cid: "cid-old-link", value: map[string]any{"$type": "app.bsky.actor.link", "url": "https://old.example/"}}
	pds.loseNextResponse = true
	service, err := NewCompoundCommandService(CompoundCommandServiceConfig{Store: store, Lifecycles: owners, Now: func() time.Time { return now }, NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) {
		return &setCommandBoundary{client: pds}, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	request := CompoundPutCommandRequest{Owner: owner, OwnerGeneration: 1, SessionID: "session", OperationKind: "profile.update", OperationKey: uuid.New(), Intent: json.RawMessage(`{"links":["https://new.example/"],"removedLinks":["old"]}`),
		Records: []CompoundPutRecordRequest{{URI: profileURI}, {URI: craftURI}, {URI: newURI}, {URI: oldURI, Delete: true}},
		BuildRecords: func([]AuthoritativeRecord) ([]json.RawMessage, error) {
			return []json.RawMessage{
				json.RawMessage(`{"$type":"app.bsky.actor.profile","links":[{"uri":"` + newURI.String() + `","cid":"cid-new"}]}`),
				json.RawMessage(`{"$type":"social.craftsky.actor.profile","crafts":[]}`),
				json.RawMessage(`{"$type":"app.bsky.actor.link","url":"https://new.example/","createdAt":"2026-10-27T12:00:00Z"}`), nil}, nil
		},
		Accepted: func(records []AuthoritativeRecord) (TerminalResult, error) {
			if len(records) != 4 || records[3].CID != "" {
				t.Fatal("deleted record survived authoritative read")
			}
			return TerminalResult{State: CommandAccepted, HTTPStatus: 200}, nil
		},
		Rejected: func(error) TerminalResult { return TerminalResult{State: CommandRejected, HTTPStatus: 409} },
	}
	first, err := service.Put(ctx, request)
	if err != nil || first.State != CommandAmbiguous {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	second, err := service.Put(ctx, request)
	if err != nil || second.State != CommandAccepted || pds.applyCalls != 1 {
		t.Fatalf("retry=%+v calls=%d err=%v", second, pds.applyCalls, err)
	}
	if _, exists := pds.records[oldURI]; exists {
		t.Fatal("removed link still exists")
	}
	if len(pds.lastWrites) != 4 || pds.lastWrites[3].Action != "delete" {
		t.Fatalf("writes=%+v", pds.lastWrites)
	}
	// Replaying the same accepted operation cannot dispatch a second deletion.
	if _, err := service.Put(ctx, request); err != nil || pds.applyCalls != 1 {
		t.Fatal("terminal replay dispatched")
	}
}
