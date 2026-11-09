package video

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestPDSBlobLimitsDiscoveryAndCache(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		want       int64
	}{
		{"advertised", `{"blobUploadLimit":50}`, 200, 50},
		{"larger than app", `{"blobUploadLimit":900000000}`, 200, maxVideoBlobBytes},
		{"absent", `{}`, 200, maxVideoBlobBytes},
		{"zero", `{"blobUploadLimit":0}`, 200, maxVideoBlobBytes},
		{"negative", `{"blobUploadLimit":-1}`, 200, maxVideoBlobBytes},
		{"malformed", `{"blobUploadLimit":"50"}`, 200, maxVideoBlobBytes},
		{"unsupported", `{}`, 404, maxVideoBlobBytes},
		{"redirect", `{}`, 302, maxVideoBlobBytes},
		{"bounded", strings.Repeat("x", maxLimitsResponseBytes+1), 200, maxVideoBlobBytes},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			p := NewPDSBlobLimits(&http.Client{Transport: bunnyRoundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Path != "/xrpc/com.atproto.server.describeServer" || r.Header.Get("Authorization") != "" || r.Header.Get("DPoP") != "" {
					t.Fatal("capability discovery must be public")
				}
				return &http.Response{StatusCode: tc.status, Header: http.Header{"Location": []string{"https://other.example"}}, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			})})
			for range 2 {
				if got := p.Get(t.Context(), "https://pds.example"); got != tc.want {
					t.Fatalf("got %d want %d", got, tc.want)
				}
			}
			if calls != 1 {
				t.Fatal("did not cache discovery", calls)
			}
			p.Get(t.Context(), "https://migrated.example")
			if calls != 2 {
				t.Fatal("reused previous PDS capability")
			}
			p.mu.Lock()
			entry := p.cache["https://pds.example"]
			entry.expires = time.Now().Add(-time.Second)
			p.cache["https://pds.example"] = entry
			p.mu.Unlock()
			p.Get(t.Context(), "https://pds.example")
			if calls != 3 {
				t.Fatal("expired capability not refreshed")
			}
		})
	}
}

func TestPDSBlobLimitsNetworkFailureIsOptional(t *testing.T) {
	p := NewPDSBlobLimits(&http.Client{Transport: bunnyRoundTrip(func(*http.Request) (*http.Response, error) { return nil, errors.New("offline") })})
	if got := p.Get(t.Context(), "https://pds.example"); got != maxVideoBlobBytes {
		t.Fatal(got)
	}
}

func TestBunnySelectsFittingRendition(t *testing.T) {
	const mp4 = "\x00\x00\x00\x18ftypisomsmall"
	for _, unknownLength := range []bool{false, true} {
		t.Run(map[bool]string{false: "content length", true: "streamed"}[unknownLength], func(t *testing.T) {
			var paths []string
			b := &Bunny{cdn: "test.b-cdn.net", client: &http.Client{Transport: bunnyRoundTrip(func(r *http.Request) (*http.Response, error) {
				paths = append(paths, r.URL.Path)
				body := mp4
				if strings.Contains(r.URL.Path, "720p") {
					body += strings.Repeat("x", 40)
				}
				length := int64(len(body))
				if unknownLength {
					length = -1
				}
				return &http.Response{StatusCode: 200, ContentLength: length, Body: io.NopCloser(strings.NewReader(body))}, nil
			})}}
			file, err := b.downloadFittingRendition(context.Background(), "video", "720p,1080p,480p,360p", 20)
			if err != nil {
				t.Fatal(err)
			}
			name := file.Name()
			defer os.Remove(name)
			defer file.Close()
			body, _ := io.ReadAll(file)
			if string(body) != mp4 || strings.Join(paths, ",") != "/video/play_720p.mp4,/video/play_480p.mp4" {
				t.Fatal("wrong rendition", paths, string(body))
			}
			paths = nil
			exact, err := b.downloadFittingRendition(t.Context(), "video", "1080p,360p", int64(len(mp4)))
			if err != nil {
				t.Fatal("exact-size rendition rejected", err)
			}
			exact.Close()
			os.Remove(exact.Name())
			if strings.Join(paths, ",") != "/video/play_1080p.mp4" {
				t.Fatal("existing rendition preference changed", paths)
			}
			file, err = b.downloadFittingRendition(t.Context(), "video", "720p,480p", 12)
			if file != nil || !errors.Is(err, errVideoBlobTooLarge) {
				t.Fatal("oversized rendition accepted", err)
			}
		})
	}
}
