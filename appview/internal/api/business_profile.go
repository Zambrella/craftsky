package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"social.craftsky/appview/internal/observability"
	"strings"

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
	businessProfileNSID         = syntax.NSID("social.craftsky.business.profile")
	businessProfileRkey         = syntax.RecordKey("self")
	maxBusinessProfileBodyBytes = 1 << 20
)

type businessProfileResponse struct {
	CID syntax.CID `json:"cid"`
}

type businessProfileRequest struct {
	PreserveProducts             bool                         `json:"preserveProducts,omitempty"`
	PreserveUnknownCatalogValues bool                         `json:"preserveUnknownCatalogValues,omitempty"`
	BusinessTypes                []string                     `json:"businessTypes,omitempty"`
	Offerings                    []string                     `json:"offerings,omitempty"`
	Tagline                      *string                      `json:"tagline,omitempty"`
	HoursNote                    *string                      `json:"hoursNote,omitempty"`
	ServiceArea                  *string                      `json:"serviceArea,omitempty"`
	Location                     *business.Location           `json:"location,omitempty"`
	PrimaryAction                *business.Action             `json:"primaryAction,omitempty"`
	Products                     []businessProfileProductBody `json:"products,omitempty"`
}

type businessProfileProductBody struct {
	Title string                    `json:"title"`
	URI   string                    `json:"uri"`
	Image *businessProfileImageBody `json:"image,omitempty"`
	Price *business.Price           `json:"price,omitempty"`
}

type businessProfileImageBody struct {
	Image       map[string]any        `json:"image"`
	Alt         string                `json:"alt,omitempty"`
	AspectRatio *PostImageAspectRatio `json:"aspectRatio,omitempty"`
}

type BusinessProfileHandlerOptions struct {
	Commands       AddressedPutCommandExecutor
	DeleteCommands AddressedDeleteCommandExecutor
}

func PutBusinessProfileHandler(
	newEffects pdseffects.ExecutorFactory,
	options ...BusinessProfileHandlerOptions,
) http.Handler {
	var commands AddressedPutCommandExecutor
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
		expectedCID, err := ParseBusinessIfMatch(r)
		if err != nil {
			WritePDSRecordConflict(w, runID)
			return
		}
		replacement, fieldErr := decodeBusinessProfileRequest(r.Body)
		if fieldErr != nil {
			status := http.StatusBadRequest
			if fieldErr.Code == "validation_failed" {
				status = http.StatusUnprocessableEntity
			}
			envelope.WriteError(w, status, fieldErr.Code, "invalid business profile", runID, fieldErr.Fields)
			return
		}
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
			intent, err := json.Marshal(replacement)
			if err != nil {
				observability.ReportRequestFailure(r.Context(), err, "api.PutBusinessProfileHandler", "handler")
				envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "could not prepare business profile", runID, nil)
				return
			}
			uri := syntax.ATURI("at://" + owner.String() + "/" + businessProfileNSID.String() + "/" + businessProfileRkey.String())
			result, err := commands.Put(r.Context(), pdscommands.AddressedPutCommandRequest{
				Owner: owner, OwnerGeneration: generation, SessionID: sessionID,
				OperationKind: "business_profile.put", OperationKey: operationKey,
				URI: uri, ExpectedCID: expectedCID, Intent: intent, Blobs: businessProfileBlobReferences(replacement),
				BuildRecord: func(current pdscommands.AuthoritativeRecord) (json.RawMessage, error) {
					return buildBusinessProfileReplacement(current.Record, replacement)
				},
				Accepted: func(authoritative pdscommands.AuthoritativeRecord) (pdscommands.TerminalResult, error) {
					return acceptedBusinessProfilePutResult(owner, authoritative)
				},
				Rejected: func(err error) pdscommands.TerminalResult {
					return rejectedBusinessProfileResult(runID, err)
				},
			})
			if err != nil {
				WriteCommandError(w, runID, err, r.Context())
				return
			}
			WriteCommandResponse(w, CommandResultFromStored(result))
			return
		}
		owner, generation, executor, expectedOwners, ok := businessProfileExecutor(w, r, runID, newEffects)
		if !ok {
			return
		}
		read := businessProfileReadRequest(owner, generation, expectedOwners)
		var current json.RawMessage
		currentCID, readErr := executor.ReadRecord(r.Context(), read, &current)
		exists := readErr == nil
		if !exists && !errors.Is(readErr, auth.ErrRecordNotFound) {
			writePDSError(w, http.StatusBadGateway, "pds_read_failed", "could not read business profile", runID, readErr)
			return
		}
		if RequireBusinessRecordPrecondition(expectedCID, currentCID, exists) != nil {
			WritePDSRecordConflict(w, runID)
			return
		}

		record, buildErr := buildBusinessProfileReplacement(current, replacement)
		if buildErr != nil {
			var fieldErr *FieldError
			if errors.As(buildErr, &fieldErr) {
				envelope.WriteError(w, http.StatusUnprocessableEntity, fieldErr.Code, "invalid business profile", runID, fieldErr.Fields)
			} else {
				observability.ReportRequestFailure(r.Context(), buildErr, "api.PutBusinessProfileHandler", "handler")
				envelope.WriteError(w, http.StatusBadGateway, "pds_read_failed", "could not read business profile", runID, nil)
			}
			return
		}
		operationID, mutationKey := immediateEffectIdentity(runID, "business_profile.put")
		result, putErr := executor.PutRecord(r.Context(), pdseffects.PutRecordRequest{
			OperationID: operationID, MutationKey: mutationKey,
			Owner: owner, OwnerGeneration: generation, ExpectedOwners: expectedOwners,
			Collection: businessProfileNSID, Rkey: businessProfileRkey,
			Record: record, ExpectedCID: expectedCID,
		})
		if isPDSRecordConflict(putErr) {
			WritePDSRecordConflict(w, runID)
			return
		}
		if putErr != nil {
			writePDSError(w, http.StatusBadGateway, "pds_write_failed", "could not write business profile", runID, putErr)
			return
		}
		envelope.WriteJSON(w, http.StatusOK, businessProfileResponse{CID: result.CID})
	})
}

