package app

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/pdscommands"
)

type recordingSuggestionCommands struct {
	request pdscommands.SetCommandRequest
	result  pdscommands.CommandResult
	err     error
}

func (commands *recordingSuggestionCommands) Execute(
	_ context.Context,
	request pdscommands.SetCommandRequest,
) (pdscommands.CommandResult, error) {
	commands.request = request
	return commands.result, commands.err
}

func TestInstagramSuggestionFollowAdapterBuildsAuthoritativeSetCommand(t *testing.T) {
	owner := syntax.DID("did:plc:importer")
	target := syntax.DID("did:plc:target")
	suggestionID := uuid.MustParse("75000000-0000-4000-8000-000000000001")
	operationKey := uuid.MustParse("018f4d5c-7a61-7d40-a1a2-0123456789ab")
	commands := &recordingSuggestionCommands{result: pdscommands.CommandResult{
		TerminalResult:    pdscommands.TerminalResult{State: pdscommands.CommandAmbiguous},
		RetryAfterSeconds: 2,
	}}
	adapter := instagramSuggestionFollowAdapter{commands: commands}

	result, err := adapter.FollowSuggestion(context.Background(), instagram.SuggestionFollowRequest{
		OperationKey: operationKey, RequestID: "request-one", SessionID: "session-one",
		SuggestionID: suggestionID, Owner: owner, Target: target,
		OwnerGeneration: 7, TargetGeneration: 11,
		Rkey:      "3l75000000000040008000000000000001",
		CreatedAt: time.Date(2026, 8, 14, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.State != instagram.SuggestionCommandAmbiguous || result.RetryAfterSeconds != 2 {
		t.Fatalf("result = %+v", result)
	}
	request := commands.request
	if request.OperationKind != "instagram.suggestion.accept" || request.OperationKey != operationKey ||
		request.Owner != owner || request.OwnerGeneration != 7 || request.Target != target ||
		request.TargetGeneration != 11 || request.SessionID != "session-one" ||
		request.Collection != instagramFollowCollection || !request.DesiredActive ||
		request.SelectedRkey != "3l75000000000040008000000000000001" {
		t.Fatalf("set command request = %+v", request)
	}
	if !request.Matches(authoritativeSuggestionFollow(t, owner, "external", target)) {
		t.Fatal("valid external follow did not match semantically")
	}
	external, err := request.AcceptedPresent(authoritativeSuggestionFollow(t, owner, "external", target), false)
	if err != nil || string(external.ResponseBody) != `{"suggestionId":"75000000-0000-4000-8000-000000000001","state":"alreadyFollowing"}` {
		t.Fatalf("external acceptance = %s err=%v", external.ResponseBody, err)
	}
	selected, err := request.AcceptedPresent(authoritativeSuggestionFollow(t, owner, request.SelectedRkey, target), true)
	if err != nil || string(selected.ResponseBody) != `{"suggestionId":"75000000-0000-4000-8000-000000000001","state":"followed"}` {
		t.Fatalf("selected acceptance = %s err=%v", selected.ResponseBody, err)
	}
	record, err := request.CreateRecord(time.Date(2026, 8, 14, 11, 0, 0, 123, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	var follow map[string]any
	if err := json.Unmarshal(record, &follow); err != nil {
		t.Fatal(err)
	}
	if follow["$type"] != instagramFollowCollection.String() || follow["subject"] != target.String() {
		t.Fatalf("created follow = %#v", follow)
	}
	if follow["createdAt"] != "2026-08-14T10:00:00Z" {
		t.Fatalf("createdAt = %#v, want suggestion-stable timestamp", follow["createdAt"])
	}
}

func authoritativeSuggestionFollow(t *testing.T, owner syntax.DID, rkey syntax.RecordKey, target syntax.DID) pdscommands.AuthoritativeRecord {
	t.Helper()
	record, err := json.Marshal(map[string]string{
		"$type": instagramFollowCollection.String(), "subject": target.String(),
		"createdAt": time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatal(err)
	}
	return pdscommands.AuthoritativeRecord{
		URI: syntax.ATURI("at://" + owner.String() + "/" + instagramFollowCollection.String() + "/" + rkey.String()),
		CID: "bafycid", Record: record,
	}
}

var _ apiSetCommandExecutor = (*recordingSuggestionCommands)(nil)

type apiSetCommandExecutor interface {
	Execute(context.Context, pdscommands.SetCommandRequest) (pdscommands.CommandResult, error)
}
