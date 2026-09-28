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

func TestAppendCommandServiceFreezesIdentityAndReconcilesLostResponse(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:append-owner")
	target := syntax.DID("did:plc:append-target")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
		) VALUES($1,'active',3,1,'test',$3,$3,$3),($2,'active',5,1,'test',$3,$3,$3)
	`, owner, target, now); err != nil {
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
	boundary := &setCommandBoundary{client: pds}
	rkeys := []syntax.RecordKey{"3appendfirst", "3appendsecond", "3appendthird"}
	service, err := NewAppendCommandService(AppendCommandServiceConfig{
		Store: commandStore, Lifecycles: lifecycles,
		NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) { return boundary, nil },
		Now:         func() time.Time { return now },
		NewRecordKey: func() (syntax.RecordKey, error) {
			rkey := rkeys[0]
			rkeys = rkeys[1:]
			return rkey, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	request := AppendCommandRequest{
		Owner: owner, OwnerGeneration: 3, Targets: []syntax.DID{target}, SessionID: "session",
		OperationKind: "post.create", OperationKey: uuid.MustParse("018f4d5c-7a61-7d40-a1a2-999999999999"),
		Collection: "social.craftsky.feed.post", Intent: json.RawMessage(`{"text":"hello"}`),
		BuildRecord: func(createdAt time.Time) (json.RawMessage, error) {
			return json.Marshal(map[string]any{"$type": "social.craftsky.feed.post", "text": "hello", "createdAt": createdAt.Format(time.RFC3339Nano)})
		},
		Accepted: func(record AuthoritativeRecord) (TerminalResult, error) {
			body, _ := json.Marshal(map[string]string{"uri": record.URI.String(), "cid": record.CID.String()})
			return TerminalResult{State: CommandAccepted, HTTPStatus: 201, ResponseBody: body}, nil
		},
		Rejected: setCommandRejectedResult,
	}

	pds.loseNextResponse = true
	first, err := service.Execute(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if first.State != CommandAmbiguous || pds.applyCalls != 1 {
		t.Fatalf("first result=%+v applyCalls=%d", first, pds.applyCalls)
	}
	second, err := service.Execute(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if second.State != CommandAccepted || second.HTTPStatus != 201 || pds.applyCalls != 1 {
		t.Fatalf("reconciled result=%+v applyCalls=%d", second, pds.applyCalls)
	}
	var response map[string]string
	if err := json.Unmarshal(second.ResponseBody, &response); err != nil {
		t.Fatal(err)
	}
	if response["uri"] != "at://did:plc:append-owner/social.craftsky.feed.post/3appendfirst" {
		t.Fatalf("selected URI = %q", response["uri"])
	}
	record := pds.records[syntax.ATURI(response["uri"])]
	recordJSON, _ := json.Marshal(record.value)
	if !json.Valid(recordJSON) || !containsJSONField(t, recordJSON, "createdAt", now.Format(time.RFC3339Nano)) {
		t.Fatalf("frozen record = %s", recordJSON)
	}
	if len(boundary.expected) != 2 || boundary.expected[0].Generation != 3 || boundary.expected[1].Generation != 5 {
		t.Fatalf("lifecycle fence = %+v", boundary.expected)
	}

	changed := request
	changed.Intent = json.RawMessage(`{"text":"changed"}`)
	if _, err := service.Execute(ctx, changed); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("changed request error = %v, want idempotency conflict", err)
	}
}

func TestAppendCommandServiceKeepsRecordReplacedAfterDispatchAmbiguous(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 24, 13, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:append-race-owner")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',$2,$2,$2)
	`, owner, now); err != nil {
		t.Fatal(err)
	}
	commandStore, err := NewStore(StoreConfig{Pool: pool, Now: func() time.Time { return now }})
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
	selectedURI := syntax.ATURI("at://" + owner.String() + "/social.craftsky.feed.post/3appendrace")
	pds.afterApply = func() {
		pds.records[selectedURI] = setCommandRecord{
			cid: "bafy-replacement", value: map[string]any{"text": "replacement"},
		}
	}
	service, err := NewAppendCommandService(AppendCommandServiceConfig{
		Store: commandStore, Lifecycles: lifecycles,
		NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) {
			return &setCommandBoundary{client: pds}, nil
		},
		NewRecordKey: func() (syntax.RecordKey, error) { return "3appendrace", nil },
		Now:          func() time.Time { return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Execute(ctx, AppendCommandRequest{
		Owner: owner, OwnerGeneration: 1, SessionID: "session",
		OperationKind: "post.create", OperationKey: uuid.New(),
		Collection: "social.craftsky.feed.post", Intent: json.RawMessage(`{"text":"frozen"}`),
		BuildRecord: func(time.Time) (json.RawMessage, error) {
			return json.RawMessage(`{"text":"frozen"}`), nil
		},
		Accepted: func(AuthoritativeRecord) (TerminalResult, error) {
			return TerminalResult{State: CommandAccepted, HTTPStatus: 201}, nil
		},
		Rejected: setCommandRejectedResult,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.State != CommandAmbiguous || pds.applyCalls != 1 {
		t.Fatalf("result=%+v applyCalls=%d, want unresolved outcome after one write", result, pds.applyCalls)
	}
}

func TestValidateSelectedAppendIdentity(t *testing.T) {
	valid := FencedAppendCommandRequest{
		Command: AppendCommandRequest{
			Owner: "did:plc:alice", Collection: "social.craftsky.feed.post",
		},
		SelectedURI:  "at://did:plc:alice/social.craftsky.feed.post/3selected",
		SelectedRkey: "3selected",
	}
	if err := validateSelectedAppendIdentity(valid); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*FencedAppendCommandRequest){
		"owner":      func(request *FencedAppendCommandRequest) { request.Command.Owner = "did:plc:bob" },
		"collection": func(request *FencedAppendCommandRequest) { request.Command.Collection = "social.craftsky.feed.project" },
		"rkey":       func(request *FencedAppendCommandRequest) { request.SelectedRkey = "3other" },
	} {
		t.Run(name, func(t *testing.T) {
			request := valid
			mutate(&request)
			if err := validateSelectedAppendIdentity(request); !errors.Is(err, ErrMalformedCommand) {
				t.Fatalf("error=%v, want %v", err, ErrMalformedCommand)
			}
		})
	}
}

func containsJSONField(t *testing.T, raw json.RawMessage, key string, want any) bool {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatal(err)
	}
	return value[key] == want
}
