package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"path"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api/envelope"
	craftskylex "social.craftsky/appview/internal/lexicon/craftsky"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/relationships"
)

type likeCommandStore interface {
	DirectedInteractionAuthorizer
	postTargetReader
}

type unlikeCommandStore interface {
	postTargetReader
}

type SetCommandExecutor interface {
	Execute(context.Context, pdscommands.SetCommandRequest) (pdscommands.CommandResult, error)
}

func CommandLikePostHandler(store likeCommandStore, commands SetCommandExecutor, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		runID := middleware.GetRunID(request.Context())
		caller, ok := middleware.GetDID(request.Context())
		if !ok {
			envelope.WriteError(writer, http.StatusInternalServerError, "internal_error", "no did in context", runID, nil)
			return
		}
		generation, ok := requirePDSEffectGeneration(writer, request, runID)
		if !ok {
			return
		}
		operationKey, ok := requireCommandOperationKey(writer, request, runID)
		if !ok {
			return
		}
		if err := rejectNonEmptyBody(request); err != nil {
			envelope.WriteError(writer, http.StatusBadRequest, "unexpected_field", "request body rejected", runID, nil)
			return
		}
		targetDID, target, ok := resolveLikeCommandTarget(writer, request, store, runID)
		if !ok {
			return
		}
		if !authorizeDirectedInteraction(writer, request, store, caller, targetDID, relationships.OperationLikeCreate) {
			return
		}
		rkey, err := newImmediateRecordKey()
		if err != nil {
			envelope.WriteError(writer, http.StatusInternalServerError, "internal_error", "could not prepare like", runID, nil)
			return
		}
		result, err := executeLikeCommand(request, commands, caller, generation, targetDID, target, operationKey, rkey, true, runID)
		if err != nil {
			logger.Warn("like command failed", slog.Any("error", err))
			WriteCommandError(writer, runID, err)
			return
		}
		WriteCommandResponse(writer, CommandResultFromStored(result))
	})
}

func CommandUnlikePostHandler(store unlikeCommandStore, commands SetCommandExecutor, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		runID := middleware.GetRunID(request.Context())
		caller, ok := middleware.GetDID(request.Context())
		if !ok {
			envelope.WriteError(writer, http.StatusInternalServerError, "internal_error", "no did in context", runID, nil)
			return
		}
		generation, ok := requirePDSEffectGeneration(writer, request, runID)
		if !ok {
			return
		}
		operationKey, ok := requireCommandOperationKey(writer, request, runID)
		if !ok {
			return
		}
		targetDID, target, ok := resolveLikeCommandTarget(writer, request, store, runID)
		if !ok {
			return
		}
		result, err := executeLikeCommand(request, commands, caller, generation, targetDID, target, operationKey, "", false, runID)
		if err != nil {
			logger.Warn("unlike command failed", slog.Any("error", err))
			WriteCommandError(writer, runID, err)
			return
		}
		WriteCommandResponse(writer, CommandResultFromStored(result))
	})
}

