package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

func newTestIndigoPDSClient(
	t *testing.T,
	host string,
	onSessionExpired ...func(context.Context),
) *IndigoPDSClient {
	t.Helper()
	newAPIClient := func() *atclient.APIClient {
		client := atclient.NewAPIClient(host)
		client.Client = &http.Client{Transport: http.DefaultTransport}
		return client
	}
	var expired func(context.Context)
	if len(onSessionExpired) > 0 {
		expired = onSessionExpired[0]
	}
	client, err := NewIndigoPDSClient(newAPIClient(), newAPIClient(), expired)
	if err != nil {
		t.Fatalf("NewIndigoPDSClient: %v", err)
	}
	return client
}

type pdsRoundTripFunc func(*http.Request) (*http.Response, error)

func (roundTrip pdsRoundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func TestNewIndigoPDSClientFailsClosedWithoutSeparatePurposeClients(t *testing.T) {
	t.Parallel()
	api := atclient.NewAPIClient("https://pds.example")
	api.Client = &http.Client{Transport: http.DefaultTransport}
	if client, err := NewIndigoPDSClient(api, api, nil); err == nil || client != nil {
		t.Fatalf("client/error = %#v/%v, want fail-closed construction", client, err)
	}
}

func TestIndigoPDSClientUploadUsesUploadPurposeClient(t *testing.T) {
	t.Parallel()

	jsonCalls := 0
	uploadCalls := 0
	jsonAPI := atclient.NewAPIClient("https://pds.example")
	jsonAPI.Client = &http.Client{Transport: pdsRoundTripFunc(func(*http.Request) (*http.Response, error) {
		jsonCalls++
		return nil, errors.New("JSON client must not handle an upload")
	})}
	uploadAPI := atclient.NewAPIClient("https://pds.example")
	uploadAPI.Client = &http.Client{Transport: pdsRoundTripFunc(func(*http.Request) (*http.Response, error) {
		uploadCalls++
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(
				`{"blob":{"ref":{"$link":"bafkimage"},"mimeType":"image/jpeg","size":3}}`,
			)),
		}, nil
	})}
	client, err := NewIndigoPDSClient(jsonAPI, uploadAPI, nil)
	if err != nil {
		t.Fatalf("NewIndigoPDSClient: %v", err)
	}
	if _, err := client.UploadBlob(context.Background(), "image/jpeg", []byte("img")); err != nil {
		t.Fatalf("UploadBlob: %v", err)
	}
	if uploadCalls != 1 || jsonCalls != 0 {
		t.Fatalf("upload/json calls = %d/%d, want 1/0", uploadCalls, jsonCalls)
	}
}

// translateGetRecordError is the error-translation helper the indigo
// adapter uses; the test exercises each branch without making a real
// HTTP call.

func TestTranslateGetRecordError_RecordNotFoundByName(t *testing.T) {
	// Real-world case observed in the OAuth callback: HTTP 400 with a
	// typed `RecordNotFound` error body. atproto PDSes do NOT use HTTP
	// 404 for missing records — the XRPC `error` field is the signal.
	apiErr := &atclient.APIError{
		StatusCode: 400,
		Name:       "RecordNotFound",
		Message:    "Could not locate record: at://did:plc:x/social.craftsky.actor.profile/self",
	}
	if got := translateGetRecordError(apiErr); !errors.Is(got, ErrRecordNotFound) {
		t.Errorf("want ErrRecordNotFound; got %v", got)
	}
}

