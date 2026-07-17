// appview/internal/api/notifications_test.go
package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
)

type fakeNotificationStore struct {
	rows         []*api.NotificationRow
	cursor       string
	err          error
	engagement   map[string]api.EngagementSummary
	lastViewer   string
	lastLimit    int
	lastCursor   string
	engagementIn []string
	handles      map[string]syntax.Handle
	handleCalls  int
}

func (f *fakeNotificationStore) ListNotifications(_ context.Context, viewerDID string, limit int, cursor string) ([]*api.NotificationRow, string, error) {
	f.lastViewer = viewerDID
	f.lastLimit = limit
	f.lastCursor = cursor
	return f.rows, f.cursor, f.err
}

func (f *fakeNotificationStore) EngagementSummaries(_ context.Context, _ string, postURIs []string) (map[string]api.EngagementSummary, error) {
	f.engagementIn = append([]string(nil), postURIs...)
	out := make(map[string]api.EngagementSummary, len(postURIs))
	for _, uri := range postURIs {
		out[uri] = api.EngagementSummary{}
	}
	for uri, summary := range f.engagement {
		out[uri] = summary
	}
	return out, nil
}

func (f *fakeNotificationStore) NotificationHandles(_ context.Context, dids []string) (map[string]syntax.Handle, error) {
	f.handleCalls++
	out := make(map[string]syntax.Handle, len(dids))
	for _, did := range dids {
		if handle, ok := f.handles[did]; ok {
			out[did] = handle
		}
	}
	return out, nil
}

func TestNotificationsHandlerUsesOneIndexedHandleBatchAndSurvivesMissingActors(t *testing.T) {
	rows := make([]*api.NotificationRow, 0, 50)
	for i := 0; i < 50; i++ {
		rows = append(rows, &api.NotificationRow{
			ID: fmt.Sprintf("id-%d", i), Type: api.NotificationTypeFollow,
			ActorDID: fmt.Sprintf("did:plc:actor%d", i), CreatedAt: time.Now(), IndexedAt: time.Now(),
		})
	}
	store := &fakeNotificationStore{rows: rows, handles: map[string]syntax.Handle{"did:plc:actor0": "actor0.example"}}
	handler := api.ListNotificationsHandler(store, fakeResolver{err: errors.New("directory unavailable")}, nilLogger())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, authedReq(http.MethodGet, "/v1/notifications?limit=50", "", "did:plc:viewer"))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if store.handleCalls != 1 {
		t.Fatalf("handle batch calls=%d, want 1", store.handleCalls)
	}
	var page api.NotificationPage
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 50 || !page.Items[0].Actor.Available || page.Items[1].Actor.Available {
		t.Fatalf("actor availability not explicit: first=%+v second=%+v", page.Items[0].Actor, page.Items[1].Actor)
	}
}

