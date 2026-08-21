package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
)

type recordingProfileEffects struct {
	reads      []pdseffects.ReadRecordRequest
	puts       []pdseffects.PutRecordRequest
	readBodies map[syntax.NSID]map[string]any
	readCIDs   map[syntax.NSID]syntax.CID
	readErrors map[syntax.NSID]error
	putErrors  map[syntax.NSID]error
}

func (effects *recordingProfileEffects) ResolveExpectedOwners(
	_ context.Context,
	ownerGeneration int64,
	targets []syntax.DID,
) ([]ownerlifecycle.ExpectedOwner, error) {
	if len(targets) != 0 {
		return nil, errors.New("profile effects must be owner-only")
	}
	return []ownerlifecycle.ExpectedOwner{{Owner: "did:plc:me", Generation: ownerGeneration}}, nil
}

func (effects *recordingProfileEffects) ReadRecord(
	_ context.Context,
	request pdseffects.ReadRecordRequest,
	out any,
) (syntax.CID, error) {
	effects.reads = append(effects.reads, request)
	if err := effects.readErrors[request.Collection]; err != nil {
		return "", err
	}
	body, ok := effects.readBodies[request.Collection]
	if !ok {
		return "", auth.ErrRecordNotFound
	}
	target, ok := out.(*map[string]any)
	if !ok {
		return "", errors.New("unexpected profile read output")
	}
	*target = body
	return effects.readCIDs[request.Collection], nil
}

func (effects *recordingProfileEffects) PutRecord(
	_ context.Context,
	request pdseffects.PutRecordRequest,
) (pdseffects.RecordResult, error) {
	effects.puts = append(effects.puts, request)
	if err := effects.putErrors[request.Collection]; err != nil {
		return pdseffects.RecordResult{}, err
	}
	return pdseffects.RecordResult{
		URI: syntax.ATURI("at://" + request.Owner.String() + "/" + request.Collection.String() + "/" + request.Rkey.String()),
		CID: syntax.CID("bafy-updated"),
	}, nil
}

func (*recordingProfileEffects) DeleteRecord(
	context.Context,
	pdseffects.DeleteRecordRequest,
) (pdseffects.RecordResult, error) {
	return pdseffects.RecordResult{}, errors.New("unexpected profile delete")
}

func (*recordingProfileEffects) UploadBlob(
	context.Context,
	pdseffects.UploadBlobRequest,
) (*auth.UploadedBlob, error) {
	return nil, errors.New("unexpected profile blob upload")
}

