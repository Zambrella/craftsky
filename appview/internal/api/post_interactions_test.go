package api_test

import (
	"encoding/json"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"net/http"
	"net/http/httptest"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"testing"
	"time"
)

func TestLikePost_CreatesPDSLikeRecord(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{
		createURI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.like/likeSrv"),
		createCID: syntax.CID("bafyLike"),
	}
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	h := api.LikePostHandler(store, newPDSEffectsFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/likes", "   ", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.lastCreateRepo != "did:plc:alice" || pds.lastCreateColl != "social.craftsky.feed.like" {
		t.Fatalf("CreateRecord repo/coll = %q/%q", pds.lastCreateRepo, pds.lastCreateColl)
	}
	rec, ok := pds.lastCreateRec.(map[string]any)
	if !ok {
		t.Fatalf("record type = %T", pds.lastCreateRec)
	}
	if rec["$type"] != "social.craftsky.feed.like" {
		t.Errorf("$type = %v", rec["$type"])
	}
	subject := rec["subject"].(map[string]any)
	if subject["uri"] != store.target.URI || subject["cid"] != store.target.CID {
		t.Errorf("subject = %+v", subject)
	}
	if _, err := time.Parse(time.RFC3339, rec["createdAt"].(string)); err != nil {
		t.Errorf("createdAt is not RFC3339: %v", err)
	}
	var resp api.InteractionWriteResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.URI != string(pds.createURI) || resp.CID != string(pds.createCID) || resp.Rkey != "likeSrv" {
		t.Errorf("resp identity = %+v", resp)
	}
	if resp.Subject.URI != store.target.URI || resp.Subject.CID != store.target.CID {
		t.Errorf("resp subject = %+v", resp.Subject)
	}
	if store.lastTargetDID != "did:plc:bob" || store.lastTargetRkey != "post1" {
		t.Errorf("target lookup = %q/%q", store.lastTargetDID, store.lastTargetRkey)
	}
	if store.lastActiveLikeDID != "did:plc:alice" || store.lastActiveLikeURI != store.target.URI {
		t.Errorf("active lookup = %q/%q", store.lastActiveLikeDID, store.lastActiveLikeURI)
	}
}

func TestLikePost_AlreadyLikedReturnsExistingIdentity(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	created := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	store := &fakePostStore{
		target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeLike: &api.InteractionRow{
			URI: "at://did:plc:alice/social.craftsky.feed.like/existing", DID: "did:plc:alice", Rkey: "existing", CID: "bafyExisting",
			SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1", SubjectCID: "bafyPost", CreatedAt: created,
		},
	}
	h := api.LikePostHandler(store, newPDSEffectsFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/likes", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.lastCreateColl != "" {
		t.Fatalf("CreateRecord called for already-liked path: %q", pds.lastCreateColl)
	}
	var resp api.InteractionWriteResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.URI != store.activeLike.URI || resp.CID != store.activeLike.CID || resp.Rkey != store.activeLike.Rkey {
		t.Errorf("resp = %+v", resp)
	}
}

func TestLikePost_RejectsNonEmptyBody(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	h := api.LikePostHandler(store, newPDSEffectsFactory(&fakePostEffectState{}), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/likes", `{"foo":true}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "unexpected_field" {
		t.Errorf("error = %q", body.Error)
	}
	if pdsLookup := store.lastTargetDID; pdsLookup != "" {
		t.Errorf("target lookup should not run, got %q", pdsLookup)
	}
}

