package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"social.craftsky/appview/internal/observability"
	"strings"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/pdseffects"
)

const (
	businessEventNSID         = syntax.NSID("social.craftsky.business.event")
	maxBusinessEventBodyBytes = 1 << 20
)

type businessEventMutationResponse struct {
	DID  syntax.DID       `json:"did"`
	Rkey syntax.RecordKey `json:"rkey"`
	URI  syntax.ATURI     `json:"uri"`
	CID  syntax.CID       `json:"cid"`
}

type BusinessEventPage struct {
	Items  []business.EventView `json:"items"`
	Cursor string               `json:"cursor,omitempty"`
}

type businessEventResponsePage struct {
	Items  []BusinessEventResponse `json:"items"`
	Cursor string                  `json:"cursor,omitempty"`
}

type BusinessEventResponse struct {
	DID                      syntax.DID           `json:"did"`
	Rkey                     syntax.RecordKey     `json:"rkey"`
	URI                      syntax.ATURI         `json:"uri"`
	CID                      syntax.CID           `json:"cid"`
	Name                     string               `json:"name"`
	StartsAt                 string               `json:"startsAt"`
	EndsAt                   string               `json:"endsAt"`
	Roles                    []business.OpenValue `json:"roles"`
	Mode                     *business.OpenValue  `json:"mode,omitempty"`
	Status                   business.OpenValue   `json:"status"`
	TimeZone                 string               `json:"timeZone,omitempty"`
	IsAllDay                 bool                 `json:"isAllDay"`
	Summary                  string               `json:"summary,omitempty"`
	VenueName                string               `json:"venueName,omitempty"`
	EventURI                 string               `json:"eventUri,omitempty"`
	RegistrationURI          string               `json:"registrationUri,omitempty"`
	Image                    *BusinessImageView   `json:"image,omitempty"`
	CreatedAt                string               `json:"createdAt"`
	Past                     bool                 `json:"past"`
	PublicSuppressionReasons []string             `json:"publicSuppressionReasons"`
	UpcomingExclusionReasons []string             `json:"upcomingExclusionReasons"`
}

func BuildBusinessEventResponse(event business.EventView) BusinessEventResponse {
	return BusinessEventResponse{
		DID: event.DID, Rkey: event.Rkey, URI: event.URI, CID: event.CID,
		Name: event.Name, StartsAt: event.StartsAt, EndsAt: event.EndsAt,
		Roles: event.Roles, Mode: event.Mode, Status: event.Status,
		TimeZone: event.TimeZone, IsAllDay: event.IsAllDay, Summary: event.Summary,
		VenueName: event.VenueName, EventURI: event.EventURI, RegistrationURI: event.RegistrationURI,
		Image: buildBusinessImageView(event.DID, event.Image), CreatedAt: event.CreatedAt, Past: event.Past,
		PublicSuppressionReasons: event.PublicSuppressionReasons,
		UpcomingExclusionReasons: event.UpcomingExclusionReasons,
	}
}

func buildBusinessEventResponses(events []business.EventView) []BusinessEventResponse {
	responses := make([]BusinessEventResponse, len(events))
	for index, event := range events {
		responses[index] = BuildBusinessEventResponse(event)
	}
	return responses
}

type UpcomingBusinessEventLister interface {
	ListUpcomingEvents(context.Context, business.UpcomingEventListInput) ([]business.EventView, error)
}

type OwnerBusinessEventLister interface {
	ListOwnerEvents(context.Context, business.OwnerEventListInput) ([]business.EventView, error)
}

type BusinessEventReader interface {
	ReadEvent(context.Context, business.EventReadInput) (business.EventView, error)
}

func GetBusinessEventHandler(store BusinessEventReader, now func() time.Time) http.Handler {
	if now == nil {
		now = time.Now
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		caller, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", runID, nil)
			return
		}
		owner, err := syntax.ParseDID(r.PathValue("did"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_identifier", "not a valid DID", runID, nil)
			return
		}
		rkey, err := syntax.ParseRecordKey(r.PathValue("rkey"))
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_rkey", "not a valid record key", runID, nil)
			return
		}
		event, err := store.ReadEvent(r.Context(), business.EventReadInput{
			CallerDID: caller,
			OwnerDID:  owner,
			Rkey:      rkey,
			AsOf:      now().UTC(),
		})
		if errors.Is(err, business.ErrEventNotFound) {
			envelope.WriteError(w, http.StatusNotFound, "event_not_found", "event not found", runID, nil)
			return
		}
		if err != nil {
			observability.ReportRequestFailure(r.Context(), err, "api.GetBusinessEventHandler", "handler")
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "event read failed", runID, nil)
			return
		}
		envelope.WriteJSON(w, http.StatusOK, BuildBusinessEventResponse(event))
	})
}

