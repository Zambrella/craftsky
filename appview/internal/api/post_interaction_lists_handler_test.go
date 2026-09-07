package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/languages"
	"social.craftsky/appview/internal/middleware"
)

type fakePostInteractionListReader struct {
	accountPage api.ProfileAccountPage
	quoteItems  []*api.PostResponse
	quoteCursor string
	targetErr   error
	listErr     error

	viewer           syntax.DID
	owner            syntax.DID
	rkey             syntax.RecordKey
	kind             api.PostInteractionKind
	limit            int
	cursor           string
	contentLanguages []string
	resolver         api.HandleResolver
}

func (f *fakePostInteractionListReader) ResolveInteractionTarget(
	_ context.Context,
	viewer syntax.DID,
	owner syntax.DID,
	rkey syntax.RecordKey,
	kind api.PostInteractionKind,
) (*api.PostInteractionTarget, error) {
	f.viewer, f.owner, f.rkey, f.kind = viewer, owner, rkey, kind
	if f.targetErr != nil {
		return nil, f.targetErr
	}
	return &api.PostInteractionTarget{
		URI:    syntax.ATURI("at://" + owner.String() + "/social.craftsky.feed.post/" + rkey.String()),
		Author: owner,
	}, nil
}

func (f *fakePostInteractionListReader) ListPostInteractionAccounts(
	_ context.Context,
	_ syntax.DID,
	_ *api.PostInteractionTarget,
	_ api.PostInteractionKind,
	limit int,
	cursor string,
) (api.ProfileAccountPage, error) {
	f.limit, f.cursor = limit, cursor
	return f.accountPage, f.listErr
}

func (f *fakePostInteractionListReader) ListQuotePostResponses(
	_ context.Context,
	_ syntax.DID,
	_ *api.PostInteractionTarget,
	contentLanguages []string,
	limit int,
	cursor string,
	resolver api.HandleResolver,
) ([]*api.PostResponse, string, error) {
	f.contentLanguages = append([]string(nil), contentLanguages...)
	f.limit, f.cursor, f.resolver = limit, cursor, resolver
	return f.quoteItems, f.quoteCursor, f.listErr
}

type postInteractionLanguageReader struct {
	preferences languages.Preferences
}

func (f postInteractionLanguageReader) Get(_ context.Context, _ syntax.DID) (languages.Preferences, error) {
	return f.preferences, nil
}

