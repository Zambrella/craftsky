// appview/internal/api/profile_request.go
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"unicode/utf8"
)

// ProfilePutRequest is the decoded request body for PUT /v1/profiles/me.
// Avatar and banner are deliberately absent — the handler rejects bodies
// that carry them. See spec §5.3.
type ProfilePutRequest struct {
	DisplayName *string  `json:"displayName,omitempty"`
	Description *string  `json:"description,omitempty"`
	Crafts      []string `json:"crafts,omitempty"`
}

// FieldError is returned by DecodeProfilePut and ValidateProfilePut when
// the request body has per-field problems. Handlers translate it into
// either 400 unexpected_field or 422 validation_failed per spec §5.3.
type FieldError struct {
	Code   string
	Fields map[string]string
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("%s: %v", e.Code, e.Fields)
}

// DecodeProfilePut reads a JSON body into ProfilePutRequest, rejecting
// any unknown keys and any occurrence of "avatar" or "banner" (which
// are deliberately not writable in v1). Returns a *FieldError with
// Code = "unexpected_field" in the latter case.
func DecodeProfilePut(body io.Reader) (ProfilePutRequest, error) {
	raw, err := io.ReadAll(body)
	if err != nil {
		return ProfilePutRequest{}, &FieldError{
			Code:   "malformed_body",
			Fields: map[string]string{"_": err.Error()},
		}
	}
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		return ProfilePutRequest{}, &FieldError{
			Code:   "malformed_body",
			Fields: map[string]string{"_": err.Error()},
		}
	}
	rejected := map[string]string{}
	for _, k := range []string{"avatar", "banner"} {
		if _, present := rawMap[k]; present {
			rejected[k] = "not writable in v1"
		}
	}
	if len(rejected) > 0 {
		return ProfilePutRequest{}, &FieldError{
			Code:   "unexpected_field",
			Fields: rejected,
		}
	}
	out := ProfilePutRequest{}
	strict := json.NewDecoder(bytes.NewReader(raw))
	strict.DisallowUnknownFields()
	if err := strict.Decode(&out); err != nil {
		return ProfilePutRequest{}, &FieldError{
			Code:   "unexpected_field",
			Fields: map[string]string{"_": err.Error()},
		}
	}
	return out, nil
}

// ValidateProfilePut enforces the length/count constraints from spec §5.3.
// NOTE: "graphemes" per spec are strictly Unicode extended grapheme clusters,
// which the stdlib can't count without golang.org/x/text. For v1 we accept
// `utf8.RuneCountInString` as a slightly stricter upper bound (combining
// marks count as extra runes), matching how other atproto clients ship.
func ValidateProfilePut(req ProfilePutRequest) error {
	fields := map[string]string{}
	if req.DisplayName != nil {
		if len(*req.DisplayName) > 640 || utf8.RuneCountInString(*req.DisplayName) > 64 {
			fields["displayName"] = "exceeds 64 graphemes / 640 bytes"
		}
	}
	if req.Description != nil {
		if len(*req.Description) > 2560 || utf8.RuneCountInString(*req.Description) > 256 {
			fields["description"] = "exceeds 256 graphemes / 2560 bytes"
		}
	}
	if req.Crafts != nil {
		if len(req.Crafts) > 10 {
			fields["crafts"] = "exceeds maximum of 10 entries"
		}
		for i, c := range req.Crafts {
			if len(c) > 50 || utf8.RuneCountInString(c) > 50 {
				fields[fmt.Sprintf("crafts[%d]", i)] = "exceeds 50 graphemes / 50 bytes"
			}
		}
	}
	if len(fields) > 0 {
		return &FieldError{Code: "validation_failed", Fields: fields}
	}
	return nil
}
