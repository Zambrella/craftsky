package api

import (
	"context"
	"encoding/json"
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
	"social.craftsky/appview/internal/instagram"
)

func TestInstagramImportHandlersExactWireContractAndOpaquePagination(t *testing.T) {
	t.Parallel()
	alice := syntax.DID("did:plc:synthetic-alice")
	id := uuid.MustParse("00000000-0000-0000-0000-000000000231")
	createdAt := time.Date(2026, 7, 19, 15, 0, 0, 0, time.UTC)
	expiresAt := createdAt.AddDate(1, 0, 0)
	item := instagram.GraphImport{
		ID: id, OwnerDID: alice, State: instagram.ImportActive,
		SourceType: instagram.ImportSourceInstagramJSON, RetainUnmatched: true,
		RetentionExpiresAt: &expiresAt, FollowingCount: 1, FollowerCount: 1,
		CreatedAt: createdAt,
	}
	service := &stubInstagramImportService{
		created: instagram.CreateImportResult{
			Import: item, Counts: instagram.ImportCounts{Following: 1, Follower: 1},
			InitialSuggestionCount: 0,
		},
		items:      []instagram.GraphImport{item},
		nextCursor: &instagram.ImportCursor{CreatedAt: createdAt, ID: id},
		item:       item,
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	createRequest := authenticatedInstagramRequest(http.MethodPost, "/v1/migrations/instagram/imports", `{
		"sourceType":"instagramJson",
		"retainUnmatched":true,
		"entries":[
			{"username":" Synthetic.One ","direction":"following"},
			{"username":"synthetic.two","direction":"follower"}
		]
	}`, alice)
	createResponse := httptest.NewRecorder()
	CreateInstagramImportHandler(service, logger).ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", createResponse.Code, createResponse.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if len(created) != 3 || created["initialSuggestionCount"] != float64(0) {
		t.Fatalf("created = %#v", created)
	}
	counts, ok := created["counts"].(map[string]any)
	if !ok || counts["followingCount"] != float64(1) || counts["followerCount"] != float64(1) || len(counts) != 2 {
		t.Fatalf("counts = %#v", created["counts"])
	}
	if len(service.createEntries) != 2 || service.createEntries[0].Username != " Synthetic.One " {
		t.Fatalf("service entries = %+v", service.createEntries)
	}

	listRequest := authenticatedInstagramRequest(http.MethodGet, "/v1/migrations/instagram/imports?limit=20", "", alice)
	listResponse := httptest.NewRecorder()
	ListInstagramImportsHandler(service, logger).ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listResponse.Code, listResponse.Body.String())
	}
	var page map[string]any
	if err := json.Unmarshal(listResponse.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	cursor, ok := page["cursor"].(string)
	if !ok || cursor == "" || strings.Contains(cursor, alice.String()) || strings.Contains(cursor, id.String()) {
		t.Fatalf("cursor is not opaque: %#v", page["cursor"])
	}

	secondRequest := authenticatedInstagramRequest(http.MethodGet, "/v1/migrations/instagram/imports?cursor="+cursor, "", alice)
	secondResponse := httptest.NewRecorder()
	ListInstagramImportsHandler(service, logger).ServeHTTP(secondResponse, secondRequest)
	if secondResponse.Code != http.StatusOK || service.receivedCursor == nil || service.receivedCursor.ID != id || !service.receivedCursor.CreatedAt.Equal(createdAt) {
		t.Fatalf("second status=%d cursor=%+v body=%s", secondResponse.Code, service.receivedCursor, secondResponse.Body.String())
	}

	getRequest := authenticatedInstagramRequest(http.MethodGet, "/v1/migrations/instagram/imports/"+id.String(), "", alice)
	getRequest.SetPathValue("importId", id.String())
	getResponse := httptest.NewRecorder()
	GetInstagramImportHandler(service, logger).ServeHTTP(getResponse, getRequest)
	if getResponse.Code != http.StatusOK {
		t.Fatalf("get status=%d body=%s", getResponse.Code, getResponse.Body.String())
	}
	var detail map[string]any
	if err := json.Unmarshal(getResponse.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if len(detail) != 8 || detail["importId"] != id.String() || detail["state"] != "active" || detail["retentionExpiresAt"] != expiresAt.Format(time.RFC3339) {
		t.Fatalf("detail = %#v", detail)
	}

	patchRequest := authenticatedInstagramRequest(http.MethodPatch, "/v1/migrations/instagram/imports/"+id.String(), `{"retainUnmatched":false,"reactivate":true}`, alice)
	patchRequest.SetPathValue("importId", id.String())
	patchResponse := httptest.NewRecorder()
	PatchInstagramImportHandler(service, logger).ServeHTTP(patchResponse, patchRequest)
	if patchResponse.Code != http.StatusOK || service.retainUnmatched == nil || *service.retainUnmatched || service.reactivate == nil || !*service.reactivate {
		t.Fatalf("patch status=%d retain=%v reactivate=%v body=%s", patchResponse.Code, service.retainUnmatched, service.reactivate, patchResponse.Body.String())
	}
}

func TestInstagramImportHandlersRejectRawFieldsInvalidInputsAndMapSafeErrors(t *testing.T) {
	t.Parallel()
	alice := syntax.DID("did:plc:synthetic-alice")
	id := uuid.MustParse("00000000-0000-0000-0000-000000000232")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name       string
		handler    func(InstagramImportService, *slog.Logger) http.Handler
		method     string
		target     string
		body       string
		pathID     string
		serviceErr error
		status     int
		code       string
	}{
		{name: "raw archive field", handler: CreateInstagramImportHandler, method: http.MethodPost, target: "/v1/migrations/instagram/imports", body: `{"sourceType":"manual","retainUnmatched":false,"entries":[],"rawArchive":"synthetic-private-canary"}`, status: 400, code: "invalid_request"},
		{name: "invalid import", handler: CreateInstagramImportHandler, method: http.MethodPost, target: "/v1/migrations/instagram/imports", body: `{"sourceType":"manual","retainUnmatched":false,"entries":[]}`, serviceErr: instagram.ErrInvalidInstagramImport, status: 422, code: "invalid_instagram_import"},
		{name: "invalid cursor", handler: ListInstagramImportsHandler, method: http.MethodGet, target: "/v1/migrations/instagram/imports?cursor=not-valid!!!", status: 400, code: "invalid_cursor"},
		{name: "foreign get", handler: GetInstagramImportHandler, method: http.MethodGet, target: "/v1/migrations/instagram/imports/" + id.String(), pathID: id.String(), serviceErr: instagram.ErrInstagramResourceNotFound, status: 404, code: "instagram_import_not_found"},
		{name: "empty patch", handler: PatchInstagramImportHandler, method: http.MethodPatch, target: "/v1/migrations/instagram/imports/" + id.String(), pathID: id.String(), body: `{}`, status: 400, code: "invalid_request"},
		{name: "expired patch", handler: PatchInstagramImportHandler, method: http.MethodPatch, target: "/v1/migrations/instagram/imports/" + id.String(), pathID: id.String(), body: `{"reactivate":true}`, serviceErr: instagram.ErrInstagramImportExpired, status: 409, code: "instagram_import_expired"},
		{name: "discarded patch", handler: PatchInstagramImportHandler, method: http.MethodPatch, target: "/v1/migrations/instagram/imports/" + id.String(), pathID: id.String(), body: `{"retainUnmatched":true}`, serviceErr: instagram.ErrUnmatchedDataUnavailable, status: 409, code: "unmatched_data_unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service := &stubInstagramImportService{err: test.serviceErr}
			request := authenticatedInstagramRequest(test.method, test.target, test.body, alice)
			if test.pathID != "" {
				request.SetPathValue("importId", test.pathID)
			}
			response := httptest.NewRecorder()
			test.handler(service, logger).ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status=%d want=%d body=%s", response.Code, test.status, response.Body.String())
			}
			var apiError envelope.Error
			if err := json.Unmarshal(response.Body.Bytes(), &apiError); err != nil {
				t.Fatalf("decode error: %v", err)
			}
			if apiError.Error != test.code || strings.Contains(response.Body.String(), "synthetic-private-canary") {
				t.Fatalf("error response = %s", response.Body.String())
			}
		})
	}
}

