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
)

type followSetCommands struct {
	calls   int
	request pdscommands.SetCommandRequest
	result  pdscommands.CommandResult
	replay  bool
}

func (commands *followSetCommands) Execute(
	_ context.Context,
	request pdscommands.SetCommandRequest,
) (pdscommands.CommandResult, error) {
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
			URI:    syntax.ATURI("at://did:plc:alice/app.bsky.graph.follow/" + request.SelectedRkey.String()),
			CID:    "bafy-follow",
			Record: recordBody,
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

func TestCommandFollowRequiresCanonicalOperationKeyBeforeTargetResolution(t *testing.T) {
	profiles := &fakeFollowProfileStore{row: commandFollowProfile(false, 2)}
	commands := &followSetCommands{}
	handler := api.CommandFollowProfileHandler(
		profiles,
		fakeResolver{didFor: "did:plc:bob", handleFor: "bob.example"},
		commands,
		nilLogger(),
	)

	for _, key := range []string{"", "018F4D5C-7A61-7D40-A1A2-0123456789AB", "not-a-uuid"} {
		request := commandFollowRequest(http.MethodPost, key)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
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
	if commands.calls != 0 || profiles.lastProfileDID != "" {
		t.Fatalf("invalid keys reached target/command: target=%q calls=%d", profiles.lastProfileDID, commands.calls)
	}
}

func TestCommandFollowPreservesTargetMembershipSelfAndBlockChecks(t *testing.T) {
	key := "018f4d5c-7a61-7d40-a1a2-0123456789ab"
	tests := []struct {
		name      string
		method    string
		profiles  *fakeFollowProfileStore
		resolver  fakeResolver
		wantCode  int
		wantError string
	}{
		{
			name: "invalid target", method: http.MethodPost,
			profiles: &fakeFollowProfileStore{}, resolver: fakeResolver{},
			wantCode: http.StatusBadRequest, wantError: "invalid_identifier",
		},
		{
			name: "self follow", method: http.MethodPost,
			profiles: &fakeFollowProfileStore{}, resolver: fakeResolver{didFor: "did:plc:alice"},
			wantCode: http.StatusBadRequest, wantError: "self_follow_not_allowed",
		},
		{
			name: "non-member follow", method: http.MethodPost,
			profiles: &fakeFollowProfileStore{err: api.ErrProfileNotFound}, resolver: fakeResolver{didFor: "did:plc:bob"},
			wantCode: http.StatusNotFound, wantError: "profile_not_found",
		},
		{
			name: "blocked follow", method: http.MethodPost,
			profiles: &fakeFollowProfileStore{row: &api.ProfileRow{DID: "did:plc:bob", IsCraftskyProfile: true, BlockedBy: true}}, resolver: fakeResolver{didFor: "did:plc:bob"},
			wantCode: http.StatusForbidden, wantError: "interaction_blocked",
		},
		{
			name: "self unfollow", method: http.MethodDelete,
			profiles: &fakeFollowProfileStore{}, resolver: fakeResolver{didFor: "did:plc:alice"},
			wantCode: http.StatusBadRequest, wantError: "self_follow_not_allowed",
		},
		{
			name: "non-member unfollow", method: http.MethodDelete,
			profiles: &fakeFollowProfileStore{err: api.ErrProfileNotFound}, resolver: fakeResolver{didFor: "did:plc:bob"},
			wantCode: http.StatusNotFound, wantError: "profile_not_found",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commands := &followSetCommands{}
			var handler http.Handler
			if test.method == http.MethodPost {
				handler = api.CommandFollowProfileHandler(test.profiles, test.resolver, commands, nilLogger())
			} else {
				handler = api.CommandUnfollowProfileHandler(test.profiles, test.resolver, commands, nilLogger())
			}
			request := commandFollowRequest(test.method, key)
			if test.name == "invalid target" {
				request.SetPathValue("handleOrDid", "NOT VALID")
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantCode || commands.calls != 0 {
				t.Fatalf("status=%d calls=%d body=%s", response.Code, commands.calls, response.Body.String())
			}
			var body envelope.Error
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Error != test.wantError {
				t.Fatalf("error = %q, want %q", body.Error, test.wantError)
			}
		})
	}
}

func TestCommandFollowAndUnfollowPreserveProfileResponseAndCommandIntent(t *testing.T) {
	key := "018f4d5c-7a61-7d40-a1a2-0123456789ab"
	tests := []struct {
		name               string
		method             string
		initialFollowing   bool
		initialCount       int
		wantFollowing      bool
		wantCount          int
		wantOperationKind  string
		wantDesiredActive  bool
		assertSelectedRkey bool
	}{
		{
			name: "follow", method: http.MethodPost,
			initialCount: 2, wantFollowing: true, wantCount: 3,
			wantOperationKind: "profile.follow", wantDesiredActive: true, assertSelectedRkey: true,
		},
		{
			name: "unfollow", method: http.MethodDelete,
			initialFollowing: true, initialCount: 2, wantCount: 1,
			wantOperationKind: "profile.unfollow",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			profiles := &fakeFollowProfileStore{row: commandFollowProfile(test.initialFollowing, test.initialCount)}
			commands := &followSetCommands{}
			var handler http.Handler
			if test.wantDesiredActive {
				handler = api.CommandFollowProfileHandler(profiles, fakeResolver{didFor: "did:plc:bob", handleFor: "bob.example"}, commands, nilLogger())
			} else {
				handler = api.CommandUnfollowProfileHandler(profiles, fakeResolver{didFor: "did:plc:bob", handleFor: "bob.example"}, commands, nilLogger())
			}

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, commandFollowRequest(test.method, key))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
			}
			var profile api.ProfileResponse
			if err := json.Unmarshal(response.Body.Bytes(), &profile); err != nil {
				t.Fatal(err)
			}
			if profile.DID != "did:plc:bob" || profile.Handle != "bob.example" ||
				profile.ViewerIsFollowing != test.wantFollowing || profile.FollowerCount == nil || *profile.FollowerCount != test.wantCount {
				t.Fatalf("profile response = %+v", profile)
			}
			request := commands.request
			if commands.calls != 1 || request.OperationKind != test.wantOperationKind ||
				request.OperationKey != uuid.MustParse(key) || request.DesiredActive != test.wantDesiredActive ||
				request.Owner != "did:plc:alice" || request.OwnerGeneration != 1 ||
				request.Target != "did:plc:bob" || request.SessionID != "session-alice" ||
				request.Collection != "app.bsky.graph.follow" || string(request.Intent) != `{"targetDid":"did:plc:bob"}` {
				t.Fatalf("command request = %+v calls=%d", request, commands.calls)
			}
			if test.assertSelectedRkey {
				if _, err := syntax.ParseTID(request.SelectedRkey.String()); err != nil {
					t.Fatalf("selected rkey %q is not a TID: %v", request.SelectedRkey, err)
				}
			} else if request.SelectedRkey != "" {
				t.Fatalf("unfollow selected rkey = %q", request.SelectedRkey)
			}
		})
	}
}

