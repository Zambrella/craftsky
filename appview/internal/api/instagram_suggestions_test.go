package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/instagram"
)

func TestInstagramSuggestionHandlersExactSafeWireContract(t *testing.T) {
	t.Parallel()
	alice := syntax.DID("did:plc:synthetic-alice")
	bob := syntax.DID("did:plc:synthetic-bob")
	id := uuid.MustParse("00000000-0000-0000-0000-000000000261")
	createdAt := time.Date(2026, 7, 19, 20, 0, 0, 0, time.UTC)
	service := &stubInstagramSuggestionService{
		items: []instagram.Suggestion{{
			ID: id, ImporterDID: alice, TargetDID: bob,
			State:     instagram.SuggestionPending,
			Reason:    instagram.SuggestionReasonVerifiedInstagramFollow,
			CreatedAt: createdAt, UpdatedAt: createdAt,
		}},
		next:         &instagram.SuggestionCursor{CreatedAt: createdAt, ID: id},
		accepted:     instagram.Suggestion{ID: id, ImporterDID: alice, TargetDID: bob, State: instagram.SuggestionAccepted, Reason: instagram.SuggestionReasonVerifiedInstagramFollow},
		invokeWriter: true,
	}
	profiles := &stubInstagramSuggestionProfiles{row: &ProfileRow{
		DID: bob.String(), Crafts: []string{}, CreatedAt: createdAt,
		IsCraftskyProfile: true, DisplayName: instagramSuggestionString("Synthetic Bob"),
		AvatarCID: instagramSuggestionString("devmedia:synthetic-avatar"), AvatarMime: instagramSuggestionString("image/png"),
	}}
	resolver := stubInstagramSuggestionResolver{handle: syntax.Handle("bob.synthetic.invalid")}
	pds := &instagramSuggestionPDS{}
	newPDS := func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return pds, nil }
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	listRequest := authenticatedInstagramRequest(http.MethodGet, "/v1/migrations/instagram/suggestions?limit=20", "", alice)
	listResponse := httptest.NewRecorder()
	ListInstagramSuggestionsHandler(service, profiles, resolver, logger).ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listResponse.Code, listResponse.Body.String())
	}
	var page map[string]any
	if err := json.Unmarshal(listResponse.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	items, ok := page["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("page=%#v", page)
	}
	item := items[0].(map[string]any)
	profile := item["profile"].(map[string]any)
	if len(item) != 4 || item["suggestionId"] != id.String() || item["reason"] != "verifiedInstagramFollow" || item["state"] != "pending" || profile["did"] != bob.String() || profile["handle"] != "bob.synthetic.invalid" || profile["displayName"] != "Synthetic Bob" {
		t.Fatalf("item=%#v", item)
	}
	cursor, ok := page["cursor"].(string)
	if !ok || cursor == "" || strings.Contains(cursor, alice.String()) || strings.Contains(cursor, id.String()) {
		t.Fatalf("cursor=%#v", page["cursor"])
	}

	secondRequest := authenticatedInstagramRequest(http.MethodGet, "/v1/migrations/instagram/suggestions?cursor="+cursor, "", alice)
	secondResponse := httptest.NewRecorder()
	ListInstagramSuggestionsHandler(service, profiles, resolver, logger).ServeHTTP(secondResponse, secondRequest)
	if secondResponse.Code != http.StatusOK || service.cursor == nil || service.cursor.ID != id || !service.cursor.CreatedAt.Equal(createdAt) {
		t.Fatalf("second status=%d cursor=%+v body=%s", secondResponse.Code, service.cursor, secondResponse.Body.String())
	}

	acceptRequest := authenticatedInstagramRequest(http.MethodPost, "/v1/migrations/instagram/suggestions/"+id.String()+"/accept", "", alice)
	acceptRequest.SetPathValue("suggestionId", id.String())
	acceptResponse := httptest.NewRecorder()
	AcceptInstagramSuggestionHandler(service, newPDS, logger).ServeHTTP(acceptResponse, acceptRequest)
	if acceptResponse.Code != http.StatusOK {
		t.Fatalf("accept status=%d body=%s", acceptResponse.Code, acceptResponse.Body.String())
	}
	var accepted map[string]any
	if err := json.Unmarshal(acceptResponse.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if len(accepted) != 2 || accepted["suggestionId"] != id.String() || accepted["state"] != "accepted" {
		t.Fatalf("accepted=%#v", accepted)
	}
	if pds.putCollection != "app.bsky.graph.follow" || pds.putRepo != alice || pds.putRkey != "3kapiwriter2z" {
		t.Fatalf("PDS put repo=%s collection=%s rkey=%s", pds.putRepo, pds.putCollection, pds.putRkey)
	}
	record := pds.putRecord.(map[string]any)
	if record["subject"] != bob.String() || record["$type"] != "app.bsky.graph.follow" || record["createdAt"] != createdAt.Format(time.RFC3339) {
		t.Fatalf("follow record=%#v", record)
	}
}

func TestInstagramSuggestionHandlersMapSafeErrorsAndPrivateDelete(t *testing.T) {
	t.Parallel()
	alice := syntax.DID("did:plc:synthetic-alice")
	id := uuid.MustParse("00000000-0000-0000-0000-000000000262")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "not found", err: instagram.ErrInstagramResourceNotFound, status: 404, code: "instagram_suggestion_not_found"},
		{name: "ineligible", err: instagram.ErrInstagramSuggestionIneligible, status: 409, code: "instagram_suggestion_ineligible"},
		{name: "follow unavailable", err: instagram.ErrInstagramFollowWriteUnavailable, status: 503, code: "follow_write_unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &stubInstagramSuggestionService{err: test.err}
			request := authenticatedInstagramRequest(http.MethodPost, "/v1/migrations/instagram/suggestions/"+id.String()+"/accept", "", alice)
			request.SetPathValue("suggestionId", id.String())
			response := httptest.NewRecorder()
			AcceptInstagramSuggestionHandler(service, nil, logger).ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			var body envelope.Error
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || body.Error != test.code {
				t.Fatalf("error=%+v decode=%v", body, err)
			}
		})
	}

	invalidCursor := authenticatedInstagramRequest(http.MethodGet, "/v1/migrations/instagram/suggestions?cursor=invalid!!!", "", alice)
	invalidCursorResponse := httptest.NewRecorder()
	ListInstagramSuggestionsHandler(&stubInstagramSuggestionService{}, &stubInstagramSuggestionProfiles{}, stubInstagramSuggestionResolver{}, logger).ServeHTTP(invalidCursorResponse, invalidCursor)
	if invalidCursorResponse.Code != http.StatusBadRequest {
		t.Fatalf("invalid cursor status=%d", invalidCursorResponse.Code)
	}

	service := &stubInstagramSuggestionService{}
	deleteHandler := DeleteInstagramSuggestionHandler(service, logger)
	for _, rawID := range []string{id.String(), "invalid", ""} {
		request := authenticatedInstagramRequest(http.MethodDelete, "/v1/migrations/instagram/suggestions/"+rawID, "", alice)
		request.SetPathValue("suggestionId", rawID)
		response := httptest.NewRecorder()
		deleteHandler.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
			t.Fatalf("delete %q status=%d body=%q", rawID, response.Code, response.Body.String())
		}
	}
	if len(service.dismissed) != 1 || service.dismissed[0] != id {
		t.Fatalf("dismissed=%v", service.dismissed)
	}
}

