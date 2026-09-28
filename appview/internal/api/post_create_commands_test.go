package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/pdscommands"
)

func TestCommandCreatePostRequiresCanonicalOperationKeyBeforeDependencies(t *testing.T) {
	commands := &recordingAppendCommands{}
	handler := api.CreatePostHandler(
		&fakePostStore{},
		nil,
		fakeResolver{},
		api.DefaultMediaLimits(),
		nilLogger(),
		api.CreatePostHandlerOptions{Commands: commands},
	)
	request := httptest.NewRequest(http.MethodPost, "/v1/posts", strings.NewReader(`{"text":"hello","sponsored":false}`))
	ctx := middleware.WithDID(request.Context(), syntax.DID("did:plc:alice"))
	ctx = middleware.WithOwnerGeneration(ctx, 1)
	ctx = middleware.WithOAuthSessionID(ctx, "session-alice")
	request = request.WithContext(ctx)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	assertAPIError(t, response, http.StatusBadRequest, "invalid_idempotency_key")
	if commands.calls != 0 {
		t.Fatalf("append command calls = %d, want 0", commands.calls)
	}
}

func TestCommandCreatePostBuildsFrozenAppendAndReturnsAcceptedPost(t *testing.T) {
	createdAt := time.Date(2026, 9, 24, 13, 0, 0, 0, time.UTC)
	commands := &recordingAppendCommands{}
	commands.execute = func(request pdscommands.AppendCommandRequest) (pdscommands.CommandResult, error) {
		record, err := request.BuildRecord(createdAt)
		if err != nil {
			return pdscommands.CommandResult{}, err
		}
		commands.record = record
		terminal, err := request.Accepted(pdscommands.AuthoritativeRecord{
			URI: "at://did:plc:alice/social.craftsky.feed.post/3frozenpost",
			CID: "bafy-frozen-post", Record: record,
		})
		return pdscommands.CommandResult{TerminalResult: terminal}, err
	}
	handler := api.CreatePostHandler(
		&fakePostStore{}, nil, fakeResolver{handleFor: "alice.example"},
		api.DefaultMediaLimits(), nilLogger(), api.CreatePostHandlerOptions{Commands: commands},
	)
	request := httptest.NewRequest(http.MethodPost, "/v1/posts", strings.NewReader(`{"text":"hello","sponsored":true}`))
	request.Header.Set("Idempotency-Key", "018f4d5c-7a61-7d40-a1a2-777777777777")
	ctx := middleware.WithDID(request.Context(), syntax.DID("did:plc:alice"))
	ctx = middleware.WithOwnerGeneration(ctx, 7)
	ctx = middleware.WithOAuthSessionID(ctx, "session-alice")
	request = request.WithContext(ctx)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	if commands.calls != 1 || commands.request.Owner != "did:plc:alice" || commands.request.OwnerGeneration != 7 ||
		commands.request.SessionID != "session-alice" || commands.request.OperationKind != "post.create" ||
		commands.request.OperationKey != uuid.MustParse("018f4d5c-7a61-7d40-a1a2-777777777777") ||
		commands.request.Collection != "social.craftsky.feed.post" {
		t.Fatalf("append request = %+v calls=%d", commands.request, commands.calls)
	}
	var record map[string]any
	if err := json.Unmarshal(commands.record, &record); err != nil {
		t.Fatal(err)
	}
	if record["createdAt"] != createdAt.Format(time.RFC3339Nano) || record["text"] != "hello" {
		t.Fatalf("frozen record = %#v", record)
	}
	var post api.PostResponse
	if err := json.Unmarshal(response.Body.Bytes(), &post); err != nil {
		t.Fatal(err)
	}
	if post.URI != "at://did:plc:alice/social.craftsky.feed.post/3frozenpost" ||
		post.CID != "bafy-frozen-post" || post.Rkey != "3frozenpost" || !post.Sponsored {
		t.Fatalf("accepted post = %+v", post)
	}
}

type recordingAppendCommands struct {
	calls   int
	request pdscommands.AppendCommandRequest
	result  pdscommands.CommandResult
	err     error
	record  json.RawMessage
	execute func(pdscommands.AppendCommandRequest) (pdscommands.CommandResult, error)
}

func (commands *recordingAppendCommands) Execute(
	_ context.Context,
	request pdscommands.AppendCommandRequest,
) (pdscommands.CommandResult, error) {
	commands.calls++
	commands.request = request
	if commands.execute != nil {
		return commands.execute(request)
	}
	return commands.result, commands.err
}

var _ api.AppendCommandExecutor = (*recordingAppendCommands)(nil)
