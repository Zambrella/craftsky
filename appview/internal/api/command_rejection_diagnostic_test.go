package api_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/getsentry/sentry-go"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/routes"
)

func TestProfileRejectionRetainsOriginalCauseInsteadOfGeneric502(t *testing.T) {
	for _, alreadyHandled := range []bool{false, true} {
		t.Run(map[bool]string{false: "owns_issue", true: "lower_layer_owned_issue"}[alreadyHandled], func(t *testing.T) {
			var local bytes.Buffer
			logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
			transport := &sentry.MockTransport{}
			observer := observability.New(observability.Config{Env: "test", Logger: logger, LogsEnabled: true, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
			catalogue, err := routes.NewV1Catalogue(routes.V1RoutePolicies(routes.EnvProd, routes.Config{}))
			if err != nil {
				t.Fatal(err)
			}
			commands := &fakeCompoundProfileCommands{put: func(request pdscommands.CompoundPutCommandRequest) (pdscommands.CommandResult, error) {
				cause := errors.Join(pdscommands.ErrDispatchRejected, &atclient.APIError{StatusCode: 400, Name: "InvalidRecord", Message: "PRIVATE_PROFILE_BODY"})
				return pdscommands.CommandResult{TerminalResult: request.Rejected(cause)}, nil
			}}
			profile := newPutHandler(t, &fakeStore{}, &fakePDSForPut{}, fakeResolver{}, commands)
			handler := middleware.Logging(logger, catalogue)(middleware.HTTPMetrics(observer)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if alreadyHandled {
					observability.MarkCaptured(r.Context())
				}
				profile.ServeHTTP(w, r)
			})))
			request := httptest.NewRequest(http.MethodPut, "/v1/profiles/me", strings.NewReader(`{"displayName":"PRIVATE_PROFILE_BODY"}`))
			request = request.WithContext(middleware.WithDID(request.Context(), "did:plc:actor"))
			request.Header.Set("Idempotency-Key", "018f4d5c-7a61-7d40-a1a2-aaaaaaaaaaad")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			observer.Flush(time.Second)
			if response.Code != http.StatusBadGateway || !strings.Contains(response.Body.String(), "pds_write_rejected") {
				t.Fatalf("response changed: %d %s", response.Code, response.Body.String())
			}
			issues := 0
			for _, event := range transport.Events() {
				if len(event.Exception) == 0 {
					continue
				}
				issues++
				if event.Tags["operation"] != "pds.command" || event.Tags["failure_stage"] != "rejection" || event.Tags["provider_code"] != "pds.InvalidRecord" || event.Tags["upstream_status"] != "400" {
					t.Fatalf("original cause replaced: %+v", event.Tags)
				}
				if event.Contexts["correlation"]["run_id"] == nil {
					t.Fatal("request correlation lost")
				}
			}
			wantIssues := 1
			if alreadyHandled {
				wantIssues = 0
			}
			if issues != wantIssues {
				t.Fatalf("issues=%d, want %d", issues, wantIssues)
			}
			for _, want := range []string{"pds.InvalidRecord", "400", "rejection", "run_id"} {
				if !strings.Contains(local.String(), want) {
					t.Fatalf("supporting log missing %s", want)
				}
			}
			serialized, _ := json.Marshal(transport.Events())
			if strings.Contains(local.String()+string(serialized)+response.Body.String(), "PRIVATE_PROFILE_BODY") {
				t.Fatal("private profile text escaped")
			}
		})
	}
}
