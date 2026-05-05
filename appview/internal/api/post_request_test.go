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

func TestDecodePostCreate_RejectsImagesField(t *testing.T) {
	t.Parallel()
	body := strings.NewReader(`{"text":"hi","images":[]}`)
	_, err := api.DecodePostCreate(body)
	var fe *api.FieldError
	if !errors.As(err, &fe) {
		t.Fatalf("want *FieldError, got %v", err)
	}
	if fe.Code != "unexpected_field" {
		t.Errorf("code = %q", fe.Code)
	}
	if _, ok := fe.Fields["images"]; !ok {
		t.Errorf("expected images in fields, got %v", fe.Fields)
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