func GetOwnerBusinessEventsHandler(store OwnerBusinessEventLister, cursors *EventCursorCodec, now func() time.Time) http.Handler {
	if now == nil {
		now = time.Now
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", runID, nil)
			return
		}
		query := r.URL.Query()
		limit, err := NormalizeEventLimit(query.Get("limit"), 20)
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_limit", "limit must be a positive integer", runID, nil)
			return
		}
		asOf := now().UTC()
		filter := business.OwnerEventFilter("")
		cursorKind := EventCursorManagement
		if values, present := query["filter"]; present {
			if len(values) != 1 {
				envelope.WriteError(w, http.StatusBadRequest, "invalid_filter", "filter must be upcoming or history", runID, nil)
				return
			}
			switch values[0] {
			case string(business.OwnerEventUpcoming):
				filter = business.OwnerEventUpcoming
				cursorKind = EventCursorOwnerUpcoming
			case string(business.OwnerEventHistory):
				filter = business.OwnerEventHistory
				cursorKind = EventCursorOwnerHistory
			default:
				envelope.WriteError(w, http.StatusBadRequest, "invalid_filter", "filter must be upcoming or history", runID, nil)
				return
			}
		}
		var seek *business.OwnerEventSeek
		if encoded := query.Get("cursor"); encoded != "" {
			if cursors == nil {
				envelope.WriteError(w, http.StatusBadRequest, "invalid_cursor", "cursor could not be decoded", runID, nil)
				return
			}
			cursor, err := cursors.Decode(encoded, cursorKind, owner)
			if err != nil {
				envelope.WriteError(w, http.StatusBadRequest, "invalid_cursor", "cursor could not be decoded", runID, nil)
				return
			}
			seek = &business.OwnerEventSeek{StartsAt: cursor.StartsAt, URI: cursor.URI}
			if filter != "" {
				asOf = cursor.AsOf
			}
		}
		items, err := store.ListOwnerEvents(r.Context(), business.OwnerEventListInput{
			OwnerDID: owner, Filter: filter, AsOf: asOf, Limit: limit, Seek: seek,
		})
		if err != nil {
			observability.ReportRequestFailure(r.Context(), err, "api.GetOwnerBusinessEventsHandler", "handler")
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "event list failed", runID, nil)
			return
		}
		page := businessEventResponsePage{Items: buildBusinessEventResponses(items)}
		if len(items) > limit {
			page.Items = page.Items[:limit]
			last := page.Items[len(page.Items)-1]
			startsAt, err := time.Parse(time.RFC3339Nano, last.StartsAt)
			if err != nil {
				observability.ReportRequestFailure(r.Context(), err, "api.GetOwnerBusinessEventsHandler", "handler")
				envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "event cursor could not be created", runID, nil)
				return
			}
			if cursors == nil {
				envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "event cursor could not be created", runID, nil)
				return
			}
			page.Cursor, err = cursors.Encode(EventCursor{
				Kind: cursorKind, AsOf: asOf, StartsAt: startsAt, URI: last.URI,
			}, owner)
			if err != nil {
				observability.ReportRequestFailure(r.Context(), err, "api.GetOwnerBusinessEventsHandler", "handler")
				envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "event cursor could not be created", runID, nil)
				return
			}
		}
		envelope.WriteJSON(w, http.StatusOK, page)
	})
}

