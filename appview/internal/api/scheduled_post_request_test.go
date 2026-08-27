package api

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/scheduledposts"
)

func TestScheduledExternalSourceURIUsesNormalizedComposerIdentity(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	body := func(sourceURI string) string {
		return `{"operationId":"00000000-0000-4000-8000-000000000620","scheduledAt":"2026-08-02T12:05:00Z","payload":{"kind":"standard","text":"link","external":{"sourceUri":"` + sourceURI + `","uri":"https://final.example/pattern","title":"Pattern","description":"Description"}}}`
	}

	t.Run("canonicalizes scheme host and default port", func(t *testing.T) {
		request, err := decodeScheduledPostCreate(strings.NewReader(body("HTTPS://SOURCE.EXAMPLE:443/pattern?size=2")))
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		_, _, encoded, err := validateScheduledPostRequest(now, request, DefaultMediaLimits())
		if err != nil {
			t.Fatalf("validate: %v", err)
		}
		payload, err := scheduledposts.DecodePayload(encoded)
		if err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if got, want := payload.External.SourceURI, "https://source.example/pattern?size=2"; got != want {
			t.Fatalf("sourceUri = %q, want %q", got, want)
		}
	})

	for _, test := range []struct {
		name      string
		sourceURI string
	}{
		{name: "empty", sourceURI: ""},
		{name: "userinfo", sourceURI: "https://member:secret@source.example/pattern"},
		{name: "fragment", sourceURI: "https://source.example/pattern#section"},
		{name: "bare domain", sourceURI: "source.example/pattern"},
		{name: "unsupported scheme", sourceURI: "file:///private/pattern"},
		{name: "noncanonical escaped host", sourceURI: "https://source%2Eexample/pattern"},
	} {
		t.Run(test.name, func(t *testing.T) {
			request, err := decodeScheduledPostCreate(strings.NewReader(body(test.sourceURI)))
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			_, _, _, err = validateScheduledPostRequest(now, request, DefaultMediaLimits())
			var fieldError *FieldError
			if !errors.As(err, &fieldError) || fieldError.Fields["payload.external.sourceUri"] == "" {
				t.Fatalf("error = %v, want sourceUri FieldError", err)
			}
		})
	}
}

func TestScheduledExternalThumbnailPassesPostBlobShapeValidation(t *testing.T) {
	t.Parallel()

	const body = `{"operationId":"00000000-0000-4000-8000-000000000621","scheduledAt":"2026-08-02T12:05:00Z","payload":{"kind":"standard","text":"link","external":{"sourceUri":"https://source.example/pattern","uri":"https://final.example/pattern","title":"Pattern","description":"Description","thumbMediaId":"55555555-5555-4555-8555-555555555555"}}}`
	request, err := decodeScheduledPostCreate(strings.NewReader(body))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	_, mediaIDs, _, err := validateScheduledPostRequest(
		time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC),
		request,
		DefaultMediaLimits(),
	)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if got, want := mediaIDs, []uuid.UUID{uuid.MustParse("55555555-5555-4555-8555-555555555555")}; !slices.Equal(got, want) {
		t.Fatalf("media IDs = %v, want %v", got, want)
	}
}

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
			name: "project external",
			body: `{"operationId":"00000000-0000-4000-8000-000000000616","scheduledAt":"2026-08-02T12:05:00Z","payload":{"kind":"project","text":"project","external":{"sourceUri":"https://source.example/pattern","uri":"https://final.example/pattern","title":"Pattern","description":"Description"}}}`,
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

func TestScheduledPostCreateRejectsOversizedExternalThumbnailMedia(t *testing.T) {
	store, pool := newScheduledPostAPITestStore(t)
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	mediaID := uuid.MustParse("55555555-5555-4555-8555-555555555555")
	insertScheduledPostRequestMedia(t, pool, mediaID, "image/png", 1_000_001, now)
	handler := CreateScheduledPostHandler(store, DefaultMediaLimits(), func() time.Time { return now }, slog.New(slog.NewTextHandler(io.Discard, nil)))

	response := serveScheduledPostRequest(t, handler, http.MethodPost, "/v1/scheduled-posts", scheduledExternalCreateBody("00000000-0000-4000-8000-000000000617", mediaID), "did:plc:alice")

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	assertScheduledPostError(t, response, "scheduled_media_invalid")
}