func TestNotificationsHandlerUT020IncludesDisplayReadyActorAvatar(t *testing.T) {
	avatarCID := "bafy-avatar"
	avatarMIME := "image/jpeg"
	store := &fakeNotificationStore{
		handles: map[string]syntax.Handle{"did:plc:alice": "alice.example"},
		rows: []*api.NotificationRow{{
			ID: "avatar-notification", Type: api.NotificationTypeFollow,
			ActorDID: "did:plc:alice", ActorAvatarCID: &avatarCID, ActorAvatarMime: &avatarMIME,
			CreatedAt: time.Now(), IndexedAt: time.Now(),
		}},
	}
	handler := api.ListNotificationsHandler(store, fakeResolver{}, nilLogger())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, authedReq(http.MethodGet, "/v1/notifications", "", "did:plc:viewer"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var page api.NotificationPage
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Actor.Avatar == nil {
		t.Fatalf("actor=%+v, want display-ready avatar", page.Items[0].Actor)
	}
	if got, want := *page.Items[0].Actor.Avatar, "https://cdn.bsky.app/img/avatar/plain/did:plc:alice/bafy-avatar@jpeg"; got != want {
		t.Fatalf("avatar=%q, want %q", got, want)
	}
}

func TestNotificationsHandlerUT022IncludesActorFollowState(t *testing.T) {
	store := &fakeNotificationStore{
		handles: map[string]syntax.Handle{"did:plc:alice": "alice.example"},
		rows: []*api.NotificationRow{{
			ID: "follow-state-notification", Type: api.NotificationTypeFollow,
			ActorDID: "did:plc:alice", ActorViewerIsFollowing: true,
			CreatedAt: time.Now(), IndexedAt: time.Now(),
		}},
	}
	handler := api.ListNotificationsHandler(store, fakeResolver{}, nilLogger())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, authedReq(http.MethodGet, "/v1/notifications", "", "did:plc:viewer"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var page api.NotificationPage
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || !page.Items[0].Actor.ViewerIsFollowing {
		t.Fatalf("actor=%+v, want viewerIsFollowing=true", page.Items[0].Actor)
	}
}

func TestNotificationsHandler_IgnoresUnknownParamsUsesLimitsAndSessionViewer(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantLimit  int
		wantCursor string
	}{
		{name: "default", path: "/v1/notifications?foo=bar&did=did:plc:other", wantLimit: 20},
		{name: "cap", path: "/v1/notifications?limit=999&cursor=opaque&viewerDid=did:plc:other", wantLimit: 50, wantCursor: "opaque"},
		{name: "invalid", path: "/v1/notifications?limit=nope", wantLimit: 20},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeNotificationStore{}
			handler := api.ListNotificationsHandler(store, fakeResolver{handleFor: "viewer.example"}, nilLogger())
			req := authedReq(http.MethodGet, tt.path, "", "did:plc:viewer")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
			}
			if store.lastViewer != "did:plc:viewer" {
				t.Fatalf("store viewer = %q, want authenticated viewer", store.lastViewer)
			}
			if store.lastLimit != tt.wantLimit || store.lastCursor != tt.wantCursor {
				t.Fatalf("store limit/cursor = %d/%q, want %d/%q", store.lastLimit, store.lastCursor, tt.wantLimit, tt.wantCursor)
			}
		})
	}
}

func TestNotificationsHandler_ReturnsCamelCaseNotificationPage(t *testing.T) {
	subject := testPostRow("did:plc:viewer", "root", "viewer post", time.Date(2026, 5, 28, 17, 0, 0, 0, time.UTC))
	store := &fakeNotificationStore{handles: map[string]syntax.Handle{
		"did:plc:alice": "alice.example", "did:plc:viewer": "viewer.example",
	}, rows: []*api.NotificationRow{
		{
			Type: api.NotificationTypeLike, URI: "at://did:plc:alice/social.craftsky.feed.like/like1", CID: "bafylike", Rkey: "like1",
			ActorDID: "did:plc:alice", CreatedAt: subject.CreatedAt, IndexedAt: subject.IndexedAt, SubjectPost: subject,
		},
		{
			Type: api.NotificationTypeReply, URI: "at://did:plc:alice/social.craftsky.feed.post/reply1", CID: "bafyreply", Rkey: "reply1",
			ActorDID: "did:plc:alice", CreatedAt: subject.CreatedAt, IndexedAt: subject.IndexedAt, SubjectPost: subject,
			Reply: &api.NotificationReplyRef{URI: "at://did:plc:alice/social.craftsky.feed.post/reply1", CID: "bafyreply", Rkey: "reply1"},
		},
	}, cursor: "next-cursor"}
	handler := api.ListNotificationsHandler(store, fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice":  "alice.example",
		"did:plc:viewer": "viewer.example",
	}}, nilLogger())

	req := authedReq(http.MethodGet, "/v1/notifications", "", "did:plc:viewer")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if _, ok := raw["totalCount"]; ok {
		t.Fatalf("body contains totalCount: %s", rec.Body.String())
	}
	var page api.NotificationPage
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("body not notification page: %v", err)
	}
	if page.Cursor != "next-cursor" || len(page.Items) != 2 {
		t.Fatalf("page = %+v, want two items and cursor", page)
	}
	item := page.Items[0]
	if item.Type != api.NotificationTypeLike || item.Actor.Handle != "alice.example" || item.SubjectPost == nil || item.SubjectPost.URI != subject.URI {
		t.Fatalf("item = %+v, want like with actor handle and subject post", item)
	}
	var body struct {
		Items []map[string]json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not raw notification page: %v", err)
	}
	var replyRaw map[string]json.RawMessage
	if err := json.Unmarshal(body.Items[1]["reply"], &replyRaw); err != nil {
		t.Fatalf("reply not object: %v", err)
	}
	for _, key := range []string{"uri", "cid", "rkey"} {
		if _, ok := replyRaw[key]; !ok {
			t.Fatalf("reply raw = %v, want camelCase key %q", replyRaw, key)
		}
	}
	for _, key := range []string{"URI", "CID", "Rkey"} {
		if _, ok := replyRaw[key]; ok {
			t.Fatalf("reply raw = %v, must not contain Go field key %q", replyRaw, key)
		}
	}
}

