package ctxkeys

import (
	"context"
	"fmt"
	"log/slog"
)

type diagnosticBoundaryKey struct{}
type diagnosticOccurrenceKey struct{}

// WithDiagnosticOccurrenceFactory supplies worker job isolation without making
// auth depend on the telemetry implementation.
func WithDiagnosticOccurrenceFactory(ctx context.Context, factory func(context.Context) context.Context) context.Context {
	return context.WithValue(ctx, diagnosticOccurrenceKey{}, factory)
}

// WithDiagnosticOccurrence starts one job's failure boundary. Without telemetry
// wiring, the original context remains usable and no backend is required.
func WithDiagnosticOccurrence(ctx context.Context) context.Context {
	if factory, ok := ctx.Value(diagnosticOccurrenceKey{}).(func(context.Context) context.Context); ok {
		return factory(ctx)
	}
	return ctx
}

// DiagnosticBoundary lets auth retain causes using the request's existing owner
// without importing middleware/observability and forming a dependency cycle.
type DiagnosticBoundary func(context.Context, error, map[string]any) []slog.Attr

func WithDiagnosticBoundary(ctx context.Context, boundary DiagnosticBoundary) context.Context {
	return context.WithValue(ctx, diagnosticBoundaryKey{}, boundary)
}
func DiagnosticFailureAttrs(ctx context.Context, err error, fields map[string]any) []slog.Attr {
	if err == nil {
		return nil
	}
	if boundary, ok := ctx.Value(diagnosticBoundaryKey{}).(DiagnosticBoundary); ok {
		return boundary(ctx, err, fields)
	}
	// No request backend: retain concrete type without formatting unknown prose.
	return []slog.Attr{slog.String("error_type", fmt.Sprintf("%T", err)), slog.String("cause_message", "operation failed")}
}
