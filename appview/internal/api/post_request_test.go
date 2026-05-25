// appview/internal/api/post_request_test.go
package api_test

import (
	"errors"
	"strings"
	"testing"

	"social.craftsky/appview/internal/api"
)

func TestDecodePostCreate_HappyPathTextOnly(t *testing.T) {
	t.Parallel()
	body := strings.NewReader(`{"text":"hello"}`)
	req, err := api.DecodePostCreate(body)
	if err != nil {
		t.Fatalf("DecodePostCreate: %v", err)
	}
	if req.Text != "hello" {
		t.Errorf("text = %q", req.Text)
	}
}

func TestDecodePostCreate_AcceptsImagesField(t *testing.T) {
	t.Parallel()
	body := strings.NewReader(`{"text":"hi","images":[{"image":{"$type":"blob","ref":{"$link":"bafk1"},"mimeType":"image/jpeg","size":1},"alt":"alt"}]}`)
	req, err := api.DecodePostCreate(body)
	if err != nil {
		t.Fatalf("DecodePostCreate: %v", err)
	}
	if len(req.Images) != 1 {
		t.Fatalf("images len = %d, want 1", len(req.Images))
	}
}

func TestDecodePostCreate_RejectsProjectField(t *testing.T) {
	t.Parallel()
	body := strings.NewReader(`{"text":"hi","project":{}}`)
	_, err := api.DecodePostCreate(body)
	var fe *api.FieldError
	if !errors.As(err, &fe) || fe.Code != "unexpected_field" {
		t.Fatalf("want unexpected_field, got %v", err)
	}
	if _, ok := fe.Fields["project"]; !ok {
		t.Errorf("expected project in fields")
	}
}

func TestDecodePostCreate_RejectsCreatedAtField(t *testing.T) {
	t.Parallel()
	body := strings.NewReader(`{"text":"hi","createdAt":"2026-05-04T12:00:00Z"}`)
	_, err := api.DecodePostCreate(body)
	var fe *api.FieldError
	if !errors.As(err, &fe) || fe.Code != "unexpected_field" {
		t.Fatalf("want unexpected_field, got %v", err)
	}
}

func TestDecodePostCreate_MalformedJSON(t *testing.T) {
	t.Parallel()
	body := strings.NewReader(`{not json`)
	_, err := api.DecodePostCreate(body)
	var fe *api.FieldError
	if !errors.As(err, &fe) || fe.Code != "malformed_body" {
		t.Fatalf("want malformed_body, got %v", err)
	}
}

func TestValidatePostCreate_RejectsEmptyText(t *testing.T) {
	t.Parallel()
	err := api.ValidatePostCreate(api.PostCreateRequest{Text: ""})
	var fe *api.FieldError
	if !errors.As(err, &fe) || fe.Code != "validation_failed" {
		t.Fatalf("want validation_failed, got %v", err)
	}
	if _, ok := fe.Fields["text"]; !ok {
		t.Errorf("expected text in fields")
	}
}

func TestValidatePostCreate_RejectsTextOver2000Graphemes(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("a", 2001)
	err := api.ValidatePostCreate(api.PostCreateRequest{Text: long})
	var fe *api.FieldError
	if !errors.As(err, &fe) || fe.Code != "validation_failed" {
		t.Fatalf("want validation_failed, got %v", err)
	}
}

func TestValidatePostCreate_AcceptsValidReply(t *testing.T) {
	t.Parallel()
	err := api.ValidatePostCreate(api.PostCreateRequest{
		Text: "hi",
		Reply: &api.ReplyRef{
			Root:   api.StrongRef{URI: "at://did:plc:abc/social.craftsky.feed.post/rk1", CID: "bafy1"},
			Parent: api.StrongRef{URI: "at://did:plc:abc/social.craftsky.feed.post/rk2", CID: "bafy2"},
		},
	})
	if err != nil {
		t.Fatalf("want nil, got %v", err)
	}
}

