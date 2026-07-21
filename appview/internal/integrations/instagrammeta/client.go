package instagrammeta

import (
	"context"
	"errors"
	"time"
)

// Client is the complete provider surface needed by the verification worker.
// Implementations must not expose IGSIDs, usernames, message text, tokens, or
// upstream response bodies through returned errors.
type Client interface {
	LookupUsername(ctx context.Context, senderIGSID string) (string, error)
	SendReply(ctx context.Context, senderIGSID, text string) error
}

type ProviderErrorKind string

const (
	ProviderErrorTransient       ProviderErrorKind = "transient"
	ProviderErrorAuthentication  ProviderErrorKind = "authentication"
	ProviderErrorRateLimited     ProviderErrorKind = "rateLimited"
	ProviderErrorNotFound        ProviderErrorKind = "notFound"
	ProviderErrorInvalidResponse ProviderErrorKind = "invalidResponse"
	ProviderErrorPermanent       ProviderErrorKind = "permanent"
)

// ProviderError carries only bounded control-plane facts. It intentionally
// omits the request URL, identity, response body, token, and wrapped transport
// error so ordinary diagnostic formatting cannot disclose provider data.
type ProviderError struct {
	kind       ProviderErrorKind
	retryAfter time.Duration
}

func (e *ProviderError) Error() string {
	return "Instagram provider request failed (" + string(e.kind) + ")"
}

func (e *ProviderError) Kind() ProviderErrorKind {
	return e.kind
}

func (e *ProviderError) RetryAfter() time.Duration {
	return e.retryAfter
}

func ProviderErrorDetails(err error) (kind ProviderErrorKind, retryAfter time.Duration, ok bool) {
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) {
		return "", 0, false
	}
	return providerErr.Kind(), providerErr.RetryAfter(), true
}

func IsRetryableProviderError(err error) bool {
	kind, _, ok := ProviderErrorDetails(err)
	return ok && (kind == ProviderErrorTransient || kind == ProviderErrorRateLimited)
}
