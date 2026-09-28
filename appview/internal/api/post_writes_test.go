package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/relationships"
	"social.craftsky/appview/internal/video"
	"strings"
	"testing"
	"time"
)

func TestDirectedPostCreatesRejectBlockedPairBeforePDSWrite(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		operation relationships.Operation
		handler   func(*fakePostStore, *fakePostEffectState) http.Handler
		request   func() *http.Request
	}{
		{
			name:      "like",
			operation: relationships.OperationLikeCreate,
			handler: func(store *fakePostStore, pds *fakePostEffectState) http.Handler {
				return api.LikePostHandler(store, newPDSEffectsFactory(pds), nilLogger())
			},
			request: func() *http.Request {
				return authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/likes", "", "did:plc:alice")
			},
		},
		{
			name:      "repost",
			operation: relationships.OperationRepostCreate,
			handler: func(store *fakePostStore, pds *fakePostEffectState) http.Handler {
				return api.RepostPostHandler(store, newPDSEffectsFactory(pds), nilLogger())
			},
			request: func() *http.Request {
				return authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
			},
		},
		{
			name:      "reply",
			operation: relationships.OperationReplyCreate,
			handler: func(store *fakePostStore, pds *fakePostEffectState) http.Handler {
				return api.CreatePostHandler(store, newPDSEffectsFactory(pds), fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger())
			},
			request: func() *http.Request {
				body := `{"text":"reply","sponsored":false,"reply":{"root":{"uri":"at://did:plc:bob/social.craftsky.feed.post/root","cid":"bafyRoot"},"parent":{"uri":"at://did:plc:bob/social.craftsky.feed.post/post1","cid":"bafyPost"}}}`
				return authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
			},
		},
		{
			name:      "quote",
			operation: relationships.OperationQuoteCreate,
			handler: func(store *fakePostStore, pds *fakePostEffectState) http.Handler {
				return api.CreatePostHandler(store, newPDSEffectsFactory(pds), fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger())
			},
			request: func() *http.Request {
				body := `{"text":"quote","sponsored":false,"embed":{"quote":{"uri":"at://did:plc:bob/social.craftsky.feed.post/post1","cid":"bafyPost"}}}`
				return authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
			},
		},
		{
			name:      "mention",
			operation: relationships.OperationMentionCreate,
			handler: func(store *fakePostStore, pds *fakePostEffectState) http.Handler {
				return api.CreatePostHandler(store, newPDSEffectsFactory(pds), fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger())
			},
			request: func() *http.Request {
				body := `{"text":"@bob.example","sponsored":false,"facets":[{"index":{"byteStart":0,"byteEnd":12},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:bob"}]}]}`
				return authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pds := &fakePostEffectState{}
			store := &fakePostStore{
				authorizationErr: api.ErrInteractionBlocked,
				target:           &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
				shareTarget:      &api.ShareTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
			}
			rr := httptest.NewRecorder()
			test.handler(store, pds).ServeHTTP(rr, test.request())

			if rr.Code != http.StatusForbidden {
				t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
			}
			var env envelope.Error
			_ = json.NewDecoder(rr.Body).Decode(&env)
			if env.Error != "interaction_blocked" {
				t.Fatalf("error = %q, want interaction_blocked", env.Error)
			}
			if pds.createCalls != 0 {
				t.Fatalf("PDS CreateRecord calls = %d, want 0", pds.createCalls)
			}
			if len(store.authorizationCalls) != 1 || store.authorizationCalls[0] != (authorizationCall{
				Actor: "did:plc:alice", Subject: "did:plc:bob", Operation: test.operation,
			}) {
				t.Fatalf("authorization calls = %+v", store.authorizationCalls)
			}
		})
	}
}

func TestCreatePost_HappyPath(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	store := &fakePostStore{}
	resolver := fakeResolver{handleFor: "alice.example"}
	h := api.CreatePostHandler(store, newPDSEffectsFactory(pds), resolver, api.DefaultMediaLimits(), nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":"hello","sponsored":true}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.PostResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Text != "hello" || resp.URI == "" || resp.CID == "" {
		t.Errorf("resp = %+v", resp)
	}
	if !resp.Sponsored {
		t.Error("response sponsored = false, want true")
	}
	if resp.Rkey != "rkSrv" {
		t.Errorf("rkey not derived from PDS uri: %q", resp.Rkey)
	}
	if resp.Author.Handle != "alice.example" {
		t.Errorf("author.handle = %q", resp.Author.Handle)
	}
	if resp.LikeCount != 0 || resp.RepostCount != 0 || resp.ReplyCount != 0 || resp.ViewerHasLiked || resp.ViewerHasReposted {
		t.Errorf("engagement defaults = %+v", resp)
	}

	body, _ := pds.lastCreateRec.(map[string]any)
	if body["$type"] != "social.craftsky.feed.post" {
		t.Errorf("missing/wrong $type: %v", body["$type"])
	}
	if _, ok := body["createdAt"].(string); !ok {
		t.Errorf("createdAt missing or non-string: %v", body["createdAt"])
	}
	if body["sponsored"] != true {
		t.Errorf("sponsored = %#v, want true", body["sponsored"])
	}
}

