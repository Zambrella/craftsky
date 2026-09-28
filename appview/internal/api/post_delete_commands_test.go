package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/pdscommands"
)

const postDeleteTestCID = "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"

func TestCommandDeletePostRequiresKeyAndCurrentCIDBeforeCommand(t *testing.T) {
	commands := &recordingAddressedCommands{}
	handler := api.CommandDeletePostHandler(commands, nilLogger())

	missingKey := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:alice/3deletepost", "", "did:plc:alice")
	missingKey.SetPathValue("did", "did:plc:alice")
	missingKey.SetPathValue("rkey", "3deletepost")
	missingKey.Header.Set("If-Match", postDeleteTestCID)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, missingKey)
	assertAPIError(t, response, http.StatusBadRequest, "invalid_idempotency_key")

	stale := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:alice/3deletepost", "", "did:plc:alice")
	stale.SetPathValue("did", "did:plc:alice")
	stale.SetPathValue("rkey", "3deletepost")
	stale.Header.Set("Idempotency-Key", "018f4d5c-7a61-7d40-a1a2-666666666666")
	stale.Header.Set("If-Match", "not-a-cid")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, stale)
	assertAPIError(t, response, http.StatusConflict, "pds_record_conflict")

	if commands.calls != 0 {
		t.Fatalf("addressed command calls = %d, want 0", commands.calls)
	}
}

func TestCommandDeletePostReturnsExactStoredOutcomes(t *testing.T) {
	tests := []struct {
		name       string
		result     pdscommands.CommandResult
		wantStatus int
		wantBody   string
		wantRetry  string
	}{
		{
			name: "accepted", result: pdscommands.CommandResult{TerminalResult: pdscommands.TerminalResult{
				State: pdscommands.CommandAccepted, HTTPStatus: http.StatusNoContent,
			}}, wantStatus: http.StatusNoContent,
		},
		{
			name: "ambiguous", result: pdscommands.CommandResult{
				TerminalResult: pdscommands.TerminalResult{State: pdscommands.CommandAmbiguous}, RetryAfterSeconds: 2,
			}, wantStatus: http.StatusAccepted, wantBody: "{\"status\":\"ambiguous\"}\n", wantRetry: "2",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commands := &recordingAddressedCommands{result: test.result}
			handler := api.CommandDeletePostHandler(commands, nilLogger())
			request := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:alice/3deletepost", "", "did:plc:alice")
			request.SetPathValue("did", "did:plc:alice")
			request.SetPathValue("rkey", "3deletepost")
			request.Header.Set("Idempotency-Key", "018f4d5c-7a61-7d40-a1a2-666666666666")
			request.Header.Set("If-Match", postDeleteTestCID)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus || response.Body.String() != test.wantBody ||
				response.Header().Get("Retry-After") != test.wantRetry || response.Header().Get("Location") != "" {
				t.Fatalf("status=%d body=%q retry=%q", response.Code, response.Body.String(), response.Header().Get("Retry-After"))
			}
			if commands.calls != 1 || commands.request.OperationKey != uuid.MustParse("018f4d5c-7a61-7d40-a1a2-666666666666") ||
				commands.request.URI != "at://did:plc:alice/social.craftsky.feed.post/3deletepost" ||
				commands.request.ExpectedCID != postDeleteTestCID {
				t.Fatalf("addressed request = %+v calls=%d", commands.request, commands.calls)
			}
		})
	}
}

type recordingAddressedCommands struct {
	calls   int
	request pdscommands.AddressedDeleteCommandRequest
	result  pdscommands.CommandResult
	err     error
}

func (commands *recordingAddressedCommands) Delete(
	_ context.Context,
	request pdscommands.AddressedDeleteCommandRequest,
) (pdscommands.CommandResult, error) {
	commands.calls++
	commands.request = request
	return commands.result, commands.err
}

var _ api.AddressedDeleteCommandExecutor = (*recordingAddressedCommands)(nil)
