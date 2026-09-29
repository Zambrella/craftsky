package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/pdscommands"
)

func TestCommandCreateBusinessEventRequiresCanonicalOperationKeyBeforeDependencies(t *testing.T) {
	commands := &recordingAppendCommands{}
	handler := api.PostBusinessEventHandler(
		nil,
		func() time.Time { return businessEventNow },
		api.BusinessEventHandlerOptions{Commands: commands},
	)

	response := serveBusinessEventRequest(t, handler, http.MethodPost, "/v1/events", validBusinessEventBody(false), "", "", "")

	assertAPIError(t, response, http.StatusBadRequest, "invalid_idempotency_key")
	if commands.calls != 0 {
		t.Fatalf("append command calls = %d, want 0", commands.calls)
	}
}

func TestCommandCreateBusinessEventBuildsFrozenAppendAndReturnsAcceptedEvent(t *testing.T) {
	commands := &recordingAppendCommands{}
	commands.execute = func(request pdscommands.AppendCommandRequest) (pdscommands.CommandResult, error) {
		record, err := request.BuildRecord(businessEventNow)
		if err != nil {
			return pdscommands.CommandResult{}, err
		}
		commands.record = record
		terminal, err := request.Accepted(pdscommands.AuthoritativeRecord{
			URI:    "at://did:plc:owner/social.craftsky.business.event/3mzzzzzzzzzzz",
			CID:    businessEventCID1,
			Record: record,
		})
		return pdscommands.CommandResult{TerminalResult: terminal}, err
	}
	handler := api.PostBusinessEventHandler(
		nil,
		func() time.Time { return businessEventNow },
		api.BusinessEventHandlerOptions{Commands: commands},
	)
	requestKey := "018f4d5c-7a61-7d40-a1a2-888888888888"
	request := httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader(validBusinessEventBody(true)))
	request.Header.Set("Idempotency-Key", requestKey)
	ctx := middleware.WithDID(request.Context(), "did:plc:owner")
	ctx = middleware.WithOwnerGeneration(ctx, 7)
	ctx = middleware.WithOAuthSessionID(ctx, "oauth-owner-session")
	ctx = ctxkeys.WithRunID(ctx, "business-event-request")
	request = request.WithContext(ctx)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	if commands.calls != 1 || commands.request.Owner != "did:plc:owner" || commands.request.OwnerGeneration != 7 ||
		commands.request.SessionID != "oauth-owner-session" || commands.request.OperationKind != "business_event.create" ||
		commands.request.OperationKey != uuid.MustParse(requestKey) || commands.request.Collection != "social.craftsky.business.event" {
		t.Fatalf("append request = %+v calls=%d", commands.request, commands.calls)
	}
	if len(commands.request.Blobs) != 1 || commands.request.Blobs[0].MIMEType != "image/webp" || commands.request.Blobs[0].Size != 1024 {
		t.Fatalf("append blob references = %+v", commands.request.Blobs)
	}
	var record map[string]any
	if err := json.Unmarshal(commands.record, &record); err != nil {
		t.Fatal(err)
	}
	if record["$type"] != "social.craftsky.business.event" || record["createdAt"] != "2026-09-01T12:34:56Z" || record["name"] != "Fiber Fair" {
		t.Fatalf("frozen record = %#v", record)
	}
	created := decodeBusinessEventMutationResponse(t, response)
	if created.DID != "did:plc:owner" || created.Rkey != "3mzzzzzzzzzzz" ||
		created.URI != "at://did:plc:owner/social.craftsky.business.event/3mzzzzzzzzzzz" || created.CID != businessEventCID1 {
		t.Fatalf("accepted event = %+v", created)
	}
}

func TestCommandCreateBusinessEventReturnsAmbiguousContract(t *testing.T) {
	commands := &recordingAppendCommands{result: pdscommands.CommandResult{
		TerminalResult:    pdscommands.TerminalResult{State: pdscommands.CommandAmbiguous},
		RetryAfterSeconds: 9,
	}}
	handler := api.PostBusinessEventHandler(
		nil,
		func() time.Time { return businessEventNow },
		api.BusinessEventHandlerOptions{Commands: commands},
	)
	request := httptest.NewRequest(http.MethodPost, "/v1/events", strings.NewReader(validBusinessEventBody(false)))
	request.Header.Set("Idempotency-Key", "018f4d5c-7a61-7d40-a1a2-999999999999")
	ctx := middleware.WithDID(request.Context(), "did:plc:owner")
	ctx = middleware.WithOwnerGeneration(ctx, 7)
	ctx = middleware.WithOAuthSessionID(ctx, "oauth-owner-session")
	ctx = ctxkeys.WithRunID(ctx, "business-event-request")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request.WithContext(ctx))

	if response.Code != http.StatusAccepted || response.Header().Get("Retry-After") != "5" ||
		response.Header().Get("Location") != "" || strings.TrimSpace(response.Body.String()) != `{"status":"ambiguous"}` {
		t.Fatalf("ambiguous response = %d headers=%v body=%s", response.Code, response.Header(), response.Body.String())
	}
}

