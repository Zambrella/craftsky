package video

import (
	"context"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"io"
	"net/http"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/pdseffects"
	"strings"
	"testing"
	"time"
)

type blockedVideoEffects struct{ started chan struct{} }

func (f *blockedVideoEffects) UploadBlob(ctx context.Context, _ pdseffects.UploadBlobRequest) (*auth.UploadedBlob, error) {
	close(f.started)
	<-ctx.Done()
	return nil, &pdseffects.OutcomeAmbiguousError{Cause: ctx.Err()}
}
func TestBunnyCancelStopsFinalisationAndDeletesOnlyVendorAsset(t *testing.T) {
	b, w, id, _, _ := cleanupFixture(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:bunny-cleanup-owner")
	if _, err := b.pool.Exec(ctx, "UPDATE bunny_video_jobs SET state='created',blob_cid=NULL,blob_size=NULL WHERE id=$1", id); err != nil {
		t.Fatal(err)
	}
	effects := &blockedVideoEffects{started: make(chan struct{})}
	b.effects = func(context.Context, syntax.DID, string) (BunnyBlobEffects, error) { return effects, nil }
	deleted, purged := false, false
	b.client.Transport = bunnyRoundTrip(func(r *http.Request) (*http.Response, error) {
		body := `{"guid":"aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa","videoLibraryId":123,"status":4,"length":10,"hasMP4Fallback":true,"availableResolutions":"720p"}`
		if r.Method == "DELETE" {
			deleted = true
			body = `{}`
		} else if r.URL.Host == "api.bunny.net" {
			purged = true
			body = `{}`
		} else if r.URL.Host == "test.b-cdn.net" {
			body = "\x00\x00\x00\x18ftypisomsynthetic-video"
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), ContentLength: int64(len(body))}, nil
	})
	done := make(chan struct{})
	go func() { defer close(done); _, _ = b.Status(ctx, owner, "session", id) }()
	select {
	case <-effects.started:
	case <-time.After(3 * time.Second):
		t.Fatal("upload did not start")
	}
	if err := b.Cancel(ctx, "did:plc:foreign", id); !errors.Is(err, ErrVideoJobNotFound) {
		t.Fatal("foreign cancellation accepted", err)
	}
	if err := b.Cancel(ctx, owner, id); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("PDS upload did not stop")
	}
	if err := b.Cancel(ctx, owner, id); err != nil {
		t.Fatal("repeat cancellation failed", err)
	}
	status, err := b.Status(ctx, owner, "session", id)
	if err != nil || status.State != "JOB_STATE_FAILED" || status.Error != "video_upload_canceled" {
		t.Fatal("canceled job resumed", status, err)
	}
	for range 3 {
		cleanupStep(t, w)
	}
	if !deleted || !purged || cleanupState(t, b, id) != "deleted" {
		t.Fatal("vendor cancellation was not cleaned up")
	}
}
func TestBunnyCancelProtectsPublishedAndReservedVideo(t *testing.T) {
	b, _, id, add, _ := cleanupFixture(t)
	owner := syntax.DID("did:plc:bunny-cleanup-owner")
	if _, err := b.pool.Exec(context.Background(), "UPDATE bunny_video_jobs SET lease_until=now()+interval '2 minutes' WHERE id=$1", id); err != nil {
		t.Fatal(err)
	}
	if err := b.Cancel(context.Background(), owner, id); !errors.Is(err, ErrVideoCancellationTooLate) {
		t.Fatal("reserved video canceled", err)
	}
	if _, err := b.pool.Exec(context.Background(), "UPDATE bunny_video_jobs SET lease_until='-infinity' WHERE id=$1", id); err != nil {
		t.Fatal(err)
	}
	add("published")
	if err := b.Cancel(context.Background(), owner, id); !errors.Is(err, ErrVideoCancellationTooLate) {
		t.Fatal("published video canceled", err)
	}
	if cleanupState(t, b, id) != "ready" {
		t.Fatal("published video changed")
	}
}
