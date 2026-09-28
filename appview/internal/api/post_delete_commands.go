package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/pdscommands"
)

type AddressedDeleteCommandExecutor interface {
	Delete(context.Context, pdscommands.AddressedDeleteCommandRequest) (pdscommands.CommandResult, error)
}

type AddressedCommandExecutor interface {
	AddressedDeleteCommandExecutor
	AddressedPutCommandExecutor
}

func CommandDeletePostHandler(commands AddressedDeleteCommandExecutor, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		runID := middleware.GetRunID(request.Context())
		caller, ok := middleware.GetDID(request.Context())
		if !ok {
			envelope.WriteError(writer, http.StatusInternalServerError, "internal_error", "no did in context", runID, nil)
			return
		}
		ownerGeneration, ok := requirePDSEffectGeneration(writer, request, runID)
		if !ok {
			return
		}
		operationKey, ok := requireCommandOperationKey(writer, request, runID)
		if !ok {
			return
		}
		owner, err := syntax.ParseDID(request.PathValue("did"))
		if err != nil {
			envelope.WriteError(writer, http.StatusBadRequest, "invalid_identifier", "did path segment is not a valid DID", runID, nil)
			return
		}
		if owner != caller {
			envelope.WriteError(writer, http.StatusForbidden, "forbidden", "cannot delete another user's post", runID, nil)
			return
		}
		rkey, err := syntax.ParseRecordKey(request.PathValue("rkey"))
		if err != nil {
			envelope.WriteError(writer, http.StatusBadRequest, "invalid_identifier", "rkey path segment is not a valid record key", runID, nil)
			return
		}
		expectedCID, err := ParseBusinessIfMatch(request)
		if err != nil || expectedCID == "*" {
			WritePDSRecordConflict(writer, runID)
			return
		}
		uri := syntax.ATURI("at://" + owner.String() + "/" + craftskyPostNSID + "/" + rkey.String())
		intent, err := json.Marshal(struct {
			URI         syntax.ATURI `json:"uri"`
			ExpectedCID syntax.CID   `json:"expectedCid"`
		}{URI: uri, ExpectedCID: expectedCID})
		if err != nil {
			envelope.WriteError(writer, http.StatusInternalServerError, "internal_error", "could not prepare post delete", runID, nil)
			return
		}
		sessionID, _ := middleware.GetOAuthSessionID(request.Context())
		result, err := commands.Delete(request.Context(), pdscommands.AddressedDeleteCommandRequest{
			Owner: owner, OwnerGeneration: ownerGeneration, SessionID: sessionID,
			OperationKind: "post.delete", OperationKey: operationKey,
			URI: uri, ExpectedCID: expectedCID, Intent: intent,
			AcceptedAbsent: func() pdscommands.TerminalResult {
				return pdscommands.TerminalResult{State: pdscommands.CommandAccepted, HTTPStatus: http.StatusNoContent}
			},
			Rejected: func(err error) pdscommands.TerminalResult { return RejectedCommandResult(runID, err) },
		})
		if err != nil {
			if logger != nil {
				logger.Warn("post delete command failed", slog.Any("error", err))
			}
			WriteCommandError(writer, runID, err)
			return
		}
		WriteCommandResponse(writer, CommandResultFromStored(result))
	})
}
