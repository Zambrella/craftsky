package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/relationships"
)

const blueskyBlockCollection syntax.NSID = "app.bsky.graph.block"

type relationshipCommandStore interface {
	relationships.MembershipLookup
	State(context.Context, syntax.DID, syntax.DID) (relationships.State, error)
}

func CommandBlockProfileHandler(
	store relationshipCommandStore,
	resolver HandleResolver,
	commands SetCommandExecutor,
	logger *slog.Logger,
) http.Handler {
	return commandBlockProfileHandler(store, resolver, commands, nil, logger, true)
}

func CommandUnblockProfileHandler(
	store relationshipCommandStore,
	resolver HandleResolver,
	commands SetCommandExecutor,
	restoration relationships.RelationshipSafetyRestorationEnqueuer,
	logger *slog.Logger,
) http.Handler {
	return commandBlockProfileHandler(store, resolver, commands, restoration, logger, false)
}

func commandBlockProfileHandler(
	store relationshipCommandStore,
	resolver HandleResolver,
	commands SetCommandExecutor,
	restoration relationships.RelationshipSafetyRestorationEnqueuer,
	logger *slog.Logger,
	desiredActive bool,
) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		runID := middleware.GetRunID(request.Context())
		owner, ok := middleware.GetDID(request.Context())
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
		if desiredActive {
			if err := rejectNonEmptyBody(request); err != nil {
				envelope.WriteError(writer, http.StatusBadRequest, "unexpected_field", "request body rejected", runID, nil)
				return
			}
		}
		kind := "profile.block"
		if !desiredActive {
			kind = "profile.unblock"
		}
		replay, handled := existingSetCommand(writer, request, commands, owner, generation, kind, operationKey, request.PathValue("handleOrDid"), runID)
		if handled {
			return
		}
		if replay != nil {
			var intent struct {
				TargetDID syntax.DID          `json:"targetDid"`
				State     relationships.State `json:"state"`
			}
			if json.Unmarshal(replay.Intent, &intent) != nil || intent.TargetDID == "" {
				WriteCommandError(writer, runID, pdscommands.ErrIdempotencyConflict)
				return
			}
			result, err := executeBlockCommand(request, commands, owner, generation, intent.TargetDID, operationKey, replay.SelectedRkey, desiredActive, intent.State, runID)
			if err != nil {
				WriteCommandError(writer, runID, err)
				return
			}
			if !desiredActive && result.State == pdscommands.CommandAccepted && restoration != nil {
				if err := restoration.EnqueueRelationshipSafetyRestoration(request.Context(), owner, intent.TargetDID); err != nil && logger != nil {
					logger.Error("enqueue unblock relationship restoration failed", slog.Any("error", err))
				}
			}
			WriteCommandResponse(writer, CommandResultFromStored(result))
			return
		}

		subject, err := relationships.ResolveTarget(
			request.Context(), request.PathValue("handleOrDid"), owner, resolver, store,
		)
		if err != nil {
			writeRelationshipTargetError(writer, runID, err)
			return
		}
		state, err := store.State(request.Context(), owner, subject)
		if err != nil {
			envelope.WriteError(writer, http.StatusInternalServerError, "internal_error", "relationship state unavailable", runID, nil)
			return
		}
		selectedRkey := syntax.RecordKey("")
		if desiredActive {
			selectedRkey, err = newImmediateRecordKey()
			if err != nil {
				envelope.WriteError(writer, http.StatusInternalServerError, "internal_error", "could not prepare block", runID, nil)
				return
			}
		}
		result, err := executeBlockCommand(
			request, commands, owner, generation, subject, operationKey,
			selectedRkey, desiredActive, state, runID,
		)
		if err != nil {
			if logger != nil {
				operation := "block"
				if !desiredActive {
					operation = "unblock"
				}
				logger.Warn(operation+" command failed", slog.Any("error", err))
			}
			WriteCommandError(writer, runID, err)
			return
		}
		if !desiredActive && result.State == pdscommands.CommandAccepted && restoration != nil {
			if err := restoration.EnqueueRelationshipSafetyRestoration(request.Context(), owner, subject); err != nil && logger != nil {
				logger.Error("enqueue unblock relationship restoration failed", slog.Any("error", err))
			}
		}
		WriteCommandResponse(writer, CommandResultFromStored(result))
	})
}

