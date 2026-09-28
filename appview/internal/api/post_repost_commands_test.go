package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/pdscommands"
)

func TestCommandRepostPostRequiresCanonicalOperationKeyBeforePDSAccess(t *testing.T) {
	store := &fakePostStore{target: &api.PostTargetRef{
		URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost",
	}}
	commands := &recordingSetCommands{}
	handler := api.CommandRepostPostHandler(store, commands, nilLogger())

	for _, key := range []string{"", "018F4D5C-7A61-7D40-A1A2-0123456789AB", "not-a-uuid"} {
		request := commandRepostRequest(http.MethodPost, key)
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
	if commands.calls != 0 || store.lastShareTargetDID != "" {
		t.Fatalf("invalid keys reached target/command: target=%q calls=%d", store.lastShareTargetDID, commands.calls)
	}
}

func TestCommandRepostAndUnrepostPreserveEndpointResponses(t *testing.T) {
	store := &fakePostStore{target: &api.PostTargetRef{
		URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost",
	}}
	key := "018f4d5c-7a61-7d40-a1a2-0123456789ab"
	tests := []struct {
		name       string
		method     string
		handler    func(*recordingSetCommands) http.Handler
		result     pdscommands.CommandResult
		wantStatus int
		wantKind   string
		wantActive bool
	}{
		{
			name: "repost", method: http.MethodPost,
			handler: func(commands *recordingSetCommands) http.Handler {
				return api.CommandRepostPostHandler(store, commands, nilLogger())
			},
			result: pdscommands.CommandResult{TerminalResult: pdscommands.TerminalResult{
				State: pdscommands.CommandAccepted, HTTPStatus: http.StatusCreated,
				ResponseBody: json.RawMessage(`{"uri":"at://did:plc:alice/social.craftsky.feed.repost/one","cid":"bafyRepost","rkey":"one","subject":{"uri":"at://did:plc:bob/social.craftsky.feed.post/post1","cid":"bafyPost"},"createdAt":"2026-09-23T21:00:00Z"}`),
			}},
			wantStatus: http.StatusCreated, wantKind: "post.repost", wantActive: true,
		},
		{
			name: "unrepost", method: http.MethodDelete,
			handler: func(commands *recordingSetCommands) http.Handler {
				return api.CommandUnrepostPostHandler(store, commands, nilLogger())
			},
			result: pdscommands.CommandResult{TerminalResult: pdscommands.TerminalResult{
				State: pdscommands.CommandAccepted, HTTPStatus: http.StatusNoContent,
			}},
			wantStatus: http.StatusNoContent, wantKind: "post.unrepost", wantActive: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commands := &recordingSetCommands{result: test.result}
			request := commandRepostRequest(test.method, key)
			response := httptest.NewRecorder()
			test.handler(commands).ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, body=%s", response.Code, response.Body.String())
			}
			if test.wantStatus == http.StatusNoContent && response.Body.Len() != 0 {
				t.Fatalf("204 body = %q", response.Body.String())
			}
			if commands.calls != 1 || commands.request.OperationKind != test.wantKind ||
				commands.request.OperationKey != uuid.MustParse(key) || commands.request.DesiredActive != test.wantActive ||
				commands.request.Owner != "did:plc:alice" || commands.request.OwnerGeneration != 1 ||
				commands.request.Target != "did:plc:bob" || commands.request.SessionID != "session-alice" ||
				commands.request.Collection != "social.craftsky.feed.repost" {
				t.Fatalf("command request = %+v calls=%d", commands.request, commands.calls)
			}
		})
	}
}

func TestCommandRepostReturnsExactAmbiguousContract(t *testing.T) {
	store := &fakePostStore{target: &api.PostTargetRef{
		URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost",
	}}
	commands := &recordingSetCommands{result: pdscommands.CommandResult{
		TerminalResult: pdscommands.TerminalResult{State: pdscommands.CommandAmbiguous}, RetryAfterSeconds: 3,
	}}
	request := commandRepostRequest(http.MethodPost, "018f4d5c-7a61-7d40-a1a2-0123456789ab")
	response := httptest.NewRecorder()
	api.CommandRepostPostHandler(store, commands, nilLogger()).ServeHTTP(response, request)
	if response.Code != http.StatusAccepted || response.Header().Get("Retry-After") != "3" ||
		response.Header().Get("Location") != "" || response.Body.String() != "{\"status\":\"ambiguous\"}\n" {
		t.Fatalf("ambiguous response status=%d headers=%v body=%q", response.Code, response.Header(), response.Body.String())
	}
}

func TestCommandRepostRejectsReplyBeforePDSAccess(t *testing.T) {
	store := &fakePostStore{shareTarget: &api.ShareTargetRef{
		URI: "at://did:plc:bob/social.craftsky.feed.post/reply1", CID: "bafyReply", IsReply: true,
	}}
	commands := &recordingSetCommands{}
	request := commandRepostRequest(http.MethodPost, "018f4d5c-7a61-7d40-a1a2-0123456789ab")
	response := httptest.NewRecorder()
	api.CommandRepostPostHandler(store, commands, nilLogger()).ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity || commands.calls != 0 {
		t.Fatalf("status=%d calls=%d body=%s", response.Code, commands.calls, response.Body.String())
	}
}

func commandRepostRequest(method, key string) *http.Request {
	request := authedPostPathReq(method, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	request.Header.Set("Idempotency-Key", key)
	return request.WithContext(middleware.WithOAuthSessionID(request.Context(), "session-alice"))
}
