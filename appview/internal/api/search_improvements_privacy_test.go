package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/observability"
)

// IT-012 / RULE-003 / AC-016: serialized actual local and SDK sinks.
func TestSearchImprovementsSerializedPrivacy(t *testing.T) {
	pool, _, now := searchFixture(t)
	for _, project := range []bool{false, true} {
		searchFixturePost(t, pool, project, map[bool]string{false: "post", true: "project"}[project], "privatequeryneedle blanket", nil, now)
	}
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO moderation_cases(id,subject_key,subject_type,subject_did,owner_did,safe_snapshot) VALUES('11111111-1111-4111-8111-111111111111','test','account','did:plc:author','did:plc:author','{"text":"privatequiltcanary"}')`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at) VALUES('did:plc:author','active',1,1,'test',now())`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO scheduled_posts(id,owner_did,owner_generation,operation_id,request_hash,status,scheduled_at,next_attempt_at,payload_bytes,payload_hash) VALUES('22222222-2222-4222-8222-222222222222','did:plc:author',1,'33333333-3333-4333-8333-333333333333',decode(repeat('00',32),'hex'),'scheduled',now()+interval '1 day',now()+interval '1 day',convert_to('{"text":"privateschedulecanary"}','UTF8'),decode(repeat('00',32),'hex'))`); err != nil {
		t.Fatal(err)
	}
	var moderationMarker, scheduledMarker string
	if err := pool.QueryRow(ctx, `SELECT coalesce((SELECT safe_snapshot->>'text' FROM moderation_cases WHERE id='11111111-1111-4111-8111-111111111111'),'')`).Scan(&moderationMarker); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT coalesce((SELECT convert_from(payload_bytes,'UTF8')::jsonb->>'text' FROM scheduled_posts WHERE id='22222222-2222-4222-8222-222222222222'),'')`).Scan(&scheduledMarker); err != nil {
		t.Fatal(err)
	}
	if moderationMarker != "privatequiltcanary" || scheduledMarker != "privateschedulecanary" {
		t.Fatal("private-content canaries missing from captured observer fixture")
	}
	var local bytes.Buffer
	logger := slog.New(observability.NewDiagnosticHandler(slog.NewJSONHandler(&local, nil)))
	transport := &sentry.MockTransport{}
	observer := observability.New(observability.Config{Env: "test", Logger: logger, SentryDSN: "https://public@example.invalid/1", SentryTransport: transport, TracingEnabled: true, TracesSampleRate: 1})
	store := api.NewSearchStore(pool, observer)
	run := func(route, q, extra string, want int) api.SearchPostPageResponse {
		t.Helper()
		request := authedReq("GET", route+"?q="+url.QueryEscape(q)+extra, "", "did:plc:viewer")
		ctx, span := observer.StartSpan(request.Context(), observability.SpanContext{Operation: "http.server", Component: "http"})
		request = request.WithContext(ctx)
		response := httptest.NewRecorder()
		if strings.HasSuffix(route, "posts") {
			api.SearchPostsHandler(store, fakeResolver{}, logger).ServeHTTP(response, request)
		} else {
			api.SearchProjectsHandler(store, fakeResolver{}, logger).ServeHTTP(response, request)
		}
		span.Finish("success")
		if response.Code != want {
			t.Fatalf("status=%d expected=%d body=%s", response.Code, want, response.Body.String())
		}
		var page api.SearchPostPageResponse
		if want == 200 {
			if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
				t.Fatal(err)
			}
		}
		return page
	}
	for _, route := range []string{"/v1/search/posts", "/v1/search/projects"} {
		run(route, "privatequeryneedlex blanket", "", 200)
		run(route, "emptymarkerprivate", "", 200)
		for _, marker := range []string{moderationMarker, scheduledMarker} {
			page := run(route, marker, "", 200)
			if len(page.Items) != 0 || page.Cursor != "" {
				t.Fatal("private-only marker created a search result")
			}
		}
		run(route, "privatequeryneedlex blanket", "&limit=101", 400)
	}
	pool.Close()
	for _, route := range []string{"/v1/search/posts", "/v1/search/projects"} {
		run(route, "privatequeryneedlex blanket", "", 500)
	}
	if !observer.Flush(2 * time.Second) {
		t.Fatal("flush failed")
	}
	events, err := json.Marshal(transport.Events())
	if err != nil {
		t.Fatal(err)
	}
	for name, payload := range map[string]string{"local": local.String(), "sdk": string(events)} {
		for _, protected := range []string{"privatequeryneedlex", "privatequeryneedle", "emptymarkerprivate", "privatequiltcanary", "privateschedulecanary"} {
			if strings.Contains(payload, protected) {
				t.Fatalf("%s serialized protected value %q", name, protected)
			}
		}
		if !strings.Contains(payload, "search.posts") || !strings.Contains(payload, "search.projects") {
			t.Fatalf("%s missing permitted operation context", name)
		}
	}
	if len(transport.Events()) == 0 || local.Len() == 0 {
		t.Fatal("empty sink is not privacy evidence")
	}
}
