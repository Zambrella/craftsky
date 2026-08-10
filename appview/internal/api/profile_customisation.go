package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

const maxProfileCustomisationBodyBytes = 1024

// ProfileCustomisation is the public wire value stored for a Craftsky member.
// The client renders these stable keys using its bundled visual catalogue.
type ProfileCustomisation struct {
	Colour     string `json:"colour"`
	Border     string `json:"profileBorder"`
	Background string `json:"profileBackground"`
}

type ProfileCustomisationWriter interface {
	Put(context.Context, syntax.DID, ProfileCustomisation) (ProfileCustomisation, error)
}

var (
	ProfileColourCatalogue = []string{
		"cobalt",
		"orchid",
		"rose",
		"amber",
		"lime",
		"teal",
		"ink",
	}
	ProfileBorderCatalogue = []string{
		"thin",
		"medium",
		"thick",
	}
	ProfileBackgroundCatalogue = []string{
		"none",
		"bayerdark",
		"cubedark",
		"dotcrossdark",
		"scallopdark",
		"skewdark",
		"x2",
	}
)

var DefaultProfileCustomisation = ProfileCustomisation{
	Colour:     "cobalt",
	Border:     "medium",
	Background: "none",
}

func defaultProfileCustomisationPointer() *ProfileCustomisation {
	value := DefaultProfileCustomisation
	return &value
}

// EffectiveProfileCustomisation replaces unsupported values independently so
// a future key in one field cannot discard supported sibling values.
func EffectiveProfileCustomisation(value ProfileCustomisation) ProfileCustomisation {
	if !catalogueContains(ProfileColourCatalogue, value.Colour) {
		value.Colour = DefaultProfileCustomisation.Colour
	}
	if !catalogueContains(ProfileBorderCatalogue, value.Border) {
		value.Border = DefaultProfileCustomisation.Border
	}
	if !catalogueContains(ProfileBackgroundCatalogue, value.Background) {
		value.Background = DefaultProfileCustomisation.Background
	}
	return value
}

// DecodeProfileCustomisationPut decodes the complete replacement body for a
// profile customisation mutation. It accepts stable catalogue keys only; the
// body cannot carry colours, URLs, assets, or other rendering resources.
func DecodeProfileCustomisationPut(body io.Reader) (ProfileCustomisation, error) {
	raw, err := io.ReadAll(io.LimitReader(body, maxProfileCustomisationBodyBytes+1))
	if err != nil {
		return ProfileCustomisation{}, malformedProfileCustomisationBody(err.Error())
	}
	if len(raw) > maxProfileCustomisationBodyBytes {
		return ProfileCustomisation{}, malformedProfileCustomisationBody("body exceeds 1024 bytes")
	}
	if err := rejectDuplicateProfileCustomisationKeys(raw); err != nil {
		return ProfileCustomisation{}, malformedProfileCustomisationBody(err.Error())
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return ProfileCustomisation{}, malformedProfileCustomisationBody(err.Error())
	}

	allowed := map[string]struct{}{
		"colour":            {},
		"profileBorder":     {},
		"profileBackground": {},
	}
	unexpected := make(map[string]string)
	for field := range fields {
		if _, ok := allowed[field]; !ok {
			unexpected[field] = "unknown field"
		}
	}
	if len(unexpected) > 0 {
		return ProfileCustomisation{}, &FieldError{
			Code:   "unexpected_field",
			Fields: unexpected,
		}
	}

	value := ProfileCustomisation{}
	invalid := make(map[string]string)
	unsupported := make(map[string]string)
	decodeProfileCustomisationKey(fields, "colour", ProfileColourCatalogue, &value.Colour, invalid, unsupported)
	decodeProfileCustomisationKey(fields, "profileBorder", ProfileBorderCatalogue, &value.Border, invalid, unsupported)
	decodeProfileCustomisationKey(fields, "profileBackground", ProfileBackgroundCatalogue, &value.Background, invalid, unsupported)
	if len(invalid) > 0 {
		return ProfileCustomisation{}, &FieldError{
			Code:   "invalid_request",
			Fields: invalid,
		}
	}
	if len(unsupported) > 0 {
		return ProfileCustomisation{}, &FieldError{
			Code:   "validation_failed",
			Fields: unsupported,
		}
	}
	return value, nil
}

func rejectDuplicateProfileCustomisationKeys(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	opening, err := decoder.Token()
	if err != nil {
		return err
	}
	if delimiter, ok := opening.(json.Delim); !ok || delimiter != '{' {
		return errors.New("body must be a JSON object")
	}
	seen := make(map[string]struct{})
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := token.(string)
		if !ok {
			return errors.New("object key must be a string")
		}
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("duplicate field %q", key)
		}
		seen[key] = struct{}{}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return err
		}
	}
	if _, err := decoder.Token(); err != nil {
		return err
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("body contains trailing JSON")
		}
		return err
	}
	return nil
}

func PutProfileCustomisationHandler(store ProfileCustomisationWriter) http.Handler {
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
		value, err := DecodeProfileCustomisationPut(r.Body)
		if err != nil {
			if fieldErr, ok := err.(*FieldError); ok {
				status := http.StatusBadRequest
				if fieldErr.Code == "validation_failed" {
					status = http.StatusUnprocessableEntity
				}
				envelope.WriteError(w, status, fieldErr.Code, "invalid profile customisation", runID, fieldErr.Fields)
				return
			}
			envelope.WriteError(w, http.StatusBadRequest, "malformed_body", "invalid profile customisation", runID, nil)
			return
		}
		stored, err := store.Put(r.Context(), owner, value)
		if err != nil {
			envelope.WriteError(w, http.StatusInternalServerError, "internal_error", "profile customisation update failed", runID, nil)
			return
		}
		envelope.WriteJSON(w, http.StatusOK, stored)
	})
}

func decodeProfileCustomisationKey(
	fields map[string]json.RawMessage,
	field string,
	catalogue []string,
	destination *string,
	invalid map[string]string,
	unsupported map[string]string,
) {
	raw, ok := fields[field]
	if !ok {
		invalid[field] = "is required"
		return
	}
	if err := json.Unmarshal(raw, destination); err != nil {
		invalid[field] = "must be a string"
		return
	}
	if !catalogueContains(catalogue, *destination) {
		unsupported[field] = "is not a supported value"
	}
}

func malformedProfileCustomisationBody(detail string) *FieldError {
	return &FieldError{
		Code:   "malformed_body",
		Fields: map[string]string{"_": detail},
	}
}

func catalogueContains(catalogue []string, value string) bool {
	for _, candidate := range catalogue {
		if candidate == value {
			return true
		}
	}
	return false
}
