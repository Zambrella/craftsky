package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/getsentry/sentry-go"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/testdb"
)

type privateSentinelHandleResolver struct {
	err error
}

func (r privateSentinelHandleResolver) ResolveHandle(context.Context, syntax.DID) (syntax.Handle, error) {
	return "", r.err
}

func (privateSentinelHandleResolver) ResolveDID(context.Context, syntax.Handle) (syntax.DID, error) {
	return "", errors.New("not used")
}

func TestSavedPostDiagnosticsRedactPrivateStateAndClassifyIdentityFailure(t *testing.T) {
	const (
		ownerSentinel  = "did:plc:private-owner-sentinel"
		targetSentinel = "did:plc:private-target-sentinel"
		folderSentinel = "private-folder-id-and-name-sentinel"
	)
	uriSentinel := syntax.ATURI("at://" + targetSentinel + "/social.craftsky.feed.post/private-uri-sentinel")
	resolverErr := errors.New(ownerSentinel + " " + uriSentinel.String() + " " + folderSentinel)
	refs := &fakeSavedPostRefStore{refs: []api.SavedPostRef{{
		PostURI: uriSentinel,
		SavedAt: time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC),
	}}}
	hydrator := &fakeSavedPostHydrator{
		rows: map[syntax.ATURI]*api.PostRow{
			uriSentinel: {URI: uriSentinel.String(), DID: targetSentinel},
		},
	}
	service := api.NewSavedPostService(refs, hydrator, privateSentinelHandleResolver{err: resolverErr})

	_, err := service.ListSavedPosts(context.Background(), syntax.DID(ownerSentinel), api.SavedPostListFilter{
		Scope: api.SavedPostScopeAll,
		Sort:  api.SavedPostSortNewest,
		Limit: 50,
	})
	if !errors.Is(err, api.ErrSavedPostIdentityUnavailable) {
		t.Fatalf("ListSavedPosts error = %v, want identity unavailable", err)
	}
	for _, sentinel := range []string{ownerSentinel, targetSentinel, uriSentinel.String(), folderSentinel} {
		if strings.Contains(err.Error(), sentinel) {
			t.Fatalf("service error leaked private sentinel %q: %v", sentinel, err)
		}
	}

	recorder := observability.NewInMemoryMetricRecorder()
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{
		Env:              "test",
		MetricRecorder:   recorder,
		SentryDSN:        "https://public@example.invalid/1",
		SentryTransport:  transport,
		TracingEnabled:   true,
		TracesSampleRate: 1,
	})
	mux := http.NewServeMux()
	mux.Handle("GET /v1/saved-posts", middleware.HTTPInFlight(observer)(api.ListSavedPostsHandler(service)))
	handler := middleware.HTTPMetrics(observer)(mux)
	req := httptest.NewRequest(http.MethodGet, "/v1/saved-posts", nil)
	req = req.WithContext(middleware.WithDID(req.Context(), syntax.DID(ownerSentinel)))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusBadGateway || !strings.Contains(response.Body.String(), "identity_unavailable") {
		t.Fatalf("identity failure response = %d/%s, want 502 identity_unavailable", response.Code, response.Body.String())
	}

	diagnosticText := response.Body.String() + fmt.Sprint(recorder.Calls())
	for _, sentinel := range []string{ownerSentinel, targetSentinel, uriSentinel.String(), folderSentinel} {
		if strings.Contains(diagnosticText, sentinel) {
			t.Fatalf("response or metrics leaked private sentinel %q: %s", sentinel, diagnosticText)
		}
	}
	var sawBoundedHTTPResult bool
	for _, call := range recorder.Calls() {
		if err := observability.ValidateMetricCall(call); err != nil {
			t.Fatalf("metric call failed validation: %v; call=%#v", err, call)
		}
		if call.Name == "craftsky_appview_http_requests_total" &&
			call.Attributes["method"] == http.MethodGet &&
			call.Attributes["route_pattern"] == "/v1/saved-posts" &&
			call.Attributes["status_class"] == "5xx" {
			sawBoundedHTTPResult = true
		}
	}
	if !sawBoundedHTTPResult {
		t.Fatalf("missing bounded saved-post HTTP result metric: %#v", recorder.Calls())
	}

	successService := api.NewSavedPostService(
		&fakeSavedPostRefStore{},
		&fakeSavedPostHydrator{},
		privateSentinelHandleResolver{},
	)
	successMux := http.NewServeMux()
	successMux.Handle("GET /v1/saved-posts", middleware.HTTPInFlight(observer)(api.ListSavedPostsHandler(successService)))
	successReq := httptest.NewRequest(http.MethodGet, "/v1/saved-posts", nil)
	successReq = successReq.WithContext(middleware.WithDID(successReq.Context(), syntax.DID(ownerSentinel)))
	successResponse := httptest.NewRecorder()
	middleware.HTTPMetrics(observer)(successMux).ServeHTTP(successResponse, successReq)
	if successResponse.Code != http.StatusOK {
		t.Fatalf("successful saved-list response = %d/%s", successResponse.Code, successResponse.Body.String())
	}

	var sawBoundedHTTPSuccess bool
	for _, call := range recorder.Calls() {
		if call.Name == "craftsky_appview_http_requests_total" &&
			call.Attributes["method"] == http.MethodGet &&
			call.Attributes["route_pattern"] == "/v1/saved-posts" &&
			call.Attributes["status_class"] == "2xx" {
			sawBoundedHTTPSuccess = true
		}
	}
	if !sawBoundedHTTPSuccess {
		t.Fatalf("missing bounded saved-post HTTP success metric: %#v", recorder.Calls())
	}
	if !observer.Flush(time.Second) {
		t.Fatal("flush saved-post telemetry returned false")
	}
	telemetryText := fmt.Sprint(transport.Events()) + fmt.Sprint(recorder.Calls())
	for _, sentinel := range []string{ownerSentinel, targetSentinel, uriSentinel.String(), folderSentinel} {
		if strings.Contains(telemetryText, sentinel) {
			t.Fatalf("captured metrics/traces/errors leaked private sentinel %q: %s", sentinel, telemetryText)
		}
	}
}

