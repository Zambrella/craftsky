# Bluesky Profile Backfill on Craftsky Membership — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the MST-ordering race where `app.bsky.actor.profile` is dropped by the membership gate during Tap backfill because the `social.craftsky.actor.profile` row hasn't landed yet. When `CraftskyProfile.Handle` commits a genuinely new row, synchronously fetch the user's Bluesky profile from an anonymous PDS client and feed it back through `BlueskyProfile.Handle`.

**Architecture:** Two surgical changes plus one new component. (1) `PDSClient.GetRecord` grows a CID return value (small blast radius, all callers enumerated). (2) A new `auth.AnonymousPDSClient` wraps an unauthenticated `atclient.APIClient`. (3) A new `index.BlueskyBackfiller` chains the PDS read into `BlueskyProfile.Handle` via a synthesised `tap.Event`. (4) `CraftskyProfile.Handle` uses `RETURNING xmax = 0 AS created` to detect true inserts and invokes the backfiller — errors swallowed (best-effort). Everything intra-package in `internal/index`; no circular-import risk.

**Tech Stack:** Go 1.22+, `pgx/v5` (including `pgx.ErrNoRows`), `indigo` (`atproto/atclient`, `atproto/identity`, `atproto/syntax`), stdlib `net/http`. Tests run via `just test` against the compose Postgres at `localhost:5433`.

---

## Background reading for the implementer

Read these before starting. All paths relative to repo root.

- [`docs/superpowers/specs/2026-04-24-bluesky-backfill-ordering-race-design.md`](../specs/2026-04-24-bluesky-backfill-ordering-race-design.md) — **the spec this plan implements. Primary source of truth.**
- [`docs/superpowers/specs/2026-04-23-profile-onboarding-design.md`](../specs/2026-04-23-profile-onboarding-design.md) — the profile feature this extends; §3 defines membership gating.
- [`appview/internal/index/craftsky_profile.go`](../../../appview/internal/index/craftsky_profile.go) and [`bluesky_profile.go`](../../../appview/internal/index/bluesky_profile.go) — the two indexers being modified/called.
- [`appview/internal/auth/pds_client.go`](../../../appview/internal/auth/pds_client.go) and [`pds_client_indigo.go`](../../../appview/internal/auth/pds_client_indigo.go) — interface + existing authenticated implementation.
- [`appview/internal/auth/initialize_profile.go`](../../../appview/internal/auth/initialize_profile.go) — existing caller of `PDSClient.GetRecord`.
- [`appview/internal/app/deps.go`](../../../appview/internal/app/deps.go) — wiring; `identity.Directory` already constructed here.
- [`AGENTS.md`](../../../AGENTS.md) — project rules. Coding conventions, Go toolchain, testing posture.

## Conventions this plan follows

- **TDD.** Every task writes the test first, confirms it fails, writes the minimal code to pass, confirms it passes, commits. This is rigid — don't batch.
- **One commit per task.** Tasks are small. Frequent commits make reverts cheap.
- **`just test` is the only test runner.** Requires `just dev-d` running (Postgres on `localhost:5433`).
- **`just fmt` after every non-trivial change.** Do not commit Go files that haven't been `gofmt`'d.
- **Naming.** camelCase in JSON on `/v1/*` (not relevant here), snake_case for SQL identifiers, UpperCamelCase for Go exports.
- **No emojis in code or commit messages.**

## File structure

All paths relative to repo root.

**New files:**

- `appview/internal/auth/anonymous_pds_client.go` — `auth.AnonymousPDSClient`: read-only, unauthenticated, DID-doc-resolving. ~80 lines.
- `appview/internal/auth/anonymous_pds_client_test.go` — unit tests with a stubbed `identity.Directory`.
- `appview/internal/index/bluesky_backfiller.go` — `BlueskyBackfiller` interface + `blueskyBackfiller` concrete impl. ~60 lines.
- `appview/internal/index/bluesky_backfiller_test.go` — unit tests with a fake `auth.PDSClient` and a real `BlueskyProfile` against a test-schema pool.

**Modified files:**

- `appview/internal/auth/pds_client.go` — `GetRecord` signature gains `(cid string, err error)` return.
- `appview/internal/auth/pds_client_indigo.go` — `IndigoPDSClient.GetRecord` returns the response's `cid` field.
- `appview/internal/auth/initialize_profile.go` — ignore the new `cid` return (two call sites).
- `appview/internal/auth/initialize_profile_test.go` — update `mockPDS.GetRecord` signature.
- `appview/internal/auth/handlers_test.go` — update `noopPDSClient.GetRecord` and `erroringGetPDSClient.GetRecord`.
- `appview/internal/api/profile.go` — ignore the new `cid` return (one call site).
- `appview/internal/api/profile_test.go` — update `fakePDSForPut.GetRecord`.
- `appview/internal/index/craftsky_profile.go` — constructor gains `(BlueskyBackfiller, *slog.Logger)`; upsert uses `RETURNING xmax = 0 AS created`; new-row branch calls backfiller.
- `appview/internal/index/craftsky_profile_test.go` — update all `NewCraftskyProfile` call sites to pass a fake backfiller and a logger; add new-row/replay assertions.
- `appview/internal/app/deps.go` — wire `AnonymousPDSClient` + `blueskyBackfiller` into `NewCraftskyProfile`.

**Unchanged but exercised by tests:**

- `appview/internal/index/bluesky_profile.go` — its membership gate + upsert is the thing the backfiller feeds. No code change, but the backfiller integration test asserts the gate passes post-create.

---

## Chunk 1: Signature change — `PDSClient.GetRecord` gains a CID return

The backfiller needs the record CID to synthesise a `tap.Event` that `BlueskyProfile.Handle` can write into `bluesky_profiles.record_cid` (`NOT NULL`). Today `PDSClient.GetRecord` drops it. This chunk changes the signature and sweeps every call site and mock in one compile-clean commit so the tree stays green.

Do this first because every later chunk depends on the new signature.

### Task 1.1: Change the interface

**Files:**
- Modify: `appview/internal/auth/pds_client.go`

- [ ] **Step 1: Edit the interface**

Replace:

```go
type PDSClient interface {
    GetRecord(ctx context.Context, repo syntax.DID, collection string, rkey string, out any) error
    PutRecord(ctx context.Context, repo syntax.DID, collection string, rkey string, record any) error
}
```

with:

```go
type PDSClient interface {
    GetRecord(ctx context.Context, repo syntax.DID, collection string, rkey string, out any) (cid string, err error)
    PutRecord(ctx context.Context, repo syntax.DID, collection string, rkey string, record any) error
}
```

Update the doc comment on `PDSClient` to note that `GetRecord` returns the record CID alongside the decoded value; `cid` is always populated on success and empty on error.

- [ ] **Step 2: Run `go build ./...` to confirm the break is visible**

Run: `cd appview && go build ./...`
Expected: FAIL — every implementation and mock fails to compile because they haven't been updated yet. Read the list of errors; it should match §3.5 of the spec exactly. If a call site you don't recognise shows up, stop and investigate.

- [ ] **Step 3: Do not commit yet**

Tree doesn't compile. Move on to 1.2.

