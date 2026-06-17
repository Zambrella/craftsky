package api_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/auth"
)

func TestImageBlobUpload_HappyPath_ForwardsToPDSAndReturnsMetadata(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{
		uploadResp: &auth.UploadedBlob{
			Raw:  map[string]any{"$type": "blob", "ref": map[string]any{"$link": "bafkimage"}, "mimeType": "image/jpeg", "size": float64(253496)},
			CID:  "bafkimage",
			MIME: "image/jpeg",
			Size: 253496,
		},
	}
	h := api.ImageBlobUploadHandler(newPDSFactory(pds), api.DefaultMediaLimits(), nilLogger())
	body := []byte("fake-jpeg-bytes")
	req := authedReq(http.MethodPost, "/v1/blobs/images", "", "did:plc:alice")
	req.Body = ioNopCloser{Reader: bytes.NewReader(body)}
	req.Header.Set("Content-Type", "image/jpeg")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.uploadCalls != 1 {
		t.Fatalf("uploadCalls = %d, want 1", pds.uploadCalls)
	}
	if pds.lastUploadMIME != "image/jpeg" {
		t.Fatalf("mime = %q, want image/jpeg", pds.lastUploadMIME)
	}
	if !bytes.Equal(pds.lastUploadBody, body) {
		t.Fatalf("forwarded bytes mismatch")
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
	pds := &fakePDS{}
	h := api.ImageBlobUploadHandler(newPDSFactory(pds), api.DefaultMediaLimits(), nilLogger())
	req := authedReq(http.MethodPost, "/v1/blobs/images", "", "did:plc:alice")
	req.Body = ioNopCloser{Reader: bytes.NewReader([]byte("gif-bytes"))}
	req.Header.Set("Content-Type", "image/gif")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.uploadCalls != 0 {
		t.Fatalf("uploadCalls = %d, want 0", pds.uploadCalls)
	}
	var errBody envelope.Error
	_ = json.NewDecoder(rr.Body).Decode(&errBody)
	if errBody.Error != "validation_failed" {
		t.Fatalf("error = %q", errBody.Error)
	}
}

func TestImageBlobUpload_OversizedBody_RejectsWithoutCallingPDS(t *testing.T) {
	t.Parallel()
	pds := &fakePDS{}
	h := api.ImageBlobUploadHandler(newPDSFactory(pds), api.DefaultMediaLimits(), nilLogger())
	over := bytes.Repeat([]byte("a"), int(api.MaxImageUploadBytes+1))
	req := authedReq(http.MethodPost, "/v1/blobs/images", "", "did:plc:alice")
	req.Body = ioNopCloser{Reader: bytes.NewReader(over)}
	req.Header.Set("Content-Type", "image/jpeg")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if pds.uploadCalls != 0 {
		t.Fatalf("uploadCalls = %d, want 0", pds.uploadCalls)
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
	pds := &fakePDS{uploadErr: errors.New("pds down")}
	h := api.ImageBlobUploadHandler(newPDSFactory(pds), api.DefaultMediaLimits(), logger)

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
	pds := &fakePDS{uploadErr: auth.ErrPDSSessionExpired}
	h := api.ImageBlobUploadHandler(newPDSFactory(pds), api.DefaultMediaLimits(), nilLogger())
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

type ioNopCloser struct{ Reader *bytes.Reader }

func (n ioNopCloser) Read(p []byte) (int, error) { return n.Reader.Read(p) }
func (n ioNopCloser) Close() error               { return nil }
