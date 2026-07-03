package observability

import (
	"context"
	"errors"
	"net"

	"github.com/jackc/pgx/v5"
	"social.craftsky/appview/internal/auth"
)

type ClassifiedError struct {
	Category string
	Code     string
	Stage    string
	Result   string
	Message  string
}

func ClassifyError(err error, eventCtx EventContext) ClassifiedError {
	classified := ClassifiedError{
		Category: safeContextString(eventCtx, "error_category", "unexpected"),
		Code:     "appview.unexpected",
		Stage:    safeContextString(eventCtx, "failure_stage", "unexpected"),
		Result:   safeContextString(eventCtx, "result", "error"),
		Message:  "appview unexpected error",
	}
	if err == nil {
		classified.Category = "none"
		classified.Code = "appview.none"
		classified.Message = "appview no error"
		return classified
	}

	switch {
	case errors.Is(err, auth.ErrAuthTokenInvalid):
		classified.Category, classified.Code, classified.Message = "auth", "auth.session_invalid", "auth session invalid"
	case errors.Is(err, auth.ErrCraftskySessionNotFound):
		classified.Category, classified.Code, classified.Message = "auth", "auth.session_not_found", "auth session not found"
	case errors.Is(err, auth.ErrOAuthSessionNotFound):
		classified.Category, classified.Code, classified.Message = "auth", "auth.oauth_session_not_found", "oauth session not found"
	case errors.Is(err, auth.ErrPDSSessionExpired):
		classified.Category, classified.Code, classified.Message = "auth", "auth.pds_session_expired", "pds session expired"
	case errors.Is(err, auth.ErrRecordNotFound):
		classified.Category, classified.Code, classified.Message = "not_found", "pds.record_not_found", "pds record not found"
	case errors.Is(err, context.DeadlineExceeded):
		classified.Category, classified.Code, classified.Message = "timeout", "timeout.deadline_exceeded", "operation timed out"
	case isNetworkError(err):
		classified.Category, classified.Code, classified.Message = "network", "network.unavailable", "network unavailable"
	case errors.Is(err, pgx.ErrNoRows):
		classified.Category, classified.Code, classified.Message = "not_found", "db.no_rows", "database row not found"
	case classified.Category == "validation":
		classified.Code, classified.Message = "appview.validation", "validation failed"
	case classified.Category == "rate_limited":
		classified.Code, classified.Message = "appview.rate_limited", "rate limited"
	case classified.Category == "forbidden":
		classified.Code, classified.Message = "appview.forbidden", "forbidden"
	case classified.Category == "server":
		classified.Code, classified.Message = "appview.server", "server error"
	case safeContextString(eventCtx, "component", "") == "db":
		classified.Category, classified.Code, classified.Message = "db", "db.error", "database error"
	case safeContextString(eventCtx, "component", "") == "tap":
		classified.Category, classified.Code, classified.Message = "tap", "tap.error", "tap error"
	case safeContextString(eventCtx, "component", "") == "indexer":
		classified.Category, classified.Code, classified.Message = "indexer", "indexer.error", "indexer error"
	}
	return classified
}

func isNetworkError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	var opErr *net.OpError
	return errors.As(err, &opErr)
}

func safeContextString(eventCtx EventContext, key string, fallback string) string {
	value, ok := eventCtx[key]
	if !ok {
		return fallback
	}
	switch v := value.(type) {
	case string:
		if forbiddenTelemetryValue(v) || v == "" {
			return fallback
		}
		return v
	default:
		return fallback
	}
}
