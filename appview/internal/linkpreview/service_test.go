package linkpreview

import (
	"bytes"
	"context"
	"errors"
	"image"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// IT-004: the preview pipeline combines bounded page decoding and best-effort
// image validation under page-phase and total request budgets.
func TestServiceFetchPreviewPipeline(t *testing.T) {
	t.Parallel()

	t.Run("legacy page and image fallback", func(t *testing.T) {
		t.Parallel()
		validPNG := encodeImage(t, "png", image.NewRGBA(image.Rect(0, 0, 3, 2)))
		fetcher := &scriptedResourceFetcher{
			page: resourceResult{response: resourceResponse(http.StatusOK, "text/html", []byte("<head><meta charset=windows-1252><title>caf\xe9</title><meta property=og:image content='https://images.example/bad.jpg'><meta property=og:image content='https://images.example/good.png'><meta property=og:image content='https://images.example/third.png'><meta property=og:image content='https://images.example/fourth.png'></head>"))},
			images: map[string]resourceResult{
				"https://images.example/bad.jpg":  {response: resourceResponse(http.StatusOK, "image/jpeg", []byte("corrupt"))},
				"https://images.example/good.png": {response: resourceResponse(http.StatusOK, "image/jpeg", validPNG)},
			},
		}
		preview, err := NewService(fetcher).FetchPreview(context.Background(), "https://page.example/pattern")
		if err != nil {
			t.Fatalf("FetchPreview(): %v", err)
		}
		if preview.Title != "café" || preview.URL.String() != "https://page.example/pattern" {
			t.Fatalf("preview = %#v", preview)
		}
		if preview.Thumbnail == nil || preview.Thumbnail.MIMEType != "image/png" || preview.Thumbnail.Width != 3 || preview.Thumbnail.Height != 2 {
			t.Fatalf("thumbnail = %#v", preview.Thumbnail)
		}
		if got := fetcher.imageRequests; len(got) != 2 || got[0] != "https://images.example/bad.jpg" || got[1] != "https://images.example/good.png" {
			t.Fatalf("image requests = %v", got)
		}
		if fetcher.pageBudget <= 0 || fetcher.pageBudget > pagePhaseTimeout {
			t.Fatalf("page budget = %v, want <= %v", fetcher.pageBudget, pagePhaseTimeout)
		}
	})

	t.Run("image exhaustion preserves metadata and caps attempts", func(t *testing.T) {
		t.Parallel()
		fetcher := &scriptedResourceFetcher{
			page:   resourceResult{response: resourceResponse(http.StatusOK, "application/xhtml+xml; charset=utf-8", []byte(`<head><title>Pattern</title><meta property=og:image content="https://images.example/1"><meta property=og:image content="https://images.example/2"><meta property=og:image content="https://images.example/3"><meta property=og:image content="https://images.example/4"></head>`))},
			images: map[string]resourceResult{},
		}
		preview, err := NewService(fetcher).FetchPreview(context.Background(), "https://page.example/pattern")
		if err != nil {
			t.Fatalf("FetchPreview(): %v", err)
		}
		if preview.Title != "Pattern" || preview.Thumbnail != nil || len(fetcher.imageRequests) != 3 {
			t.Fatalf("preview = %#v, image requests = %v", preview, fetcher.imageRequests)
		}
	})

	t.Run("YouTube uses oEmbed metadata and its thumbnail", func(t *testing.T) {
		t.Parallel()
		validPNG := encodeImage(t, "png", image.NewRGBA(image.Rect(0, 0, 16, 9)))
		fetcher := &scriptedResourceFetcher{
			json: resourceResult{response: resourceResponse(http.StatusOK, "application/json", []byte(`{
				"title": "A knitting tutorial",
				"author_name": "A Crafter",
				"thumbnail_url": "https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg"
			}`))},
			images: map[string]resourceResult{
				"https://i.ytimg.com/vi/dQw4w9WgXcQ/hqdefault.jpg": {
					response: resourceResponse(http.StatusOK, "image/png", validPNG),
				},
			},
		}

		preview, err := NewService(fetcher).FetchPreview(context.Background(), "https://youtu.be/dQw4w9WgXcQ")
		if err != nil {
			t.Fatalf("FetchPreview(): %v", err)
		}
		if preview.Title != "A knitting tutorial" || preview.Description != "A Crafter" {
			t.Fatalf("preview metadata = %#v", preview)
		}
		if preview.URL.String() != "https://youtu.be/dQw4w9WgXcQ" {
			t.Fatalf("preview URL = %q", preview.URL)
		}
		if preview.Thumbnail == nil || preview.Thumbnail.Width != 16 || preview.Thumbnail.Height != 9 {
			t.Fatalf("preview thumbnail = %#v", preview.Thumbnail)
		}
		if len(fetcher.jsonRequests) != 1 || fetcher.pageRequests != 0 {
			t.Fatalf("JSON requests = %v, page requests = %d", fetcher.jsonRequests, fetcher.pageRequests)
		}
	})

	t.Run("private YouTube video is not presented as a rich preview", func(t *testing.T) {
		t.Parallel()
		fetcher := &scriptedResourceFetcher{
			json:   resourceResult{response: resourceResponse(http.StatusForbidden, "application/json", nil)},
			images: map[string]resourceResult{},
		}

		_, err := NewService(fetcher).FetchPreview(context.Background(), "https://youtu.be/pQ9NBUuwDMg")

		if !errors.Is(err, ErrUnsupported) {
			t.Fatalf("FetchPreview() error = %v, want ErrUnsupported", err)
		}
		if fetcher.pageRequests != 0 || len(fetcher.imageRequests) != 0 {
			t.Fatalf("page requests = %d, image requests = %v", fetcher.pageRequests, fetcher.imageRequests)
		}
	})

	for _, tt := range []struct {
		name        string
		result      resourceResult
		wantTimeout bool
	}{
		{name: "non-2xx", result: resourceResult{response: resourceResponse(http.StatusBadGateway, "text/html", nil)}},
		{name: "non-HTML", result: resourceResult{response: resourceResponse(http.StatusOK, "application/json", []byte(`{}`))}},
		{name: "oversized unclosed head", result: resourceResult{response: resourceResponse(http.StatusOK, "text/html", append([]byte("<head>"), bytes.Repeat([]byte("x"), maxPageBytes)...))}},
		{name: "page timeout", result: resourceResult{err: context.DeadlineExceeded}, wantTimeout: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fetcher := &scriptedResourceFetcher{page: tt.result}
			_, err := NewService(fetcher).FetchPreview(context.Background(), "https://page.example/pattern")
			if err == nil {
				t.Fatal("FetchPreview() error = nil, want failure")
			}
			if tt.wantTimeout && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("FetchPreview() error = %v, want deadline exceeded", err)
			}
		})
	}
}

