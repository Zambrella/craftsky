package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/subscriptions"
)

type redactorAccessStub struct{ access subscriptions.SelfAccess }

func (s redactorAccessStub) SelfAccess(context.Context, syntax.DID, time.Time) (subscriptions.SelfAccess, error) {
	return s.access, nil
}

func TestSavedFolderIdentityIsRedactedAcrossFreePostResponses(t *testing.T) {
	h := SavedFolderRedactor(redactorAccessStub{access: subscriptions.SelfAccess{EffectiveTier: subscriptions.TierFree}}).Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"viewerSavedFolderId":"private-folder","quote":{"viewerSavedFolderId":"another-private-folder"}}]}`))
	}))
	r := httptest.NewRequest(http.MethodGet, "/v1/feed/timeline", nil)
	r = r.WithContext(middleware.WithDID(r.Context(), "did:plc:viewer"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK || strings.Contains(w.Body.String(), "private-folder") || !strings.Contains(w.Body.String(), `"viewerSavedFolderId":null`) {
		t.Fatalf("free response status/body=%d/%s", w.Code, w.Body.String())
	}
}