func TestLikePost_MissingSubjectReturns404(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{targetErr: api.ErrPostNotFound}
	h := api.LikePostHandler(store, newPDSEffectsFactory(&fakePostEffectState{}), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/missing/likes", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "post_not_found" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestLikePost_PDSCreateFailureReturns502(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	h := api.LikePostHandler(store, newPDSEffectsFactory(&fakePostEffectState{createErr: errors.New("pds down")}), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/likes", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "pds_write_failed" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestLikePost_PDSSessionExpiredReturns401(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	h := api.LikePostHandler(store, newPDSEffectsFactory(&fakePostEffectState{createErr: auth.ErrPDSSessionExpired}), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/likes", "", "did:plc:alice")
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

func TestUnlikePost_ExistingDeletesPDSRecord(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	store := &fakePostStore{
		target:     &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeLike: &api.InteractionRow{URI: "at://did:plc:alice/social.craftsky.feed.like/like1", DID: "did:plc:alice", Rkey: "like1", CID: "bafyLike", SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1", SubjectCID: "bafyPost"},
	}
	h := api.UnlikePostHandler(store, newPDSEffectsFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/likes", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.lastDeleteRepo != "did:plc:alice" || pds.lastDeleteColl != "social.craftsky.feed.like" || pds.lastDeleteRkey != "like1" {
		t.Errorf("DeleteRecord = repo %q coll %q rkey %q", pds.lastDeleteRepo, pds.lastDeleteColl, pds.lastDeleteRkey)
	}
	if pds.deleteCalls != 1 {
		t.Errorf("deleteCalls = %d, want 1", pds.deleteCalls)
	}
}

func TestUnlikePost_AbsentActiveLikeIsIdempotent(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	h := api.UnlikePostHandler(store, newPDSEffectsFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/likes", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.deleteCalls != 0 {
		t.Errorf("deleteCalls = %d, want 0", pds.deleteCalls)
	}
}

func TestUnlikePost_PDSDeleteFailureReturns502(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{
		target:     &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeLike: &api.InteractionRow{Rkey: "like1", SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1", SubjectCID: "bafyPost"},
	}
	h := api.UnlikePostHandler(store, newPDSEffectsFactory(&fakePostEffectState{deleteErr: errors.New("pds down")}), nilLogger())
	req := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/likes", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "pds_unavailable" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestRepostPost_CreatesPDSRepostRecord(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{
		createURI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.repost/repostSrv"),
		createCID: syntax.CID("bafyRepost"),
	}
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	h := api.RepostPostHandler(store, newPDSEffectsFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/reposts", "   ", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.lastCreateRepo != "did:plc:alice" || pds.lastCreateColl != "social.craftsky.feed.repost" {
		t.Fatalf("CreateRecord repo/coll = %q/%q", pds.lastCreateRepo, pds.lastCreateColl)
	}
	rec, ok := pds.lastCreateRec.(map[string]any)
	if !ok {
		t.Fatalf("record type = %T", pds.lastCreateRec)
	}
	if rec["$type"] != "social.craftsky.feed.repost" {
		t.Errorf("$type = %v", rec["$type"])
	}
	subject := rec["subject"].(map[string]any)
	if subject["uri"] != store.target.URI || subject["cid"] != store.target.CID {
		t.Errorf("subject = %+v", subject)
	}
	if _, hasText := rec["text"]; hasText {
		t.Errorf("repost record unexpectedly had text: %+v", rec)
	}
	if _, hasEmbed := rec["embed"]; hasEmbed {
		t.Errorf("repost record unexpectedly had embed: %+v", rec)
	}
	if _, err := time.Parse(time.RFC3339, rec["createdAt"].(string)); err != nil {
		t.Errorf("createdAt is not RFC3339: %v", err)
	}
	var resp api.InteractionWriteResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.URI != string(pds.createURI) || resp.CID != string(pds.createCID) || resp.Rkey != "repostSrv" {
		t.Errorf("resp identity = %+v", resp)
	}
	if resp.Subject.URI != store.target.URI || resp.Subject.CID != store.target.CID {
		t.Errorf("resp subject = %+v", resp.Subject)
	}
	if store.lastShareTargetDID != "did:plc:bob" || store.lastShareTargetRkey != "post1" {
		t.Errorf("share target lookup = %q/%q", store.lastShareTargetDID, store.lastShareTargetRkey)
	}
	if store.lastActiveRepostDID != "did:plc:alice" || store.lastActiveRepostURI != store.target.URI {
		t.Errorf("active lookup = %q/%q", store.lastActiveRepostDID, store.lastActiveRepostURI)
	}
}

func TestRepostPost_AllowsSelfRepost(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{
		createURI: syntax.ATURI("at://did:plc:alice/social.craftsky.feed.repost/selfRepost"),
		createCID: syntax.CID("bafySelfRepost"),
	}
	store := &fakePostStore{shareTarget: &api.ShareTargetRef{URI: "at://did:plc:alice/social.craftsky.feed.post/own", CID: "bafyOwn"}}
	h := api.RepostPostHandler(store, newPDSEffectsFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:alice/own/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if store.lastShareTargetDID != "did:plc:alice" || store.lastShareTargetRkey != "own" {
		t.Fatalf("share target lookup = %s/%s", store.lastShareTargetDID, store.lastShareTargetRkey)
	}
	if pds.lastCreateRepo != "did:plc:alice" || pds.lastCreateColl != "social.craftsky.feed.repost" {
		t.Fatalf("CreateRecord repo/coll = %q/%q", pds.lastCreateRepo, pds.lastCreateColl)
	}
}

func TestRepostPost_AlreadyRepostedReturnsExistingIdentity(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	created := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	store := &fakePostStore{
		target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeRepost: &api.InteractionRow{
			URI: "at://did:plc:alice/social.craftsky.feed.repost/existing", DID: "did:plc:alice", Rkey: "existing", CID: "bafyExisting",
			SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1", SubjectCID: "bafyPost", CreatedAt: created,
		},
	}
	h := api.RepostPostHandler(store, newPDSEffectsFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.lastCreateColl != "" {
		t.Fatalf("CreateRecord called for already-reposted path: %q", pds.lastCreateColl)
	}
	var resp api.InteractionWriteResponse
	_ = json.NewDecoder(rr.Body).Decode(&resp)
	if resp.URI != store.activeRepost.URI || resp.CID != store.activeRepost.CID || resp.Rkey != store.activeRepost.Rkey {
		t.Errorf("resp = %+v", resp)
	}
}

func TestRepostPost_RejectsReplyTargetBeforePDSWrite(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	store := &fakePostStore{
		shareTarget: &api.ShareTargetRef{
			URI:     "at://did:plc:bob/social.craftsky.feed.post/reply",
			CID:     "bafyReply",
			IsReply: true,
		},
	}
	h := api.RepostPostHandler(store, newPDSEffectsFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/reply/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.createCalls != 0 {
		t.Fatalf("CreateRecord calls = %d, want 0", pds.createCalls)
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "validation_failed" {
		t.Fatalf("error = %q, want validation_failed", body.Error)
	}
}

func TestRepostPost_RejectsNonEmptyBody(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	h := api.RepostPostHandler(store, newPDSEffectsFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/reposts", `{"text":"quote-like body","embed":{"foo":true}}`, "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "unexpected_field" {
		t.Errorf("error = %q", body.Error)
	}
	if store.lastTargetDID != "" {
		t.Errorf("target lookup should not run, got %q", store.lastTargetDID)
	}
	if pds.lastCreateColl != "" {
		t.Errorf("CreateRecord should not run, got coll %q", pds.lastCreateColl)
	}
}

func TestRepostPost_MissingSubjectReturns404(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{shareTargetErr: api.ErrPostNotFound}
	h := api.RepostPostHandler(store, newPDSEffectsFactory(&fakePostEffectState{}), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/missing/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "post_not_found" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestRepostPost_PDSCreateFailureReturns502(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	h := api.RepostPostHandler(store, newPDSEffectsFactory(&fakePostEffectState{createErr: errors.New("pds down")}), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "pds_write_failed" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestRepostPost_NewPDSFailureReturns502(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	h := api.RepostPostHandler(store, failingPDSEffectsFactory(errors.New("session missing")), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "pds_unavailable" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestRepostPost_TargetLookupFailureReturns500(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{shareTargetErr: errors.New("database unavailable")}
	h := api.RepostPostHandler(store, newPDSEffectsFactory(&fakePostEffectState{}), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "internal_error" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestRepostPost_ActiveLookupFailureReturns500(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{
		target:          &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeRepostErr: errors.New("database unavailable"),
	}
	h := api.RepostPostHandler(store, newPDSEffectsFactory(&fakePostEffectState{}), nilLogger())
	req := authedPostPathReq(http.MethodPost, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "internal_error" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestUnrepostPost_ExistingDeletesPDSRecord(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	store := &fakePostStore{
		target:       &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeRepost: &api.InteractionRow{URI: "at://did:plc:alice/social.craftsky.feed.repost/repost1", DID: "did:plc:alice", Rkey: "repost1", CID: "bafyRepost", SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1", SubjectCID: "bafyPost"},
	}
	h := api.UnrepostPostHandler(store, newPDSEffectsFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.lastDeleteRepo != "did:plc:alice" || pds.lastDeleteColl != "social.craftsky.feed.repost" || pds.lastDeleteRkey != "repost1" {
		t.Errorf("DeleteRecord = repo %q coll %q rkey %q", pds.lastDeleteRepo, pds.lastDeleteColl, pds.lastDeleteRkey)
	}
	if pds.deleteCalls != 1 {
		t.Errorf("deleteCalls = %d, want 1", pds.deleteCalls)
	}
}

func TestUnrepostPost_WithAuthoredQuoteDeletesOnlyStraightRepost(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	quoteRkey := "quote1"
	store := &fakePostStore{
		target:       &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeRepost: &api.InteractionRow{URI: "at://did:plc:alice/social.craftsky.feed.repost/repost1", DID: "did:plc:alice", Rkey: "repost1", CID: "bafyRepost", SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1", SubjectCID: "bafyPost"},
	}
	h := api.UnrepostPostHandler(store, newPDSEffectsFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.deleteCalls != 1 {
		t.Fatalf("deleteCalls = %d, want 1", pds.deleteCalls)
	}
	if pds.lastDeleteColl != "social.craftsky.feed.repost" || pds.lastDeleteRkey != "repost1" {
		t.Fatalf("DeleteRecord = coll %q rkey %q, want straight repost", pds.lastDeleteColl, pds.lastDeleteRkey)
	}
	if pds.lastDeleteColl == "social.craftsky.feed.post" || pds.lastDeleteRkey == quoteRkey {
		t.Fatalf("DeleteRecord targeted quote post coll %q rkey %q", pds.lastDeleteColl, pds.lastDeleteRkey)
	}
}

func TestUnrepostPost_AbsentActiveRepostIsIdempotent(t *testing.T) {
	t.Parallel()
	pds := &fakePostEffectState{}
	store := &fakePostStore{target: &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"}}
	h := api.UnrepostPostHandler(store, newPDSEffectsFactory(pds), nilLogger())
	req := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.deleteCalls != 0 {
		t.Errorf("deleteCalls = %d, want 0", pds.deleteCalls)
	}
}

func TestUnrepostPost_NewPDSFailureReturns502(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{
		target:       &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeRepost: &api.InteractionRow{Rkey: "repost1", SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1", SubjectCID: "bafyPost"},
	}
	h := api.UnrepostPostHandler(store, failingPDSEffectsFactory(errors.New("session missing")), nilLogger())
	req := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "pds_unavailable" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestUnrepostPost_TargetLookupFailureReturns500(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{targetErr: errors.New("database unavailable")}
	h := api.UnrepostPostHandler(store, newPDSEffectsFactory(&fakePostEffectState{}), nilLogger())
	req := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "internal_error" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestUnrepostPost_ActiveLookupFailureReturns500(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{
		target:          &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeRepostErr: errors.New("database unavailable"),
	}
	h := api.UnrepostPostHandler(store, newPDSEffectsFactory(&fakePostEffectState{}), nilLogger())
	req := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "internal_error" {
		t.Errorf("error = %q", body.Error)
	}
}

func TestUnrepostPost_PDSRecordAlreadyGoneIsIdempotent(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{
		target:       &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeRepost: &api.InteractionRow{Rkey: "repost1", SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1", SubjectCID: "bafyPost"},
	}
	h := api.UnrepostPostHandler(store, newPDSEffectsFactory(&fakePostEffectState{deleteErr: auth.ErrRecordNotFound}), nilLogger())
	req := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
}

func TestUnrepostPost_PDSDeleteFailureReturns502(t *testing.T) {
	t.Parallel()
	store := &fakePostStore{
		target:       &api.PostTargetRef{URI: "at://did:plc:bob/social.craftsky.feed.post/post1", CID: "bafyPost"},
		activeRepost: &api.InteractionRow{Rkey: "repost1", SubjectURI: "at://did:plc:bob/social.craftsky.feed.post/post1", SubjectCID: "bafyPost"},
	}
	h := api.UnrepostPostHandler(store, newPDSEffectsFactory(&fakePostEffectState{deleteErr: errors.New("pds down")}), nilLogger())
	req := authedPostPathReq(http.MethodDelete, "/v1/posts/did:plc:bob/post1/reposts", "", "did:plc:alice")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var body envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&body)
	if body.Error != "pds_unavailable" {
		t.Errorf("error = %q", body.Error)
	}
}
