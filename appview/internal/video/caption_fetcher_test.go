package video

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

const (
	captionTestDID = syntax.DID("did:web:alice.example")
	captionTestCID = syntax.CID("bafyreicaption+encoded")
)

type captionDirectory struct {
	identity *identity.Identity
	err      error
}

func (directory captionDirectory) LookupDID(context.Context, syntax.DID) (*identity.Identity, error) {
	return directory.identity, directory.err
}

func (captionDirectory) LookupHandle(context.Context, syntax.Handle) (*identity.Identity, error) {
	return nil, errors.New("not used")
}

func (captionDirectory) Lookup(context.Context, syntax.AtIdentifier) (*identity.Identity, error) {
	return nil, errors.New("not used")
}

func (captionDirectory) Purge(context.Context, syntax.AtIdentifier) error { return nil }

type captionOriginValidator struct {
	origin *url.URL
	err    error
	raw    string
}

func (validator *captionOriginValidator) ValidateOrigin(_ context.Context, raw string) (*url.URL, error) {
	validator.raw = raw
	return validator.origin, validator.err
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

type errorReadCloser struct {
	err error
}

func (body errorReadCloser) Read([]byte) (int, error) { return 0, body.err }
func (errorReadCloser) Close() error                  { return nil }

type cancelingReadCloser struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func (body cancelingReadCloser) Read([]byte) (int, error) {
	body.cancel()
	return 0, body.ctx.Err()
}

func (cancelingReadCloser) Close() error { return nil }

func TestNewCaptionFetcherRejectsNilDependencies(t *testing.T) {
	t.Parallel()
	directory := captionTestDirectory("https://pds.example")
	client := &http.Client{}
	origins := &captionOriginValidator{origin: mustURL(t, "https://pds.example")}

	tests := []struct {
		name      string
		directory identity.Directory
		client    *http.Client
		origins   PDSOriginValidator
	}{
		{name: "directory", client: client, origins: origins},
		{name: "client", directory: directory, origins: origins},
		{name: "origin validator", directory: directory, client: client},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			fetcher, err := NewCaptionFetcher(test.directory, test.client, test.origins)
			if fetcher != nil || !errors.Is(err, ErrCaptionUnavailable) {
				t.Fatalf("NewCaptionFetcher() = %#v, %v", fetcher, err)
			}
		})
	}
}

func TestCaptionFetcherBuildsValidatedBlobRequest(t *testing.T) {
	t.Parallel()
	origins := &captionOriginValidator{origin: mustURL(t, "https://validated.example/pds/root?preserved=yes")}
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet {
			t.Fatalf("method = %q", request.Method)
		}
		if request.URL.Scheme != "https" || request.URL.Host != "validated.example" {
			t.Fatalf("origin = %s", request.URL)
		}
		if request.URL.Path != "/pds/root/xrpc/com.atproto.sync.getBlob" {
			t.Fatalf("path = %q", request.URL.Path)
		}
		wantQuery := url.Values{
			"cid":       {captionTestCID.String()},
			"did":       {captionTestDID.String()},
			"preserved": {"yes"},
		}
		if got := request.URL.Query(); !equalValues(got, wantQuery) {
			t.Fatalf("query = %v, want %v", got, wantQuery)
		}
		if !strings.Contains(request.URL.RawQuery, "did=did%3Aweb%3Aalice.example") || !strings.Contains(request.URL.RawQuery, "cid=bafyreicaption%2Bencoded") {
			t.Fatalf("query was not encoded: %q", request.URL.RawQuery)
		}
		return captionResponse(http.StatusOK, "text/vtt; charset=utf-8", 7, io.NopCloser(strings.NewReader("WEBVTT\n"))), nil
	})}
	fetcher := mustCaptionFetcher(t, captionTestDirectory("https://untrusted.example/service"), client, origins)

	body, err := fetcher.Fetch(context.Background(), captionTestDID, captionTestCID)
	if err != nil || string(body) != "WEBVTT\n" {
		t.Fatalf("Fetch() = %q, %v", body, err)
	}
	if origins.raw != "https://untrusted.example/service" {
		t.Fatalf("validated raw origin = %q", origins.raw)
	}
}

