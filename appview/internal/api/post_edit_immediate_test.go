package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5/pgxpool"
	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/auth"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/pdscommands"
	"social.craftsky/appview/internal/testdb"
)

// IT-001 / FR-002: a successful publication is editable before Tap receives it.
func TestEditPostSavesBeforeTapIngestion(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	seedMember(t, pool, "did:plc:alice")
	if _, err := pool.Exec(context.Background(), `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES('did:plc:alice','active',1,1,'test',now(),now(),now())`); err != nil {
		t.Fatal(err)
	}
	const cid = "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"
	commands := &recordingPostEditCommands{pool: pool, current: pdscommands.AuthoritativeRecord{URI: "at://did:plc:alice/social.craftsky.feed.post/3aaaaaaaaaaa2", CID: cid, Record: json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"just posted","sponsored":false,"createdAt":"2026-10-27T12:00:00Z","futureField":9007199254740993}`)}}
	handler := api.CommandEditPostHandler(api.NewPostStore(pool), commands, fakeResolver{handleFor: "alice.example"}, api.DefaultMediaLimits())
	request := httptest.NewRequest(http.MethodPatch, "/v1/posts/did:plc:alice/3aaaaaaaaaaa2", strings.NewReader(`{"changes":{"text":"corrected immediately"}}`))
	request.SetPathValue("did", "did:plc:alice")
	request.SetPathValue("rkey", "3aaaaaaaaaaa2")
	request.Header.Set("If-Match", cid)
	request.Header.Set("Idempotency-Key", "018f4d5c-7a61-7d40-a1a2-777777777777")
	ctx := middleware.WithDID(request.Context(), "did:plc:alice")
	ctx = middleware.WithOwnerGeneration(ctx, 1)
	ctx = middleware.WithOAuthSessionID(ctx, "alice-session")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request.WithContext(ctx))
	if response.Code != 200 {
		t.Fatalf("immediate edit blocked: %d %s", response.Code, response.Body.String())
	}
	if !strings.Contains(string(commands.desired), `"text":"corrected immediately"`) || !strings.Contains(string(commands.desired), `9007199254740993`) {
		t.Fatalf("record lost edits/extensions: %s", commands.desired)
	}
	var result api.PostEditResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.AcceptedCID == "" {
		t.Fatalf("unprojected edit exposed as public: %s", response.Body.String())
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM tap_source_records`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("test requires no Tap source: count=%d err=%v", count, err)
	}
}

// IT-002 / FR-002: authoritative owner prefill exists before any Tap source.
func TestPostEditSeedLoadsAuthoritativeRecordBeforeTap(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	seedMember(t, pool, "did:plc:alice")
	if _, err := pool.Exec(context.Background(), `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES('did:plc:alice','active',1,1,'test',now(),now(),now())`); err != nil {
		t.Fatal(err)
	}
	const cid = syntax.CID("bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq")
	reads := 0
	load := api.NewPostEditSeedLoader(api.NewPostStore(pool), func(ctx context.Context, owner syntax.DID, key syntax.RecordKey, generation int64, session string) (json.RawMessage, syntax.CID, error) {
		reads++
		if owner != "did:plc:alice" || key != "3aaaaaaaaaaa2" || generation != 1 || session != "alice-session" {
			t.Fatal("owner/session binding lost")
		}
		return json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"authoritative caption","sponsored":false,"createdAt":"2026-10-27T12:00:00Z","futureField":9007199254740993}`), cid, nil
	})
	snapshot, err := load(context.Background(), "did:plc:alice", "3aaaaaaaaaaa2", 1, "alice-session")
	if err != nil {
		t.Fatalf("authoritative editor still waits for Tap: %v", err)
	}
	if reads != 1 || snapshot.Post.CID != cid.String() || snapshot.Post.Text != "authoritative caption" || snapshot.Revision != "" {
		t.Fatalf("incorrect authoritative prefill: reads=%d snapshot=%+v", reads, snapshot)
	}
	var count int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM tap_source_records`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("loader mutated Tap state: %d %v", count, err)
	}
}

func immediatePostEditPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool := testdb.WithMigratedSchema(t)
	seedMember(t, pool, "did:plc:alice")
	if _, err := pool.Exec(context.Background(), `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at,created_at,updated_at) VALUES('did:plc:alice','active',1,1,'test',now(),now(),now())`); err != nil {
		t.Fatal(err)
	}
	return pool
}

