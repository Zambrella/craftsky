package observability

import (
	"context"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"net/url"
	"social.craftsky/appview/internal/ctxkeys"
	"strings"
)

type RequestDiagnosticContext struct {
	Path          string
	RoutePattern  string
	PublicTargets bool
	PublicRecord  PublicRecordContext
}

// A resolver describes the route; it never executes a handler or changes the
// request, authorization, admission, or routing decision.
type RequestDiagnosticResolver interface {
	Resolve(method string, value *url.URL) RequestDiagnosticContext
}

// BoundDiagnosticPath bounds an already classified path; it does not grant
// permission to retain raw URL/path/query content.
func BoundDiagnosticPath(path string) string { return sanitizeKnownDiagnosticText(path) }

type requestDiagnosticContextKey struct{}
type privateDiagnosticContextKey struct{}

// WithPrivateDiagnosticContext marks the whole operation as private, including
// lower dependency reports. Only an actual failure may select account/job IDs.
func WithPrivateDiagnosticContext(ctx context.Context, actor syntax.DID, workflowID string) context.Context {
	return context.WithValue(ctx, privateDiagnosticContextKey{}, privateFailureContext{actor: actor, workflowID: workflowID})
}

func WithRequestDiagnosticContext(ctx context.Context, request RequestDiagnosticContext) context.Context {
	return context.WithValue(ctx, requestDiagnosticContextKey{}, request)
}

func workflowForRequest(ctx context.Context, workflow WorkflowContext) WorkflowContext {
	if _, private := ctx.Value(privateDiagnosticContextKey{}).(privateFailureContext); private {
		if _, failure := workflow.(privateFailureContext); !failure {
			return nil
		}
	}
	if _, ok := workflow.(privateFailureContext); ok {
		return workflow
	}
	if request, ok := ctx.Value(requestDiagnosticContextKey{}).(RequestDiagnosticContext); ok && !request.PublicTargets {
		return nil
	}
	return workflow
}

// ConservativeRequestDiagnostic is the no-catalogue fallback for shared
// callers/tests: only fixed metadata/probes and the version prefix survive.
func ConservativeRequestDiagnostic(value *url.URL) RequestDiagnosticContext {
	result := RequestDiagnosticContext{Path: "/[REDACTED]", RoutePattern: "unmatched"}
	if value == nil {
		return result
	}
	switch value.Path {
	case "/health", "/healthz", "/oauth/callback", "/oauth/client-metadata.json", "/oauth/jwks.json":
		result.Path = value.Path
		return result
	}
	parts := strings.Split(strings.TrimPrefix(value.Path, "/"), "/")
	for i := range parts {
		if i != 0 || parts[i] != "v1" {
			parts[i] = "[REDACTED]"
		}
	}
	result.Path = BoundDiagnosticPath("/" + strings.Join(parts, "/"))
	return result
}

// RequestPublicWorkflow uses only identities selected at the diagnostic request
// boundary. Query/body values and private-route membership are never consulted.
func RequestPublicWorkflow(ctx context.Context) WorkflowContext {
	request, ok := ctx.Value(requestDiagnosticContextKey{}).(RequestDiagnosticContext)
	if !ok || !request.PublicTargets {
		return nil
	}
	selected := request.PublicRecord
	if actor, ok := ctxkeys.GetDID(ctx); ok {
		selected.ActorDID = actor
	}
	return selected
}

// RequestFailureWorkflow selects the authenticated operation account only on a
// cause-bearing private request. Route/body targets and session IDs are unused.
func RequestFailureWorkflow(ctx context.Context, cause error) WorkflowContext {
	if cause == nil {
		return nil
	}
	request, ok := ctx.Value(requestDiagnosticContextKey{}).(RequestDiagnosticContext)
	if !ok {
		return nil
	}
	if request.PublicTargets {
		return RequestPublicWorkflow(ctx)
	}
	actor, ok := ctxkeys.GetDID(ctx)
	if !ok {
		return nil
	}
	return PrivateFailureContext(cause, actor, "")
}
