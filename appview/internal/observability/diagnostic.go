package observability

import (
	"context"
	"fmt"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"log/slog"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"

	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/getsentry/sentry-go"
	"github.com/jackc/pgx/v5/pgconn"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/integrations/instagrammeta"
)

// DiagnosticCause contains selected error details, never an unknown error's
// rendered message. Ordinary errors deliberately carry no synthetic stack.
type DiagnosticCause struct {
	Type    string
	Message string
	Code    string
}

var sqlStatePattern = regexp.MustCompile(`^[0-9A-Z]{5}$`)

// DescribeError selects each cause independently. A public target does not
// establish that an exception's message is public.
func DescribeError(err error, eventCtx EventContext) []DiagnosticCause {
	nodes, omitted := boundedErrorNodes(err)
	causes := make([]DiagnosticCause, 0, len(nodes)+1)
	if omitted && len(nodes) >= 8 {
		nodes = nodes[:7]
	}
	for _, current := range nodes {
		cause := DiagnosticCause{Type: fmt.Sprintf("%T", current), Message: ClassifyError(current, eventCtx).Message}
		switch current.(type) {
		case *runtime.TypeAssertionError, *runtime.PanicNilError:
			cause.Message = sanitizeKnownDiagnosticText(current.Error())
		}
		for _, sentinel := range []error{context.DeadlineExceeded, context.Canceled, auth.ErrRecordNotFound, auth.ErrAuthTokenInvalid, auth.ErrCraftskySessionNotFound, auth.ErrOAuthSessionNotFound, auth.ErrPDSSessionExpired} {
			if sameDiagnosticError(current, sentinel) {
				cause.Message = sentinel.Error()
				break
			}
		}
		if pg, ok := current.(*pgconn.PgError); ok && sqlStatePattern.MatchString(pg.Code) {
			cause.Code = pg.Code
			cause.Message = "database error (SQLSTATE " + pg.Code + ")"
		}
		if upstream, ok := current.(*atclient.APIError); ok && upstream.StatusCode >= 100 && upstream.StatusCode <= 599 {
			cause.Code = strconv.Itoa(upstream.StatusCode)
			cause.Message = "PDS request failed (HTTP " + cause.Code + ")"
		}
		if storage, ok := current.(*smithyhttp.ResponseError); ok && storage.Response != nil && storage.Response.Response != nil {
			status := storage.HTTPStatusCode()
			if status >= 100 && status <= 599 {
				cause.Code = strconv.Itoa(status)
				cause.Message = "Storage request failed (HTTP " + cause.Code + ")"
			}
		}
		if provider, ok := current.(*instagrammeta.ProviderError); ok {
			switch provider.Kind() {
			case instagrammeta.ProviderErrorTransient, instagrammeta.ProviderErrorAuthentication, instagrammeta.ProviderErrorRateLimited, instagrammeta.ProviderErrorNotFound, instagrammeta.ProviderErrorInvalidResponse, instagrammeta.ProviderErrorPermanent:
				cause.Message = "Instagram provider request failed (" + string(provider.Kind()) + ")"
			}
			if provider.StatusCode() >= 100 && provider.StatusCode() <= 599 {
				cause.Code = strconv.Itoa(provider.StatusCode())
			}
		}
		causes = append(causes, cause)
	}
	if omitted {
		causes = append(causes, DiagnosticCause{Type: "DiagnosticOmission", Message: "[OMITTED: cyclic or excess causes]"})
	}

	return causes
}

func diagnosticExceptions(causes []DiagnosticCause) []sentry.Exception {
	exceptions := make([]sentry.Exception, 0, len(causes))
	// Sentry orders the root cause first and the outer wrapper last.
	for i := len(causes) - 1; i >= 0; i-- {
		exceptions = append(exceptions, sentry.Exception{Type: causes[i].Type, Value: causes[i].Message})
	}
	return exceptions
}

