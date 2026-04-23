// appview/internal/api/profile.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
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
// routing. The mux pattern is registered as /v1/profiles/{handleOrDid}
// (ServeMux does not support literal-prefix wildcards), so clients supply
// the "@" prefix and this handler strips it before resolving the identity.
func GetProfileHandler(store ProfileReader, resolver HandleResolver, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimPrefix(r.PathValue("handleOrDid"), "@")
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

// Constants repeated from internal/auth/initialize_profile.go. Keeping
// them local keeps the `api` package's surface self-contained for handler
// logic; the canonical declarations live in the auth package.
const (
	blueskyProfileNSID  = "app.bsky.actor.profile"
	craftskyProfileNSID = "social.craftsky.actor.profile"
	profileRecordKey    = "self"
)

// PutMeProfileHandler serves PUT /v1/profiles/me.
func PutMeProfileHandler(
	store ProfileReader,
	resolver HandleResolver,
	newPDS auth.PDSClientFactory,
	logger *slog.Logger,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())

		didStr, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		did, derr := syntax.ParseDID(didStr)
		if derr != nil {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "invalid did in context", runID, nil)
			return
		}
		sessionID, _ := middleware.GetOAuthSessionID(r.Context())

		reqBody, err := DecodeProfilePut(r.Body)
		if err != nil {
			if fe, ok := err.(*FieldError); ok {
				envelope.WriteError(w, http.StatusBadRequest, fe.Code,
					"request body rejected", runID, fe.Fields)
				return
			}
			envelope.WriteError(w, http.StatusBadRequest,
				"malformed_body", "could not parse body", runID, nil)
			return
		}
		if err := ValidateProfilePut(reqBody); err != nil {
			if fe, ok := err.(*FieldError); ok {
				envelope.WriteError(w, http.StatusUnprocessableEntity,
					fe.Code, "validation failed", runID, fe.Fields)
				return
			}
			envelope.WriteError(w, http.StatusUnprocessableEntity,
				"validation_failed", "validation failed", runID, nil)
			return
		}

		pds, err := newPDS(r.Context(), did, sessionID)
		if err != nil {
			logger.Error("profile: newPDS failed", slog.String("err", err.Error()))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_unavailable", "could not contact PDS", runID, nil)
			return
		}

		// Read-before-write on Bluesky so we preserve avatar/banner.
		var bsky map[string]any
		if err := pds.GetRecord(r.Context(), did, blueskyProfileNSID, profileRecordKey, &bsky); err != nil {
			logger.Warn("profile: bluesky getRecord failed", slog.String("err", err.Error()))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_read_failed", "could not read current bluesky profile", runID, nil)
			return
		}
		mergedBsky := mergeBlueskyRecord(bsky, reqBody)
		cskyBody := map[string]any{
			"$type":  craftskyProfileNSID,
			"crafts": nonNilStrings(reqBody.Crafts),
		}

		type writeResult struct {
			err error
		}
		var wg sync.WaitGroup
		wg.Add(2)
		bskyRes := make(chan writeResult, 1)
		cskyRes := make(chan writeResult, 1)
		go func() {
			defer wg.Done()
			err := pds.PutRecord(r.Context(), did, blueskyProfileNSID, profileRecordKey, mergedBsky)
			bskyRes <- writeResult{err: err}
		}()
		go func() {
			defer wg.Done()
			err := pds.PutRecord(r.Context(), did, craftskyProfileNSID, profileRecordKey, cskyBody)
			cskyRes <- writeResult{err: err}
		}()
		wg.Wait()
		close(bskyRes)
		close(cskyRes)
		bskyErr := (<-bskyRes).err
		cskyErr := (<-cskyRes).err

		switch {
		case bskyErr == nil && cskyErr == nil:
			handle, herr := resolver.ResolveHandle(r.Context(), did)
			if herr != nil {
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve handle", runID, nil)
				return
			}
			row := syntheticRow(did.String(), mergedBsky, reqBody.Crafts)
			resp := BuildProfileResponse(row, handle.String(), false)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(resp)
		case bskyErr != nil && cskyErr != nil:
			logger.Error("profile: both PDS writes failed",
				slog.String("bsky_err", bskyErr.Error()),
				slog.String("csky_err", cskyErr.Error()))
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_write_failed", "both profile writes failed", runID, nil)
		default:
			logger.Warn("profile: partial PDS write",
				slog.Any("bsky_err", bskyErr), slog.Any("csky_err", cskyErr))
			fields := map[string]string{
				"bsky":     okOrFailed(bskyErr),
				"craftsky": okOrFailed(cskyErr),
			}
			envelope.WriteError(w, http.StatusBadGateway,
				"pds_write_partial", "partial profile write", runID, fields)
		}
	})
}

// mergeBlueskyRecord returns a fresh record body formed from `existing`
// (preserving avatar/banner/etc.) with displayName and description
// overridden by the request. If the request field is nil, the existing
// value is cleared from the output, matching PUT-clears-missing semantics.
func mergeBlueskyRecord(existing map[string]any, req ProfilePutRequest) map[string]any {
	out := map[string]any{"$type": blueskyProfileNSID}
	for k, v := range existing {
		switch k {
		case "$type", "displayName", "description":
			continue
		default:
			out[k] = v
		}
	}
	if req.DisplayName != nil {
		out["displayName"] = *req.DisplayName
	}
	if req.Description != nil {
		out["description"] = *req.Description
	}
	return out
}

// syntheticRow constructs a ProfileRow from the bodies we just wrote,
// used to render the PUT response without a DB round-trip.
func syntheticRow(did string, bsky map[string]any, crafts []string) *ProfileRow {
	row := &ProfileRow{DID: did, Crafts: nonNilStrings(crafts)}
	if dn, ok := bsky["displayName"].(string); ok {
		row.DisplayName = &dn
	}
	if desc, ok := bsky["description"].(string); ok {
		row.Description = &desc
	}
	if av, ok := bsky["avatar"].(map[string]any); ok {
		if cid := blobCID(av); cid != "" {
			row.AvatarCID = &cid
		}
		if mime, ok := av["mimeType"].(string); ok && mime != "" {
			row.AvatarMime = &mime
		}
	}
	if bn, ok := bsky["banner"].(map[string]any); ok {
		if cid := blobCID(bn); cid != "" {
			row.BannerCID = &cid
		}
		if mime, ok := bn["mimeType"].(string); ok && mime != "" {
			row.BannerMime = &mime
		}
	}
	return row
}

func blobCID(blob map[string]any) string {
	ref, ok := blob["ref"].(map[string]any)
	if !ok {
		return ""
	}
	link, _ := ref["$link"].(string)
	return link
}

func nonNilStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

func okOrFailed(err error) string {
	if err == nil {
		return "ok"
	}
	return "failed"
}
