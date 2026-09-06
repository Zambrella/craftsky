package video

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

const maxCaptionBytes = 20_000

var ErrCaptionUnavailable = errors.New("video caption unavailable")

type PDSOriginValidator interface {
	ValidateOrigin(context.Context, string) (*url.URL, error)
}

type CaptionFetcher struct {
	directory identity.Directory
	client    *http.Client
	origins   PDSOriginValidator
}

func NewCaptionFetcher(directory identity.Directory, client *http.Client, origins PDSOriginValidator) (*CaptionFetcher, error) {
	if directory == nil || client == nil || origins == nil {
		return nil, ErrCaptionUnavailable
	}
	return &CaptionFetcher{directory: directory, client: client, origins: origins}, nil
}

func (fetcher *CaptionFetcher) Fetch(ctx context.Context, did syntax.DID, captionCID syntax.CID) ([]byte, error) {
	if fetcher == nil || fetcher.directory == nil || fetcher.client == nil || fetcher.origins == nil {
		return nil, ErrCaptionUnavailable
	}
	identityValue, err := fetcher.directory.LookupDID(ctx, did)
	if err != nil || identityValue == nil || identityValue.PDSEndpoint() == "" {
		return nil, ErrCaptionUnavailable
	}
	origin, err := fetcher.origins.ValidateOrigin(ctx, identityValue.PDSEndpoint())
	if err != nil {
		return nil, ErrCaptionUnavailable
	}
	endpoint := origin.JoinPath("xrpc", "com.atproto.sync.getBlob")
	query := endpoint.Query()
	query.Set("did", did.String())
	query.Set("cid", captionCID.String())
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, ErrCaptionUnavailable
	}
	noRedirectClient := *fetcher.client
	noRedirectClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := noRedirectClient.Do(request)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, ErrCaptionUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, ErrCaptionUnavailable
	}
	contentType, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || contentType != "text/vtt" || response.ContentLength > maxCaptionBytes {
		return nil, ErrCaptionUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxCaptionBytes+1))
	if err != nil || len(body) > maxCaptionBytes {
		return nil, ErrCaptionUnavailable
	}
	return body, nil
}
