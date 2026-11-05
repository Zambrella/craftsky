package video

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"social.craftsky/appview/internal/testdb"
)

func metadataResponse(body string) *http.Response {
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}
}

func TestBunnyCollectionConcurrentAllocationsReuseExactDID(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	const collection = "bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb"
	const owner = "did:plc:collection-owner"
	var mu sync.Mutex
	created := 0
	transport := bunnyRoundTrip(func(r *http.Request) (*http.Response, error) {
		mu.Lock()
		defer mu.Unlock()
		if r.Header.Get("AccessKey") != "key-canary" || r.URL.Host != "video.bunnycdn.com" {
			t.Error("wrong credential boundary")
		}
		if r.Method == "GET" {
			if r.URL.Query().Get("search") != owner {
				t.Error("DID search was not encoded correctly")
			}
			if created > 0 {
				return metadataResponse(`{"totalItems":1,"items":[{"guid":"` + collection + `","name":"` + owner + `"}]}`), nil
			}
			// Similar names returned by substring search must not be reused.
			return metadataResponse(`{"totalItems":1,"items":[{"guid":"aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa","name":"` + owner + `-other"}]}`), nil
		}
		var input map[string]string
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input["name"] != owner {
			t.Error("collection name is not the exact DID", err)
		}
		created++
		return metadataResponse(`{"guid":"` + collection + `"}`), nil
	})
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Independent service instances share only the database lock.
			b := &Bunny{pool: pool, library: "123", apiKey: "key-canary", client: &http.Client{Transport: transport}}
			id, err := b.ownerCollection(context.Background(), owner)
			if err != nil || id != collection {
				t.Errorf("collection = %q, %v", id, err)
			}
		}()
	}
	wg.Wait()
	if created != 1 {
		t.Fatalf("created %d collections for one owner", created)
	}
}

func TestBunnyCollectionPaginationAndImportMetadata(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	b := &Bunny{pool: pool, library: "123", apiKey: "key-canary"}
	const collection = "bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb"
	const video = "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa"
	const job = "cccccccc-cccc-4ccc-cccc-cccccccccccc"
	var calls []string
	b.client = &http.Client{Transport: bunnyRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		if strings.HasSuffix(r.URL.Path, "/collections") {
			if r.Method != "GET" {
				t.Fatal("created a collection instead of finding page two")
			}
			if r.URL.Query().Get("page") == "1" {
				return metadataResponse(`{"totalItems":101,"items":[]}`), nil
			}
			return metadataResponse(`{"totalItems":101,"items":[{"guid":"` + collection + `","name":"did:plc:owner"}]}`), nil
		}
		var input map[string]any
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(r.URL.Path, "/videos") {
			if input["title"] != "2026-11-05 12:34 UTC · cccccccc" || input["collectionId"] != collection {
				t.Fatal("incorrect title or collection", input)
			}
			return metadataResponse(`{"guid":"` + video + `"}`), nil
		}
		if len(input) != 1 {
			t.Fatal("unexpected metadata payload fields")
		}
		tags := input["metaTags"].([]any)
		want := map[string]string{"craftsky:uploadJobId": job, "craftsky:source": "pds-import", "craftsky:pdsBlobCid": "bafy-source"}
		for _, raw := range tags {
			tag := raw.(map[string]any)
			property := tag["property"].(string)
			if want[property] != tag["value"] {
				t.Fatal("incorrect import metadata", tag)
			}
			delete(want, property)
		}
		if len(want) != 0 {
			t.Fatal("missing import metadata", want)
		}
		return metadataResponse(`{}`), nil
	})}
	when := time.Date(2026, 11, 5, 13, 34, 0, 0, time.FixedZone("+1", 3600))
	id, err := b.createOrganizedVideo(context.Background(), "did:plc:owner", job, when)
	if err != nil || id.ID != video {
		t.Fatal(id, err)
	}
	if err = b.setVideoMetadata(context.Background(), id.ID, job, "pds-import", "bafy-source"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 4 {
		t.Fatal(fmt.Sprint(calls))
	}
}

func TestBunnyCollectionLookupFailureDoesNotCreate(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	b := &Bunny{pool: pool, library: "123", apiKey: "key-canary"}
	b.client = &http.Client{Transport: bunnyRoundTrip(func(r *http.Request) (*http.Response, error) {
		if r.Method != "GET" {
			t.Fatal("collection created after failed lookup")
		}
		return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})}
	if _, err := b.ownerCollection(context.Background(), "did:plc:owner"); err != ErrBunnyUnavailable {
		t.Fatal(err)
	}
}

func TestBunnyInitialMetadataFailureRetainsTrackedAsset(t *testing.T) {
	pool := testdb.WithMigratedSchema(t)
	ctx := context.Background()
	const owner = "did:plc:owner"
	const video = "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa"
	if _, err := pool.Exec(ctx, `INSERT INTO owner_lifecycles(owner_did,state,generation,auth_epoch,transition_reason,transitioned_at)
 VALUES($1,'active',1,1,'test',now())`, owner); err != nil {
		t.Fatal(err)
	}
	b, err := NewBunny(pool, "123", "key-canary", "test.b-cdn.net", func(context.Context, syntax.DID, string) (BunnyBlobEffects, error) { return &bunnyEffects{}, nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	b.client.Transport = bunnyRoundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/library/123/collections" {
			return metadataResponse(`{"totalItems":1,"items":[{"guid":"bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb","name":"` + owner + `"}]}`), nil
		}
		if r.URL.Path == "/library/123/videos" {
			return metadataResponse(`{"guid":"` + video + `"}`), nil
		}
		return &http.Response{StatusCode: 503, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
	})
	if _, err = b.IssueUpload(ctx, owner, "session"); err != ErrBunnyUnavailable {
		t.Fatal(err)
	}
	var tracked string
	if err = pool.QueryRow(ctx, `SELECT video_id::text FROM bunny_video_jobs WHERE owner_did=$1`, owner).Scan(&tracked); err != nil || tracked != video {
		t.Fatal("metadata failure lost cleanup authority", tracked, err)
	}
}