func GetProfileBusinessEventsHandler(
	store UpcomingBusinessEventLister,
	resolver HandleResolver,
	cursors *EventCursorCodec,
	now func() time.Time,
) http.Handler {
	if now == nil {
		now = time.Now
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		caller, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", runID, nil)
			return
		}
		owner, err := resolveToDID(r.Context(), strings.TrimPrefix(r.PathValue("handleOrDid"), "@"), resolver)
		if err != nil {
			if errors.Is(err, errInvalidIdentifier) {
				envelope.WriteError(w, http.StatusBadRequest, "invalid_identifier", "not a valid handle or DID", runID, nil)
			} else {
				observability.ReportRequestFailure(r.Context(), err, "api.GetProfileBusinessEventsHandler", "handler")
				envelope.WriteError(w, http.StatusBadGateway, "identity_unavailable", "could not resolve identity", runID, nil)
			}
			return
		}
		limit, err := NormalizeEventLimit(r.URL.Query().Get("limit"), 10)
		if err != nil {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_limit", "limit must be a positive integer", runID, nil)
			return
		}

		asOf := now().UTC()
		var seek *business.UpcomingEventSeek
		if encoded := r.URL.Query().Get("cursor"); encoded != "" {
			if cursors == nil {
				envelope.WriteError(w, http.StatusBadRequest, "invalid_cursor", "cursor could not be decoded", runID, nil)
				return
			}
			cursor, err := cursors.Decode(encoded, EventCursorUpcoming, owner)
			if err != nil {
				envelope.WriteError(w, http.StatusBadRequest, "invalid_cursor", "cursor could not be decoded", runID, nil)
				return
			}
			asOf = cursor.AsOf
			seek = &business.UpcomingEventSeek{StartsAt: cursor.StartsAt, URI: cursor.URI}
		}

		items, err := store.ListUpcomingEvents(r.Context(), business.UpcomingEventListInput{
			CallerDID: caller,
			OwnerDID:  owner,
			AsOf:      asOf,
			Limit:     limit,
			Seek:      seek,
		})
		if err != nil {
			observability.ReportRequestFailure(r.Context(), err, "api.GetProfileBusinessEventsHandler", "handler")
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "event list failed", runID, nil)
			return
		}
		page := businessEventResponsePage{Items: buildBusinessEventResponses(items)}
		if len(items) > limit {
			page.Items = page.Items[:limit]
			last := page.Items[len(page.Items)-1]
			startsAt, err := time.Parse(time.RFC3339Nano, last.StartsAt)
			if err != nil {
				observability.ReportRequestFailure(r.Context(), err, "api.GetProfileBusinessEventsHandler", "handler")
				envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "event cursor could not be created", runID, nil)
				return
			}
			if cursors == nil {
				envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "event cursor could not be created", runID, nil)
				return
			}
			page.Cursor, err = cursors.Encode(EventCursor{
				Kind: EventCursorUpcoming, AsOf: asOf, StartsAt: startsAt, URI: last.URI,
			}, owner)
			if err != nil {
				observability.ReportRequestFailure(r.Context(), err, "api.GetProfileBusinessEventsHandler", "handler")
				envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "event cursor could not be created", runID, nil)
				return
			}
		}
		envelope.WriteJSON(w, http.StatusOK, page)
	})
}

type businessEventRequest struct {
	Name            string                  `json:"name"`
	StartsAt        string                  `json:"startsAt"`
	EndsAt          string                  `json:"endsAt"`
	Roles           []string                `json:"roles"`
	Mode            string                  `json:"mode"`
	Status          string                  `json:"status"`
	TimeZone        string                  `json:"timeZone"`
	IsAllDay        bool                    `json:"isAllDay,omitempty"`
	Summary         *string                 `json:"summary,omitempty"`
	VenueName       *string                 `json:"venueName,omitempty"`
	EventURI        *string                 `json:"eventUri,omitempty"`
	RegistrationURI *string                 `json:"registrationUri,omitempty"`
	Image           *businessEventImageBody `json:"image,omitempty"`
	CreatedAt       json.RawMessage         `json:"createdAt,omitempty"`
}

type businessEventImageBody struct {
	Image       map[string]any        `json:"image"`
	Alt         string                `json:"alt,omitempty"`
	AspectRatio *PostImageAspectRatio `json:"aspectRatio,omitempty"`
}

type BusinessEventHandlerOptions struct {
	Commands          AppendCommandExecutor
	AddressedCommands AddressedPutCommandExecutor
	DeleteCommands    AddressedDeleteCommandExecutor
}

type AddressedPutCommandExecutor interface {
	Put(context.Context, pdscommands.AddressedPutCommandRequest) (pdscommands.CommandResult, error)
}

