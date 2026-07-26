package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/relationships"
)

type relationshipListFake struct {
	kind         string
	owner        syntax.DID
	limit        int
	afterCreated time.Time
	afterSubject syntax.DID
	items        []relationships.ListItem
	more         bool
	err          error
}

func (f *relationshipListFake) ListMutes(_ context.Context, owner syntax.DID, limit int, after time.Time, subject syntax.DID) ([]relationships.ListItem, bool, error) {
	f.kind, f.owner, f.limit, f.afterCreated, f.afterSubject = "mutes", owner, limit, after, subject
	return f.items, f.more, f.err
}

func (f *relationshipListFake) ListBlocks(_ context.Context, owner syntax.DID, limit int, after time.Time, subject syntax.DID) ([]relationships.ListItem, bool, error) {
	f.kind, f.owner, f.limit, f.afterCreated, f.afterSubject = "blocks", owner, limit, after, subject
	return f.items, f.more, f.err
}

func TestRelationshipListHandlersReturnOwnerScopedCurrentSummaries(t *testing.T) {
	alice := syntax.DID("did:plc:alice")
	bob := syntax.DID("did:plc:bob")
	createdAt := time.Date(2026, 7, 19, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		wantKind     string
		wantMuted    bool
		wantBlocking bool
		newHandler   func(*relationshipListFake) http.Handler
	}{
		{name: "mutes", wantKind: "mutes", wantMuted: true, newHandler: func(store *relationshipListFake) http.Handler {
			return api.ListMutedProfilesHandler(store, fakeResolver{handleFor: "bob.current.example"}, nilLogger())
		}},
		{name: "blocks", wantKind: "blocks", wantBlocking: true, newHandler: func(store *relationshipListFake) http.Handler {
			return api.ListBlockedProfilesHandler(store, fakeResolver{handleFor: "bob.current.example"}, nilLogger())
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &relationshipListFake{
				items: []relationships.ListItem{{SubjectDID: bob, CreatedAt: createdAt}},
				more:  true,
			}
			h := tt.newHandler(store)
			req := httptest.NewRequest(http.MethodGet, "/v1/profiles/me/"+tt.wantKind+"?limit=1", nil)
			req = req.WithContext(middleware.WithDID(req.Context(), alice))
			rr := httptest.NewRecorder()

			h.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, body=%s", rr.Code, rr.Body.String())
			}
			if store.kind != tt.wantKind || store.owner != alice || store.limit != 1 {
				t.Fatalf("list call = %s owner=%s limit=%d", store.kind, store.owner, store.limit)
			}
			var body struct {
				Items []struct {
					DID       string `json:"did"`
					Handle    string `json:"handle"`
					Muted     bool   `json:"muted"`
					Blocking  bool   `json:"blocking"`
					BlockedBy bool   `json:"blockedBy"`
				} `json:"items"`
				Cursor string `json:"cursor"`
			}
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if len(body.Items) != 1 {
				t.Fatalf("items = %+v, want one", body.Items)
			}
			item := body.Items[0]
			if item.DID != bob.String() || item.Handle != "bob.current.example" || item.Muted != tt.wantMuted || item.Blocking != tt.wantBlocking || item.BlockedBy {
				t.Fatalf("item = %+v", item)
			}
			if body.Cursor == "" {
				t.Fatal("cursor is empty while store reports another page")
			}
		})
	}
}

func TestRelationshipListHandlerRejectsInvalidRequestBeforeStore(t *testing.T) {
	store := &relationshipListFake{}
	h := api.ListMutedProfilesHandler(store, fakeResolver{}, nilLogger())
	req := httptest.NewRequest(http.MethodGet, "/v1/profiles/me/mutes?limit=101", nil)
	req = req.WithContext(middleware.WithDID(req.Context(), syntax.DID("did:plc:alice")))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rr.Code, rr.Body.String())
	}
	if store.kind != "" {
		t.Fatalf("invalid request called store kind %q", store.kind)
	}
}