func TestCreatePost_LanguagesReachPDSAndSyntheticResponse(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	handler := api.CreatePostHandler(
		&fakePostStore{},
		newPDSEffectsFactory(pds),
		fakeResolver{handleFor: "alice.example"},
		api.DefaultMediaLimits(),
		nilLogger(),
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(
		response,
		authedReq(
			http.MethodPost,
			"/v1/posts",
			`{"text":"hello","sponsored":false,"langs":["en","fr"]}`,
			"did:plc:alice",
		),
	)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	record, ok := pds.lastCreateRec.(map[string]any)
	if !ok {
		t.Fatalf("PDS record type = %T", pds.lastCreateRec)
	}
	langs, ok := record["langs"].([]string)
	if !ok || len(langs) != 2 || langs[0] != "en" || langs[1] != "fr" {
		t.Fatalf("PDS langs = %#v", record["langs"])
	}
	var post api.PostResponse
	if err := json.NewDecoder(response.Body).Decode(&post); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(post.Langs) != 2 || post.Langs[0] != "en" || post.Langs[1] != "fr" {
		t.Fatalf("response langs = %v", post.Langs)
	}

	invalidPDS := &fakePostEffectState{}
	invalidHandler := api.CreatePostHandler(
		&fakePostStore{},
		newPDSEffectsFactory(invalidPDS),
		fakeResolver{handleFor: "alice.example"},
		api.DefaultMediaLimits(),
		nilLogger(),
	)
	invalidResponse := httptest.NewRecorder()
	invalidHandler.ServeHTTP(
		invalidResponse,
		authedReq(
			http.MethodPost,
			"/v1/posts",
			`{"text":"hello","sponsored":false,"langs":["en","en"]}`,
			"did:plc:alice",
		),
	)
	if invalidResponse.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d, body = %s", invalidResponse.Code, invalidResponse.Body.String())
	}
	if invalidPDS.createCalls != 0 {
		t.Fatalf("invalid request reached PDS %d times", invalidPDS.createCalls)
	}
}