func PostBusinessEventHandler(
	newEffects pdseffects.ExecutorFactory,
	now func() time.Time,
	options ...BusinessEventHandlerOptions,
) http.Handler {
	if now == nil {
		now = time.Now
	}
	var commands AppendCommandExecutor
	if len(options) > 0 {
		commands = options[0].Commands
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		operationKey := uuid.Nil
		if commands != nil {
			var ok bool
			operationKey, ok = requireCommandOperationKey(w, r, runID)
			if !ok {
				return
			}
		}
		currentTime := now()
		request, record, fieldErr := decodeBusinessEventRequest(r.Body, currentTime, false)
		if fieldErr != nil {
			writeBusinessEventFieldError(w, runID, fieldErr)
			return
		}
		authored, err := business.PrepareEventCreate(business.EventAuthoringInput{
			Event: request.eventWrite(), CreatedAtPresent: request.CreatedAt != nil,
		}, currentTime)
		if err != nil {
			writeBusinessEventFieldError(w, runID, createdAtFieldError())
			return
		}
		record["createdAt"] = authored.CreatedAt
		if commands != nil {
			owner, ok := middleware.GetDID(r.Context())
			if !ok {
				envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", runID, nil)
				return
			}
			generation, ok := requirePDSEffectGeneration(w, r, runID)
			if !ok {
				return
			}
			sessionID, _ := middleware.GetOAuthSessionID(r.Context())
			intent, err := json.Marshal(request)
			if err != nil {
				observability.ReportRequestFailure(r.Context(), err, "api.PostBusinessEventHandler", "handler")
				envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "could not prepare business event", runID, nil)
				return
			}
			result, err := commands.Execute(r.Context(), pdscommands.AppendCommandRequest{
				Owner: owner, OwnerGeneration: generation, SessionID: sessionID,
				OperationKind: "business_event.create", OperationKey: operationKey,
				Collection: businessEventNSID, Intent: intent, Blobs: businessEventBlobReferences(request),
				BuildRecord: func(createdAt time.Time) (json.RawMessage, error) {
					frozen := make(map[string]any, len(record))
					for key, value := range record {
						frozen[key] = value
					}
					frozenAuthored, err := business.PrepareEventCreate(business.EventAuthoringInput{
						Event: request.eventWrite(), CreatedAtPresent: request.CreatedAt != nil,
					}, createdAt)
					if err != nil {
						return nil, err
					}
					frozen["createdAt"] = frozenAuthored.CreatedAt
					return json.Marshal(frozen)
				},
				Accepted: func(authoritative pdscommands.AuthoritativeRecord) (pdscommands.TerminalResult, error) {
					return acceptedBusinessEventCreateResult(owner, authoritative)
				},
				Rejected: func(err error) pdscommands.TerminalResult {
					return RejectedCommandResult(runID, err)
				},
			})
			if err != nil {
				WriteCommandError(w, runID, err, r.Context())
				return
			}
			WriteCommandResponse(w, CommandResultFromStored(result))
			return
		}

		owner, generation, executor, expectedOwners, ok := businessEventExecutor(w, r, runID, newEffects)
		if !ok {
			return
		}
		rkey, err := newImmediateRecordKey()
		if err != nil {
			observability.ReportRequestFailure(r.Context(), err, "api.PostBusinessEventHandler", "handler")
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "could not allocate event record key", runID, nil)
			return
		}
		operationID, mutationKey := immediateEffectIdentity(runID, "business_event.create")
		result, err := executor.PutRecord(r.Context(), pdseffects.PutRecordRequest{
			OperationID: operationID, MutationKey: mutationKey,
			Owner: owner, OwnerGeneration: generation, ExpectedOwners: expectedOwners,
			Collection: businessEventNSID, Rkey: rkey, Record: record,
		})
		if isPDSRecordConflict(err) {
			WritePDSRecordConflict(w, runID)
			return
		}
		if err != nil {
			writePDSError(w, http.StatusBadGateway, "pds_write_failed", "could not create business event", runID, err)
			return
		}
		envelope.WriteJSON(w, http.StatusCreated, businessEventMutationResponse{
			DID: owner, Rkey: rkey, URI: result.URI, CID: result.CID,
		})
	})
}

func acceptedBusinessEventCreateResult(
	owner syntax.DID,
	record pdscommands.AuthoritativeRecord,
) (pdscommands.TerminalResult, error) {
	return acceptedBusinessEventMutationResult(owner, http.StatusCreated, record)
}

func acceptedBusinessEventMutationResult(
	owner syntax.DID,
	status int,
	record pdscommands.AuthoritativeRecord,
) (pdscommands.TerminalResult, error) {
	uri, err := syntax.ParseATURI(record.URI.String())
	if err != nil || uri.Authority().DID() != owner || uri.Collection() != businessEventNSID || record.CID == "" {
		return pdscommands.TerminalResult{}, pdscommands.ErrMalformedCommand
	}
	rkey := uri.RecordKey()
	if _, err := syntax.ParseTID(rkey.String()); err != nil {
		return pdscommands.TerminalResult{}, pdscommands.ErrMalformedCommand
	}
	body, err := json.Marshal(businessEventMutationResponse{
		DID: owner, Rkey: rkey, URI: record.URI, CID: record.CID,
	})
	if err != nil {
		return pdscommands.TerminalResult{}, err
	}
	return pdscommands.TerminalResult{
		State: pdscommands.CommandAccepted, HTTPStatus: status, ResponseBody: body,
	}, nil
}