func TestScheduledPostCreateAcceptsExactLimitExternalThumbnailMedia(t *testing.T) {
	store, pool := newScheduledPostAPITestStore(t)
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	mediaID := uuid.MustParse("55555555-5555-4555-8555-555555555556")
	insertScheduledPostRequestMedia(t, pool, mediaID, "image/webp", 1_000_000, now)
	handler := CreateScheduledPostHandler(store, DefaultMediaLimits(), func() time.Time { return now }, slog.New(slog.NewTextHandler(io.Discard, nil)))

	response := serveScheduledPostRequest(t, handler, http.MethodPost, "/v1/scheduled-posts", scheduledExternalCreateBody("00000000-0000-4000-8000-000000000618", mediaID), "did:plc:alice")
	if response.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestScheduledPostCreateRejectsUnsupportedExternalThumbnailMIME(t *testing.T) {
	store, pool := newScheduledPostAPITestStore(t)
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	mediaID := uuid.MustParse("55555555-5555-4555-8555-555555555557")
	insertScheduledPostRequestMedia(t, pool, mediaID, "image/gif", 100, now)
	handler := CreateScheduledPostHandler(store, DefaultMediaLimits(), func() time.Time { return now }, slog.New(slog.NewTextHandler(io.Discard, nil)))

	response := serveScheduledPostRequest(t, handler, http.MethodPost, "/v1/scheduled-posts", scheduledExternalCreateBody("00000000-0000-4000-8000-000000000619", mediaID), "did:plc:alice")
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	assertScheduledPostError(t, response, "scheduled_media_invalid")
}

func scheduledExternalCreateBody(operationID string, mediaID uuid.UUID) string {
	return `{
		"operationId":"` + operationID + `",
		"scheduledAt":"2026-08-02T12:05:00Z",
		"payload":{
			"kind":"standard",
			"text":"Use https://source.example/pattern ",
			"external":{
				"sourceUri":"https://source.example/pattern",
				"uri":"https://final.example/pattern",
				"title":"Pattern",
				"description":"Description",
				"thumbMediaId":"` + mediaID.String() + `"
			}
		}
	}`
}

func insertScheduledPostRequestMedia(t *testing.T, pool *pgxpool.Pool, mediaID uuid.UUID, mimeType string, size int64, now time.Time) {
	t.Helper()
	objectKey, attemptID, err := scheduledposts.NewGenerationObjectKey("did:plc:alice", 1, mediaID)
	if err != nil {
		t.Fatalf("derive media key: %v", err)
	}
	if _, err := pool.Exec(t.Context(), `
		INSERT INTO scheduled_post_object_attempts (
			upload_attempt_id,media_id,owner_did,owner_generation,
			upload_generation,object_key,request_fingerprint,remote_outcome,
			remote_started_at,remote_deadline,dispatched_at,completed_at,
			created_at,updated_at
		) VALUES ($1,$2,'did:plc:alice',1,1,$3,decode(repeat('03',32),'hex'),
			'accepted',$4,$5,$4,$4,$4,$4)
	`, attemptID, mediaID, objectKey, now, now.Add(time.Minute)); err != nil {
		t.Fatalf("insert media attempt: %v", err)
	}
	if _, err := pool.Exec(t.Context(), `
		INSERT INTO scheduled_post_media (
			id,owner_did,owner_generation,upload_generation,upload_attempt_id,
			object_key,state,mime_type,size_bytes,sha256,blob_cid,
			unclaimed_expires_at
		) VALUES ($1,'did:plc:alice',1,1,$2,$3,'ready',$4,$5,
			decode(repeat('03',32),'hex'),'bafk-test',$6)
	`, mediaID, attemptID, objectKey, mimeType, size, now.Add(time.Hour)); err != nil {
		t.Fatalf("insert scheduled media: %v", err)
	}
}
