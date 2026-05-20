// appview/internal/api/post_request.go
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/bluesky-social/indigo/atproto/syntax"
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
	Quote *StrongRef `json:"quote,omitempty"`
}

// PostCreateRequest is the decoded body of POST /v1/posts.
// createdAt is server-stamped; project is not writable in this pass.
type PostCreateRequest struct {
	Text string `json:"text"`
	// Facets is opaque raw JSON deliberately. The lexicon's
	// app.bsky.richtext.facet shape (including a possibly-present "$type"
	// discriminator on the outer object and the inner union variants) is
	// pass-through to the PDS, which validates it. The tag-extraction
	// path in the create handler does its own non-strict decode for the
	// synthetic response.
	Facets json.RawMessage `json:"facets,omitempty"`
	Reply  *ReplyRef       `json:"reply,omitempty"`
	Embed  *EmbedRequest   `json:"embed,omitempty"`
	Images []PostImage     `json:"images,omitempty"`
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
var rejectedPostFields = []string{"project", "createdAt"}

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
// and AT-URI parseability on reply/quote pointers.
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
	if req.Reply != nil {
		validateStrongRef(fields, "reply.root", req.Reply.Root)
		validateStrongRef(fields, "reply.parent", req.Reply.Parent)
	}
	if req.Embed != nil && req.Embed.Quote != nil {
		validateStrongRef(fields, "embed.quote", *req.Embed.Quote)
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
		if strings.TrimSpace(img.Alt) == "" {
			fields[prefix+".alt"] = "must not be empty"
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
	if len(fields) > 0 {
		return &FieldError{Code: "validation_failed", Fields: fields}
	}
	return nil
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
	switch n := v.(type) {
	case int:
		return n > 0
	case int8:
		return n > 0
	case int16:
		return n > 0
	case int32:
		return n > 0
	case int64:
		return n > 0
	case uint:
		return n > 0
	case uint8:
		return n > 0
	case uint16:
		return n > 0
	case uint32:
		return n > 0
	case uint64:
		return n > 0
	case float32:
		return n > 0 && n == float32(int64(n))
	case float64:
		return n > 0 && n == float64(int64(n))
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return i > 0
		}
		return false
	default:
		return false
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
