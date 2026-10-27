package linkpreview

import (
	"context"
	"image"
	"net/http"
	"testing"
)

func TestProfileLinkPreviewResolvesFaviconAndReturnsBoundedPNG(t *testing.T) {
	png := encodeImage(t, "png", image.NewRGBA(image.Rect(0, 0, 256, 128)))
	fetcher := &scriptedResourceFetcher{page: resourceResult{response: resourceResponse(http.StatusOK, "text/html", []byte(`<head><link rel="shortcut icon" href="/site.png"></head>`))}, images: map[string]resourceResult{"https://page.example/site.png": {response: resourceResponse(http.StatusOK, "image/png", png)}}}
	preview, err := NewService(fetcher).FetchProfileLinkPreview(context.Background(), "https://page.example/shop")
	if err != nil || !preview.Reachable || preview.Icon == nil {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
	if preview.Icon.MIMEType != "image/png" || preview.Icon.Width != 64 || preview.Icon.Height != 64 || len(preview.Icon.Bytes) > 100_000 {
		t.Fatal("icon exceeds the profile-link policy")
	}
}

func TestProfileLinkPreviewMissingDestinationIsAdvisory(t *testing.T) {
	fetcher := &scriptedResourceFetcher{page: resourceResult{response: resourceResponse(http.StatusNotFound, "text/html", nil)}}
	preview, err := NewService(fetcher).FetchProfileLinkPreview(context.Background(), "https://page.example/shop")
	if err != nil || preview.Reachable || preview.Icon != nil {
		t.Fatalf("preview=%+v err=%v", preview, err)
	}
}
