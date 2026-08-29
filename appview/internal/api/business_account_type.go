package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/business"
	"social.craftsky/appview/internal/middleware"
)

const maxBusinessAccountTypeBodyBytes = 1024

type BusinessAccountTypeWriter interface {
	PutAccountType(context.Context, syntax.DID, business.AccountType) error
}

type businessAccountTypeResponse struct {
	AccountType business.AccountType `json:"accountType"`
}

func PutBusinessAccountTypeHandler(store BusinessAccountTypeWriter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		runID := middleware.GetRunID(r.Context())
		if len(r.URL.Query()) != 0 {
			envelope.WriteError(w, http.StatusBadRequest, "invalid_request", "query parameters are not supported", runID, nil)
			return
		}
		owner, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError, "missing_authenticated_did", "authenticated DID missing", runID, nil)
			return
		}
		accountType, fieldErr := decodeBusinessAccountType(r.Body)
		if fieldErr != nil {
			status := http.StatusBadRequest
			if fieldErr.Code == "validation_failed" {
				status = http.StatusUnprocessableEntity
			}
			envelope.WriteError(w, status, fieldErr.Code, "invalid account type", runID, fieldErr.Fields)
			return
		}
		if err := store.PutAccountType(r.Context(), owner, accountType); err != nil {
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "account type update failed", runID, nil)
			return
		}
		envelope.WriteJSON(w, http.StatusOK, businessAccountTypeResponse{AccountType: accountType})
	})
}

func decodeBusinessAccountType(body io.Reader) (business.AccountType, *FieldError) {
	raw, err := io.ReadAll(io.LimitReader(body, maxBusinessAccountTypeBodyBytes+1))
	if err != nil || len(raw) > maxBusinessAccountTypeBodyBytes {
		return "", &FieldError{Code: "malformed_body", Fields: map[string]string{"_": "invalid request body"}}
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return "", &FieldError{Code: "malformed_body", Fields: map[string]string{"_": err.Error()}}
	}
	for field := range fields {
		if field != "accountType" {
			return "", &FieldError{Code: "unexpected_field", Fields: map[string]string{field: "unknown field"}}
		}
	}
	value, ok := fields["accountType"]
	if !ok {
		return "", &FieldError{Code: "invalid_request", Fields: map[string]string{"accountType": "is required"}}
	}
	var rawAccountType string
	if err := json.Unmarshal(value, &rawAccountType); err != nil {
		return "", &FieldError{Code: "invalid_request", Fields: map[string]string{"accountType": "must be a string"}}
	}
	accountType, err := business.ParseAccountType(rawAccountType)
	if errors.Is(err, business.ErrInvalidAccountType) {
		return "", &FieldError{Code: "validation_failed", Fields: map[string]string{"accountType": "is not a supported value"}}
	}
	return accountType, nil
}
