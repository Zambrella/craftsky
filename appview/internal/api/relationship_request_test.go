package api_test

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
)

func TestParseRelationshipListRequestLimitsAndCursor(t *testing.T) {
	createdAt := time.Date(2026, 7, 19, 12, 34, 56, 789, time.UTC)
	subject := syntax.DID("did:plc:visible-subject")
	cursor, err := api.EncodeRelationshipCursor(createdAt, subject)
	if err != nil {
		t.Fatalf("encode cursor: %v", err)
	}
	if cursor == "" || strings.Contains(cursor, subject.String()) {
		t.Fatalf("cursor = %q, want non-empty opaque value without plaintext DID", cursor)
	}

	tests := []struct {
		name      string
		query     string
		wantLimit int
		wantErr   error
	}{
		{name: "defaults to fifty", query: "", wantLimit: 50},
		{name: "accepts fifty", query: "?limit=50", wantLimit: 50},
		{name: "accepts maximum one hundred", query: "?limit=100", wantLimit: 100},
		{name: "rejects zero", query: "?limit=0", wantErr: api.ErrInvalidRelationshipLimit},
		{name: "rejects above maximum", query: "?limit=101", wantErr: api.ErrInvalidRelationshipLimit},
		{name: "rejects non numeric", query: "?limit=many", wantErr: api.ErrInvalidRelationshipLimit},
		{name: "accepts typed cursor", query: "?limit=1&cursor=" + cursor, wantLimit: 1},
		{name: "rejects malformed cursor", query: "?cursor=bad@@", wantErr: envelope.ErrInvalidCursor},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/v1/profiles/me/mutes"+tt.query, nil)
			got, err := api.ParseRelationshipListRequest(req)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseRelationshipListRequest: %v", err)
			}
			if got.Limit != tt.wantLimit {
				t.Fatalf("limit = %d, want %d", got.Limit, tt.wantLimit)
			}
		})
	}

	decodedAt, decodedSubject, err := api.DecodeRelationshipCursor(cursor)
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	if !decodedAt.Equal(createdAt) || decodedSubject != subject {
		t.Fatalf("decoded cursor = %s/%s, want %s/%s", decodedAt, decodedSubject, createdAt, subject)
	}

	wrongKind, err := envelope.EncodeCursor(map[string]any{
		"kind": "timeline",
		"at":   createdAt.Format(time.RFC3339Nano),
		"did":  subject.String(),
	})
	if err != nil {
		t.Fatalf("encode wrong-kind cursor: %v", err)
	}
	if _, _, err := api.DecodeRelationshipCursor(wrongKind); !errors.Is(err, envelope.ErrInvalidCursor) {
		t.Fatalf("wrong-kind cursor error = %v, want ErrInvalidCursor", err)
	}
}
