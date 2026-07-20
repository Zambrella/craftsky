package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/relationships"
)

type RelationshipMutationService interface {
	Mute(context.Context, syntax.DID, syntax.DID) (relationships.State, error)
	Unmute(context.Context, syntax.DID, syntax.DID) (relationships.State, error)
	Block(context.Context, syntax.DID, syntax.DID, string) (relationships.BlockMutationResult, error)
	Unblock(context.Context, syntax.DID, syntax.DID, string) (relationships.BlockMutationResult, error)
}

type relationshipMutationResponse struct {
	Muted     bool   `json:"muted"`
	Blocking  bool   `json:"blocking"`
	BlockedBy bool   `json:"blockedBy"`
	URI       string `json:"uri,omitempty"`
	CID       string `json:"cid,omitempty"`
	Rkey      string `json:"rkey,omitempty"`
}

type relationshipMutation uint8

const (
	mutationMute relationshipMutation = iota
	mutationUnmute
	mutationBlock
	mutationUnblock
)

func MuteProfileHandler(
	service RelationshipMutationService,
	membership relationships.MembershipLookup,
	resolver HandleResolver,
	logger *slog.Logger,
) http.Handler {
	return relationshipProfileMutationHandler(service, membership, resolver, logger, mutationMute)
}

func UnmuteProfileHandler(
	service RelationshipMutationService,
	membership relationships.MembershipLookup,
	resolver HandleResolver,
	logger *slog.Logger,
) http.Handler {
	return relationshipProfileMutationHandler(service, membership, resolver, logger, mutationUnmute)
}

func BlockProfileHandler(
	service RelationshipMutationService,
	membership relationships.MembershipLookup,
	resolver HandleResolver,
	logger *slog.Logger,
) http.Handler {
	return relationshipProfileMutationHandler(service, membership, resolver, logger, mutationBlock)
}

func UnblockProfileHandler(
	service RelationshipMutationService,
	membership relationships.MembershipLookup,
	resolver HandleResolver,
	logger *slog.Logger,
) http.Handler {
	return relationshipProfileMutationHandler(service, membership, resolver, logger, mutationUnblock)
}

func relationshipProfileMutationHandler(
	service RelationshipMutationService,
	membership relationships.MembershipLookup,
	resolver HandleResolver,
	logger *slog.Logger,
	mutation relationshipMutation,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}

		subject, err := relationships.ResolveTarget(
			r.Context(), r.PathValue("handleOrDid"), owner, resolver, membership,
		)
		if err != nil {
			writeRelationshipTargetError(w, runID, err)
			return
		}

		var state relationships.State
		var block relationships.BlockMutationResult
		switch mutation {
		case mutationMute:
			state, err = service.Mute(r.Context(), owner, subject)
		case mutationUnmute:
			state, err = service.Unmute(r.Context(), owner, subject)
		case mutationBlock:
			sid, _ := middleware.GetOAuthSessionID(r.Context())
			block, err = service.Block(r.Context(), owner, subject, sid)
			state = block.State
		case mutationUnblock:
			sid, _ := middleware.GetOAuthSessionID(r.Context())
			block, err = service.Unblock(r.Context(), owner, subject, sid)
			state = block.State
		}
		if err != nil {
			if logger != nil {
				logger.Error("relationship mutation failed",
					slog.String("operation", mutation.operation()),
					slog.String("run_id", runID),
					slog.String("stage", "mutation"))
			}
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "relationship mutation failed", runID, nil)
			return
		}

		resp := relationshipMutationResponse{
			Muted:     state.Muted,
			Blocking:  state.Blocking,
			BlockedBy: state.BlockedBy,
			URI:       block.URI.String(),
			CID:       block.CID.String(),
			Rkey:      block.Rkey.String(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})
}

func writeRelationshipTargetError(w http.ResponseWriter, runID string, err error) {
	switch {
	case errors.Is(err, relationships.ErrInvalidIdentifier):
		envelope.WriteError(w, http.StatusBadRequest,
			"invalid_identifier", "not a valid handle or DID", runID, nil)
	case errors.Is(err, relationships.ErrSelfRelationship):
		envelope.WriteError(w, http.StatusBadRequest,
			"self_relationship_not_allowed", "cannot target yourself", runID, nil)
	case errors.Is(err, relationships.ErrProfileNotFound):
		envelope.WriteError(w, http.StatusNotFound,
			"profile_not_found", "profile not found", runID, nil)
	case errors.Is(err, relationships.ErrMembershipUnavailable):
		envelope.WriteError(w, http.StatusInternalServerError,
			"internal_error", "membership lookup failed", runID, nil)
	default:
		envelope.WriteError(w, http.StatusBadGateway,
			"identity_unavailable", "could not resolve identity", runID, nil)
	}
}

func (m relationshipMutation) operation() string {
	switch m {
	case mutationMute:
		return "mute.create"
	case mutationUnmute:
		return "mute.delete"
	case mutationBlock:
		return "block.create"
	case mutationUnblock:
		return "block.delete"
	default:
		return "relationship.unknown"
	}
}