func TestInstagramImportDeleteIsOwnerScopedPermanentPrivacyNoOp(t *testing.T) {
	t.Parallel()
	alice := syntax.DID("did:plc:synthetic-alice")
	service := &stubInstagramImportService{}
	handler := DeleteInstagramImportHandler(service, slog.Default())
	for _, rawID := range []string{"00000000-0000-0000-0000-000000000233", "not-a-uuid", ""} {
		request := authenticatedInstagramRequest(http.MethodDelete, "/v1/migrations/instagram/imports/"+rawID, "", alice)
		request.SetPathValue("importId", rawID)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
			t.Fatalf("delete %q status=%d body=%q", rawID, response.Code, response.Body.String())
		}
	}
	if len(service.deleted) != 1 {
		t.Fatalf("delete calls=%d, want one valid ID", len(service.deleted))
	}
}

type stubInstagramImportService struct {
	created         instagram.CreateImportResult
	items           []instagram.GraphImport
	nextCursor      *instagram.ImportCursor
	item            instagram.GraphImport
	err             error
	createEntries   []instagram.ImportEntry
	receivedCursor  *instagram.ImportCursor
	retainUnmatched *bool
	reactivate      *bool
	deleted         []uuid.UUID
}

func (s *stubInstagramImportService) CreateImport(_ context.Context, _ syntax.DID, _ instagram.ImportSourceType, _ bool, entries []instagram.ImportEntry) (instagram.CreateImportResult, error) {
	s.createEntries = entries
	return s.created, s.err
}

func (s *stubInstagramImportService) ListImports(_ context.Context, _ syntax.DID, _ int, cursor *instagram.ImportCursor) ([]instagram.GraphImport, *instagram.ImportCursor, error) {
	s.receivedCursor = cursor
	return s.items, s.nextCursor, s.err
}

func (s *stubInstagramImportService) GetImport(context.Context, syntax.DID, uuid.UUID) (instagram.GraphImport, error) {
	return s.item, s.err
}

func (s *stubInstagramImportService) UpdateImport(_ context.Context, _ syntax.DID, _ uuid.UUID, retainUnmatched, reactivate *bool) (instagram.GraphImport, error) {
	s.retainUnmatched = retainUnmatched
	s.reactivate = reactivate
	return s.item, s.err
}

func (s *stubInstagramImportService) DeleteImport(_ context.Context, _ syntax.DID, id uuid.UUID) error {
	s.deleted = append(s.deleted, id)
	return s.err
}

var _ InstagramImportService = (*stubInstagramImportService)(nil)
