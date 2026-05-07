// appview/internal/api/post_request.go
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
// createdAt is server-stamped; project, images are not writable in this
// pass and are explicitly rejected.
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
}

// rejectedPostFields enumerates wire fields that are NOT writable here.
var rejectedPostFields = []string{"images", "project", "createdAt"}

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
	if len(fields) > 0 {
		return &FieldError{Code: "validation_failed", Fields: fields}
	}
	return nil
}

func validateStrongRef(fields map[string]string, prefix string, ref StrongRef) {
	if _, err := syntax.ParseATURI(ref.URI); err != nil {
		fields[prefix+".uri"] = fmt.Sprintf("not a valid AT-URI: %s", err)
	}
	if ref.CID == "" {
		fields[prefix+".cid"] = "must not be empty"
	}
}
