package app

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	appbsky "github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"github.com/ipfs/go-cid"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/notifications"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdseffects"
	"social.craftsky/appview/internal/tap"
	"social.craftsky/appview/internal/testdb"
	"social.craftsky/appview/internal/video"
)

const videoPostFlowDDL = indexerWiringDDL + `
ALTER TABLE craftsky_posts
    ADD COLUMN langs TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN is_project BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN project_craft_type TEXT,
    ADD COLUMN external_import_source TEXT,
    ADD COLUMN profile_sort_at TIMESTAMPTZ NOT NULL DEFAULT now();
CREATE TABLE craftsky_project_posts (
    uri TEXT PRIMARY KEY REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    raw_project JSONB NOT NULL
);
CREATE TABLE moderation_outputs (
    id TEXT PRIMARY KEY,
    source_did TEXT NOT NULL,
    subject_type TEXT NOT NULL,
    subject_did TEXT NOT NULL,
    subject_uri TEXT,
    value TEXT NOT NULL,
    action TEXT NOT NULL,
    expires_at TIMESTAMPTZ,
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE profile_pins (
    owner_did TEXT NOT NULL,
    slot TEXT NOT NULL,
    post_uri TEXT NOT NULL,
    state_token UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (owner_did, slot)
);
CREATE TABLE craftsky_post_mentions (
    post_uri TEXT NOT NULL REFERENCES craftsky_posts(uri) ON DELETE CASCADE,
    mentioned_did TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    indexed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (post_uri, mentioned_did)
);
`

type videoPostFlowEffects struct {
	record any
	uri    syntax.ATURI
	cid    syntax.CID
}

func (effects *videoPostFlowEffects) ResolveExpectedOwners(_ context.Context, generation int64, _ []syntax.DID) ([]ownerlifecycle.ExpectedOwner, error) {
	return []ownerlifecycle.ExpectedOwner{{Owner: "did:plc:alice", Generation: generation}}, nil
}

func (*videoPostFlowEffects) ReadRecord(context.Context, pdseffects.ReadRecordRequest, any) (syntax.CID, error) {
	panic("unexpected ReadRecord call")
}

func (effects *videoPostFlowEffects) PutRecord(_ context.Context, request pdseffects.PutRecordRequest) (pdseffects.RecordResult, error) {
	effects.record = request.Record
	effects.uri = syntax.ATURI("at://" + request.Owner.String() + "/" + request.Collection.String() + "/" + request.Rkey.String())
	effects.cid = "bafy-post-cid"
	return pdseffects.RecordResult{URI: effects.uri, CID: effects.cid}, nil
}

func (*videoPostFlowEffects) DeleteRecord(context.Context, pdseffects.DeleteRecordRequest) (pdseffects.RecordResult, error) {
	panic("unexpected DeleteRecord call")
}

func (*videoPostFlowEffects) UploadBlob(context.Context, pdseffects.UploadBlobRequest) (*auth.UploadedBlob, error) {
	panic("unexpected UploadBlob call")
}

type videoPostFlowResolver struct{}

func (videoPostFlowResolver) ResolveHandle(context.Context, syntax.DID) (syntax.Handle, error) {
	return "alice.example", nil
}

func (videoPostFlowResolver) ResolveDID(context.Context, syntax.Handle) (syntax.DID, error) {
	return "did:plc:alice", nil
}

type videoPostFlowJobStatus struct {
	status *appbsky.VideoDefs_JobStatus
}

func (status videoPostFlowJobStatus) GetJobStatus(context.Context, string) (*appbsky.VideoDefs_JobStatus, error) {
	return status.status, nil
}