// IT-005: FR-008, FR-009; AC-014, AC-015.
func TestPostInteractionListHandlers_SuccessWireContracts(t *testing.T) {
	cursor := "next-page"
	displayName := "Dana"
	account := api.ProfileAccountSummary{
		DID:               syntax.DID("did:plc:dana"),
		Handle:            syntax.Handle("dana.test"),
		DisplayName:       &displayName,
		IsCraftskyProfile: true,
	}
	quote := &api.PostResponse{
		URI:    "at://did:plc:dana/social.craftsky.feed.post/quote",
		CID:    "bafyquote",
		Rkey:   "quote",
		Text:   "quoted",
		Tags:   []string{},
		Langs:  []string{"fr"},
		Author: api.PostAuthor{DID: "did:plc:dana", Handle: "dana.test"},
	}
	resolver := fakeResolver{handleFor: "dana.test"}
	preferences := postInteractionLanguageReader{preferences: languages.Preferences{ContentLanguages: []string{"fr", "en"}}}

	tests := []struct {
		name        string
		store       *fakePostInteractionListReader
		handler     func(*fakePostInteractionListReader) http.Handler
		wantKind    api.PostInteractionKind
		wantCursor  bool
		wantTotal   bool
		wantNoPin   bool
		wantItemKey string
	}{
		{
			name: "likes account page with total and cursor",
			store: &fakePostInteractionListReader{accountPage: api.ProfileAccountPage{
				Items: []api.ProfileAccountSummary{account}, Cursor: &cursor, TotalCount: 7,
			}},
			handler: func(store *fakePostInteractionListReader) http.Handler {
				return api.ListPostLikesHandler(store, nilLogger())
			},
			wantKind: api.PostInteractionLikes, wantCursor: true, wantTotal: true, wantItemKey: "did",
		},
		{
			name: "reposts final account page omits cursor",
			store: &fakePostInteractionListReader{accountPage: api.ProfileAccountPage{
				Items: []api.ProfileAccountSummary{account}, TotalCount: 1,
			}},
			handler: func(store *fakePostInteractionListReader) http.Handler {
				return api.ListPostRepostsHandler(store, nilLogger())
			},
			wantKind: api.PostInteractionReposts, wantTotal: true, wantItemKey: "did",
		},
		{
			name:  "quotes final post page omits cursor and pin metadata",
			store: &fakePostInteractionListReader{quoteItems: []*api.PostResponse{quote}},
			handler: func(store *fakePostInteractionListReader) http.Handler {
				return api.ListPostQuotesHandler(store, resolver, nilLogger(), preferences)
			},
			wantKind: api.PostInteractionQuotes, wantNoPin: true, wantItemKey: "uri",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := servePostInteractionList(tt.handler(tt.store), "/v1/posts/did:plc:alice/root/x", "did:plc:viewer")
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			items, ok := body["items"].([]any)
			if !ok || len(items) != 1 || items[0].(map[string]any)[tt.wantItemKey] == nil {
				t.Fatalf("items = %#v", body["items"])
			}
			_, hasCursor := body["cursor"]
			if hasCursor != tt.wantCursor {
				t.Fatalf("cursor present = %v, want %v; body=%s", hasCursor, tt.wantCursor, recorder.Body.String())
			}
			_, hasTotal := body["totalCount"]
			if hasTotal != tt.wantTotal || (tt.wantTotal && body["totalCount"] != float64(tt.store.accountPage.TotalCount)) {
				t.Fatalf("totalCount = %v (present %v)", body["totalCount"], hasTotal)
			}
			if tt.wantNoPin {
				if _, present := body["pinnedPostUri"]; present {
					t.Fatalf("quote body contains pin metadata: %s", recorder.Body.String())
				}
				if !slices.Equal(tt.store.contentLanguages, []string{"fr", "en"}) || tt.store.resolver == nil {
					t.Fatalf("quote dependencies = languages %v resolver %v", tt.store.contentLanguages, tt.store.resolver)
				}
			}
			if tt.store.viewer != syntax.DID("did:plc:viewer") || tt.store.owner != syntax.DID("did:plc:alice") ||
				tt.store.rkey != syntax.RecordKey("root") || tt.store.kind != tt.wantKind {
				t.Fatalf("typed request = viewer %q owner %q rkey %q kind %q", tt.store.viewer, tt.store.owner, tt.store.rkey, tt.store.kind)
			}
		})
	}
}

