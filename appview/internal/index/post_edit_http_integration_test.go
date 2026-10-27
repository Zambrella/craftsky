package index_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atdata"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/ownerlifecycle"
	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/recordjson"
)

// IT-001 / IT-004 / IT-007: production HTTP handlers, durable commands and
// source projection converge over a real socket with a conditional PDS fixture.
// Authentication context is supplied by the fixture; policy gates have their
// separate routed tests. This does not contact an external PDS.
func TestPostEditHTTPCommandAndProjectionConverge(t *testing.T) {
	f := newNotificationPipeline(t)
	ctx := context.Background()
	event := notificationEvent(901, "did:plc:actor", "3aaaaaaaaaaa5", `,"futureValue":9007199254740993`)
	event.CID = httpEditRecordCID(t, event.Record)
	f.ingest(event)
	f.drain()
	pds := &httpEditPDS{t: t, uri: event.URI, record: event.Record, recordCID: event.CID,
		commit: auth.RepositoryCommit{CID: httpEditRecordCID(t, json.RawMessage(`{"head":1}`)), Revision: event.Rev}}
	store, err := pdscommands.NewStore(pdscommands.StoreConfig{Pool: f.pool, Now: time.Now, NewUUID: uuid.New})
	if err != nil {
		t.Fatal(err)
	}
	fence, err := ownerlifecycle.NewFencer(f.pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	lifecycle, err := ownerlifecycle.NewStore(f.pool, fence, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	commands, err := pdscommands.NewAddressedCommandService(pdscommands.AddressedCommandServiceConfig{
		Store: store, Lifecycles: lifecycle, Now: time.Now, Sleep: func(time.Duration) {},
		NewBoundary: func(context.Context, syntax.DID, string) (auth.ActiveEffectPDSBoundary, error) { return pds, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	posts := api.NewPostStore(f.pool)
	mux := http.NewServeMux()
	mux.Handle("GET /v1/posts/{did}/{rkey}/edit", api.GetPostEditHandler(posts, postEditProjectionResolver{}))
	mux.Handle("PATCH /v1/posts/{did}/{rkey}", api.CommandEditPostHandler(posts, commands, postEditProjectionResolver{}, api.DefaultMediaLimits()))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := middleware.WithDID(r.Context(), event.DID)
		ctx = middleware.WithOwnerGeneration(ctx, 1)
		ctx = middleware.WithOAuthSessionID(ctx, "http-edit-session")
		mux.ServeHTTP(w, r.WithContext(ctx))
	}))
	defer server.Close()
	path := "/v1/posts/" + event.DID.String() + "/" + event.Rkey.String()
	call := func(method, suffix, body, key string, expected syntax.CID) (int, []byte) {
		t.Helper()
		req, err := http.NewRequestWithContext(ctx, method, server.URL+path+suffix, bytes.NewBufferString(body))
		if err != nil {
			t.Fatal(err)
		}
		if key != "" {
			req.Header.Set("Idempotency-Key", key)
		}
		if expected != "" {
			req.Header.Set("If-Match", expected.String())
		}
		response, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		raw, err := io.ReadAll(response.Body)
		if err != nil {
			t.Fatal(err)
		}
		return response.StatusCode, raw
	}
	if status, raw := call("GET", "/edit", "", "", ""); status != 200 || !bytes.Contains(raw, []byte(event.CID)) {
		t.Fatalf("seed=%d %s", status, raw)
	}
	key := uuid.NewString()
	body := `{"changes":{"text":"edited over HTTP"}}`
	status, raw := call("PATCH", "", body, key, event.CID)
	if status != 200 {
		t.Fatalf("save=%d %s", status, raw)
	}
	var accepted api.PostEditResult
	if err := json.Unmarshal(raw, &accepted); err != nil {
		t.Fatal(err)
	}
	if !accepted.Changed || !accepted.ProjectionPending || accepted.Post != nil || accepted.AcceptedCID != pds.recordCID || accepted.EditedAt == nil || pds.writes != 1 {
		t.Fatalf("acceptance=%s writes=%d", raw, pds.writes)
	}
	if !bytes.Contains(pds.record, []byte(`9007199254740993`)) {
		t.Fatalf("lost opaque value: %s", pds.record)
	}
	event.ID++
	event.Action = "update"
	event.CID = pds.recordCID
	event.Record = pds.record
	event.Rev = pds.commit.Revision
	f.ingest(event)
	f.drain()
	status, raw = call("PATCH", "", body, key, syntax.CID("bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"))
	// An operation key cannot be reused with a different expected CID.
	if status != 409 || pds.writes != 1 {
		t.Fatalf("mutated retry=%d %s writes=%d", status, raw, pds.writes)
	}
	// Replay the exact immutable request; it hydrates only the accepted visible version.
	originalCID := httpEditRecordCID(t, json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"hello recipient","sponsored":false,"createdAt":"2026-05-04T12:00:00Z","futureValue":9007199254740993}`))
	status, raw = call("PATCH", "", body, key, originalCID)
	var replay api.PostEditResult
	if err := json.Unmarshal(raw, &replay); err != nil {
		t.Fatal(err)
	}
	if status != 200 || replay.Post == nil || replay.Post.Text != "edited over HTTP" || replay.ProjectionPending || pds.writes != 1 {
		t.Fatalf("replay=%d %s writes=%d", status, raw, pds.writes)
	}
	status, raw = call("PATCH", "", body, uuid.NewString(), event.CID)
	var unchanged api.PostEditResult
	if err := json.Unmarshal(raw, &unchanged); err != nil {
		t.Fatal(err)
	}
	if status != 200 || unchanged.Changed || unchanged.AcceptedCID != event.CID || unchanged.EditedAt == nil || !unchanged.EditedAt.Equal(*accepted.EditedAt) || pds.writes != 1 {
		t.Fatalf("no-op=%d %s writes=%d", status, raw, pds.writes)
	}
	status, raw = call("PATCH", "", `{"changes":{"text":"stale overwrite"}}`, uuid.NewString(), originalCID)
	if status != 409 || pds.writes != 1 {
		t.Fatalf("stale=%d %s writes=%d", status, raw, pds.writes)
	}
}

type httpEditPDS struct {
	auth.PDSClient
	auth.RepositoryCommandPDSClient
	auth.PDSRecordLister
	t         *testing.T
	uri       syntax.ATURI
	record    json.RawMessage
	recordCID syntax.CID
	commit    auth.RepositoryCommit
	writes    int
}

func (p *httpEditPDS) WithActiveEffects(ctx context.Context, _ []ownerlifecycle.ExpectedOwner, op auth.ActiveEffectPDSOperation) error {
	return op(ctx, p)
}
func (p *httpEditPDS) GetRecord(_ context.Context, _ syntax.DID, _, _ string, out any) (string, error) {
	return p.recordCID.String(), recordjson.Unmarshal(p.record, out)
}
func (p *httpEditPDS) LatestCommit(context.Context, syntax.DID) (syntax.CID, error) {
	return p.commit.CID, nil
}
func (p *httpEditPDS) LatestCommitMetadata(context.Context, syntax.DID) (auth.RepositoryCommit, error) {
	return p.commit, nil
}
func (p *httpEditPDS) ApplyWrites(ctx context.Context, did syntax.DID, head syntax.CID, writes []auth.RepositoryWrite) error {
	_, err := p.ApplyWritesWithCommit(ctx, did, head, writes)
	return err
}
func (p *httpEditPDS) ApplyWritesWithCommit(_ context.Context, _ syntax.DID, head syntax.CID, writes []auth.RepositoryWrite) (auth.RepositoryWriteReceipt, error) {
	if head != p.commit.CID {
		return auth.RepositoryWriteReceipt{}, auth.ErrRepositorySwapConflict
	}
	if len(writes) != 1 || writes[0].Action != "update" || writes[0].ExpectedCID != p.recordCID {
		return auth.RepositoryWriteReceipt{}, auth.ErrRepositorySwapConflict
	}
	raw, err := json.Marshal(writes[0].Record)
	if err != nil {
		return auth.RepositoryWriteReceipt{}, err
	}
	p.writes++
	p.record = raw
	p.recordCID = httpEditRecordCID(p.t, raw)
	p.commit = auth.RepositoryCommit{CID: httpEditRecordCID(p.t, json.RawMessage(`{"head":2}`)), Revision: "3aaaaaaaaaaa3"}
	return auth.RepositoryWriteReceipt{Commit: p.commit, Records: []auth.RepositoryRecordResult{{URI: p.uri, CID: p.recordCID}}}, nil
}
func httpEditRecordCID(t *testing.T, raw json.RawMessage) syntax.CID {
	t.Helper()
	object, err := recordjson.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := atdata.MarshalCBOR(object)
	if err != nil {
		t.Fatal(err)
	}
	id, err := (cid.Prefix{Version: 1, Codec: cid.DagCBOR, MhType: multihash.SHA2_256, MhLength: 32}).Sum(encoded)
	if err != nil {
		t.Fatal(err)
	}
	return syntax.CID(id.String())
}
