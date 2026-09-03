package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
)

type fakeBlobEffects struct {
	uploadRequests []pdseffects.UploadBlobRequest
	uploadResp     *auth.UploadedBlob
	uploadErr      error
	factoryOwner   syntax.DID
	factorySession string
}

type stubImageValidator struct {
	err error
}

func (validator stubImageValidator) Validate(context.Context, string, []byte) (api.ValidatedScheduledImage, error) {
	return api.ValidatedScheduledImage{Format: "jpeg", Width: 1, Height: 1}, validator.err
}

var acceptingImageValidator = stubImageValidator{}

func (*fakeBlobEffects) ResolveExpectedOwners(
	context.Context,
	int64,
	[]syntax.DID,
) ([]ownerlifecycle.ExpectedOwner, error) {
	return nil, errors.New("unexpected blob target resolution")
}

func (*fakeBlobEffects) ReadRecord(
	context.Context,
	pdseffects.ReadRecordRequest,
	any,
) (syntax.CID, error) {
	return "", errors.New("unexpected blob record read")
}

func (*fakeBlobEffects) PutRecord(
	context.Context,
	pdseffects.PutRecordRequest,
) (pdseffects.RecordResult, error) {
	return pdseffects.RecordResult{}, errors.New("unexpected blob record put")
}

func (*fakeBlobEffects) DeleteRecord(
	context.Context,
	pdseffects.DeleteRecordRequest,
) (pdseffects.RecordResult, error) {
	return pdseffects.RecordResult{}, errors.New("unexpected blob record delete")
}

func (effects *fakeBlobEffects) UploadBlob(
	_ context.Context,
	request pdseffects.UploadBlobRequest,
) (*auth.UploadedBlob, error) {
	effects.uploadRequests = append(effects.uploadRequests, request)
	if effects.uploadErr != nil {
		return nil, effects.uploadErr
	}
	if effects.uploadResp != nil {
		return effects.uploadResp, nil
	}
	return &auth.UploadedBlob{
		Raw: map[string]any{
			"$type": "blob", "ref": map[string]any{"$link": "bafk-default"},
			"mimeType": request.MIME, "size": float64(len(request.Bytes)),
		},
		CID: "bafk-default", MIME: request.MIME, Size: int64(len(request.Bytes)),
	}, nil
}

func blobEffectsFactory(effects *fakeBlobEffects) pdseffects.ExecutorFactory {
	return func(_ context.Context, owner syntax.DID, sessionID string) (pdseffects.EffectExecutor, error) {
		effects.factoryOwner = owner
		effects.factorySession = sessionID
		return effects, nil
	}
}