func TestIndigoPDSClientListRecordsReturnsTypedPaginatedBlocks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xrpc/com.atproto.repo.listRecords" || r.Method != http.MethodGet {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("repo") != "did:plc:alice" || query.Get("collection") != "app.bsky.graph.block" || query.Get("cursor") != "page-1" || query.Get("limit") != "100" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"cursor":"page-2",
			"records":[{
				"uri":"at://did:plc:alice/app.bsky.graph.block/one",
				"cid":"bafy-one",
				"value":{"$type":"app.bsky.graph.block","subject":"did:plc:bob","createdAt":"2026-07-19T12:00:00Z"}
			}]
		}`))
	}))
	defer srv.Close()

	client := newTestIndigoPDSClient(t, srv.URL)
	records, cursor, err := client.ListRecords(context.Background(), syntax.DID("did:plc:alice"), "app.bsky.graph.block", "page-1", 100)
	if err != nil {
		t.Fatalf("ListRecords: %v", err)
	}
	if cursor != "page-2" || len(records) != 1 || records[0].URI.String() != "at://did:plc:alice/app.bsky.graph.block/one" || records[0].CID != syntax.CID("bafy-one") {
		t.Fatalf("records/cursor = %+v/%q", records, cursor)
	}
	block, ok := records[0].Value.(*bsky.GraphBlock)
	if !ok || block.Subject != "did:plc:bob" {
		t.Fatalf("record value = %#v (%T), want typed block", records[0].Value, records[0].Value)
	}
}

func TestTranslateGetRecordError_Wrapped(t *testing.T) {
	// indigo may wrap APIError; errors.As must still find it.
	apiErr := &atclient.APIError{StatusCode: 400, Name: "RecordNotFound"}
	wrapped := fmt.Errorf("outer: %w", apiErr)
	if got := translateGetRecordError(wrapped); !errors.Is(got, ErrRecordNotFound) {
		t.Errorf("want ErrRecordNotFound through wrap; got %v", got)
	}
}

func TestTranslateGetRecordError_HTTP404Fallback(t *testing.T) {
	// Some upstreams (or future indigo changes) may return HTTP 404
	// without a body-level `RecordNotFound` name. Treat that as missing
	// too — it's the semantic a plain 404 conveys.
	apiErr := &atclient.APIError{StatusCode: 404}
	if got := translateGetRecordError(apiErr); !errors.Is(got, ErrRecordNotFound) {
		t.Errorf("want ErrRecordNotFound for plain 404; got %v", got)
	}
}

func TestTranslateGetRecordError_OtherErrorPassesThrough(t *testing.T) {
	apiErr := &atclient.APIError{StatusCode: 500, Name: "InternalError"}
	if got := translateGetRecordError(apiErr); errors.Is(got, ErrRecordNotFound) {
		t.Errorf("500 must not translate to ErrRecordNotFound; got %v", got)
	}
}

func TestTranslateGetRecordError_NonAPIErrorPassesThrough(t *testing.T) {
	boom := errors.New("network unreachable")
	if got := translateGetRecordError(boom); errors.Is(got, ErrRecordNotFound) {
		t.Errorf("non-APIError must not translate; got %v", got)
	}
}

func TestTranslateGetRecordError_NilIsNil(t *testing.T) {
	if got := translateGetRecordError(nil); got != nil {
		t.Errorf("nil in → nil out; got %v", got)
	}
}

func TestIndigoPDSClient_CreateRecord_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xrpc/com.atproto.repo.createRecord" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["repo"] != "did:plc:xyz" || body["collection"] != "social.craftsky.feed.post" {
			t.Fatalf("unexpected body: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"uri":"at://did:plc:xyz/social.craftsky.feed.post/3lf2abc","cid":"bafyabc"}`))
	}))
	defer srv.Close()
	cli := newTestIndigoPDSClient(t, srv.URL)

	uri, cid, err := cli.CreateRecord(
		context.Background(),
		syntax.DID("did:plc:xyz"),
		"social.craftsky.feed.post",
		map[string]any{"$type": "social.craftsky.feed.post", "text": "hi"},
	)
	if err != nil {
		t.Fatalf("CreateRecord: %v", err)
	}
	if string(uri) != "at://did:plc:xyz/social.craftsky.feed.post/3lf2abc" {
		t.Fatalf("uri = %q", uri)
	}
	if string(cid) != "bafyabc" {
		t.Fatalf("cid = %q", cid)
	}
}