// LogDiagnostic formats selected fields only. The supplied error is never
// handed to slog for automatic formatting.
func LogDiagnostic(ctx context.Context, logger *slog.Logger, input DiagnosticInput) {
	level := slog.LevelError
	if input.Context["result"] == "retry" || input.Context["result"] == "expected" {
		level = slog.LevelWarn
	}
	logDiagnosticAt(ctx, logger, input, level)
}
func logDiagnosticAt(ctx context.Context, logger *slog.Logger, input DiagnosticInput, level slog.Level) {
	if logger == nil || input.Error == nil {
		return
	}
	if input.Workflow == nil {
		input.Workflow = RequestFailureWorkflow(ctx, input.Error)
	}
	attrs := eventContextSlogAttrs(SanitizeEventContext(withDiagnosticCorrelation(ctx, input.Context)))
	attrs = append(attrs, slog.Any("causes", DescribeError(input.Error, input.Context)))
	if selected := workflowForRequest(ctx, input.Workflow); selected != nil {
		attrs = append(attrs, slog.Any("diagnostic", workflowLogValue{selected}))
	}
	logger.Log(ctx, level, "operation failed", attrs...)
}

// The private adapter also keeps selection intact with an injected plain slog
// logger. Production's protected handler recognizes it before resolving it.
type workflowLogValue struct{ workflow WorkflowContext }

func (v workflowLogValue) LogValue() slog.Value {
	return slog.AnyValue(boundSelectedFields(v.workflow.diagnosticFields()))
}

// LogPanic preserves selected recovered detail and a recovery-time stack in
// local output even when no remote reporter exists.
func LogPanic(ctx context.Context, logger *slog.Logger, eventCtx EventContext, recovered any) {
	if logger == nil || recovered == nil {
		return
	}
	exceptions := diagnosticPanicExceptions(recovered, eventCtx)
	attrs := eventContextSlogAttrs(SanitizeEventContext(withDiagnosticCorrelation(ctx, eventCtx)))
	attrs = append(attrs, slog.Any("panic", panicLogRecord{Exceptions: exceptions}))
	message := "panic recovered"
	if eventCtx["component"] == "http" {
		message = "HTTP panic recovered"
	}
	logger.Log(ctx, slog.LevelError, message, attrs...)
}

func diagnosticPanicExceptions(recovered any, eventCtx EventContext) []sentry.Exception {
	exceptions := []sentry.Exception{{Type: fmt.Sprintf("%T", recovered), Value: "panic recovered"}}
	if err, ok := recovered.(error); ok {
		exceptions = diagnosticExceptions(DescribeError(err, eventCtx))
	}
	stack := sentry.NewStacktrace()
	if stack != nil {
		for i := range stack.Frames {
			frame := &stack.Frames[i]
			if frame.AbsPath != "" {
				frame.Filename = filepath.Base(frame.AbsPath)
				frame.AbsPath = ""
			}
		}
	}
	exceptions[len(exceptions)-1].Stacktrace = stack
	return exceptions
}

// Never delegate traversal to errors.Is/As: arbitrary cause graphs may cycle.
func sameDiagnosticError(a, b error) bool {
	return a != nil && b != nil && reflect.TypeOf(a).Comparable() && a == b
}
func boundedErrorNodes(root error) ([]error, bool) {
	pending := []error{root}
	nodes := make([]error, 0, 8)
	omitted := false
	for len(pending) > 0 && len(nodes) < 8 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if current == nil {
			continue
		}
		seen := false
		for _, previous := range nodes {
			if sameDiagnosticError(current, previous) {
				seen = true
				break
			}
		}
		if seen {
			omitted = true
			continue
		}
		nodes = append(nodes, current)
		switch wrapped := current.(type) {
		case interface{ Unwrap() []error }:
			children := wrapped.Unwrap()
			if len(children) > 8 {
				children = children[:8]
				omitted = true
			}
			for i := len(children) - 1; i >= 0; i-- {
				pending = append(pending, children[i])
			}
		case interface{ Unwrap() error }:
			pending = append(pending, wrapped.Unwrap())
		}
	}
	return nodes, omitted || len(pending) > 0
}

func diagnosticHasCause(err, target error) bool {
	nodes, _ := boundedErrorNodes(err)
	for _, node := range nodes {
		if sameDiagnosticError(node, target) {
			return true
		}
	}
	return false
}

// DiagnosticWorkflowAttr preserves the closed, bounded selection even for an
// injected plain logger. A private request cannot authorize public activity.
func DiagnosticWorkflowAttr(ctx context.Context, workflow WorkflowContext) slog.Attr {
	if selected := workflowForRequest(ctx, workflow); selected != nil {
		return slog.Any("diagnostic", workflowLogValue{selected})
	}
	return slog.Attr{}
}