### Task 1.2: Update `IndigoPDSClient.GetRecord` to return the CID

**Files:**
- Modify: `appview/internal/auth/pds_client_indigo.go`

- [ ] **Step 1: Change `IndigoPDSClient.GetRecord`**

The existing body already unmarshals a `resp` struct with a `CID` field. Return it. Replace:

```go
func (i *IndigoPDSClient) GetRecord(ctx context.Context, repo syntax.DID, collection, rkey string, out any) error {
    nsid, err := syntax.ParseNSID("com.atproto.repo.getRecord")
    if err != nil {
        return fmt.Errorf("parse nsid: %w", err)
    }
    var resp struct {
        URI   string `json:"uri"`
        CID   string `json:"cid"`
        Value any    `json:"value"`
    }
    params := map[string]any{
        "repo":       repo.String(),
        "collection": collection,
        "rkey":       rkey,
    }
    if err := i.Client.Get(ctx, nsid, params, &resp); err != nil {
        return translateGetRecordError(err)
    }
    if m, ok := out.(*map[string]any); ok {
        if v, ok := resp.Value.(map[string]any); ok {
            *m = v
            return nil
        }
        return fmt.Errorf("getRecord value has unexpected type %T", resp.Value)
    }
    return fmt.Errorf("unsupported out type %T", out)
}
```

with:

```go
func (i *IndigoPDSClient) GetRecord(ctx context.Context, repo syntax.DID, collection, rkey string, out any) (string, error) {
    nsid, err := syntax.ParseNSID("com.atproto.repo.getRecord")
    if err != nil {
        return "", fmt.Errorf("parse nsid: %w", err)
    }
    var resp struct {
        URI   string `json:"uri"`
        CID   string `json:"cid"`
        Value any    `json:"value"`
    }
    params := map[string]any{
        "repo":       repo.String(),
        "collection": collection,
        "rkey":       rkey,
    }
    if err := i.Client.Get(ctx, nsid, params, &resp); err != nil {
        return "", translateGetRecordError(err)
    }
    if m, ok := out.(*map[string]any); ok {
        if v, ok := resp.Value.(map[string]any); ok {
            *m = v
            return resp.CID, nil
        }
        return "", fmt.Errorf("getRecord value has unexpected type %T", resp.Value)
    }
    return "", fmt.Errorf("unsupported out type %T", out)
}
```

- [ ] **Step 2: Tree still broken; move on to 1.3**

### Task 1.3: Update production callers

**Files:**
- Modify: `appview/internal/auth/initialize_profile.go`
- Modify: `appview/internal/api/profile.go`

- [ ] **Step 1: Edit `initialize_profile.go`**

Two call sites near lines 43 and 51. Both discard the CID.

Line 43 becomes:

```go
if _, err := client.GetRecord(ctx, did, blueskyProfileNSID, profileRecordKey, &bskyRecord); err != nil {
```

Line 51 becomes:

```go
_, err := client.GetRecord(ctx, did, craftskyProfileNSID, profileRecordKey, &cskyRecord)
```

- [ ] **Step 2: Edit `profile.go`**

Single call site near line 204. Discard the CID:

```go
if _, err := pds.GetRecord(r.Context(), did, blueskyProfileNSID, profileRecordKey, &bsky); err != nil {
```

- [ ] **Step 3: `go build` — expect only test-file breakage**

Run: `cd appview && go build ./...`
Expected: PASS. Production source is now consistent.

### Task 1.4: Update test mocks

**Files:**
- Modify: `appview/internal/auth/initialize_profile_test.go`
- Modify: `appview/internal/auth/handlers_test.go`
- Modify: `appview/internal/api/profile_test.go`

- [ ] **Step 1: `initialize_profile_test.go`'s `mockPDS.GetRecord`**

Change signature to return `(string, error)`. Wherever the existing mock returned `nil`, return `"", nil`. Wherever it returned `err`, return `"", err`. For tests that care about the CID value propagation (there aren't any in this file — `InitializeProfile` doesn't use the CID), return `""` unconditionally.

Open the file, find the method `func (m *mockPDS) GetRecord(...)`, update both the signature and every return statement in its body.

- [ ] **Step 2: `handlers_test.go`'s two PDS mocks**

```go
func (noopPDSClient) GetRecord(_ context.Context, _ syntax.DID, _, _ string, _ any) (string, error) {
    return "", auth.ErrRecordNotFound
}
```

```go
func (erroringGetPDSClient) GetRecord(_ context.Context, _ syntax.DID, _, _ string, _ any) (string, error) {
    return "", errors.New("boom")
}
```

- [ ] **Step 3: `profile_test.go`'s `fakePDSForPut.GetRecord`**

Update the signature the same way. The existing body returns `nil` after populating `out` — new return is `"", nil`.

- [ ] **Step 4: `go build` and `go vet`**

Run: `cd appview && go build ./... && go vet ./...`
Expected: PASS.

- [ ] **Step 5: Run the full test suite — must pass before commit**

Run: `just test`
Expected: PASS. Every package should be green. No behavioural change yet.

- [ ] **Step 6: Format and commit**

```bash
just fmt
git add appview/internal/auth/pds_client.go \
        appview/internal/auth/pds_client_indigo.go \
        appview/internal/auth/initialize_profile.go \
        appview/internal/auth/initialize_profile_test.go \
        appview/internal/auth/handlers_test.go \
        appview/internal/api/profile.go \
        appview/internal/api/profile_test.go
git commit -m "auth: return record CID from PDSClient.GetRecord"
```

Single commit; the tree compiles between steps only via the pending test-file edits. That's acceptable for a mechanical signature sweep where the intermediate states are unreleasable and everyone lands together.

---

## Chunk 2: `auth.AnonymousPDSClient`

A read-only `PDSClient` that resolves the PDS URL from the user's DID doc and talks to it via an unauthenticated `atclient.APIClient`. The backfiller uses this; nothing else.

### Task 2.1: Test scaffolding — stubbed `identity.Directory`

**Files:**
- Create: `appview/internal/auth/anonymous_pds_client_test.go`

- [ ] **Step 1: Write the test file skeleton with a fake directory**

This test file will grow across tasks in this chunk. Start with the shared fake so subsequent tasks just add `func TestX(...)`.

```go
package auth_test

import (
    "context"
    "errors"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/bluesky-social/indigo/atproto/identity"
    "github.com/bluesky-social/indigo/atproto/syntax"

    "social.craftsky/appview/internal/auth"
)

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
```

- [ ] **Step 2: Run the scaffold test**

Run: `cd appview && go test ./internal/auth -run TestFakeDirectory_ShapesIdentity -v`
Expected: FAIL with "cannot find package" OR the symbol `auth.AnonymousPDSClient` isn't referenced yet so `go test` compiles this file fine — it should **PASS** because it doesn't touch `auth.AnonymousPDSClient`. If the shape of `identity.Identity` is different from what I've written here, the compile fails here and you need to look at the indigo source at `$GOPATH/pkg/mod/github.com/bluesky-social/indigo@.../atproto/identity/identity.go` to see how `PDSEndpoint()` is derived, then adjust the `Services` initialisation.