func TestImageBlobUpload_HappyPath_ForwardsToPDSAndReturnsMetadata(t *testing.T) {
	t.Parallel()
	effects := &fakeBlobEffects{
		uploadResp: &auth.UploadedBlob{
			Raw:  map[string]any{"$type": "blob", "ref": map[string]any{"$link": "bafkimage"}, "mimeType": "image/jpeg", "size": float64(253496)},
			CID:  "bafkimage",
			MIME: "image/jpeg",
			Size: 253496,
		},
	}
	h := api.ImageBlobUploadHandler(blobEffectsFactory(effects), api.DefaultMediaLimits(), acceptingImageValidator, nilLogger())
	body := []byte("fake-jpeg-bytes")
	req := authedReq(http.MethodPost, "/v1/blobs/images", "", "did:plc:alice")
	req.Body = ioNopCloser{Reader: bytes.NewReader(body)}
	req.Header.Set("Content-Type", "image/jpeg")
	ctx := middleware.WithOAuthSessionID(req.Context(), "oauth-blob-session")
	ctx = ctxkeys.WithRunID(ctx, "blob-request-id")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if len(effects.uploadRequests) != 1 {
		t.Fatalf("uploadCalls = %d, want 1", len(effects.uploadRequests))
	}
	if effects.uploadRequests[0].MIME != "image/jpeg" {
		t.Fatalf("mime = %q, want image/jpeg", effects.uploadRequests[0].MIME)
	}
	if !bytes.Equal(effects.uploadRequests[0].Bytes, body) {
		t.Fatalf("forwarded bytes mismatch")
	}
	request := effects.uploadRequests[0]
	if effects.factoryOwner != "did:plc:alice" || effects.factorySession != "oauth-blob-session" ||
		request.Owner != "did:plc:alice" || request.OwnerGeneration != 1 ||
		len(request.ExpectedOwners) != 1 || request.ExpectedOwners[0] != (ownerlifecycle.ExpectedOwner{Owner: "did:plc:alice", Generation: 1}) ||
		request.OperationID == "" || request.MutationKey != request.OperationID ||
		!strings.Contains(request.OperationID, "blob-request-id") {
		t.Fatalf("durable upload scope = factory %q/%q request %+v", effects.factoryOwner, effects.factorySession, request)
	}
	var resp struct {
		Blob map[string]any `json:"blob"`
		CID  string         `json:"cid"`
		MIME string         `json:"mime"`
		Size int64          `json:"size"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.CID != "bafkimage" || resp.MIME != "image/jpeg" || resp.Size != 253496 {
		t.Fatalf("resp = %+v", resp)
	}
}

func TestImageBlobUpload_UnsupportedMIME_RejectsWithoutCallingPDS(t *testing.T) {
	t.Parallel()
	effects := &fakeBlobEffects{}
	h := api.ImageBlobUploadHandler(blobEffectsFactory(effects), api.DefaultMediaLimits(), acceptingImageValidator, nilLogger())
	req := authedReq(http.MethodPost, "/v1/blobs/images", "", "did:plc:alice")
	req.Body = ioNopCloser{Reader: bytes.NewReader([]byte("gif-bytes"))}
	req.Header.Set("Content-Type", "image/gif")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if len(effects.uploadRequests) != 0 {
		t.Fatalf("uploadCalls = %d, want 0", len(effects.uploadRequests))
	}
	var errBody envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&errBody)
	if errBody.Error != "validation_failed" {
		t.Fatalf("error = %q", errBody.Error)
	}
}

func TestImageBlobUpload_InvalidImage_RejectsBeforeCreatingPDSEffects(t *testing.T) {
	t.Parallel()
	factoryCalls := 0
	h := api.ImageBlobUploadHandler(
		func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
			factoryCalls++
			return &fakeBlobEffects{}, nil
		},
		api.DefaultMediaLimits(),
		stubImageValidator{err: api.ErrScheduledImageInvalid},
		nilLogger(),
	)
	req := withOAuthSession(authedReq(http.MethodPost, "/v1/blobs/images", "not-an-image", "did:plc:alice"))
	req.Header.Set("Content-Type", "image/jpeg")
	recorder := httptest.NewRecorder()

	h.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnprocessableEntity || factoryCalls != 0 {
		t.Fatalf("status/factory calls = %d/%d, body = %s", recorder.Code, factoryCalls, recorder.Body.String())
	}
	var response envelope.Error
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Error != "validation_failed" || response.Fields["image"] == "" {
		t.Fatalf("response = %#v", response)
	}
}

func TestImageBlobUpload_OversizedBody_RejectsWithoutCallingPDS(t *testing.T) {
	t.Parallel()
	effects := &fakeBlobEffects{}
	h := api.ImageBlobUploadHandler(blobEffectsFactory(effects), api.DefaultMediaLimits(), acceptingImageValidator, nilLogger())
	over := bytes.Repeat([]byte("a"), int(api.MaxImageUploadBytes+1))
	req := authedReq(http.MethodPost, "/v1/blobs/images", "", "did:plc:alice")
	req.Body = ioNopCloser{Reader: bytes.NewReader(over)}
	req.Header.Set("Content-Type", "image/jpeg")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if len(effects.uploadRequests) != 0 {
		t.Fatalf("uploadCalls = %d, want 0", len(effects.uploadRequests))
	}
	var errBody envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&errBody)
	if errBody.Error != "validation_failed" {
		t.Fatalf("error = %q", errBody.Error)
	}
}

func TestImageBlobUpload_FailureLogsExcludeImageBytesAndToken(t *testing.T) {
	t.Parallel()
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	effects := &fakeBlobEffects{uploadErr: errors.New("pds down")}
	h := api.ImageBlobUploadHandler(blobEffectsFactory(effects), api.DefaultMediaLimits(), acceptingImageValidator, logger)

	const sentinelBytes = "SENSITIVE_IMAGE_BYTES"
	const sentinelToken = "SENSITIVE_TOKEN"
	req := authedReq(http.MethodPost, "/v1/blobs/images", "", "did:plc:alice")
	req.Body = io.NopCloser(strings.NewReader(sentinelBytes))
	req.Header.Set("Content-Type", "image/jpeg")
	req.Header.Set("Authorization", "Bearer "+sentinelToken)
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	out := logs.String()
	if strings.Contains(out, sentinelBytes) {
		t.Fatalf("logs leaked image bytes: %s", out)
	}
	if strings.Contains(out, sentinelToken) {
		t.Fatalf("logs leaked token: %s", out)
	}
}

func TestImageBlobUpload_PDSSessionExpiredReturns401(t *testing.T) {
	t.Parallel()
	effects := &fakeBlobEffects{uploadErr: auth.ErrPDSSessionExpired}
	h := api.ImageBlobUploadHandler(blobEffectsFactory(effects), api.DefaultMediaLimits(), acceptingImageValidator, nilLogger())
	req := authedReq(http.MethodPost, "/v1/blobs/images", "", "did:plc:alice")
	req.Body = io.NopCloser(strings.NewReader("jpeg-bytes"))
	req.Header.Set("Content-Type", "image/jpeg")
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

func TestImageBlobUploadMissingGenerationFailsBeforeEffectFactory(t *testing.T) {
	t.Parallel()
	factoryCalls := 0
	h := api.ImageBlobUploadHandler(
		func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
			factoryCalls++
			return nil, errors.New("must not be called")
		},
		api.DefaultMediaLimits(),
		acceptingImageValidator,
		nilLogger(),
	)
	req := httptest.NewRequest(http.MethodPost, "/v1/blobs/images", strings.NewReader("jpeg-bytes"))
	ctx := middleware.WithDID(req.Context(), "did:plc:alice")
	ctx = middleware.WithOAuthSessionID(ctx, "oauth-blob-session")
	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "image/jpeg")
	recorder := httptest.NewRecorder()

	h.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusServiceUnavailable || factoryCalls != 0 {
		t.Fatalf("status/factory calls = %d/%d, body = %s", recorder.Code, factoryCalls, recorder.Body.String())
	}
}

func TestImageBlobUploadFactoryFailureOccursBeforeDurableUpload(t *testing.T) {
	t.Parallel()
	factoryCalls := 0
	h := api.ImageBlobUploadHandler(
		func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) {
			factoryCalls++
			return nil, errors.New("session resume failed")
		},
		api.DefaultMediaLimits(),
		acceptingImageValidator,
		nilLogger(),
	)
	req := withOAuthSession(authedReq(http.MethodPost, "/v1/blobs/images", "jpeg-bytes", "did:plc:alice"))
	req.Header.Set("Content-Type", "image/jpeg")
	recorder := httptest.NewRecorder()

	h.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadGateway || factoryCalls != 1 {
		t.Fatalf("status/factory calls = %d/%d, body = %s", recorder.Code, factoryCalls, recorder.Body.String())
	}
}

func TestImageBlobUploadDurableAmbiguityAndConflictFailSafely(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		err  error
	}{
		{
			name: "ambiguous",
			err: &pdseffects.OutcomeAmbiguousError{
				OperationID: "secret-blob-operation", ExactKey: "pds-blob://secret",
			},
		},
		{
			name: "conflict",
			err: &pdseffects.ConflictError{
				OperationID: "secret-blob-operation", ExactKey: "pds-blob://secret",
			},
		},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			effects := &fakeBlobEffects{uploadErr: test.err}
			h := api.ImageBlobUploadHandler(blobEffectsFactory(effects), api.DefaultMediaLimits(), acceptingImageValidator, nilLogger())
			req := withOAuthSession(authedReq(http.MethodPost, "/v1/blobs/images", "jpeg-bytes", "did:plc:alice"))
			req.Header.Set("Content-Type", "image/jpeg")
			recorder := httptest.NewRecorder()

			h.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusBadGateway || len(effects.uploadRequests) != 1 {
				t.Fatalf("status/uploads = %d/%d, body = %s", recorder.Code, len(effects.uploadRequests), recorder.Body.String())
			}
			if strings.Contains(recorder.Body.String(), "secret-blob-operation") || strings.Contains(recorder.Body.String(), "pds-blob://") {
				t.Fatalf("response leaked durable effect identity: %s", recorder.Body.String())
			}
			var response envelope.Error
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Error != "pds_write_failed" {
				t.Fatalf("error response = %+v", response)
			}
		})
	}
}

type ioNopCloser struct{ Reader *bytes.Reader }

func (n ioNopCloser) Read(p []byte) (int, error) { return n.Reader.Read(p) }
func (n ioNopCloser) Close() error               { return nil }

var _ pdseffects.EffectExecutor = (*fakeBlobEffects)(nil)
