package routes

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/observability"
)

func TestV1UploadRoutesShareAdmissionBeforeHandlerBodyRead(t *testing.T) {
	admission, err := middleware.NewUploadBodyAdmission(1, 20*time.Millisecond)
	if err != nil {
		t.Fatalf("construct upload admission: %v", err)
	}
	pass := func(next http.Handler) http.Handler { return next }
	mw := v1Middleware{
		authCurrentMember: pass,
		authRecovery:      pass,
		deviceID:          pass,
		member:            pass,
		bodyLimit: middleware.BodyLimitConfig{
			UploadBytes:       1024,
			UploadReadTimeout: time.Second,
		},
		uploadAdmission: admission,
		rateLimit:       map[RateClass]func(http.Handler) http.Handler{},
		observer:        observability.New(observability.Config{}),
	}
	policy := RoutePolicy{
		Method:      http.MethodPut,
		PathPattern: "/v1/upload",
		RateClass:   RateClassUpload,
		BodyKind:    BodyUpload,
		AccessClass: AccessAnonymous,
	}

	entered := make(chan struct{})
	release := make(chan struct{})
	first := mw.wrap(policy, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			t.Errorf("read admitted upload: %v", err)
		}
		close(entered)
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))
	firstDone := make(chan int, 1)
	go func() {
		response := httptest.NewRecorder()
		first.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/v1/upload", bytes.NewReader([]byte("first"))))
		firstDone <- response.Code
	}()
	<-entered

	var reads atomic.Int64
	secondCalled := atomic.Bool{}
	second := mw.wrap(policy, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secondCalled.Store(true)
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	body := &routeCountingUploadBody{Reader: bytes.NewReader([]byte("second")), reads: &reads}
	request := httptest.NewRequest(http.MethodPut, "/v1/upload", body)
	response := httptest.NewRecorder()
	second.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("second upload status = %d, want 503; body=%s", response.Code, response.Body.String())
	}
	if secondCalled.Load() {
		t.Fatal("saturated upload reached its handler")
	}
	if got := reads.Load(); got != 0 {
		t.Fatalf("saturated upload body reads = %d, want 0", got)
	}

	close(release)
	if status := <-firstDone; status != http.StatusNoContent {
		t.Fatalf("first upload status = %d, want 204", status)
	}
}

type routeCountingUploadBody struct {
	*bytes.Reader
	reads *atomic.Int64
}

func (body *routeCountingUploadBody) Read(buffer []byte) (int, error) {
	body.reads.Add(1)
	return body.Reader.Read(buffer)
}

func (*routeCountingUploadBody) Close() error { return nil }

var _ io.ReadCloser = (*routeCountingUploadBody)(nil)