- [ ] **Step 3: Commit the test scaffold**

```bash
just fmt
git add appview/internal/auth/anonymous_pds_client_test.go
git commit -m "auth: test scaffolding for anonymous PDS client"
```

### Task 2.2: Happy-path `GetRecord`

**Files:**
- Create: `appview/internal/auth/anonymous_pds_client.go`
- Modify: `appview/internal/auth/anonymous_pds_client_test.go`

- [ ] **Step 1: Write the failing test**

Append to `anonymous_pds_client_test.go`:

```go
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
```

Add `"strings"` and `"time"` to the imports.

- [ ] **Step 2: Run — expect FAIL with undefined symbol**

Run: `cd appview && go test ./internal/auth -run TestAnonymousPDSClient_GetRecord_HappyPath -v`
Expected: FAIL with `undefined: auth.NewAnonymousPDSClient`.

- [ ] **Step 3: Create the production file**

```go
// appview/internal/auth/anonymous_pds_client.go
package auth

import (
    "context"
    "errors"
    "fmt"
    "net/http"
    "time"

    "github.com/bluesky-social/indigo/atproto/atclient"
    "github.com/bluesky-social/indigo/atproto/identity"
    "github.com/bluesky-social/indigo/atproto/syntax"
)

// AnonymousPDSClient is a read-only PDSClient that resolves each caller's
// PDS URL from their DID doc and talks to it via an unauthenticated
// atclient.APIClient. com.atproto.repo.getRecord is defined as public in
// the atproto lexicon; no DPoP or OAuth session is required.
//
// Used by the Bluesky backfill path in internal/index: when
// CraftskyProfile.Handle commits a new membership row we fetch the user's
// app.bsky.actor.profile record here and feed it back through
// BlueskyProfile.Handle as a synthesised tap.Event.
type AnonymousPDSClient struct {
    dir     identity.Directory
    timeout time.Duration
}

var _ PDSClient = (*AnonymousPDSClient)(nil)

// NewAnonymousPDSClient returns a client that honours the given per-request
// HTTP timeout. Tap's ACK timeout is ~10s; values in the 2–5s range keep
// backfill from wedging the pipeline on a slow PDS.
func NewAnonymousPDSClient(dir identity.Directory, timeout time.Duration) *AnonymousPDSClient {
    return &AnonymousPDSClient{dir: dir, timeout: timeout}
}

// ErrReadOnlyPDSClient is returned when a caller tries to write through
// the anonymous client. The interface satisfies PDSClient for convenience
// (single dependency type) but writes have no meaning here.
var ErrReadOnlyPDSClient = errors.New("pds: read-only client")

// GetRecord resolves the caller's PDS URL from their DID doc, then calls
// com.atproto.repo.getRecord. RecordNotFound errors are translated to
// ErrRecordNotFound via the shared translateGetRecordError helper.
func (c *AnonymousPDSClient) GetRecord(ctx context.Context, repo syntax.DID, collection, rkey string, out any) (string, error) {
    ident, err := c.dir.LookupDID(ctx, repo)
    if err != nil {
        return "", fmt.Errorf("resolve did %s: %w", repo, err)
    }
    host := ident.PDSEndpoint()
    if host == "" {
        return "", fmt.Errorf("did %s: no atproto_pds service endpoint in DID doc", repo)
    }

    api := atclient.NewAPIClient(host)
    // NewAPIClient defaults Client.Client to http.DefaultClient. Replace
    // with our own *http.Client so we can pin a short per-request timeout
    // without mutating global state.
    api.Client = &http.Client{Timeout: c.timeout}

    nsid, err := syntax.ParseNSID("com.atproto.repo.getRecord")
    if err != nil {
        return "", fmt.Errorf("parse nsid: %w", err)
    }
    var resp struct {
        URI   string `json:"uri"`
        CID   string `json:"cid"`
        Value any    `json:"value"`
    }
    params := map[string]any{
        "repo":       repo.String(),
        "collection": collection,
        "rkey":       rkey,
    }
    if err := api.Get(ctx, nsid, params, &resp); err != nil {
        return "", translateGetRecordError(err)
    }
    if m, ok := out.(*map[string]any); ok {
        if v, ok := resp.Value.(map[string]any); ok {
            *m = v
            return resp.CID, nil
        }
        return "", fmt.Errorf("getRecord value has unexpected type %T", resp.Value)
    }
    return "", fmt.Errorf("unsupported out type %T", out)
}

// PutRecord is not supported by the anonymous client.
func (c *AnonymousPDSClient) PutRecord(_ context.Context, _ syntax.DID, _, _ string, _ any) error {
    return ErrReadOnlyPDSClient
}
```

**Note on `api.Client`:** `atclient.APIClient` has a field called `Client` of type `*http.Client`. Re-assigning it replaces the default. Verify the field name is `Client` (not `HTTP` or similar) by looking at `$GOPATH/pkg/mod/github.com/bluesky-social/indigo@.../atproto/atclient/apiclient.go` around line 41. If indigo renames this field in a future version, the line `api.Client = &http.Client{...}` is the place to adjust.

- [ ] **Step 4: Run the test**

Run: `cd appview && go test ./internal/auth -run TestAnonymousPDSClient_GetRecord_HappyPath -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
just fmt
git add appview/internal/auth/anonymous_pds_client.go appview/internal/auth/anonymous_pds_client_test.go
git commit -m "auth: add AnonymousPDSClient for unauthenticated PDS reads"
```

### Task 2.3: RecordNotFound translation

**Files:**
- Modify: `appview/internal/auth/anonymous_pds_client_test.go`

- [ ] **Step 1: Write the failing test**

Append:

```go
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
```

- [ ] **Step 2: Run — expect PASS already**

Run: `cd appview && go test ./internal/auth -run TestAnonymousPDSClient_GetRecord_RecordNotFound -v`
Expected: PASS — `translateGetRecordError` already handles this case; the test is a regression guard, not a driver.

If it fails, it means `atclient.APIClient.Get` doesn't wrap a 400 JSON error body into `*atclient.APIError`. Check that assumption at `apiclient.go`. This is why we run the test: to confirm behaviour, not just to drive code.

- [ ] **Step 3: Commit**

```bash
just fmt
git add appview/internal/auth/anonymous_pds_client_test.go
git commit -m "auth: regression test for anonymous RecordNotFound translation"
```

### Task 2.4: Missing PDS endpoint in DID doc

**Files:**
- Modify: `appview/internal/auth/anonymous_pds_client_test.go`

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run — expect PASS**

Run: `cd appview && go test ./internal/auth -run TestAnonymousPDSClient_GetRecord_NoPDSEndpoint -v`
Expected: PASS. Already implemented. Regression guard.

- [ ] **Step 3: Commit**

```bash
git add appview/internal/auth/anonymous_pds_client_test.go
git commit -m "auth: regression test for missing PDS endpoint in DID doc"
```

### Task 2.5: Directory lookup failure