func TestCaptionFetcherRejectsIdentityAndOriginFailuresBeforeRequest(t *testing.T) {
	t.Parallel()
	wantFailure := errors.New("sensitive upstream failure")
	tests := []struct {
		name      string
		directory identity.Directory
		origin    *url.URL
		originErr error
	}{
		{name: "lookup error", directory: captionDirectory{err: wantFailure}, origin: mustURL(t, "https://pds.example")},
		{name: "missing identity", directory: captionDirectory{}, origin: mustURL(t, "https://pds.example")},
		{name: "missing PDS service", directory: captionTestDirectory(""), origin: mustURL(t, "https://pds.example")},
		{name: "rejected origin", directory: captionTestDirectory("https://pds.example"), originErr: wantFailure},
		{name: "nil validated origin", directory: captionTestDirectory("https://pds.example")},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			called := false
			client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				called = true
				return nil, errors.New("unexpected request")
			})}
			fetcher := mustCaptionFetcher(t, test.directory, client, &captionOriginValidator{origin: test.origin, err: test.originErr})
			_, err := fetcher.Fetch(context.Background(), captionTestDID, captionTestCID)
			if !errors.Is(err, ErrCaptionUnavailable) || called {
				t.Fatalf("Fetch() error = %v, request called = %t", err, called)
			}
		})
	}
}

func TestCaptionFetcherRejectsRedirectWithoutMutatingSourceClient(t *testing.T) {
	t.Parallel()
	redirectChecks := 0
	wantRedirectError := errors.New("source redirect policy")
	requests := 0
	client := &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			requests++
			if requests > 1 {
				t.Fatal("caption fetch followed redirect")
			}
			response := captionResponse(http.StatusFound, "text/plain", 0, http.NoBody)
			response.Header.Set("Location", "https://redirect.example/caption.vtt")
			response.Request = request
			return response, nil
		}),
		CheckRedirect: func(*http.Request, []*http.Request) error {
			redirectChecks++
			return wantRedirectError
		},
	}
	fetcher := mustCaptionFetcher(t, captionTestDirectory("https://pds.example"), client, &captionOriginValidator{origin: mustURL(t, "https://pds.example")})

	_, err := fetcher.Fetch(context.Background(), captionTestDID, captionTestCID)
	if !errors.Is(err, ErrCaptionUnavailable) || requests != 1 || redirectChecks != 0 {
		t.Fatalf("Fetch() error=%v requests=%d redirect checks=%d", err, requests, redirectChecks)
	}
	if err := client.CheckRedirect(nil, nil); !errors.Is(err, wantRedirectError) || redirectChecks != 1 {
		t.Fatalf("source CheckRedirect was mutated: error=%v checks=%d", err, redirectChecks)
	}
}

func TestCaptionFetcherValidatesStatusAndContentType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		status      int
		contentType string
		wantOK      bool
	}{
		{name: "VTT", status: http.StatusOK, contentType: "text/vtt", wantOK: true},
		{name: "VTT parameters", status: http.StatusOK, contentType: "text/vtt; charset=utf-8", wantOK: true},
		{name: "non-200", status: http.StatusNotFound, contentType: "text/vtt"},
		{name: "missing content type", status: http.StatusOK},
		{name: "wrong content type", status: http.StatusOK, contentType: "text/plain"},
		{name: "malformed content type", status: http.StatusOK, contentType: `text/vtt; charset="`},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			fetcher := captionFetcherForResponse(t, captionResponse(test.status, test.contentType, 7, io.NopCloser(strings.NewReader("WEBVTT\n"))))
			body, err := fetcher.Fetch(context.Background(), captionTestDID, captionTestCID)
			if test.wantOK {
				if err != nil || string(body) != "WEBVTT\n" {
					t.Fatalf("Fetch() = %q, %v", body, err)
				}
				return
			}
			if !errors.Is(err, ErrCaptionUnavailable) || body != nil {
				t.Fatalf("Fetch() = %q, %v", body, err)
			}
		})
	}
}

