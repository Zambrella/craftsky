package index_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"net/http/httptest"
	"testing"

	"social.craftsky/appview/internal/api"
	"social.craftsky/appview/internal/middleware"
	"social.craftsky/appview/internal/tap"
)

// IT-002 / FR-002: the private owner form is based on one current version.
func TestPostEditSeedLoadsCurrentOwnerVersion(t *testing.T) {
	f := newNotificationPipeline(t)
	event := notificationEvent(101, "did:plc:actor", "3aaaaaaaaaaa5", "")
	event.CID = "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"
	f.ingest(event)
	f.drain()
	handler := indexedEditHandler(api.NewPostStore(f.pool), postEditProjectionResolver{}, f.pool)
	request := httptest.NewRequest(http.MethodGet, "/v1/posts/did:plc:actor/3aaaaaaaaaaa5/edit", nil)
	request.SetPathValue("did", event.DID.String())
	request.SetPathValue("rkey", event.Rkey.String())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request.WithContext(middleware.WithDID(request.Context(), event.DID)))
	if response.Code != 200 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var result struct {
		Post                    api.PostResponse
		BaseCID, RecordRevision string
		Editable                struct {
			Text      string
			Sponsored bool
		}
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Post.CID != event.CID.String() || result.BaseCID != event.CID.String() || result.RecordRevision != "" || result.Editable.Text != "hello recipient" {
		t.Fatalf("incomplete owner seed: %s", response.Body.String())
	}
	if response.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatalf("private seed may be cached: %v", response.Header())
	}
}

// IT-002 / FR-002: a post arriving before its membership dependency can
// finish projection while retaining the initial pending source disposition.
func TestPostEditSeedLoadsAfterMembershipDependencyProjection(t *testing.T) {
	f := newNotificationPipeline(t)
	f.exec(`DELETE FROM craftsky_profiles WHERE did='did:plc:actor'`)
	event := notificationEvent(101, "did:plc:actor", "3aaaaaaaaaaa5", "")
	f.ingest(event)
	seedCraftskyMember(t, f.pool, event.DID.String())
	if err := f.store.WakeDependency(context.Background(), tap.Dependency{Kind: "member_did", Key: event.DID.String()}); err != nil {
		t.Fatal(err)
	}
	f.drain()
	var disposition string
	if err := f.pool.QueryRow(context.Background(), `SELECT projection_disposition FROM tap_source_records WHERE uri=$1`, event.URI).Scan(&disposition); err != nil {
		t.Fatal(err)
	}
	if disposition != "pending" {
		t.Fatalf("fixture disposition=%s, want initial pending", disposition)
	}
	handler := indexedEditHandler(api.NewPostStore(f.pool), postEditProjectionResolver{}, f.pool)
	request := httptest.NewRequest(http.MethodGet, "/edit", nil)
	request.SetPathValue("did", event.DID.String())
	request.SetPathValue("rkey", event.Rkey.String())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request.WithContext(middleware.WithDID(request.Context(), event.DID)))
	if response.Code != http.StatusOK {
		t.Fatalf("completed current projection stays blocked: status=%d body=%s", response.Code, response.Body.String())
	}
}

