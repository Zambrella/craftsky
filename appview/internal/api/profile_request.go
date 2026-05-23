// appview/internal/api/profile_request.go
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
	"unicode/utf8"
)

// ProfilePutRequest is the decoded request body for PUT /v1/profiles/me.
type ProfilePutRequest struct {
	DisplayName *string            `json:"displayName,omitempty"`
	Description *string            `json:"description,omitempty"`
	Crafts      []string           `json:"crafts,omitempty"`
	Avatar      ProfileImageUpdate `json:"-"`
	Banner      ProfileImageUpdate `json:"-"`
}

// ProfileImageUpdate is a tri-state field used for avatar/banner updates:
// absent preserves the current blob, present with nil clears it, present
// with a blob replaces it.
type ProfileImageUpdate struct {
	Present bool
	Blob    map[string]any
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
// unknown keys. avatar/banner accept an atproto blob object or explicit
// null; omitted fields preserve the current profile image.
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
	allowed := map[string]struct{}{
		"displayName": {},
		"description": {},
		"crafts":      {},
		"avatar":      {},
		"banner":      {},
	}
	unknown := map[string]string{}
	for k := range rawMap {
		if _, ok := allowed[k]; !ok {
			unknown[k] = "unknown field"
		}
	}
	if len(unknown) > 0 {
		return ProfilePutRequest{}, &FieldError{
			Code:   "unexpected_field",
			Fields: unknown,
		}
	}

	type scalarProfilePutRequest struct {
		DisplayName *string  `json:"displayName,omitempty"`
		Description *string  `json:"description,omitempty"`
		Crafts      []string `json:"crafts,omitempty"`
	}
	scalarMap := map[string]json.RawMessage{}
	for _, k := range []string{"displayName", "description", "crafts"} {
		if v, ok := rawMap[k]; ok {
			scalarMap[k] = v
		}
	}
	scalarRaw, err := json.Marshal(scalarMap)
	if err != nil {
		return ProfilePutRequest{}, &FieldError{
			Code:   "malformed_body",
			Fields: map[string]string{"_": err.Error()},
		}
	}
	var scalars scalarProfilePutRequest
	strict := json.NewDecoder(bytes.NewReader(scalarRaw))
	strict.DisallowUnknownFields()
	if err := strict.Decode(&scalars); err != nil {
		return ProfilePutRequest{}, &FieldError{
			Code:   "unexpected_field",
			Fields: map[string]string{"_": err.Error()},
		}
	}
	out := ProfilePutRequest{
		DisplayName: scalars.DisplayName,
		Description: scalars.Description,
		Crafts:      scalars.Crafts,
	}
	if rawAvatar, present := rawMap["avatar"]; present {
		update, err := decodeProfileImageUpdate("avatar", rawAvatar)
		if err != nil {
			return ProfilePutRequest{}, err
		}
		out.Avatar = update
	}
	if rawBanner, present := rawMap["banner"]; present {
		update, err := decodeProfileImageUpdate("banner", rawBanner)
		if err != nil {
			return ProfilePutRequest{}, err
		}
		out.Banner = update
	}
	return out, nil
}

func decodeProfileImageUpdate(field string, raw json.RawMessage) (ProfileImageUpdate, error) {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return ProfileImageUpdate{Present: true}, nil
	}
	var blob map[string]any
	if err := json.Unmarshal(raw, &blob); err != nil {
		return ProfileImageUpdate{}, &FieldError{
			Code:   "malformed_body",
			Fields: map[string]string{field: err.Error()},
		}
	}
	if len(blob) == 0 {
		return ProfileImageUpdate{}, &FieldError{
			Code:   "validation_failed",
			Fields: map[string]string{field: "must be a blob object or null"},
		}
	}
	return ProfileImageUpdate{Present: true, Blob: blob}, nil
}

// ValidateProfilePut enforces the length/count constraints from spec §5.3.
// NOTE: "graphemes" per spec are strictly Unicode extended grapheme clusters,
// which the stdlib can't count without golang.org/x/text. For v1 we accept
// `utf8.RuneCountInString` as a slightly stricter upper bound (combining
// marks count as extra runes), matching how other atproto clients ship.
func ValidateProfilePut(req ProfilePutRequest) error {
	return ValidateProfilePutWithLimits(req, DefaultMediaLimits())
}

func ValidateProfilePutWithLimits(req ProfilePutRequest, limits MediaLimits) error {
	limits = normalizeMediaLimits(limits)
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
	validateProfileImageUpdate(fields, "avatar", req.Avatar, limits)
	validateProfileImageUpdate(fields, "banner", req.Banner, limits)
	if len(fields) > 0 {
		return &FieldError{Code: "validation_failed", Fields: fields}
	}
	return nil
}

func validateProfileImageUpdate(fields map[string]string, prefix string, update ProfileImageUpdate, limits MediaLimits) {
	if !update.Present || update.Blob == nil {
		return
	}
	validatePostImageBlob(fields, prefix, update.Blob)
	if typ, ok := update.Blob["$type"].(string); ok && typ != "blob" {
		fields[prefix+".$type"] = "must be blob"
	}
	mime, _ := update.Blob["mimeType"].(string)
	mime = strings.TrimSpace(mime)
	if _, ok := allowedImageUploadMIMETypes[mime]; mime != "" && !ok {
		fields[prefix+".mimeType"] = "must be one of image/jpeg, image/png, image/webp"
	}
	if size, ok := positiveIntegerAsInt64(update.Blob["size"]); ok && size > limits.MaxImageUploadBytes {
		fields[prefix+".size"] = "must be <= " + formatInt64(limits.MaxImageUploadBytes) + " bytes"
	}
}

func positiveIntegerAsInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), n > 0
	case int8:
		return int64(n), n > 0
	case int16:
		return int64(n), n > 0
	case int32:
		return int64(n), n > 0
	case int64:
		return n, n > 0
	case uint:
		return int64(n), n > 0
	case uint8:
		return int64(n), n > 0
	case uint16:
		return int64(n), n > 0
	case uint32:
		return int64(n), n > 0
	case uint64:
		if n > uint64(math.MaxInt64) {
			return 0, false
		}
		return int64(n), n > 0
	case float32:
		return int64(n), n > 0 && n == float32(int64(n))
	case float64:
		return int64(n), n > 0 && n == float64(int64(n))
	case json.Number:
		i, err := n.Int64()
		return i, err == nil && i > 0
	default:
		return 0, false
	}
}
