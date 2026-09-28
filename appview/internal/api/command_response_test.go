package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"social.craftsky/appview/internal/pdscommands"
)

func TestWriteCommandResponsePreservesAcceptedDeleteAndAmbiguousContracts(t *testing.T) {
	tests := []struct {
		name           string
		result         CommandHTTPResult
		wantStatus     int
		wantBody       string
		wantRetryAfter string
	}{
		{
			name: "accepted JSON", result: CommandHTTPResult{
				State: pdscommands.CommandAccepted, HTTPStatus: http.StatusOK,
				Body: json.RawMessage(`{"uri":"at://did:plc:actor/social.craftsky.feed.like/3aaaaaaaaaaa2"}`),
			}, wantStatus: http.StatusOK, wantBody: `{"uri":"at://did:plc:actor/social.craftsky.feed.like/3aaaaaaaaaaa2"}`,
		},
		{
			name: "accepted delete", result: CommandHTTPResult{
				State: pdscommands.CommandAccepted, HTTPStatus: http.StatusNoContent,
			}, wantStatus: http.StatusNoContent,
		},
		{
			name: "ambiguous clamps low retry", result: CommandHTTPResult{
				State: pdscommands.CommandAmbiguous, RetryAfterSeconds: 0,
			}, wantStatus: http.StatusAccepted, wantBody: `{"status":"ambiguous"}` + "\n", wantRetryAfter: "1",
		},
		{
			name: "ambiguous clamps high retry", result: CommandHTTPResult{
				State: pdscommands.CommandAmbiguous, RetryAfterSeconds: 9,
			}, wantStatus: http.StatusAccepted, wantBody: `{"status":"ambiguous"}` + "\n", wantRetryAfter: "5",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			WriteCommandResponse(recorder, test.result)
			response := recorder.Result()
			if response.StatusCode != test.wantStatus || recorder.Body.String() != test.wantBody {
				t.Fatalf("response = %d %q, want %d %q", response.StatusCode, recorder.Body.String(), test.wantStatus, test.wantBody)
			}
			if got := response.Header.Get("Retry-After"); got != test.wantRetryAfter {
				t.Fatalf("Retry-After = %q, want %q", got, test.wantRetryAfter)
			}
			if location := response.Header.Get("Location"); location != "" {
				t.Fatalf("ambiguous response exposed Location %q", location)
			}
		})
	}
}

func TestWriteCommandErrorUsesStandardEnvelopeAndStableCodes(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{pdscommands.ErrIdempotencyConflict, http.StatusConflict, "idempotency_conflict"},
		{pdscommands.ErrRepositoryConflict, http.StatusConflict, "pds_repository_conflict"},
		{pdscommands.ErrAtomicMutationTooLarge, http.StatusUnprocessableEntity, "pds_atomic_mutation_too_large"},
		{pdscommands.ErrMalformedCommand, http.StatusBadRequest, "invalid_request"},
		{pdscommands.ErrDispatchUnavailable, http.StatusServiceUnavailable, "pds_dispatch_unavailable"},
	}
	for _, test := range tests {
		recorder := httptest.NewRecorder()
		WriteCommandError(recorder, "request-command", test.err)
		var body map[string]any
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if recorder.Code != test.wantStatus || body["error"] != test.wantCode || body["requestId"] != "request-command" || body["message"] == "" || len(body) != 3 {
			t.Fatalf("response = %d %+v", recorder.Code, body)
		}
	}

	recorder := httptest.NewRecorder()
	WriteCommandError(recorder, "request-command", errors.New("network secret must not escape"))
	if recorder.Code != http.StatusServiceUnavailable || !json.Valid(recorder.Body.Bytes()) {
		t.Fatalf("unknown response = %d %q", recorder.Code, recorder.Body.String())
	}
}
