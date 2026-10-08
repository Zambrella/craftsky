package observability_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"
	"log/slog"
	"net/url"
	"social.craftsky/appview/internal/observability"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/routes"
)

// UT-007 / FR-004, RULE-004 / AC-004, AC-020.
func TestRequestDiagnosticPathPolicy(t *testing.T) {
	catalogue, err := routes.NewV1Catalogue(routes.V1RoutePolicies(routes.EnvDev, routes.Config{}))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ method, path, want string }{
		{"GET", "/v1/posts/did:plc:target/record?cursor=credential#fragment", "/v1/posts/did:plc:target/record"},
		{"GET", "/v1/profiles/@target.example.invalid", "/v1/profiles/@target.example.invalid"},
		{"POST", "/v1/posts/did:plc:private/secret/saves", "/v1/posts/[REDACTED]/[REDACTED]/saves"},
		{"POST", "/v1/posts/did:plc:private/secret/reports", "/v1/posts/[REDACTED]/[REDACTED]/reports"},
		{"POST", "/v1/profiles/@private.example.invalid/mutes", "/v1/profiles/[REDACTED]/mutes"},
		{"GET", "/v1/search/hashtags/private-term/posts", "/v1/search/hashtags/[REDACTED]/posts"},
		{"GET", "/v1/scheduled-post-media/private-media", "/v1/scheduled-post-media/[REDACTED]"},
		{"GET", "/v1/saved-post-folders/private-folder", "/v1/saved-post-folders/[REDACTED]"},
		{"GET", "/v1/unclassified/private-value", "/v1/[REDACTED]/[REDACTED]"},
		{"GET", "/oauth/callback?code=credential&state=secret", "/oauth/callback"},
		{"BREW", "/v1/posts/did:plc:target/record", "/v1/posts/[REDACTED]/[REDACTED]"},
		{"GET", "/v1/posts/did:plc:target/record%2Fsecret", "/v1/posts/[REDACTED]/[REDACTED]/[REDACTED]"},
		{"GET", "/v1/posts/did:plc:target/record%252Fsecret", "/v1/posts/[REDACTED]/[REDACTED]"},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			value, err := url.Parse(tc.path)
			if err != nil {
				t.Fatal(err)
			}
			before := value.String()
			result := catalogue.Resolve(tc.method, value)
			if result.Path != tc.want {
				t.Fatalf("safe path=%q, want %q", result.Path, tc.want)
			}
			if value.String() != before {
				t.Fatal("diagnostic resolver mutated request URL")
			}
		})
	}
}

func TestRequestPrivatePathCannotReintroduceTargetContext(t *testing.T) {
	catalogue, err := routes.NewV1Catalogue(routes.V1RoutePolicies(routes.EnvDev, routes.Config{}))
	if err != nil {
		t.Fatal(err)
	}
	value, _ := url.Parse("/v1/posts/did:plc:private/secret/saves")
	ctx := observability.WithRequestDiagnosticContext(context.Background(), catalogue.Resolve("POST", value))
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{Env: "test", SentryDSN: "https://public@example.invalid/1", SentryTransport: transport})
	input := observability.DiagnosticInput{Error: errors.New("opaque private prose"), Workflow: observability.PublicRecordContext{TargetDID: syntax.DID("did:plc:private"), URI: syntax.ATURI("at://did:plc:private/social.craftsky.feed.post/secret")}, Context: observability.EventContext{"operation": "post.read", "target_did": "did:plc:private"}}
	observability.LogDiagnostic(ctx, logger, input)
	observer.CaptureDiagnostic(ctx, input)
	observer.Flush(time.Second)
	data, _ := json.Marshal(transport.Events())
	if strings.Contains(logs.String()+string(data), "did:plc:private") {
		t.Fatal("path-redacted target reintroduced in companion context")
	}
	if !strings.Contains(logs.String(), "post.read") || len(transport.Events()) != 1 {
		t.Fatal("operational diagnostics lost")
	}
}

// IT-011: an invalid/omitted public path value is also denied companion fields.
func TestRequestInvalidPublicPathCannotReintroduceTarget(t *testing.T) {
	catalogue, err := routes.NewV1Catalogue(routes.V1RoutePolicies(routes.EnvDev, routes.Config{}))
	if err != nil {
		t.Fatal(err)
	}
	value, _ := url.Parse("/v1/posts/invalid-private-target/record")
	request := catalogue.Resolve("GET", value)
	if request.PublicTargets {
		t.Fatal("redacted public segment still authorizes companion targets")
	}
}
