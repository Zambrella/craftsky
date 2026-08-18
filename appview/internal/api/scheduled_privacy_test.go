package api

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/scheduledposts"
)

func TestScheduledHandlerCaptureExcludesPrivateCanaries(t *testing.T) {
	canaries := []string{
		"private-scheduled-text-canary",
		"did:plc:private-owner-canary",
		"private-filename-canary.jpg",
		"private-alt-canary",
		"private-facet-canary",
		"private-project-canary",
		"private-token-canary",
		"scheduled-media/private-object-key-canary",
		"https://objects.invalid/private-signed-url-canary?signature=secret",
		`{"private":"provider-response-canary"}`,
	}
	rawDependencyError := errors.New(strings.Join(canaries, " | "))
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	responses := make([]string, 0, 6)

	store, pool := newScheduledPostAPITestStore(t)
	pool.Close()
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	owner := syntax.DID(canaries[1])
	create := CreateScheduledPostHandler(
		store, DefaultMediaLimits(), func() time.Time { return now }, logger,
	)
	created := serveScheduledPostRequest(
		t,
		create,
		http.MethodPost,
		"/v1/scheduled-posts",
		`{"operationId":"00000000-0000-4000-8000-000000000781","scheduledAt":"2026-08-02T12:05:00Z","payload":{"kind":"standard","text":"private-scheduled-text-canary","langs":["en"]}}`,
		owner,
	)
	if created.Code != http.StatusInternalServerError {
		t.Fatalf("closed-store create status=%d body=%s", created.Code, created.Body.String())
	}
	responses = append(responses, created.Body.String())

	id := "00000000-0000-4000-8000-000000000782"
	update := UpdateScheduledPostHandler(
		store, DefaultMediaLimits(), func() time.Time { return now }, logger,
	)
	edited := serveScheduledPostPathRequest(
		t,
		update,
		http.MethodPut,
		id,
		`{"scheduledAt":"2026-08-02T12:05:00Z","payload":{"kind":"standard","text":"private-scheduled-text-canary","langs":["en"]}}`,
		owner,
	)
	if edited.Code != http.StatusInternalServerError {
		t.Fatalf("closed-store update status=%d body=%s", edited.Code, edited.Body.String())
	}
	responses = append(responses, edited.Body.String())

	remove := DeleteScheduledPostHandler(store, func() time.Time { return now }, logger)
	deleted := serveScheduledPostPathRequest(t, remove, http.MethodDelete, id, "", owner)
	if deleted.Code != http.StatusInternalServerError {
		t.Fatalf("closed-store delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}
	responses = append(responses, deleted.Body.String())

	mediaID := uuid.MustParse("00000000-0000-4000-8000-000000000783")
	media := canaryFailingScheduledMediaService{err: rawDependencyError}
	mediaHandlers := []struct {
		method      string
		contentType string
		body        []byte
		handler     http.Handler
	}{
		{
			method: http.MethodPut, contentType: "image/jpeg",
			body: validJPEGBytes(t),
			handler: PutScheduledMediaHandler(
				media, DefaultMediaLimits(), mustTestImageValidator(t), logger,
			),
		},
		{method: http.MethodGet, handler: GetScheduledMediaHandler(media, logger)},
		{
			method:  http.MethodDelete,
			handler: DeleteScheduledMediaHandler(media, func() time.Time { return now }, logger),
		},
	}
	for _, target := range mediaHandlers {
		request := httptest.NewRequest(
			target.method,
			"/v1/scheduled-post-media/"+mediaID.String(),
			bytes.NewReader(target.body),
		)
		request.SetPathValue("mediaId", mediaID.String())
		request.Header.Set("Content-Type", target.contentType)
		request.Header.Set("Authorization", "Bearer "+canaries[6])
		request = request.WithContext(middleware.WithOwnerGeneration(middleware.WithDID(request.Context(), owner), 5))
		response := httptest.NewRecorder()
		target.handler.ServeHTTP(response, request)
		if response.Code != http.StatusInternalServerError {
			t.Fatalf("%s media failure status=%d body=%s", target.method, response.Code, response.Body.String())
		}
		responses = append(responses, response.Body.String())
	}

	captured := logs.String() + strings.Join(responses, "\n")
	for _, want := range []string{
		"scheduled post create failed",
		"scheduled post update failed",
		"scheduled post delete failed",
		"scheduled media upload failed",
		"scheduled media read failed",
		"scheduled media delete failed",
		`"error_class":"unknown"`,
		`"error":"internal_error"`,
	} {
		if !strings.Contains(captured, want) {
			t.Fatalf("capture missing safe marker %q:\n%s", want, captured)
		}
	}
	for _, canary := range canaries {
		if strings.Contains(captured, canary) {
			t.Fatalf("scheduled handler capture leaked %q:\n%s", canary, captured)
		}
	}
}

type canaryFailingScheduledMediaService struct {
	err error
}

func (service canaryFailingScheduledMediaService) Put(
	context.Context,
	scheduledposts.PutPrivateMediaParams,
) (scheduledposts.PrivateMedia, error) {
	return scheduledposts.PrivateMedia{}, service.err
}

func (service canaryFailingScheduledMediaService) Open(
	context.Context,
	syntax.DID,
	uuid.UUID,
) (scheduledposts.OpenedPrivateMedia, error) {
	return scheduledposts.OpenedPrivateMedia{}, service.err
}

func (service canaryFailingScheduledMediaService) Delete(
	context.Context,
	syntax.DID,
	uuid.UUID,
	time.Time,
	...int64,
) error {
	return service.err
}

var _ scheduledMediaService = canaryFailingScheduledMediaService{}