func TestPutProfileUsesDurableConditionalEffectsWithExactLifecycleScope(t *testing.T) {
	t.Parallel()
	effects := &recordingProfileEffects{
		readBodies: map[syntax.NSID]map[string]any{
			"app.bsky.actor.profile": {
				"$type":       "app.bsky.actor.profile",
				"displayName": "Old",
				"avatar": map[string]any{
					"$type": "blob", "ref": map[string]any{"$link": "bafy-avatar"},
					"mimeType": "image/jpeg", "size": float64(42),
				},
			},
			"social.craftsky.actor.profile": {
				"$type": "social.craftsky.actor.profile", "crafts": []any{"sewing"},
			},
		},
		readCIDs: map[syntax.NSID]syntax.CID{
			"app.bsky.actor.profile":        "bafy-bsky-before",
			"social.craftsky.actor.profile": "bafy-craftsky-before",
		},
		readErrors: make(map[syntax.NSID]error),
		putErrors:  make(map[syntax.NSID]error),
	}
	var factoryOwner syntax.DID
	var factorySession string
	handler := api.PutMeProfileHandler(
		&fakeStore{row: &api.ProfileRow{DID: "did:plc:me", CreatedAt: time.Now()}},
		fakeResolver{handleFor: "me.example"},
		func(_ context.Context, owner syntax.DID, sessionID string) (pdseffects.EffectExecutor, error) {
			factoryOwner = owner
			factorySession = sessionID
			return effects, nil
		},
		api.DefaultMediaLimits(),
		nilLogger(),
	)
	req := httptest.NewRequest(http.MethodPut, "/v1/profiles/me", strings.NewReader(`{"displayName":"New","crafts":["weaving"]}`))
	ctx := middleware.WithDID(req.Context(), "did:plc:me")
	ctx = middleware.WithOwnerGeneration(ctx, 7)
	ctx = middleware.WithOAuthSessionID(ctx, "oauth-profile-session")
	ctx = ctxkeys.WithRunID(ctx, "profile-request-id")
	req = req.WithContext(ctx)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if factoryOwner != "did:plc:me" || factorySession != "oauth-profile-session" {
		t.Fatalf("factory scope = owner %q session %q", factoryOwner, factorySession)
	}
	if len(effects.reads) != 2 || len(effects.puts) != 2 {
		t.Fatalf("effect calls = reads %d puts %d", len(effects.reads), len(effects.puts))
	}
	wantExpected := []ownerlifecycle.ExpectedOwner{{Owner: "did:plc:me", Generation: 7}}
	for _, read := range effects.reads {
		if read.Owner != "did:plc:me" || read.OwnerGeneration != 7 ||
			len(read.ExpectedOwners) != 1 || read.ExpectedOwners[0] != wantExpected[0] ||
			read.Rkey != "self" {
			t.Fatalf("read scope = %+v", read)
		}
	}
	if effects.puts[0].Collection != "app.bsky.actor.profile" ||
		effects.puts[0].ExpectedCID != "bafy-bsky-before" ||
		effects.puts[1].Collection != "social.craftsky.actor.profile" ||
		effects.puts[1].ExpectedCID != "bafy-craftsky-before" {
		t.Fatalf("conditional puts = %+v", effects.puts)
	}
	for _, put := range effects.puts {
		if put.Owner != "did:plc:me" || put.OwnerGeneration != 7 ||
			len(put.ExpectedOwners) != 1 || put.ExpectedOwners[0] != wantExpected[0] ||
			put.Rkey != "self" || put.OperationID == "" || put.MutationKey != put.OperationID ||
			!strings.Contains(put.OperationID, "profile-request-id") {
			t.Fatalf("put identity/scope = %+v", put)
		}
	}
	if effects.puts[0].OperationID == effects.puts[1].OperationID {
		t.Fatalf("profile operations share durable identity %q", effects.puts[0].OperationID)
	}
	bskyBody, ok := effects.puts[0].Record.(map[string]any)
	if !ok || bskyBody["displayName"] != "New" || bskyBody["avatar"] == nil {
		t.Fatalf("bsky body = %#v", effects.puts[0].Record)
	}
	craftskyBody, ok := effects.puts[1].Record.(map[string]any)
	if !ok || len(craftskyBody["crafts"].([]string)) != 1 || craftskyBody["crafts"].([]string)[0] != "weaving" {
		t.Fatalf("craftsky body = %#v", effects.puts[1].Record)
	}
}

func TestPutProfileMissingGenerationFailsBeforeEffectFactory(t *testing.T) {
	t.Parallel()
	factoryCalls := 0
	handler := api.PutMeProfileHandler(
		&fakeStore{},
		fakeResolver{},
		func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
			factoryCalls++
			return nil, errors.New("must not be called")
		},
		api.DefaultMediaLimits(),
		nilLogger(),
	)
	req := httptest.NewRequest(http.MethodPut, "/v1/profiles/me", strings.NewReader(`{"displayName":"New"}`))
	ctx := middleware.WithDID(req.Context(), "did:plc:me")
	ctx = middleware.WithOAuthSessionID(ctx, "oauth-profile-session")
	req = req.WithContext(ctx)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusServiceUnavailable || factoryCalls != 0 {
		t.Fatalf("status/factory calls = %d/%d, body = %s", recorder.Code, factoryCalls, recorder.Body.String())
	}
}

func TestPutProfileFactoryFailureOccursBeforePDSReadOrWrite(t *testing.T) {
	t.Parallel()
	factoryCalls := 0
	handler := api.PutMeProfileHandler(
		&fakeStore{},
		fakeResolver{},
		func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
			factoryCalls++
			return nil, errors.New("session resume failed")
		},
		api.DefaultMediaLimits(),
		nilLogger(),
	)
	req := httptest.NewRequest(http.MethodPut, "/v1/profiles/me", strings.NewReader(`{"displayName":"New"}`))
	ctx := middleware.WithDID(req.Context(), "did:plc:me")
	ctx = middleware.WithOwnerGeneration(ctx, 7)
	ctx = middleware.WithOAuthSessionID(ctx, "oauth-profile-session")
	req = req.WithContext(ctx)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadGateway || factoryCalls != 1 {
		t.Fatalf("status/factory calls = %d/%d, body = %s", recorder.Code, factoryCalls, recorder.Body.String())
	}
}

