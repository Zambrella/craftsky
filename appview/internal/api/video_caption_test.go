package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
		var local bytes.Buffer
		handler := api.VideoCaptionHandler(captionPostReader{row: row}, fetcher, slog.New(slog.NewJSONHandler(&local, nil)))
		request := httptest.NewRequest(http.MethodGet, "/caption", nil)
		request.SetPathValue("did", "did:plc:alice")
		request.SetPathValue("rkey", "rk1")
		request.SetPathValue("captionCid", memberCID)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if !strings.Contains(local.String(), "video.caption.fetch") || !strings.Contains(local.String(), "causes") {
			t.Fatalf("caption cause/stage missing: %s", local.String())
		}
		if strings.Contains(local.String(), "private PDS URL") || strings.Contains(local.String(), "not captions") {
			t.Fatal("upstream text leaked")
		}
		if recorder.Code != http.StatusBadGateway || recorder.Body.String() == "" {
			t.Fatalf("response = %d %q", recorder.Code, recorder.Body.String())
		}
	}
}

// AT-005 / IT-011: caption blobs are readable only through an eligible post.
func TestPublicVideoCaptionChecksGuestEligibility(t *testing.T) {
	pool, _ := publicReadFixture(t)
	ctx := context.Background()
	const captionCID = "bafkreigxxxkul4e5rjz4fomqgn6ieeoxbcqeztmxjbrhnbpe7r44ya4ahe"
	for _, owner := range []string{"public", "hidden", "unknown"} {
		uri := seedSearchProject(t, pool, "did:plc:"+owner, "caption", "caption", "knitting", "video", time.Now().UTC())
		embed := `{"$type":"app.bsky.embed.video","captions":[{"lang":"en","file":{"ref":{"$link":"` + captionCID + `"},"mimeType":"text/vtt","size":20}}]}`
		if _, err := pool.Exec(ctx, `UPDATE craftsky_posts SET record=jsonb_set(record,'{embed}',$2::jsonb) WHERE uri=$1`, uri, embed); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) SELECT uri,'post',cid,'clear' FROM craftsky_posts WHERE uri=$1`, uri); err != nil {
			t.Fatal(err)
		}
	}
	fetcher := &captionBlobFetcher{body: []byte("WEBVTT\n\n00:00.000 --> 00:01.000\nPublic caption\n")}
	handler := api.PublicVideoCaptionHandler(api.NewPostStore(pool), fetcher, nil)
	for _, owner := range []string{"public", "hidden", "unknown"} {
		r := httptest.NewRequest("GET", "/v1/public/posts/did:plc:"+owner+"/caption/video-captions/"+captionCID, nil)
		r.SetPathValue("did", "did:plc:"+owner)
		r.SetPathValue("rkey", "caption")
		r.SetPathValue("captionCid", captionCID)
		r.Header.Set("Authorization", "Bearer spoof")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		want := 404
		if owner == "public" {
			want = 200
		}
		if w.Code != want || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("owner=%s status=%d body=%s cache=%s", owner, w.Code, w.Body, w.Header().Get("Cache-Control"))
		}
	}
	if fetcher.calls != 1 {
		t.Fatalf("fetched ineligible blob: %d", fetcher.calls)
	}
}