func TestIndigoPDSClient_CreateRecord_EmptyURIErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"uri":"","cid":"bafyabc"}`))
	}))
	defer srv.Close()
	cli := newTestIndigoPDSClient(t, srv.URL)
	_, _, err := cli.CreateRecord(context.Background(),
		syntax.DID("did:plc:xyz"), "social.craftsky.feed.post", map[string]any{})
	if err == nil {
		t.Fatal("want error on empty uri, got nil")
	}
}

func TestIndigoPDSClient_CreateRecord_EmptyCIDErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"uri":"at://did:plc:xyz/social.craftsky.feed.post/3lf2abc","cid":""}`))
	}))
	defer srv.Close()
	cli := newTestIndigoPDSClient(t, srv.URL)
	_, _, err := cli.CreateRecord(context.Background(),
		syntax.DID("did:plc:xyz"), "social.craftsky.feed.post", map[string]any{})
	if err == nil {
		t.Fatal("want error on empty cid, got nil")
	}
}

func TestIndigoPDSClient_CreateRecord_AuthErrorExpiresSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"AuthenticationRequired","message":"invalid access token"}`))
	}))
	defer srv.Close()

	expired := false
	cli := newTestIndigoPDSClient(t, srv.URL, func(context.Context) {
		expired = true
	})
	_, _, err := cli.CreateRecord(context.Background(),
		syntax.DID("did:plc:xyz"), "social.craftsky.feed.post", map[string]any{})
	if !errors.Is(err, ErrPDSSessionExpired) {
		t.Fatalf("want ErrPDSSessionExpired, got %v", err)
	}
	if !expired {
		t.Fatal("OnSessionExpired was not called")
	}
}

func TestIndigoPDSClient_DeleteRecord_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xrpc/com.atproto.repo.deleteRecord" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["repo"] != "did:plc:xyz" || body["collection"] != "social.craftsky.feed.post" || body["rkey"] != "3lf2abc" {
			t.Fatalf("unexpected body: %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	cli := newTestIndigoPDSClient(t, srv.URL)
	if err := cli.DeleteRecord(context.Background(),
		syntax.DID("did:plc:xyz"), "social.craftsky.feed.post", "3lf2abc"); err != nil {
		t.Fatalf("DeleteRecord: %v", err)
	}
}

func TestIndigoPDSClient_DeleteRecord_NotFound_TranslatesToErrRecordNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"RecordNotFound","message":"no such record"}`))
	}))
	defer srv.Close()
	cli := newTestIndigoPDSClient(t, srv.URL)
	err := cli.DeleteRecord(context.Background(),
		syntax.DID("did:plc:xyz"), "social.craftsky.feed.post", "3lf2abc")
	if !errors.Is(err, ErrRecordNotFound) {
		t.Fatalf("want ErrRecordNotFound, got %v", err)
	}
}

func TestIndigoPDSClientDeleteRecordWithSwapSendsExpectedCID(t *testing.T) {
	client := newTestIndigoPDSClientWithTransport(t, pdsRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["repo"] != "did:plc:xyz" ||
			body["collection"] != "social.craftsky.feed.post" ||
			body["rkey"] != "3lf2abc" || body["swapRecord"] != "bafy-expected-record" {
			t.Fatalf("conditional delete body = %+v", body)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}, nil
	}))
	if err := client.DeleteRecordWithSwap(
		context.Background(),
		syntax.DID("did:plc:xyz"),
		"social.craftsky.feed.post",
		"3lf2abc",
		syntax.CID("bafy-expected-record"),
	); err != nil {
		t.Fatal(err)
	}
}

func TestIndigoPDSClientDeleteRecordWithSwapTranslatesInvalidSwap(t *testing.T) {
	client := newTestIndigoPDSClientWithTransport(t, pdsRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(
				`{"error":"InvalidSwap","message":"record changed"}`,
			)),
		}, nil
	}))
	err := client.DeleteRecordWithSwap(
		context.Background(),
		syntax.DID("did:plc:xyz"),
		"social.craftsky.feed.post",
		"3lf2abc",
		syntax.CID("bafy-stale-record"),
	)
	if !errors.Is(err, ErrRecordSwapConflict) {
		t.Fatalf("conditional delete error = %v, want ErrRecordSwapConflict", err)
	}
}