func TestCreatePost_MalformedBody_400(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	h := api.CreatePostHandler(&fakePostStore{}, newPDSEffectsFactory(pds), fakeResolver{}, api.DefaultMediaLimits(), nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{not json`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	assertAPIError(t, rr, http.StatusBadRequest, "malformed_body")
	if pds.createCalls != 0 {
		t.Fatalf("PDS writes = %d, want 0", pds.createCalls)
	}
}

func TestCreatePost_VideoProofVerifiedBeforePDSWrite(t *testing.T) {
	t.Parallel()
	const videoCID = "bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"
	const body = `{"text":"video post","sponsored":false,"embed":{"video":{"jobId":"job-1","blob":{"$type":"blob","ref":{"$link":"` + videoCID + `"},"mimeType":"video/mp4","size":123}}}}`

	tests := []struct {
		name       string
		verifyErr  error
		wantStatus int
		wantCode   string
	}{
		{name: "verified", wantStatus: http.StatusCreated},
		{name: "rejected", verifyErr: &video.VerificationError{Kind: video.VerificationRejected}, wantStatus: http.StatusUnprocessableEntity, wantCode: "video_verification_failed"},
		{name: "unavailable", verifyErr: &video.VerificationError{Kind: video.VerificationUnavailable}, wantStatus: http.StatusBadGateway, wantCode: "video_service_unavailable"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			pds := &fakePostEffectState{}
			verifier := &fakeVideoCompletionVerifier{err: test.verifyErr}
			handler := api.CreatePostHandler(
				&fakePostStore{},
				newPDSEffectsFactory(pds),
				fakeResolver{handleFor: "alice.example"},
				api.DefaultMediaLimits(),
				nilLogger(),
				api.CreatePostHandlerOptions{VideoCompletionVerifier: verifier},
			)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice"))

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			if verifier.calls != 1 || verifier.owner != syntax.DID("did:plc:alice") || verifier.jobID != "job-1" {
				t.Fatalf("verifier calls=%d owner=%q job=%q", verifier.calls, verifier.owner, verifier.jobID)
			}
			if verifier.blob != (video.Blob{CID: syntax.CID(videoCID), MIMEType: "video/mp4", Size: 123}) {
				t.Fatalf("submitted blob = %+v", verifier.blob)
			}
			if test.verifyErr == nil {
				if pds.createCalls != 1 {
					t.Fatalf("PDS writes = %d, want 1", pds.createCalls)
				}
				return
			}
			if pds.createCalls != 0 {
				t.Fatalf("rejected proof reached PDS %d times", pds.createCalls)
			}
			if !strings.Contains(recorder.Body.String(), test.wantCode) || strings.Contains(recorder.Body.String(), videoCID) {
				t.Fatalf("unsafe or incorrect error body: %s", recorder.Body.String())
			}
		})
	}
}

func TestCreatePost_VerifiedVideoWritesStandardEmbed(t *testing.T) {
	t.Parallel()
	const submittedCID = "bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"
	const verifiedCID = "bafkreigxxxkul4e5rjz4fomqgn6ieeoxbcqeztmxjbrhnbpe7r44ya4ahe"
	pds := &fakePostEffectState{}
	verifier := &fakeVideoCompletionVerifier{verified: video.Blob{
		CID: syntax.CID(verifiedCID), MIMEType: "video/mp4", Size: 321,
	}}
	handler := api.CreatePostHandler(
		&fakePostStore{},
		newPDSEffectsFactory(pds),
		fakeResolver{handleFor: "alice.example"},
		api.DefaultMediaLimits(),
		nilLogger(),
		api.CreatePostHandlerOptions{VideoCompletionVerifier: verifier},
	)
	body := `{"text":"video post","sponsored":false,"embed":{"video":{"jobId":"private-job-id","blob":{"$type":"blob","ref":{"$link":"` + submittedCID + `"},"mimeType":"video/mp4","size":123},"alt":"Hands knitting","aspectRatio":{"width":16,"height":9}}}}`
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice"))

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	record, ok := pds.lastCreateRec.(map[string]any)
	if !ok {
		t.Fatalf("PDS record type = %T", pds.lastCreateRec)
	}
	embed, ok := record["embed"].(map[string]any)
	if !ok || embed["$type"] != "app.bsky.embed.video" || embed["alt"] != "Hands knitting" {
		t.Fatalf("embed = %#v", record["embed"])
	}
	blob, ok := embed["video"].(map[string]any)
	if !ok || blob["mimeType"] != "video/mp4" || blob["size"] != int64(321) {
		t.Fatalf("video blob = %#v", embed["video"])
	}
	ref, _ := blob["ref"].(map[string]any)
	if ref["$link"] != verifiedCID {
		t.Fatalf("video CID = %v, want verifier result %s", ref["$link"], verifiedCID)
	}
	if _, exists := embed["jobId"]; exists || strings.Contains(mustJSON(t, record), "private-job-id") {
		t.Fatalf("job proof leaked into PDS record: %#v", record)
	}
	ratio, _ := embed["aspectRatio"].(map[string]any)
	if ratio["width"] != 16 || ratio["height"] != 9 {
		t.Fatalf("aspect ratio = %#v", ratio)
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal JSON: %v", err)
	}
	return string(encoded)
}

func TestCreatePost_TextEmpty_422(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	h := api.CreatePostHandler(&fakePostStore{}, newPDSEffectsFactory(pds), fakeResolver{}, api.DefaultMediaLimits(), nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":"","sponsored":false}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	assertAPIError(t, rr, http.StatusUnprocessableEntity, "validation_failed")
	if pds.createCalls != 0 {
		t.Fatalf("PDS writes = %d, want 0", pds.createCalls)
	}
}

func TestCreatePost_PDSWriteFailed_502(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{createErr: errors.New("pds rejected the create")}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSEffectsFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":"hi","sponsored":false}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	// The response envelope should distinguish write-failed from unavailable.
	body := rr.Body.String()
	if !strings.Contains(body, "pds_write_failed") {
		t.Errorf("expected pds_write_failed in body, got: %s", body)
	}
}

func TestCreatePost_MissingVerifiedVideoBlobReturnsRecoverableError(t *testing.T) {
	t.Parallel()
	const videoCID = "bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"
	pds := &fakePostEffectState{createErr: &atclient.APIError{
		StatusCode: http.StatusBadRequest,
		Name:       "BlobNotFound",
	}}
	handler := api.CreatePostHandler(
		&fakePostStore{},
		newPDSEffectsFactory(pds),
		fakeResolver{handleFor: "alice.example"},
		api.DefaultMediaLimits(),
		nilLogger(),
		api.CreatePostHandlerOptions{VideoCompletionVerifier: &fakeVideoCompletionVerifier{}},
	)
	body := `{"text":"video post","sponsored":false,"embed":{"video":{"jobId":"job-1","blob":{"$type":"blob","ref":{"$link":"` + videoCID + `"},"mimeType":"video/mp4","size":123}}}}`
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice"))

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response envelope.Error
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error != "video_blob_missing" || strings.Contains(recorder.Body.String(), videoCID) {
		t.Fatalf("unsafe or incorrect response: %s", recorder.Body.String())
	}
}

func TestCreatePost_PDSWriteFailure_LogsExcludeRequestTextAndToken(t *testing.T) {
	t.Parallel()
	var logs strings.Builder
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	const sentinelText = "SENSITIVE_POST_TEXT"
	const sentinelToken = "SENSITIVE_TOKEN"
	pds := &fakePostEffectState{createErr: errors.New("pds rejected create")}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSEffectsFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), logger)
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":"`+sentinelText+`","sponsored":false}`, "did:plc:alice")
	req.Header.Set("Authorization", "Bearer "+sentinelToken)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	out := logs.String()
	if strings.Contains(out, sentinelText) {
		t.Fatalf("logs leaked request text: %s", out)
	}
	if strings.Contains(out, sentinelToken) {
		t.Fatalf("logs leaked token: %s", out)
	}
}

func TestCreatePost_PDSUnavailable_502(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{}
	// Factory itself fails — the PDS RPC layer is never reached.
	failingFactory := func(_ context.Context, _ syntax.DID, _ string) (pdseffects.EffectExecutor, error) {
		return nil, errors.New("session lookup failed")
	}
	h := api.CreatePostHandler(store, failingFactory, fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":"hi","sponsored":false}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "pds_unavailable") {
		t.Errorf("expected pds_unavailable in body, got: %s", body)
	}
}

func TestCreatePost_PDSSessionExpiredReturns401(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSEffectsFactory(&fakePostEffectState{createErr: auth.ErrPDSSessionExpired}), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":"hi","sponsored":false}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "pds_session_expired" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestCreatePost_QuoteEmbed_TranslatedToLexiconShape(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	store := &fakePostStore{shareTarget: &api.ShareTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/r1", CID: "bafyB"}}
	h := api.CreatePostHandler(store, newPDSEffectsFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	body := `{"text":"hi","sponsored":false,"embed":{"quote":{"uri":"at://did:plc:bob/social.craftsky.feed.post/r1","cid":"bafyB"}}}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	rec, _ := pds.lastCreateRec.(map[string]any)
	embed, _ := rec["embed"].(map[string]any)
	if embed["$type"] != "social.craftsky.feed.post#quoteEmbed" {
		t.Errorf("embed $type: %v", embed["$type"])
	}
	r, _ := embed["record"].(map[string]any)
	if r["uri"] != "at://did:plc:bob/social.craftsky.feed.post/r1" {
		t.Errorf("embed.record.uri = %v", r["uri"])
	}
}

func TestCreatePost_ExternalEmbed_WritesStandardShapeAndRejectsConflicts(t *testing.T) {
	valid := []struct {
		name         string
		body         string
		wantExternal map[string]any
	}{
		{
			name: "metadata only without matching facet",
			body: `{"text":"plain text","sponsored":false,"embed":{"external":{"uri":"https://final.example/pattern","title":"Pattern","description":"A useful pattern"}}}`,
			wantExternal: map[string]any{
				"uri":         "https://final.example/pattern",
				"title":       "Pattern",
				"description": "A useful pattern",
			},
		},
		{
			name: "thumbnail",
			body: `{"text":"link","sponsored":false,"embed":{"external":{"uri":"https://final.example/pattern","title":"Pattern","description":"","thumb":{"$type":"blob","ref":{"$link":"bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"},"mimeType":"image/webp","size":321}}}}`,
			wantExternal: map[string]any{
				"uri":         "https://final.example/pattern",
				"title":       "Pattern",
				"description": "",
				"thumb": map[string]any{
					"$type":    "blob",
					"ref":      map[string]any{"$link": "bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"},
					"mimeType": "image/webp",
					"size":     float64(321),
				},
			},
		},
	}
	for _, test := range valid {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			pds := &fakePostEffectState{}
			handler := api.CreatePostHandler(&fakePostStore{}, newPDSEffectsFactory(pds), fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger())
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, authedReq(http.MethodPost, "/v1/posts", test.body, "did:plc:alice"))

			if recorder.Code != http.StatusCreated {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			record, ok := pds.lastCreateRec.(map[string]any)
			if !ok {
				t.Fatalf("record type = %T", pds.lastCreateRec)
			}
			embed, ok := record["embed"].(map[string]any)
			if !ok || embed["$type"] != "app.bsky.embed.external" {
				t.Fatalf("embed = %#v", record["embed"])
			}
			if !reflect.DeepEqual(embed["external"], test.wantExternal) {
				t.Fatalf("external = %#v, want %#v", embed["external"], test.wantExternal)
			}
			var response api.PostResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("response JSON: %v", err)
			}
			if response.External == nil || response.External.URI != test.wantExternal["uri"] || response.External.Title != test.wantExternal["title"] {
				t.Fatalf("response external = %#v", response.External)
			}
		})
	}

	conflicts := []struct {
		name string
		body string
	}{
		{name: "quote", body: `{"text":"link","sponsored":false,"embed":{"quote":{"uri":"at://did:plc:bob/social.craftsky.feed.post/r1","cid":"bafyquote"},"external":{"uri":"https://final.example","title":"Pattern","description":""}}}`},
		{name: "images", body: `{"text":"link","sponsored":false,"images":[{"image":{"ref":{"$link":"bafyimage"},"mimeType":"image/jpeg","size":1},"alt":""}],"embed":{"external":{"uri":"https://final.example","title":"Pattern","description":""}}}`},
		{name: "project", body: `{"text":"link","sponsored":false,"project":{"common":{"craftType":"social.craftsky.feed.defs#knitting"}},"embed":{"external":{"uri":"https://final.example","title":"Pattern","description":""}}}`},
	}
	for _, test := range conflicts {
		t.Run(test.name+" conflict", func(t *testing.T) {
			t.Parallel()
			pds := &fakePostEffectState{}
			handler := api.CreatePostHandler(&fakePostStore{}, newPDSEffectsFactory(pds), fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger())
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, authedReq(http.MethodPost, "/v1/posts", test.body, "did:plc:alice"))

			if recorder.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			if pds.createCalls != 0 {
				t.Fatalf("PDS writes = %d, want 0", pds.createCalls)
			}
		})
	}
}

func TestCreatePost_ExternalThumbnailRejectsMalformedBlobBeforePDSWrite(t *testing.T) {
	t.Parallel()

	const canonicalCID = "bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"
	tests := []struct {
		name  string
		thumb string
	}{
		{name: "missing blob type", thumb: `{"ref":{"$link":"` + canonicalCID + `"},"mimeType":"image/png","size":1}`},
		{name: "wrong blob type", thumb: `{"$type":"not-a-blob","ref":{"$link":"` + canonicalCID + `"},"mimeType":"image/png","size":1}`},
		{name: "malformed CID", thumb: `{"$type":"blob","ref":{"$link":"not-a-cid"},"mimeType":"image/png","size":1}`},
		{name: "noncanonical CID", thumb: `{"$type":"blob","ref":{"$link":"BAFKREIE3W2XQ7U6RS5SZU6VLLSQ5XH7Y7UV3F6BLQL6UZ4EP6TXV6M4O6A"},"mimeType":"image/png","size":1}`},
		{name: "unknown blob field", thumb: `{"$type":"blob","ref":{"$link":"` + canonicalCID + `"},"mimeType":"image/png","size":1,"extra":true}`},
		{name: "unknown ref field", thumb: `{"$type":"blob","ref":{"$link":"` + canonicalCID + `","extra":true},"mimeType":"image/png","size":1}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			pds := &fakePostEffectState{}
			handler := api.CreatePostHandler(&fakePostStore{}, newPDSEffectsFactory(pds), fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger())
			body := `{"text":"link","sponsored":false,"embed":{"external":{"uri":"https://final.example/pattern","title":"Pattern","description":"","thumb":` + test.thumb + `}}}`
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice"))

			if recorder.Code < 400 || recorder.Code >= 500 {
				t.Fatalf("status = %d, want 4xx; body = %s", recorder.Code, recorder.Body.String())
			}
			if pds.createCalls != 0 {
				t.Fatalf("PDS writes = %d, want 0", pds.createCalls)
			}
		})
	}
}

