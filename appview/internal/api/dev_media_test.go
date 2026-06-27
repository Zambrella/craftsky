package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDevMediaHandler_GeneratesJPEGForKnownSafeName(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("GET /v1/dev/media/{name}", DevMediaHandler())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", "/v1/dev/media/knit-cardigan-moss", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "image/jpeg") {
		t.Fatalf("content-type = %q, want image/jpeg", got)
	}
	if body := rec.Body.Bytes(); len(body) < 2 || body[0] != 0xff || body[1] != 0xd8 {
		t.Fatalf("body does not look like a JPEG, len=%d", len(body))
	}
}

func TestDevMediaHandler_RejectsPathTraversal(t *testing.T) {
	handler := DevMediaHandler()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/v1/dev/media/..", nil)
	req.SetPathValue("name", "..")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
