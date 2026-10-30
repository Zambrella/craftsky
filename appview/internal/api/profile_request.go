// appview/internal/api/profile_request.go
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/rivo/uniseg"
	"social.craftsky/appview/internal/profilelinks"
)

// ProfilePutRequest is the decoded request body for PUT /v1/profiles/me.
type ProfilePutRequest struct {
	Links        *[]ProfileLinkInput      `json:"links,omitempty"`
	RemovedLinks []profilelinks.StrongRef `json:"removedLinks,omitempty"`
	DisplayName  *string                  `json:"displayName,omitempty"`
	Description  *string                  `json:"description,omitempty"`
	Pronouns     *string                  `json:"pronouns,omitempty"`
	Crafts       []string                 `json:"crafts,omitempty"`
	Avatar       ProfileImageUpdate       `json:"-"`
	Banner       ProfileImageUpdate       `json:"-"`
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
		"displayName":  {},
		"description":  {},
		"pronouns":     {},
		"crafts":       {},
		"avatar":       {},
		"banner":       {},
		"links":        {},
		"removedLinks": {},
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
		Pronouns    *string  `json:"pronouns,omitempty"`
		Crafts      []string `json:"crafts,omitempty"`
	}
	scalarMap := map[string]json.RawMessage{}
	for _, k := range []string{"displayName", "description", "pronouns", "crafts"} {
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
		Pronouns:    scalars.Pronouns,
		Crafts:      scalars.Crafts,
	}
	if rawLinks, present := rawMap["links"]; present {
		var links []ProfileLinkInput
		decoder := json.NewDecoder(bytes.NewReader(rawLinks))
		decoder.DisallowUnknownFields()
		if bytes.Equal(bytes.TrimSpace(rawLinks), []byte("null")) || decoder.Decode(&links) != nil {
			return ProfilePutRequest{}, &FieldError{Code: "malformed_body", Fields: map[string]string{"links": "must be an array of profile links"}}
		}
		out.Links = &links
	}
	if rawRemoved, present := rawMap["removedLinks"]; present {
		decoder := json.NewDecoder(bytes.NewReader(rawRemoved))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&out.RemovedLinks) != nil {
			return ProfilePutRequest{}, &FieldError{Code: "malformed_body", Fields: map[string]string{"removedLinks": "must be strong references"}}
		}
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
func ValidateProfilePut(req ProfilePutRequest) error {
	return ValidateProfilePutWithLimits(req, DefaultMediaLimits())
}

func ValidateProfilePutWithLimits(req ProfilePutRequest, limits MediaLimits) error {
	limits = normalizeMediaLimits(limits)
	fields := map[string]string{}
	if len(req.RemovedLinks) > 10 || (len(req.RemovedLinks) > 0 && req.Links == nil) {
		fields["removedLinks"] = "requires a profile links update and at most ten removed links"
	}
	if req.Links != nil {
		if len(*req.Links) > profilelinks.MaxLinks {
			fields["links"] = "maximum of 10 links"
		}
		seen := map[string]bool{}
		for i, link := range *req.Links {
			prefix := fmt.Sprintf("links[%d]", i)
			normalized, err := profilelinks.Normalize(link.URL)
			if err != nil || normalized != link.URL {
				fields[prefix+".url"] = "must be a normalized HTTPS URL without a URL shortener"
			}
			if seen[normalized] {
				fields[prefix+".url"] = "duplicate link"
			}
			seen[normalized] = true
			if len(link.Title) > 320 || uniseg.GraphemeClusterCount(link.Title) > 40 {
				fields[prefix+".title"] = "exceeds 40 graphemes / 320 bytes"
			}
			if _, err := syntax.ParseTID(link.Rkey.String()); err != nil {
				fields[prefix+".rkey"] = "must be a TID"
			}
			if _, err := syntax.ParseDatetime(link.CreatedAt); err != nil {
				fields[prefix+".createdAt"] = "must be a datetime"
			}
			if link.Icon != nil {
				iconLimits := limits
				iconLimits.MaxImageUploadBytes = 100_000
				validateProfileImageUpdate(fields, prefix+".icon", ProfileImageUpdate{Present: true, Blob: link.Icon}, iconLimits)
			}
		}
	}
	if req.DisplayName != nil {
		if len(*req.DisplayName) > 640 || uniseg.GraphemeClusterCount(*req.DisplayName) > 64 {
			fields["displayName"] = "exceeds 64 graphemes / 640 bytes"
		}
	}
	if req.Description != nil {
		if len(*req.Description) > 2560 || uniseg.GraphemeClusterCount(*req.Description) > 256 {
			fields["description"] = "exceeds 256 graphemes / 2560 bytes"
		}
	}
	if req.Pronouns != nil {
		if len(*req.Pronouns) > 200 || uniseg.GraphemeClusterCount(*req.Pronouns) > 20 {
			fields["pronouns"] = "exceeds 20 graphemes / 200 bytes"
		}
	}
	if req.Crafts != nil {
		if len(req.Crafts) > 10 {
			fields["crafts"] = "exceeds maximum of 10 entries"
		}
		for i, c := range req.Crafts {
			if len(c) > 50 || uniseg.GraphemeClusterCount(c) > 50 {
				fields[fmt.Sprintf("crafts[%d]", i)] = "exceeds 50 graphemes / 50 bytes"
			}
		}
	}
	// Bluesky profile images have a stricter lexicon limit than post images.
	limits.MaxImageUploadBytes = min(limits.MaxImageUploadBytes, 1_000_000)
	validateActorProfileImageUpdate(fields, "avatar", req.Avatar, limits)
	validateActorProfileImageUpdate(fields, "banner", req.Banner, limits)
	if len(fields) > 0 {
		return &FieldError{Code: "validation_failed", Fields: fields}
	}
	return nil
}

func validateActorProfileImageUpdate(fields map[string]string, prefix string, update ProfileImageUpdate, limits MediaLimits) {
	validateProfileImageUpdate(fields, prefix, update, limits)
	if !update.Present || update.Blob == nil {
		return
	}
	mime, _ := update.Blob["mimeType"].(string)
	mime = strings.TrimSpace(mime)
	if mime != "" && mime != "image/jpeg" && mime != "image/png" {
		fields[prefix+".mimeType"] = "must be one of image/jpeg, image/png"
	}
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