func TestPutProfileAmbiguousFirstWriteReturnsBoundedPartialResult(t *testing.T) {
	t.Parallel()
	effects := &recordingProfileEffects{
		readBodies: map[syntax.NSID]map[string]any{
			"app.bsky.actor.profile":        {"$type": "app.bsky.actor.profile"},
			"social.craftsky.actor.profile": {"$type": "social.craftsky.actor.profile"},
		},
		readCIDs: map[syntax.NSID]syntax.CID{
			"app.bsky.actor.profile":        "bafy-bsky-before",
			"social.craftsky.actor.profile": "bafy-craftsky-before",
		},
		readErrors: make(map[syntax.NSID]error),
		putErrors: map[syntax.NSID]error{
			"app.bsky.actor.profile": &pdseffects.OutcomeAmbiguousError{
				OperationID: "secret-operation-id", ExactKey: "at://secret/profile/self",
			},
		},
	}
	handler := api.PutMeProfileHandler(
		&fakeStore{},
		fakeResolver{},
		func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
			return effects, nil
		},
		api.DefaultMediaLimits(),
		nilLogger(),
	)
	req := httptest.NewRequest(http.MethodPut, "/v1/profiles/me", strings.NewReader(`{"displayName":"New"}`))
	ctx := middleware.WithDID(req.Context(), "did:plc:me")
	ctx = middleware.WithOwnerGeneration(ctx, 7)
	ctx = middleware.WithOAuthSessionID(ctx, "oauth-profile-session")
	req = req.WithContext(ctx)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadGateway || len(effects.puts) != 2 {
		t.Fatalf("status/puts = %d/%d, body = %s", recorder.Code, len(effects.puts), recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "secret-operation-id") || strings.Contains(recorder.Body.String(), "at://secret") {
		t.Fatalf("response leaked durable effect identity: %s", recorder.Body.String())
	}
	var response envelope.Error
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Error != "pds_write_partial" || response.Fields["bsky"] != "failed" || response.Fields["craftsky"] != "ok" {
		t.Fatalf("partial response = %+v", response)
	}
}

func TestPutProfileCraftSkyReadFailureOccursBeforeEitherMutation(t *testing.T) {
	t.Parallel()
	effects := &recordingProfileEffects{
		readBodies: map[syntax.NSID]map[string]any{
			"app.bsky.actor.profile": {"$type": "app.bsky.actor.profile"},
		},
		readCIDs: map[syntax.NSID]syntax.CID{
			"app.bsky.actor.profile": "bafy-bsky-before",
		},
		readErrors: map[syntax.NSID]error{
			"social.craftsky.actor.profile": errors.New("CraftSky read unavailable"),
		},
		putErrors: make(map[syntax.NSID]error),
	}
	handler := api.PutMeProfileHandler(
		&fakeStore{},
		fakeResolver{},
		func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
			return effects, nil
		},
		api.DefaultMediaLimits(),
		nilLogger(),
	)
	req := httptest.NewRequest(http.MethodPut, "/v1/profiles/me", strings.NewReader(`{"displayName":"New"}`))
	ctx := middleware.WithDID(req.Context(), "did:plc:me")
	ctx = middleware.WithOwnerGeneration(ctx, 7)
	ctx = middleware.WithOAuthSessionID(ctx, "oauth-profile-session")
	req = req.WithContext(ctx)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadGateway || len(effects.reads) != 2 || len(effects.puts) != 0 {
		t.Fatalf(
			"status/reads/puts = %d/%d/%d, body = %s",
			recorder.Code,
			len(effects.reads),
			len(effects.puts),
			recorder.Body.String(),
		)
	}
	var response envelope.Error
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Error != "pds_read_failed" {
		t.Fatalf("read failure response = %+v", response)
	}
}

var _ pdseffects.EffectExecutor = (*recordingProfileEffects)(nil)