func executeLikeCommand(
	request *http.Request,
	commands SetCommandExecutor,
	caller syntax.DID,
	generation int64,
	targetDID syntax.DID,
	target *PostTargetRef,
	operationKey uuid.UUID,
	selectedRkey syntax.RecordKey,
	desiredActive bool,
	runID string,
) (pdscommands.CommandResult, error) {
	if commands == nil {
		return pdscommands.CommandResult{}, pdscommands.ErrDispatchUnavailable
	}
	intent, err := json.Marshal(struct {
		SubjectURI string `json:"subjectUri"`
		SubjectCID string `json:"subjectCid"`
	}{SubjectURI: target.URI, SubjectCID: target.CID})
	if err != nil {
		return pdscommands.CommandResult{}, err
	}
	operationKind := "post.like"
	if !desiredActive {
		operationKind = "post.unlike"
	}
	sessionID, _ := middleware.GetOAuthSessionID(request.Context())
	return commands.Execute(request.Context(), pdscommands.SetCommandRequest{
		Owner: caller, OwnerGeneration: generation, Target: targetDID, SessionID: sessionID,
		OperationKind: operationKind, OperationKey: operationKey,
		Collection: syntax.NSID(craftskyLikeNSID), DesiredActive: desiredActive,
		Intent: intent, SelectedRkey: selectedRkey,
		Matches: func(record pdscommands.AuthoritativeRecord) bool {
			like, _, valid := decodeAuthoritativeLike(record)
			return valid && like.Subject.Uri == target.URI
		},
		CreateRecord: func(createdAt time.Time) (json.RawMessage, error) {
			return json.Marshal(likeRecordBody(target, createdAt))
		},
		AcceptedPresent: func(record pdscommands.AuthoritativeRecord, created bool) (pdscommands.TerminalResult, error) {
			like, createdAt, valid := decodeAuthoritativeLike(record)
			if !valid {
				return pdscommands.TerminalResult{}, pdscommands.ErrMalformedCommand
			}
			status := http.StatusOK
			if created {
				status = http.StatusCreated
			}
			body, err := json.Marshal(InteractionWriteResponse{
				URI: record.URI.String(), CID: record.CID.String(), Rkey: path.Base(record.URI.String()),
				Subject: ResponseStrongRef{URI: like.Subject.Uri, CID: like.Subject.Cid}, CreatedAt: createdAt,
			})
			return pdscommands.TerminalResult{State: pdscommands.CommandAccepted, HTTPStatus: status, ResponseBody: body}, err
		},
		AcceptedAbsent: func() pdscommands.TerminalResult {
			return pdscommands.TerminalResult{State: pdscommands.CommandAccepted, HTTPStatus: http.StatusNoContent}
		},
		Rejected: func(err error) pdscommands.TerminalResult { return RejectedCommandResult(runID, err) },
	})
}

func decodeAuthoritativeLike(record pdscommands.AuthoritativeRecord) (craftskylex.FeedLike, time.Time, bool) {
	var like craftskylex.FeedLike
	if record.URI == "" || record.CID == "" || json.Unmarshal(record.Record, &like) != nil ||
		like.LexiconTypeID != craftskyLikeNSID || like.Subject == nil || like.Subject.Uri == "" || like.Subject.Cid == "" {
		return craftskylex.FeedLike{}, time.Time{}, false
	}
	createdAt, err := time.Parse(time.RFC3339Nano, like.CreatedAt)
	if err != nil {
		return craftskylex.FeedLike{}, time.Time{}, false
	}
	return like, createdAt, true
}

func requireCommandOperationKey(writer http.ResponseWriter, request *http.Request, runID string) (uuid.UUID, bool) {
	raw := request.Header.Get("Idempotency-Key")
	key, err := uuid.Parse(raw)
	if err != nil || key == uuid.Nil || key.String() != raw {
		envelope.WriteError(writer, http.StatusBadRequest, "invalid_idempotency_key", "Idempotency-Key must be a canonical UUID", runID, nil)
		return uuid.Nil, false
	}
	return key, true
}

func resolveLikeCommandTarget(
	writer http.ResponseWriter,
	request *http.Request,
	store postTargetReader,
	runID string,
) (syntax.DID, *PostTargetRef, bool) {
	targetDID, err := syntax.ParseDID(request.PathValue("did"))
	if err != nil {
		envelope.WriteError(writer, http.StatusBadRequest, "invalid_identifier", "did path segment is not a valid DID", runID, nil)
		return "", nil, false
	}
	target, err := store.ResolvePostTarget(request.Context(), targetDID.String(), request.PathValue("rkey"))
	if errors.Is(err, ErrPostNotFound) {
		envelope.WriteError(writer, http.StatusNotFound, "post_not_found", "post not found", runID, nil)
		return "", nil, false
	}
	if err != nil {
		envelope.WriteError(writer, http.StatusInternalServerError, "internal_error", "could not resolve post", runID, nil)
		return "", nil, false
	}
	return targetDID, target, true
}
