// Package federatedhttp provides the AppView's outbound trust boundary for
// OAuth and PDS HTTP traffic.
package federatedhttp

import (
	"context"
	"errors"
	"fmt"
)

// Kind is a bounded error category suitable for control flow, logs, and
// metrics. It intentionally contains no remote URL, host, address, or body.
type Kind string

const (
	KindDestinationRejected Kind = "destination_rejected"
	KindRedirectRejected    Kind = "redirect_rejected"
	KindResponseTooLarge    Kind = "response_too_large"
	KindTimeout             Kind = "timeout"
	KindCanceled            Kind = "canceled"
	KindUpstreamFailure     Kind = "upstream_failure"
)

var (
	ErrDestinationRejected = errors.New("federated http: destination rejected")
	ErrRedirectRejected    = errors.New("federated http: redirect rejected")
	ErrResponseTooLarge    = errors.New("federated http: response too large")
)

// Error is a redacted federated-boundary failure. Cause remains available for
// errors.Is/errors.As, but Error never formats it because resolver and network
// errors commonly contain remote topology.
type Error struct {
	Kind    Kind
	Purpose Purpose
	Cause   error
}

func (e *Error) Error() string {
	if e.Purpose != "" {
		return fmt.Sprintf("federated http %s: %s", e.Purpose, e.Kind)
	}
	return fmt.Sprintf("federated http: %s", e.Kind)
}

func (e *Error) Unwrap() error { return e.Cause }

func (e *Error) Is(target error) bool {
	switch target {
	case ErrDestinationRejected:
		return e.Kind == KindDestinationRejected
	case ErrRedirectRejected:
		return e.Kind == KindRedirectRejected
	case ErrResponseTooLarge:
		return e.Kind == KindResponseTooLarge
	default:
		return false
	}
}

// Classify reduces an error to a bounded category. Boundary errors take
// precedence over their underlying causes.
func Classify(err error) Kind {
	var boundaryErr *Error
	if errors.As(err, &boundaryErr) {
		return boundaryErr.Kind
	}
	return classifyCause(err)
}

func classifyCause(err error) Kind {
	if errors.Is(err, context.Canceled) {
		return KindCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return KindTimeout
	}
	var timeout interface{ Timeout() bool }
	if errors.As(err, &timeout) && timeout.Timeout() {
		return KindTimeout
	}
	return KindUpstreamFailure
}

func failure(purpose Purpose, cause error) error {
	return &Error{Kind: classifyCause(cause), Purpose: purpose, Cause: cause}
}