func TestCreatePost_QuoteEmbed_UsesResolvedTargetCID(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	store := &fakePostStore{
		shareTarget: &api.ShareTargetRef{
			URI: "at://did:plc:bob/social.craftsky.feed.post/r1",
			CID: "bafyActual",
		},
	}
	h := api.CreatePostHandler(store, newPDSEffectsFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	body := `{"text":"hi","sponsored":false,"embed":{"quote":{"uri":"at://did:plc:bob/social.craftsky.feed.post/r1","cid":"bafyStale"}}}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	rec, _ := pds.lastCreateRec.(map[string]any)
	embed, _ := rec["embed"].(map[string]any)
	record, _ := embed["record"].(map[string]any)
	if record["uri"] != "at://did:plc:bob/social.craftsky.feed.post/r1" || record["cid"] != "bafyActual" {
		t.Fatalf("quote record = %+v, want resolved target strongRef", record)
	}
	var resp api.PostResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response JSON: %v", err)
	}
	if resp.Quote == nil || resp.Quote.CID != "bafyActual" {
		t.Fatalf("response quote = %+v, want resolved target CID", resp.Quote)
	}
}

func TestCreatePost_QuoteEmbed_AttachesCompactQuoteView(t *testing.T) {
	t.Parallel()
	quoteURI := "at://did:plc:bob/social.craftsky.feed.post/r1"
	quoteCID := "bafyActual"
	pds := &fakePostEffectState{}
	quoted := testPostRow("did:plc:bob", "r1", "quoted text", time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	quoted.CID = quoteCID
	store := &fakePostStore{
		shareTarget: &api.ShareTargetRef{
			URI: quoteURI,
			CID: quoteCID,
		},
		quoteViews: map[string]*api.QuoteViewRow{
			quoteURI: {State: "visible", Post: quoted},
		},
	}
	h := api.CreatePostHandler(store, newPDSEffectsFactory(pds), fakeResolver{handlesByDID: map[string]syntax.Handle{
		"did:plc:alice": "alice.example",
		"did:plc:bob":   "bob.example",
	}}, api.DefaultMediaLimits(), nilLogger())
	body := `{"text":"hi","sponsored":false,"embed":{"quote":{"uri":"at://did:plc:bob/social.craftsky.feed.post/r1","cid":"bafyStale"}}}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if len(store.lastQuoteViewRefs) != 1 || store.lastQuoteViewRefs[0].URI != quoteURI || store.lastQuoteViewRefs[0].CID != quoteCID {
		t.Fatalf("quote view refs = %+v", store.lastQuoteViewRefs)
	}
	var resp api.PostResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response JSON: %v", err)
	}
	if resp.Quote == nil || resp.Quote.CID != quoteCID {
		t.Fatalf("response quote = %+v, want resolved target CID", resp.Quote)
	}
	if resp.QuoteView == nil || resp.QuoteView.State != "visible" || resp.QuoteView.Post == nil {
		t.Fatalf("quoteView = %+v, want visible preview", resp.QuoteView)
	}
	if resp.QuoteView.Post.URI != quoteURI || resp.QuoteView.Post.Text != "quoted text" || resp.QuoteView.Post.Author.Handle != "bob.example" {
		t.Fatalf("quoteView.post = %+v", resp.QuoteView.Post)
	}
}

