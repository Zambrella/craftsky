package ctxkeys

import (
	"context"
	"fmt"
	"log/slog"
)

type diagnosticBoundaryKey struct{}

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