func TestCommandUpdateBusinessEventPreservesCreatedAtAndUsesAddressedPut(t *testing.T) {
	commands := &recordingAddressedPutCommands{}
	commands.put = func(request pdscommands.AddressedPutCommandRequest) (pdscommands.CommandResult, error) {
		record, err := request.BuildRecord(pdscommands.AuthoritativeRecord{
			URI: request.URI, CID: businessEventCID1, Record: json.RawMessage(`{
				"$type":"social.craftsky.business.event",
				"name":"Fiber Fair",
				"createdAt":"2026-08-15T09:30:00Z"
			}`),
		})
		if err != nil {
			return pdscommands.CommandResult{}, err
		}
		commands.record = record
		terminal, err := request.Accepted(pdscommands.AuthoritativeRecord{
			URI: request.URI, CID: businessEventCID2, Record: record,
		})
		return pdscommands.CommandResult{TerminalResult: terminal}, err
	}
	handler := api.PutBusinessEventHandler(
		nil,
		func() time.Time { return businessEventNow },
		api.BusinessEventHandlerOptions{AddressedCommands: commands},
	)
	request := httptest.NewRequest(http.MethodPut, "/v1/events/did:plc:owner/3mzzzzzzzzzzz", strings.NewReader(strings.Replace(validBusinessEventBody(false), "Fiber Fair", "Fiber Fair Updated", 1)))
	request.SetPathValue("did", "did:plc:owner")
	request.SetPathValue("rkey", "3mzzzzzzzzzzz")
	request.Header.Set("Idempotency-Key", "018f4d5c-7a61-7d40-a1a2-999999999993")
	request.Header.Set("If-Match", businessEventCID1)
	ctx := middleware.WithDID(request.Context(), "did:plc:owner")
	ctx = middleware.WithOwnerGeneration(ctx, 7)
	ctx = middleware.WithOAuthSessionID(ctx, "oauth-owner-session")
	ctx = ctxkeys.WithRunID(ctx, "business-event-request")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request.WithContext(ctx))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	if commands.putCalls != 1 || commands.putRequest.Owner != "did:plc:owner" || commands.putRequest.OwnerGeneration != 7 ||
		commands.putRequest.OperationKind != "business_event.update" || commands.putRequest.URI != "at://did:plc:owner/social.craftsky.business.event/3mzzzzzzzzzzz" ||
		commands.putRequest.ExpectedCID != businessEventCID1 {
		t.Fatalf("addressed put request = %+v calls=%d", commands.putRequest, commands.putCalls)
	}
	var record map[string]any
	if err := json.Unmarshal(commands.record, &record); err != nil {
		t.Fatal(err)
	}
	if record["createdAt"] != "2026-08-15T09:30:00Z" || record["name"] != "Fiber Fair Updated" {
		t.Fatalf("updated record = %#v", record)
	}
	updated := decodeBusinessEventMutationResponse(t, response)
	if updated.CID != businessEventCID2 || updated.Rkey != "3mzzzzzzzzzzz" {
		t.Fatalf("accepted event = %+v", updated)
	}
}

func TestCommandDeleteBusinessEventUsesExactAddressAndReturnsEmptyNoContent(t *testing.T) {
	commands := &recordingAddressedCommands{result: pdscommands.CommandResult{TerminalResult: pdscommands.TerminalResult{
		State: pdscommands.CommandAccepted, HTTPStatus: http.StatusNoContent,
	}}}
	handler := api.DeleteBusinessEventHandler(
		nil,
		api.BusinessEventHandlerOptions{DeleteCommands: commands},
	)
	request := httptest.NewRequest(http.MethodDelete, "/v1/events/did:plc:owner/3mzzzzzzzzzzz", nil)
	request.SetPathValue("did", "did:plc:owner")
	request.SetPathValue("rkey", "3mzzzzzzzzzzz")
	request.Header.Set("Idempotency-Key", "018f4d5c-7a61-7d40-a1a2-999999999994")
	request.Header.Set("If-Match", businessEventCID2)
	ctx := middleware.WithDID(request.Context(), "did:plc:owner")
	ctx = middleware.WithOwnerGeneration(ctx, 7)
	ctx = middleware.WithOAuthSessionID(ctx, "oauth-owner-session")
	ctx = ctxkeys.WithRunID(ctx, "business-event-request")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request.WithContext(ctx))

	if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
		t.Fatalf("status = %d body=%q", response.Code, response.Body.String())
	}
	if commands.calls != 1 || commands.request.Owner != "did:plc:owner" || commands.request.OwnerGeneration != 7 ||
		commands.request.OperationKind != "business_event.delete" || commands.request.URI != "at://did:plc:owner/social.craftsky.business.event/3mzzzzzzzzzzz" ||
		commands.request.ExpectedCID != businessEventCID2 {
		t.Fatalf("addressed delete request = %+v calls=%d", commands.request, commands.calls)
	}
}

type recordingAddressedPutCommands struct {
	putCalls   int
	putRequest pdscommands.AddressedPutCommandRequest
	record     json.RawMessage
	put        func(pdscommands.AddressedPutCommandRequest) (pdscommands.CommandResult, error)
}

func (commands *recordingAddressedPutCommands) Put(
	_ context.Context,
	request pdscommands.AddressedPutCommandRequest,
) (pdscommands.CommandResult, error) {
	commands.putCalls++
	commands.putRequest = request
	if commands.put != nil {
		return commands.put(request)
	}
	return pdscommands.CommandResult{}, nil
}

var _ api.AddressedPutCommandExecutor = (*recordingAddressedPutCommands)(nil)