func TestCaptionFetcherEnforcesDeclaredAndStreamedSizeLimits(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		contentLength int64
		bodySize      int
		wantOK        bool
	}{
		{name: "exact declared limit", contentLength: maxCaptionBytes, bodySize: maxCaptionBytes, wantOK: true},
		{name: "declared oversized", contentLength: maxCaptionBytes + 1, bodySize: 0},
		{name: "exact chunked limit", contentLength: -1, bodySize: maxCaptionBytes, wantOK: true},
		{name: "chunked oversized", contentLength: -1, bodySize: maxCaptionBytes + 1},
		{name: "incorrect small declaration", contentLength: 1, bodySize: maxCaptionBytes + 1},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			response := captionResponse(http.StatusOK, "text/vtt", test.contentLength, io.NopCloser(strings.NewReader(strings.Repeat("x", test.bodySize))))
			body, err := captionFetcherForResponse(t, response).Fetch(context.Background(), captionTestDID, captionTestCID)
			if test.wantOK {
				if err != nil || len(body) != maxCaptionBytes {
					t.Fatalf("Fetch() length=%d error=%v", len(body), err)
				}
				return
			}
			if !errors.Is(err, ErrCaptionUnavailable) || body != nil {
				t.Fatalf("Fetch() length=%d error=%v", len(body), err)
			}
		})
	}
}

func TestCaptionFetcherHandlesReadAndRequestErrors(t *testing.T) {
	t.Parallel()
	readFailure := errors.New("read failure")
	requestFailure := errors.New("request failure")
	tests := []struct {
		name      string
		transport roundTripFunc
		want      error
	}{
		{
			name: "request error",
			transport: func(*http.Request) (*http.Response, error) {
				return nil, requestFailure
			},
			want: ErrCaptionUnavailable,
		},
		{
			name: "read error",
			transport: func(*http.Request) (*http.Response, error) {
				return captionResponse(http.StatusOK, "text/vtt", -1, errorReadCloser{err: readFailure}), nil
			},
			want: ErrCaptionUnavailable,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			client := &http.Client{Transport: test.transport}
			fetcher := mustCaptionFetcher(t, captionTestDirectory("https://pds.example"), client, &captionOriginValidator{origin: mustURL(t, "https://pds.example")})
			_, err := fetcher.Fetch(context.Background(), captionTestDID, captionTestCID)
			if !errors.Is(err, test.want) || errors.Is(err, readFailure) || errors.Is(err, requestFailure) {
				t.Fatalf("Fetch() error = %v", err)
			}
		})
	}
}

func TestCaptionFetcherPropagatesCancellation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		duringRead bool
	}{
		{name: "during request"},
		{name: "during body read", duringRead: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				if test.duringRead {
					return captionResponse(http.StatusOK, "text/vtt", -1, cancelingReadCloser{ctx: request.Context(), cancel: cancel}), nil
				}
				cancel()
				<-request.Context().Done()
				return nil, request.Context().Err()
			})}
			fetcher := mustCaptionFetcher(t, captionTestDirectory("https://pds.example"), client, &captionOriginValidator{origin: mustURL(t, "https://pds.example")})
			_, err := fetcher.Fetch(ctx, captionTestDID, captionTestCID)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("Fetch() error = %v", err)
			}
		})
	}
}

func TestNilCaptionFetcherFailsClosed(t *testing.T) {
	t.Parallel()
	var fetcher *CaptionFetcher
	if _, err := fetcher.Fetch(context.Background(), captionTestDID, captionTestCID); !errors.Is(err, ErrCaptionUnavailable) {
		t.Fatalf("Fetch() error = %v", err)
	}
}

func captionTestDirectory(endpoint string) identity.Directory {
	value := &identity.Identity{DID: captionTestDID}
	if endpoint != "" {
		value.Services = map[string]identity.ServiceEndpoint{
			"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: endpoint},
		}
	}
	return captionDirectory{identity: value}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	value, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	return value
}

func mustCaptionFetcher(t *testing.T, directory identity.Directory, client *http.Client, origins PDSOriginValidator) *CaptionFetcher {
	t.Helper()
	fetcher, err := NewCaptionFetcher(directory, client, origins)
	if err != nil {
		t.Fatalf("NewCaptionFetcher: %v", err)
	}
	return fetcher
}

func captionFetcherForResponse(t *testing.T, response *http.Response) *CaptionFetcher {
	t.Helper()
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return response, nil
	})}
	return mustCaptionFetcher(t, captionTestDirectory("https://pds.example"), client, &captionOriginValidator{origin: mustURL(t, "https://pds.example")})
}

func captionResponse(status int, contentType string, contentLength int64, body io.ReadCloser) *http.Response {
	header := make(http.Header)
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	return &http.Response{StatusCode: status, Header: header, ContentLength: contentLength, Body: body}
}

func equalValues(left, right url.Values) bool {
	for key := range left {
		slices.Sort(left[key])
	}
	for key := range right {
		slices.Sort(right[key])
	}
	return len(left) == len(right) && left.Encode() == right.Encode()
}
