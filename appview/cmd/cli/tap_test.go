package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTapStatusExitConnected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","db":"ok","tap":{"connected":true,"last_event_at":"2026-04-17T14:23:11Z","reconnect_attempt":0,"last_error":""}}`))
	}))
	defer srv.Close()

	code := tapStatus(srv.URL, nil)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
}

func TestTapStatusExitDisconnected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"degraded","db":"ok","tap":{"connected":false,"last_event_at":"","reconnect_attempt":3,"last_error":"dial tcp: ..."}}`))
	}))
	defer srv.Close()

	code := tapStatus(srv.URL, nil)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
}

func TestTapStatusExitTransport(t *testing.T) {
	// Point at a closed port.
	code := tapStatus("http://127.0.0.1:1", nil)
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}

func TestTapStatusExitNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"tap":{"connected":false}}`))
	}))
	defer srv.Close()

	code := tapStatus(srv.URL, nil)
	if code != 2 {
		t.Errorf("exit code = %d, want 2 (non-2xx should be transport error, not disconnected)", code)
	}
}

func TestTapStatusExitGarbageBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	code := tapStatus(srv.URL, nil)
	if code != 2 {
		t.Errorf("exit code = %d, want 2 (parse error)", code)
	}
}

func TestTapHTTPBaseURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "ws channel", in: "ws://tap:2480/channel", want: "http://tap:2480"},
		{name: "wss channel", in: "wss://tap.example/channel", want: "https://tap.example"},
		{name: "http base", in: "http://tap:2480", want: "http://tap:2480"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tapHTTPBaseURL(tt.in)
			if err != nil {
				t.Fatalf("tapHTTPBaseURL: %v", err)
			}
			if got != tt.want {
				t.Fatalf("tapHTTPBaseURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestTapHTTPBaseURLUnsupportedScheme(t *testing.T) {
	if _, err := tapHTTPBaseURL("ftp://tap/channel"); err == nil {
		t.Fatal("tapHTTPBaseURL unsupported scheme err = nil, want error")
	}
}

func TestDiffRepoRecords(t *testing.T) {
	indexedAt := time.Date(2026, 6, 27, 20, 0, 0, 0, time.UTC)
	local := map[string]repoRecord{
		"at://did:plc:a/social.craftsky.feed.post/stale":    {URI: "at://did:plc:a/social.craftsky.feed.post/stale", CID: "local-stale", Rkey: "stale", IndexedAt: indexedAt},
		"at://did:plc:a/social.craftsky.feed.post/same":     {URI: "at://did:plc:a/social.craftsky.feed.post/same", CID: "same", Rkey: "same"},
		"at://did:plc:a/social.craftsky.feed.post/mismatch": {URI: "at://did:plc:a/social.craftsky.feed.post/mismatch", CID: "old", Rkey: "mismatch"},
	}
	remote := map[string]repoRecord{
		"at://did:plc:a/social.craftsky.feed.post/same":     {URI: "at://did:plc:a/social.craftsky.feed.post/same", CID: "same", Rkey: "same"},
		"at://did:plc:a/social.craftsky.feed.post/mismatch": {URI: "at://did:plc:a/social.craftsky.feed.post/mismatch", CID: "new", Rkey: "mismatch"},
		"at://did:plc:a/social.craftsky.feed.post/missing":  {URI: "at://did:plc:a/social.craftsky.feed.post/missing", CID: "remote-missing", Rkey: "missing"},
	}

	diff := diffRepoRecords(local, remote)
	if len(diff.StaleLocal) != 1 || diff.StaleLocal[0].Rkey != "stale" {
		t.Fatalf("stale local = %+v, want stale", diff.StaleLocal)
	}
	if len(diff.MissingLocal) != 1 || diff.MissingLocal[0].Rkey != "missing" {
		t.Fatalf("missing local = %+v, want missing", diff.MissingLocal)
	}
	if len(diff.CIDMismatch) != 1 || diff.CIDMismatch[0].LocalCID != "old" || diff.CIDMismatch[0].PDSCID != "new" {
		t.Fatalf("cid mismatch = %+v, want old/new", diff.CIDMismatch)
	}
}

func TestRkeyFromURI(t *testing.T) {
	got := rkeyFromURI("at://did:plc:a/social.craftsky.feed.post/3abc")
	if got != "3abc" {
		t.Fatalf("rkeyFromURI = %q, want 3abc", got)
	}
}
