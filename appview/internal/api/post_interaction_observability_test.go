package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
	"social.craftsky/appview/internal/testdb"
)

// IT-014: NFR-005; AC-022.
func TestPostInteractionListHandlerObservabilityIsBoundedAndIdentityFree(t *testing.T) {
	const (
		viewerCanary = "did:plc:private-viewer-canary"
		ownerCanary  = "did:plc:private-owner-canary"
		handleCanary = "private-handle-canary.test"
		cursorCanary = "private-cursor-canary"
		itemCanary   = "private-item-canary"
	)

	type endpoint struct {
		name      string
		kind      api.PostInteractionKind
		pattern   string
		path      string
		construct func(*fakePostInteractionListReader, *slog.Logger) http.Handler
	}
	endpoints := []endpoint{
		{
			name: "likes", kind: api.PostInteractionLikes,
			pattern: "GET /v1/posts/{did}/{rkey}/likes",
			path:    "/v1/posts/" + ownerCanary + "/private-post-canary/likes",
			construct: func(store *fakePostInteractionListReader, logger *slog.Logger) http.Handler {
				return api.ListPostLikesHandler(store, logger)
			},
		},
		{
			name: "reposts", kind: api.PostInteractionReposts,
			pattern: "GET /v1/posts/{did}/{rkey}/reposts",
			path:    "/v1/posts/" + ownerCanary + "/private-post-canary/reposts",
			construct: func(store *fakePostInteractionListReader, logger *slog.Logger) http.Handler {
				return api.ListPostRepostsHandler(store, logger)
			},
		},
		{
			name: "quotes", kind: api.PostInteractionQuotes,
			pattern: "GET /v1/posts/{did}/{rkey}/quotes",
			path:    "/v1/posts/" + ownerCanary + "/private-post-canary/quotes",
			construct: func(store *fakePostInteractionListReader, logger *slog.Logger) http.Handler {
				return api.ListPostQuotesHandler(store, fakeResolver{handleFor: handleCanary}, logger)
			},
		},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			successStore := &fakePostInteractionListReader{}
			if endpoint.kind == api.PostInteractionQuotes {
				successStore.quoteItems = []*api.PostResponse{{
					URI:    itemCanary,
					Author: api.PostAuthor{DID: ownerCanary, Handle: handleCanary},
				}}
				successStore.quoteCursor = cursorCanary
			} else {
				cursor := cursorCanary
				successStore.accountPage = api.ProfileAccountPage{
					Items: []api.ProfileAccountSummary{{
						DID: syntax.DID(ownerCanary), Handle: syntax.Handle(handleCanary),
					}},
					Cursor: &cursor,
				}
			}

			cases := []struct {
				name           string
				store          *fakePostInteractionListReader
				path           string
				wantStatus     int
				wantResult     string
				wantErrorClass string
				wantPageFields bool
			}{
				{name: "success", store: successStore, path: endpoint.path + "?limit=7&cursor=" + cursorCanary, wantStatus: http.StatusOK, wantResult: "success", wantPageFields: true},
				{name: "validation", store: &fakePostInteractionListReader{}, path: strings.Replace(endpoint.path, ownerCanary, "invalid-did", 1), wantStatus: http.StatusBadRequest, wantResult: "error", wantErrorClass: "validation"},
				{name: "unavailable", store: &fakePostInteractionListReader{targetErr: api.ErrPostNotFound}, path: endpoint.path, wantStatus: http.StatusNotFound, wantResult: "error", wantErrorClass: "unavailable"},
				{name: "identity", store: &fakePostInteractionListReader{listErr: fmt.Errorf("%w: %s %s", api.ErrPostInteractionIdentityUnavailable, handleCanary, ownerCanary)}, path: endpoint.path, wantStatus: http.StatusBadGateway, wantResult: "error", wantErrorClass: "identity"},
				{name: "internal", store: &fakePostInteractionListReader{listErr: errors.New("database failed for " + ownerCanary + " " + cursorCanary)}, path: endpoint.path, wantStatus: http.StatusInternalServerError, wantResult: "error", wantErrorClass: "internal"},
			}

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					var logs bytes.Buffer
					logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
					metrics := observability.NewInMemoryMetricRecorder()
					observer := observability.New(observability.Config{MetricRecorder: metrics})
					mux := http.NewServeMux()
					mux.Handle(endpoint.pattern, endpoint.construct(tc.store, logger))
					handler := middleware.HTTPMetrics(observer)(mux)
					request := httptest.NewRequest(http.MethodGet, tc.path, nil)
					ctx := middleware.WithDID(request.Context(), syntax.DID(viewerCanary))
					request = request.WithContext(ctxkeys.WithRunID(ctx, "interaction-observability-request"))
					response := httptest.NewRecorder()
					handler.ServeHTTP(response, request)

					if response.Code != tc.wantStatus {
						t.Fatalf("status = %d, want %d; body=%s", response.Code, tc.wantStatus, response.Body.String())
					}
					attrs := decodeSingleJSONLog(t, logs.Bytes())
					if attrs["operation"] != endpoint.name || attrs["result"] != tc.wantResult {
						t.Fatalf("bounded result attrs = %#v, want operation=%s result=%s", attrs, endpoint.name, tc.wantResult)
					}
					if tc.wantErrorClass != "" && attrs["error_class"] != tc.wantErrorClass {
						t.Fatalf("error_class = %v, want %s; attrs=%#v", attrs["error_class"], tc.wantErrorClass, attrs)
					}
					if tc.wantPageFields {
						if attrs["limit"] != float64(7) || attrs["row_count"] != float64(1) || attrs["has_cursor"] != true {
							t.Fatalf("page attrs = %#v, want limit=7 row_count=1 has_cursor=true", attrs)
						}
					}

					calls := metrics.Calls()
					if len(calls) == 0 {
						t.Fatal("HTTP metrics were not emitted")
					}
					var sawHTTPResult bool
					for _, call := range calls {
						if err := observability.ValidateMetricCall(call); err != nil {
							t.Fatalf("invalid metric call: %v; call=%#v", err, call)
						}
						if call.Name == "craftsky_appview_http_requests_total" &&
							call.Attributes["route_pattern"] == strings.TrimPrefix(endpoint.pattern, "GET ") &&
							call.Attributes["status"] == fmt.Sprint(tc.wantStatus) {
							sawHTTPResult = true
						}
					}
					if !sawHTTPResult {
						t.Fatalf("missing bounded HTTP route/status metric: %#v", calls)
					}
					assertInteractionTelemetryExcludes(t, logs.String()+fmt.Sprint(calls), []string{
						viewerCanary, ownerCanary, handleCanary, cursorCanary, itemCanary,
					})
				})
			}
		})
	}
}