func TestCommandFollowMatchesOnlyValidAuthoritativeTargetRecords(t *testing.T) {
	commands := &followSetCommands{}
	handler := api.CommandFollowProfileHandler(
		&fakeFollowProfileStore{row: commandFollowProfile(false, 0)},
		fakeResolver{didFor: "did:plc:bob", handleFor: "bob.example"},
		commands,
		nilLogger(),
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, commandFollowRequest(http.MethodPost, "018f4d5c-7a61-7d40-a1a2-0123456789ab"))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
	}

	valid := pdscommands.AuthoritativeRecord{
		URI: "at://did:plc:alice/app.bsky.graph.follow/external", CID: "bafy-valid",
		Record: json.RawMessage(`{"$type":"app.bsky.graph.follow","subject":"did:plc:bob","createdAt":"2026-09-24T12:00:00Z"}`),
	}
	if !commands.request.Matches(valid) {
		t.Fatal("valid target follow did not match")
	}
	for name, record := range map[string]pdscommands.AuthoritativeRecord{
		"wrong target": {URI: valid.URI, CID: valid.CID, Record: json.RawMessage(`{"$type":"app.bsky.graph.follow","subject":"did:plc:carol","createdAt":"2026-09-24T12:00:00Z"}`)},
		"wrong type":   {URI: valid.URI, CID: valid.CID, Record: json.RawMessage(`{"$type":"app.bsky.graph.block","subject":"did:plc:bob","createdAt":"2026-09-24T12:00:00Z"}`)},
		"invalid did":  {URI: valid.URI, CID: valid.CID, Record: json.RawMessage(`{"$type":"app.bsky.graph.follow","subject":"not-a-did","createdAt":"2026-09-24T12:00:00Z"}`)},
		"invalid time": {URI: valid.URI, CID: valid.CID, Record: json.RawMessage(`{"$type":"app.bsky.graph.follow","subject":"did:plc:bob","createdAt":"yesterday"}`)},
		"missing cid":  {URI: valid.URI, Record: valid.Record},
	} {
		t.Run(name, func(t *testing.T) {
			if commands.request.Matches(record) {
				t.Fatalf("invalid record matched: %s", record.Record)
			}
		})
	}
}