func TestCreatePost_AllowsMultipleQuotePostsForSameSubject(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	store := &fakePostStore{shareTarget: &api.ShareTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/r1", CID: "bafyB"}}
	h := api.CreatePostHandler(store, newPDSEffectsFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	body := `{"text":"another angle","sponsored":false,"embed":{"quote":{"uri":"at://did:plc:bob/social.craftsky.feed.post/r1","cid":"bafyB"}}}`

	for i := 0; i < 2; i++ {
		req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("request %d status = %d, body = %s", i+1, rr.Code, rr.Body.String())
		}
	}
	if pds.createCalls != 2 {
		t.Fatalf("createCalls = %d, want 2 independent quote post writes", pds.createCalls)
	}
	if store.lastShareTargetDID != "did:plc:bob" || store.lastShareTargetRkey != "r1" {
		t.Fatalf("last share target = %s/%s", store.lastShareTargetDID, store.lastShareTargetRkey)
	}
}

func TestCreatePost_AllowsSelfQuote(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	store := &fakePostStore{shareTarget: &api.ShareTargetRef{URI: "at://did:plc:alice/social.craftsky.feed.post/own", CID: "bafyOwn"}}
	h := api.CreatePostHandler(store, newPDSEffectsFactory(pds), fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger())
	body := `{"text":"adding more context","sponsored":false,"embed":{"quote":{"uri":"at://did:plc:alice/social.craftsky.feed.post/own","cid":"bafyOwn"}}}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if store.lastShareTargetDID != "did:plc:alice" || store.lastShareTargetRkey != "own" {
		t.Fatalf("share target lookup = %s/%s", store.lastShareTargetDID, store.lastShareTargetRkey)
	}
	rec, _ := pds.lastCreateRec.(map[string]any)
	embed, _ := rec["embed"].(map[string]any)
	record, _ := embed["record"].(map[string]any)
	if record["uri"] != "at://did:plc:alice/social.craftsky.feed.post/own" {
		t.Fatalf("quote target uri = %v", record["uri"])
	}
}

func TestCreatePost_QuoteRejectsReplyTargetBeforePDSWrite(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	store := &fakePostStore{
		shareTarget: &api.ShareTargetRef{
			URI:     "at://did:plc:bob/social.craftsky.feed.post/reply",
			CID:     "bafyReply",
			IsReply: true,
		},
	}
	h := api.CreatePostHandler(store, newPDSEffectsFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	body := `{"text":"hi","sponsored":false,"embed":{"quote":{"uri":"at://did:plc:bob/social.craftsky.feed.post/reply","cid":"bafyReply"}}}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.createCalls != 0 {
		t.Fatalf("CreateRecord calls = %d, want 0", pds.createCalls)
	}
	var env envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&env)
	if env.Error != "validation_failed" {
		t.Fatalf("error = %q, want validation_failed", env.Error)
	}
	if store.lastShareTargetDID != "did:plc:bob" || store.lastShareTargetRkey != "reply" {
		t.Fatalf("share target lookup = %q/%q", store.lastShareTargetDID, store.lastShareTargetRkey)
	}
}