// IT-006: FR-009, NFR-002; AC-015, AC-016.
func TestPostInteractionListHandlers_ValidationAndErrors(t *testing.T) {
	constructors := []struct {
		name    string
		handler func(*fakePostInteractionListReader) http.Handler
	}{
		{name: "likes", handler: func(store *fakePostInteractionListReader) http.Handler {
			return api.ListPostLikesHandler(store, nilLogger())
		}},
		{name: "reposts", handler: func(store *fakePostInteractionListReader) http.Handler {
			return api.ListPostRepostsHandler(store, nilLogger())
		}},
		{name: "quotes", handler: func(store *fakePostInteractionListReader) http.Handler {
			return api.ListPostQuotesHandler(store, fakeResolver{handleFor: "dana.test"}, nilLogger())
		}},
	}

	for _, endpoint := range constructors {
		t.Run(endpoint.name+" invalid did", func(t *testing.T) {
			store := &fakePostInteractionListReader{}
			recorder := servePostInteractionListValues(endpoint.handler(store), "not-a-did", "root", "", "did:plc:viewer")
			assertPostInteractionError(t, recorder, http.StatusBadRequest, "invalid_identifier")
			if store.owner != "" {
				t.Fatal("store called for invalid DID")
			}
		})
		t.Run(endpoint.name+" invalid rkey", func(t *testing.T) {
			store := &fakePostInteractionListReader{}
			recorder := servePostInteractionListValues(endpoint.handler(store), "did:plc:alice", "bad/key", "", "did:plc:viewer")
			assertPostInteractionError(t, recorder, http.StatusBadRequest, "invalid_identifier")
			if store.owner != "" {
				t.Fatal("store called for invalid record key")
			}
		})
		t.Run(endpoint.name+" invalid cursor", func(t *testing.T) {
			store := &fakePostInteractionListReader{listErr: envelope.ErrInvalidCursor}
			recorder := servePostInteractionListValues(endpoint.handler(store), "did:plc:alice", "root", "?cursor=bad", "did:plc:viewer")
			assertPostInteractionError(t, recorder, http.StatusBadRequest, "invalid_cursor")
		})
		t.Run(endpoint.name+" unavailable target", func(t *testing.T) {
			store := &fakePostInteractionListReader{targetErr: api.ErrPostNotFound}
			recorder := servePostInteractionList(endpoint.handler(store), "/v1/posts/did:plc:alice/missing/x", "did:plc:viewer")
			assertPostInteractionError(t, recorder, http.StatusNotFound, "post_not_found")
		})
	}

	for _, endpoint := range constructors[1:] {
		t.Run(endpoint.name+" rejects reply target", func(t *testing.T) {
			store := &fakePostInteractionListReader{targetErr: api.ErrPostNotFound}
			recorder := servePostInteractionList(endpoint.handler(store), "/v1/posts/did:plc:alice/reply/x", "did:plc:viewer")
			assertPostInteractionError(t, recorder, http.StatusNotFound, "post_not_found")
		})
	}

	for _, endpoint := range constructors {
		t.Run(endpoint.name+" identity unavailable", func(t *testing.T) {
			store := &fakePostInteractionListReader{listErr: api.ErrPostInteractionIdentityUnavailable}
			recorder := servePostInteractionList(endpoint.handler(store), "/v1/posts/did:plc:alice/root/x", "did:plc:viewer")
			assertPostInteractionError(t, recorder, http.StatusBadGateway, "identity_unavailable")
		})
		t.Run(endpoint.name+" internal failure", func(t *testing.T) {
			store := &fakePostInteractionListReader{listErr: errors.New("database unavailable")}
			recorder := servePostInteractionList(endpoint.handler(store), "/v1/posts/did:plc:alice/root/x", "did:plc:viewer")
			assertPostInteractionError(t, recorder, http.StatusInternalServerError, "internal_error")
		})
	}
}

func TestPostInteractionListHandlers_UseBoundedParseLimit(t *testing.T) {
	tests := []struct {
		query string
		want  int
	}{
		{query: "", want: 50},
		{query: "?limit=7", want: 7},
		{query: "?limit=999", want: 100},
		{query: "?limit=nope", want: 50},
		{query: "?limit=0", want: 50},
	}
	for _, tt := range tests {
		store := &fakePostInteractionListReader{}
		handler := api.ListPostLikesHandler(store, nilLogger())
		recorder := servePostInteractionListValues(handler, "did:plc:alice", "root", tt.query, "did:plc:viewer")
		if recorder.Code != http.StatusOK || store.limit != tt.want {
			t.Errorf("query %q: status=%d limit=%d, want 200/%d", tt.query, recorder.Code, store.limit, tt.want)
		}
	}
}

func servePostInteractionList(handler http.Handler, target, viewer string) *httptest.ResponseRecorder {
	parts := []string{"", "v1", "posts", "did:plc:alice", "root", "x"}
	if target == "/v1/posts/did:plc:alice/missing/x" {
		parts[4] = "missing"
	} else if target == "/v1/posts/did:plc:alice/reply/x" {
		parts[4] = "reply"
	}
	return servePostInteractionListValues(handler, parts[3], parts[4], "", viewer)
}

func servePostInteractionListValues(handler http.Handler, did, rkey, query, viewer string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/v1/posts/"+did+"/"+rkey+"/interactions"+query, nil)
	request.SetPathValue("did", did)
	request.SetPathValue("rkey", rkey)
	ctx := middleware.WithDID(request.Context(), syntax.DID(viewer))
	request = request.WithContext(ctxkeys.WithRunID(ctx, "interaction-request"))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func assertPostInteractionError(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, status, recorder.Body.String())
	}
	var raw map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if !reflect.DeepEqual(sortedMapKeys(raw), []string{"error", "message", "requestId"}) {
		t.Fatalf("error keys = %v, want standard envelope", sortedMapKeys(raw))
	}
	if raw["error"] != code || raw["message"] == "" || raw["requestId"] != "interaction-request" {
		t.Fatalf("error envelope = %#v", raw)
	}
}
