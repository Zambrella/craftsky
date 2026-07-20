package tap

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

func TestAdminClientAddRepoUsesTapHTTPEndpointAndIsRetrySafe(t *testing.T) {
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodPost || r.URL.Path != "/repos/add" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			DIDs []string `json:"dids"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if len(body.DIDs) != 1 || body.DIDs[0] != "did:plc:joining" {
			t.Fatalf("dids = %v", body.DIDs)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/channel"
	client, err := NewAdminClient(wsURL, srv.Client())
	if err != nil {
		t.Fatalf("NewAdminClient: %v", err)
	}
	for i := 0; i < 2; i++ {
		if err := client.AddRepo(context.Background(), syntax.DID("did:plc:joining")); err != nil {
			t.Fatalf("AddRepo retry %d: %v", i, err)
		}
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2 idempotent tracking requests", requests)
	}
}

func TestAdminClientRejectsUnsupportedURLAndNonSuccess(t *testing.T) {
	if _, err := NewAdminClient("ftp://tap.example/channel", http.DefaultClient); err == nil {
		t.Fatal("unsupported URL succeeded")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not ready", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	client, err := NewAdminClient(srv.URL+"/channel", srv.Client())
	if err != nil {
		t.Fatalf("NewAdminClient: %v", err)
	}
	if err := client.AddRepo(context.Background(), syntax.DID("did:plc:joining")); err == nil {
		t.Fatal("non-success response succeeded")
	}
}
