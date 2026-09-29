package imagesafety

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"net/url"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/ipfs/go-cid"
	_ "golang.org/x/image/webp"
)

var ErrBlobUnavailable = errors.New("image blob unavailable")

type OriginValidator interface {
	ValidateOrigin(context.Context, string) (*url.URL, error)
}

type PDSBlobFetcher struct {
	directory identity.Directory
	client    *http.Client
	origins   OriginValidator
	maxBytes  int64
}

func NewPDSBlobFetcher(
	directory identity.Directory,
	client *http.Client,
	origins OriginValidator,
	maxBytes int64,
) (*PDSBlobFetcher, error) {
	if directory == nil || client == nil || origins == nil || maxBytes <= 0 {
		return nil, ErrBlobUnavailable
	}
	return &PDSBlobFetcher{directory: directory, client: client, origins: origins, maxBytes: maxBytes}, nil
}

func (fetcher *PDSBlobFetcher) Fetch(ctx context.Context, source BlobSource) ([]byte, error) {
	if fetcher == nil || source.DID == "" || source.BlobCID == "" || source.DeclaredSize < 0 ||
		source.DeclaredSize > fetcher.maxBytes || !allowedImageMIME(source.DeclaredMIME) {
		return nil, ErrBlobUnavailable
	}
	identityValue, err := fetcher.directory.LookupDID(ctx, source.DID)
	if err != nil || identityValue == nil || identityValue.PDSEndpoint() == "" {
		return nil, ErrBlobUnavailable
	}
	origin, err := fetcher.origins.ValidateOrigin(ctx, identityValue.PDSEndpoint())
	if err != nil {
		return nil, ErrBlobUnavailable
	}
	endpoint := origin.JoinPath("xrpc", "com.atproto.sync.getBlob")
	query := endpoint.Query()
	query.Set("did", source.DID.String())
	query.Set("cid", source.BlobCID.String())
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, ErrBlobUnavailable
	}
	client := *fetcher.client
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(request)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return nil, ErrBlobUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.ContentLength > fetcher.maxBytes {
		return nil, ErrBlobUnavailable
	}
	responseMIME, _, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if err != nil || responseMIME != source.DeclaredMIME || !allowedImageMIME(responseMIME) {
		return nil, ErrBlobUnavailable
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, fetcher.maxBytes+1))
	if err != nil || int64(len(body)) > fetcher.maxBytes || int64(len(body)) != source.DeclaredSize {
		return nil, ErrBlobUnavailable
	}
	expected, err := cid.Parse(source.BlobCID.String())
	if err != nil {
		return nil, ErrBlobUnavailable
	}
	actual, err := expected.Prefix().Sum(body)
	if err != nil || !actual.Equals(expected) {
		return nil, ErrBlobUnavailable
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil || canonicalImageFormat(format) != responseMIME {
		return nil, ErrBlobUnavailable
	}
	return body, nil
}

func allowedImageMIME(value string) bool {
	return value == "image/jpeg" || value == "image/png" || value == "image/webp"
}

func canonicalImageFormat(format string) string {
	switch format {
	case "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	default:
		return ""
	}
}