**Files:**
- Modify: `appview/internal/auth/anonymous_pds_client_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestAnonymousPDSClient_GetRecord_DirectoryError(t *testing.T) {
    t.Parallel()
    dir := &fakeDirectory{did: syntax.DID("did:plc:abc"), err: errors.New("dns failure")}
    cli := auth.NewAnonymousPDSClient(dir, 2*time.Second)

    var out map[string]any
    _, err := cli.GetRecord(context.Background(),
        syntax.DID("did:plc:abc"), "app.bsky.actor.profile", "self", &out)
    if err == nil {
        t.Fatal("want error when directory lookup fails; got nil")
    }
    if !strings.Contains(err.Error(), "dns failure") {
        t.Errorf("err = %v; want wrapped underlying error", err)
    }
}
```

- [ ] **Step 2: Run — expect PASS**

Run: `cd appview && go test ./internal/auth -run TestAnonymousPDSClient_GetRecord_DirectoryError -v`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add appview/internal/auth/anonymous_pds_client_test.go
git commit -m "auth: regression test for anonymous directory lookup failure"
```

### Task 2.6: PutRecord is read-only

**Files:**
- Modify: `appview/internal/auth/anonymous_pds_client_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestAnonymousPDSClient_PutRecord_ReadOnly(t *testing.T) {
    t.Parallel()
    cli := auth.NewAnonymousPDSClient(&fakeDirectory{}, time.Second)
    err := cli.PutRecord(context.Background(),
        syntax.DID("did:plc:x"), "any.nsid", "self", map[string]any{})
    if !errors.Is(err, auth.ErrReadOnlyPDSClient) {
        t.Errorf("want ErrReadOnlyPDSClient; got %v", err)
    }
}
```

- [ ] **Step 2: Run — expect PASS**

Run: `cd appview && go test ./internal/auth -run TestAnonymousPDSClient_PutRecord_ReadOnly -v`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add appview/internal/auth/anonymous_pds_client_test.go
git commit -m "auth: enforce read-only on anonymous PDS client"
```

### Task 2.7: Full suite run

- [ ] **Step 1: Run the whole test binary**

Run: `just test`
Expected: PASS. If any unrelated test regressed, it's likely a leaked signature change from Chunk 1; fix before proceeding.

---

## Chunk 3: `index.BlueskyBackfiller`

The backfiller interface plus concrete impl that (1) fetches `app.bsky.actor.profile` via the anonymous client, (2) synthesises a `tap.Event`, (3) dispatches to `BlueskyProfile.Handle`. Lives in `internal/index`. Intra-package, no circular imports.

### Task 3.1: Interface file

**Files:**
- Create: `appview/internal/index/bluesky_backfiller.go`
- Create: `appview/internal/index/bluesky_backfiller_test.go`

- [ ] **Step 1: Write the failing test for the interface shape**

Create `appview/internal/index/bluesky_backfiller_test.go`:

```go
package index_test

import (
    "context"
    "testing"

    "github.com/bluesky-social/indigo/atproto/syntax"

    "social.craftsky/appview/internal/index"
)

// fakeBackfiller is used by CraftskyProfile tests in Chunk 4 but we also
// verify here that it satisfies the exported interface.
type fakeBackfiller struct {
    calls []syntax.DID
    err   error
}

func (f *fakeBackfiller) Backfill(_ context.Context, did syntax.DID) error {
    f.calls = append(f.calls, did)
    return f.err
}

func TestBlueskyBackfiller_InterfaceShape(t *testing.T) {
    var _ index.BlueskyBackfiller = (*fakeBackfiller)(nil)
}
```

- [ ] **Step 2: Run — expect FAIL with undefined symbol**

Run: `cd appview && go test ./internal/index -run TestBlueskyBackfiller_InterfaceShape -v`
Expected: FAIL — `undefined: index.BlueskyBackfiller`.

- [ ] **Step 3: Create the interface**

Create `appview/internal/index/bluesky_backfiller.go`:

```go
// appview/internal/index/bluesky_backfiller.go
package index

import (
    "context"

    "github.com/bluesky-social/indigo/atproto/syntax"
)

// BlueskyBackfiller eagerly populates bluesky_profiles for a newly-
// onboarded Craftsky member by fetching their app.bsky.actor.profile
// record from their PDS and feeding it back through BlueskyProfile.Handle.
//
// This exists to sidestep a race during Tap backfill: the MST emits
// records in key-sorted order, so app.bsky.actor.profile arrives before
// social.craftsky.actor.profile and is dropped by the membership gate.
// CraftskyProfile.Handle invokes Backfill only when it commits a
// genuinely new membership row (see craftsky_profile.go's xmax check).
//
// Implementations must tolerate a missing Bluesky record (many users
// won't have one) and return nil in that case.
type BlueskyBackfiller interface {
    Backfill(ctx context.Context, did syntax.DID) error
}
```

- [ ] **Step 4: Run the test**

Run: `cd appview && go test ./internal/index -run TestBlueskyBackfiller_InterfaceShape -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
just fmt
git add appview/internal/index/bluesky_backfiller.go appview/internal/index/bluesky_backfiller_test.go
git commit -m "index: BlueskyBackfiller interface"
```

### Task 3.2: Concrete impl — happy path

**Files:**
- Modify: `appview/internal/index/bluesky_backfiller.go`
- Modify: `appview/internal/index/bluesky_backfiller_test.go`

- [ ] **Step 1: Write the failing test**

The backfiller needs two dependencies: a `PDSClient` (provides `GetRecord`) and something to dispatch `tap.Event` into — we call `*BlueskyProfile` directly. Since `BlueskyProfile.Handle` needs the membership gate to pass, this test seeds a craftsky_profiles row so the write completes end-to-end.

Append to `bluesky_backfiller_test.go`:

```go
import (
    "context"
    "errors"
    "testing"

    "github.com/bluesky-social/indigo/atproto/syntax"

    "social.craftsky/appview/internal/auth"
    "social.craftsky/appview/internal/index"
    "social.craftsky/appview/internal/testdb"
)

// fakeAnonPDS implements auth.PDSClient for backfiller tests. GetRecord
// returns the configured value+cid; PutRecord is never used.
type fakeAnonPDS struct {
    cid   string
    value map[string]any
    err   error
    calls int
}

func (f *fakeAnonPDS) GetRecord(_ context.Context, _ syntax.DID, _, _ string, out any) (string, error) {
    f.calls++
    if f.err != nil {
        return "", f.err
    }
    if m, ok := out.(*map[string]any); ok {
        *m = f.value
    }
    return f.cid, nil
}

func (f *fakeAnonPDS) PutRecord(_ context.Context, _ syntax.DID, _, _ string, _ any) error {
    return errors.New("not used")
}

func TestBlueskyBackfiller_Backfill_RecordPresent_WritesBlueskyRow(t *testing.T) {
    t.Parallel()
    pool := testdb.WithSchema(t, craftskyProfilesDDL)

    // Seed membership so BlueskyProfile.Handle's gate passes.
    if _, err := pool.Exec(context.Background(),
        `INSERT INTO craftsky_profiles (did, record_cid) VALUES ($1, $2)`,
        "did:plc:abc", "cidcsky"); err != nil {
        t.Fatal(err)
    }

    pds := &fakeAnonPDS{
        cid:   "bafbluesky",
        value: map[string]any{"displayName": "alice"},
    }
    bsky := index.NewBlueskyProfile(pool)
    bf := index.NewBlueskyBackfiller(pds, bsky)

    if err := bf.Backfill(context.Background(), syntax.DID("did:plc:abc")); err != nil {
        t.Fatalf("Backfill: %v", err)
    }
    if pds.calls != 1 {
        t.Errorf("PDS GetRecord called %d times; want 1", pds.calls)
    }

    var displayName, recordCID string
    if err := pool.QueryRow(context.Background(),
        `SELECT display_name, record_cid FROM bluesky_profiles WHERE did = $1`,
        "did:plc:abc").Scan(&displayName, &recordCID); err != nil {
        t.Fatalf("select: %v", err)
    }
    if displayName != "alice" {
        t.Errorf("display_name = %q", displayName)
    }
    if recordCID != "bafbluesky" {
        t.Errorf("record_cid = %q", recordCID)
    }
}
```

