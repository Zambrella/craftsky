package api_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
)

func TestPDSWriteHandlersEmitObservedOperations(t *testing.T) {
	recorder := observability.NewInMemoryMetricRecorder()
	observer := observability.New(observability.Config{Env: "test", MetricRecorder: recorder})

	profilePDS := &fakePDSForPut{
		getBsky:     func() (map[string]any, error) { return map[string]any{}, nil },
		putBsky:     func(map[string]any) error { return nil },
		putCraftsky: func(map[string]any) error { return nil },
	}
	profileHandler := api.PutMeProfileHandler(
		&fakeStore{row: &api.ProfileRow{DID: "did:plc:alice", CreatedAt: time.Now()}},
		fakeResolver{handleFor: "alice.example"},
		observer.WrapPDSFactory(func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return profilePDS, nil }),
		api.DefaultMediaLimits(),
		nilLogger(),
	)
	req := authedReq(http.MethodPut, "/v1/profiles/me", `{"displayName":"Alice","crafts":["sewing"]}`, "did:plc:alice")
	serveObservedPDSRequest(t, profileHandler, withOAuthSession(req), http.StatusOK)

	postPDS := &fakePDS{createURI: "at://did:plc:alice/social.craftsky.feed.post/post1", createCID: "bafyPost"}
	createPostHandler := api.CreatePostHandler(&fakePostStore{}, observer.WrapPDSFactory(newPDSFactory(postPDS)), fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger())
	serveObservedPDSRequest(t, createPostHandler, withOAuthSession(authedReq(http.MethodPost, "/v1/posts", `{"text":"hello"}`, "did:plc:alice")), http.StatusCreated)

	deletePostHandler := api.DeletePostHandler(observer.WrapPDSFactory(newPDSFactory(&fakePDS{})), nilLogger())
	deletePostReq := withOAuthSession(authedReq(http.MethodDelete, "/v1/posts/did:plc:alice/post1", "", "did:plc:alice"))
	deletePostReq.SetPathValue("did", "did:plc:alice")
	deletePostReq.SetPathValue("rkey", "post1")
	serveObservedPDSRequest(t, deletePostHandler, deletePostReq, http.StatusNoContent)

	blobHandler := api.ImageBlobUploadHandler(observer.WrapPDSFactory(newPDSFactory(&fakePDS{})), api.DefaultMediaLimits(), nilLogger())
	blobReq := withOAuthSession(authedReq(http.MethodPost, "/v1/blobs/images", "jpeg-bytes", "did:plc:alice"))
	blobReq.Header.Set("Content-Type", "image/jpeg")
	serveObservedPDSRequest(t, blobHandler, blobReq, http.StatusOK)

	likeStore := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	likePDS := &fakePDS{createURI: "at://did:plc:alice/social.craftsky.feed.like/like1", createCID: "bafyLike"}
	likeHandler := api.LikePostHandler(likeStore, observer.WrapPDSFactory(newPDSFactory(likePDS)), nilLogger())
	serveObservedPDSRequest(t, likeHandler, withOAuthSession(authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/likes", "", "did:plc:alice")), http.StatusCreated)

	unlikeStore := &fakePostStore{
		target:     &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeLike: &api.InteractionRow{URI: "at://did:plc:alice/social.craftsky.feed.like/like1", DID: "did:plc:alice", Rkey: "like1", CID: "bafyLike", SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1", SubjectCID: "bafyPost"},
	}
	unlikeHandler := api.UnlikePostHandler(unlikeStore, observer.WrapPDSFactory(newPDSFactory(&fakePDS{})), nilLogger())
	serveObservedPDSRequest(t, unlikeHandler, withOAuthSession(authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/likes", "", "did:plc:alice")), http.StatusNoContent)

	repostStore := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	repostPDS := &fakePDS{createURI: "at://did:plc:alice/social.craftsky.feed.repost/repost1", createCID: "bafyRepost"}
	repostHandler := api.RepostPostHandler(repostStore, observer.WrapPDSFactory(newPDSFactory(repostPDS)), nilLogger())
	serveObservedPDSRequest(t, repostHandler, withOAuthSession(authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")), http.StatusCreated)

	unrepostStore := &fakePostStore{
		target:       &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeRepost: &api.InteractionRow{URI: "at://did:plc:alice/social.craftsky.feed.repost/repost1", DID: "did:plc:alice", Rkey: "repost1", CID: "bafyRepost", SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1", SubjectCID: "bafyPost"},
	}
	unrepostHandler := api.UnrepostPostHandler(unrepostStore, observer.WrapPDSFactory(newPDSFactory(&fakePDS{})), nilLogger())
	serveObservedPDSRequest(t, unrepostHandler, withOAuthSession(authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")), http.StatusNoContent)

	followProfiles := &fakeFollowProfileStore{row: &api.ProfileRow{DID: "did:plc:bob", Crafts: []string{}, CreatedAt: time.Now(), IsCraftskyProfile: true}}
	followResolver := fakeResolver{didFor: "did:plc:bob", handleFor: "bob.example"}
	followHandler := api.FollowProfileHandler(
		&fakeFollowGraphStore{},
		followProfiles,
		followResolver,
		observer.WrapPDSFactory(func(context.Context, syntax.DID, string) (auth.PDSClient, error) {
			return &fakeFollowPDS{createURI: "at://did:plc:alice/app.bsky.graph.follow/f1", createCID: "bafyFollow"}, nil
		}),
		nilLogger(),
	)
	followReq := withOAuthSession(authedReq(http.MethodPost, "/v1/profiles/@bob.example/follows", "", "did:plc:alice"))
	followReq.SetPathValue("handleOrDid", "bob.example")
	serveObservedPDSRequest(t, followHandler, followReq, http.StatusOK)

	unfollowHandler := api.UnfollowProfileHandler(
		&fakeFollowGraphStore{active: &api.FollowRow{URI: "at://did:plc:alice/app.bsky.graph.follow/f1", DID: "did:plc:alice", Rkey: "f1", SubjectDID: "did:plc:bob", CreatedAt: time.Now()}},
		followProfiles,
		followResolver,
		observer.WrapPDSFactory(func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return &fakeFollowPDS{}, nil }),
		nilLogger(),
	)
	unfollowReq := withOAuthSession(authedReq(http.MethodDelete, "/v1/profiles/@bob.example/follows", "", "did:plc:alice"))
	unfollowReq.SetPathValue("handleOrDid", "bob.example")
	serveObservedPDSRequest(t, unfollowHandler, unfollowReq, http.StatusOK)

	calls := recorder.Calls()
	for _, want := range []string{
		"oauth.session_resume",
		"profile.put_bsky",
		"profile.put_craftsky",
		"post.create",
		"post.delete",
		"blob.upload",
		"follow.create",
		"follow.delete",
		"like.create",
		"like.delete",
		"repost.create",
		"repost.delete",
	} {
		if !metricCallWithOperation(calls, want) {
			t.Fatalf("PDS metric calls missing operation %q: %#v", want, calls)
		}
	}
	for _, call := range calls {
		if err := observability.ValidateMetricCall(call); err != nil {
			t.Fatalf("PDS metric call failed validation: %v; call=%#v", err, call)
		}
	}
}

func metricCallWithOperation(calls []observability.MetricCall, operation string) bool {
	for _, call := range calls {
		if call.Name == "craftsky_appview_pds_write_duration_seconds" && call.Attributes["operation"] == operation {
			return true
		}
	}
	return false
}

func withOAuthSession(req *http.Request) *http.Request {
	return req.WithContext(middleware.WithOAuthSessionID(req.Context(), "sess-alice"))
}

func serveObservedPDSRequest(t *testing.T, h http.Handler, req *http.Request, wantStatus int) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("%s %s status = %d, want %d; body=%s", req.Method, req.URL.Path, rec.Code, wantStatus, rec.Body.String())
	}
}

func TestPDSWriteHandlerLogsUseBoundedContextWithoutRawIdentitySessionOrContent(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

	profilePDS := &fakePDSForPut{
		getBsky: func() (map[string]any, error) {
			return map[string]any{"displayName": "old secret body", "avatar": "bafkAvatar"}, nil
		},
		putBsky:     func(map[string]any) error { return nil },
		putCraftsky: func(map[string]any) error { return nil },
	}
	profileHandler := api.PutMeProfileHandler(
		&fakeStore{row: &api.ProfileRow{DID: "did:plc:alice", CreatedAt: time.Now()}},
		fakeResolver{handleFor: "alice.example"},
		func(context.Context, syntax.DID, string) (auth.PDSClient, error) { return profilePDS, nil },
		api.DefaultMediaLimits(),
		logger,
	)
	profileReq := loggedWriteReq(http.MethodPut, "/v1/profiles/me", `{"displayName":"secret body","crafts":["lace"]}`)
	serveObservedPDSRequest(t, profileHandler, profileReq, http.StatusOK)

	createPostHandler := api.CreatePostHandler(
		&fakePostStore{},
		newPDSFactory(&fakePDS{createURI: "at://did:plc:alice/social.craftsky.feed.post/post1", createCID: "bafyPost"}),
		fakeResolver{handleFor: "alice.example"},
		api.DefaultMediaLimits(),
		logger,
	)
	serveObservedPDSRequest(t, createPostHandler, loggedWriteReq(http.MethodPost, "/v1/posts", `{"text":"secret body"}`), http.StatusCreated)

	deletePostHandler := api.DeletePostHandler(newPDSFactory(&fakePDS{deleteErr: errors.New("pds down")}), logger)
	deleteReq := loggedWriteReq(http.MethodDelete, "/v1/posts/did:plc:alice/post1", "")
	deleteReq.SetPathValue("did", "did:plc:alice")
	deleteReq.SetPathValue("rkey", "post1")
	serveObservedPDSRequest(t, deletePostHandler, deleteReq, http.StatusBadGateway)

	likeStore := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyTarget"}}
	likeHandler := api.LikePostHandler(likeStore, newPDSFactory(&fakePDS{createErr: errors.New("pds down")}), logger)
	serveObservedPDSRequest(t, likeHandler, loggedWritePostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/likes", ""), http.StatusBadGateway)

	blobHandler := api.ImageBlobUploadHandler(
		newPDSFactory(&fakePDS{uploadResp: &auth.UploadedBlob{
			Raw:  map[string]any{"$type": "blob", "ref": map[string]any{"$link": "bafkBlob"}, "mimeType": "image/jpeg", "size": float64(11)},
			CID:  "bafkBlob",
			MIME: "image/jpeg",
			Size: 11,
		}}),
		api.DefaultMediaLimits(),
		logger,
	)
	blobReq := loggedWriteReq(http.MethodPost, "/v1/blobs/images", "")
	blobReq.Body = io.NopCloser(strings.NewReader("secret media bytes"))
	blobReq.Header.Set("Content-Type", "image/jpeg")
	serveObservedPDSRequest(t, blobHandler, blobReq, http.StatusOK)

	out := logs.String()
	for _, want := range []string{
		`"run_id":"run-test"`,
		`"component":"pds"`,
		`"operation":"profile.put_bsky"`,
		`"operation":"post.create"`,
		`"operation":"post.delete"`,
		`"operation":"like.create"`,
		`"operation":"blob.upload"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("logs missing bounded field %q:\n%s", want, out)
		}
	}
	for _, forbidden := range []string{
		"did:plc:alice",
		"did:plc:bob",
		"session-secret",
		"post1",
		"like1",
		"at://",
		"bafyPost",
		"bafyTarget",
		"bafkBlob",
		"bafkAvatar",
		"secret body",
		"secret media bytes",
		"request=",
		"record=",
		`"request"`,
		`"record"`,
		`"response"`,
		`"blob"`,
	} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("logs contain forbidden value %q:\n%s", forbidden, out)
		}
	}
}

func TestReadHandlerLogsUseBoundedContextWithoutRawIdentityOrContent(t *testing.T) {
	var debugLogs bytes.Buffer
	debugLogger := slog.New(slog.NewJSONHandler(&debugLogs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	var prodLogs bytes.Buffer
	prodLogger := slog.New(slog.NewJSONHandler(&prodLogs, &slog.HandlerOptions{Level: slog.LevelInfo}))

	base := time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)
	root := testPostRow("did:plc:alice", "post1", "secret body", base)
	root.CID = "bafyPost"
	comment := testReplyRow("did:plc:alice", "post1", "secret body", root.URI, root.URI, base)
	comment.CID = "bafyComment"

	getPost := api.GetPostHandler(
		&fakePostStore{one: root, engagement: map[string]api.EngagementSummary{root.URI: {LikeCount: 1}}},
		fakeResolver{handleFor: "alice.example"},
		debugLogger,
	)
	serveObservedPDSRequest(t, getPost, loggedReadPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/post1", ""), http.StatusOK)

	replies := api.ListCommentRepliesHandler(
		&fakePostStore{one: comment, replyCursor: "secret-next"},
		fakeResolver{},
		debugLogger,
	)
	serveObservedPDSRequest(t, replies, loggedReadPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/post1/replies?cursor=secret-cursor", ""), http.StatusOK)

	comments := api.GetPostCommentsHandler(
		&fakePostStore{one: root, commentCursor: "secret-next"},
		fakeResolver{handlesByDID: map[string]syntax.Handle{"did:plc:alice": "alice.example"}},
		debugLogger,
	)
	serveObservedPDSRequest(t, comments, loggedReadPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/post1/comments?cursor=secret-cursor&focus=at://did:plc:bob/social.craftsky.feed.post/reply1", ""), http.StatusOK)

	authorList := api.ListPostsByAuthorHandler(
		&fakePostStore{listCursor: "secret-next"},
		fakeResolver{didFor: "did:plc:alice", handleFor: "alice.example"},
		debugLogger,
	)
	authorReq := loggedReadReq(http.MethodGet, "/v1/profiles/@alice.example/posts?cursor=secret-cursor", "")
	authorReq.SetPathValue("handleOrDid", "alice.example")
	serveObservedPDSRequest(t, authorList, authorReq, http.StatusOK)

	profile := api.GetProfileHandler(
		&fakeStore{row: &api.ProfileRow{
			DID:         "did:plc:alice",
			DisplayName: strPtr("secret body"),
			CreatedAt:   base,
		}},
		fakeResolver{didFor: "did:plc:alice", handleFor: "alice.example"},
		debugLogger,
	)
	profileReq := loggedReadReq(http.MethodGet, "/v1/profiles/@alice.example", "")
	profileReq.SetPathValue("handleOrDid", "alice.example")
	serveObservedPDSRequest(t, profile, profileReq, http.StatusOK)

	prodPost := api.GetPostHandler(
		&fakePostStore{oneErr: errors.New("db failed for did:plc:alice/post1 secret body")},
		fakeResolver{handleFor: "alice.example"},
		prodLogger,
	)
	serveObservedPDSRequest(t, prodPost, loggedReadPostPathReq(http.MethodGet, "/v1/posts/did:plc:alice/post1", ""), http.StatusInternalServerError)

	prodProfile := api.GetProfileHandler(
		&fakeStore{},
		fakeResolver{err: errors.New("resolver failed for alice.example did:plc:alice secret body")},
		prodLogger,
	)
	prodProfileReq := loggedReadReq(http.MethodGet, "/v1/profiles/@alice.example", "")
	prodProfileReq.SetPathValue("handleOrDid", "alice.example")
	serveObservedPDSRequest(t, prodProfile, prodProfileReq, http.StatusBadGateway)

	mutualFollowers := api.GetMutualFollowersHandler(
		&fakeStore{err: errors.New("list failed for did:plc:alice did:plc:viewer secret-cursor")},
		fakeResolver{didFor: "did:plc:alice"},
		prodLogger,
	)
	mutualReq := loggedReadReq(http.MethodGet, "/v1/profiles/@alice.example/mutual-followers?cursor=secret-cursor", "")
	mutualReq.SetPathValue("handleOrDid", "alice.example")
	serveObservedPDSRequest(t, mutualFollowers, mutualReq, http.StatusInternalServerError)

	combined := debugLogs.String() + prodLogs.String()
	for _, want := range []string{
		`"run_id":"run-test"`,
		`"component":"api"`,
		`"operation":"post.get"`,
		`"operation":"post.replies.list"`,
		`"operation":"post.comments.list"`,
		`"operation":"post.author.list"`,
		`"operation":"profile.get"`,
		`"operation":"profile.mutual_followers.list"`,
	} {
		if !strings.Contains(combined, want) {
			t.Fatalf("logs missing bounded field %q:\n%s", want, combined)
		}
	}
	for _, forbidden := range []string{
		"did:plc:alice",
		"did:plc:bob",
		"did:plc:viewer",
		"alice.example",
		"post1",
		"reply1",
		"at://",
		"bafyPost",
		"bafyComment",
		"secret body",
		"secret-cursor",
		"secret-next",
		`"did"`,
		`"viewer_did"`,
		`"profile_did"`,
		`"handle"`,
		`"input"`,
		`"rkey"`,
		`"uri"`,
		`"target_uri"`,
		`"root_uri"`,
		`"cursor"`,
		`"next_cursor"`,
		`"focus"`,
		`"row"`,
		`"response"`,
		`"err"`,
	} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("logs contain forbidden read value %q:\n%s", forbidden, combined)
		}
	}
}

func loggedWriteReq(method, path, body string) *http.Request {
	req := withOAuthSession(authedReq(method, path, body, "did:plc:alice"))
	ctx := ctxkeys.WithRunID(req.Context(), "run-test")
	ctx = middleware.WithOAuthSessionID(ctx, "session-secret")
	return req.WithContext(ctx)
}

func loggedWritePostPathReq(method, urlPath, body string) *http.Request {
	req := loggedWriteReq(method, urlPath, body)
	parts := strings.Split(strings.Trim(urlPath, "/"), "/")
	req.SetPathValue("did", parts[2])
	req.SetPathValue("rkey", parts[3])
	return req
}

func loggedReadReq(method, path, body string) *http.Request {
	req := authedReq(method, path, body, "did:plc:viewer")
	ctx := ctxkeys.WithRunID(req.Context(), "run-test")
	return req.WithContext(ctx)
}

func loggedReadPostPathReq(method, urlPath, body string) *http.Request {
	req := loggedReadReq(method, urlPath, body)
	parts := strings.Split(strings.Trim(urlPath, "/"), "/")
	req.SetPathValue("did", parts[2])
	req.SetPathValue("rkey", parts[3])
	return req
}
