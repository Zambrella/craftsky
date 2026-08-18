package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/languages"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/testdb"
)

func TestProfilePinFirstPagePromotionAndMetadataOmission(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000035_profile_pins.up.sql")
	if err != nil {
		t.Fatalf("read profile pin migration: %v", err)
	}
	pool := testdb.WithSchema(t, postStoreDDL+string(migration))
	ctx := ownerlifecycle.WithExpectedGeneration(context.Background(), 1)
	base := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	for _, did := range []string{"did:plc:alice", "did:plc:bob", "did:plc:carol", "did:plc:viewer"} {
		seedMember(t, pool, did)
		seedBskyProfile(t, pool, did, did, "")
	}

	standardPin := seedPost(t, pool, "did:plc:alice", "standard-pin", "Pinned standard", base)
	projectPin := seedPost(t, pool, "did:plc:alice", "project-pin", "Pinned project", base)
	seedProjectMaterialization(t, pool, projectPin, "social.craftsky.feed.defs#knitting", "Pinned project")
	for index := 0; index < 13; index++ {
		when := base.Add(time.Duration(index+1) * time.Minute)
		seedPost(t, pool, "did:plc:alice", "standard-"+twoDigits(index), "Standard", when)
		projectURI := seedPost(t, pool, "did:plc:alice", "project-"+twoDigits(index), "Project", when)
		seedProjectMaterialization(t, pool, projectURI, "social.craftsky.feed.defs#knitting", "Project")
	}

	pinStore := api.NewProfilePinStore(pool)
	owner := syntax.DID("did:plc:alice")
	if _, err := pinStore.Pin(ctx, owner, owner, syntax.RecordKey("standard-pin")); err != nil {
		t.Fatalf("pin standard: %v", err)
	}
	if _, err := pinStore.Pin(ctx, owner, owner, syntax.RecordKey("project-pin")); err != nil {
		t.Fatalf("pin project: %v", err)
	}
	postStore := api.NewPostStore(pool)
	resolver := fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:bob":   "bob.example",
		"did:plc:carol": "carol.example",
	}}

	standard := serveProfilePinPage(
		t,
		api.ListPostsByAuthorHandler(postStore, resolver, nilLogger(), pinStore),
		"did:plc:alice",
		10,
	)
	if len(standard.Items) != 10 || standard.Items[0].URI != standardPin {
		t.Fatalf("standard page = %+v, want pin first and exactly 10", postResponseURIs(standard.Items))
	}
	if standard.PinnedPostURI == nil || *standard.PinnedPostURI != standardPin {
		t.Fatalf("standard pinnedPostUri = %v, want %q", standard.PinnedPostURI, standardPin)
	}
	if standard.Items[1].Rkey != "standard-12" || standard.Items[9].Rkey != "standard-04" {
		t.Fatalf("standard chronology = %v", postResponseURIs(standard.Items))
	}
	if standard.Cursor == "" {
		t.Fatal("standard page cursor is empty")
	}

	project := serveProfilePinPage(
		t,
		api.ListProjectsByAuthorHandler(postStore, resolver, nilLogger(), pinStore),
		"did:plc:alice",
		2,
	)
	if len(project.Items) != 2 || project.Items[0].URI != projectPin || project.Items[1].Rkey != "project-12" {
		t.Fatalf("project page = %v, want pin plus newest", postResponseURIs(project.Items))
	}
	if project.PinnedPostURI == nil || *project.PinnedPostURI != projectPin {
		t.Fatalf("project pinnedPostUri = %v, want %q", project.PinnedPostURI, projectPin)
	}

	limitOne := serveProfilePinPage(
		t,
		api.ListPostsByAuthorHandler(postStore, resolver, nilLogger(), pinStore),
		"did:plc:alice",
		1,
	)
	if len(limitOne.Items) != 1 || limitOne.Items[0].URI != standardPin || limitOne.Cursor == "" {
		t.Fatalf("limit-one page = items:%v cursor:%q", postResponseURIs(limitOne.Items), limitOne.Cursor)
	}

	bobNewest := seedPost(t, pool, "did:plc:bob", "newest", "Newest", base.Add(2*time.Minute))
	seedPost(t, pool, "did:plc:bob", "older", "Older", base.Add(time.Minute))
	noPin := serveProfilePinPage(
		t,
		api.ListPostsByAuthorHandler(postStore, resolver, nilLogger(), pinStore),
		"did:plc:bob",
		2,
	)
	if len(noPin.Items) != 2 || noPin.Items[0].URI != bobNewest || noPin.HasPinnedPostURI {
		t.Fatalf("no-pin page = items:%v hasPinnedPostUri:%v", postResponseURIs(noPin.Items), noPin.HasPinnedPostURI)
	}

	hiddenPin := seedPost(t, pool, "did:plc:carol", "hidden-pin", "Hidden", base)
	visibleNewest := seedPost(t, pool, "did:plc:carol", "visible-newest", "Visible", base.Add(2*time.Minute))
	visibleOlder := seedPost(t, pool, "did:plc:carol", "visible-older", "Visible", base.Add(time.Minute))
	carol := syntax.DID("did:plc:carol")
	if _, err := pinStore.Pin(ctx, carol, carol, syntax.RecordKey("hidden-pin")); err != nil {
		t.Fatalf("pin before hide: %v", err)
	}
	seedModerationOutput(t, pool, "post", carol.String(), hiddenPin, "hide", base.Add(3*time.Minute))
	hidden := serveProfilePinPage(
		t,
		api.ListPostsByAuthorHandler(postStore, resolver, nilLogger(), pinStore),
		"did:plc:carol",
		2,
	)
	if got := postResponseURIs(hidden.Items); len(got) != 2 || got[0] != visibleNewest || got[1] != visibleOlder || hidden.HasPinnedPostURI {
		t.Fatalf("hidden-pin page = items:%v hasPinnedPostUri:%v", got, hidden.HasPinnedPostURI)
	}
}

