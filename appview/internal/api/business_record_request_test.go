package api

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api/envelope"
)

func TestBusinessRecordIfMatch(t *testing.T) {
	const current = "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"

	request := httptest.NewRequest("PUT", "/v1/profiles/me/business", nil)
	request.Header.Set("If-Match", current)
	got, err := ParseBusinessIfMatch(request)
	if err != nil || got != syntax.CID(current) {
		t.Fatalf("ParseBusinessIfMatch(current) = (%q, %v)", got, err)
	}
	if err := RequireCurrentCID(got, syntax.CID(current)); err != nil {
		t.Fatalf("RequireCurrentCID(current): %v", err)
	}
	request.Header.Set("If-Match", "*")
	create, err := ParseBusinessIfMatch(request)
	if err != nil || create != "*" {
		t.Fatalf("ParseBusinessIfMatch(*) = (%q, %v)", create, err)
	}
	if err := RequireBusinessRecordPrecondition(create, "", false); err != nil {
		t.Fatalf("RequireBusinessRecordPrecondition(absent): %v", err)
	}
	if err := RequireBusinessRecordPrecondition(create, syntax.CID(current), true); !errors.Is(err, ErrPDSRecordConflict) {
		t.Fatalf("RequireBusinessRecordPrecondition(existing) error = %v", err)
	}
	if err := RequireBusinessRecordPrecondition(syntax.CID(current), "", false); !errors.Is(err, ErrPDSRecordConflict) {
		t.Fatalf("RequireBusinessRecordPrecondition(missing existing record) error = %v", err)
	}

	for name, value := range map[string]string{
		"missing":   "",
		"malformed": "not-a-cid",
		"quoted":    `"` + current + `"`,
	} {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest("PUT", "/v1/profiles/me/business", nil)
			request.Header.Set("If-Match", value)
			if _, err := ParseBusinessIfMatch(request); !errors.Is(err, ErrPDSRecordConflict) {
				t.Fatalf("ParseBusinessIfMatch(%q) error = %v, want ErrPDSRecordConflict", value, err)
			}
		})
	}
	if err := RequireCurrentCID(syntax.CID(current), syntax.CID("bafyreidifferent")); !errors.Is(err, ErrPDSRecordConflict) {
		t.Fatalf("RequireCurrentCID(stale) error = %v, want ErrPDSRecordConflict", err)
	}

	recorder := httptest.NewRecorder()
	WritePDSRecordConflict(recorder, "request-123")
	if recorder.Code != 409 {
		t.Fatalf("conflict status = %d, want 409", recorder.Code)
	}
	var body envelope.Error
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode conflict envelope: %v", err)
	}
	if body.Error != "pds_record_conflict" || body.Message == "" || body.RequestID != "request-123" {
		t.Fatalf("conflict envelope = %+v", body)
	}
}
