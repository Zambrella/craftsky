package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/testdb"
)

func TestLikeEndpointReconcilesLostResponseWithoutBlindReplay(t *testing.T) {
	ctx := context.Background()
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 24, 0, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	targetDID := syntax.DID("did:plc:bob")
	if _, err := pool.Exec(ctx, `
		INSERT INTO owner_lifecycles(
			owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
		) VALUES($1,'active',1,1,'test',$3,$3,$3),($2,'active',2,1,'test',$3,$3,$3)
	`, owner, targetDID, now); err != nil {
		t.Fatal(err)
	}
	commandStore, err := pdscommands.NewStore(pdscommands.StoreConfig{
		Pool: pool, Now: func() time.Time { return now }, NewUUID: uuid.New,
	})
	if err != nil {
		t.Fatal(err)
	}
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	pds := &endpointCommandPDS{owner: owner, head: "bafy-head-one", records: make(map[syntax.ATURI]endpointCommandRecord), loseResponse: true}
	commands, err := pdscommands.NewSetCommandService(pdscommands.SetCommandServiceConfig{
		Store: commandStore, Lifecycles: lifecycles,
		NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) {
			return endpointCommandBoundary{client: pds}, nil
		},
		Now: func() time.Time { return now }, Sleep: func(time.Duration) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	store := &fakePostStore{target: &api.PostTargetRef{
		URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost",
	}}
	handler := api.CommandLikePostHandler(store, commands, nilLogger())
	key := "018f4d5c-7a61-7d40-a1a2-0123456789ab"

	firstRequest := commandPostRequest(http.MethodPost, key)
	firstResponse := httptest.NewRecorder()
	handler.ServeHTTP(firstResponse, firstRequest)
	if firstResponse.Code != http.StatusAccepted || firstResponse.Body.String() != "{\"status\":\"ambiguous\"}\n" ||
		firstResponse.Header().Get("Retry-After") != "1" || pds.applyCalls != 1 {
		t.Fatalf("first response status=%d headers=%v body=%q applyCalls=%d", firstResponse.Code, firstResponse.Header(), firstResponse.Body.String(), pds.applyCalls)
	}
	// Tap can remove the target before the PDS command's lost response is
	// recovered. The retry must use the frozen subject rather than the index.
	store.targetErr = api.ErrPostNotFound

	retryRequest := commandPostRequest(http.MethodPost, key)
	retryResponse := httptest.NewRecorder()
	handler.ServeHTTP(retryResponse, retryRequest)
	if retryResponse.Code != http.StatusCreated || pds.applyCalls != 1 {
		t.Fatalf("retry response status=%d body=%q applyCalls=%d", retryResponse.Code, retryResponse.Body.String(), pds.applyCalls)
	}
	var response api.InteractionWriteResponse
	if err := json.Unmarshal(retryResponse.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.URI != pds.createdURI.String() || response.CID == "" || response.Subject.URI != store.target.URI {
		t.Fatalf("reconciled response = %+v", response)
	}

	replayRequest := commandPostRequest(http.MethodPost, key)
	replayResponse := httptest.NewRecorder()
	handler.ServeHTTP(replayResponse, replayRequest)
	if replayResponse.Code != http.StatusCreated || replayResponse.Body.String() != retryResponse.Body.String() || pds.applyCalls != 1 {
		t.Fatalf("terminal replay status=%d body=%q applyCalls=%d", replayResponse.Code, replayResponse.Body.String(), pds.applyCalls)
	}
	otherPath := commandPostRequest(http.MethodPost, key)
	otherPath.SetPathValue("rkey", "other-post")
	conflict := httptest.NewRecorder()
	handler.ServeHTTP(conflict, otherPath)
	if conflict.Code != http.StatusConflict || pds.applyCalls != 1 {
		t.Fatalf("key reused on another target status=%d body=%q applyCalls=%d", conflict.Code, conflict.Body.String(), pds.applyCalls)
	}
}

func TestCreatePostEndpointReconcilesLostResponseWithoutAllocatingAnotherIdentity(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 24, 0, 30, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	seedCommandOwners(t, pool, now, owner)
	pds := &endpointCommandPDS{owner: owner, head: "bafy-head-one", records: make(map[syntax.ATURI]endpointCommandRecord), loseResponse: true}
	commandStore, err := pdscommands.NewStore(pdscommands.StoreConfig{Pool: pool, Now: func() time.Time { return now }, NewUUID: uuid.New})
	if err != nil {
		t.Fatal(err)
	}
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	allocated := 0
	commands, err := pdscommands.NewAppendCommandService(pdscommands.AppendCommandServiceConfig{
		Store: commandStore, Lifecycles: lifecycles,
		NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) {
			return endpointCommandBoundary{client: pds}, nil
		},
		Now: func() time.Time { return now },
		NewRecordKey: func() (syntax.RecordKey, error) {
			allocated++
			return syntax.RecordKey("3append" + string(rune('a'+allocated))), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := api.CreatePostHandler(
		&fakePostStore{}, nil, fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger(),
		api.CreatePostHandlerOptions{Commands: commands},
	)
	key := "018f4d5c-7a61-7d40-a1a2-0123456789ad"
	request := func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/v1/posts", strings.NewReader(`{"text":"hello","sponsored":false}`))
		req.Header.Set("Idempotency-Key", key)
		requestCtx := middleware.WithDID(req.Context(), owner)
		requestCtx = middleware.WithOwnerGeneration(requestCtx, 1)
		requestCtx = middleware.WithOAuthSessionID(requestCtx, "session-alice")
		return req.WithContext(requestCtx)
	}

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, request())
	if first.Code != http.StatusAccepted || first.Body.String() != "{\"status\":\"ambiguous\"}\n" || pds.applyCalls != 1 {
		t.Fatalf("first status=%d body=%q applyCalls=%d", first.Code, first.Body.String(), pds.applyCalls)
	}
	retry := httptest.NewRecorder()
	handler.ServeHTTP(retry, request())
	if retry.Code != http.StatusCreated || pds.applyCalls != 1 {
		t.Fatalf("retry status=%d body=%q applyCalls=%d", retry.Code, retry.Body.String(), pds.applyCalls)
	}
	var accepted api.PostResponse
	if err := json.Unmarshal(retry.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if accepted.URI != pds.createdURI.String() || accepted.Rkey != pds.createdURI.RecordKey().String() || accepted.CID == "" {
		t.Fatalf("accepted post = %+v createdURI=%s", accepted, pds.createdURI)
	}
	replay := httptest.NewRecorder()
	handler.ServeHTTP(replay, request())
	if replay.Code != http.StatusCreated || replay.Body.String() != retry.Body.String() || pds.applyCalls != 1 || allocated != 1 {
		t.Fatalf("replay status=%d body=%q applyCalls=%d allocated=%d", replay.Code, replay.Body.String(), pds.applyCalls, allocated)
	}
}

func TestPutProfileEndpointReconcilesBothFixedRecordsAfterLostResponse(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 24, 0, 40, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	seedCommandOwners(t, pool, now, owner)
	bskyURI := syntax.ATURI("at://did:plc:alice/app.bsky.actor.profile/self")
	craftskyURI := syntax.ATURI("at://did:plc:alice/social.craftsky.actor.profile/self")
	pds := &endpointCommandPDS{
		owner: owner, head: "bafy-head-one", loseResponse: true,
		records: map[syntax.ATURI]endpointCommandRecord{
			bskyURI:     {cid: "bafy-old-bsky", value: map[string]any{"$type": "app.bsky.actor.profile", "displayName": "Before"}},
			craftskyURI: {cid: "bafy-old-craftsky", value: map[string]any{"$type": "social.craftsky.actor.profile", "crafts": []string{"sewing"}}},
		},
	}
	commandStore, err := pdscommands.NewStore(pdscommands.StoreConfig{
		Pool: pool, Now: func() time.Time { return now }, NewUUID: uuid.New,
	})
	if err != nil {
		t.Fatal(err)
	}
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	commands, err := pdscommands.NewCompoundCommandService(pdscommands.CompoundCommandServiceConfig{
		Store: commandStore, Lifecycles: lifecycles,
		NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) {
			return endpointCommandBoundary{client: pds}, nil
		},
		Now: func() time.Time { return now }, Sleep: func(time.Duration) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := api.PutMeProfileHandler(
		&fakeStore{}, fakeResolver{handleFor: "alice.example"}, nil,
		api.DefaultMediaLimits(), nilLogger(), commands,
	)
	key := "018f4d5c-7a61-7d40-a1a2-0123456789ac"
	request := func() *http.Request {
		req := httptest.NewRequest(http.MethodPut, "/v1/profiles/me", strings.NewReader(`{"displayName":"After","crafts":["quilting"]}`))
		req.Header.Set("Idempotency-Key", key)
		requestCtx := middleware.WithDID(req.Context(), owner)
		requestCtx = middleware.WithOwnerGeneration(requestCtx, 1)
		requestCtx = middleware.WithOAuthSessionID(requestCtx, "session-alice")
		return req.WithContext(requestCtx)
	}

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, request())
	if first.Code != http.StatusAccepted || first.Body.String() != "{\"status\":\"ambiguous\"}\n" || pds.applyCalls != 1 {
		t.Fatalf("first status=%d body=%q applyCalls=%d", first.Code, first.Body.String(), pds.applyCalls)
	}
	if len(pds.lastWrites) != 2 || pds.lastWrites[0].Collection != "app.bsky.actor.profile" ||
		pds.lastWrites[0].RKey != "self" || pds.lastWrites[1].Collection != "social.craftsky.actor.profile" ||
		pds.lastWrites[1].RKey != "self" {
		t.Fatalf("ordered compound writes = %+v", pds.lastWrites)
	}

	retry := httptest.NewRecorder()
	handler.ServeHTTP(retry, request())
	if retry.Code != http.StatusOK || pds.applyCalls != 1 {
		t.Fatalf("retry status=%d body=%q applyCalls=%d", retry.Code, retry.Body.String(), pds.applyCalls)
	}
	var profile api.ProfileResponse
	if err := json.Unmarshal(retry.Body.Bytes(), &profile); err != nil {
		t.Fatal(err)
	}
	if profile.DisplayName == nil || *profile.DisplayName != "After" || len(profile.Crafts) != 1 || profile.Crafts[0] != "quilting" {
		t.Fatalf("profile response = %+v", profile)
	}
}

func TestDeletePostEndpointReconcilesLostResponseToEmptyNoContent(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 24, 0, 45, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	seedCommandOwners(t, pool, now, owner)
	uri := syntax.ATURI("at://did:plc:alice/social.craftsky.feed.post/3deletepost")
	pds := &endpointCommandPDS{
		owner: owner, head: "bafy-head-one", loseResponse: true,
		records: map[syntax.ATURI]endpointCommandRecord{
			uri: {cid: postDeleteTestCID, value: map[string]any{"$type": "social.craftsky.feed.post", "text": "delete me"}},
		},
	}
	commandStore, err := pdscommands.NewStore(pdscommands.StoreConfig{Pool: pool, Now: func() time.Time { return now }, NewUUID: uuid.New})
	if err != nil {
		t.Fatal(err)
	}
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	commands, err := pdscommands.NewAddressedCommandService(pdscommands.AddressedCommandServiceConfig{
		Store: commandStore, Lifecycles: lifecycles,
		NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) {
			return endpointCommandBoundary{client: pds}, nil
		},
		Now: func() time.Time { return now }, Sleep: func(time.Duration) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := api.CommandDeletePostHandler(commands, nilLogger())
	key := "018f4d5c-7a61-7d40-a1a2-0123456789ae"
	request := func() *http.Request {
		req := httptest.NewRequest(http.MethodDelete, "/v1/posts/did:plc:alice/3deletepost", nil)
		req.SetPathValue("did", owner.String())
		req.SetPathValue("rkey", "3deletepost")
		req.Header.Set("Idempotency-Key", key)
		req.Header.Set("If-Match", postDeleteTestCID)
		requestCtx := middleware.WithDID(req.Context(), owner)
		requestCtx = middleware.WithOwnerGeneration(requestCtx, 1)
		requestCtx = middleware.WithOAuthSessionID(requestCtx, "session-alice")
		return req.WithContext(requestCtx)
	}

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, request())
	if first.Code != http.StatusAccepted || first.Body.String() != "{\"status\":\"ambiguous\"}\n" || pds.applyCalls != 1 {
		t.Fatalf("first status=%d body=%q applyCalls=%d", first.Code, first.Body.String(), pds.applyCalls)
	}
	retry := httptest.NewRecorder()
	handler.ServeHTTP(retry, request())
	if retry.Code != http.StatusNoContent || retry.Body.Len() != 0 || pds.applyCalls != 1 {
		t.Fatalf("retry status=%d body=%q applyCalls=%d", retry.Code, retry.Body.String(), pds.applyCalls)
	}
	replay := httptest.NewRecorder()
	handler.ServeHTTP(replay, request())
	if replay.Code != http.StatusNoContent || replay.Body.Len() != 0 || pds.applyCalls != 1 {
		t.Fatalf("replay status=%d body=%q applyCalls=%d", replay.Code, replay.Body.String(), pds.applyCalls)
	}
}

func TestFollowEndpointNoOpsExistingExternalMatchAndReplaysExactProfile(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 24, 1, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	target := syntax.DID("did:plc:bob")
	seedCommandOwners(t, pool, now, owner, target)
	pds := &endpointCommandPDS{owner: owner, head: "bafy-head-one", records: map[syntax.ATURI]endpointCommandRecord{
		"at://did:plc:alice/app.bsky.graph.follow/3aaaaaaaaaaa2": {
			cid: "bafy-external-follow",
			value: map[string]any{
				"$type": "app.bsky.graph.follow", "subject": target.String(), "createdAt": now.Format(time.RFC3339),
			},
		},
	}}
	commands := newEndpointSetCommands(t, pool, now, pds)
	followerCount := 4
	profiles := &fakeFollowProfileStore{row: &api.ProfileRow{
		DID: target.String(), Crafts: []string{}, CreatedAt: now, IsCraftskyProfile: true,
		ViewerIsFollowing: false, FollowerCount: &followerCount,
	}}
	handler := api.CommandFollowProfileHandler(
		profiles,
		fakeResolver{didFor: target, handleFor: "bob.example"},
		commands,
		nilLogger(),
	)
	key := "018f4d5c-7a61-7d40-a1a2-0123456789ab"

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, commandFollowRequest(http.MethodPost, key))
	if first.Code != http.StatusOK || pds.applyCalls != 0 {
		t.Fatalf("first status=%d body=%q applyCalls=%d", first.Code, first.Body.String(), pds.applyCalls)
	}
	var profile api.ProfileResponse
	if err := json.Unmarshal(first.Body.Bytes(), &profile); err != nil {
		t.Fatal(err)
	}
	if !profile.ViewerIsFollowing || profile.FollowerCount == nil || *profile.FollowerCount != 5 {
		t.Fatalf("profile response = %+v", profile)
	}

	profiles.row.DisplayName = stringPointer("changed after acceptance")
	replay := httptest.NewRecorder()
	handler.ServeHTTP(replay, commandFollowRequest(http.MethodPost, key))
	if replay.Code != http.StatusOK || replay.Body.String() != first.Body.String() || pds.applyCalls != 0 {
		t.Fatalf("replay status=%d body=%q applyCalls=%d, want exact %q", replay.Code, replay.Body.String(), pds.applyCalls, first.Body.String())
	}
}

func TestUnfollowEndpointAtomicallyRemovesAllValidMatchesAndReturnsProfile(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	now := time.Date(2026, 9, 24, 2, 0, 0, 0, time.UTC)
	owner := syntax.DID("did:plc:alice")
	target := syntax.DID("did:plc:bob")
	seedCommandOwners(t, pool, now, owner, target)
	matchingRecord := func(createdAt time.Time) map[string]any {
		return map[string]any{
			"$type": "app.bsky.graph.follow", "subject": target.String(), "createdAt": createdAt.Format(time.RFC3339),
		}
	}
	pds := &endpointCommandPDS{owner: owner, head: "bafy-head-one", records: map[syntax.ATURI]endpointCommandRecord{
		"at://did:plc:alice/app.bsky.graph.follow/3aaaaaaaaaaa2": {cid: "bafy-follow-one", value: matchingRecord(now)},
		"at://did:plc:alice/app.bsky.graph.follow/3aaaaaaaaaaa3": {cid: "bafy-follow-two", value: matchingRecord(now.Add(time.Second))},
		"at://did:plc:alice/app.bsky.graph.follow/3aaaaaaaaaaa4": {cid: "bafy-follow-other", value: map[string]any{"$type": "app.bsky.graph.follow", "subject": "did:plc:carol", "createdAt": now.Format(time.RFC3339)}},
	}}
	commands := newEndpointSetCommands(t, pool, now, pds)
	followerCount := 2
	handler := api.CommandUnfollowProfileHandler(
		&fakeFollowProfileStore{row: &api.ProfileRow{
			DID: target.String(), Crafts: []string{}, CreatedAt: now, IsCraftskyProfile: true,
			ViewerIsFollowing: true, FollowerCount: &followerCount,
		}},
		fakeResolver{didFor: target, handleFor: "bob.example"},
		commands,
		nilLogger(),
	)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, commandFollowRequest(http.MethodDelete, "018f4d5c-7a61-7d40-a1a2-0123456789ac"))
	if response.Code != http.StatusOK || pds.applyCalls != 1 || len(pds.lastWrites) != 2 {
		t.Fatalf("status=%d body=%q applyCalls=%d writes=%+v", response.Code, response.Body.String(), pds.applyCalls, pds.lastWrites)
	}
	for _, write := range pds.lastWrites {
		if write.Action != "delete" || write.Collection != "app.bsky.graph.follow" || (write.RKey != "3aaaaaaaaaaa2" && write.RKey != "3aaaaaaaaaaa3") {
			t.Fatalf("unexpected atomic write: %+v", write)
		}
	}
	if _, ok := pds.records["at://did:plc:alice/app.bsky.graph.follow/3aaaaaaaaaaa4"]; !ok || len(pds.records) != 1 {
		t.Fatalf("remaining authoritative records = %+v", pds.records)
	}
	var profile api.ProfileResponse
	if err := json.Unmarshal(response.Body.Bytes(), &profile); err != nil {
		t.Fatal(err)
	}
	if profile.ViewerIsFollowing || profile.FollowerCount == nil || *profile.FollowerCount != 1 {
		t.Fatalf("profile response = %+v", profile)
	}
}

func seedCommandOwners(t *testing.T, pool *pgxpool.Pool, now time.Time, owners ...syntax.DID) {
	t.Helper()
	for index, owner := range owners {
		if _, err := pool.Exec(context.Background(), `
			INSERT INTO owner_lifecycles(
				owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at
			) VALUES($1,'active',$2,1,'test',$3,$3,$3)
		`, owner, index+1, now); err != nil {
			t.Fatal(err)
		}
	}
}

func newEndpointSetCommands(t *testing.T, pool *pgxpool.Pool, now time.Time, pds auth.PDSClient) *pdscommands.SetCommandService {
	t.Helper()
	commandStore, err := pdscommands.NewStore(pdscommands.StoreConfig{
		Pool: pool, Now: func() time.Time { return now }, NewUUID: uuid.New,
	})
	if err != nil {
		t.Fatal(err)
	}
	fencer, err := ownerlifecycle.NewFencer(pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycles, err := ownerlifecycle.NewStore(pool, fencer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	commands, err := pdscommands.NewSetCommandService(pdscommands.SetCommandServiceConfig{
		Store: commandStore, Lifecycles: lifecycles,
		NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) {
			return endpointCommandBoundary{client: pds}, nil
		},
		Now: func() time.Time { return now }, Sleep: func(time.Duration) {},
	})
	if err != nil {
		t.Fatal(err)
	}
	return commands
}

type endpointCommandBoundary struct{ client auth.PDSClient }

func (boundary endpointCommandBoundary) WithActiveEffects(
	ctx context.Context,
	_ []ownerlifecycle.ExpectedOwner,
	operation auth.ActiveEffectPDSOperation,
) error {
	return operation(ctx, boundary.client)
}

type endpointCommandRecord struct {
	cid   syntax.CID
	value any
}

type endpointCommandPDS struct {
	owner        syntax.DID
	head         syntax.CID
	records      map[syntax.ATURI]endpointCommandRecord
	loseResponse bool
	applyCalls   int
	createdURI   syntax.ATURI
	lastWrites   []auth.RepositoryWrite
}

func (pds *endpointCommandPDS) GetRecord(_ context.Context, _ syntax.DID, collection, rkey string, out any) (string, error) {
	uri := syntax.ATURI("at://" + pds.owner.String() + "/" + collection + "/" + rkey)
	record, ok := pds.records[uri]
	if !ok {
		return "", auth.ErrRecordNotFound
	}
	raw, _ := json.Marshal(record.value)
	if err := json.Unmarshal(raw, out); err != nil {
		return "", err
	}
	return record.cid.String(), nil
}

func (*endpointCommandPDS) PutRecord(context.Context, syntax.DID, string, string, any) error {
	return errors.New("unused")
}
func (*endpointCommandPDS) CreateRecord(context.Context, syntax.DID, string, any) (syntax.ATURI, syntax.CID, error) {
	return "", "", errors.New("unused")
}
func (*endpointCommandPDS) DeleteRecord(context.Context, syntax.DID, string, string) error {
	return errors.New("unused")
}
func (*endpointCommandPDS) UploadBlob(context.Context, string, []byte) (*auth.UploadedBlob, error) {
	return nil, errors.New("unused")
}

func (pds *endpointCommandPDS) ListRecords(_ context.Context, _ syntax.DID, collection, cursor string, _ int) ([]auth.PDSRecord, string, error) {
	if cursor != "" {
		return nil, "", nil
	}
	var records []auth.PDSRecord
	for uri, record := range pds.records {
		parsed, _ := syntax.ParseATURI(uri.String())
		if parsed.Collection().String() == collection {
			records = append(records, auth.PDSRecord{URI: uri, CID: record.cid, Value: record.value})
		}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].URI < records[j].URI })
	return records, "", nil
}

