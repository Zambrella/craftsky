package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// FolderAssignment preserves the difference between an omitted folderId and
// an explicit JSON null. Existing saves preserve their folder for omission and
// become unfiled only for explicit null.
type FolderAssignment struct {
	Present bool
	ID      *string
}

// DecodeSavePostRequest accepts an absent body for the low-friction unfiled
// save action and otherwise enforces the strict v1 JSON contract.
func DecodeSavePostRequest(body io.Reader) (FolderAssignment, error) {
	raw, err := io.ReadAll(body)
	if err != nil {
		return FolderAssignment{}, savedPostBodyError("malformed_body", err.Error())
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return FolderAssignment{}, nil
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return FolderAssignment{}, savedPostBodyError("malformed_body", err.Error())
	}
	if fields == nil {
		return FolderAssignment{}, savedPostBodyError("malformed_body", "body must be a JSON object")
	}
	for key := range fields {
		if key != "folderId" {
			return FolderAssignment{}, &FieldError{
				Code:   "unexpected_field",
				Fields: map[string]string{key: "not writable in v1"},
			}
		}
	}

	rawFolderID, present := fields["folderId"]
	if !present {
		return FolderAssignment{}, nil
	}
	assignment := FolderAssignment{Present: true}
	if bytes.Equal(bytes.TrimSpace(rawFolderID), []byte("null")) {
		return assignment, nil
	}
	var folderID string
	if err := json.Unmarshal(rawFolderID, &folderID); err != nil {
		return FolderAssignment{}, savedPostBodyError("malformed_body", err.Error())
	}
	assignment.ID = &folderID
	return assignment, nil
}

func savedPostBodyError(code, detail string) *FieldError {
	return &FieldError{Code: code, Fields: map[string]string{"_": detail}}
}

func DecodeSavedPostFolderRequest(body io.Reader) (string, error) {
	raw, err := io.ReadAll(body)
	if err != nil {
		return "", savedPostBodyError("malformed_body", err.Error())
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		if err == nil {
			return "", savedPostBodyError("malformed_body", "body must be a JSON object")
		}
		return "", savedPostBodyError("malformed_body", err.Error())
	}
	for key := range fields {
		if key != "name" {
			return "", &FieldError{Code: "unexpected_field", Fields: map[string]string{key: "not writable in v1"}}
		}
	}
	rawName, present := fields["name"]
	if !present {
		return "", &FieldError{Code: "validation_failed", Fields: map[string]string{"name": "is required"}}
	}
	var name string
	if err := json.Unmarshal(rawName, &name); err != nil {
		return "", savedPostBodyError("malformed_body", err.Error())
	}
	return NormalizeSavedPostFolderName(name)
}

func ParseSavedPostListQuery(values url.Values) (SavedPostListFilter, error) {
	filter := SavedPostListFilter{Scope: SavedPostScopeAll, Sort: SavedPostSortNewest, Limit: 50}
	allowed := map[string]bool{"folderId": true, "unfiled": true, "sort": true, "limit": true, "cursor": true}
	for key, entries := range values {
		if !allowed[key] || len(entries) != 1 {
			return SavedPostListFilter{}, &FieldError{Code: "validation_failed", Fields: map[string]string{key: "is invalid"}}
		}
	}
	if folderID, present := singleQueryValue(values, "folderId"); present {
		if folderID == "" {
			return SavedPostListFilter{}, &FieldError{Code: "validation_failed", Fields: map[string]string{"folderId": "must not be empty"}}
		}
		filter.Scope = SavedPostScopeFolder
		filter.FolderID = folderID
	}
	if unfiled, present := singleQueryValue(values, "unfiled"); present {
		if unfiled != "true" {
			return SavedPostListFilter{}, &FieldError{Code: "validation_failed", Fields: map[string]string{"unfiled": "must be true when provided"}}
		}
		if filter.Scope == SavedPostScopeFolder {
			return SavedPostListFilter{}, &FieldError{Code: "validation_failed", Fields: map[string]string{"unfiled": "cannot be combined with folderId"}}
		}
		filter.Scope = SavedPostScopeUnfiled
	}
	if sortValue, present := singleQueryValue(values, "sort"); present {
		filter.Sort = SavedPostSort(sortValue)
		if !validSavedPostSort(filter.Sort) {
			return SavedPostListFilter{}, &FieldError{Code: "validation_failed", Fields: map[string]string{"sort": "must be newest or oldest"}}
		}
	}
	if limitValue, present := singleQueryValue(values, "limit"); present {
		limit, err := strconv.Atoi(limitValue)
		if err != nil || limit < 1 || limit > 100 {
			return SavedPostListFilter{}, &FieldError{Code: "validation_failed", Fields: map[string]string{"limit": "must be an integer from 1 to 100"}}
		}
		filter.Limit = limit
	}
	filter.Cursor, _ = singleQueryValue(values, "cursor")
	return filter, nil
}

func ParseSavedPostFolderListQuery(values url.Values) (limit int, cursor string, err error) {
	limit = 50
	for key, entries := range values {
		if (key != "limit" && key != "cursor") || len(entries) != 1 {
			return 0, "", &FieldError{Code: "validation_failed", Fields: map[string]string{key: "is invalid"}}
		}
	}
	if limitValue, present := singleQueryValue(values, "limit"); present {
		parsed, parseErr := strconv.Atoi(limitValue)
		if parseErr != nil || parsed < 1 || parsed > 100 {
			return 0, "", &FieldError{Code: "validation_failed", Fields: map[string]string{"limit": "must be an integer from 1 to 100"}}
		}
		limit = parsed
	}
	cursor, _ = singleQueryValue(values, "cursor")
	return limit, cursor, nil
}

func singleQueryValue(values url.Values, key string) (string, bool) {
	entries, present := values[key]
	if !present || len(entries) == 0 {
		return "", false
	}
	return entries[0], true
}

// NormalizeSavedPostFolderName applies the saved-folder wire contract before
// persistence. Folder identity is independent of this display value, so the
// function deliberately performs no uniqueness or case folding.
func NormalizeSavedPostFolderName(name string) (string, error) {
	normalized := strings.TrimSpace(name)
	fields := map[string]string{}
	switch count := utf8.RuneCountInString(normalized); {
	case count == 0:
		fields["name"] = "must contain at least 1 character"
	case count > 100:
		fields["name"] = "exceeds 100 characters"
	}
	if strings.ContainsAny(normalized, `/\`) {
		fields["name"] = "must not contain slash or backslash"
	}
	for _, r := range normalized {
		if unicode.IsControl(r) {
			fields["name"] = "must not contain control characters"
			break
		}
	}
	if len(fields) > 0 {
		return "", &FieldError{Code: "validation_failed", Fields: fields}
	}
	return normalized, nil
}