func TestNotificationsHandlerReturnsCompleteTypeSpecificReferenceMatrix(t *testing.T) {
	now := time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC)
	postFor := func(rkey string) *api.PostRow {
		return testPostRow("did:plc:viewer", rkey, "visible", now)
	}
	ref := func(uri, cid, rkey string) api.NotificationReference {
		return api.NotificationReference{Available: true, URI: uri, CID: cid, Rkey: rkey}
	}
	postRef := func(uri, cid string) *api.NotificationReference {
		value := ref(uri, cid, "")
		return &value
	}
	rows := []*api.NotificationRow{}
	for i, category := range []api.NotificationType{
		api.NotificationTypeFollow, api.NotificationTypeLike, api.NotificationTypeRepost,
		api.NotificationTypeReply, api.NotificationTypeMention, api.NotificationTypeQuote,
		api.NotificationTypeEverythingElse,
	} {
		source := ref(fmt.Sprintf("at://did:plc:alice/event/%d", i), fmt.Sprintf("source-cid-%d", i), fmt.Sprintf("r%d", i))
		row := &api.NotificationRow{ID: fmt.Sprintf("id-%d", i), Type: category, ActorDID: "did:plc:alice", CreatedAt: now, IndexedAt: now, References: api.NotificationReferences{Source: source}}
		switch category {
		case api.NotificationTypeLike, api.NotificationTypeRepost:
			row.SubjectPost = postFor(string(category))
			row.References.Subject = postRef(row.SubjectPost.URI, row.SubjectPost.CID)
		case api.NotificationTypeReply:
			row.SubjectPost = postFor("reply-parent")
			row.References.Subject = postRef(row.SubjectPost.URI, row.SubjectPost.CID)
			row.References.Parent = postRef(row.SubjectPost.URI, row.SubjectPost.CID)
			row.References.Root = postRef(row.SubjectPost.URI, row.SubjectPost.CID)
			row.Reply = &api.NotificationReplyRef{Available: true, URI: source.URI, CID: source.CID, Rkey: source.Rkey}
		case api.NotificationTypeMention:
			row.SubjectPost = postFor("mention")
			row.References.Subject = postRef(row.SubjectPost.URI, row.SubjectPost.CID)
		case api.NotificationTypeQuote:
			row.SubjectPost = postFor("quote")
			quotedURI := "at://did:plc:viewer/social.craftsky.feed.post/quoted"
			quotedCID := "quoted-cid"
			row.SubjectPost.QuoteURI = &quotedURI
			row.SubjectPost.QuoteCID = &quotedCID
			row.References.Subject = postRef(row.SubjectPost.URI, row.SubjectPost.CID)
			row.References.Quoted = postRef(quotedURI, quotedCID)
		}
		rows = append(rows, row)
	}
	store := &fakeNotificationStore{rows: rows, handles: map[string]syntax.Handle{"did:plc:alice": "alice.example", "did:plc:viewer": "viewer.example"}}
	recorder := httptest.NewRecorder()
	api.ListNotificationsHandler(store, fakeResolver{}, nilLogger()).ServeHTTP(recorder, authedReq(http.MethodGet, "/v1/notifications", "", "did:plc:viewer"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var page api.NotificationPage
	if err := json.Unmarshal(recorder.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 7 {
		t.Fatalf("items=%d", len(page.Items))
	}
	for _, item := range page.Items {
		if !item.References.Source.Available || item.URI == "" || item.CID == "" || item.Rkey == "" {
			t.Fatalf("%s source=%+v item=%+v", item.Type, item.References.Source, item)
		}
		switch item.Type {
		case api.NotificationTypeFollow, api.NotificationTypeEverythingElse:
			if item.SubjectPost != nil || item.Reply != nil || item.ContentAvailable != nil || item.References.Subject != nil || item.References.Parent != nil || item.References.Root != nil || item.References.Quoted != nil {
				t.Fatalf("%s has unrelated metadata: %+v", item.Type, item)
			}
		case api.NotificationTypeLike, api.NotificationTypeRepost:
			if item.SubjectPost == nil || item.References.Subject == nil || item.ContentAvailable == nil || !*item.ContentAvailable || item.Reply != nil || item.References.Parent != nil || item.References.Root != nil || item.References.Quoted != nil {
				t.Fatalf("%s metadata=%+v", item.Type, item)
			}
		case api.NotificationTypeReply:
			if item.SubjectPost == nil || item.Reply == nil || !item.Reply.Available || item.References.Subject == nil || item.References.Parent == nil || item.References.Root == nil || item.References.Quoted != nil || item.ContentAvailable == nil || !*item.ContentAvailable {
				t.Fatalf("reply metadata=%+v", item)
			}
		case api.NotificationTypeMention:
			if item.SubjectPost == nil || item.References.Subject == nil || item.Reply != nil || item.References.Parent != nil || item.References.Root != nil || item.References.Quoted != nil || item.ContentAvailable == nil || !*item.ContentAvailable {
				t.Fatalf("mention metadata=%+v", item)
			}
		case api.NotificationTypeQuote:
			if item.SubjectPost == nil || item.SubjectPost.Quote == nil || item.References.Subject == nil || item.References.Quoted == nil || item.Reply != nil || item.References.Parent != nil || item.References.Root != nil || item.ContentAvailable == nil || !*item.ContentAvailable {
				t.Fatalf("quote metadata=%+v", item)
			}
		}
	}
}

func TestNotificationsHandler_InvalidCursorUsesStandardErrorEnvelope(t *testing.T) {
	store := &fakeNotificationStore{err: envelope.ErrInvalidCursor}
	handler := api.ListNotificationsHandler(store, fakeResolver{handleFor: "viewer.example"}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/notifications?cursor=bad", "", "did:plc:viewer")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	var env envelope.Error
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil || env.Error != "invalid_cursor" || env.Message == "" {
		t.Fatalf("envelope = %+v err=%v, want invalid_cursor", env, err)
	}
}

func TestNotificationsHandler_StoreFailureUsesServerErrorEnvelope(t *testing.T) {
	store := &fakeNotificationStore{err: errors.New("database exploded")}
	handler := api.ListNotificationsHandler(store, fakeResolver{handleFor: "viewer.example"}, nilLogger())
	req := authedReq(http.MethodGet, "/v1/notifications", "", "did:plc:viewer")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() == "" || errors.Is(store.err, envelope.ErrInvalidCursor) {
		t.Fatalf("unexpected body/error: %s", rec.Body.String())
	}
}
