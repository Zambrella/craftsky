package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/ctxkeys"
	"social.craftsky/appview/internal/linkpreview"
)

// IT-001: the singular handler strips source fragments and returns the fixed
// camelCase preview contract with standard padded base64 thumbnail bytes.
func TestLinkPreviewHandlerContract(t *testing.T) {
	t.Parallel()

	finalURL, _ := url.Parse("https://final.example/pattern#redirect-fragment")
	service := &fakeLinkPreviewService{preview: linkpreview.Preview{
		URL:         finalURL,
		Title:       "Pattern",
		Description: "A knitting pattern",
		Thumbnail: &linkpreview.Thumbnail{
			Bytes: []byte{1, 2}, MIMEType: "image/png", Width: 3, Height: 2,
		},
	}}
	handler := api.LinkPreviewHandler(service, true)
	request := httptest.NewRequest(http.MethodPost, "/v1/link-previews", strings.NewReader(`{"url":"https://source.example/start#source-fragment"}`))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	if service.raw != "https://source.example/start" {
		t.Fatalf("service URL = %q, want fragmentless source", service.raw)
	}
	var body struct {
		URL         string `json:"url"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Thumbnail   struct {
			Bytes    string `json:"bytes"`
			MIMEType string `json:"mimeType"`
			Width    int    `json:"width"`
			Height   int    `json:"height"`
		} `json:"thumbnail"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.URL != finalURL.String() || body.Title != "Pattern" || body.Description != "A knitting pattern" {
		t.Fatalf("response = %+v", body)
	}
	if body.Thumbnail.Bytes != "AQI=" || body.Thumbnail.MIMEType != "image/png" || body.Thumbnail.Width != 3 || body.Thumbnail.Height != 2 {
		t.Fatalf("thumbnail = %+v", body.Thumbnail)
	}

	badRequest := httptest.NewRequest(http.MethodPost, "/v1/link-previews", strings.NewReader(`{}`))
	badRequest = badRequest.WithContext(ctxkeys.WithRunID(badRequest.Context(), "request-123"))
	badRecorder := httptest.NewRecorder()
	handler.ServeHTTP(badRecorder, badRequest)
	var failure map[string]any
	_ = json.Unmarshal(badRecorder.Body.Bytes(), &failure)
	if badRecorder.Code != http.StatusBadRequest || failure["error"] != "invalid_request" || failure["requestId"] != "request-123" {
		t.Fatalf("failure status/body = %d %#v", badRecorder.Code, failure)
	}
}

func TestLinkPreviewHandlerMapsStalledPageBodyToTimeout(t *testing.T) {
	t.Parallel()
	handler := api.LinkPreviewHandler(linkpreview.NewService(stalledPageFetcher{}), true)
	request := httptest.NewRequest(http.MethodPost, "/v1/link-previews", strings.NewReader(`{"url":"https://source.example/pattern"}`))
	request = request.WithContext(ctxkeys.WithRunID(request.Context(), "request-timeout"))
	ctx, cancel := context.WithTimeout(request.Context(), 20*time.Millisecond)
	defer cancel()
	request = request.WithContext(ctx)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode timeout response: %v", err)
	}
	if recorder.Code != http.StatusGatewayTimeout || body["error"] != "link_preview_timeout" || body["requestId"] != "request-timeout" {
		t.Fatalf("status/body = %d %#v, want 504 timeout envelope", recorder.Code, body)
	}
}

type stalledPageFetcher struct{}

func (stalledPageFetcher) FetchPage(ctx context.Context, raw string) (*http.Response, *url.URL, error) {
	finalURL, _ := url.Parse(raw)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": {"text/html"}},
		Body:       stalledPageBody{ctx: ctx},
	}, finalURL, nil
}

func (stalledPageFetcher) FetchImage(context.Context, string) (*http.Response, *url.URL, error) {
	return nil, nil, errors.New("image fetch must not start")
}

type stalledPageBody struct {
	ctx context.Context
}

func (body stalledPageBody) Read([]byte) (int, error) {
	<-body.ctx.Done()
	return 0, body.ctx.Err()
}

func (stalledPageBody) Close() error { return nil }

type fakeLinkPreviewService struct {
	preview linkpreview.Preview
	err     error
	raw     string
}

func (s *fakeLinkPreviewService) FetchPreview(_ context.Context, raw string) (linkpreview.Preview, error) {
	s.raw = raw
	return s.preview, s.err
}
