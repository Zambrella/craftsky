package auth

import (
	"context"
	"errors"
	"log/slog"
	"social.craftsky/appview/internal/ctxkeys"
	"strings"
)

func authLogAttrs(runID, operation string) []any {
	attrs := []any{
		slog.String("component", "auth"),
		slog.String("operation", operation),
	}
	if runID != "" {
		attrs = append(attrs, slog.String("run_id", runID))
	}
	return attrs
}

func authLogSuccessAttrs(runID, operation string) []any {
	return append(authLogAttrs(runID, operation), slog.String("result", "success"))
}

func authLogErrorAttrs(ctx context.Context, runID, operation, category string, err error) []any {
	attrs := authLogAttrs(runID, operation)
	fields := map[string]any{"component": "auth", "operation": operation, "error_category": category, "run_id": runID, "failure_stage": category}
	var diagnostic *OAuthCallbackDiagnostic
	if errors.As(err, &diagnostic) && errors.Is(err, ErrOAuthFlowInvalid) && strings.HasPrefix(diagnostic.Reason(), "scope_") {
		fields["error_category"] = "validation"
	}
	for _, attr := range ctxkeys.DiagnosticFailureAttrs(ctx, err, fields) {
		attrs = append(attrs, attr)
	}
	return append(attrs,
		slog.String("result", "error"),
		slog.String("error_category", category))
}
