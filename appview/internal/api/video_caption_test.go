package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/api"
)

type captionPostReader struct{ row *api.PostRow }

func (reader captionPostReader) ReadOne(context.Context, string, string) (*api.PostRow, error) {
	if reader.row == nil {
		return nil, api.ErrPostNotFound
	}
	return reader.row, nil
}

type captionBlobFetcher struct {
	calls int
	body  []byte
	err   error
}

func (fetcher *captionBlobFetcher) Fetch(_ context.Context, _ syntax.DID, _ syntax.CID) ([]byte, error) {
	fetcher.calls++
	return fetcher.body, fetcher.err
}

func TestVideoCaptionHandlerRequiresIndexedMembershipAndReturnsOnlyWebVTT(t *testing.T) {
	t.Parallel()
	const memberCID = "bafkreigxxxkul4e5rjz4fomqgn6ieeoxbcqeztmxjbrhnbpe7r44ya4ahe"
	row := baseRow()
	row.RawEmbed = json.RawMessage(`{"$type":"app.bsky.embed.video","video":{"ref":{"$link":"bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"},"mimeType":"video/mp4","size":1},"captions":[{"lang":"en","file":{"ref":{"$link":"` + memberCID + `"},"mimeType":"text/vtt","size":20}}]}`)
	fetcher := &captionBlobFetcher{body: []byte("WEBVTT\n\n00:00.000 --> 00:01.000\nHello\n")}
	handler := api.VideoCaptionHandler(captionPostReader{row: row}, fetcher, nil)

	request := httptest.NewRequest(http.MethodGet, "/v1/posts/did:plc:alice/rk1/video-captions/"+memberCID, nil)
	request.SetPathValue("did", "did:plc:alice")
	request.SetPathValue("rkey", "rk1")
	request.SetPathValue("captionCid", memberCID)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Header().Get("Content-Type") != "text/vtt; charset=utf-8" || recorder.Body.String() != string(fetcher.body) {
		t.Fatalf("response = %d %q %q", recorder.Code, recorder.Header().Get("Content-Type"), recorder.Body.String())
	}

	foreign := httptest.NewRequest(http.MethodGet, "/v1/posts/did:plc:alice/rk1/video-captions/bafkreifqaaa7v7zgm4myr6tv22zowykwi5b3m5i4m5w4a2j4aa2qgq7lpa", nil)
	foreign.SetPathValue("did", "did:plc:alice")
	foreign.SetPathValue("rkey", "rk1")
	foreign.SetPathValue("captionCid", "bafkreifqaaa7v7zgm4myr6tv22zowykwi5b3m5i4m5w4a2j4aa2qgq7lpa")
	foreignRecorder := httptest.NewRecorder()
	handler.ServeHTTP(foreignRecorder, foreign)
	if foreignRecorder.Code != http.StatusNotFound || fetcher.calls != 1 {
		t.Fatalf("foreign response = %d, fetch calls = %d", foreignRecorder.Code, fetcher.calls)
	}
}

func TestVideoCaptionHandlerFailsClosedOnInvalidFetchedBody(t *testing.T) {
	t.Parallel()
	const memberCID = "bafkreigxxxkul4e5rjz4fomqgn6ieeoxbcqeztmxjbrhnbpe7r44ya4ahe"
	row := baseRow()
	row.RawEmbed = json.RawMessage(`{"$type":"app.bsky.embed.video","video":{"ref":{"$link":"bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"},"mimeType":"video/mp4","size":1},"captions":[{"lang":"en","file":{"ref":{"$link":"` + memberCID + `"},"mimeType":"text/vtt","size":20}}]}`)
	for _, fetcher := range []*captionBlobFetcher{{body: []byte("not captions")}, {err: errors.New("private PDS URL")}} {
		handler := api.VideoCaptionHandler(captionPostReader{row: row}, fetcher, nil)
		request := httptest.NewRequest(http.MethodGet, "/caption", nil)
		request.SetPathValue("did", "did:plc:alice")
		request.SetPathValue("rkey", "rk1")
		request.SetPathValue("captionCid", memberCID)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadGateway || recorder.Body.String() == "" {
			t.Fatalf("response = %d %q", recorder.Code, recorder.Body.String())
		}
	}
}