func TestServiceThumbnailSkipsWorkAfterTotalContextCancellation(t *testing.T) {
	t.Parallel()
	validPNG := encodeImage(t, "png", image.NewRGBA(image.Rect(0, 0, 3, 2)))
	fetcher := &scriptedResourceFetcher{images: map[string]resourceResult{
		"https://images.example/late.png": {response: resourceResponse(http.StatusOK, "image/png", validPNG)},
	}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	thumbnail := NewService(fetcher).fetchThumbnail(ctx, "https://images.example/late.png")

	if thumbnail != nil || len(fetcher.imageRequests) != 0 {
		t.Fatalf("thumbnail=%#v requests=%v, want no work after cancellation", thumbnail, fetcher.imageRequests)
	}
}

func TestServiceThumbnailIgnoresDecodeResultAfterTotalDeadline(t *testing.T) {
	t.Parallel()
	nearLimit := bytes.Repeat([]byte{1}, maxThumbnailBytes)
	fetcher := &scriptedResourceFetcher{images: map[string]resourceResult{
		"https://images.example/near-limit.png": {response: resourceResponse(http.StatusOK, "image/png", nearLimit)},
	}}
	service := NewService(fetcher)
	decodeStarted := make(chan struct{})
	releaseDecode := make(chan struct{})
	service.validateThumbnail = func(input []byte) (Thumbnail, error) {
		if len(input) != maxThumbnailBytes {
			t.Errorf("decode input bytes=%d, want %d", len(input), maxThumbnailBytes)
		}
		close(decodeStarted)
		<-releaseDecode
		return Thumbnail{Bytes: input, MIMEType: "image/png", Width: 4000, Height: 4000}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	result := make(chan *Thumbnail, 1)
	go func() {
		result <- service.fetchThumbnail(ctx, "https://images.example/near-limit.png")
	}()
	<-decodeStarted

	select {
	case thumbnail := <-result:
		close(releaseDecode)
		if thumbnail != nil {
			t.Fatalf("thumbnail=%#v, want ignored result after deadline", thumbnail)
		}
	case <-time.After(200 * time.Millisecond):
		close(releaseDecode)
		<-result
		t.Fatal("fetchThumbnail waited for decode after total deadline")
	}
}

func TestServiceThumbnailDecodeConcurrencyIsBounded(t *testing.T) {
	t.Parallel()
	nearLimit := bytes.Repeat([]byte{1}, maxThumbnailBytes)
	service := NewService(staticImageFetcher{body: nearLimit})
	releaseDecode := make(chan struct{})
	started := make(chan struct{}, 5)
	var active atomic.Int32
	var maximum atomic.Int32
	service.validateThumbnail = func(input []byte) (Thumbnail, error) {
		current := active.Add(1)
		for {
			prior := maximum.Load()
			if current <= prior || maximum.CompareAndSwap(prior, current) {
				break
			}
		}
		started <- struct{}{}
		<-releaseDecode
		active.Add(-1)
		return Thumbnail{Bytes: input, MIMEType: "image/png", Width: 4000, Height: 4000}, nil
	}
	var workers sync.WaitGroup
	for index := 0; index < 5; index++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			service.fetchThumbnail(context.Background(), "https://images.example/near-limit.png")
		}()
	}
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()
	startCount := 0
waitForStarts:
	for startCount < 3 {
		select {
		case <-started:
			startCount++
		case <-timer.C:
			break waitForStarts
		}
	}
	if got := maximum.Load(); got > 2 {
		close(releaseDecode)
		workers.Wait()
		t.Fatalf("concurrent decodes=%d, want at most 2", got)
	}
	close(releaseDecode)
	workers.Wait()
}

