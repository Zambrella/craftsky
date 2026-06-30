package auth

import "log/slog"

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

func authLogErrorAttrs(runID, operation, category string) []any {
	return append(authLogAttrs(runID, operation),
		slog.String("result", "error"),
		slog.String("error_category", category))
}
