// appview/internal/api/post_request.go
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/ipfs/go-cid"
	"github.com/rivo/uniseg"

	"social.craftsky/appview/internal/languages"
)

const (
	maxProjectMaterials         = 10
	maxProjectMaterialGraphemes = 100
)

// StrongRef is the wire shape of a strongRef ({uri, cid}). Used for
// reply pointers and quote embeds. cid uses a string rather than
// syntax.CID so unmarshalling never fails on the "informal helper"
// type — the validator runs as a separate step.
type StrongRef struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

// ReplyRef mirrors the lexicon's #replyRef.
type ReplyRef struct {
	Root   StrongRef `json:"root"`
	Parent StrongRef `json:"parent"`
}

// EmbedRequest mirrors what the wire accepts on a create request. Today
// only quote embeds are supported. The wire shape uses {embed: {quote:
// {uri, cid}}}; the AppView translates it to the lexicon's
// {embed: {$type: ..#quoteEmbed, record: {uri, cid}}} before writing.
type EmbedRequest struct {
	Quote    *StrongRef            `json:"quote,omitempty"`
	External *ExternalEmbedRequest `json:"external,omitempty"`
}

type ExternalEmbedRequest struct {
	URI         string         `json:"uri"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Thumb       map[string]any `json:"thumb,omitempty"`
}

// PostCreateRequest is the decoded body of POST /v1/posts.
// createdAt is server-stamped.
type PostCreateRequest struct {
	Text string `json:"text"`
	// Facets is opaque raw JSON deliberately. The lexicon's
	// app.bsky.richtext.facet shape (including a possibly-present "$type"
	// discriminator on the outer object and the inner union variants) is
	// pass-through to the PDS, which validates it. The tag-extraction
	// path in the create handler does its own non-strict decode for the
	// synthetic response.
	Facets  json.RawMessage `json:"facets,omitempty"`
	Reply   *ReplyRef       `json:"reply,omitempty"`
	Embed   *EmbedRequest   `json:"embed,omitempty"`
	Images  []PostImage     `json:"images,omitempty"`
	Project *Project        `json:"project,omitempty"`
	Langs   []string        `json:"langs,omitempty"`
}

type PostImage struct {
	Image       map[string]any        `json:"image"`
	Alt         string                `json:"alt"`
	AspectRatio *PostImageAspectRatio `json:"aspectRatio,omitempty"`
}

type PostImageAspectRatio struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// rejectedPostFields enumerates wire fields that are NOT writable here.
var rejectedPostFields = []string{"createdAt"}

// DecodePostCreate reads a JSON body into PostCreateRequest. Rejects
// any of rejectedPostFields and any unknown keys with code
// "unexpected_field"; malformed JSON with "malformed_body".
func DecodePostCreate(body io.Reader) (PostCreateRequest, error) {
	raw, err := io.ReadAll(body)
	if err != nil {
		return PostCreateRequest{}, &FieldError{
			Code:   "malformed_body",
			Fields: map[string]string{"_": err.Error()},
		}
	}
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &rawMap); err != nil {
		return PostCreateRequest{}, &FieldError{
			Code:   "malformed_body",
			Fields: map[string]string{"_": err.Error()},
		}
	}
	rejected := map[string]string{}
	for _, k := range rejectedPostFields {
		if _, present := rawMap[k]; present {
			rejected[k] = "not writable in v1"
		}
	}
	if len(rejected) > 0 {
		return PostCreateRequest{}, &FieldError{
			Code:   "unexpected_field",
			Fields: rejected,
		}
	}
	out := PostCreateRequest{}
	strict := json.NewDecoder(bytes.NewReader(raw))
	strict.DisallowUnknownFields()
	if err := strict.Decode(&out); err != nil {
		return PostCreateRequest{}, &FieldError{
			Code:   "unexpected_field",
			Fields: map[string]string{"_": err.Error()},
		}
	}
	return out, nil
}

// ValidatePostCreate enforces lexicon rules: non-empty text, ≤ 2000
// graphemes (approximated by rune count, matching profile_request),
// image blob shape, and AT-URI parseability on reply/quote pointers.
func ValidatePostCreate(req PostCreateRequest) error {
	return ValidatePostCreateWithLimits(req, DefaultMediaLimits())
}

// ValidatePostCreateWithLimits enforces lexicon rules and deployment media limits.
func ValidatePostCreateWithLimits(req PostCreateRequest, limits MediaLimits) error {
	limits = normalizeMediaLimits(limits)
	fields := map[string]string{}
	if req.Text == "" {
		fields["text"] = "must not be empty"
	} else if utf8.RuneCountInString(req.Text) > 2000 {
		fields["text"] = "exceeds 2000 graphemes"
	}
	if err := languages.ValidatePostTags(req.Langs); err != nil {
		fields["langs"] = "must contain no more than three distinct valid language tags"
	}
	if req.Reply != nil {
		validateStrongRef(fields, "reply.root", req.Reply.Root)
		validateStrongRef(fields, "reply.parent", req.Reply.Parent)
	}
	if req.Embed != nil && req.Embed.Quote != nil {
		validateStrongRef(fields, "embed.quote", *req.Embed.Quote)
	}
	if req.Embed != nil && req.Embed.External != nil {
		validateExternalEmbed(fields, *req.Embed.External)
		if req.Embed.Quote != nil {
			fields["embed"] = "quote and external embeds are mutually exclusive"
		}
		if len(req.Images) > 0 {
			fields["embed.external"] = "external embeds cannot coexist with images"
		}
		if req.Project != nil {
			fields["embed.external"] = "project posts cannot contain external embeds"
		}
	}
	if len(req.Images) > limits.MaxPostImages {
		fields["images"] = fmt.Sprintf("exceeds maximum of %d entries", limits.MaxPostImages)
	}
	for i, img := range req.Images {
		prefix := fmt.Sprintf("images[%d]", i)
		if len(img.Image) == 0 {
			fields[prefix+".image"] = "must not be empty"
		} else {
			validatePostImageBlob(fields, prefix+".image", img.Image)
		}
		if img.AspectRatio != nil {
			if img.AspectRatio.Width <= 0 {
				fields[prefix+".aspectRatio.width"] = "must be a positive integer"
			}
			if img.AspectRatio.Height <= 0 {
				fields[prefix+".aspectRatio.height"] = "must be a positive integer"
			}
		}
	}
	if req.Project != nil {
		craftType := strings.TrimSpace(req.Project.Common.CraftType)
		if craftType == "" {
			fields["project.common.craftType"] = "must not be empty"
		} else if !IsSupportedProjectCraftType(craftType) {
			fields["project.common.craftType"] = "must be a supported craft type"
		}
		if len(req.Project.Common.Materials) > maxProjectMaterials {
			fields["project.common.materials"] = fmt.Sprintf("exceeds maximum of %d entries", maxProjectMaterials)
		}
		for i, material := range req.Project.Common.Materials {
			key := fmt.Sprintf("project.common.materials[%d].text", i)
			text := strings.TrimSpace(material.Text)
			if text == "" {
				fields[key] = "must not be empty"
			} else if utf8.RuneCountInString(text) > maxProjectMaterialGraphemes {
				fields[key] = fmt.Sprintf("exceeds %d graphemes", maxProjectMaterialGraphemes)
			} else if strings.ContainsAny(text, "\r\n") {
				fields[key] = "must be a single line"
			}
		}
		if req.Reply != nil {
			fields["project"] = "project posts must be standalone and cannot be replies"
		}
		if req.Embed != nil && req.Embed.Quote != nil {
			fields["project"] = "project posts must be standalone and cannot quote another post"
		}
	}
	if len(fields) > 0 {
		return &FieldError{Code: "validation_failed", Fields: fields}
	}
	return nil
}

func validateExternalEmbed(fields map[string]string, external ExternalEmbedRequest) {
	uri, err := url.Parse(external.URI)
	if err != nil || uri.Scheme != "http" && uri.Scheme != "https" || uri.Hostname() == "" || uri.User != nil || len(external.URI) > maxLinkPreviewURLBytes {
		fields["embed.external.uri"] = "must be an HTTP or HTTPS URL"
	}
	validateExternalMetadata(fields, "embed.external.title", external.Title, 200, 2_000, true)
	validateExternalMetadata(fields, "embed.external.description", external.Description, 300, 3_000, false)
	if external.Thumb != nil {
		validatePostImageBlob(fields, "embed.external.thumb", external.Thumb)
		for key := range external.Thumb {
			if key != "$type" && key != "ref" && key != "mimeType" && key != "size" {
				fields["embed.external.thumb."+key] = "is not allowed"
			}
		}
		if blobType, ok := external.Thumb["$type"].(string); !ok || blobType != "blob" {
			fields["embed.external.thumb.$type"] = "must be blob"
		}
		if ref, ok := external.Thumb["ref"].(map[string]any); ok {
			for key := range ref {
				if key != "$link" {
					fields["embed.external.thumb.ref."+key] = "is not allowed"
				}
			}
			if link, ok := ref["$link"].(string); ok {
				parsed, err := cid.Parse(link)
				if err != nil || parsed.String() != link {
					fields["embed.external.thumb.ref.$link"] = "must be a canonical CID"
				}
			}
		}
		mimeType, _ := external.Thumb["mimeType"].(string)
		switch mimeType {
		case "image/jpeg", "image/png", "image/webp":
		default:
			fields["embed.external.thumb.mimeType"] = "must be image/jpeg, image/png, or image/webp"
		}
		if size, ok := positiveIntegerValue(external.Thumb["size"]); !ok || size > 1_000_000 {
			fields["embed.external.thumb.size"] = "must be a positive integer no greater than 1000000"
		}
	}
}

func validateExternalMetadata(fields map[string]string, key, value string, maxGraphemes, maxBytes int, required bool) {
	if !utf8.ValidString(value) || len(value) > maxBytes || uniseg.GraphemeClusterCount(value) > maxGraphemes {
		fields[key] = "exceeds metadata limits"
	} else if required && strings.TrimSpace(value) == "" {
		fields[key] = "must not be empty"
	}
}

func validatePostImageBlob(fields map[string]string, prefix string, blob map[string]any) {
	refRaw, ok := blob["ref"]
	if !ok {
		fields[prefix+".ref"] = "must include ref"
	} else {
		ref, ok := refRaw.(map[string]any)
		if !ok {
			fields[prefix+".ref"] = "must include ref"
		} else {
			linkRaw, ok := ref["$link"]
			link, linkIsString := linkRaw.(string)
			if !ok || !linkIsString || strings.TrimSpace(link) == "" {
				fields[prefix+".ref.$link"] = "must not be empty"
			}
		}
	}
	mime, ok := blob["mimeType"].(string)
	if !ok || strings.TrimSpace(mime) == "" {
		fields[prefix+".mimeType"] = "must not be empty"
	}
	if !isPositiveIntegerValue(blob["size"]) {
		fields[prefix+".size"] = "must be a positive integer"
	}
}

func isPositiveIntegerValue(v any) bool {
	_, ok := positiveIntegerValue(v)
	return ok
}

func positiveIntegerValue(v any) (uint64, bool) {
	switch n := v.(type) {
	case int:
		return uint64(n), n > 0
	case int8:
		return uint64(n), n > 0
	case int16:
		return uint64(n), n > 0
	case int32:
		return uint64(n), n > 0
	case int64:
		return uint64(n), n > 0
	case uint:
		return uint64(n), n > 0
	case uint8:
		return uint64(n), n > 0
	case uint16:
		return uint64(n), n > 0
	case uint32:
		return uint64(n), n > 0
	case uint64:
		return n, n > 0
	case float32:
		return uint64(n), n > 0 && n == float32(uint64(n))
	case float64:
		return uint64(n), n > 0 && n == float64(uint64(n))
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return uint64(i), i > 0
		}
		return 0, false
	default:
		return 0, false
	}
}

func validateStrongRef(fields map[string]string, prefix string, ref StrongRef) {
	if _, err := syntax.ParseATURI(ref.URI); err != nil {
		fields[prefix+".uri"] = fmt.Sprintf("not a valid AT-URI: %s", err)
	}
	if ref.CID == "" {
		fields[prefix+".cid"] = "must not be empty"
	}
}