func TestSavedPostMutationIsPrivateAndAuthorSeesNoSignal(t *testing.T) {
	pool := testdb.WithSchema(t, postStoreDDL)
	if _, err := pool.Exec(context.Background(), `
		CREATE TABLE saved_post_folders (
			id UUID NOT NULL PRIMARY KEY,
			owner_did TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL,
			UNIQUE (owner_did, id)
		)
	`); err != nil {
		t.Fatalf("create saved folder table: %v", err)
	}
	for _, did := range []string{"did:plc:alice", "did:plc:bob"} {
		seedMember(t, pool, did)
	}
	createdAt := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	postURI := seedPost(t, pool, "did:plc:bob", "private-save-target", "target", createdAt)
	postStore := api.NewPostStore(pool)
	savedStore := api.NewSavedPostStore(pool, api.SavedPostStoreOptions{Now: func() time.Time { return createdAt }})

	before, err := postStore.EngagementSummaries(context.Background(), "did:plc:bob", []string{postURI})
	if err != nil {
		t.Fatalf("author summary before save: %v", err)
	}
	result, err := savedStore.Save(context.Background(), syntax.DID("did:plc:alice"), syntax.ATURI(postURI), api.FolderAssignment{})
	if err != nil || !result.Created {
		t.Fatalf("private save result = %+v, err %v", result, err)
	}
	authorAfter, err := postStore.EngagementSummaries(context.Background(), "did:plc:bob", []string{postURI})
	if err != nil {
		t.Fatalf("author summary after save: %v", err)
	}
	aliceAfter, err := postStore.EngagementSummaries(context.Background(), "did:plc:alice", []string{postURI})
	if err != nil {
		t.Fatalf("owner summary after save: %v", err)
	}
	if before[postURI] != authorAfter[postURI] || authorAfter[postURI].ViewerHasSaved || authorAfter[postURI].ViewerSavedFolderID != nil {
		t.Fatalf("author received a save signal: before=%+v after=%+v", before[postURI], authorAfter[postURI])
	}
	if !aliceAfter[postURI].ViewerHasSaved || aliceAfter[postURI].ViewerSavedFolderID != nil {
		t.Fatalf("Alice private viewer state = %+v, want unfiled saved", aliceAfter[postURI])
	}

	folder, err := savedStore.CreateFolder(context.Background(), syntax.DID("did:plc:alice"), "Private folder")
	if err != nil {
		t.Fatalf("create private folder: %v", err)
	}
	if _, err := savedStore.Save(
		context.Background(),
		syntax.DID("did:plc:alice"),
		syntax.ATURI(postURI),
		api.FolderAssignment{Present: true, ID: &folder.ID},
	); err != nil {
		t.Fatalf("move private save: %v", err)
	}
	if err := savedStore.DeleteFolder(
		context.Background(),
		syntax.DID("did:plc:alice"),
		folder.ID,
		api.SavedPostFolderRemoveSaves,
	); err != nil {
		t.Fatalf("delete private folder and saves: %v", err)
	}
	authorAfterDelete, err := postStore.EngagementSummaries(context.Background(), "did:plc:bob", []string{postURI})
	if err != nil {
		t.Fatalf("author summary after private delete: %v", err)
	}
	if authorAfterDelete[postURI] != before[postURI] {
		t.Fatalf("private delete changed author/public summary: before=%+v after=%+v", before[postURI], authorAfterDelete[postURI])
	}
	if _, err := savedStore.ReadState(context.Background(), syntax.DID("did:plc:alice"), syntax.ATURI(postURI)); !errors.Is(err, api.ErrSavedPostNotFound) {
		t.Fatalf("Alice state after private delete = %v, want not found", err)
	}
}