type staticImageFetcher struct {
	body []byte
}

func (fetcher staticImageFetcher) FetchPage(context.Context, string) (*http.Response, *url.URL, error) {
	return nil, nil, errors.New("page fetch must not start")
}

func (fetcher staticImageFetcher) FetchImage(_ context.Context, raw string) (*http.Response, *url.URL, error) {
	finalURL, _ := url.Parse(raw)
	return resourceResponse(http.StatusOK, "image/png", fetcher.body), finalURL, nil
}

func (fetcher staticImageFetcher) FetchJSON(context.Context, string) (*http.Response, *url.URL, error) {
	return nil, nil, errors.New("JSON fetch must not start")
}

type resourceResult struct {
	response *http.Response
	finalURL *url.URL
	err      error
}

type scriptedResourceFetcher struct {
	page          resourceResult
	json          resourceResult
	images        map[string]resourceResult
	imageRequests []string
	jsonRequests  []string
	pageRequests  int
	pageBudget    time.Duration
}

func (f *scriptedResourceFetcher) FetchPage(ctx context.Context, raw string) (*http.Response, *url.URL, error) {
	f.pageRequests++
	if deadline, ok := ctx.Deadline(); ok {
		f.pageBudget = time.Until(deadline)
	}
	return completeResourceResult(f.page, raw)
}

func (f *scriptedResourceFetcher) FetchImage(_ context.Context, raw string) (*http.Response, *url.URL, error) {
	f.imageRequests = append(f.imageRequests, raw)
	return completeResourceResult(f.images[raw], raw)
}

func (f *scriptedResourceFetcher) FetchJSON(_ context.Context, raw string) (*http.Response, *url.URL, error) {
	f.jsonRequests = append(f.jsonRequests, raw)
	return completeResourceResult(f.json, raw)
}

func completeResourceResult(result resourceResult, raw string) (*http.Response, *url.URL, error) {
	if result.err != nil {
		return nil, nil, result.err
	}
	if result.response == nil {
		return nil, nil, errors.New("scripted image failure")
	}
	if result.finalURL == nil {
		result.finalURL, _ = url.Parse(raw)
	}
	return result.response, result.finalURL, nil
}

func resourceResponse(status int, contentType string, body []byte) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {contentType}},
		Body:       io.NopCloser(strings.NewReader(string(body))),
	}
}