func TestValidatePostCreate_RejectsReplyWithBadURI(t *testing.T) {
	t.Parallel()
	err := api.ValidatePostCreate(api.PostCreateRequest{
		Text: "hi",
		Reply: &api.ReplyRef{
			Root:   api.StrongRef{URI: "not-a-uri", CID: "bafy1"},
			Parent: api.StrongRef{URI: "at://did:plc:abc/social.craftsky.feed.post/rk2", CID: "bafy2"},
		},
	})
	var fe *api.FieldError
	if !errors.As(err, &fe) {
		t.Fatalf("want *FieldError, got %v", err)
	}
	if _, ok := fe.Fields["reply.root.uri"]; !ok {
		t.Errorf("expected reply.root.uri in fields, got %v", fe.Fields)
	}
}

func TestValidatePostCreate_RejectsReplyWithEmptyCID(t *testing.T) {
	t.Parallel()
	err := api.ValidatePostCreate(api.PostCreateRequest{
		Text: "hi",
		Reply: &api.ReplyRef{
			Root:   api.StrongRef{URI: "at://did:plc:abc/social.craftsky.feed.post/rk1", CID: ""},
			Parent: api.StrongRef{URI: "at://did:plc:abc/social.craftsky.feed.post/rk2", CID: "bafy2"},
		},
	})
	var fe *api.FieldError
	if !errors.As(err, &fe) || fe.Code != "validation_failed" {
		t.Fatalf("want validation_failed, got %v", err)
	}
	if _, ok := fe.Fields["reply.root.cid"]; !ok {
		t.Errorf("expected reply.root.cid in fields, got %v", fe.Fields)
	}
}

func TestDecodeAndValidatePostCreate_AcceptsValidImagesPayload(t *testing.T) {
	t.Parallel()
	body := strings.NewReader(`{
		"text":"hi",
		"images":[
			{
				"image":{"$type":"blob","ref":{"$link":"bafkimage"},"mimeType":"image/jpeg","size":253496},
				"alt":"project photo",
				"aspectRatio":{"width":919,"height":2000}
			}
		]
	}`)
	req, err := api.DecodePostCreate(body)
	if err != nil {
		t.Fatalf("DecodePostCreate: %v", err)
	}
	if err := api.ValidatePostCreate(req); err != nil {
		t.Fatalf("ValidatePostCreate: %v", err)
	}
}

func TestDecodeAndValidatePostCreate_AcceptsImagesWithoutAltText(t *testing.T) {
	t.Parallel()
	body := strings.NewReader(`{
		"text":"hi",
		"images":[
			{"image":{"$type":"blob","ref":{"$link":"bafkimage"},"mimeType":"image/jpeg","size":253496}},
			{"image":{"$type":"blob","ref":{"$link":"bafkimage2"},"mimeType":"image/png","size":123},"alt":""}
		]
	}`)
	req, err := api.DecodePostCreate(body)
	if err != nil {
		t.Fatalf("DecodePostCreate: %v", err)
	}
	if err := api.ValidatePostCreate(req); err != nil {
		t.Fatalf("ValidatePostCreate: %v", err)
	}
}

func TestValidatePostCreate_RejectsMoreThanFourImages(t *testing.T) {
	t.Parallel()
	body := strings.NewReader(`{
		"text":"hi",
		"images":[
			{"image":{"$type":"blob","ref":{"$link":"bafk1"},"mimeType":"image/jpeg","size":1},"alt":"1"},
			{"image":{"$type":"blob","ref":{"$link":"bafk2"},"mimeType":"image/jpeg","size":1},"alt":"2"},
			{"image":{"$type":"blob","ref":{"$link":"bafk3"},"mimeType":"image/jpeg","size":1},"alt":"3"},
			{"image":{"$type":"blob","ref":{"$link":"bafk4"},"mimeType":"image/jpeg","size":1},"alt":"4"},
			{"image":{"$type":"blob","ref":{"$link":"bafk5"},"mimeType":"image/jpeg","size":1},"alt":"5"}
		]
	}`)
	req, err := api.DecodePostCreate(body)
	if err != nil {
		t.Fatalf("DecodePostCreate: %v", err)
	}
	err = api.ValidatePostCreate(req)
	var fe *api.FieldError
	if !errors.As(err, &fe) || fe.Code != "validation_failed" {
		t.Fatalf("want validation_failed, got %v", err)
	}
	if _, ok := fe.Fields["images"]; !ok {
		t.Fatalf("fields=%v, want images", fe.Fields)
	}
}

