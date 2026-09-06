package tap_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/tap"
)

// IT-011 pins only the Tap behavior Craftsky consumes: DID-scoped tracking and
// lossless delivery of live/backfill revision metadata. Snapshot completeness,
// signing-key verification, and reset-chain convergence belong to AppView.
func TestPinnedTapMigrationContract(t *testing.T) {
	compose, err := os.ReadFile("../../../docker-compose.yml")
	if err != nil {
		t.Fatalf("read compose Tap pin: %v", err)
	}
	if !strings.Contains(string(compose), "image: ghcr.io/bluesky-social/indigo/tap:0.1.10") {
		t.Fatal("Craftsky migration contract requires the reviewed Tap 0.1.10 pin")
	}

	owner := syntax.DID("did:plc:migrating")
	for _, variant := range []string{"unchanged handle", "changed handle"} {
		t.Run(variant, func(t *testing.T) {
			var tracked []syntax.DID
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/repos/add" {
					t.Errorf("Tap admin request = %s %s", r.Method, r.URL.Path)
				}
				var body struct {
					DIDs []syntax.DID `json:"dids"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Errorf("decode Tap admin request: %v", err)
				}
				tracked = append(tracked, body.DIDs...)
				w.WriteHeader(http.StatusNoContent)
			}))
			t.Cleanup(server.Close)
			client, err := tap.NewAdminClient(server.URL+"/channel", server.Client())
			if err != nil {
				t.Fatalf("new Tap admin client: %v", err)
			}
			if err := client.AddRepo(context.Background(), owner); err != nil {
				t.Fatalf("track migrated repository: %v", err)
			}
			if len(tracked) != 1 || tracked[0] != owner {
				t.Fatalf("tracked repositories = %v, want only stable owner DID", tracked)
			}

			ingestor := &durableIngestorSpy{}
			for _, envelope := range []string{
				`{"id":41,"type":"record","record":{"live":true,"rev":"3zzzzzzzzzzzz","did":"did:plc:migrating","collection":"social.craftsky.feed.post","rkey":"updated","action":"update","cid":"bafyold","record":{"$type":"social.craftsky.feed.post","text":"old","createdAt":"2026-09-03T12:00:00Z"}}}`,
				`{"id":42,"type":"record","record":{"live":false,"rev":"3bbbbbbbbbbb2","did":"did:plc:migrating","collection":"social.craftsky.feed.post","rkey":"created","action":"create","cid":"bafynew","record":{"$type":"social.craftsky.feed.post","text":"backfill","createdAt":"2026-09-03T12:00:00Z"}}}`,
			} {
				outcome, err := tap.ReplayEnvelope(context.Background(), []byte(envelope), ingestor)
				if err != nil || outcome.Kind != tap.OutcomeApplied {
					t.Fatalf("replay migration envelope: outcome=%+v err=%v", outcome, err)
				}
			}
			events := ingestor.recordEvents()
			if len(events) != 2 || events[0].DID != owner || events[1].DID != owner ||
				!events[0].Live || events[1].Live || events[1].Rev != "3bbbbbbbbbbb2" {
				t.Fatalf("Tap migration events = %+v", events)
			}
		})
	}
}