func serveAuthoritativePostEditSeed(store *api.PostStore, read api.PostEditRecordReader, caller syntax.DID) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, "/v1/posts/did:plc:alice/3aaaaaaaaaaa2/edit", nil)
	request.SetPathValue("did", "did:plc:alice")
	request.SetPathValue("rkey", "3aaaaaaaaaaa2")
	ctx := middleware.WithDID(request.Context(), caller)
	ctx = middleware.WithOwnerGeneration(ctx, 1)
	ctx = middleware.WithOAuthSessionID(ctx, "alice-session")
	response := httptest.NewRecorder()
	api.GetPostEditHandler(store, fakeResolver{handleFor: "alice.example"}, api.NewPostEditSeedLoader(store, read)).ServeHTTP(response, request.WithContext(ctx))
	return response
}

func TestPostEditAuthoritativePrefillVariantsBeforeTap(t *testing.T) {
	const cid = syntax.CID("bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq")
	const blob = `{"$type":"blob","ref":{"$link":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"},"mimeType":"image/jpeg","size":200}`
	const ref = `{"uri":"at://did:plc:bob/social.craftsky.feed.post/3aaaaaaaaaaa3","cid":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"}`
	for _, tc := range []struct{ name, extra string }{
		{"projectImage", `,"images":[{"image":` + blob + `,"alt":"Retained image","aspectRatio":{"width":4,"height":3}}],"project":{"futureProject":"server only","common":{"craftType":"social.craftsky.feed.defs#sewing","title":"Shirt","materials":[{"text":"Linen","futureMaterial":"server only"}],"pattern":{"selfDrafted":true}},"details":{"$type":"social.craftsky.project.sewing#details","sizeMade":"M"}}`},
		{"external", `,"embed":{"$type":"app.bsky.embed.external","external":{"uri":"https://example.com/pattern","title":"Pattern","description":"Details","thumb":` + blob + `}}`},
		{"video", `,"embed":{"$type":"app.bsky.embed.video","video":{"$type":"blob","ref":{"$link":"bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"},"mimeType":"video/mp4","size":200},"alt":"Retained video"}`},
		{"reply", `,"reply":{"root":` + ref + `,"parent":` + ref + `}`},
		{"quote", `,"embed":{"$type":"social.craftsky.feed.post#quoteEmbed","record":` + ref + `}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool := immediatePostEditPool(t)
			raw := json.RawMessage(`{"$type":"social.craftsky.feed.post","text":"PDS current caption","sponsored":true,"langs":["fr"],"createdAt":"2026-10-27T12:00:00Z"` + tc.extra + `}`)
			// A stale serving row is deliberately present; it must not seed the editor.
			seedPost(t, pool, "did:plc:alice", "3aaaaaaaaaaa2", "Stale indexed caption", time.Now())
			response := serveAuthoritativePostEditSeed(api.NewPostStore(pool), func(context.Context, syntax.DID, syntax.RecordKey, int64, string) (json.RawMessage, syntax.CID, error) {
				return raw, cid, nil
			}, "did:plc:alice")
			if response.Code != 200 {
				t.Fatalf("prefill=%d %s", response.Code, response.Body.String())
			}
			var seed struct {
				Post           api.PostResponse
				BaseCID        string
				RecordRevision *string
				Editable       map[string]json.RawMessage
			}
			if err := json.Unmarshal(response.Body.Bytes(), &seed); err != nil {
				t.Fatal(err)
			}
			if seed.BaseCID != cid.String() || seed.Post.Text != "PDS current caption" || !seed.Post.Sponsored || len(seed.Post.Langs) != 1 || seed.Post.Langs[0] != "fr" || seed.RecordRevision != nil {
				t.Fatalf("prefill used stale/invented metadata: %s", response.Body.String())
			}
			if response.Header().Get("Cache-Control") != "private, no-store" {
				t.Fatal("owner seed may be cached")
			}
			editable, _ := json.Marshal(seed.Editable)
			if strings.Contains(response.Body.String(), "futureProject") || strings.Contains(string(editable), "future") {
				t.Fatalf("raw extension leaked to editor: %s", editable)
			}
			switch tc.name {
			case "projectImage":
				if seed.Post.Project == nil || seed.Post.Project.Common.Title == nil || *seed.Post.Project.Common.Title != "Shirt" || len(seed.Post.Images) != 1 || seed.Post.Images[0].Fullsize == "" || seed.Post.Images[0].Alt != "Retained image" || !strings.Contains(string(seed.Editable["project"]), `"existingIndex":0`) {
					t.Fatalf("project/media incomplete: %s", response.Body.String())
				}
			case "external":
				if seed.Post.External == nil || seed.Post.External.Thumb == nil {
					t.Fatalf("external incomplete: %s", response.Body.String())
				}
			case "video":
				if seed.Post.Video == nil || seed.Post.Video.Alt != "Retained video" {
					t.Fatalf("video incomplete: %s", response.Body.String())
				}
			case "reply":
				if seed.Post.Reply == nil {
					t.Fatal("fixed reply lost")
				}
			case "quote":
				if seed.Post.Quote == nil {
					t.Fatal("fixed quote lost")
				}
			}
		})
	}
}

func TestPostEditAuthoritativeSeedPreservesRestrictions(t *testing.T) {
	for _, name := range []string{"nonOwner", "missingMember", "departed", "postHidden", "accountHidden", "matchedMedia"} {
		t.Run(name, func(t *testing.T) {
			pool := immediatePostEditPool(t)
			caller := syntax.DID("did:plc:alice")
			switch name {
			case "nonOwner":
				caller = "did:plc:bob"
			case "missingMember":
				if _, err := pool.Exec(context.Background(), `DELETE FROM craftsky_profiles WHERE did='did:plc:alice'`); err != nil {
					t.Fatal(err)
				}
			case "departed":
				if _, err := pool.Exec(context.Background(), `UPDATE owner_lifecycles SET state='departed',generation=2 WHERE owner_did='did:plc:alice'`); err != nil {
					t.Fatal(err)
				}
			case "postHidden":
				seedModerationOutput(t, pool, "post", "did:plc:alice", "at://did:plc:alice/social.craftsky.feed.post/3aaaaaaaaaaa2", "hide", time.Now())
			case "accountHidden":
				seedModerationOutput(t, pool, "account", "did:plc:alice", "", "takedown", time.Now())
			case "matchedMedia":
				if _, err := pool.Exec(context.Background(), `INSERT INTO image_subject_states(subject_uri,subject_kind,source_cid,visibility_state) VALUES('at://did:plc:alice/social.craftsky.feed.post/3aaaaaaaaaaa2','post','source-cid','blocked');
   WITH scan AS (INSERT INTO image_scan_results(id,blob_cid,scanner_id,policy_version,corpus_version,state,completed_at) VALUES(gen_random_uuid(),'blob-cid','test','1','1','match',now()) RETURNING id)
   INSERT INTO image_subject_requirements(subject_uri,source_cid,subject_kind,image_slot,blob_cid,scan_result_id) SELECT 'at://did:plc:alice/social.craftsky.feed.post/3aaaaaaaaaaa2','source-cid','post','images.0','blob-cid',id FROM scan`); err != nil {
					t.Fatal(err)
				}
			}
			reads := 0
			response := serveAuthoritativePostEditSeed(api.NewPostStore(pool), func(context.Context, syntax.DID, syntax.RecordKey, int64, string) (json.RawMessage, syntax.CID, error) {
				reads++
				return nil, "", nil
			}, caller)
			want := 404
			if name == "nonOwner" {
				want = 403
			}
			if response.Code != want || reads != 0 {
				t.Fatalf("restriction bypassed: status=%d want=%d PDS reads=%d body=%s", response.Code, want, reads, response.Body.String())
			}
		})
	}
}

func TestPostEditAuthoritativeSeedRejectsInvalidOrMissingRecord(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		err       error
		status    int
	}{
		{"missing", "", auth.ErrRecordNotFound, 404},
		{"invalidTimestamp", `{"$type":"social.craftsky.feed.post","text":"private invalid text","sponsored":false,"createdAt":"2026-01-01T00:00:00Z","editedAt":"invalid"}`, nil, 422},
		{"invalidLanguage", `{"$type":"social.craftsky.feed.post","text":"private invalid text","sponsored":false,"createdAt":"2026-01-01T00:00:00Z","langs":["invalid_language"]}`, nil, 422},
		{"expiredSession", "", auth.ErrPDSSessionExpired, 401},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pool := immediatePostEditPool(t)
			response := serveAuthoritativePostEditSeed(api.NewPostStore(pool), func(context.Context, syntax.DID, syntax.RecordKey, int64, string) (json.RawMessage, syntax.CID, error) {
				return json.RawMessage(tc.raw), "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq", tc.err
			}, "did:plc:alice")
			if response.Code != tc.status || strings.Contains(response.Body.String(), "private invalid text") {
				t.Fatalf("invalid/missing source exposed: %d %s", response.Code, response.Body.String())
			}
		})
	}
}
