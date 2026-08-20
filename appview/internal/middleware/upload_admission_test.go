package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestUploadBodyAdmissionRejectsSaturationWithoutReadingBody(t *testing.T) {
	admission, err := NewUploadBodyAdmission(1, 20*time.Millisecond)
	if err != nil {
		t.Fatalf("construct upload admission: %v", err)
	}
	started := make(chan struct{})
	release := make(chan struct{})
	handler := admission.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))

	firstDone := make(chan int, 1)
	go func() {
		request := httptest.NewRequest(http.MethodPut, "/v1/scheduled-post-media/id", bytes.NewReader([]byte("first")))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		firstDone <- response.Code
	}()
	<-started

	body := &countingUploadBody{Reader: bytes.NewReader([]byte("must-not-be-read"))}
	request := httptest.NewRequest(http.MethodPost, "/v1/blobs/images", body)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("saturated status = %d, want 503; body=%s", response.Code, response.Body.String())
	}
	if response.Header().Get("Retry-After") != "1" {
		t.Fatalf("Retry-After = %q, want 1", response.Header().Get("Retry-After"))
	}
	if reads := body.reads.Load(); reads != 0 {
		t.Fatalf("saturated upload body reads = %d, want 0", reads)
	}
	if !request.Close || response.Header().Get("Connection") != "close" {
		t.Fatalf("saturated upload did not disable connection reuse")
	}

	close(release)
	if status := <-firstDone; status != http.StatusNoContent {
		t.Fatalf("admitted status = %d, want 204", status)
	}
}

func TestUploadBodyAdmissionBoundsWaiters(t *testing.T) {
	admission, err := NewUploadBodyAdmission(1, time.Second)
	if err != nil {
		t.Fatalf("construct upload admission: %v", err)
	}
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	handler := admission.Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		entered <- struct{}{}
		<-release
		w.WriteHeader(http.StatusNoContent)
	}))

	statuses := make(chan int, 2)
	for range 2 {
		go func() {
			request := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader([]byte("body")))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			statuses <- response.Code
		}()
	}
	<-entered
	deadline := time.Now().Add(time.Second)
	for len(admission.waiters) != 1 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if waiting := len(admission.waiters); waiting != 1 {
		t.Fatalf("bounded waiters = %d, want 1", waiting)
	}

	third := httptest.NewRecorder()
	handler.ServeHTTP(third, httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader([]byte("third"))))
	if third.Code != http.StatusServiceUnavailable {
		t.Fatalf("third status = %d, want prompt 503", third.Code)
	}

	close(release)
	for range 2 {
		if status := <-statuses; status != http.StatusNoContent {
			t.Fatalf("admitted/waiting status = %d, want 204", status)
		}
	}
}

type countingUploadBody struct {
	*bytes.Reader
	reads atomic.Int64
}

func (body *countingUploadBody) Read(buffer []byte) (int, error) {
	body.reads.Add(1)
	return body.Reader.Read(buffer)
}

func (*countingUploadBody) Close() error { return nil }

var _ io.ReadCloser = (*countingUploadBody)(nil)