func executeBlockCommand(
	request *http.Request,
	commands SetCommandExecutor,
	owner syntax.DID,
	generation int64,
	subject syntax.DID,
	operationKey uuid.UUID,
	selectedRkey syntax.RecordKey,
	desiredActive bool,
	state relationships.State,
	runID string,
) (pdscommands.CommandResult, error) {
	if commands == nil {
		return pdscommands.CommandResult{}, pdscommands.ErrDispatchUnavailable
	}
	intent, err := json.Marshal(struct {
		TargetDID     syntax.DID          `json:"targetDid"`
		RequestTarget string              `json:"requestTarget"`
		State         relationships.State `json:"state"`
	}{TargetDID: subject, RequestTarget: request.PathValue("handleOrDid"), State: state})
	if err != nil {
		return pdscommands.CommandResult{}, err
	}
	operationKind := "profile.block"
	if !desiredActive {
		operationKind = "profile.unblock"
	}
	sessionID, _ := middleware.GetOAuthSessionID(request.Context())
	return commands.Execute(request.Context(), pdscommands.SetCommandRequest{
		Owner: owner, OwnerGeneration: generation, Target: subject, SessionID: sessionID,
		OperationKind: operationKind, OperationKey: operationKey,
		Collection: blueskyBlockCollection, DesiredActive: desiredActive,
		Intent: intent, SelectedRkey: selectedRkey,
		Matches: func(record pdscommands.AuthoritativeRecord) bool {
			block, _, valid := decodeAuthoritativeBlock(record)
			return valid && block.Subject == subject.String()
		},
		CreateRecord: func(createdAt time.Time) (json.RawMessage, error) {
			return json.Marshal(bsky.GraphBlock{
				LexiconTypeID: blueskyBlockCollection.String(),
				Subject:       subject.String(),
				CreatedAt:     createdAt.UTC().Format(time.RFC3339Nano),
			})
		},
		AcceptedPresent: func(record pdscommands.AuthoritativeRecord, _ bool) (pdscommands.TerminalResult, error) {
			block, _, valid := decodeAuthoritativeBlock(record)
			if !valid || block.Subject != subject.String() {
				return pdscommands.TerminalResult{}, pdscommands.ErrMalformedCommand
			}
			responseBody, err := json.Marshal(relationshipMutationResponse{
				Muted: state.Muted, Blocking: true, BlockedBy: state.BlockedBy,
				URI: record.URI.String(), CID: record.CID.String(), Rkey: record.URI.RecordKey().String(),
			})
			if err != nil {
				return pdscommands.TerminalResult{}, err
			}
			return pdscommands.TerminalResult{
				State: pdscommands.CommandAccepted, HTTPStatus: http.StatusOK, ResponseBody: responseBody,
				ResponseHeaders: json.RawMessage(`{"Content-Type":"application/json"}`),
			}, nil
		},
		AcceptedAbsent: func() pdscommands.TerminalResult {
			return pdscommands.TerminalResult{State: pdscommands.CommandAccepted, HTTPStatus: http.StatusNoContent}
		},
		Rejected: func(err error) pdscommands.TerminalResult { return RejectedCommandResult(runID, err) },
	})
}

func decodeAuthoritativeBlock(record pdscommands.AuthoritativeRecord) (bsky.GraphBlock, time.Time, bool) {
	var block bsky.GraphBlock
	if record.URI == "" || record.CID == "" || json.Unmarshal(record.Record, &block) != nil ||
		block.LexiconTypeID != blueskyBlockCollection.String() {
		return bsky.GraphBlock{}, time.Time{}, false
	}
	if _, err := syntax.ParseDID(block.Subject); err != nil {
		return bsky.GraphBlock{}, time.Time{}, false
	}
	createdAt, err := time.Parse(time.RFC3339Nano, block.CreatedAt)
	if err != nil {
		return bsky.GraphBlock{}, time.Time{}, false
	}
	return block, createdAt, true
}
