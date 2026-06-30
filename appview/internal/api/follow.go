// appview/internal/api/follow.go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
)

const blueskyFollowCollection = "app.bsky.graph.follow"

// FollowGraphStore is the follow-graph read/write subset handlers need.
type FollowGraphStore interface {
	FindActiveFollow(ctx context.Context, did string, subjectDID string) (*FollowRow, error)
}

// FollowProfileHandler serves POST /v1/profiles/@{handleOrDid}/follows.
func FollowProfileHandler(
	graph FollowGraphStore,
	profiles ProfileReader,
	resolver HandleResolver,
	newPDS auth.PDSClientFactory,
	logger *slog.Logger,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		caller, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}

		target, err := resolveFollowTargetDID(r.Context(), strings.TrimPrefix(r.PathValue("handleOrDid"), "@"), resolver)
		if err != nil {
			switch {
			case errors.Is(err, errInvalidIdentifier):
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_identifier", "not a valid handle or DID", runID, nil)
			default:
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve identity", runID, nil)
			}
			return
		}
		if caller == target {
			envelope.WriteError(w, http.StatusBadRequest,
				"self_follow_not_allowed", "cannot follow yourself", runID, nil)
			return
		}

		active, err := graph.FindActiveFollow(r.Context(), caller.String(), target.String())
		if err != nil {
			logger.Error("follow: active lookup failed",
				apiLogErrorAttrs(runID, "follow.create", "store")...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "follow graph lookup failed", runID, nil)
			return
		}
		if active == nil {
			record := map[string]any{
				"$type":     blueskyFollowCollection,
				"subject":   target.String(),
				"createdAt": time.Now().UTC().Format(time.RFC3339),
			}
			sid, _ := middleware.GetOAuthSessionID(r.Context())
			pds, err := newPDS(r.Context(), caller, sid)
			if err != nil {
				writePDSError(w, http.StatusBadGateway,
					"pds_unavailable", "could not contact PDS", runID, err)
				return
			}
			_, _, err = pds.CreateRecord(r.Context(), caller, blueskyFollowCollection, record)
			if err != nil {
				writePDSError(w, http.StatusBadGateway,
					"pds_write_failed", "could not write follow", runID, err)
				return
			}
		}

		writeFollowProfileResponse(w, r, profiles, resolver, target, followingOverride(true))
	})
}

// UnfollowProfileHandler serves DELETE /v1/profiles/@{handleOrDid}/follows.
func UnfollowProfileHandler(
	graph FollowGraphStore,
	profiles ProfileReader,
	resolver HandleResolver,
	newPDS auth.PDSClientFactory,
	logger *slog.Logger,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		caller, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}

		target, err := resolveFollowTargetDID(r.Context(), strings.TrimPrefix(r.PathValue("handleOrDid"), "@"), resolver)
		if err != nil {
			switch {
			case errors.Is(err, errInvalidIdentifier):
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_identifier", "not a valid handle or DID", runID, nil)
			default:
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve identity", runID, nil)
			}
			return
		}
		if caller == target {
			envelope.WriteError(w, http.StatusBadRequest,
				"self_follow_not_allowed", "cannot unfollow yourself", runID, nil)
			return
		}

		active, err := graph.FindActiveFollow(r.Context(), caller.String(), target.String())
		if err != nil {
			logger.Error("unfollow: active lookup failed",
				apiLogErrorAttrs(runID, "follow.delete", "store")...)
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "follow graph lookup failed", runID, nil)
			return
		}
		if active != nil {
			sid, _ := middleware.GetOAuthSessionID(r.Context())
			pds, err := newPDS(r.Context(), caller, sid)
			if err != nil {
				writePDSError(w, http.StatusBadGateway,
					"pds_unavailable", "could not contact PDS", runID, err)
				return
			}
			if err := pds.DeleteRecord(r.Context(), caller, blueskyFollowCollection, active.Rkey); err != nil {
				if !errors.Is(err, auth.ErrRecordNotFound) {
					writePDSError(w, http.StatusBadGateway,
						"pds_write_failed", "could not delete follow", runID, err)
					return
				}
			}
		}

		writeFollowProfileResponse(w, r, profiles, resolver, target, followingOverride(false))
	})
}

func writeFollowProfileResponse(
	w http.ResponseWriter,
	r *http.Request,
	profiles ProfileReader,
	resolver HandleResolver,
	did syntax.DID,
	overrides ...func(*ProfileRow),
) {
	runID := middleware.GetRunID(r.Context())
	viewerDID := ""
	if viewer, ok := middleware.GetDID(r.Context()); ok {
		viewerDID = viewer.String()
	}
	row, err := profiles.Read(r.Context(), did.String(), viewerDID)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			envelope.WriteError(w, http.StatusNotFound,
				"profile_not_found", "profile not found", runID, nil)
			return
		}
		if errors.Is(err, ErrProfileCountsUnavailable) {
			envelope.WriteError(w, http.StatusInternalServerError,
				"profile_counts_unavailable", "required profile counts unavailable", runID, nil)
			return
		}
		envelope.WriteError(w, http.StatusInternalServerError,
			"internal_error", "profile read failed", runID, nil)
		return
	}
	for _, apply := range overrides {
		apply(row)
	}
	handle, err := resolver.ResolveHandle(r.Context(), did)
	if err != nil {
		envelope.WriteError(w, http.StatusBadGateway,
			"identity_unavailable", "could not resolve handle", runID, nil)
		return
	}
	resp := BuildProfileResponse(row, handle, row.IsCraftskyProfile)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func followingOverride(value bool) func(*ProfileRow) {
	return func(row *ProfileRow) {
		previous := row.ViewerIsFollowing
		row.ViewerIsFollowing = value
		if row.IsCraftskyProfile && previous != value {
			if row.FollowerCount != nil {
				if value {
					next := *row.FollowerCount + 1
					row.FollowerCount = &next
				} else {
					next := *row.FollowerCount
					if next > 0 {
						next--
					}
					row.FollowerCount = &next
				}
			}
		}
	}
}

func resolveFollowTargetDID(ctx context.Context, raw string, resolver HandleResolver) (syntax.DID, error) {
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