func acceptedBusinessProfilePutResult(
	owner syntax.DID,
	record pdscommands.AuthoritativeRecord,
) (pdscommands.TerminalResult, error) {
	uri, err := syntax.ParseATURI(record.URI.String())
	if err != nil || uri.Authority().DID() != owner || uri.Collection() != businessProfileNSID ||
		uri.RecordKey() != businessProfileRkey || record.CID == "" {
		return pdscommands.TerminalResult{}, pdscommands.ErrMalformedCommand
	}
	body, err := json.Marshal(businessProfileResponse{CID: record.CID})
	if err != nil {
		return pdscommands.TerminalResult{}, err
	}
	return pdscommands.TerminalResult{
		State: pdscommands.CommandAccepted, HTTPStatus: http.StatusOK, ResponseBody: body,
	}, nil
}

func businessProfileBlobReferences(record map[string]any) []pdscommands.BlobReference {
	var blobs []pdscommands.BlobReference
	var walk func(any)
	walk = func(value any) {
		switch typed := value.(type) {
		case []any:
			for _, item := range typed {
				walk(item)
			}
		case map[string]any:
			if typed["$type"] == "blob" {
				ref, _ := typed["ref"].(map[string]any)
				cid, _ := ref["$link"].(string)
				mimeType, _ := typed["mimeType"].(string)
				size, ok := positiveIntegerAsInt64(typed["size"])
				if cid != "" && mimeType != "" && ok {
					blobs = append(blobs, pdscommands.BlobReference{CID: syntax.CID(cid), MIMEType: mimeType, Size: size})
				}
			}
			for _, item := range typed {
				walk(item)
			}
		}
	}
	walk(record)
	slices.SortFunc(blobs, func(left, right pdscommands.BlobReference) int {
		if compared := strings.Compare(left.CID.String(), right.CID.String()); compared != 0 {
			return compared
		}
		if compared := strings.Compare(left.MIMEType, right.MIMEType); compared != 0 {
			return compared
		}
		return int(left.Size - right.Size)
	})
	return slices.Compact(blobs)
}