func TestValidatePostCreate_UsesConfiguredImageCountLimit(t *testing.T) {
	t.Parallel()
	body := strings.NewReader(`{
		"text":"hi",
		"images":[
			{"image":{"$type":"blob","ref":{"$link":"bafk1"},"mimeType":"image/jpeg","size":1},"alt":"1"},
			{"image":{"$type":"blob","ref":{"$link":"bafk2"},"mimeType":"image/jpeg","size":1},"alt":"2"}
		]
	}`)
	req, err := api.DecodePostCreate(body)
	if err != nil {
		t.Fatalf("DecodePostCreate: %v", err)
	}
	err = api.ValidatePostCreateWithLimits(req, api.MediaLimits{MaxPostImages: 1, MaxImageUploadBytes: api.DefaultMaxImageUploadBytes})
	var fe *api.FieldError
	if !errors.As(err, &fe) || fe.Code != "validation_failed" {
		t.Fatalf("want validation_failed, got %v", err)
	}
	if got := fe.Fields["images"]; got != "exceeds maximum of 1 entries" {
		t.Fatalf("images error = %q", got)
	}
}

func TestValidatePostCreate_RejectsMissingBlobOrInvalidAspectRatio(t *testing.T) {
	t.Parallel()
	body := strings.NewReader(`{
		"text":"hi",
		"images":[
			{"alt":"missing blob"},
			{"image":{"$type":"blob","ref":{"$link":"bafk2"},"mimeType":"image/jpeg","size":1},"alt":"ok","aspectRatio":{"width":0,"height":10}},
			{"image":{"$type":"blob","ref":{"$link":"bafk3"},"mimeType":"image/jpeg","size":1},"alt":"ok","aspectRatio":{"width":10,"height":-1}}
		]
	}`)
	req, err := api.DecodePostCreate(body)
	if err != nil {
		t.Fatalf("DecodePostCreate: %v", err)
	}
	err = api.ValidatePostCreate(req)
	var fe *api.FieldError
	if !errors.As(err, &fe) || fe.Code != "validation_failed" {
		t.Fatalf("want validation_failed, got %v", err)
	}
	for _, key := range []string{"images[0].image", "images[1].aspectRatio.width", "images[2].aspectRatio.height"} {
		if _, ok := fe.Fields[key]; !ok {
			t.Fatalf("fields=%v, want key %q", fe.Fields, key)
		}
	}
}

func TestValidatePostCreate_RejectsImageWithMissingBlobMetadata(t *testing.T) {
	t.Parallel()
	body := strings.NewReader(`{
		"text":"hi",
		"images":[
			{"image":{"$type":"blob","mimeType":"image/jpeg","size":1},"alt":"ok"},
			{"image":{"$type":"blob","ref":{},"mimeType":"image/jpeg","size":1},"alt":"ok"},
			{"image":{"$type":"blob","ref":{"$link":"bafk3"},"mimeType":"","size":1},"alt":"ok"},
			{"image":{"$type":"blob","ref":{"$link":"bafk4"},"mimeType":"image/jpeg","size":0},"alt":"ok"}
		]
	}`)
	req, err := api.DecodePostCreate(body)
	if err != nil {
		t.Fatalf("DecodePostCreate: %v", err)
	}
	err = api.ValidatePostCreate(req)
	var fe *api.FieldError
	if !errors.As(err, &fe) || fe.Code != "validation_failed" {
		t.Fatalf("want validation_failed, got %v", err)
	}
	for _, key := range []string{"images[0].image.ref", "images[1].image.ref.$link", "images[2].image.mimeType", "images[3].image.size"} {
		if _, ok := fe.Fields[key]; !ok {
			t.Fatalf("fields=%v, want key %q", fe.Fields, key)
		}
	}
}

func TestDecodePostCreate_RejectsVideoField(t *testing.T) {
	t.Parallel()
	body := strings.NewReader(`{"text":"hi","video":{"blob":"x"}}`)
	_, err := api.DecodePostCreate(body)
	var fe *api.FieldError
	if !errors.As(err, &fe) {
		t.Fatalf("want *FieldError, got %v", err)
	}
	if fe.Code != "unexpected_field" {
		t.Fatalf("code = %q, want unexpected_field", fe.Code)
	}
}
