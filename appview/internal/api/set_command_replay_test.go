package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/pdscommands"
)

type replaySetCommands struct {
	recordingSetCommands
	replay *pdscommands.ReplayCommand
}

func (commands *replaySetCommands) LookupCommand(_ context.Context, _ syntax.DID, _ string, _ uuid.UUID) (*pdscommands.ReplayCommand, error) {
	return commands.replay, nil
}

func TestFollowTerminalReplayDoesNotRequireCurrentTargetMembership(t *testing.T) {
	key := "018f4d5c-7a61-7d40-a1a2-0123456789ab"
	body := json.RawMessage(`{"did":"did:plc:bob","viewerIsFollowing":true}`)
	commands := &replaySetCommands{replay: &pdscommands.ReplayCommand{
		OwnerGeneration: 1,
		Intent:          json.RawMessage(`{"targetDid":"did:plc:bob","requestTarget":"bob.example","response":` + string(body) + `}`),
		Result: pdscommands.CommandResult{TerminalResult: pdscommands.TerminalResult{
			State: pdscommands.CommandAccepted, HTTPStatus: http.StatusOK, ResponseBody: body,
		}},
	}}
	handler := api.CommandFollowProfileHandler(&fakeFollowProfileStore{}, fakeResolver{}, commands, nilLogger())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, commandFollowRequest(http.MethodPost, key))
	if response.Code != http.StatusOK || response.Body.String() != string(body) || commands.calls != 0 {
		t.Fatalf("replay status=%d body=%q calls=%d", response.Code, response.Body.String(), commands.calls)
	}

	other := commandFollowRequest(http.MethodPost, key)
	other.SetPathValue("handleOrDid", "other.example")
	conflict := httptest.NewRecorder()
	handler.ServeHTTP(conflict, other)
	if conflict.Code != http.StatusConflict {
		t.Fatalf("changed target status=%d body=%q", conflict.Code, conflict.Body.String())
	}
}

func TestBlockPendingReplayUsesFrozenTargetWithoutRelationshipRead(t *testing.T) {
	key := "018f4d5c-7a61-7d40-a1a2-0123456789ab"
	commands := &replaySetCommands{replay: &pdscommands.ReplayCommand{
		OwnerGeneration: 1, SelectedRkey: "3aaaaaaaaaaa2",
		Intent: json.RawMessage(`{"targetDid":"did:plc:bob","requestTarget":"bob.example","state":{"Muted":true}}`),
		Result: pdscommands.CommandResult{TerminalResult: pdscommands.TerminalResult{State: pdscommands.CommandAmbiguous}, RetryAfterSeconds: 1},
	}, recordingSetCommands: recordingSetCommands{result: pdscommands.CommandResult{
		TerminalResult: pdscommands.TerminalResult{State: pdscommands.CommandAmbiguous}, RetryAfterSeconds: 1,
	}}}
	handler := api.CommandBlockProfileHandler(commandBlockStore{}, fakeResolver{}, commands, nilLogger())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, commandBlockRequest(http.MethodPost, key))
	if response.Code != http.StatusAccepted || commands.calls != 1 || commands.request.Target != "did:plc:bob" ||
		commands.request.SelectedRkey != "3aaaaaaaaaaa2" {
		t.Fatalf("replay status=%d body=%q request=%+v", response.Code, response.Body.String(), commands.request)
	}
}
