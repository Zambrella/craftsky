package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/instagram"
	"social.craftsky/appview/internal/middleware"
)

func TestInstagramSuggestionHandlersKeepMatchesPrivateAndUseCapturedSession(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 14, 16, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:suggestion-owner")
	target := syntax.DID("did:plc:suggestion-target")
	id := uuid.MustParse("80000000-0000-4000-8000-000000000001")
	service := &stubInstagramSuggestionService{
		items: []instagram.PrivateSuggestion{{
			ID: id, ImporterDID: owner, TargetDID: target,
			ImporterGeneration: 2, TargetGeneration: 3,
			State: instagram.SuggestionPending, CreatedAt: now,
		}},
		next: &instagram.SuggestionCursor{
			CreatedAt: now, ID: id,
		},
		accepted: instagram.PrivateSuggestion{ID: id, State: instagram.SuggestionFollowed},
	}
	displayName := "Target Maker"
	profiles := &suggestionProfileReader{row: &api.ProfileRow{
		DID: target.String(), DisplayName: &displayName, Crafts: []string{},
		CreatedAt: now, IsCraftskyProfile: true,
	}}
	resolver := fakeResolver{handleFor: "target.craftsky.social"}

	list := api.ListInstagramSuggestionsHandler(service, profiles, resolver, nilLogger())
	listRequest := httptest.NewRequest(http.MethodGet, "/v1/migrations/instagram/suggestions?limit=20", nil)
	listRequest = listRequest.WithContext(middleware.WithDID(listRequest.Context(), owner))
	listResponse := httptest.NewRecorder()
	list.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d body=%s", listResponse.Code, listResponse.Body.String())
	}
	var page map[string]any
	if err := json.Unmarshal(listResponse.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	items, ok := page["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("list items = %#v", page["items"])
	}
	item := items[0].(map[string]any)
	identity := item["target"].(map[string]any)
	if item["suggestionId"] != id.String() || identity["did"] != target.String() ||
		identity["handle"] != "target.craftsky.social" || identity["displayName"] != displayName {
		t.Fatalf("suggestion response = %#v", item)
	}
	for _, forbidden := range []string{"importId", "importedUsername", "reason", "ownerGeneration", "sessionId"} {
		if _, exists := item[forbidden]; exists {
			t.Errorf("private suggestion leaked %q", forbidden)
		}
	}
	if cursor, _ := page["cursor"].(string); cursor == "" {
		t.Fatal("opaque next cursor is missing")
	}

	accept := api.AcceptInstagramSuggestionHandler(service, nilLogger())
	acceptRequest := httptest.NewRequest(http.MethodPost, "/v1/migrations/instagram/suggestions/"+id.String()+"/accept", nil)
	acceptRequest.SetPathValue("suggestionId", id.String())
	acceptContext := middleware.WithDID(acceptRequest.Context(), owner)
	acceptContext = middleware.WithOAuthSessionID(acceptContext, "captured-session")
	acceptRequest = acceptRequest.WithContext(acceptContext)
	acceptResponse := httptest.NewRecorder()
	accept.ServeHTTP(acceptResponse, acceptRequest)
	if acceptResponse.Code != http.StatusOK {
		t.Fatalf("accept status = %d body=%s", acceptResponse.Code, acceptResponse.Body.String())
	}
	if service.acceptOwner != owner || service.acceptID != id || service.acceptSession != "captured-session" {
		t.Fatalf("accept call owner/id/session = %s/%s/%q", service.acceptOwner, service.acceptID, service.acceptSession)
	}

	dismiss := api.DismissInstagramSuggestionHandler(service, nilLogger())
	dismissRequest := httptest.NewRequest(http.MethodDelete, "/v1/migrations/instagram/suggestions/not-a-uuid", nil)
	dismissRequest.SetPathValue("suggestionId", "not-a-uuid")
	dismissRequest = dismissRequest.WithContext(middleware.WithDID(dismissRequest.Context(), owner))
	dismissResponse := httptest.NewRecorder()
	dismiss.ServeHTTP(dismissResponse, dismissRequest)
	if dismissResponse.Code != http.StatusNoContent || service.dismissCalls != 0 {
		t.Fatalf("foreign/absent dismiss status=%d calls=%d", dismissResponse.Code, service.dismissCalls)
	}
}

func TestAcceptInstagramSuggestionHidesForeignID(t *testing.T) {
	t.Parallel()
	service := &stubInstagramSuggestionService{acceptErr: instagram.ErrInstagramResourceNotFound}
	id := uuid.MustParse("81000000-0000-4000-8000-000000000001")
	handler := api.AcceptInstagramSuggestionHandler(service, nilLogger())
	request := httptest.NewRequest(http.MethodPost, "/v1/migrations/instagram/suggestions/"+id.String()+"/accept", nil)
	request.SetPathValue("suggestionId", id.String())
	ctx := middleware.WithDID(request.Context(), "did:plc:foreign-caller")
	ctx = middleware.WithOAuthSessionID(ctx, "foreign-session")
	request = request.WithContext(ctx)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != "suggestion_not_found" {
		t.Fatalf("error = %#v", body)
	}
}

type stubInstagramSuggestionService struct {
	items         []instagram.PrivateSuggestion
	next          *instagram.SuggestionCursor
	listErr       error
	accepted      instagram.PrivateSuggestion
	acceptErr     error
	acceptOwner   syntax.DID
	acceptID      uuid.UUID
	acceptSession string
	dismissCalls  int
	dismissErr    error
}

func (service *stubInstagramSuggestionService) ListPending(
	_ context.Context,
	_ syntax.DID,
	_ int,
	_ *instagram.SuggestionCursor,
) ([]instagram.PrivateSuggestion, *instagram.SuggestionCursor, error) {
	return service.items, service.next, service.listErr
}

func (service *stubInstagramSuggestionService) Accept(
	_ context.Context,
	owner syntax.DID,
	id uuid.UUID,
	session string,
) (instagram.PrivateSuggestion, error) {
	service.acceptOwner = owner
	service.acceptID = id
	service.acceptSession = session
	return service.accepted, service.acceptErr
}

func (service *stubInstagramSuggestionService) Dismiss(
	_ context.Context,
	_ syntax.DID,
	_ uuid.UUID,
) (bool, error) {
	service.dismissCalls++
	return service.dismissErr == nil, service.dismissErr
}

type suggestionProfileReader struct {
	row *api.ProfileRow
	err error
}

func (reader *suggestionProfileReader) Read(context.Context, string, string) (*api.ProfileRow, error) {
	return reader.row, reader.err
}

var _ api.InstagramSuggestionService = (*stubInstagramSuggestionService)(nil)
var _ api.ProfileReader = (*suggestionProfileReader)(nil)
