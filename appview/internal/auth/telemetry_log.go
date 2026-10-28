package auth

import (
	"context"
	"errors"
	"github.com/bluesky-social/indigo/atproto/identity"
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
	var registrationFailure *TrustedRegistrationFailure
	if errors.As(err, &registrationFailure) && registrationFailure.Code == RegistrationFailureCanceled {
		fields["result"] = "expected"
	}
	var loginFailure *OAuthLoginDiagnostic
	if errors.As(err, &loginFailure) {
		fields["failure_stage"] = loginFailure.Stage()
		if loginFailure.Stage() == "identity" &&
			(errors.Is(err, identity.ErrHandleNotFound) || errors.Is(err, identity.ErrInvalidHandle) || errors.Is(err, identity.ErrHandleReservedTLD)) {
			fields["error_category"] = "validation"
		}
	}
	var diagnostic *OAuthCallbackDiagnostic
	if errors.As(err, &diagnostic) && diagnostic.Reason() == "provider_callback_error_access_denied" {
		fields["result"] = "expected"
	}
	if errors.As(err, &diagnostic) && errors.Is(err, ErrOAuthFlowInvalid) &&
		(strings.HasPrefix(diagnostic.Reason(), "scope_") || diagnostic.Reason() == "callback_request_unavailable") {
		fields["error_category"] = "validation"
	}
	for _, attr := range ctxkeys.DiagnosticFailureAttrs(ctx, err, fields) {
		attrs = append(attrs, attr)
	}
	result, _ := fields["result"].(string)
	if result == "" {
		result = "error"
	}
	return append(attrs,
		slog.String("result", result),
		slog.String("failure_stage", fields["failure_stage"].(string)),
		slog.String("error_category", fields["error_category"].(string)))
}