func TestCreatePost_ProjectQuoteRejectedBeforePDSWrite(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSEffectsFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	body := `{
		"text":"project quote",
		"sponsored":false,
		"project":{"common":{"craftType":"social.craftsky.feed.defs#knitting"}},
		"embed":{"quote":{"uri":"at://did:plc:bob/social.craftsky.feed.post/target","cid":"bafyTarget"}}
	}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.createCalls != 0 {
		t.Fatalf("CreateRecord calls = %d, want 0", pds.createCalls)
	}
	if store.lastShareTargetDID != "" || store.lastShareTargetRkey != "" {
		t.Fatalf("share target lookup = %q/%q, want none", store.lastShareTargetDID, store.lastShareTargetRkey)
	}
	var env envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&env)
	if env.Error != "validation_failed" {
		t.Fatalf("error = %q, want validation_failed", env.Error)
	}
}

func TestCreatePost_TagsExtractedFromFacets(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSEffectsFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	body := `{"text":"hi #knit","sponsored":false,"facets":[{"index":{"byteStart":3,"byteEnd":8},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"Knitting"}]}]}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.PostResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if len(resp.Tags) != 1 || resp.Tags[0] != "knitting" {
		t.Errorf("tags = %v, want [knitting]", resp.Tags)
	}
}

