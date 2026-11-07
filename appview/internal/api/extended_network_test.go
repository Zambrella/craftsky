package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/languages"
)

type fakeExtendedNetworkStore struct {
	fakeTimelineStore
	called bool
}

func (s *fakeExtendedNetworkStore) ListExtendedNetworkWithLanguages(ctx context.Context, viewerDID string, langs []string, limit int, cursor string) ([]*api.TimelineFeedItemRow, string, error) {
	s.called = true
	return s.ListTimelineWithLanguages(ctx, viewerDID, langs, limit, cursor)
}

func TestExtendedNetworkHandlerHydratesAttributionAndUsesAccountLanguages(t *testing.T) {
	row := testPostRow("did:plc:bob", "post", "Hello", time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC))
	item := authoredTimelineItem(row)
	display := "Alice"
	item.FollowedBy = &api.TimelineFollowedByRow{DID: "did:plc:alice", DisplayName: &display}
	store := &fakeExtendedNetworkStore{fakeTimelineStore: fakeTimelineStore{rows: []*api.TimelineFeedItemRow{item}, cursor: "next", engagement: map[string]api.EngagementSummary{row.URI: {LikeCount: 3}}}}
	prefs := &fakeLanguagePreferenceReader{preferences: languages.Preferences{PrimaryLanguage: "fr", ContentLanguages: []string{"en", "de"}}}
	handler := api.ListExtendedNetworkHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{"did:plc:bob": "bob.example", "did:plc:alice": "alice.example"}}, nilLogger(), prefs)
	req := authedReq(http.MethodGet, "/v1/feed/extended-network?limit=999&cursor=opaque", "", "did:plc:viewer")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !store.called || store.lastLimit != 50 || store.lastCursor != "opaque" || store.lastViewerDID != "did:plc:viewer" || !slices.Equal(store.lastContentLanguages, []string{"en", "de"}) {
		t.Fatalf("store=%+v", store)
	}
	var page api.TimelinePage
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Cursor != "next" {
		t.Fatalf("page=%+v", page)
	}
	got := page.Items[0]
	if got.FollowedBy == nil || got.FollowedBy.DID != "did:plc:alice" || got.FollowedBy.Handle != "alice.example" || got.FollowedBy.DisplayName == nil || *got.FollowedBy.DisplayName != "Alice" || got.Reason != nil || got.Post.LikeCount != 3 {
		t.Fatalf("item=%+v", got)
	}
}

func TestExtendedNetworkHandlerErrorsAndEmptyShape(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want int
		code string
	}{
		{"empty", nil, http.StatusOK, ""},
		{"cursor", envelope.ErrInvalidCursor, http.StatusBadRequest, "invalid_cursor"},
		{"store", errors.New("database failure"), http.StatusInternalServerError, "internal_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeExtendedNetworkStore{fakeTimelineStore: fakeTimelineStore{err: tc.err}}
			handler := api.ListExtendedNetworkHandler(store, fakeResolver{handleFor: "viewer.example"}, nilLogger())
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, authedReq(http.MethodGet, "/v1/feed/extended-network", "", "did:plc:viewer"))
			if rec.Code != tc.want {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
			if tc.code != "" {
				var env envelope.Error
				if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
					t.Fatal(err)
				}
				if env.Error != tc.code {
					t.Fatalf("envelope=%+v", env)
				}
			} else if rec.Body.String() != "{\"items\":[]}\n" {
				t.Fatalf("empty shape=%s", rec.Body.String())
			}
		})
	}
}
