package api

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"

	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/scheduledposts"
)

// TestScheduledImageReleaseConcurrentUploads is executed again inside the
// release-runtime media evidence image by scripts/appview-check. It drives the
// real upload handler and registered WebP decoder with one maximum-size
// admitted body. A concurrent second upload is rejected before reading while
// the first retains its bytes in the remote-write stage.
func TestScheduledImageReleaseConcurrentUploads(t *testing.T) {
	owner := syntax.DID("did:plc:release-media-probe")
	encoded := mustDecodeBase64(t, worstCaseWebPBase64)
	payload := make([]byte, int(DefaultMaxImageUploadBytes))
	copy(payload, encoded)
	observer := &releaseImageProbeObserver{}
	limits := DefaultImageDecodeLimits()
	validator, err := NewImageValidatorWithObserver(limits, observer)
	if err != nil {
		t.Fatalf("construct release image validator: %v", err)
	}
	service := &releaseImageProbeMediaService{
		owner: owner, entered: make(chan struct{}), release: make(chan struct{}),
	}
	handler := PutScheduledMediaHandler(
		service,
		DefaultMediaLimits(),
		validator,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	admission, err := middleware.NewUploadBodyAdmission(
		limits.MaxConcurrentDecodes,
		limits.AdmissionWait,
	)
	if err != nil {
		t.Fatalf("construct release upload admission: %v", err)
	}
	handler = admission.Handler(handler)

	firstResponse := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, releaseMediaProbeRequest(
			owner,
			"00000000-0000-4000-8000-000000000711",
			bytes.NewReader(payload),
		))
		firstResponse <- response
	}()
	<-service.entered

	secondBody := &releaseUnreadBody{Reader: bytes.NewReader(payload)}
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, releaseMediaProbeRequest(
		owner,
		"00000000-0000-4000-8000-000000000712",
		secondBody,
	))
	if second.Code != http.StatusServiceUnavailable {
		t.Fatalf("saturated release upload status = %d, want 503; body=%s", second.Code, second.Body.String())
	}
	if reads := secondBody.reads.Load(); reads != 0 {
		t.Fatalf("saturated release upload reads = %d, want 0", reads)
	}
	close(service.release)
	if response := <-firstResponse; response.Code != http.StatusOK {
		t.Fatalf("admitted release upload status = %d, want 200; body=%s", response.Code, response.Body.String())
	}
	if calls := service.callCount(); calls != 1 {
		t.Fatalf("scheduled media Put calls = %d, want 1", calls)
	}
	if maximum := observer.maximum.Load(); maximum != 1 {
		t.Fatalf("simultaneous full decodes = %d, want exactly 1", maximum)
	}
}

func releaseMediaProbeRequest(
	owner syntax.DID,
	mediaID string,
	body io.Reader,
) *http.Request {
	request := httptest.NewRequest(
		http.MethodPut,
		"/v1/scheduled-post-media/"+mediaID,
		body,
	)
	request.SetPathValue("mediaId", mediaID)
	request.Header.Set("Content-Type", "image/webp")
	return request.WithContext(middleware.WithOwnerGeneration(
		middleware.WithDID(request.Context(), owner),
		1,
	))
}

type releaseUnreadBody struct {
	*bytes.Reader
	reads atomic.Int64
}

func (body *releaseUnreadBody) Read(buffer []byte) (int, error) {
	body.reads.Add(1)
	return body.Reader.Read(buffer)
}

func (*releaseUnreadBody) Close() error { return nil }

type releaseImageProbeObserver struct {
	maximum atomic.Int64
}

func (observer *releaseImageProbeObserver) ObserveScheduledImageValidation(
	result string,
	_ string,
	_ time.Duration,
	inFlight int,
) {
	if result != "started" {
		return
	}
	for {
		current := observer.maximum.Load()
		if int64(inFlight) <= current || observer.maximum.CompareAndSwap(current, int64(inFlight)) {
			return
		}
	}
}

type releaseImageProbeMediaService struct {
	owner   syntax.DID
	entered chan struct{}
	release chan struct{}
	mu      sync.Mutex
	calls   int
}

func (service *releaseImageProbeMediaService) Put(
	_ context.Context,
	params scheduledposts.PutPrivateMediaParams,
) (scheduledposts.PrivateMedia, error) {
	service.mu.Lock()
	service.calls++
	service.mu.Unlock()
	close(service.entered)
	<-service.release
	if params.OwnerDID != service.owner || params.OwnerGeneration != 1 {
		return scheduledposts.PrivateMedia{}, scheduledposts.ErrScheduledMediaNotFound
	}
	return scheduledposts.PrivateMedia{
		ID:        params.ID,
		OwnerDID:  params.OwnerDID,
		MIMEType:  params.MIMEType,
		SizeBytes: int64(len(params.Bytes)),
		State:     "ready",
		BlobCID:   syntax.CID("bafkreleasemediaprobe"),
		ObjectKey: "release-media-probe",
	}, nil
}

func (service *releaseImageProbeMediaService) Open(
	context.Context,
	syntax.DID,
	uuid.UUID,
) (scheduledposts.OpenedPrivateMedia, error) {
	return scheduledposts.OpenedPrivateMedia{}, scheduledposts.ErrScheduledMediaNotFound
}

func (service *releaseImageProbeMediaService) Delete(
	context.Context,
	syntax.DID,
	uuid.UUID,
	time.Time,
	...int64,
) error {
	return scheduledposts.ErrScheduledMediaNotFound
}

func (service *releaseImageProbeMediaService) callCount() int {
	service.mu.Lock()
	defer service.mu.Unlock()
	return service.calls
}
