package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/relationships"
)

type commandBlockStore struct {
	current map[syntax.DID]bool
	state   relationships.State
}

func (store commandBlockStore) IsCurrentMember(_ context.Context, did syntax.DID) (bool, error) {
	return store.current[did], nil
}

func (store commandBlockStore) State(context.Context, syntax.DID, syntax.DID) (relationships.State, error) {
	return store.state, nil
}

type blockSetCommands struct {
	calls   int
	request pdscommands.SetCommandRequest
	result  pdscommands.CommandResult
	replay  bool
}

func (commands *blockSetCommands) Execute(_ context.Context, request pdscommands.SetCommandRequest) (pdscommands.CommandResult, error) {
	commands.calls++
	commands.request = request
	if commands.replay {
		return commands.result, nil
	}
	var terminal pdscommands.TerminalResult
	if request.DesiredActive {
		recordBody, err := request.CreateRecord(time.Date(2026, 9, 24, 12, 0, 0, 123, time.UTC))
		if err != nil {
			return pdscommands.CommandResult{}, err
		}
		terminal, err = request.AcceptedPresent(pdscommands.AuthoritativeRecord{
			URI: syntax.ATURI("at://did:plc:alice/app.bsky.graph.block/" + request.SelectedRkey.String()),
			CID: "bafy-block", Record: recordBody,
		}, true)
		if err != nil {
			return pdscommands.CommandResult{}, err
		}
	} else {
		terminal = request.AcceptedAbsent()
	}
	commands.result = pdscommands.CommandResult{TerminalResult: terminal}
	commands.replay = true
	return commands.result, nil
}

type blockRestorationSpy struct {
	pairs [][2]syntax.DID
}

func (spy *blockRestorationSpy) EnqueueRelationshipSafetyRestoration(_ context.Context, owner, subject syntax.DID) error {
	spy.pairs = append(spy.pairs, [2]syntax.DID{owner, subject})
	return nil
}

func TestCommandBlockRequiresCanonicalOperationKeyBeforeTargetResolution(t *testing.T) {
	store := commandBlockStore{current: map[syntax.DID]bool{"did:plc:bob": true}}
	commands := &blockSetCommands{}
	handler := api.CommandBlockProfileHandler(store, fakeResolver{didFor: "did:plc:bob"}, commands, nilLogger())

	for _, key := range []string{"", "018F4D5C-7A61-7D40-A1A2-0123456789AB", "not-a-uuid"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, commandBlockRequest(http.MethodPost, key))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("key %q status = %d, body=%s", key, response.Code, response.Body.String())
		}
		var body envelope.Error
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Error != "invalid_idempotency_key" {
			t.Fatalf("key %q error = %q", key, body.Error)
		}
	}
	if commands.calls != 0 {
		t.Fatalf("invalid keys reached command executor %d times", commands.calls)
	}
}

func TestCommandBlockAndUnblockUseAuthoritativeSetContract(t *testing.T) {
	key := "018f4d5c-7a61-7d40-a1a2-0123456789ab"
	store := commandBlockStore{
		current: map[syntax.DID]bool{"did:plc:bob": true},
		state:   relationships.State{Muted: true, BlockedBy: true},
	}

	t.Run("block", func(t *testing.T) {
		commands := &blockSetCommands{}
		handler := api.CommandBlockProfileHandler(store, fakeResolver{didFor: "did:plc:bob"}, commands, nilLogger())
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, commandBlockRequest(http.MethodPost, key))
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body["muted"] != true || body["blocking"] != true || body["blockedBy"] != true ||
			body["cid"] != "bafy-block" || body["rkey"] == "" || body["uri"] == "" {
			t.Fatalf("block response = %#v", body)
		}
		request := commands.request
		if commands.calls != 1 || request.OperationKind != "profile.block" ||
			request.OperationKey != uuid.MustParse(key) || !request.DesiredActive ||
			request.Owner != "did:plc:alice" || request.OwnerGeneration != 1 ||
			request.Target != "did:plc:bob" || request.SessionID != "session-alice" ||
			request.Collection != "app.bsky.graph.block" {
			t.Fatalf("command request = %+v calls=%d", request, commands.calls)
		}
		var intent struct {
			TargetDID     string              `json:"targetDid"`
			RequestTarget string              `json:"requestTarget"`
			State         relationships.State `json:"state"`
		}
		if err := json.Unmarshal(request.Intent, &intent); err != nil || intent.TargetDID != "did:plc:bob" ||
			intent.RequestTarget != "bob.example" || intent.State != store.state {
			t.Fatalf("command intent = %s, err=%v", request.Intent, err)
		}
		if _, err := syntax.ParseTID(request.SelectedRkey.String()); err != nil {
			t.Fatalf("selected rkey %q is not a TID: %v", request.SelectedRkey, err)
		}
	})

	t.Run("unblock", func(t *testing.T) {
		commands := &blockSetCommands{}
		restoration := &blockRestorationSpy{}
		handler := api.CommandUnblockProfileHandler(store, fakeResolver{didFor: "did:plc:bob"}, commands, restoration, nilLogger())
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, commandBlockRequest(http.MethodDelete, key))
		if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
			t.Fatalf("status = %d, body=%q", response.Code, response.Body.String())
		}
		request := commands.request
		if commands.calls != 1 || request.OperationKind != "profile.unblock" || request.DesiredActive || request.SelectedRkey != "" {
			t.Fatalf("command request = %+v calls=%d", request, commands.calls)
		}
		if len(restoration.pairs) != 1 || restoration.pairs[0] != [2]syntax.DID{"did:plc:alice", "did:plc:bob"} {
			t.Fatalf("restoration pairs = %+v", restoration.pairs)
		}
	})
}