func businessEventBlobReferences(request businessEventRequest) []pdscommands.BlobReference {
	if request.Image == nil {
		return nil
	}
	ref, _ := request.Image.Image["ref"].(map[string]any)
	cid, _ := ref["$link"].(string)
	mimeType, _ := request.Image.Image["mimeType"].(string)
	size, ok := positiveIntegerAsInt64(request.Image.Image["size"])
	if cid == "" || mimeType == "" || !ok {
		return nil
	}
	return []pdscommands.BlobReference{{CID: syntax.CID(cid), MIMEType: mimeType, Size: size}}
}

func PutBusinessEventHandler(
	newEffects pdseffects.ExecutorFactory,
	now func() time.Time,
	options ...BusinessEventHandlerOptions,
) http.Handler {
	if now == nil {
		now = time.Now
	}
	var commands AddressedPutCommandExecutor
	if len(options) > 0 {
		commands = options[0].AddressedCommands
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		operationKey := uuid.Nil
		if commands != nil {
			var ok bool
			operationKey, ok = requireCommandOperationKey(w, r, runID)
			if !ok {
				return
			}
		}
		owner, rkey, ok := businessEventPathOwner(w, r, runID)
		if !ok {
			return
		}
		expectedCID, err := ParseBusinessIfMatch(r)
		if err != nil || expectedCID == "*" {
			WritePDSRecordConflict(w, runID)
			return
		}
		request, record, fieldErr := decodeBusinessEventRequest(r.Body, now(), true)
		if fieldErr != nil {
			writeBusinessEventFieldError(w, runID, fieldErr)
			return
		}
		if commands != nil {
			generation, ok := requirePDSEffectGeneration(w, r, runID)
			if !ok {
				return
			}
			sessionID, _ := middleware.GetOAuthSessionID(r.Context())
			intent, err := json.Marshal(request)
			if err != nil {
				observability.ReportRequestFailure(r.Context(), err, "api.PutBusinessEventHandler", "handler")
				envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "could not prepare business event", runID, nil)
				return
			}
			uri := syntax.ATURI("at://" + owner.String() + "/" + businessEventNSID.String() + "/" + rkey.String())
			result, err := commands.Put(r.Context(), pdscommands.AddressedPutCommandRequest{
				Owner: owner, OwnerGeneration: generation, SessionID: sessionID,
				OperationKind: "business_event.update", OperationKey: operationKey,
				URI: uri, ExpectedCID: expectedCID, Intent: intent, Blobs: businessEventBlobReferences(request),
				BuildRecord: func(current pdscommands.AuthoritativeRecord) (json.RawMessage, error) {
					var stored map[string]any
					if json.Unmarshal(current.Record, &stored) != nil {
						return nil, pdscommands.ErrMalformedCommand
					}
					storedCreatedAt, ok := stored["createdAt"].(string)
					if !ok || storedCreatedAt == "" {
						return nil, pdscommands.ErrMalformedCommand
					}
					authored, err := business.PrepareEventUpdate(business.EventAuthoringInput{
						Event: request.eventWrite(), CreatedAtPresent: request.CreatedAt != nil,
					}, storedCreatedAt)
					if err != nil {
						return nil, err
					}
					updated := make(map[string]any, len(record)+1)
					for key, value := range record {
						updated[key] = value
					}
					updated["createdAt"] = authored.CreatedAt
					return json.Marshal(updated)
				},
				Accepted: func(authoritative pdscommands.AuthoritativeRecord) (pdscommands.TerminalResult, error) {
					return acceptedBusinessEventMutationResult(owner, http.StatusOK, authoritative)
				},
				Rejected: func(err error) pdscommands.TerminalResult {
					return RejectedCommandResult(runID, err)
				},
			})
			if err != nil {
				WriteCommandError(w, runID, err, r.Context())
				return
			}
			WriteCommandResponse(w, CommandResultFromStored(result))
			return
		}
		generation, executor, expectedOwners, ok := businessEventExecutorForOwner(w, r, runID, newEffects, owner)
		if !ok {
			return
		}
		var current map[string]any
		currentCID, err := executor.ReadRecord(r.Context(), businessEventReadRequest(owner, generation, expectedOwners, rkey), &current)
		if err != nil {
			if errors.Is(err, auth.ErrRecordNotFound) {
				WritePDSRecordConflict(w, runID)
				return
			}
			writePDSError(w, http.StatusBadGateway, "pds_read_failed", "could not read business event", runID, err)
			return
		}
		if RequireCurrentCID(expectedCID, currentCID) != nil {
			WritePDSRecordConflict(w, runID)
			return
		}
		storedCreatedAt, ok := current["createdAt"].(string)
		if !ok || storedCreatedAt == "" {
			envelope.WriteError(w, http.StatusBadGateway, "pds_read_failed", "could not read business event", runID, nil)
			return
		}
		authored, err := business.PrepareEventUpdate(business.EventAuthoringInput{
			Event: request.eventWrite(), CreatedAtPresent: request.CreatedAt != nil,
		}, storedCreatedAt)
		if err != nil {
			writeBusinessEventFieldError(w, runID, createdAtFieldError())
			return
		}
		record["createdAt"] = authored.CreatedAt
		operationID, mutationKey := immediateEffectIdentity(runID, "business_event.update")
		result, err := executor.PutRecord(r.Context(), pdseffects.PutRecordRequest{
			OperationID: operationID, MutationKey: mutationKey,
			Owner: owner, OwnerGeneration: generation, ExpectedOwners: expectedOwners,
			Collection: businessEventNSID, Rkey: rkey, Record: record, ExpectedCID: expectedCID,
		})
		if isPDSRecordConflict(err) {
			WritePDSRecordConflict(w, runID)
			return
		}
		if err != nil {
			writePDSError(w, http.StatusBadGateway, "pds_write_failed", "could not update business event", runID, err)
			return
		}
		envelope.WriteJSON(w, http.StatusOK, businessEventMutationResponse{
			DID: owner, Rkey: rkey, URI: result.URI, CID: result.CID,
		})
	})
}

