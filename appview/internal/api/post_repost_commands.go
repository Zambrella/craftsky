package api

import (
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

type repostCommandStore interface {
	DirectedInteractionAuthorizer
	shareTargetReader
}

type unrepostCommandStore interface {
	postTargetReader
}

func CommandRepostPostHandler(store repostCommandStore, commands SetCommandExecutor, logger *slog.Logger) http.Handler {
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
		targetURI, targetDID, ok := commandPostURI(writer, request, runID)
		if !ok {
			return
		}
		replay, handled := existingSetCommand(writer, request, commands, caller, generation, "post.repost", operationKey, targetURI, runID)
		if handled {
			return
		}
		if replay != nil {
			target, valid := replayPostTarget(replay.Intent)
			if !valid {
				WriteCommandError(writer, runID, pdscommands.ErrIdempotencyConflict, request.Context())
				return
			}
			result, err := executeRepostCommand(request, commands, caller, generation, targetDID, target, operationKey, replay.SelectedRkey, true, runID)
			if err != nil {
				WriteCommandError(writer, runID, err, request.Context())
				return
			}
			WriteCommandResponse(writer, CommandResultFromStored(result))
			return
		}
		targetDID, target, ok := resolveRepostCommandTarget(writer, request, store, runID)
		if !ok {
			return
		}
		if !authorizeDirectedInteraction(writer, request, store, caller, targetDID, relationships.OperationRepostCreate) {
			return
		}
		if target.IsReply {
			envelope.WriteError(writer, http.StatusUnprocessableEntity, "validation_failed", "validation failed", runID,
				map[string]string{"target": "reply posts cannot be reposted"})
			return
		}
		rkey, err := newImmediateRecordKey()
		if err != nil {
			envelope.WriteError(writer, http.StatusInternalServerError, "internal_error", "could not prepare repost", runID, nil)
			return
		}
		postTarget := &PostTargetRef{URI: target.URI, CID: target.CID}
		result, err := executeRepostCommand(request, commands, caller, generation, targetDID, postTarget, operationKey, rkey, true, runID)
		if err != nil {
			logger.Warn("repost command failed", slog.Any("error", err))
			WriteCommandError(writer, runID, err, request.Context())
			return
		}
		WriteCommandResponse(writer, CommandResultFromStored(result))
	})
}

func CommandUnrepostPostHandler(store unrepostCommandStore, commands SetCommandExecutor, logger *slog.Logger) http.Handler {
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
		targetURI, targetDID, ok := commandPostURI(writer, request, runID)
		if !ok {
			return
		}
		replay, handled := existingSetCommand(writer, request, commands, caller, generation, "post.unrepost", operationKey, targetURI, runID)
		if handled {
			return
		}
		if replay != nil {
			target, valid := replayPostTarget(replay.Intent)
			if !valid {
				WriteCommandError(writer, runID, pdscommands.ErrIdempotencyConflict, request.Context())
				return
			}
			result, err := executeRepostCommand(request, commands, caller, generation, targetDID, target, operationKey, "", false, runID)
			if err != nil {
				WriteCommandError(writer, runID, err, request.Context())
				return
			}
			WriteCommandResponse(writer, CommandResultFromStored(result))
			return
		}
		targetDID, target, ok := resolveLikeCommandTarget(writer, request, store, runID)
		if !ok {
			return
		}
		result, err := executeRepostCommand(request, commands, caller, generation, targetDID, target, operationKey, "", false, runID)
		if err != nil {
			logger.Warn("unrepost command failed", slog.Any("error", err))
			WriteCommandError(writer, runID, err, request.Context())
			return
		}
		WriteCommandResponse(writer, CommandResultFromStored(result))
	})
}

func executeRepostCommand(
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
	operationKind := "post.repost"
	if !desiredActive {
		operationKind = "post.unrepost"
	}
	sessionID, _ := middleware.GetOAuthSessionID(request.Context())
	return commands.Execute(request.Context(), pdscommands.SetCommandRequest{
		Owner: caller, OwnerGeneration: generation, Target: targetDID, SessionID: sessionID,
		OperationKind: operationKind, OperationKey: operationKey,
		Collection: syntax.NSID(craftskyRepostNSID), DesiredActive: desiredActive,
		Intent: intent, SelectedRkey: selectedRkey,
		Matches: func(record pdscommands.AuthoritativeRecord) bool {
			repost, _, valid := decodeAuthoritativeRepost(record)
			return valid && repost.Subject.Uri == target.URI
		},
		CreateRecord: func(createdAt time.Time) (json.RawMessage, error) {
			return json.Marshal(repostRecordBody(target, createdAt))
		},
		AcceptedPresent: func(record pdscommands.AuthoritativeRecord, created bool) (pdscommands.TerminalResult, error) {
			repost, createdAt, valid := decodeAuthoritativeRepost(record)
			if !valid {
				return pdscommands.TerminalResult{}, pdscommands.ErrMalformedCommand
			}
			status := http.StatusOK
			if created {
				status = http.StatusCreated
			}
			body, err := json.Marshal(InteractionWriteResponse{
				URI: record.URI.String(), CID: record.CID.String(), Rkey: path.Base(record.URI.String()),
				Subject: ResponseStrongRef{URI: repost.Subject.Uri, CID: repost.Subject.Cid}, CreatedAt: createdAt,
			})
			return pdscommands.TerminalResult{State: pdscommands.CommandAccepted, HTTPStatus: status, ResponseBody: body}, err
		},
		AcceptedAbsent: func() pdscommands.TerminalResult {
			return pdscommands.TerminalResult{State: pdscommands.CommandAccepted, HTTPStatus: http.StatusNoContent}
		},
		Rejected: func(err error) pdscommands.TerminalResult { return RejectedCommandResult(runID, err) },
	})
}

func decodeAuthoritativeRepost(record pdscommands.AuthoritativeRecord) (craftskylex.FeedRepost, time.Time, bool) {
	var repost craftskylex.FeedRepost
	if !validAuthoritativeSetRecord(record, syntax.NSID(craftskyRepostNSID)) || json.Unmarshal(record.Record, &repost) != nil ||
		repost.Subject == nil || repost.Subject.Uri == "" || repost.Subject.Cid == "" {
		return craftskylex.FeedRepost{}, time.Time{}, false
	}
	createdAt, err := time.Parse(time.RFC3339Nano, repost.CreatedAt)
	if err != nil {
		return craftskylex.FeedRepost{}, time.Time{}, false
	}
	return repost, createdAt, true
}

func resolveRepostCommandTarget(
	writer http.ResponseWriter,
	request *http.Request,
	store shareTargetReader,
	runID string,
) (syntax.DID, *ShareTargetRef, bool) {
	targetDID, err := syntax.ParseDID(request.PathValue("did"))
	if err != nil {
		envelope.WriteError(writer, http.StatusBadRequest, "invalid_identifier", "did path segment is not a valid DID", runID, nil)
		return "", nil, false
	}
	target, err := store.ResolveShareTarget(request.Context(), targetDID.String(), request.PathValue("rkey"))
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
