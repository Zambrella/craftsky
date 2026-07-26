package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/relationships"
)

type RelationshipListReader interface {
	ListMutes(context.Context, syntax.DID, int, time.Time, syntax.DID) ([]relationships.ListItem, bool, error)
	ListBlocks(context.Context, syntax.DID, int, time.Time, syntax.DID) ([]relationships.ListItem, bool, error)
}

type relationshipListKind uint8

const (
	relationshipListMutes relationshipListKind = iota
	relationshipListBlocks
)

type relationshipAccountSummary struct {
	DID               syntax.DID    `json:"did"`
	Handle            syntax.Handle `json:"handle"`
	IsCraftskyProfile bool          `json:"isCraftskyProfile"`
	Muted             bool          `json:"muted"`
	Blocking          bool          `json:"blocking"`
	BlockedBy         bool          `json:"blockedBy"`
}

type relationshipListResponse struct {
	Items  []relationshipAccountSummary `json:"items"`
	Cursor string                       `json:"cursor,omitempty"`
}

func ListMutedProfilesHandler(store RelationshipListReader, resolver HandleResolver, logger *slog.Logger) http.Handler {
	return relationshipListHandler(store, resolver, logger, relationshipListMutes)
}

func ListBlockedProfilesHandler(store RelationshipListReader, resolver HandleResolver, logger *slog.Logger) http.Handler {
	return relationshipListHandler(store, resolver, logger, relationshipListBlocks)
}

func relationshipListHandler(
	store RelationshipListReader,
	resolver HandleResolver,
	logger *slog.Logger,
	kind relationshipListKind,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context", runID, nil)
			return
		}
		request, err := ParseRelationshipListRequest(r)
		if err != nil {
			switch {
			case errors.Is(err, ErrInvalidRelationshipLimit):
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_limit", "limit must be between 1 and 100", runID, nil)
			default:
				envelope.WriteError(w, http.StatusBadRequest,
					"invalid_cursor", "cursor could not be decoded", runID, nil)
			}
			return
		}

		var items []relationships.ListItem
		var more bool
		switch kind {
		case relationshipListMutes:
			items, more, err = store.ListMutes(r.Context(), owner, request.Limit, request.AfterCreated, request.AfterSubject)
		case relationshipListBlocks:
			items, more, err = store.ListBlocks(r.Context(), owner, request.Limit, request.AfterCreated, request.AfterSubject)
		}
		if err != nil {
			if logger != nil {
				logger.Error("relationship list failed",
					slog.String("operation", kind.operation()),
					slog.String("run_id", runID),
					slog.String("stage", "store"))
			}
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "relationship list failed", runID, nil)
			return
		}

		summaries := make([]relationshipAccountSummary, 0, len(items))
		for _, item := range items {
			handle, err := resolver.ResolveHandle(r.Context(), item.SubjectDID)
			if err != nil {
				envelope.WriteError(w, http.StatusBadGateway,
					"identity_unavailable", "could not resolve identity", runID, nil)
				return
			}
			summaries = append(summaries, relationshipAccountSummary{
				DID:               item.SubjectDID,
				Handle:            handle,
				IsCraftskyProfile: true,
				Muted:             kind == relationshipListMutes,
				Blocking:          kind == relationshipListBlocks,
			})
		}

		var cursor string
		if more {
			if len(items) == 0 {
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "relationship list cursor failed", runID, nil)
				return
			}
			last := items[len(items)-1]
			cursor, err = EncodeRelationshipCursor(last.CreatedAt, last.SubjectDID)
			if err != nil {
				envelope.WriteError(w, http.StatusInternalServerError,
					"internal_error", "relationship list cursor failed", runID, nil)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(relationshipListResponse{Items: summaries, Cursor: cursor})
	})
}

func (k relationshipListKind) operation() string {
	if k == relationshipListBlocks {
		return "block.list"
	}
	return "mute.list"
}
