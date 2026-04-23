// appview/internal/api/profile.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

// ProfileReader is the read surface the profile GET handlers use. The
// concrete production implementation is *ProfileStore. Tests inject a
// fake.
type ProfileReader interface {
	Read(ctx context.Context, did string) (*ProfileRow, error)
}

// GetProfileHandler serves GET /v1/profiles/@{handleOrDid}.
//
// The "{handleOrDid}" path segment arrives URL-decoded by net/http's
// routing; this handler does not strip the leading "@" (the mux pattern
// includes the "@" literally).
func GetProfileHandler(store ProfileReader, resolver HandleResolver, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.PathValue("handleOrDid")
		runID := middleware.GetRunID(r.Context())
		did, err := resolveToDID(r.Context(), raw, resolver)
		if err != nil {
			switch {
			case errors.Is(err, errInvalidIdentifier):
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_identifier", "not a valid handle or DID", runID, nil)
			default:
				logger.Warn("profile: ResolveDID failed",
					slog.String("input", raw),
					slog.String("err", err.Error()),
					slog.String("run_id", runID))
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve identity", runID, nil)
			}
			return
		}
		writeProfileResponse(w, r, store, resolver, did, logger)
	})
}

// GetMeProfileHandler serves GET /v1/profiles/me.
func GetMeProfileHandler(store ProfileReader, resolver HandleResolver, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		didStr, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		did, err := syntax.ParseDID(didStr)
		if err != nil {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "invalid did in context", runID, nil)
			return
		}
		writeProfileResponse(w, r, store, resolver, did, logger)
	})
}

// writeProfileResponse loads the row, resolves the current handle, and
// emits the JSON response. Used by both GET handlers.
func writeProfileResponse(
	w http.ResponseWriter,
	r *http.Request,
	store ProfileReader,
	resolver HandleResolver,
	did syntax.DID,
	logger *slog.Logger,
) {
	runID := middleware.GetRunID(r.Context())
	row, err := store.Read(r.Context(), did.String())
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			envelope.WriteError(w, http.StatusNotFound,
				"profile_not_found", "profile not found", runID, nil)
			return
		}
		logger.Error("profile: store read failed",
			slog.String("did", did.String()),
			slog.String("err", err.Error()),
			slog.String("run_id", runID))
		envelope.WriteError(w, http.StatusInternalServerError,
			"internal_error", "profile read failed", runID, nil)
		return
	}
	handle, err := resolver.ResolveHandle(r.Context(), did)
	if err != nil {
		logger.Warn("profile: ResolveHandle failed",
			slog.String("did", did.String()),
			slog.String("err", err.Error()),
			slog.String("run_id", runID))
		envelope.WriteError(w, http.StatusBadGateway,
			"identity_unavailable", "could not resolve handle", runID, nil)
		return
	}
	resp := BuildProfileResponse(row, handle.String(), true)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// errInvalidIdentifier is used to signal a path-parsing failure back up
// to the handler. Not surfaced beyond this file.
var errInvalidIdentifier = errors.New("invalid identifier")

// resolveToDID parses raw as either a DID (starts with "did:") or a
// handle, and returns the DID either directly or via handle resolution.
func resolveToDID(ctx context.Context, raw string, resolver HandleResolver) (syntax.DID, error) {
	if strings.HasPrefix(raw, "did:") {
		did, err := syntax.ParseDID(raw)
		if err != nil {
			return "", errInvalidIdentifier
		}
		return did, nil
	}
	handle, err := syntax.ParseHandle(raw)
	if err != nil {
		return "", errInvalidIdentifier
	}
	return resolver.ResolveDID(ctx, handle)
}
