// appview/internal/api/notifications_test.go
package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
)

type fakeNotificationStore struct {
	rows         []*api.NotificationRow
	cursor       string
	err          error
	engagement   map[string]api.EngagementSummary
	lastViewer   string
	lastLimit    int
	lastCursor   string
	engagementIn []string
}

func (f *fakeNotificationStore) ListNotifications(_ context.Context, viewerDID string, limit int, cursor string) ([]*api.NotificationRow, string, error) {
	f.lastViewer = viewerDID
	f.lastLimit = limit
	f.lastCursor = cursor
	return f.rows, f.cursor, f.err
}

func (f *fakeNotificationStore) EngagementSummaries(_ context.Context, _ string, postURIs []string) (map[string]api.EngagementSummary, error) {
	f.engagementIn = append([]string(nil), postURIs...)
	out := make(map[string]api.EngagementSummary, len(postURIs))
	for _, uri := range postURIs {
		out[uri] = api.EngagementSummary{}
	}
	for uri, summary := range f.engagement {
		out[uri] = summary
	}
	return out, nil
}

func TestNotificationsHandler_IgnoresUnknownParamsUsesLimitsAndSessionViewer(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantLimit  int
		wantCursor string
	}{
		{name: "default", path: "/v1/notifications?foo=bar&did=did:plc:other", wantLimit: 20},
		{name: "cap", path: "/v1/notifications?limit=999&cursor=opaque&viewerDid=did:plc:other", wantLimit: 50, wantCursor: "opaque"},
		{name: "invalid", path: "/v1/notifications?limit=nope", wantLimit: 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeNotificationStore{}
			handler := api.ListNotificationsHandler(store, fakeResolver{handleFor: "viewer.example"}, nilLogger())
			req := authedReq(http.MethodGet, tt.path, "", "did:plc:viewer")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
			}
			if store.lastViewer != "did:plc:viewer" {
				t.Fatalf("store viewer = %q, want authenticated viewer", store.lastViewer)
			}
			if store.lastLimit != tt.wantLimit || store.lastCursor != tt.wantCursor {
				t.Fatalf("store limit/cursor = %d/%q, want %d/%q", store.lastLimit, store.lastCursor, tt.wantLimit, tt.wantCursor)
			}
		})
	}
}

func TestNotificationsHandler_ReturnsCamelCaseNotificationPage(t *testing.T) {
	subject := testPostRow("did:plc:viewer", "root", "viewer post", time.Date(2026, 5, 28, 17, 0, 0, 0, time.UTC))
	store := &fakeNotificationStore{rows: []*api.NotificationRow{
		{
			Type: api.NotificationTypeLike, URI: "at://did:plc:alice/social.craftsky.feed.like/like1", CID: "bafylike", Rkey: "like1",
			ActorDID: "did:plc:alice", CreatedAt: subject.CreatedAt, IndexedAt: subject.IndexedAt, SubjectPost: subject,
		},
	}, cursor: "next-cursor"}
	handler := api.ListNotificationsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice":  "alice.example",
		"did:plc:viewer": "viewer.example",
	}}, nilLogger())

	req := authedReq(http.MethodGet, "/v1/notifications", "", "did:plc:viewer")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if _, ok := raw["totalCount"]; ok {
		t.Fatalf("body contains totalCount: %s", rec.Body.String())
	}
	var page api.NotificationPage
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("body not notification page: %v", err)
	}
	if page.Cursor != "next-cursor" || len(page.Items) != 1 {
		t.Fatalf("page = %+v, want one item and cursor", page)
	}
	item := page.Items[0]
	if item.Type != api.NotificationTypeLike || item.Actor.Handle != "alice.example" || item.SubjectPost == nil || item.SubjectPost.URI != subject.URI {
		t.Fatalf("item = %+v, want like with actor handle and subject post", item)
	}
}

func TestNotificationsHandler_InvalidCursorUsesStandardErrorEnvelope(t *testing.T) {
	store := &fakeNotificationStore{err: envelope.ErrInvalidCursor}
	handler := api.ListNotificationsHandler(store, fakeResolver{handleFor: "viewer.example"}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/notifications?cursor=bad", "", "did:plc:viewer")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	var env envelope.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil || env.Error != "invalid_cursor" || env.Message == "" {
		t.Fatalf("envelope = %+v err=%v, want invalid_cursor", env, err)
	}
}

func TestNotificationsHandler_StoreFailureUsesServerErrorEnvelope(t *testing.T) {
	store := &fakeNotificationStore{err: errors.New("database exploded")}
	handler := api.ListNotificationsHandler(store, fakeResolver{handleFor: "viewer.example"}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/notifications", "", "did:plc:viewer")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() == "" || errors.Is(store.err, envelope.ErrInvalidCursor) {
		t.Fatalf("unexpected body/error: %s", rec.Body.String())
	}
}