func DeleteBusinessProfileHandler(
	newEffects pdseffects.ExecutorFactory,
	options ...BusinessProfileHandlerOptions,
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
		expectedCID, err := ParseBusinessIfMatch(r)
		if err != nil || expectedCID == "*" {
			WritePDSRecordConflict(w, runID)
			return
		}
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
			uri := syntax.ATURI("at://" + owner.String() + "/" + businessProfileNSID.String() + "/" + businessProfileRkey.String())
			intent, err := json.Marshal(struct {
				URI         syntax.ATURI `json:"uri"`
				ExpectedCID syntax.CID   `json:"expectedCid"`
			}{URI: uri, ExpectedCID: expectedCID})
			if err != nil {
				observability.ReportRequestFailure(r.Context(), err, "api.DeleteBusinessProfileHandler", "handler")
				envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "could not prepare business profile delete", runID, nil)
				return
			}
			sessionID, _ := middleware.GetOAuthSessionID(r.Context())
			result, err := commands.Delete(r.Context(), pdscommands.AddressedDeleteCommandRequest{
				Owner: owner, OwnerGeneration: generation, SessionID: sessionID,
				OperationKind: "business_profile.delete", OperationKey: operationKey,
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
		owner, generation, executor, expectedOwners, ok := businessProfileExecutor(w, r, runID, newEffects)
		if !ok {
			return
		}
		var current map[string]any
		currentCID, readErr := executor.ReadRecord(
			r.Context(), businessProfileReadRequest(owner, generation, expectedOwners), &current,
		)
		if readErr != nil {
			if errors.Is(readErr, auth.ErrRecordNotFound) {
				WritePDSRecordConflict(w, runID)
				return
			}
			writePDSError(w, http.StatusBadGateway, "pds_read_failed", "could not read business profile", runID, readErr)
			return
		}
		if RequireCurrentCID(expectedCID, currentCID) != nil {
			WritePDSRecordConflict(w, runID)
			return
		}
		operationID, mutationKey := immediateEffectIdentity(runID, "business_profile.delete")
		_, deleteErr := executor.DeleteRecord(r.Context(), pdseffects.DeleteRecordRequest{
			OperationID: operationID, MutationKey: mutationKey,
			Owner: owner, OwnerGeneration: generation, ExpectedOwners: expectedOwners,
			Collection: businessProfileNSID, Rkey: businessProfileRkey, ExpectedCID: expectedCID,
		})
		if isPDSRecordConflict(deleteErr) {
			WritePDSRecordConflict(w, runID)
			return
		}
		if deleteErr != nil {
			writePDSError(w, http.StatusBadGateway, "pds_write_failed", "could not delete business profile", runID, deleteErr)
			return
		}
		envelope.WriteJSON(w, http.StatusOK, businessProfileResponse{CID: expectedCID})
	})
}

func businessProfileExecutor(
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
	generation, ok := requirePDSEffectGeneration(w, r, runID)
	if !ok {
		return "", 0, nil, nil, false
	}
	sessionID, _ := middleware.GetOAuthSessionID(r.Context())
	executor, err := newPDSEffectExecutor(r.Context(), newEffects, owner, sessionID)
	if err != nil {
		writePDSError(w, http.StatusBadGateway, "pds_unavailable", "could not contact PDS", runID, err)
		return "", 0, nil, nil, false
	}
	expectedOwners, err := executor.ResolveExpectedOwners(r.Context(), generation, nil)
	if err != nil {
		writePDSError(w, http.StatusBadGateway, "pds_write_failed", "could not prepare business profile mutation", runID, err)
		return "", 0, nil, nil, false
	}
	return owner, generation, executor, expectedOwners, true
}

func businessProfileReadRequest(
	owner syntax.DID,
	generation int64,
	expectedOwners []ownerlifecycle.ExpectedOwner,
) pdseffects.ReadRecordRequest {
	return pdseffects.ReadRecordRequest{
		Owner: owner, OwnerGeneration: generation, ExpectedOwners: expectedOwners,
		Collection: businessProfileNSID, Rkey: businessProfileRkey,
	}
}

