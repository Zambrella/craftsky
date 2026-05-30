// appview/internal/api/moderation_request_test.go
package api_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
)

func TestDecodeSyntheticModerationRequest_ValidTrustedPostAndAccount(t *testing.T) {
	t.Parallel()
	cfg := api.ModerationRequestConfig{DefaultSourceDID: "did:plc:labeler", TrustedSourceDIDs: []string{"did:plc:labeler", "did:plc:ozone"}}

	post, err := api.DecodeSyntheticModerationRequest(strings.NewReader(`{
		"sourceDid":"did:plc:ozone",
		"subject":{"type":"post","did":"did:plc:bob","rkey":"3lf2abc"},
		"value":"hide",
		"action":"apply",
		"internalReason":"private reason",
		"expiresAt":"2026-06-01T00:00:00Z"
	}`), cfg)
	if err != nil {
		t.Fatalf("DecodeSyntheticModerationRequest post: %v", err)
	}
	if post.SourceDID != "did:plc:ozone" || post.SubjectType != api.ModerationSubjectPost || post.SubjectDID != "did:plc:bob" || post.SubjectRkey == nil || *post.SubjectRkey != "3lf2abc" || post.Value != api.ModerationValueHide || post.Action != api.ModerationActionApply {
		t.Fatalf("post request = %+v", post)
	}
	if post.SubjectURI == nil || *post.SubjectURI != "at://did:plc:bob/social.craftsky.feed.post/3lf2abc" {
		t.Fatalf("post SubjectURI = %v", post.SubjectURI)
	}
	if post.ExpiresAt == nil || !post.ExpiresAt.Equal(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("ExpiresAt = %v", post.ExpiresAt)
	}

	account, err := api.DecodeSyntheticModerationRequest(strings.NewReader(`{
		"subject":{"type":"account","did":"did:plc:bob"},
		"value":"warn",
		"action":"negate"
	}`), cfg)
	if err != nil {
		t.Fatalf("DecodeSyntheticModerationRequest account: %v", err)
	}
	if account.SourceDID != "did:plc:labeler" || account.SubjectType != api.ModerationSubjectAccount || account.SubjectDID != "did:plc:bob" || account.Value != api.ModerationValueWarn || account.Action != api.ModerationActionNegate {
		t.Fatalf("account request = %+v", account)
	}
}

func TestDecodeSyntheticModerationRequest_RejectsUntrustedSourceAndBatchPayload(t *testing.T) {
	t.Parallel()
	cfg := api.ModerationRequestConfig{DefaultSourceDID: "did:plc:labeler", TrustedSourceDIDs: []string{"did:plc:labeler"}}

	_, err := api.DecodeSyntheticModerationRequest(strings.NewReader(`{
		"sourceDid":"did:plc:attacker",
		"subject":{"type":"account","did":"did:plc:bob"},
		"value":"hide",
		"action":"apply"
	}`), cfg)
	var fe *api.FieldError
	if !errors.As(err, &fe) || fe.Code != "untrusted_moderation_source" {
		t.Fatalf("want untrusted_moderation_source, got %v", err)
	}

	_, err = api.DecodeSyntheticModerationRequest(strings.NewReader(`[]`), cfg)
	if !errors.As(err, &fe) || fe.Code != "malformed_body" {
		t.Fatalf("want malformed_body for batch payload, got %v", err)
	}
}