Consolidate imports at the top of the file (the skeleton Task 3.1 added minimal imports).

- [ ] **Step 2: Run — expect FAIL with undefined symbol**

Run: `cd appview && go test ./internal/index -run TestBlueskyBackfiller_Backfill_RecordPresent -v`
Expected: FAIL — `undefined: index.NewBlueskyBackfiller`.

- [ ] **Step 3: Implement**

Append to `bluesky_backfiller.go`:

```go
import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "log/slog"

    "github.com/bluesky-social/indigo/atproto/syntax"

    "social.craftsky/appview/internal/auth"
    "social.craftsky/appview/internal/tap"
)

// blueskyBackfiller implements BlueskyBackfiller.
type blueskyBackfiller struct {
    reader  auth.PDSClient
    indexer *BlueskyProfile
}

// NewBlueskyBackfiller wires an anonymous PDS reader to the Bluesky
// indexer. Callers pass their existing *BlueskyProfile; the backfiller
// dispatches synthesised events through it so parse, gate, and upsert
// logic stay in one place.
func NewBlueskyBackfiller(reader auth.PDSClient, indexer *BlueskyProfile) BlueskyBackfiller {
    return &blueskyBackfiller{reader: reader, indexer: indexer}
}

// Backfill fetches the user's app.bsky.actor.profile/self record from
// their PDS and feeds it to BlueskyProfile.Handle as a synthesised
// tap.Event{Action:"create"}. A missing Bluesky record (ErrRecordNotFound)
// is a no-op and returns nil — many users don't have one.
func (b *blueskyBackfiller) Backfill(ctx context.Context, did syntax.DID) error {
    var rec map[string]any
    cid, err := b.reader.GetRecord(ctx, did, "app.bsky.actor.profile", "self", &rec)
    if errors.Is(err, auth.ErrRecordNotFound) {
        return nil
    }
    if err != nil {
        return fmt.Errorf("backfill fetch %s: %w", did, err)
    }
    raw, err := json.Marshal(rec)
    if err != nil {
        return fmt.Errorf("backfill marshal %s: %w", did, err)
    }
    ev := tap.Event{
        URI:        "at://" + did.String() + "/app.bsky.actor.profile/self",
        CID:        cid,
        DID:        did.String(),
        Rkey:       "self",
        Collection: "app.bsky.actor.profile",
        Action:     "create",
        Record:     raw,
    }
    return b.indexer.Handle(ctx, ev)
}

// compile-time assertion: slog is imported for the test symmetry; if the
// linter flags it later, we move the import to craftsky_profile.go where
// the logger actually lives. Keeping it here as a guard during review.
var _ = slog.Default // suppress unused-import; remove in a follow-up if lint complains.
```

**Note:** the `slog` import here is defensive — if `go vet` flags it, delete both the import and the `var _ = slog.Default` line. We only need it in this package because Chunk 4 also uses slog, and some packages conflate imports awkwardly. If lint is clean, leave it.

Actually — simpler: drop the slog import entirely from this file. The logger lives on `CraftskyProfile`, not on the backfiller. Remove the `slog` import and the `var _` line before committing.

- [ ] **Step 4: Run the test**

Run: `just test` (the test hits Postgres via `testdb.WithSchema`; the scoped `just test` command sets `TEST_DATABASE_URL`).
Expected: PASS.

If Postgres isn't available (`just dev-d` not running), the test will skip with "TEST_DATABASE_URL and DATABASE_URL both unset" — start the compose stack first.

- [ ] **Step 5: Commit**

```bash
just fmt
git add appview/internal/index/bluesky_backfiller.go appview/internal/index/bluesky_backfiller_test.go
git commit -m "index: blueskyBackfiller happy path"
```

### Task 3.3: Record-not-found is a no-op

**Files:**
- Modify: `appview/internal/index/bluesky_backfiller_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestBlueskyBackfiller_Backfill_RecordNotFound_IsNoOp(t *testing.T) {
    t.Parallel()
    pool := testdb.WithSchema(t, craftskyProfilesDDL)
    if _, err := pool.Exec(context.Background(),
        `INSERT INTO craftsky_profiles (did, record_cid) VALUES ($1, $2)`,
        "did:plc:none", "cidcsky"); err != nil {
        t.Fatal(err)
    }
    pds := &fakeAnonPDS{err: auth.ErrRecordNotFound}
    bf := index.NewBlueskyBackfiller(pds, index.NewBlueskyProfile(pool))

    if err := bf.Backfill(context.Background(), syntax.DID("did:plc:none")); err != nil {
        t.Errorf("want nil for RecordNotFound; got %v", err)
    }
    var count int
    _ = pool.QueryRow(context.Background(),
        `SELECT count(*) FROM bluesky_profiles WHERE did = $1`,
        "did:plc:none").Scan(&count)
    if count != 0 {
        t.Errorf("bluesky_profiles count = %d; want 0", count)
    }
}
```

- [ ] **Step 2: Run — expect PASS already**

Run: `just test`
Expected: PASS. Regression guard.

- [ ] **Step 3: Commit**

```bash
git add appview/internal/index/bluesky_backfiller_test.go
git commit -m "index: blueskyBackfiller regression test for record-not-found"
```

### Task 3.4: PDS error propagates

**Files:**
- Modify: `appview/internal/index/bluesky_backfiller_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestBlueskyBackfiller_Backfill_PDSError_Propagates(t *testing.T) {
    t.Parallel()
    pool := testdb.WithSchema(t, craftskyProfilesDDL)
    if _, err := pool.Exec(context.Background(),
        `INSERT INTO craftsky_profiles (did, record_cid) VALUES ($1, $2)`,
        "did:plc:err", "cidcsky"); err != nil {
        t.Fatal(err)
    }
    boom := errors.New("pds on fire")
    pds := &fakeAnonPDS{err: boom}
    bf := index.NewBlueskyBackfiller(pds, index.NewBlueskyProfile(pool))

    err := bf.Backfill(context.Background(), syntax.DID("did:plc:err"))
    if !errors.Is(err, boom) {
        t.Errorf("want wrapped %v; got %v", boom, err)
    }
}
```

- [ ] **Step 2: Run — expect PASS**

