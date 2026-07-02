package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRoutePatternUsesMuxPatternOrUnmatched(t *testing.T) {
	matched := httptest.NewRequest(http.MethodGet, "/v1/posts/did:plc:raw/rkey123?cursor=secret", nil)
	matched.Pattern = "GET /v1/posts/{did}/{rkey}"
	if got := RoutePattern(matched); got != "/v1/posts/{did}/{rkey}" {
		t.Fatalf("matched RoutePattern = %q, want /v1/posts/{did}/{rkey}", got)
	}

	unmatched := httptest.NewRequest(http.MethodGet, "/v1/posts/did:plc:raw/rkey123?cursor=secret", nil)
	if got := RoutePattern(unmatched); got != "unmatched" {
		t.Fatalf("unmatched RoutePattern = %q, want unmatched", got)
	}
}

func TestRoutePatternRecorderSharesPatternAcrossDerivedContexts(t *testing.T) {
	outer := WithRoutePatternRecorder(context.Background())
	inner := context.WithValue(outer, struct{}{}, "derived")

	RecordRoutePattern(inner, "/v1/posts/{did}/{rkey}")

	if got := RecordedRoutePattern(outer, "unmatched"); got != "/v1/posts/{did}/{rkey}" {
		t.Fatalf("RecordedRoutePattern = %q, want /v1/posts/{did}/{rkey}", got)
	}
}