func (pds *endpointCommandPDS) LatestCommit(context.Context, syntax.DID) (syntax.CID, error) {
	return pds.head, nil
}

func (pds *endpointCommandPDS) ApplyWrites(_ context.Context, _ syntax.DID, head syntax.CID, writes []auth.RepositoryWrite) error {
	if head != pds.head {
		return auth.ErrRepositorySwapConflict
	}
	pds.applyCalls++
	pds.lastWrites = append([]auth.RepositoryWrite(nil), writes...)
	for _, write := range writes {
		uri := syntax.ATURI("at://" + pds.owner.String() + "/" + write.Collection.String() + "/" + write.RKey.String())
		if write.Action == "delete" {
			delete(pds.records, uri)
			continue
		}
		pds.createdURI = uri
		pds.records[uri] = endpointCommandRecord{cid: syntax.CID("bafy-created-like"), value: write.Record}
	}
	pds.head = "bafy-head-two"
	if pds.loseResponse {
		pds.loseResponse = false
		return errors.New("response lost")
	}
	return nil
}

func (*endpointCommandPDS) DeleteRecordWithRepositorySwap(
	context.Context,
	syntax.DID,
	syntax.NSID,
	syntax.RecordKey,
	syntax.CID,
	syntax.CID,
) error {
	return auth.ErrApplyWritesUnsupported
}