Run: `just test`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add appview/internal/index/bluesky_backfiller_test.go
git commit -m "index: blueskyBackfiller regression test for PDS error"
```

---

## Chunk 4: `CraftskyProfile` wiring + xmax detection

Change `NewCraftskyProfile` to accept a `BlueskyBackfiller` and a logger; change the upsert SQL to `RETURNING xmax = 0 AS created`; call `Backfill` only on genuine new-row inserts. Update existing tests to pass a no-op fake. Add new tests that verify the new-row trigger and the replay no-op.

### Task 4.1: Update existing tests to pass a fake backfiller (compile-driver)

The simplest way to drive the constructor change is to update every existing `index.NewCraftskyProfile(pool)` call to the new signature first, via a no-op fake. This makes the tree re-compile before we introduce behavioural assertions.

**Files:**
- Modify: `appview/internal/index/craftsky_profile_test.go`

- [ ] **Step 1: Add a no-op backfiller + helper to the test file**

At the top of `craftsky_profile_test.go` (below the existing DDL constant), add:

```go
import (
    // existing imports, plus:
    "log/slog"

    "github.com/bluesky-social/indigo/atproto/syntax"
)

// noopBackfiller is the default backfiller for existing tests that
// predate the backfill path. Satisfies index.BlueskyBackfiller.
type noopBackfiller struct{}

func (noopBackfiller) Backfill(context.Context, syntax.DID) error { return nil }

// testLogger returns a logger that discards output. Equivalent patterns
// live elsewhere in the repo — inline here to avoid a new exported helper.
func testLogger() *slog.Logger {
    return slog.New(slog.NewTextHandler(io.Discard, nil))
}
```

Add `"io"` to the imports.

- [ ] **Step 2: Update every `index.NewCraftskyProfile(pool)` call in this file**

Each becomes:

```go
idx := index.NewCraftskyProfile(pool, noopBackfiller{}, testLogger())
```

Search for `NewCraftskyProfile` and update every occurrence in this file.

- [ ] **Step 3: Run — expect FAIL at constructor (wrong arg count)**

Run: `cd appview && go test ./internal/index -count=1 -run TestCraftskyProfile -v`
Expected: FAIL. Either "too many arguments in call to NewCraftskyProfile" if we haven't updated the production signature yet (we haven't), or every test body fails.

This is the trigger for the production change in Task 4.2.

### Task 4.2: Update `NewCraftskyProfile` signature

**Files:**
- Modify: `appview/internal/index/craftsky_profile.go`

- [ ] **Step 1: Change the struct + constructor**

Replace the struct + constructor:

```go
type CraftskyProfile struct {
    pool       *pgxpool.Pool
    backfiller BlueskyBackfiller
    logger     *slog.Logger
}

var _ Indexer = (*CraftskyProfile)(nil)

// NewCraftskyProfile builds an indexer. The backfiller is invoked when
// Handle commits a genuinely new membership row; it may be a no-op in
// tests. A nil logger defaults to slog.Default() to keep call sites that
// don't care about structured logging concise.
func NewCraftskyProfile(pool *pgxpool.Pool, backfiller BlueskyBackfiller, logger *slog.Logger) *CraftskyProfile {
    if logger == nil {
        logger = slog.Default()
    }
    return &CraftskyProfile{pool: pool, backfiller: backfiller, logger: logger}
}
```

Add `"log/slog"` to imports.

- [ ] **Step 2: Run test suite to confirm compile**

Run: `cd appview && go build ./...`
Expected: PASS. Tests haven't been exercised yet; the shape is consistent.

- [ ] **Step 3: Full test — expect pre-existing tests PASS (no behavioural change yet)**

Run: `just test`
Expected: PASS. The backfiller is stored but never called. Existing assertions still hold because the upsert is unchanged.

- [ ] **Step 4: Commit**

```bash
just fmt
git add appview/internal/index/craftsky_profile.go appview/internal/index/craftsky_profile_test.go
git commit -m "index: thread BlueskyBackfiller into CraftskyProfile constructor"
```

### Task 4.3: Replace the upsert with `RETURNING xmax = 0` and call the backfiller on new-row

**Files:**
- Modify: `appview/internal/index/craftsky_profile.go`
- Modify: `appview/internal/index/craftsky_profile_test.go`

- [ ] **Step 1: Write the failing test**

Append to `craftsky_profile_test.go`:

```go
// spyBackfiller records every call so tests can assert arity and DID.
type spyBackfiller struct {
    calls []string
    err   error
}

func (s *spyBackfiller) Backfill(_ context.Context, did syntax.DID) error {
    s.calls = append(s.calls, did.String())
    return s.err
}

func TestCraftskyProfile_Handle_NewRow_CallsBackfill(t *testing.T) {
    t.Parallel()
    pool := testdb.WithSchema(t, craftskyProfilesDDL)
    spy := &spyBackfiller{}
    idx := index.NewCraftskyProfile(pool, spy, testLogger())

    ev := tap.Event{
        URI:        "at://did:plc:new/social.craftsky.actor.profile/self",
        CID:        "c1",
        DID:        "did:plc:new",
        Rkey:       "self",
        Collection: "social.craftsky.actor.profile",
        Action:     "create",
        Record:     json.RawMessage(`{"crafts":["sewing"]}`),
    }
    if err := idx.Handle(context.Background(), ev); err != nil {
        t.Fatalf("Handle: %v", err)
    }
    if len(spy.calls) != 1 || spy.calls[0] != "did:plc:new" {
        t.Errorf("backfill calls = %v; want [did:plc:new]", spy.calls)
    }
}
```

- [ ] **Step 2: Run — expect FAIL because Handle doesn't invoke Backfill yet**

Run: `cd appview && go test ./internal/index -run TestCraftskyProfile_Handle_NewRow_CallsBackfill -v`
Expected: FAIL — `backfill calls = []; want [did:plc:new]`.

- [ ] **Step 3: Implement the xmax check**

In `craftsky_profile.go`, replace the current upsert block inside the `case "create", "update":` branch:

```go
case "create", "update":
    var rec craftskyProfileRecord
    if err := json.Unmarshal(ev.Record, &rec); err != nil {
        return fmt.Errorf("unmarshal %s: %w", ev.URI, err)
    }
    if rec.Crafts == nil {
        rec.Crafts = []string{}
    }
    const q = `
        INSERT INTO craftsky_profiles (did, crafts, record_cid)
        VALUES ($1, $2, $3)
        ON CONFLICT (did) DO UPDATE SET
            crafts = EXCLUDED.crafts,
            record_cid = EXCLUDED.record_cid,
            indexed_at = now()
        WHERE craftsky_profiles.record_cid IS DISTINCT FROM EXCLUDED.record_cid
        RETURNING xmax = 0 AS created
    `
    var created bool
    err := c.pool.QueryRow(ctx, q, ev.DID, rec.Crafts, ev.CID).Scan(&created)
    switch {
    case errors.Is(err, pgx.ErrNoRows):
        // Replay of an existing row: ON CONFLICT ... WHERE IS DISTINCT FROM
        // filtered the update, so no row came back. Not an error.
        return nil
    case err != nil:
        return fmt.Errorf("upsert %s: %w", ev.URI, err)
    }
    if !created {
        // UPDATE branch — membership row already existed; no backfill needed.
        return nil
    }
    // Genuine new-row INSERT. Trigger one-shot Bluesky backfill;
    // errors are logged and swallowed so the craftsky event is still
    // acked by Tap.
    did, parseErr := syntax.ParseDID(ev.DID)
    if parseErr != nil {
        c.logger.Warn("craftsky profile: cannot parse DID for backfill",
            slog.String("did", ev.DID), slog.String("err", parseErr.Error()))
        return nil
    }
    if bfErr := c.backfiller.Backfill(ctx, did); bfErr != nil {
        c.logger.Warn("craftsky profile: bluesky backfill failed",
            slog.String("did", ev.DID), slog.String("err", bfErr.Error()))
    }
    return nil