func TestIndigoPDSClientPutRecordWithSwapSendsExpectedCID(t *testing.T) {
	client := newTestIndigoPDSClientWithTransport(t, pdsRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["repo"] != "did:plc:xyz" ||
			body["collection"] != "social.craftsky.actor.profile" ||
			body["rkey"] != "self" || body["swapRecord"] != "bafy-prior-profile" {
			t.Fatalf("conditional Put body = %+v", body)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}, nil
	}))
	if err := client.PutRecordWithSwap(
		context.Background(),
		syntax.DID("did:plc:xyz"),
		"social.craftsky.actor.profile",
		"self",
		map[string]any{"displayName": "updated"},
		syntax.CID("bafy-prior-profile"),
	); err != nil {
		t.Fatal(err)
	}
}

func TestIndigoPDSClientPutRecordWithSwapSendsNullForAbsentRecord(t *testing.T) {
	client := newTestIndigoPDSClientWithTransport(t, pdsRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		value, present := body["swapRecord"]
		if !present || value != nil {
			t.Fatalf("create-only Put swapRecord = %#v, present = %v", value, present)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{}`)),
		}, nil
	}))
	if err := client.PutRecordWithSwap(
		context.Background(),
		syntax.DID("did:plc:xyz"),
		"social.craftsky.business.profile",
		"self",
		map[string]any{"$type": "social.craftsky.business.profile"},
		syntax.CID("*"),
	); err != nil {
		t.Fatal(err)
	}
}

func TestIndigoPDSClientPutRecordWithSwapTranslatesInvalidSwap(t *testing.T) {
	client := newTestIndigoPDSClientWithTransport(t, pdsRoundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(
				`{"error":"InvalidSwap","message":"record changed"}`,
			)),
		}, nil
	}))
	err := client.PutRecordWithSwap(
		context.Background(),
		syntax.DID("did:plc:xyz"),
		"social.craftsky.actor.profile",
		"self",
		map[string]any{"displayName": "stale"},
		syntax.CID("bafy-stale-profile"),
	)
	if !errors.Is(err, ErrRecordSwapConflict) {
		t.Fatalf("conditional Put error = %v, want ErrRecordSwapConflict", err)
	}
}

func newTestIndigoPDSClientWithTransport(
	t *testing.T,
	transport http.RoundTripper,
) *IndigoPDSClient {
	t.Helper()
	newAPIClient := func() *atclient.APIClient {
		client := atclient.NewAPIClient("https://pds.example")
		client.Client = &http.Client{Transport: transport}
		return client
	}
	client, err := NewIndigoPDSClient(newAPIClient(), newAPIClient(), nil)
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestIndigoPDSClient_UploadBlob_HappyPath(t *testing.T) {
	t.Parallel()
	wantBody := []byte("fake-jpeg-bytes")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/xrpc/com.atproto.repo.uploadBlob" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "image/jpeg" {
			t.Fatalf("content-type = %q, want image/jpeg", got)
		}
		gotBody, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if !bytes.Equal(gotBody, wantBody) {
			t.Fatalf("body = %q, want %q", string(gotBody), string(wantBody))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"blob":{"$type":"blob","ref":{"$link":"bafkimage"},"mimeType":"image/jpeg","size":253496}}`))
	}))
	defer srv.Close()

	cli := newTestIndigoPDSClient(t, srv.URL)
	blob, err := cli.UploadBlob(context.Background(), "image/jpeg", wantBody)
	if err != nil {
		t.Fatalf("UploadBlob: %v", err)
	}
	if blob.CID != "bafkimage" || blob.MIME != "image/jpeg" || blob.Size != 253496 {
		t.Fatalf("blob metadata = %+v", blob)
	}
	if blob.Raw == nil {
		t.Fatal("blob.Raw is nil")
	}
	if blobRef, ok := blob.Raw["ref"].(map[string]any); !ok || blobRef["$link"] != "bafkimage" {
		t.Fatalf("raw ref = %+v", blob.Raw["ref"])
	}
}

func TestIndigoPDSClient_UploadBlob_EmptyBlobErrors(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"blob":null}`))
	}))
	defer srv.Close()

	cli := newTestIndigoPDSClient(t, srv.URL)
	_, err := cli.UploadBlob(context.Background(), "image/jpeg", []byte("img"))
	if err == nil {
		t.Fatal("want error on null blob, got nil")
	}
}