func TestCreatePost_WithProject_WritesProjectToPDSAndResponse(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSEffectsFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	body := `{
		"text":"finished shawl #FairIsle",
		"sponsored":false,
		"facets":[{"index":{"byteStart":15,"byteEnd":24},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"FairIsle"}]}],
		"project":{
			"common":{
				"craftType":"social.craftsky.feed.defs#knitting",
				"title":"Hitchhiker Shawl",
				"materials":[{"text":"3m of @alice.craftsky.social #viscose fabric","facets":[{"index":{"byteStart":6,"byteEnd":28},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:alice"}]},{"index":{"byteStart":29,"byteEnd":37},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"Viscose"}]}]}],
				"tags":[" fairisle ", "WIP"],
				"pattern":{
					"name":"#hitchhiker",
					"nameFacets":[{"index":{"byteStart":0,"byteEnd":11},"features":[{"$type":"app.bsky.richtext.facet#tag","tag":"Hitchhiker"}]}],
					"designer":"@alice.craftsky.social",
					"designerFacets":[{"index":{"byteStart":0,"byteEnd":22},"features":[{"$type":"app.bsky.richtext.facet#mention","did":"did:plc:alice"}]}]
				}
			},
			"details":{"$type":"social.craftsky.project.knitting#details","projectType":"shawl"}
		}
	}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}

	rec, _ := pds.lastCreateRec.(map[string]any)
	project, _ := rec["project"].(*api.Project)
	if project == nil || project.Common.CraftType != "social.craftsky.feed.defs#knitting" || project.Common.Title == nil || *project.Common.Title != "Hitchhiker Shawl" {
		t.Fatalf("PDS project = %#v", rec["project"])
	}
	if project.Common.Pattern == nil || project.Common.Pattern.Name == nil || *project.Common.Pattern.Name != "#hitchhiker" {
		t.Fatalf("PDS project pattern = %#v", project.Common.Pattern)
	}
	if len(project.Common.Materials) != 1 || project.Common.Materials[0].Text != "3m of @alice.craftsky.social #viscose fabric" {
		t.Fatalf("PDS project materials = %#v", project.Common.Materials)
	}
	if got := facetArrayLength(t, project.Common.Materials[0].Facets); got != 2 {
		t.Fatalf("PDS material facets len = %d, want 2", got)
	}
	if got := facetArrayLength(t, project.Common.Pattern.NameFacets); got != 1 {
		t.Fatalf("PDS pattern name facets len = %d, want 1", got)
	}
	if got := facetArrayLength(t, project.Common.Pattern.DesignerFacets); got != 1 {
		t.Fatalf("PDS pattern designer facets len = %d, want 1", got)
	}
	if _, ok := rec["createdAt"].(string); !ok {
		t.Fatalf("createdAt missing from PDS record: %#v", rec)
	}

	var resp api.PostResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Project == nil || resp.Project.Common.Title == nil || *resp.Project.Common.Title != "Hitchhiker Shawl" {
		t.Fatalf("response project = %+v", resp.Project)
	}
	wantTags := []string{"fairisle", "wip", "hitchhiker", "viscose"}
	if !reflect.DeepEqual(resp.Tags, wantTags) {
		t.Fatalf("response tags = %v, want %v", resp.Tags, wantTags)
	}
}

func TestCreatePost_AuthorHydratedFromStore(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	displayName := "Alice"
	avatarCID := "bafyAvatar"
	avatarMime := "image/jpeg"
	store := &fakePostStore{
		author: &api.PostAuthorRow{DisplayName: &displayName, AvatarCID: &avatarCID, AvatarMime: &avatarMime},
	}
	h := api.CreatePostHandler(store, newPDSEffectsFactory(pds), fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits(), nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":"hi","sponsored":false}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var resp api.PostResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.Author.DisplayName == nil || *resp.Author.DisplayName != "Alice" {
		t.Errorf("displayName = %v", resp.Author.DisplayName)
	}
	if resp.Author.AvatarCID == nil || *resp.Author.AvatarCID != "bafyAvatar" {
		t.Errorf("avatarCID = %v", resp.Author.AvatarCID)
	}
	if resp.Author.Avatar == nil || *resp.Author.Avatar != "https://cdn.bsky.app/img/avatar/plain/did:plc:alice/bafyAvatar@jpeg" {
		t.Errorf("avatar = %v", resp.Author.Avatar)
	}
}

func TestCreatePost_ResolveHandleFails_502(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSEffectsFactory(pds), fakeResolver{err: errors.New("plc down")}, api.DefaultMediaLimits(), nilLogger())
	req := authedReq(http.MethodPost, "/v1/posts", `{"text":"hi","sponsored":false}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
}

func TestDeletePost_Self_204_CallsPDS(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	h := api.DeletePostHandler(newPDSEffectsFactory(pds), nilLogger())
	req := authedReq(http.MethodDelete, "/v1/posts/did:plc:alice/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.lastDeleteRkey != "rk1" {
		t.Errorf("PDS not called: %q", pds.lastDeleteRkey)
	}
}

func TestDeletePost_OtherUser_403_NoPDSCall(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	h := api.DeletePostHandler(newPDSEffectsFactory(pds), nilLogger())
	req := authedReq(http.MethodDelete, "/v1/posts/did:plc:bob/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "did:plc:bob")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d", rr.Code)
	}
	if pds.lastDeleteRkey != "" {
		t.Errorf("PDS should not have been called")
	}
}