func decodeBusinessProfileRequest(body io.Reader) (map[string]any, *FieldError) {
	raw, err := io.ReadAll(io.LimitReader(body, maxBusinessProfileBodyBytes+1))
	if err != nil || len(raw) > maxBusinessProfileBodyBytes {
		return nil, &FieldError{Code: "malformed_body", Fields: map[string]string{"_": "invalid request body"}}
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return nil, &FieldError{Code: "malformed_body", Fields: map[string]string{"_": "invalid JSON object"}}
	}
	for field, value := range fields {
		if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, &FieldError{Code: "validation_failed", Fields: map[string]string{field: "must not be null"}}
		}
	}
	var request businessProfileRequest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return nil, &FieldError{Code: "unexpected_field", Fields: map[string]string{"_": err.Error()}}
	}
	if validationErr := validateBusinessProfileRequest(&request); validationErr != nil {
		return nil, validationErr
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		return nil, &FieldError{Code: "malformed_body", Fields: map[string]string{"_": "invalid request body"}}
	}
	var record map[string]any
	if err := json.Unmarshal(encoded, &record); err != nil {
		return nil, &FieldError{Code: "malformed_body", Fields: map[string]string{"_": "invalid request body"}}
	}
	record["$type"] = businessProfileNSID.String()
	return record, nil
}

func validateBusinessProfileRequest(request *businessProfileRequest) *FieldError {
	fields := make(map[string]string)
	if request.PreserveProducts && request.Products != nil {
		fields["products"] = "must be omitted when preserveProducts is true"
	}
	if values, err := business.ValidateBusinessTypes(request.BusinessTypes); err != nil {
		fields["businessTypes"] = "contains unsupported or duplicate values"
	} else {
		request.BusinessTypes = values
	}
	if values, err := business.ValidateOfferings(request.Offerings); err != nil {
		fields["offerings"] = "contains unsupported or duplicate values"
	} else {
		request.Offerings = values
	}
	validateOptionalBusinessText(fields, "tagline", business.TextFieldTagline, request.Tagline)
	validateOptionalBusinessText(fields, "hoursNote", business.TextFieldHoursNote, request.HoursNote)
	validateOptionalBusinessText(fields, "serviceArea", business.TextFieldServiceArea, request.ServiceArea)
	if request.Location != nil {
		country, err := business.NormalizeCountry(request.Location.Country)
		if err != nil {
			fields["location.country"] = "must be a recognized two-letter country code"
		} else {
			request.Location.Country = country
		}
		validateOptionalBusinessText(fields, "location.locality", business.TextFieldLocality, request.Location.Locality)
	}
	if request.PrimaryAction != nil {
		if business.ValidateActions([]business.Action{*request.PrimaryAction}) != nil {
			fields["primaryAction.type"] = "is not a supported value"
		} else {
			err := business.ValidateWebDestination(request.PrimaryAction.Destination)
			if request.PrimaryAction.Type == "email" {
				err = business.ValidateMailtoDestination(request.PrimaryAction.Destination)
			}
			if err != nil {
				fields["primaryAction.destination"] = "is invalid"
			}
		}
	}
	products := make([]business.Product, 0, len(request.Products))
	for index, product := range request.Products {
		prefix := "products[" + formatInt64(int64(index)) + "]"
		if business.ValidateText(business.TextFieldProductTitle, product.Title) != nil {
			fields[prefix+".title"] = "is invalid"
		}
		if business.ValidateWebDestination(product.URI) != nil {
			fields[prefix+".uri"] = "is invalid"
		}
		if product.Image == nil {
			fields[prefix+".image"] = "is required"
		} else {
			if len(product.Image.Image) == 0 {
				fields[prefix+".image.image"] = "is required"
			} else {
				validateProfileImageUpdate(fields, prefix+".image.image", ProfileImageUpdate{Present: true, Blob: product.Image.Image}, DefaultMediaLimits())
			}
			if business.ValidateText(business.TextFieldImageAlt, product.Image.Alt) != nil {
				fields[prefix+".image.alt"] = "is invalid"
			}
			if ratio := product.Image.AspectRatio; ratio != nil && (ratio.Width <= 0 || ratio.Height <= 0) {
				fields[prefix+".image.aspectRatio"] = "must have positive dimensions"
			}
		}
		if product.Price != nil && business.ValidatePrice(*product.Price) != nil {
			fields[prefix+".price"] = "is invalid"
		}
		products = append(products, business.Product{Title: product.Title, URI: product.URI})
	}
	if _, err := business.ValidateProductCollection(products); err != nil {
		fields["products"] = "must contain at most four products with distinct URIs"
	}
	if len(fields) != 0 {
		return &FieldError{Code: "validation_failed", Fields: fields}
	}
	return nil
}

