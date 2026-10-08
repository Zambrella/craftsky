package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/pdscommands"
)

type CommandHTTPResult struct {
	State             pdscommands.CommandState
	HTTPStatus        int
	Body              json.RawMessage
	Headers           map[string]string
	RetryAfterSeconds int
}

func WriteCommandError(writer http.ResponseWriter, requestID string, err error, contexts ...context.Context) {
	status, code, message := commandError(err)
	if len(contexts) > 0 && status >= 500 {
		observability.ReportRequestFailure(contexts[0], err, "pds.command", "dispatch")
	}
	envelope.WriteError(writer, status, code, message, requestID, nil)
}

func RejectedCommandResult(requestID string, err error) pdscommands.TerminalResult {
	status, code, message := commandError(err)
	body, _ := json.Marshal(envelope.Error{Error: code, Message: message, RequestID: requestID})
	return pdscommands.TerminalResult{
		State: pdscommands.CommandRejected, HTTPStatus: status, ResponseBody: body,
		ResponseHeaders: json.RawMessage(`{"Content-Type":"application/json"}`),
	}
}

func CommandResultFromStored(result pdscommands.CommandResult) CommandHTTPResult {
	headers := make(map[string]string)
	if len(result.ResponseHeaders) > 0 {
		_ = json.Unmarshal(result.ResponseHeaders, &headers)
	}
	return CommandHTTPResult{
		State: result.State, HTTPStatus: result.HTTPStatus, Body: result.ResponseBody,
		Headers: headers, RetryAfterSeconds: result.RetryAfterSeconds,
	}
}

func commandError(err error) (int, string, string) {
	status := http.StatusServiceUnavailable
	code := "pds_dispatch_unavailable"
	message := "The PDS command could not be dispatched safely."
	switch {
	case errors.Is(err, pdscommands.ErrIdempotencyConflict):
		status, code, message = http.StatusConflict, "idempotency_conflict", "The idempotency key was already used for a different command."
	case errors.Is(err, pdscommands.ErrRecordConflict):
		status, code, message = http.StatusConflict, "pds_record_conflict", "PDS record changed; refresh and try again"
	case errors.Is(err, pdscommands.ErrRepositoryConflict):
		status, code, message = http.StatusConflict, "pds_repository_conflict", "The PDS repository changed during the mutation."
	case errors.Is(err, pdscommands.ErrAtomicMutationTooLarge):
		status, code, message = http.StatusUnprocessableEntity, "pds_atomic_mutation_too_large", "The mutation exceeds the PDS atomic write limit."
	case errors.Is(err, pdscommands.ErrMalformedCommand):
		status, code, message = http.StatusBadRequest, "invalid_request", "The mutation request is invalid."
	case errors.Is(err, auth.ErrPDSSessionExpired):
		status, code, message = http.StatusUnauthorized, "pds_session_expired", "PDS session expired; sign in again."
	case errors.Is(err, pdscommands.ErrDispatchRejected):
		status, code, message = http.StatusBadGateway, "pds_write_rejected", "The PDS rejected the mutation."
	case errors.Is(err, pdscommands.ErrDispatchUnavailable):
	}
	return status, code, message
}

func WriteCommandResponse(writer http.ResponseWriter, result CommandHTTPResult) {
	if result.State == pdscommands.CommandAmbiguous {
		retryAfter := result.RetryAfterSeconds
		if retryAfter < 1 {
			retryAfter = 1
		} else if retryAfter > 5 {
			retryAfter = 5
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Retry-After", strconv.Itoa(retryAfter))
		writer.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(writer).Encode(map[string]string{"status": "ambiguous"})
		return
	}
	for key, value := range result.Headers {
		writer.Header().Set(key, value)
	}
	status := result.HTTPStatus
	if status == 0 {
		status = http.StatusOK
	}
	if len(result.Body) > 0 {
		writer.Header().Set("Content-Type", "application/json")
	}
	writer.WriteHeader(status)
	if status != http.StatusNoContent && len(result.Body) > 0 {
		_, _ = writer.Write(result.Body)
	}
}
