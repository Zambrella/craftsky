package middleware

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/ctxkeys"
)

type suspensionReaderStub struct {
	suspended bool
	err       error
	calls     int
	did       syntax.DID
}

func (s *suspensionReaderStub) IsSuspended(_ context.Context, did syntax.DID) (bool, error) {
	s.calls++
	s.did = did
	return s.suspended, s.err
}

type readTrackingBody struct {
	reads int
}

func (body *readTrackingBody) Read([]byte) (int, error) {
	body.reads++
	return 0, io.EOF
}

func (*readTrackingBody) Close() error { return nil }

func TestModerationEnforcementFailsClosedBeforeDeniedMutationSideEffects(t *testing.T) {
	for _, test := range []struct {
		name       string
		reader     SuspensionReader
		wantStatus int
	}{
		{name: "suspended", reader: &suspensionReaderStub{suspended: true}, wantStatus: http.StatusForbidden},
		{name: "lookup failure", reader: &suspensionReaderStub{err: errors.New("database unavailable")}, wantStatus: http.StatusInternalServerError},
		{name: "reader unavailable", reader: nil, wantStatus: http.StatusInternalServerError},
	} {
		t.Run(test.name, func(t *testing.T) {
			var localWrites, externalCalls, pdsMutations int
			handler := ModerationEnforcement(test.reader, false, nil)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				localWrites++
				externalCalls++
				pdsMutations++
			}))
			body := &readTrackingBody{}
			request := httptest.NewRequest(http.MethodPost, "/v1/posts", nil)
			request.Body = body
			request.ContentLength = 32
			owner := syntax.DID("did:plc:owner")
			request = request.WithContext(ctxkeys.WithDID(request.Context(), owner))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if localWrites != 0 || externalCalls != 0 || pdsMutations != 0 || body.reads != 0 {
				t.Fatalf("local/external/PDS/body reads = %d/%d/%d/%d, want all zero", localWrites, externalCalls, pdsMutations, body.reads)
			}
			if reader, ok := test.reader.(*suspensionReaderStub); ok && (reader.calls != 1 || reader.did != owner) {
				t.Fatalf("suspension lookup calls/DID = %d/%q, want 1/%q", reader.calls, reader.did, owner)
			}
			if !request.Close || response.Header().Get("Connection") != "close" || request.Body != http.NoBody {
				t.Fatalf("rejected body was not detached: close=%t connection=%q body=%T", request.Close, response.Header().Get("Connection"), request.Body)
			}
		})
	}
}

func TestModerationEnforcementAllowsRetainedReportAndUnsuspendedMutation(t *testing.T) {
	for _, test := range []struct {
		name        string
		allowed     bool
		suspended   bool
		path        string
		wantLookups int
	}{
		{name: "suspended report", allowed: true, suspended: true, path: "/v1/posts/did:plc:subject/rkey/reports", wantLookups: 0},
		{name: "unsuspended publication", allowed: false, suspended: false, path: "/v1/posts", wantLookups: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			reader := &suspensionReaderStub{suspended: test.suspended}
			sideEffects := 0
			handler := ModerationEnforcement(reader, test.allowed, nil)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				sideEffects++
				w.WriteHeader(http.StatusNoContent)
			}))
			request := httptest.NewRequest(http.MethodPost, test.path, strings.NewReader(`{"reasonType":"spam"}`))
			request = request.WithContext(ctxkeys.WithDID(request.Context(), syntax.DID("did:plc:owner")))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusNoContent {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
			}
			if sideEffects != 1 || reader.calls != test.wantLookups {
				t.Fatalf("side effects/lookups = %d/%d, want 1/%d", sideEffects, reader.calls, test.wantLookups)
			}
		})
	}
}