// UT-001 / FR-002, FR-005: seed all supported project values without asking
// the client to round-trip unknown authored extensions.
func TestPostEditSeedIncludesSupportedProjectFields(t *testing.T) {
	f := newNotificationPipeline(t)
	event := notificationEvent(101, "did:plc:actor", "3aaaaaaaaaaa5", `,"project":{"futureProject":9007199254740993,"common":{"craftType":"social.craftsky.feed.defs#sewing","title":"Linen shirt","pattern":{"selfDrafted":true,"name":"My pattern","futurePattern":"keep server-side"},"materials":[{"text":"Linen","futureMaterial":"keep server-side"}],"futureCommon":"keep server-side"},"details":{"$type":"social.craftsky.project.sewing#details","sizeMade":"M","futureDetail":"keep server-side"}}`)
	event.CID = "bafyreicdvexolyvp6j6yksqiib7hihwktt6ogalbvyzvtkj6ecrtqqw5fq"
	f.ingest(event)
	f.drain()
	handler := indexedEditHandler(api.NewPostStore(f.pool), postEditProjectionResolver{}, f.pool)
	request := httptest.NewRequest(http.MethodGet, "/edit", nil)
	request.SetPathValue("did", event.DID.String())
	request.SetPathValue("rkey", event.Rkey.String())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request.WithContext(middleware.WithDID(request.Context(), event.DID)))
	if response.Code != 200 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	var result struct {
		Editable struct {
			Project struct {
				Common struct {
					Title   string
					Pattern struct {
						SelfDrafted bool
						Name        string
					}
					Materials []struct {
						ExistingIndex int
						Text          string
					}
				}
				Details struct{ SizeMade string }
			}
		}
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	project := result.Editable.Project
	if project.Common.Title != "Linen shirt" || !project.Common.Pattern.SelfDrafted || project.Common.Pattern.Name != "My pattern" || project.Details.SizeMade != "M" || len(project.Common.Materials) != 1 || project.Common.Materials[0].Text != "Linen" {
		t.Fatalf("supported project values missing: %s", response.Body.String())
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(response.Body.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(fields["editable"], []byte("future")) {
		t.Fatalf("raw extensions leaked to editable form: %s", fields["editable"])
	}
}

func TestPostEditSeedEligibilityAndVersionBoundaries(t *testing.T) {
	for _, test := range []struct {
		name, sql string
		viewer    syntax.DID
		status    int
	}{
		{"nonOwner", "", "did:plc:recipient", 403},
		{"anonymous", "", "", 403},
		{"sourceAhead", `UPDATE tap_source_records SET cid='new-source-cid' WHERE did='did:plc:actor'`, "did:plc:actor", 200},
		{"uncertain", `UPDATE tap_source_records SET ordering_status='uncertain' WHERE did='did:plc:actor'`, "did:plc:actor", 200},
		{"generationChanged", `UPDATE tap_source_records SET projection_generation=2 WHERE did='did:plc:actor'`, "did:plc:actor", 200},
		{"pendingPostJob", `WITH pending AS (
		 UPDATE tap_source_records SET projection_disposition='pending' WHERE did='did:plc:actor' RETURNING uri
		) UPDATE tap_projection_jobs SET state='pending',completed_at=NULL WHERE source_uri IN (SELECT uri FROM pending) AND projection_kind='social_craftsky_feed_post'`, "did:plc:actor", 200},
		{"completedOtherSourceJob", `WITH pending AS (
		 UPDATE tap_source_records SET projection_disposition='pending' WHERE did='did:plc:actor' RETURNING uri
		) UPDATE tap_projection_jobs SET source_event_id=source_event_id+1 WHERE source_uri IN (SELECT uri FROM pending) AND projection_kind='social_craftsky_feed_post'`, "did:plc:actor", 200},
		{"missingPostJob", `WITH pending AS (
		 UPDATE tap_source_records SET projection_disposition='pending' WHERE did='did:plc:actor' RETURNING uri
		) DELETE FROM tap_projection_jobs WHERE source_uri IN (SELECT uri FROM pending) AND projection_kind='social_craftsky_feed_post'`, "did:plc:actor", 200},
		{"missingMembership", `DELETE FROM craftsky_profiles WHERE did='did:plc:actor'`, "did:plc:actor", 404},
		{"departed", `UPDATE owner_lifecycles SET state='departed',generation=2,transitioned_at=now(),updated_at=now() WHERE owner_did='did:plc:actor'`, "did:plc:actor", 404},
		{"deleted", `UPDATE tap_source_records SET action='delete',record=NULL,record_bytes=0 WHERE did='did:plc:actor'`, "did:plc:actor", 404},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newNotificationPipeline(t)
			event := notificationEvent(101, "did:plc:actor", "3aaaaaaaaaaa5", "")
			f.ingest(event)
			f.drain()
			if test.sql != "" {
				f.exec(test.sql)
			}
			handler := indexedEditHandler(api.NewPostStore(f.pool), postEditProjectionResolver{}, f.pool)
			request := httptest.NewRequest(http.MethodGet, "/edit", nil)
			request.SetPathValue("did", event.DID.String())
			request.SetPathValue("rkey", event.Rkey.String())
			if test.viewer != "" {
				request = request.WithContext(middleware.WithDID(request.Context(), test.viewer))
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status || (test.status != 200 && bytes.Contains(response.Body.Bytes(), []byte("hello recipient"))) {
				t.Fatalf("status=%d want=%d body=%s", response.Code, test.status, response.Body.String())
			}
		})
	}
}

func indexedEditHandler(store *api.PostStore, resolver api.HandleResolver, pool *pgxpool.Pool) http.Handler {
	handler := api.GetPostEditHandler(store, resolver, api.NewPostEditSeedLoader(store, func(ctx context.Context, owner syntax.DID, key syntax.RecordKey, _ int64, _ string) (json.RawMessage, syntax.CID, error) {
		var raw json.RawMessage
		var cid syntax.CID
		err := pool.QueryRow(ctx, `SELECT record,cid FROM tap_source_records WHERE did=$1 AND rkey=$2 AND action<>'delete'`, owner, key).Scan(&raw, &cid)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", api.ErrPostNotFound
		}
		return raw, cid, err
	}))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := middleware.WithOwnerGeneration(r.Context(), 1)
		ctx = middleware.WithOAuthSessionID(ctx, "indexed-seed-fixture")
		handler.ServeHTTP(w, r.WithContext(ctx))
	})
}