func TestCommandFollowReplaysExactPersistedProfileResponse(t *testing.T) {
	count := 2
	profiles := &fakeFollowProfileStore{row: commandFollowProfile(false, count)}
	commands := &followSetCommands{}
	handler := api.CommandFollowProfileHandler(
		profiles,
		fakeResolver{didFor: "did:plc:bob", handleFor: "bob.example"},
		commands,
		nilLogger(),
	)
	key := "018f4d5c-7a61-7d40-a1a2-0123456789ab"
	first := httptest.NewRecorder()
	handler.ServeHTTP(first, commandFollowRequest(http.MethodPost, key))
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d, body=%s", first.Code, first.Body.String())
	}

	profiles.row.DisplayName = stringPointer("changed after acceptance")
	profiles.row.ViewerIsFollowing = true
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, commandFollowRequest(http.MethodPost, key))
	if second.Code != http.StatusOK || second.Body.String() != first.Body.String() {
		t.Fatalf("replay status=%d body=%q, want exact %q", second.Code, second.Body.String(), first.Body.String())
	}
}

func TestCommandFollowReturnsExactAmbiguousContract(t *testing.T) {
	commands := &recordingSetCommands{result: pdscommands.CommandResult{
		TerminalResult: pdscommands.TerminalResult{State: pdscommands.CommandAmbiguous}, RetryAfterSeconds: 3,
	}}
	handler := api.CommandFollowProfileHandler(
		&fakeFollowProfileStore{row: commandFollowProfile(false, 2)},
		fakeResolver{didFor: "did:plc:bob", handleFor: "bob.example"},
		commands,
		nilLogger(),
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, commandFollowRequest(http.MethodPost, "018f4d5c-7a61-7d40-a1a2-0123456789ab"))
	if response.Code != http.StatusAccepted || response.Header().Get("Retry-After") != "3" ||
		response.Header().Get("Location") != "" || response.Body.String() != "{\"status\":\"ambiguous\"}\n" {
		t.Fatalf("ambiguous response status=%d headers=%v body=%q", response.Code, response.Header(), response.Body.String())
	}
}

func commandFollowRequest(method, key string) *http.Request {
	request := authedReq(method, "/v1/profiles/@bob.example/follows", "", "did:plc:alice")
	request.SetPathValue("handleOrDid", "bob.example")
	request.Header.Set("Idempotency-Key", key)
	return request.WithContext(middleware.WithOAuthSessionID(request.Context(), "session-alice"))
}

func commandFollowProfile(viewerIsFollowing bool, followerCount int) *api.ProfileRow {
	return &api.ProfileRow{
		DID: "did:plc:bob", Crafts: []string{}, CreatedAt: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		IsCraftskyProfile: true, ViewerIsFollowing: viewerIsFollowing, FollowerCount: &followerCount,
	}
}