```

Add imports: `"errors"`, `"github.com/bluesky-social/indigo/atproto/syntax"`, `"github.com/jackc/pgx/v5"`. (`pgxpool` is already imported; `pgx` is a separate import path.)

- [ ] **Step 4: Run the failing test — expect PASS**

Run: `cd appview && go test ./internal/index -run TestCraftskyProfile_Handle_NewRow_CallsBackfill -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
just fmt
git add appview/internal/index/craftsky_profile.go appview/internal/index/craftsky_profile_test.go
git commit -m "index: trigger bluesky backfill on new craftsky membership row"
```

### Task 4.4: Replay does not re-fetch

**Files:**
- Modify: `appview/internal/index/craftsky_profile_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestCraftskyProfile_Handle_Replay_SkipsBackfill(t *testing.T) {
    t.Parallel()
    pool := testdb.WithSchema(t, craftskyProfilesDDL)
    spy := &spyBackfiller{}
    idx := index.NewCraftskyProfile(pool, spy, testLogger())

    ev := tap.Event{
        URI:        "at://did:plc:re/social.craftsky.actor.profile/self",
        CID:        "c1",
        DID:        "did:plc:re",
        Rkey:       "self",
        Collection: "social.craftsky.actor.profile",
        Action:     "create",
        Record:     json.RawMessage(`{"crafts":["sewing"]}`),
    }
    // First delivery — backfill fires.
    if err := idx.Handle(context.Background(), ev); err != nil {
        t.Fatal(err)
    }
    // Second delivery, same CID — replay path. Backfill must NOT fire.
    if err := idx.Handle(context.Background(), ev); err != nil {
        t.Fatal(err)
    }
    if len(spy.calls) != 1 {
        t.Errorf("backfill calls = %d; want 1 (replay must not re-fire)", len(spy.calls))
    }
}
```

- [ ] **Step 2: Run — expect PASS**

Run: `cd appview && go test ./internal/index -run TestCraftskyProfile_Handle_Replay_SkipsBackfill -v`
Expected: PASS. The `pgx.ErrNoRows` branch covers this.

If it fails with `len = 2`, the xmax check is wrong — most likely the `RETURNING xmax = 0 AS created` branch is falling through to the backfill call even when `created == false`. Read the Task 4.3 diff carefully.

- [ ] **Step 3: Commit**

```bash
git add appview/internal/index/craftsky_profile_test.go
git commit -m "index: regression test for craftsky replay skipping backfill"
```

### Task 4.5: Update does not re-fetch

**Files:**
- Modify: `appview/internal/index/craftsky_profile_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestCraftskyProfile_Handle_Update_SkipsBackfill(t *testing.T) {
    t.Parallel()
    pool := testdb.WithSchema(t, craftskyProfilesDDL)
    spy := &spyBackfiller{}
    idx := index.NewCraftskyProfile(pool, spy, testLogger())
    ctx := context.Background()

    create := tap.Event{
        URI: "at://did:plc:up/social.craftsky.actor.profile/self", CID: "c1",
        DID: "did:plc:up", Rkey: "self",
        Collection: "social.craftsky.actor.profile", Action: "create",
        Record: json.RawMessage(`{"crafts":["a"]}`),
    }
    update := create
    update.CID = "c2"
    update.Action = "update"
    update.Record = json.RawMessage(`{"crafts":["a","b"]}`)

    if err := idx.Handle(ctx, create); err != nil {
        t.Fatal(err)
    }
    if err := idx.Handle(ctx, update); err != nil {
        t.Fatal(err)
    }
    if len(spy.calls) != 1 {
        t.Errorf("backfill calls = %d; want 1 (update must not re-fire)", len(spy.calls))
    }
}
```

- [ ] **Step 2: Run — expect PASS**

Run: `cd appview && go test ./internal/index -run TestCraftskyProfile_Handle_Update_SkipsBackfill -v`
Expected: PASS. The `if !created { return nil }` branch covers this.

- [ ] **Step 3: Commit**

```bash
git add appview/internal/index/craftsky_profile_test.go
git commit -m "index: regression test for craftsky update skipping backfill"
```

### Task 4.6: Backfill error is logged, not returned

**Files:**
- Modify: `appview/internal/index/craftsky_profile_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestCraftskyProfile_Handle_BackfillError_DoesNotFail(t *testing.T) {
    t.Parallel()
    pool := testdb.WithSchema(t, craftskyProfilesDDL)
    spy := &spyBackfiller{err: errors.New("pds fire")}
    idx := index.NewCraftskyProfile(pool, spy, testLogger())

    ev := tap.Event{
        URI:        "at://did:plc:bf/social.craftsky.actor.profile/self",
        CID:        "c1",
        DID:        "did:plc:bf",
        Rkey:       "self",
        Collection: "social.craftsky.actor.profile",
        Action:     "create",
        Record:     json.RawMessage(`{"crafts":["sewing"]}`),
    }
    if err := idx.Handle(context.Background(), ev); err != nil {
        t.Fatalf("Handle returned %v; want nil despite backfill error", err)
    }

    // Craftsky row must still be committed.
    var count int
    _ = pool.QueryRow(context.Background(),
        `SELECT count(*) FROM craftsky_profiles WHERE did = $1`, ev.DID).Scan(&count)
    if count != 1 {
        t.Errorf("craftsky_profiles count = %d; want 1", count)
    }
}
```

Add `"errors"` to the imports of the test file if not already present.

- [ ] **Step 2: Run — expect PASS**

Run: `cd appview && go test ./internal/index -run TestCraftskyProfile_Handle_BackfillError_DoesNotFail -v`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add appview/internal/index/craftsky_profile_test.go
git commit -m "index: regression test for backfill errors not failing ack"
```

### Task 4.7: End-to-end — both tables populated from one Handle call

**Files:**
- Modify: `appview/internal/index/bluesky_backfiller_test.go`

This is the integration test that proves the full chain works against real Postgres: one `CraftskyProfile.Handle(create)` populates both tables via a real (wired-up) `blueskyBackfiller`.

- [ ] **Step 1: Write the failing test**

Append to `bluesky_backfiller_test.go` (same package as the other backfiller tests):