func TestVideoPostCreateTapReadConvergence(t *testing.T) {
	const (
		owner     syntax.DID = "did:plc:alice"
		videoCID  syntax.CID = "bafkreie3w2xq7u6rs5szu6vllsq5xh7y7uv3f6blql6uz4ep6txv6m4o6a"
		videoSize            = int64(123)
	)
	pool := testdb.WithSchema(t, videoPostFlowDDL)
	if _, err := pool.Exec(context.Background(), `INSERT INTO craftsky_profiles (did, record_cid) VALUES ($1, 'profile-cid')`, owner); err != nil {
		t.Fatalf("seed Craftsky profile: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO bluesky_profiles (did, display_name, record_cid) VALUES ($1, 'Alice', 'bluesky-profile-cid')`, owner); err != nil {
		t.Fatalf("seed Bluesky profile: %v", err)
	}

	playback, err := video.NewPlaybackURLBuilder(video.PlaybackConfig{
		PlaylistTemplate:  "https://media.example/watch/{did}/{cid}/playlist.m3u8",
		ThumbnailTemplate: "https://media.example/watch/{did}/{cid}/thumbnail.jpg",
	})
	if err != nil {
		t.Fatalf("create playback builder: %v", err)
	}
	store := api.NewPostStoreWithPlayback(pool, nil, playback)
	effects := &videoPostFlowEffects{}
	parsedVideoCID := cid.MustParse(videoCID.String())
	verifier := video.NewCompletionVerifier(videoPostFlowJobStatus{status: &appbsky.VideoDefs_JobStatus{
		JobId: "job-1", Did: owner.String(), State: "JOB_STATE_COMPLETED",
		Blob: &lexutil.LexBlob{Ref: lexutil.LexLink(parsedVideoCID), MimeType: "video/mp4", Size: videoSize},
	}})
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	create := api.CreatePostHandler(
		store,
		func(context.Context, syntax.DID, string) (pdseffects.EffectExecutor, error) { return effects, nil },
		videoPostFlowResolver{},
		api.DefaultMediaLimits(),
		logger,
		api.CreatePostHandlerOptions{VideoCompletionVerifier: verifier},
	)
	body := `{"text":"video post","langs":["en"],"embed":{"video":{"jobId":"job-1","blob":{"$type":"blob","ref":{"$link":"` + videoCID.String() + `"},"mimeType":"video/mp4","size":123},"alt":"Hands knitting","aspectRatio":{"width":16,"height":9}}}}`
	request := httptest.NewRequest(http.MethodPost, "/v1/posts", strings.NewReader(body))
	request = request.WithContext(middleware.WithOwnerGeneration(middleware.WithDID(request.Context(), owner), 1))
	recorder := httptest.NewRecorder()
	create.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var created api.PostResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	record, err := json.Marshal(effects.record)
	if err != nil {
		t.Fatalf("marshal PDS record: %v", err)
	}
	if strings.Contains(string(record), "job-1") {
		t.Fatalf("PDS record contains private job proof: %s", record)
	}
	var persisted struct {
		Embed struct {
			Type  string `json:"$type"`
			Video struct {
				Ref struct {
					Link string `json:"$link"`
				} `json:"ref"`
				MIMEType string `json:"mimeType"`
				Size     int64  `json:"size"`
			} `json:"video"`
		} `json:"embed"`
	}
	if err := json.Unmarshal(record, &persisted); err != nil {
		t.Fatalf("decode PDS record: %v", err)
	}
	if persisted.Embed.Type != "app.bsky.embed.video" || persisted.Embed.Video.Ref.Link != videoCID.String() || persisted.Embed.Video.MIMEType != "video/mp4" || persisted.Embed.Video.Size != videoSize {
		t.Fatalf("PDS video embed = %+v", persisted.Embed)
	}

	dispatcher := newTransactionalIndexerDispatcherWithActorDeletion(
		pool, logger, nil,
		notifications.NoopLifecycle{}, notifications.NewActorDeletionService(pool),
	)
	projectIndexerWiringEvent(t, pool, dispatcher, tap.Event{
		URI: effects.uri, CID: effects.cid, DID: owner,
		Collection: "social.craftsky.feed.post", Rkey: effects.uri.RecordKey(),
		Action: "create", Rev: "3aaaaaaaaaaac", Record: record,
	})

	indexed, err := store.ReadOne(context.Background(), owner.String(), effects.uri.RecordKey().String())
	if err != nil {
		t.Fatalf("read indexed post: %v", err)
	}
	eventual := api.BuildPostResponse(indexed, "alice.example", playback)
	if created.Video == nil || eventual.Video == nil {
		t.Fatalf("video missing from create or eventual response: create=%+v eventual=%+v", created.Video, eventual.Video)
	}
	if !reflect.DeepEqual(eventual.Video, created.Video) {
		t.Fatalf("eventual video = %+v, create video = %+v", eventual.Video, created.Video)
	}
	if eventual.Video.CID != videoCID.String() || eventual.Video.Playlist != "https://media.example/watch/did:plc:alice/"+videoCID.String()+"/playlist.m3u8" {
		t.Fatalf("eventual normalized video = %+v", eventual.Video)
	}
	var indexedCount int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM craftsky_posts WHERE uri = $1`, effects.uri).Scan(&indexedCount); err != nil {
		t.Fatalf("count indexed posts: %v", err)
	}
	if indexedCount != 1 {
		t.Fatalf("indexed post count = %d, want 1", indexedCount)
	}
}