func TestCommandBlockMatchesOnlyValidAuthoritativeTargetRecords(t *testing.T) {
	commands := &blockSetCommands{}
	handler := api.CommandBlockProfileHandler(
		commandBlockStore{current: map[syntax.DID]bool{"did:plc:bob": true}},
		fakeResolver{didFor: "did:plc:bob"}, commands, nilLogger(),
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, commandBlockRequest(http.MethodPost, "018f4d5c-7a61-7d40-a1a2-0123456789ab"))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}

	valid := pdscommands.AuthoritativeRecord{
		URI: "at://did:plc:alice/app.bsky.graph.block/3aaaaaaaaaaa2", CID: "bafy-valid",
		Record: json.RawMessage(`{"$type":"app.bsky.graph.block","subject":"did:plc:bob","createdAt":"2026-09-24T12:00:00Z"}`),
	}
	if !commands.request.Matches(valid) {
		t.Fatal("valid target block did not match")
	}
	for name, record := range map[string]pdscommands.AuthoritativeRecord{
		"wrong target": {URI: valid.URI, CID: valid.CID, Record: json.RawMessage(`{"$type":"app.bsky.graph.block","subject":"did:plc:carol","createdAt":"2026-09-24T12:00:00Z"}`)},
		"wrong type":   {URI: valid.URI, CID: valid.CID, Record: json.RawMessage(`{"$type":"app.bsky.graph.follow","subject":"did:plc:bob","createdAt":"2026-09-24T12:00:00Z"}`)},
		"invalid did":  {URI: valid.URI, CID: valid.CID, Record: json.RawMessage(`{"$type":"app.bsky.graph.block","subject":"not-a-did","createdAt":"2026-09-24T12:00:00Z"}`)},
		"invalid time": {URI: valid.URI, CID: valid.CID, Record: json.RawMessage(`{"$type":"app.bsky.graph.block","subject":"did:plc:bob","createdAt":"yesterday"}`)},
		"missing cid":  {URI: valid.URI, Record: valid.Record},
	} {
		t.Run(name, func(t *testing.T) {
			if commands.request.Matches(record) {
				t.Fatalf("invalid record matched: %s", record.Record)
			}
		})
	}
}

func TestCommandBlockReturnsExactAmbiguousContract(t *testing.T) {
	commands := &recordingSetCommands{result: pdscommands.CommandResult{
		TerminalResult: pdscommands.TerminalResult{State: pdscommands.CommandAmbiguous}, RetryAfterSeconds: 3,
	}}
	handler := api.CommandBlockProfileHandler(
		commandBlockStore{current: map[syntax.DID]bool{"did:plc:bob": true}},
		fakeResolver{didFor: "did:plc:bob"}, commands, nilLogger(),
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, commandBlockRequest(http.MethodPost, "018f4d5c-7a61-7d40-a1a2-0123456789ab"))
	if response.Code != http.StatusAccepted || response.Header().Get("Retry-After") != "3" ||
		response.Header().Get("Location") != "" || response.Body.String() != "{\"status\":\"ambiguous\"}\n" {
		t.Fatalf("ambiguous response status=%d headers=%v body=%q", response.Code, response.Header(), response.Body.String())
	}
}

func commandBlockRequest(method, key string) *http.Request {
	request := authedReq(method, "/v1/profiles/bob.example/blocks", "", "did:plc:alice")
	request.SetPathValue("handleOrDid", "bob.example")
	request.Header.Set("Idempotency-Key", key)
	return request.WithContext(middleware.WithOAuthSessionID(request.Context(), "session-alice"))
}