func TestDeletePost_RecordAlreadyGone_204_Idempotent(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{deleteErr: auth.ErrRecordNotFound}
	h := api.DeletePostHandler(newPDSEffectsFactory(pds), nilLogger())
	req := authedReq(http.MethodDelete, "/v1/posts/did:plc:alice/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestDeletePost_PDSDown_502(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{deleteErr: errors.New("pds down")}
	h := api.DeletePostHandler(newPDSEffectsFactory(pds), nilLogger())
	req := authedReq(http.MethodDelete, "/v1/posts/did:plc:alice/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "did:plc:alice")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestDeletePost_BadDID_400(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	h := api.DeletePostHandler(newPDSEffectsFactory(pds), nilLogger())
	req := authedReq(http.MethodDelete, "/v1/posts/not-a-did/rk1", "", "did:plc:alice")
	req.SetPathValue("did", "not-a-did")
	req.SetPathValue("rkey", "rk1")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestCreatePost_WithReply_PassesThroughToPDS(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSEffectsFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	body := `{"text":"replying","sponsored":false,"reply":{"root":{"uri":"at://did:plc:bob/social.craftsky.feed.post/root1","cid":"bafyR1"},"parent":{"uri":"at://did:plc:bob/social.craftsky.feed.post/par1","cid":"bafyP1"}}}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	rec, _ := pds.lastCreateRec.(map[string]any)
	reply, _ := rec["reply"].(map[string]any)
	if reply == nil {
		t.Fatalf("expected reply in PDS body, got: %+v", rec)
	}
	root, _ := reply["root"].(map[string]any)
	if root["uri"] != "at://did:plc:bob/social.craftsky.feed.post/root1" {
		t.Errorf("reply.root.uri = %v", root["uri"])
	}
	if root["cid"] != "bafyR1" {
		t.Errorf("reply.root.cid = %v", root["cid"])
	}
	parent, _ := reply["parent"].(map[string]any)
	if parent["uri"] != "at://did:plc:bob/social.craftsky.feed.post/par1" {
		t.Errorf("reply.parent.uri = %v", parent["uri"])
	}
	if parent["cid"] != "bafyP1" {
		t.Errorf("reply.parent.cid = %v", parent["cid"])
	}
}

func TestCreatePost_WithImages_WritesTopLevelImagesToPDS(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSEffectsFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	body := `{
		"text":"image post",
		"sponsored":false,
		"images":[
			{
				"image":{"$type":"blob","ref":{"$link":"bafkimage"},"mimeType":"image/jpeg","size":253496},
				"alt":"project photo",
				"aspectRatio":{"width":919,"height":2000}
			}
		]
	}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	rec, _ := pds.lastCreateRec.(map[string]any)
	recJSON, _ := json.Marshal(rec)
	var normalized map[string]any
	if err := json.Unmarshal(recJSON, &normalized); err != nil {
		t.Fatalf("decode rec: %v", err)
	}
	imagesRaw, ok := normalized["images"].([]any)
	if !ok || len(imagesRaw) != 1 {
		t.Fatalf("images = %T %v, want single image array", normalized["images"], normalized["images"])
	}
	img, _ := imagesRaw[0].(map[string]any)
	if img["alt"] != "project photo" {
		t.Fatalf("alt = %v", img["alt"])
	}
	if _, ok := img["image"].(map[string]any); !ok {
		t.Fatalf("image blob missing: %+v", img)
	}
	aspect, _ := img["aspectRatio"].(map[string]any)
	if aspect["width"] != float64(919) || aspect["height"] != float64(2000) {
		t.Fatalf("aspectRatio = %+v", aspect)
	}
	if _, hasEmbed := normalized["embed"]; hasEmbed {
		t.Fatalf("embed must be absent for plain image post; got %+v", normalized["embed"])
	}

	var resp api.PostResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Images) != 1 {
		t.Fatalf("response images = %d, want 1", len(resp.Images))
	}
	view := resp.Images[0]
	if view.CID != "bafkimage" || view.MIME != "image/jpeg" || view.Size != 253496 || view.Alt != "project photo" {
		t.Fatalf("response image = %+v", view)
	}
	if view.AspectRatio == nil || view.AspectRatio.Width != 919 || view.AspectRatio.Height != 2000 {
		t.Fatalf("response aspectRatio = %+v", view.AspectRatio)
	}
	if view.Thumb == "" || view.Fullsize == "" {
		t.Fatalf("response image urls missing: %+v", view)
	}
}

func TestCreatePost_WithImageMissingAlt_OmitsAltInPDSRecord(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	store := &fakePostStore{}
	h := api.CreatePostHandler(store, newPDSEffectsFactory(pds), fakeResolver{handleFor: "a.example"}, api.DefaultMediaLimits(), nilLogger())
	body := `{
		"text":"image post",
		"sponsored":false,
		"images":[
			{
				"image":{"$type":"blob","ref":{"$link":"bafkimage"},"mimeType":"image/jpeg","size":253496}
			}
		]
	}`
	req := authedReq(http.MethodPost, "/v1/posts", body, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	rec, _ := pds.lastCreateRec.(map[string]any)
	imagesRaw, ok := rec["images"].([]map[string]any)
	if !ok || len(imagesRaw) != 1 {
		t.Fatalf("images = %T %v, want single image array", rec["images"], rec["images"])
	}
	if _, ok := imagesRaw[0]["alt"]; ok {
		t.Fatalf("alt should be absent when not provided: %+v", imagesRaw[0])
	}
}

func facetArrayLength(t *testing.T, raw json.RawMessage) int {
	t.Helper()
	var facets []map[string]any
	if err := json.Unmarshal(raw, &facets); err != nil {
		t.Fatalf("unmarshal facets %s: %v", raw, err)
	}
	return len(facets)
}