func TestProfilePinTraversalIsUniqueAndPinChangesInvalidateCursor(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000035_profile_pins.up.sql")
	if err != nil {
		t.Fatalf("read profile pin migration: %v", err)
	}
	pool := testdb.WithSchema(t, postStoreDDL+string(migration))
	ctx := ownerlifecycle.WithExpectedGeneration(context.Background(), 1)
	base := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	for _, did := range []string{"did:plc:alice", "did:plc:viewer"} {
		seedMember(t, pool, did)
		seedBskyProfile(t, pool, did, did, "")
	}
	pinnedURI := seedPost(t, pool, "did:plc:alice", "pinned", "Pinned", base)
	for index := 0; index < 7; index++ {
		when := base.Add(time.Duration(index+1) * time.Minute)
		if index == 3 {
			when = base.Add(3 * time.Minute)
		}
		seedPost(t, pool, "did:plc:alice", "post-"+twoDigits(index), "Post", when)
	}
	pinStore := api.NewProfilePinStore(pool)
	owner := syntax.DID("did:plc:alice")
	if _, err := pinStore.Pin(ctx, owner, owner, syntax.RecordKey("pinned")); err != nil {
		t.Fatalf("pin initial: %v", err)
	}
	postStore := api.NewPostStore(pool)
	resolver := fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
	}}
	postsHandler := api.ListPostsByAuthorHandler(postStore, resolver, nilLogger(), pinStore)
	projectsHandler := api.ListProjectsByAuthorHandler(postStore, resolver, nilLogger(), pinStore)

	var (
		cursor      string
		firstCursor string
		seen        []string
	)
	for page := 0; page < 10; page++ {
		response := requestProfilePinPage(t, postsHandler, "did:plc:alice", 2, cursor)
		if response.Code != http.StatusOK {
			t.Fatalf("page %d status = %d, body = %s", page, response.Code, response.Body.String())
		}
		var raw map[string]json.RawMessage
		var decoded struct {
			Items         []api.PostResponse `json:"items"`
			Cursor        string             `json:"cursor,omitempty"`
			PinnedPostURI *string            `json:"pinnedPostUri,omitempty"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &raw); err != nil {
			t.Fatalf("decode raw page %d: %v", page, err)
		}
		if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("decode page %d: %v", page, err)
		}
		if page == 0 {
			if decoded.PinnedPostURI == nil || *decoded.PinnedPostURI != pinnedURI {
				t.Fatalf("first pinnedPostUri = %v", decoded.PinnedPostURI)
			}
			firstCursor = decoded.Cursor
		} else if _, present := raw["pinnedPostUri"]; present {
			t.Fatalf("later page %d leaked pinnedPostUri: %s", page, response.Body.String())
		}
		seen = append(seen, postResponseURIs(decoded.Items)...)
		cursor = decoded.Cursor
		if cursor == "" {
			break
		}
	}
	if firstCursor == "" {
		t.Fatal("first cursor is empty")
	}
	if len(seen) != 8 {
		t.Fatalf("traversal returned %d rows: %v", len(seen), seen)
	}
	unique := make(map[string]struct{}, len(seen))
	for _, uri := range seen {
		if _, duplicate := unique[uri]; duplicate {
			t.Fatalf("duplicate traversal row %q in %v", uri, seen)
		}
		unique[uri] = struct{}{}
	}

	if _, err := pinStore.Pin(ctx, owner, owner, syntax.RecordKey("post-06")); err != nil {
		t.Fatalf("replace pin: %v", err)
	}
	assertInvalidProfileCursor(t, postsHandler, firstCursor)
	wrongKind := requestProfilePinPage(t, projectsHandler, "did:plc:alice", 2, firstCursor)
	if wrongKind.Code != http.StatusBadRequest {
		t.Fatalf("wrong-list-kind cursor status = %d, body = %s", wrongKind.Code, wrongKind.Body.String())
	}

	fresh := serveProfilePinPage(t, postsHandler, "did:plc:alice", 2)
	if fresh.Cursor == "" {
		t.Fatal("fresh replacement cursor is empty")
	}
	if _, err := pinStore.Unpin(ctx, owner, syntax.ATURI(fresh.Items[0].URI)); err != nil {
		t.Fatalf("clear replacement: %v", err)
	}
	assertInvalidProfileCursor(t, postsHandler, fresh.Cursor)
}

func TestProfilePinPromotionRespectsAndRetainsViewerPolicy(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/000035_profile_pins.up.sql")
	if err != nil {
		t.Fatalf("read profile pin migration: %v", err)
	}
	pool := testdb.WithSchema(t, postStoreDDL+string(migration))
	ctx := ownerlifecycle.WithExpectedGeneration(context.Background(), 1)
	base := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	for _, did := range []string{"did:plc:alice", "did:plc:viewer"} {
		seedMember(t, pool, did)
		seedBskyProfile(t, pool, did, did, "")
	}
	pinnedURI := seedPost(t, pool, "did:plc:alice", "french-pin", "Bonjour", base)
	englishNewest := seedPost(t, pool, "did:plc:alice", "english-newest", "Newest", base.Add(2*time.Minute))
	englishOlder := seedPost(t, pool, "did:plc:alice", "english-older", "Older", base.Add(time.Minute))
	if _, err := pool.Exec(ctx, `
		UPDATE craftsky_posts
		SET langs = CASE WHEN uri = $1 THEN ARRAY['fr']::text[] ELSE ARRAY['en']::text[] END
		WHERE uri IN ($1, $2, $3)
	`, pinnedURI, englishNewest, englishOlder); err != nil {
		t.Fatalf("seed profile languages: %v", err)
	}
	pinStore := api.NewProfilePinStore(pool)
	owner := syntax.DID("did:plc:alice")
	if _, err := pinStore.Pin(ctx, owner, owner, syntax.RecordKey("french-pin")); err != nil {
		t.Fatalf("pin french post: %v", err)
	}
	preferences := &fakeLanguagePreferenceReader{preferences: languages.Preferences{
		PrimaryLanguage:  "en",
		ContentLanguages: []string{"en"},
	}}
	handler := api.ListPostsByAuthorHandler(
		api.NewPostStore(pool),
		fakeResolver{handlesByDID: map[string]syntax.Handle{"did:plc:alice": "alice.example"}},
		nilLogger(),
		pinStore,
		preferences,
	)

	filtered := serveProfilePinPage(t, handler, "did:plc:alice", 2)
	if got := postResponseURIs(filtered.Items); len(got) != 2 || got[0] != englishNewest || got[1] != englishOlder || filtered.HasPinnedPostURI {
		t.Fatalf("language-filtered page = items:%v hasPinnedPostUri:%v", got, filtered.HasPinnedPostURI)
	}
	state, err := pinStore.Read(ctx, owner)
	if err != nil || state.StandardPostURI == nil || state.StandardPostURI.String() != pinnedURI {
		t.Fatalf("retained language-filtered state = %+v err=%v", state, err)
	}

	preferences.preferences.ContentLanguages = []string{"fr"}
	restored := serveProfilePinPage(t, handler, "did:plc:alice", 2)
	if len(restored.Items) != 1 || restored.Items[0].URI != pinnedURI || !restored.HasPinnedPostURI {
		t.Fatalf("language-restored page = items:%v hasPinnedPostUri:%v", postResponseURIs(restored.Items), restored.HasPinnedPostURI)
	}

	seedModerationOutput(t, pool, "post", owner.String(), pinnedURI, "hide", base.Add(3*time.Minute))
	moderated := serveProfilePinPage(t, handler, "did:plc:alice", 2)
	if len(moderated.Items) != 0 || moderated.HasPinnedPostURI {
		t.Fatalf("moderated page = items:%v hasPinnedPostUri:%v", postResponseURIs(moderated.Items), moderated.HasPinnedPostURI)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM moderation_outputs WHERE subject_uri = $1`, pinnedURI); err != nil {
		t.Fatalf("reverse moderation: %v", err)
	}
	moderationRestored := serveProfilePinPage(t, handler, "did:plc:alice", 2)
	if len(moderationRestored.Items) != 1 || moderationRestored.Items[0].URI != pinnedURI || !moderationRestored.HasPinnedPostURI {
		t.Fatalf("moderation-restored page = items:%v hasPinnedPostUri:%v", postResponseURIs(moderationRestored.Items), moderationRestored.HasPinnedPostURI)
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO atproto_blocks (uri, blocker_did, rkey, cid, subject_did, record, created_at)
		VALUES ('at://did:plc:viewer/app.bsky.graph.block/alice', 'did:plc:viewer', 'alice', 'block-cid', $1, '{}'::jsonb, $2)
	`, owner, base.Add(4*time.Minute)); err != nil {
		t.Fatalf("seed block: %v", err)
	}
	blocked := serveProfilePinPage(t, handler, "did:plc:alice", 2)
	if len(blocked.Items) != 0 || blocked.HasPinnedPostURI {
		t.Fatalf("blocked page = items:%v hasPinnedPostUri:%v", postResponseURIs(blocked.Items), blocked.HasPinnedPostURI)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM atproto_blocks WHERE blocker_did = 'did:plc:viewer'`); err != nil {
		t.Fatalf("reverse block: %v", err)
	}
	blockRestored := serveProfilePinPage(t, handler, "did:plc:alice", 2)
	if len(blockRestored.Items) != 1 || blockRestored.Items[0].URI != pinnedURI || !blockRestored.HasPinnedPostURI {
		t.Fatalf("block-restored page = items:%v hasPinnedPostUri:%v", postResponseURIs(blockRestored.Items), blockRestored.HasPinnedPostURI)
	}
}