type stubInstagramSuggestionService struct {
	items        []instagram.Suggestion
	next         *instagram.SuggestionCursor
	accepted     instagram.Suggestion
	err          error
	cursor       *instagram.SuggestionCursor
	dismissed    []uuid.UUID
	invokeWriter bool
}

func (s *stubInstagramSuggestionService) ListSuggestions(_ context.Context, _ syntax.DID, _ int, cursor *instagram.SuggestionCursor) ([]instagram.Suggestion, *instagram.SuggestionCursor, error) {
	s.cursor = cursor
	return s.items, s.next, s.err
}

func (s *stubInstagramSuggestionService) AcceptSuggestion(ctx context.Context, owner syntax.DID, id uuid.UUID, writer instagram.InstagramFollowWriter) (instagram.Suggestion, error) {
	if s.err != nil {
		return instagram.Suggestion{}, s.err
	}
	if s.invokeWriter {
		if err := writer.PutFollow(ctx, owner, s.accepted.TargetDID, syntax.RecordKey("3kapiwriter2z"), time.Date(2026, 7, 19, 20, 0, 0, 0, time.UTC)); err != nil {
			return instagram.Suggestion{}, err
		}
	}
	return s.accepted, nil
}

func (s *stubInstagramSuggestionService) DismissSuggestion(_ context.Context, _ syntax.DID, id uuid.UUID) error {
	s.dismissed = append(s.dismissed, id)
	return s.err
}

type stubInstagramSuggestionProfiles struct {
	row *ProfileRow
	err error
}

func (s *stubInstagramSuggestionProfiles) Read(context.Context, string, string) (*ProfileRow, error) {
	return s.row, s.err
}

type stubInstagramSuggestionResolver struct {
	handle syntax.Handle
	err    error
}

func (s stubInstagramSuggestionResolver) ResolveHandle(context.Context, syntax.DID) (syntax.Handle, error) {
	return s.handle, s.err
}

func (stubInstagramSuggestionResolver) ResolveDID(context.Context, syntax.Handle) (syntax.DID, error) {
	return "", errors.New("not implemented")
}

type instagramSuggestionPDS struct {
	putRepo       syntax.DID
	putCollection string
	putRkey       string
	putRecord     any
}

func (*instagramSuggestionPDS) GetRecord(context.Context, syntax.DID, string, string, any) (string, error) {
	return "", errors.New("not implemented")
}

func (p *instagramSuggestionPDS) PutRecord(_ context.Context, repo syntax.DID, collection, rkey string, record any) error {
	p.putRepo, p.putCollection, p.putRkey, p.putRecord = repo, collection, rkey, record
	return nil
}

func (*instagramSuggestionPDS) CreateRecord(context.Context, syntax.DID, string, any) (syntax.ATURI, syntax.CID, error) {
	return "", "", errors.New("not implemented")
}

func (*instagramSuggestionPDS) DeleteRecord(context.Context, syntax.DID, string, string) error {
	return errors.New("not implemented")
}

func (*instagramSuggestionPDS) UploadBlob(context.Context, string, []byte) (*auth.UploadedBlob, error) {
	return nil, errors.New("not implemented")
}

func instagramSuggestionString(value string) *string { return &value }

var _ InstagramSuggestionService = (*stubInstagramSuggestionService)(nil)
var _ ProfileReader = (*stubInstagramSuggestionProfiles)(nil)
var _ HandleResolver = stubInstagramSuggestionResolver{}
var _ auth.PDSClient = (*instagramSuggestionPDS)(nil)
