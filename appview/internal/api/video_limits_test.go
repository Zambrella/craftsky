package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/video"
)

type fakeUploadLimitsService struct {
	limits    video.UploadLimits
	err       error
	owner     syntax.DID
	sessionID string
}

func (f *fakeUploadLimitsService) Get(_ context.Context, owner syntax.DID, sessionID string) (video.UploadLimits, error) {
	f.owner = owner
	f.sessionID = sessionID
	return f.limits, f.err
}

func TestVideoUploadLimitsHandler_ReturnsNormalizedCamelCaseLimits(t *testing.T) {
	t.Parallel()
	videos, bytes := int64(1), int64(299_999_999)
	service := &fakeUploadLimitsService{limits: video.UploadLimits{
		RemainingDailyVideos: &videos, RemainingDailyBytes: &bytes, Reason: "quota_exhausted",
	}}
	handler := api.VideoUploadLimitsHandler(service, nilLogger())
	request := httptest.NewRequest(http.MethodGet, "/v1/blobs/videos/limits", nil)
	ctx := middleware.WithDID(request.Context(), syntax.DID("did:plc:alice"))
	ctx = middleware.WithOAuthSessionID(ctx, "session-alice")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request.WithContext(ctx))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body) != 4 || body["canUpload"] != false || body["remainingDailyVideos"] != float64(1) ||
		body["remainingDailyBytes"] != float64(299_999_999) || body["reason"] != "quota_exhausted" {
		t.Fatalf("response = %#v", body)
	}
	if service.owner != syntax.DID("did:plc:alice") || service.sessionID != "session-alice" {
		t.Fatalf("service owner=%q session=%q", service.owner, service.sessionID)
	}
}

func TestVideoUploadLimitsHandler_UsesStandardUnavailableError(t *testing.T) {
	t.Parallel()
	service := &fakeUploadLimitsService{err: errors.New("limits-token oauth-token provider detail")}
	handler := api.VideoUploadLimitsHandler(service, nilLogger())
	request := httptest.NewRequest(http.MethodGet, "/v1/blobs/videos/limits", nil)
	ctx := middleware.WithDID(request.Context(), syntax.DID("did:plc:alice"))
	ctx = middleware.WithOAuthSessionID(ctx, "session-alice")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request.WithContext(ctx))

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if body := recorder.Body.String(); !strings.Contains(body, `"error":"video_service_unavailable"`) ||
		strings.Contains(body, "limits-token") || strings.Contains(body, "oauth-token") || strings.Contains(body, "provider detail") {
		t.Fatalf("response = %s", body)
	}
}
