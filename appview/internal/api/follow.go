// appview/internal/api/follow.go
package api

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/relationships"
)

const blueskyFollowCollection syntax.NSID = "app.bsky.graph.follow"

// FollowGraphStore is the follow-graph read/write subset handlers need.
type FollowGraphStore interface {
	FindActiveFollow(ctx context.Context, did string, subjectDID string) (*FollowRow, error)
}

// FollowProfileHandler serves POST /v1/profiles/@{handleOrDid}/follows.
func FollowProfileHandler(
	graph FollowGraphStore,
	profiles ProfileReader,
	resolver HandleResolver,
	newEffects pdseffects.ExecutorFactory,
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
		targetProfile, err := requireFollowTargetMember(r.Context(), profiles, caller, target)
		if err != nil {
			writeFollowTargetMembershipError(w, runID, err)
			return
		}
		authorization := relationships.Authorize(relationships.OperationFollowCreate, relationships.State{
			Muted: targetProfile.Muted, Blocking: targetProfile.Blocking, BlockedBy: targetProfile.BlockedBy,
		}, false)
		if !authorization.Allowed {
			envelope.WriteError(w, http.StatusForbidden,
				"interaction_blocked", "interaction is not allowed across a block", runID, nil)
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
			executor, generation, ok := requireFollowEffectExecutor(w, r, newEffects, caller, runID)
			if !ok {
				return
			}
			expected, err := executor.ResolveExpectedOwners(r.Context(), generation, []syntax.DID{target})
			if err != nil {
				writePDSError(w, http.StatusConflict,
					"pds_write_rejected", "could not authorize follow", runID, err)
				return
			}
			rkey, err := deterministicFollowRecordKey(caller, target)
			if err != nil {
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "could not allocate follow record", runID, nil)
				return
			}
			var current bsky.GraphFollow
			_, readErr := executor.ReadRecord(r.Context(), pdseffects.ReadRecordRequest{
				Owner: caller, OwnerGeneration: generation, ExpectedOwners: expected,
				Collection: blueskyFollowCollection, Rkey: rkey,
			}, &current)
			if readErr == nil {
				if current.Subject != target.String() {
					writePDSError(w, http.StatusConflict,
						"pds_record_conflict", "follow record key is already in use", runID,
						&pdseffects.ConflictError{ExactKey: rkey.String()})
					return
				}
			} else if !errors.Is(readErr, auth.ErrRecordNotFound) {
				writePDSError(w, http.StatusBadGateway,
					"pds_read_failed", "could not read follow", runID, readErr)
				return
			} else {
				operationID, mutationKey := immediateEffectIdentity(runID, "follow.put."+rkey.String())
				_, err = executor.PutRecord(r.Context(), pdseffects.PutRecordRequest{
					OperationID: operationID, MutationKey: mutationKey,
					Owner: caller, OwnerGeneration: generation, ExpectedOwners: expected,
					Collection: blueskyFollowCollection, Rkey: rkey,
					Record: &bsky.GraphFollow{
						LexiconTypeID: blueskyFollowCollection.String(),
						Subject:       target.String(),
						CreatedAt:     time.Now().UTC().Format(time.RFC3339),
					},
				})
				if err != nil {
					writePDSError(w, http.StatusBadGateway,
						"pds_write_failed", "could not write follow", runID, err)
					return
				}
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
	newEffects pdseffects.ExecutorFactory,
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
		if _, err := requireFollowTargetMember(r.Context(), profiles, caller, target); err != nil {
			writeFollowTargetMembershipError(w, runID, err)
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
		executor, generation, ok := requireFollowEffectExecutor(w, r, newEffects, caller, runID)
		if !ok {
			return
		}
		expected, err := executor.ResolveExpectedOwners(r.Context(), generation, []syntax.DID{target})
		if err != nil {
			writePDSError(w, http.StatusConflict,
				"pds_write_rejected", "could not authorize unfollow", runID, err)
			return
		}
		known := make(map[syntax.RecordKey]syntax.CID, 2)
		stableRkey, err := deterministicFollowRecordKey(caller, target)
		if err != nil {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "could not allocate follow record", runID, nil)
			return
		}
		known[stableRkey] = ""
		if active != nil {
			rkey, parseErr := syntax.ParseRecordKey(active.Rkey)
			if parseErr != nil {
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "indexed follow record is invalid", runID, nil)
				return
			}
			known[rkey] = syntax.CID(active.CID)
		}
		rkeys := make([]syntax.RecordKey, 0, len(known))
		for rkey := range known {
			rkeys = append(rkeys, rkey)
		}
		sort.Slice(rkeys, func(i, j int) bool { return rkeys[i].String() < rkeys[j].String() })
		for _, rkey := range rkeys {
			operationID, mutationKey := immediateEffectIdentity(runID, "follow.delete."+rkey.String())
			if _, err := executor.DeleteRecord(r.Context(), pdseffects.DeleteRecordRequest{
				OperationID: operationID, MutationKey: mutationKey,
				Owner: caller, OwnerGeneration: generation, ExpectedOwners: expected,
				Collection: blueskyFollowCollection, Rkey: rkey, ExpectedCID: known[rkey],
			}); err != nil {
				writePDSError(w, http.StatusBadGateway,
					"pds_write_failed", "could not delete follow", runID, err)
				return
			}
		}

		writeFollowProfileResponse(w, r, profiles, resolver, target, followingOverride(false))
	})
}

func requireFollowEffectExecutor(
	w http.ResponseWriter,
	r *http.Request,
	factory pdseffects.ExecutorFactory,
	owner syntax.DID,
	runID string,
) (pdseffects.EffectExecutor, int64, bool) {
	generation, ok := requirePDSEffectGeneration(w, r, runID)
	if !ok {
		return nil, 0, false
	}
	sessionID, ok := middleware.GetOAuthSessionID(r.Context())
	if !ok || strings.TrimSpace(sessionID) == "" {
		envelope.WriteError(w, http.StatusUnauthorized,
			"unauthorized", "authentication required", runID, nil)
		return nil, 0, false
	}
	executor, err := newPDSEffectExecutor(r.Context(), factory, owner, sessionID)
	if err != nil {
		writePDSError(w, http.StatusBadGateway,
			"pds_unavailable", "could not contact PDS", runID, err)
		return nil, 0, false
	}
	return executor, generation, true
}

func deterministicFollowRecordKey(owner, target syntax.DID) (syntax.RecordKey, error) {
	digest := sha256.Sum256([]byte(owner.String() + "\x00" + target.String()))
	return syntax.ParseRecordKey(fmt.Sprintf("craftsky-%x", digest[:16]))
}

func requireFollowTargetMember(ctx context.Context, profiles ProfileReader, caller, target syntax.DID) (*ProfileRow, error) {
	row, err := profiles.Read(ctx, target.String(), caller.String())
	if err != nil {
		return nil, err
	}
	if row == nil || !row.IsCraftskyProfile {
		return nil, ErrProfileNotFound
	}
	return row, nil
}

func writeFollowTargetMembershipError(w http.ResponseWriter, runID string, err error) {
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