func TestPostInteractionListStoreEmitsBoundedDBObservations(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithSchema(t, postStoreDDL+postInteractionIdentityDDL)
	viewer := syntax.DID("did:plc:private-viewer-store-canary")
	owner := syntax.DID("did:plc:private-owner-store-canary")
	actor := syntax.DID("did:plc:private-actor-store-canary")
	for _, did := range []syntax.DID{viewer, owner, actor} {
		seedMember(t, pool, did.String())
		seedBskyProfile(t, pool, did.String(), "private-item-store-canary", "")
	}
	if _, err := pool.Exec(ctx, `INSERT INTO atproto_identity_cache (did, handle, handle_lower) VALUES ($1, $2, $2)`, actor, "private-store-handle-canary.test"); err != nil {
		t.Fatalf("seed identity: %v", err)
	}
	createdAt := time.Date(2026, 9, 6, 16, 0, 0, 0, time.UTC)
	targetURI := seedPost(t, pool, owner.String(), "private-post-store-canary", "target", createdAt)
	seedInteraction(t, pool, "like", actor.String(), "private-like-store-canary", targetURI, false)
	seedInteraction(t, pool, "repost", actor.String(), "private-repost-store-canary", targetURI, false)
	seedQuotePost(t, pool, actor.String(), "private-quote-store-canary", "private quote item", targetURI, "bafyprivate", createdAt.Add(time.Minute))

	metrics := observability.NewInMemoryMetricRecorder()
	store := api.NewPostStore(pool, observability.New(observability.Config{MetricRecorder: metrics}))
	for _, kind := range []api.PostInteractionKind{api.PostInteractionLikes, api.PostInteractionReposts, api.PostInteractionQuotes} {
		target, err := store.ResolveInteractionTarget(ctx, viewer, owner, syntax.RecordKey("private-post-store-canary"), kind)
		if err != nil {
			t.Fatalf("resolve %s target: %v", kind, err)
		}
		if kind == api.PostInteractionQuotes {
			if _, _, err := store.ListQuotePosts(ctx, viewer, target, []string{}, 7, ""); err != nil {
				t.Fatalf("list %s: %v", kind, err)
			}
		} else if _, err := store.ListPostInteractionAccounts(ctx, viewer, target, kind, 7, ""); err != nil {
			t.Fatalf("list %s: %v", kind, err)
		}
	}

	wantOperations := map[string]bool{"likes": false, "reposts": false, "quotes": false}
	var interactionCalls []observability.MetricCall
	for _, call := range metrics.Calls() {
		if call.Name != "craftsky_appview_db_operation_duration_seconds" || !strings.HasPrefix(call.Attributes["operation"], "post.interactions.") {
			continue
		}
		interactionCalls = append(interactionCalls, call)
		kind := strings.TrimPrefix(call.Attributes["operation"], "post.interactions.")
		if _, ok := wantOperations[kind]; ok && call.Attributes["result"] == "success" {
			wantOperations[kind] = true
		}
		if err := observability.ValidateMetricCall(call); err != nil {
			t.Fatalf("invalid DB metric: %v; call=%#v", err, call)
		}
	}
	for operation, seen := range wantOperations {
		if !seen {
			t.Errorf("missing bounded %s DB observation: %#v", operation, interactionCalls)
		}
	}
	assertInteractionTelemetryExcludes(t, fmt.Sprint(interactionCalls), []string{
		viewer.String(), owner.String(), actor.String(), targetURI,
		"private-store-handle-canary.test", "private-item-store-canary", "private quote item",
	})
}

func decodeSingleJSONLog(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	lines := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
	if len(lines) != 1 || len(lines[0]) == 0 {
		t.Fatalf("log entries = %d, want exactly one; logs=%s", len(lines), raw)
	}
	var attrs map[string]any
	if err := json.Unmarshal(lines[0], &attrs); err != nil {
		t.Fatalf("decode log: %v; log=%s", err, lines[0])
	}
	return attrs
}

func assertInteractionTelemetryExcludes(t *testing.T, telemetry string, forbidden []string) {
	t.Helper()
	for _, value := range forbidden {
		if strings.Contains(telemetry, value) {
			t.Fatalf("interaction telemetry leaked %q: %s", value, telemetry)
		}
	}
}