func DeleteBusinessEventHandler(
	newEffects pdseffects.ExecutorFactory,
	options ...BusinessEventHandlerOptions,
) http.Handler {
	var commands AddressedDeleteCommandExecutor
	if len(options) > 0 {
		commands = options[0].DeleteCommands
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		operationKey := uuid.Nil
		if commands != nil {
			var ok bool
			operationKey, ok = requireCommandOperationKey(w, r, runID)
			if !ok {
				return
			}
		}
		owner, rkey, ok := businessEventPathOwner(w, r, runID)
		if !ok {
			return
		}
		expectedCID, err := ParseBusinessIfMatch(r)
		if err != nil || expectedCID == "*" {
			WritePDSRecordConflict(w, runID)
			return
		}
		if commands != nil {
			generation, ok := requirePDSEffectGeneration(w, r, runID)
			if !ok {
				return
			}
			uri := syntax.ATURI("at://" + owner.String() + "/" + businessEventNSID.String() + "/" + rkey.String())
			intent, err := json.Marshal(struct {
				URI         syntax.ATURI `json:"uri"`
				ExpectedCID syntax.CID   `json:"expectedCid"`
			}{URI: uri, ExpectedCID: expectedCID})
			if err != nil {
				observability.ReportRequestFailure(r.Context(), err, "api.DeleteBusinessEventHandler", "handler")
				envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "could not prepare business event delete", runID, nil)
				return
			}
			sessionID, _ := middleware.GetOAuthSessionID(r.Context())
			result, err := commands.Delete(r.Context(), pdscommands.AddressedDeleteCommandRequest{
				Owner: owner, OwnerGeneration: generation, SessionID: sessionID,
				OperationKind: "business_event.delete", OperationKey: operationKey,
				URI: uri, ExpectedCID: expectedCID, Intent: intent,
				AcceptedAbsent: func() pdscommands.TerminalResult {
					return pdscommands.TerminalResult{State: pdscommands.CommandAccepted, HTTPStatus: http.StatusNoContent}
				},
				Rejected: func(err error) pdscommands.TerminalResult {
					return RejectedCommandResult(runID, err)
				},
			})
			if err != nil {
				WriteCommandError(w, runID, err, r.Context())
				return
			}
			WriteCommandResponse(w, CommandResultFromStored(result))
			return
		}
		generation, executor, expectedOwners, ok := businessEventExecutorForOwner(w, r, runID, newEffects, owner)
		if !ok {
			return
		}
		var current map[string]any
		currentCID, err := executor.ReadRecord(r.Context(), businessEventReadRequest(owner, generation, expectedOwners, rkey), &current)
		if err != nil {
			if errors.Is(err, auth.ErrRecordNotFound) {
				WritePDSRecordConflict(w, runID)
				return
			}
			writePDSError(w, http.StatusBadGateway, "pds_read_failed", "could not read business event", runID, err)
			return
		}
		if RequireCurrentCID(expectedCID, currentCID) != nil {
			WritePDSRecordConflict(w, runID)
			return
		}
		operationID, mutationKey := immediateEffectIdentity(runID, "business_event.delete")
		result, err := executor.DeleteRecord(r.Context(), pdseffects.DeleteRecordRequest{
			OperationID: operationID, MutationKey: mutationKey,
			Owner: owner, OwnerGeneration: generation, ExpectedOwners: expectedOwners,
			Collection: businessEventNSID, Rkey: rkey, ExpectedCID: expectedCID,
		})
		if isPDSRecordConflict(err) {
			WritePDSRecordConflict(w, runID)
			return
		}
		if err != nil {
			writePDSError(w, http.StatusBadGateway, "pds_write_failed", "could not delete business event", runID, err)
			return
		}
		envelope.WriteJSON(w, http.StatusOK, businessEventMutationResponse{
			DID: owner, Rkey: rkey, URI: result.URI, CID: expectedCID,
		})
	})
}