```go
func TestCraftskyProfile_Handle_NewRow_BackfillsBluesky(t *testing.T) {
    t.Parallel()
    pool := testdb.WithSchema(t, craftskyProfilesDDL)
    pds := &fakeAnonPDS{
        cid:   "bafbluesky",
        value: map[string]any{"displayName": "alice"},
    }
    bsky := index.NewBlueskyProfile(pool)
    bf := index.NewBlueskyBackfiller(pds, bsky)
    idx := index.NewCraftskyProfile(pool, bf, testLogger())

    ev := tap.Event{
        URI:        "at://did:plc:e2e/social.craftsky.actor.profile/self",
        CID:        "ccsky",
        DID:        "did:plc:e2e",
        Rkey:       "self",
        Collection: "social.craftsky.actor.profile",
        Action:     "create",
        Record:     json.RawMessage(`{"crafts":["sewing"]}`),
    }
    if err := idx.Handle(context.Background(), ev); err != nil {
        t.Fatalf("Handle: %v", err)
    }

    // Both tables populated after a single handle call.
    var craftskyCount int
    _ = pool.QueryRow(context.Background(),
        `SELECT count(*) FROM craftsky_profiles WHERE did = $1`, ev.DID).Scan(&craftskyCount)
    if craftskyCount != 1 {
        t.Errorf("craftsky_profiles count = %d; want 1", craftskyCount)
    }

    var displayName string
    if err := pool.QueryRow(context.Background(),
        `SELECT display_name FROM bluesky_profiles WHERE did = $1`, ev.DID).
        Scan(&displayName); err != nil {
        t.Fatalf("select bluesky: %v", err)
    }
    if displayName != "alice" {
        t.Errorf("display_name = %q; want alice", displayName)
    }
}
```

This test will need `testLogger` and `noopBackfiller` OR to import them from the sibling test file. Go test files in the same package (`index_test`) share exported helpers — `testLogger` is defined in `craftsky_profile_test.go` in the same `index_test` package, so it's visible here. If the compiler disagrees, either (a) lowercase it and accept the sibling-file sharing, or (b) move `testLogger` and `noopBackfiller` into a shared `testing_helpers_test.go` file in the same package.

- [ ] **Step 2: Run the test**

Run: `cd appview && go test ./internal/index -run TestCraftskyProfile_Handle_NewRow_BackfillsBluesky -v`
Expected: PASS — the chain works end-to-end.

- [ ] **Step 3: Commit**

```bash
just fmt
git add appview/internal/index/bluesky_backfiller_test.go
git commit -m "index: e2e test for craftsky+bluesky backfill chain"
```

### Task 4.8: Full test sweep

- [ ] **Step 1: Run every test**

Run: `just test`
Expected: PASS. No regressions.

If the `TestCraftskyProfile_DeleteRemovesBothRows` test from the original suite fails, it's likely because the delete branch in `Handle` wasn't modified — and shouldn't have been. Verify the spec §3.1–3.3 describes no change to delete semantics and investigate any test that changes behaviour here.

---

## Chunk 5: Wire it into `deps.go`

The last mile. `NewAnonymousPDSClient` is instantiated; the backfiller is instantiated; both get threaded into `NewCraftskyProfile`.

### Task 5.1: Compile-driver

**Files:**
- Modify: `appview/internal/app/deps.go`

- [ ] **Step 1: Run the tree to see what breaks**

Run: `cd appview && go build ./...`
Expected: FAIL — `index.NewCraftskyProfile` takes new arguments. Compile error at `deps.go:118`.

- [ ] **Step 2: Update the wiring**

Replace the block around `dispatcher.Register` (around line 117) with:

```go
dispatcher := index.NewDispatcher(index.NotImplemented{})
anonPDS := auth.NewAnonymousPDSClient(identityDir, 5*time.Second)
blueskyIdx := index.NewBlueskyProfile(pool)
backfiller := index.NewBlueskyBackfiller(anonPDS, blueskyIdx)
dispatcher.Register("social.craftsky.actor.profile",
    index.NewCraftskyProfile(pool, backfiller, logger))
dispatcher.Register("app.bsky.actor.profile", blueskyIdx)
```

Add `"time"` to imports if not already present. (Given `OAuthSessionExpiry` et al. exist on Config, `"time"` is almost certainly already imported; verify.)

- [ ] **Step 3: Build**

Run: `cd appview && go build ./...`
Expected: PASS.

- [ ] **Step 4: Full test sweep**

Run: `just test`
Expected: PASS. This is the first end-to-end validation that the wired-up chain still passes every test.

- [ ] **Step 5: Commit**

```bash
just fmt
git add appview/internal/app/deps.go
git commit -m "app: wire anonymous PDS client + bluesky backfiller into deps"
```

### Task 5.2: Manual smoke test

**Files:** none

- [ ] **Step 1: Start the stack**

Run: `just dev-d`
Expected: containers come up. `docker compose ps` should show postgres, tap, appview all healthy.

- [ ] **Step 2: Hit the OAuth flow end-to-end**

Open the app and complete the OAuth login flow with a test Bluesky account that already has an `app.bsky.actor.profile` record. (If you cleared out the dev DB, the user is seen as new.)

- [ ] **Step 3: Verify both rows are populated**

```bash
docker compose exec postgres psql -U craftsky -d craftsky_dev -c \
  "SELECT did FROM craftsky_profiles;"
docker compose exec postgres psql -U craftsky -d craftsky_dev -c \
  "SELECT did, display_name FROM bluesky_profiles;"
```

Expected: both tables contain a row for your test DID, with `display_name` matching your Bluesky handle's display name.

If `bluesky_profiles` is empty:
- Check the appview logs for warnings containing "bluesky backfill failed".
- If no such log, the backfill path wasn't invoked — check whether the craftsky row was committed (a `pgx.ErrNoRows` branch would cause a silent skip; a replay would do the same).
- If there's a log, the PDS call failed. Read the error.

- [ ] **Step 4: No commit**

This step validates the work; nothing to commit.

---

## Finalisation

### Task F.1: Rebase into a clean history (optional)

If the chunks produced many small commits and you want a tidier series before merging, use `git rebase -i origin/main` (NOT `--no-edit`) to squash mechanical compile-fix commits into their parents. Keep each behavioural change in its own commit. See AGENTS.md §Git for the preferred commit-message shape.

### Task F.2: Update CHANGELOG if the project has one

**Files:**
- Modify: `CHANGELOG.md` (if present)

- [ ] Check for `CHANGELOG.md` at repo root. If present, add a bullet under the next unreleased section:
  "Fix Bluesky profile backfill race on Craftsky onboarding. Newly-onboarded members now have their `app.bsky.actor.profile` populated synchronously instead of racing against Tap's MST-ordered backfill."

If the repo has no changelog, skip this task.

---

## Out-of-scope follow-ups (reference only, do not implement)

These are called out in spec §9. Do **not** build them in this plan.

- A background reconciliation job that scans for members with no `bluesky_profiles` row and triggers `Backfill`. Recovers currently-stuck users without making them edit their profile.
- Explicit `Tap /repos/add` and `/repos/remove` integration. Needed when account deletion lands.
- Caching/pooling of the `*atclient.APIClient` per host.
- Stricter lexicon validation.
