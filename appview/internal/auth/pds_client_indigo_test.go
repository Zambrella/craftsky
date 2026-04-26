package auth

import (
	"errors"
	"fmt"
	"testing"

	"github.com/bluesky-social/indigo/atproto/atclient"
)

// translateGetRecordError is the error-translation helper the indigo
// adapter uses; the test exercises each branch without making a real
// HTTP call.

func TestTranslateGetRecordError_RecordNotFoundByName(t *testing.T) {
	// Real-world case observed in the OAuth callback: HTTP 400 with a
	// typed `RecordNotFound` error body. atproto PDSes do NOT use HTTP
	// 404 for missing records — the XRPC `error` field is the signal.
	apiErr := &atclient.APIError{
		StatusCode: 400,
		Name:       "RecordNotFound",
		Message:    "Could not locate record: at://did:plc:x/social.craftsky.actor.profile/self",
	}
	if got := translateGetRecordError(apiErr); !errors.Is(got, ErrRecordNotFound) {
		t.Errorf("want ErrRecordNotFound; got %v", got)
	}
}

func TestTranslateGetRecordError_Wrapped(t *testing.T) {
	// indigo may wrap APIError; errors.As must still find it.
	apiErr := &atclient.APIError{StatusCode: 400, Name: "RecordNotFound"}
	wrapped := fmt.Errorf("outer: %w", apiErr)
	if got := translateGetRecordError(wrapped); !errors.Is(got, ErrRecordNotFound) {
		t.Errorf("want ErrRecordNotFound through wrap; got %v", got)
	}
}

func TestTranslateGetRecordError_HTTP404Fallback(t *testing.T) {
	// Some upstreams (or future indigo changes) may return HTTP 404
	// without a body-level `RecordNotFound` name. Treat that as missing
	// too — it's the semantic a plain 404 conveys.
	apiErr := &atclient.APIError{StatusCode: 404}
	if got := translateGetRecordError(apiErr); !errors.Is(got, ErrRecordNotFound) {
		t.Errorf("want ErrRecordNotFound for plain 404; got %v", got)
	}
}

func TestTranslateGetRecordError_OtherErrorPassesThrough(t *testing.T) {
	apiErr := &atclient.APIError{StatusCode: 500, Name: "InternalError"}
	if got := translateGetRecordError(apiErr); errors.Is(got, ErrRecordNotFound) {
		t.Errorf("500 must not translate to ErrRecordNotFound; got %v", got)
	}
}

func TestTranslateGetRecordError_NonAPIErrorPassesThrough(t *testing.T) {
	boom := errors.New("network unreachable")
	if got := translateGetRecordError(boom); errors.Is(got, ErrRecordNotFound) {
		t.Errorf("non-APIError must not translate; got %v", got)
	}
}

func TestTranslateGetRecordError_NilIsNil(t *testing.T) {
	if got := translateGetRecordError(nil); got != nil {
		t.Errorf("nil in → nil out; got %v", got)
	}
}