type profilePinPageResponse struct {
	Items            []api.PostResponse
	Cursor           string
	PinnedPostURI    *string
	HasPinnedPostURI bool
}

func serveProfilePinPage(t *testing.T, handler http.Handler, authorDID string, limit int) profilePinPageResponse {
	t.Helper()
	response := requestProfilePinPage(t, handler, authorDID, limit, "")
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode raw page: %v", err)
	}
	var decoded struct {
		Items         []api.PostResponse `json:"items"`
		Cursor        string             `json:"cursor,omitempty"`
		PinnedPostURI *string            `json:"pinnedPostUri,omitempty"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode page: %v", err)
	}
	_, hasPinnedPostURI := raw["pinnedPostUri"]
	return profilePinPageResponse{
		Items:            decoded.Items,
		Cursor:           decoded.Cursor,
		PinnedPostURI:    decoded.PinnedPostURI,
		HasPinnedPostURI: hasPinnedPostURI,
	}
}

func requestProfilePinPage(
	t *testing.T,
	handler http.Handler,
	authorDID string,
	limit int,
	cursor string,
) *httptest.ResponseRecorder {
	t.Helper()
	path := "/v1/profiles/@" + authorDID + "/posts?limit=" + twoDigits(limit)
	if cursor != "" {
		path += "&cursor=" + url.QueryEscape(cursor)
	}
	req := authedReq(http.MethodGet, path, "", "did:plc:viewer")
	req.SetPathValue("handleOrDid", authorDID)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func assertInvalidProfileCursor(t *testing.T, handler http.Handler, cursor string) {
	t.Helper()
	response := requestProfilePinPage(t, handler, "did:plc:alice", 2, cursor)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("stale cursor status = %d, body = %s", response.Code, response.Body.String())
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode stale cursor error: %v", err)
	}
	if body.Error != "invalid_cursor" {
		t.Fatalf("stale cursor error = %q", body.Error)
	}
}

func postResponseURIs(items []api.PostResponse) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.URI)
	}
	return out
}

func twoDigits(value int) string {
	if value < 10 {
		return "0" + string(rune('0'+value))
	}
	return string(rune('0'+value/10)) + string(rune('0'+value%10))
}
