package api

import (
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"
)

func TestScheduledPostCreateRejectsExcludedKindsAndInvalidTimes(t *testing.T) {
	store, pool := newScheduledPostAPITestStore(t)
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	handler := CreateScheduledPostHandler(store, DefaultMediaLimits(), func() time.Time { return now }, slog.New(slog.NewTextHandler(io.Discard, nil)))

	tests := []struct {
		name string
		body string
		code string
	}{
		{
			name: "reply",
			body: `{"operationId":"00000000-0000-4000-8000-000000000611","scheduledAt":"2026-08-02T12:05:00Z","payload":{"kind":"standard","text":"reply","reply":{"root":{"uri":"at://did:plc:bob/social.craftsky.feed.post/root","cid":"bafk-root"},"parent":{"uri":"at://did:plc:bob/social.craftsky.feed.post/parent","cid":"bafk-parent"}}}}`,
			code: "scheduled_post_ineligible",
		},
		{
			name: "quote",
			body: `{"operationId":"00000000-0000-4000-8000-000000000612","scheduledAt":"2026-08-02T12:05:00Z","payload":{"kind":"standard","text":"quote","embed":{"quote":{"uri":"at://did:plc:bob/social.craftsky.feed.post/quoted","cid":"bafk-quote"}}}}`,
			code: "scheduled_post_ineligible",
		},
		{
			name: "less than five minutes",
			body: `{"operationId":"00000000-0000-4000-8000-000000000613","scheduledAt":"2026-08-02T12:04:00Z","payload":{"kind":"standard","text":"too soon"}}`,
			code: "invalid_scheduled_at",
		},
		{
			name: "not a whole minute",
			body: `{"operationId":"00000000-0000-4000-8000-000000000614","scheduledAt":"2026-08-02T12:05:01Z","payload":{"kind":"standard","text":"seconds"}}`,
			code: "invalid_scheduled_at",
		},
		{
			name: "more than 28 days",
			body: `{"operationId":"00000000-0000-4000-8000-000000000615","scheduledAt":"2026-08-30T12:01:00Z","payload":{"kind":"standard","text":"too late"}}`,
			code: "invalid_scheduled_at",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := serveScheduledPostRequest(t, handler, http.MethodPost, "/v1/scheduled-posts", test.body, "did:plc:alice")
			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			assertScheduledPostError(t, response, test.code)
		})
	}

	var count int
	if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM scheduled_posts`).Scan(&count); err != nil {
		t.Fatalf("count rejected schedules: %v", err)
	}
	if count != 0 {
		t.Fatalf("rejected schedule count=%d, want 0", count)
	}
}
