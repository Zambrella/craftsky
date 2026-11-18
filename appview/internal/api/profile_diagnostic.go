package api

import (
	"context"
	"log/slog"
	"slices"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/observability"
)

func logProfilePutRejection(ctx context.Context, logger *slog.Logger, did syntax.DID, cause error, stage string, status int, code string) {
	fields := observability.EventContext{
		"component": "api", "operation": "profile.update", "failure_stage": stage,
		"result": "expected", "error_category": "validation", "http_status": status,
	}
	if rejected, ok := cause.(*FieldError); ok {
		code = rejected.Code
		var reasons []string
		switch stage {
		case "decode":
			// The decoder can use arbitrary input keys for unexpected_field.
			if code != "unexpected_field" {
				for _, field := range []string{"links", "removedLinks", "avatar", "banner"} {
					if _, exists := rejected.Fields[field]; exists {
						reasons = append(reasons, field)
					}
				}
			}
		case "validation":
			// The profile validator generates field paths from schema names and
			// numeric indices. The messages themselves stay in the API response.
			for field, message := range rejected.Fields {
				if strings.HasSuffix(field, ".url") {
					if message == duplicateProfileLink {
						field += ".duplicate"
					} else {
						field += ".invalid"
					}
				}
				reasons = append(reasons, field)
			}
			slices.Sort(reasons)
		}
		if len(reasons) > 0 {
			fields["failure_reason"] = strings.Join(reasons[:min(len(reasons), 8)], ",")
		}
	}
	fields["error_code"] = code
	observability.LogDiagnostic(ctx, logger, observability.DiagnosticInput{
		Error: cause, Context: fields, Workflow: observability.PrivateFailureContext(cause, did, ""),
	})
}
