package video

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/testdb"
)

type bunnyRoundTrip func(*http.Request) (*http.Response, error)

func (f bunnyRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type bunnyEffects struct {
	calls int
	body  []byte
}

func (f *bunnyEffects) UploadBlob(ctx context.Context, r pdseffects.UploadBlobRequest) (*auth.UploadedBlob, error) {
	f.calls++
	if r.Body == nil || len(r.Bytes) != 0 {
		return nil, errors.New("video must be streamed")
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	f.body = body
	cid, _, err := pdseffects.PredictBlobCID(body)
	if err != nil {
		return nil, err
	}
	return &auth.UploadedBlob{CID: cid.String(), MIME: r.MIME, Size: int64(len(body)), Raw: map[string]any{"$type": "blob"}}, nil
}
func TestBunnyPublicationOwnershipDurabilityAndPlayback(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	owner := syntax.DID("did:plc:bunny-owner")
	_, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at) VALUES($1,'active',1,1,'test',now())`, owner)
	if err != nil {
		t.Fatal(err)
	}
	effects := &bunnyEffects{}
	b, err := NewBunny(pool, "123", "private-key-canary", "test.b-cdn.net", func(context.Context, syntax.DID, string) (BunnyBlobEffects, error) { return effects, nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	const video = "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa"
	const mp4 = "\x00\x00\x00\x18ftypisomsynthetic-video"
	const collection = "bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb"
	var title, metadataJob, metadataCID string
	b.client.Transport = bunnyRoundTrip(func(r *http.Request) (*http.Response, error) {
		data := ""
		switch {
		case r.URL.Host == "test.b-cdn.net":
			if r.Header.Get("AccessKey") != "" {
				t.Fatal("key forwarded to CDN")
			}
			data = mp4
		case strings.HasSuffix(r.URL.Path, "/collections"):
			if r.URL.Query().Get("search") != owner.String() {
				t.Fatal("collection lookup must use the owner DID")
			}
			data = `{"totalItems":1,"items":[{"guid":"` + collection + `","name":"` + owner.String() + `"}]}`
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/videos"):
			if r.Header.Get("AccessKey") != "private-key-canary" {
				t.Fatal("missing management key")
			}
			var input map[string]string
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input["collectionId"] != collection {
				t.Fatal("video must be created in its owner's collection", err)
			}
			title = input["title"]
			data = `{"guid":"` + video + `"}`
		case r.Method == "POST":
			var input struct {
				Tags []struct{ Property, Value string } `json:"metaTags"`
			}
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
				t.Fatal(err)
			}
			for _, tag := range input.Tags {
				switch tag.Property {
				case "craftsky:uploadJobId":
					metadataJob = tag.Value
				case "craftsky:source":
					if tag.Value != "upload" {
						t.Fatal("wrong upload source")
					}
				case "craftsky:pdsBlobCid":
					metadataCID = tag.Value
				default:
					t.Fatal("unexpected metadata field")
				}
			}
			data = `{}`
		default:
			data = `{"guid":"` + video + `","videoLibraryId":123,"status":4,"length":3,"encodeProgress":100,"hasMP4Fallback":true,"availableResolutions":"360p,720p"}`
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(data)), ContentLength: int64(len(data))}, nil
	})
	authorization, err := b.IssueUpload(ctx, owner, "session")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(title, " UTC · "+authorization.JobID[:8]) || authorization.Title != title || authorization.CollectionID != collection || metadataJob != authorization.JobID || metadataCID != "" {
		t.Fatal("missing unique title or initial technical metadata")
	}
	expected := sha256.Sum256([]byte("123private-key-canary" + strconv.FormatInt(authorization.ExpiresAt.Unix(), 10) + video))
	if authorization.Provider != "bunny" || authorization.Token != hex.EncodeToString(expected[:]) {
		t.Fatal("incorrect upload signature")
	}
	if _, err = b.Status(ctx, "did:plc:foreign", "foreign-session", authorization.JobID); !errors.Is(err, ErrVideoJobNotFound) {
		t.Fatal("foreign owner could access job", err)
	}
	status, err := b.Status(ctx, owner, "session", authorization.JobID)
	if err != nil {
		t.Fatal(err)
	}
	if status.State != "JOB_STATE_COMPLETED" || effects.calls != 1 || string(effects.body) != mp4 {
		t.Fatalf("incomplete handoff: %+v, calls %d", status, effects.calls)
	}
	// A restarted service can recover the exact proof without another PDS upload.
	again, err := b.Status(ctx, owner, "session", authorization.JobID)
	if err != nil || again.State != "JOB_STATE_COMPLETED" || effects.calls != 1 {
		t.Fatal("completion was not durable", err)
	}
	contentCID, _, _ := pdseffects.PredictBlobCID([]byte(mp4))
	if metadataCID != contentCID.String() {
		t.Fatal("PDS blob CID not attached after handoff")
	}
	proof := Blob{CID: contentCID, MIMEType: "video/mp4", Size: int64(len(mp4))}
	if _, err = b.Verify(ctx, owner, authorization.JobID, proof); err != nil {
		t.Fatal(err)
	}
	proof.Size++
	if _, err = b.Verify(ctx, owner, authorization.JobID, proof); err == nil {
		t.Fatal("forged proof accepted")
	}
	playlist, poster := b.URLs(owner, contentCID)
	if playlist != "https://test.b-cdn.net/"+video+"/playlist.m3u8" || poster != "https://test.b-cdn.net/"+video+"/thumbnail.jpg" {
		t.Fatal("incorrect playback mapping")
	}
	_, err = pool.Exec(ctx, "UPDATE bunny_video_jobs SET created_at=now()-interval '8 days' WHERE id=$1", authorization.JobID)
	if err != nil {
		t.Fatal(err)
	}
	count, err := b.Cleanup(ctx, false)
	if err != nil || count != 1 {
		t.Fatal("cleanup preview", count, err)
	}
	_, err = pool.Exec(ctx, `INSERT INTO craftsky_posts(uri,did,rkey,cid,text,record,created_at)
 VALUES($1,$2,'testvideo','postcid','video',jsonb_build_object('embed',jsonb_build_object('video',jsonb_build_object('ref',jsonb_build_object('$link',$3::text)))) ,now())`, "at://"+owner.String()+"/social.craftsky.feed.post/testvideo", owner, contentCID.String())
	if err != nil {
		t.Fatal(err)
	}
	count, err = b.Cleanup(ctx, true)
	if err != nil || count != 0 {
		t.Fatal("referenced video must be retained", count, err)
	}
	_, err = pool.Exec(ctx, "DELETE FROM craftsky_posts WHERE did=$1", owner)
	if err != nil {
		t.Fatal(err)
	}
	count, err = b.Cleanup(ctx, true)
	if err != nil || count != 1 {
		t.Fatal("unreferenced cleanup", count, err)
	}
	_, err = pool.Exec(ctx, "UPDATE owner_lifecycles SET generation=2,state='departed',transition_reason='test departure',transitioned_at=now(),updated_at=now() WHERE owner_did=$1", owner)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = b.Status(ctx, owner, "session", authorization.JobID); !errors.Is(err, ErrVideoJobNotFound) {
		t.Fatal("stale owner generation accepted", err)
	}
}

func TestBunnyDownloadRejectsRedirectOversizeAndInvalidMP4(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		size   int64
		body   string
	}{{"redirect", 302, 0, ""}, {"oversize", 200, 300000001, ""}, {"invalid", 200, 16, "not-an-mp4-file!"}} {
		t.Run(test.name, func(t *testing.T) {
			b := &Bunny{cdn: "test.b-cdn.net", client: &http.Client{Timeout: time.Second, Transport: bunnyRoundTrip(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: test.status, ContentLength: test.size, Body: io.NopCloser(strings.NewReader(test.body))}, nil
			})}}
			if file, err := b.download(context.Background(), "video", "720p"); err == nil {
				file.Close()
				t.Fatal("unsafe media accepted")
			}
		})
	}
}

type rejectedBunnySession struct{ err error }

func (r rejectedBunnySession) WithActiveSession(context.Context, syntax.DID, string, auth.OAuthSessionOperation) error {
	return r.err
}
func TestBunnyRejectsExpiredPDSSessionBeforeAllocating(t *testing.T) {
	for _, cause := range []error{auth.ErrPDSSessionExpired, auth.ErrPDSPermissionsRequired} {
		b := &Bunny{sessions: rejectedBunnySession{err: cause}}
		if _, err := b.IssueUpload(context.Background(), "did:plc:owner", "session"); !errors.Is(err, cause) {
			t.Fatal("lost PDS authentication error", err)
		}
	}
}
