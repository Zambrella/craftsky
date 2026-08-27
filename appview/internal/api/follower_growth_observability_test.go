package api_test

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

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/followergrowth"
	"social.craftsky/appview/internal/middleware"
)

func TestFollowerGrowthAPIObservabilityDoesNotLeakPrivateData(t *testing.T) {
	const (
		owner  = "did:plc:private-growth-owner"
		handle = "private-growth-owner.example"
	)
	reader := failingFollowerGrowthReader{err: errors.New(
		"snapshot row for " + owner + " (" + handle + ") has follower_count=9876543",
	)}
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	handler := middleware.Logging(logger)(api.GetFollowerGrowthHandler(reader, time.Now))

	request := httptest.NewRequest(http.MethodGet, "/v1/profiles/me/follower-growth?period=30d", nil)
	request = request.WithContext(middleware.WithDID(request.Context(), syntax.DID(owner)))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", response.Code, response.Body.String())
	}
	diagnostics := logs.String()
	for _, forbidden := range []string{owner, handle, "follower_count", "9876543", "snapshot row"} {
		if strings.Contains(diagnostics, forbidden) {
			t.Fatalf("API telemetry leaked %q: %s", forbidden, diagnostics)
		}
	}
}

type failingFollowerGrowthReader struct {
	err error
}

func (r failingFollowerGrowthReader) Read(
	context.Context,
	syntax.DID,
	followergrowth.DateRange,
) (followergrowth.History, error) {
	return followergrowth.History{}, r.err
}
