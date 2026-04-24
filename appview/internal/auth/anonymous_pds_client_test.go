package auth_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"social.craftsky/appview/internal/auth"
)

// NOTE: the `social.craftsky/appview/internal/auth` import is added in
// Task 2.2 Step 1 when the first test references `auth.NewAnonymousPDSClient`.
// Adding it here would fail Go's unused-import check because the scaffold
// test below only touches `identity`, `syntax`, and stdlib types.

// fakeDirectory is a minimal identity.Directory that returns a hard-coded
// PDSEndpoint for a single DID. Tests set `endpoint` to an httptest
// server's URL so the anonymous client hits a controllable fake PDS.
type fakeDirectory struct {
	did      syntax.DID
	endpoint string // empty → no PDS entry in DID doc
	err      error  // set to exercise lookup-failure paths
}

func (f *fakeDirectory) LookupDID(_ context.Context, did syntax.DID) (*identity.Identity, error) {
	if f.err != nil {
		return nil, f.err
	}
	if did != f.did {
		return nil, errors.New("unknown DID")
	}
	// identity.Identity exposes PDSEndpoint() through its Services map;
	// populate that directly.
	ident := &identity.Identity{DID: did}
	if f.endpoint != "" {
		ident.Services = map[string]identity.ServiceEndpoint{
			"atproto_pds": {Type: "AtprotoPersonalDataServer", URL: f.endpoint},
		}
	}
	return ident, nil
}

func (f *fakeDirectory) LookupHandle(context.Context, syntax.Handle) (*identity.Identity, error) {
	return nil, errors.New("not used")
}

func (f *fakeDirectory) Lookup(context.Context, syntax.AtIdentifier) (*identity.Identity, error) {
	return nil, errors.New("not used")
}

func (f *fakeDirectory) Purge(context.Context, syntax.AtIdentifier) error { return nil }

// Sanity check the fake before using it in test tables below.
func TestFakeDirectory_ShapesIdentity(t *testing.T) {
	f := &fakeDirectory{did: syntax.DID("did:plc:abc"), endpoint: "https://example.test"}
	ident, err := f.LookupDID(context.Background(), syntax.DID("did:plc:abc"))
	if err != nil {
		t.Fatal(err)
	}
	if got := ident.PDSEndpoint(); got != "https://example.test" {
		t.Errorf("PDSEndpoint = %q", got)
	}
}

// helperServer returns an httptest.Server that serves a single
// com.atproto.repo.getRecord response matching the given status+body.
// The `path` var captures the incoming request path for assertion.
func helperServer(_ *testing.T, status int, body string, path *string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if path != nil {
			*path = r.URL.RequestURI()
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
}

func TestAnonymousPDSClient_GetRecord_HappyPath(t *testing.T) {
	t.Parallel()
	var gotPath string
	srv := helperServer(t, 200, `{
        "uri":"at://did:plc:abc/app.bsky.actor.profile/self",
        "cid":"bafcid",
        "value":{"displayName":"alice"}
    }`, &gotPath)
	defer srv.Close()

	dir := &fakeDirectory{did: syntax.DID("did:plc:abc"), endpoint: srv.URL}
	cli := auth.NewAnonymousPDSClient(dir, 2*time.Second)

	var out map[string]any
	cid, err := cli.GetRecord(context.Background(),
		syntax.DID("did:plc:abc"), "app.bsky.actor.profile", "self", &out)
	if err != nil {
		t.Fatalf("GetRecord: %v", err)
	}
	if cid != "bafcid" {
		t.Errorf("cid = %q, want bafcid", cid)
	}
	if out["displayName"] != "alice" {
		t.Errorf("displayName = %v", out["displayName"])
	}
	if !strings.HasPrefix(gotPath, "/xrpc/com.atproto.repo.getRecord") {
		t.Errorf("path = %q", gotPath)
	}
}

func TestAnonymousPDSClient_GetRecord_RecordNotFound(t *testing.T) {
	t.Parallel()
	// Real PDSes signal missing records with HTTP 400 + XRPC error name
	// "RecordNotFound". The translate helper recognises that shape.
	srv := helperServer(t, 400,
		`{"error":"RecordNotFound","message":"Could not locate record"}`, nil)
	defer srv.Close()

	dir := &fakeDirectory{did: syntax.DID("did:plc:abc"), endpoint: srv.URL}
	cli := auth.NewAnonymousPDSClient(dir, 2*time.Second)

	var out map[string]any
	_, err := cli.GetRecord(context.Background(),
		syntax.DID("did:plc:abc"), "app.bsky.actor.profile", "self", &out)
	if !errors.Is(err, auth.ErrRecordNotFound) {
		t.Errorf("want ErrRecordNotFound; got %v", err)
	}
}

func TestAnonymousPDSClient_GetRecord_NoPDSEndpoint(t *testing.T) {
	t.Parallel()
	dir := &fakeDirectory{did: syntax.DID("did:plc:abc")} // endpoint empty
	cli := auth.NewAnonymousPDSClient(dir, 2*time.Second)

	var out map[string]any
	_, err := cli.GetRecord(context.Background(),
		syntax.DID("did:plc:abc"), "app.bsky.actor.profile", "self", &out)
	if err == nil {
		t.Fatal("want error when DID doc has no PDS endpoint; got nil")
	}
	if !strings.Contains(err.Error(), "no atproto_pds") {
		t.Errorf("err = %v; want mention of missing endpoint", err)
	}
}
