package linkpreview

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	pagePhaseTimeout          = 6 * time.Second
	totalTimeout              = 10 * time.Second
	maxConcurrentImageDecodes = 2
)

var (
	ErrUnsupported = errors.New("unsupported link preview response")
	ErrUpstream    = errors.New("link preview upstream failed")
)

type ResourceFetcher interface {
	FetchPage(context.Context, string) (*http.Response, *url.URL, error)
	FetchImage(context.Context, string) (*http.Response, *url.URL, error)
}

type Service struct {
	fetcher           ResourceFetcher
	validateThumbnail func([]byte) (Thumbnail, error)
	decodeSlots       chan struct{}
}

type Preview struct {
	URL         *url.URL
	Title       string
	Description string
	Thumbnail   *Thumbnail
}

func NewService(fetcher ResourceFetcher) *Service {
	return &Service{
		fetcher: fetcher, validateThumbnail: ValidateThumbnail,
		decodeSlots: make(chan struct{}, maxConcurrentImageDecodes),
	}
}

func (s *Service) FetchPreview(ctx context.Context, raw string) (Preview, error) {
	ctx, cancelTotal := context.WithTimeout(ctx, totalTimeout)
	defer cancelTotal()
	pageCtx, cancelPage := context.WithTimeout(ctx, pagePhaseTimeout)
	response, finalURL, err := s.fetcher.FetchPage(pageCtx, raw)
	if err != nil {
		cancelPage()
		return Preview{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		cancelPage()
		return Preview{}, ErrUpstream
	}
	contentType := response.Header.Get("Content-Type")
	if !isHTMLContentType(contentType) {
		cancelPage()
		return Preview{}, ErrUnsupported
	}
	head, err := DecodeHTMLHead(response.Body, contentType)
	pageErr := pageCtx.Err()
	cancelPage()
	if err != nil {
		if pageErr != nil {
			return Preview{}, pageErr
		}
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return Preview{}, err
		}
		return Preview{}, ErrUnsupported
	}
	metadata, err := ExtractMetadata(head, finalURL)
	if err != nil {
		return Preview{}, ErrUnsupported
	}
	preview := Preview{URL: metadata.URL, Title: metadata.Title, Description: metadata.Description}
	for _, candidate := range metadata.ThumbnailCandidates {
		if ctx.Err() != nil {
			break
		}
		thumbnail := s.fetchThumbnail(ctx, candidate.String())
		if thumbnail != nil {
			preview.Thumbnail = thumbnail
			break
		}
	}
	return preview, nil
}

func (s *Service) fetchThumbnail(ctx context.Context, raw string) *Thumbnail {
	if ctx.Err() != nil {
		return nil
	}
	response, _, err := s.fetcher.FetchImage(ctx, raw)
	if err != nil {
		return nil
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil
	}
	input, err := io.ReadAll(io.LimitReader(response.Body, maxThumbnailBytes+1))
	if err != nil || len(input) > maxThumbnailBytes {
		return nil
	}
	if ctx.Err() != nil {
		return nil
	}
	type decodeResult struct {
		thumbnail Thumbnail
		err       error
	}
	select {
	case s.decodeSlots <- struct{}{}:
	case <-ctx.Done():
		return nil
	}
	decoded := make(chan decodeResult, 1)
	go func() {
		defer func() { <-s.decodeSlots }()
		thumbnail, err := s.validateThumbnail(input)
		decoded <- decodeResult{thumbnail: thumbnail, err: err}
	}()
	select {
	case <-ctx.Done():
		return nil
	case result := <-decoded:
		if result.err != nil || ctx.Err() != nil {
			return nil
		}
		return &result.thumbnail
	}
}

func isHTMLContentType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(value))
	return err == nil && (mediaType == "text/html" || mediaType == "application/xhtml+xml")
}
