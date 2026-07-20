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
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

type fakeFacetMentionResolver struct {
	row api.IdentityCacheRow
	err error
}

func (f fakeFacetMentionResolver) ResolveMention(context.Context, syntax.DID, syntax.Handle, time.Time) (api.IdentityCacheRow, error) {
	return f.row, f.err
}

func TestResolveFacetMentionHandlerSuccessAndMentionNotFound(t *testing.T) {
	t.Parallel()
	t.Run("valid Craftsky handle returns minimal object", func(t *testing.T) {
		t.Parallel()
		h := api.ResolveFacetMentionHandler(fakeFacetMentionResolver{row: api.IdentityCacheRow{
			DID:    syntax.DID("did:plc:alice"),
			Handle: syntax.Handle("alice.craftsky.social"),
		}}, nilLogger())
		req := httptest.NewRequest(http.MethodGet, "/v1/facets/mentions/resolve?handle=alice.craftsky.social", nil)
		req = req.WithContext(middleware.WithDID(req.Context(), syntax.DID("did:plc:viewer")))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
		}
		var body api.FacetMentionResolveResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("body JSON: %v", err)
		}
		if body.DID.String() != "did:plc:alice" || body.Handle.String() != "alice.craftsky.social" || !body.IsCraftskyProfile {
			t.Fatalf("body = %+v", body)
		}
	})

	t.Run("non-Craftsky or missing maps to mention_not_found", func(t *testing.T) {
		t.Parallel()
		h := api.ResolveFacetMentionHandler(fakeFacetMentionResolver{err: api.ErrMentionNotFound}, nilLogger())
		req := httptest.NewRequest(http.MethodGet, "/v1/facets/mentions/resolve?handle=mallory.example", nil)
		req = req.WithContext(middleware.WithDID(req.Context(), syntax.DID("did:plc:viewer")))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", rr.Code)
		}
		var body envelope.Error
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("body JSON: %v", err)
		}
		if body.Error != "mention_not_found" {
			t.Fatalf("error = %q, want mention_not_found", body.Error)
		}
	})

	t.Run("blocked pair maps to interaction_blocked", func(t *testing.T) {
		t.Parallel()
		h := api.ResolveFacetMentionHandler(fakeFacetMentionResolver{err: api.ErrInteractionBlocked}, nilLogger())
		req := httptest.NewRequest(http.MethodGet, "/v1/facets/mentions/resolve?handle=bob.example", nil)
		req = req.WithContext(middleware.WithDID(req.Context(), syntax.DID("did:plc:viewer")))
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
		}
		var body envelope.Error
		_ = json.Unmarshal(rr.Body.Bytes(), &body)
		if body.Error != "interaction_blocked" {
			t.Fatalf("error = %q, want interaction_blocked", body.Error)
		}
	})
}