func validateOptionalBusinessText(fields map[string]string, name string, kind business.TextField, value *string) {
	if value != nil && business.ValidateText(kind, *value) != nil {
		fields[name] = "is invalid"
	}
}

func isPDSRecordConflict(err error) bool {
	return errors.Is(err, auth.ErrRecordNotFound) || errors.Is(err, auth.ErrRecordSwapConflict) ||
		errors.Is(err, pdseffects.ErrEffectConflict)
}

// Preservation controls are API-only intent. Copy raw PDS fields rather than
// reserializing their lossy, display-safe projection.
func buildBusinessProfileReplacement(current json.RawMessage, replacement map[string]any) (json.RawMessage, error) {
	encoded, err := json.Marshal(replacement)
	if err != nil {
		return nil, err
	}
	var desired map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &desired); err != nil {
		return nil, err
	}
	preserveProducts, _ := replacement["preserveProducts"].(bool)
	preserveUnknown, _ := replacement["preserveUnknownCatalogValues"].(bool)
	delete(desired, "preserveProducts")
	delete(desired, "preserveUnknownCatalogValues")
	var existing map[string]json.RawMessage
	if len(current) != 0 {
		if err := json.Unmarshal(current, &existing); err != nil || existing == nil {
			return nil, business.ErrInvalidProfile
		}
	}
	if preserveProducts {
		if products, ok := existing["products"]; ok {
			desired["products"] = products
		}
	}
	if preserveUnknown {
		for _, field := range []string{"businessTypes", "offerings"} {
			var previous, selected []string
			if raw, ok := existing[field]; ok {
				if err := json.Unmarshal(raw, &previous); err != nil {
					return nil, business.ErrInvalidProfile
				}
			}
			if raw, ok := desired[field]; ok {
				if err := json.Unmarshal(raw, &selected); err != nil {
					return nil, err
				}
			}
			catalog := business.BusinessTypeCatalog()
			if field == "offerings" {
				catalog = business.OfferingCatalog()
			}
			for _, value := range previous {
				if !slices.Contains(catalog, value) && !slices.Contains(selected, value) {
					selected = append(selected, value)
				}
			}
			if len(selected) > 20 {
				return nil, &FieldError{Code: "validation_failed", Fields: map[string]string{field: "must contain at most twenty values including preserved values"}}
			}
			if len(selected) > 0 {
				desired[field], _ = json.Marshal(selected)
			}
		}
	}
	raw, err := json.Marshal(desired)
	if err != nil || len(current) == 0 {
		return raw, err
	}
	return business.MergeProfileReplacement(current, raw)
}

func rejectedBusinessProfileResult(runID string, err error) pdscommands.TerminalResult {
	var fieldErr *FieldError
	if !errors.As(err, &fieldErr) {
		return RejectedCommandResult(runID, err)
	}
	body, _ := json.Marshal(envelope.Error{Error: fieldErr.Code, Message: "invalid business profile", RequestID: runID, Fields: fieldErr.Fields})
	return pdscommands.TerminalResult{State: pdscommands.CommandRejected, HTTPStatus: http.StatusUnprocessableEntity, ResponseBody: body}
}