func decodeBusinessEventRequest(body io.Reader, now time.Time, existing bool) (businessEventRequest, map[string]any, *FieldError) {
	raw, err := io.ReadAll(io.LimitReader(body, maxBusinessEventBodyBytes+1))
	if err != nil || len(raw) > maxBusinessEventBodyBytes {
		return businessEventRequest{}, nil, &FieldError{Code: "malformed_body", Fields: map[string]string{"_": "invalid request body"}}
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return businessEventRequest{}, nil, &FieldError{Code: "malformed_body", Fields: map[string]string{"_": "invalid JSON object"}}
	}
	if _, present := fields["createdAt"]; present {
		return businessEventRequest{}, nil, createdAtFieldError()
	}
	for field, value := range fields {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return businessEventRequest{}, nil, &FieldError{Code: "validation_failed", Fields: map[string]string{field: "must not be null"}}
		}
	}
	var request businessEventRequest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return businessEventRequest{}, nil, &FieldError{Code: "unexpected_field", Fields: map[string]string{"_": err.Error()}}
	}
	if fieldErr := validateBusinessEventRequest(&request, now, existing); fieldErr != nil {
		return businessEventRequest{}, nil, fieldErr
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		return businessEventRequest{}, nil, &FieldError{Code: "malformed_body", Fields: map[string]string{"_": "invalid request body"}}
	}
	var record map[string]any
	if err := json.Unmarshal(encoded, &record); err != nil {
		return businessEventRequest{}, nil, &FieldError{Code: "malformed_body", Fields: map[string]string{"_": "invalid request body"}}
	}
	delete(record, "createdAt")
	record["$type"] = businessEventNSID.String()
	return request, record, nil
}

func validateBusinessEventRequest(request *businessEventRequest, now time.Time, existing bool) *FieldError {
	fields := make(map[string]string)
	event := request.eventWrite()
	if event.Name == "" || business.ValidateText(business.TextFieldEventName, event.Name) != nil {
		fields["name"] = "is invalid"
	}
	if request.Summary != nil && business.ValidateText(business.TextFieldEventSummary, *request.Summary) != nil {
		fields["summary"] = "is invalid"
	}
	if request.VenueName != nil && business.ValidateText(business.TextFieldVenueName, *request.VenueName) != nil {
		fields["venueName"] = "is invalid"
	}
	if business.ValidateEventCatalogs(event) != nil {
		fields["roles"] = "roles, mode, status, and timeZone must use supported values"
	}
	if business.ValidateEventTemporalPolicy(event, now, existing) != nil || business.ValidateAllDayEvent(event) != nil {
		fields["startsAt"] = "startsAt and endsAt must form a valid event time range"
	}
	if request.Image != nil {
		if len(request.Image.Image) == 0 {
			fields["image.image"] = "is required"
		} else {
			validateProfileImageUpdate(fields, "image.image", ProfileImageUpdate{Present: true, Blob: request.Image.Image}, DefaultMediaLimits())
		}
		if business.ValidateText(business.TextFieldImageAlt, request.Image.Alt) != nil {
			fields["image.alt"] = "is invalid"
		}
		if ratio := request.Image.AspectRatio; ratio != nil && (ratio.Width <= 0 || ratio.Height <= 0) {
			fields["image.aspectRatio"] = "must have positive dimensions"
		}
	}
	if business.ValidateEventMedia(event) != nil {
		fields["eventUri"] = "event links or image are invalid"
	}
	if len(fields) != 0 {
		return &FieldError{Code: "validation_failed", Fields: fields}
	}
	return nil
}

func (request businessEventRequest) eventWrite() business.EventWrite {
	event := business.EventWrite{
		Name: request.Name, StartsAt: request.StartsAt, EndsAt: request.EndsAt,
		Roles: request.Roles, Mode: request.Mode, Status: request.Status,
		TimeZone: request.TimeZone, IsAllDay: request.IsAllDay,
	}
	if request.Summary != nil {
		event.Summary = *request.Summary
	}
	if request.VenueName != nil {
		event.VenueName = *request.VenueName
	}
	if request.EventURI != nil {
		event.EventURI = *request.EventURI
	}
	if request.RegistrationURI != nil {
		event.RegistrationURI = *request.RegistrationURI
	}
	if request.Image != nil {
		mime, _ := request.Image.Image["mimeType"].(string)
		size, _ := positiveIntegerAsInt64(request.Image.Image["size"])
		image := &business.Image{MIMEType: mime, Size: size, Alt: request.Image.Alt}
		if request.Image.AspectRatio != nil {
			image.AspectRatio = &business.AspectRatio{Width: int64(request.Image.AspectRatio.Width), Height: int64(request.Image.AspectRatio.Height)}
		}
		event.Image = image
	}
	return event
}

func businessEventPathOwner(w http.ResponseWriter, r *http.Request, runID string) (syntax.DID, syntax.RecordKey, bool) {
	caller, ok := middleware.GetDID(r.Context())
	if !ok {
		envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", runID, nil)
		return "", "", false
	}
	owner, err := syntax.ParseDID(r.PathValue("did"))
	if err != nil {
		envelope.WriteError(w, http.StatusBadRequest, "invalid_identifier", "did path segment is not a valid DID", runID, nil)
		return "", "", false
	}
	if owner != caller {
		envelope.WriteError(w, http.StatusForbidden, "forbidden", "cannot mutate another user's business event", runID, nil)
		return "", "", false
	}
	tid, err := syntax.ParseTID(r.PathValue("rkey"))
	if err != nil {
		envelope.WriteError(w, http.StatusBadRequest, "invalid_identifier", "rkey path segment is not a valid TID", runID, nil)
		return "", "", false
	}
	return owner, syntax.RecordKey(tid.String()), true
}

func businessEventExecutor(
	w http.ResponseWriter,
	r *http.Request,
	runID string,
	newEffects pdseffects.ExecutorFactory,
) (syntax.DID, int64, pdseffects.EffectExecutor, []ownerlifecycle.ExpectedOwner, bool) {
	owner, ok := middleware.GetDID(r.Context())
	if !ok {
		envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", runID, nil)
		return "", 0, nil, nil, false
	}
	generation, executor, expectedOwners, ok := businessEventExecutorForOwner(w, r, runID, newEffects, owner)
	return owner, generation, executor, expectedOwners, ok
}

func businessEventExecutorForOwner(
	w http.ResponseWriter,
	r *http.Request,
	runID string,
	newEffects pdseffects.ExecutorFactory,
	owner syntax.DID,
) (int64, pdseffects.EffectExecutor, []ownerlifecycle.ExpectedOwner, bool) {
	generation, ok := requirePDSEffectGeneration(w, r, runID)
	if !ok {
		return 0, nil, nil, false
	}
	sessionID, _ := middleware.GetOAuthSessionID(r.Context())
	executor, err := newPDSEffectExecutor(r.Context(), newEffects, owner, sessionID)
	if err != nil {
		writePDSError(w, http.StatusBadGateway, "pds_unavailable", "could not contact PDS", runID, err)
		return 0, nil, nil, false
	}
	expectedOwners, err := executor.ResolveExpectedOwners(r.Context(), generation, nil)
	if err != nil {
		writePDSError(w, http.StatusBadGateway, "pds_write_failed", "could not prepare business event mutation", runID, err)
		return 0, nil, nil, false
	}
	return generation, executor, expectedOwners, true
}

func businessEventReadRequest(
	owner syntax.DID,
	generation int64,
	expectedOwners []ownerlifecycle.ExpectedOwner,
	rkey syntax.RecordKey,
) pdseffects.ReadRecordRequest {
	return pdseffects.ReadRecordRequest{
		Owner: owner, OwnerGeneration: generation, ExpectedOwners: expectedOwners,
		Collection: businessEventNSID, Rkey: rkey,
	}
}

func createdAtFieldError() *FieldError {
	return &FieldError{Code: "validation_failed", Fields: map[string]string{"createdAt": "must be omitted"}}
}

func writeBusinessEventFieldError(w http.ResponseWriter, runID string, fieldErr *FieldError) {
	status := http.StatusBadRequest
	if fieldErr.Code == "validation_failed" {
		status = http.StatusUnprocessableEntity
	}
	envelope.WriteError(w, status, fieldErr.Code, "invalid business event", runID, fieldErr.Fields)
}
