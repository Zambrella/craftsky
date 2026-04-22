# API Wire Alignment Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the last client/server contract gaps so the Flutter OAuth sign-in flow works end-to-end against `just dev`.

**Architecture:** The spec at [docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md](../specs/2026-04-22-api-wire-alignment-design.md) calls for four changes: camelCase JSON everywhere, `/v1/whoami` returning `{did, handle}`, strict `X-Craftsky-Device-Id`, and a full envelope/request-ID surface. Most server-side infrastructure (envelope package, device-id middleware, migration, session-store `TouchDeviceID`) is already on `main` from earlier work. The remaining gaps are: a handful of snake_case tags in `handlers_session.go` + the old `writeJSONError`; a tighter device-id regex with distinct `invalid_device_id` code; a brand-new `whoami` handle resolver; and on the Flutter side, device-id generation + interceptor attachment + bootstrap ordering.

**Tech Stack:** Go 1.22+ (`net/http`, `pgx`, `slog`), indigo (`atproto/identity`, `atproto/auth/oauth`), Flutter (Riverpod 3 code-gen, `dio`, `flutter_secure_storage`, `uuid`), Postgres via `just dev` compose stack.

---

## Pre-Read

Before starting any task:

1. Read the spec once end-to-end: [docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md](../specs/2026-04-22-api-wire-alignment-design.md).
2. Skim [AGENTS.md](../../../AGENTS.md) for repo conventions (migrations via `just`, `sqlc` pattern if you touch SQL, stdlib-only Go routing, `flutter_lints` + the rules in `.claude/rules/`).
3. Note: a lot of server-side infrastructure already exists. The plan only touches what is still missing. If a Task says "modify X at line Y" and the file looks different, read the surrounding code first — there may have been a concurrent change.

**Verification before every commit:**
- Go: `just test` from repo root (runs Go tests against the compose Postgres).
- Flutter: `cd app && dart run build_runner build --delete-conflicting-outputs && dart analyze && flutter test`.
- Commit only when the changed tests pass.

**Branch / worktree strategy.** If a git worktree was created for this plan (via the brainstorming hand-off), work inside it. Otherwise, create a feature branch: `git checkout -b feat/api-wire-alignment`.

**Chunk 6 (smoke test) prerequisites.** Before starting Task 16, `just dev` must be running against a clean `craftsky_sessions` table. Task 16 step 1 has the commands; mentioning it here so anyone parachuting into Chunk 6 doesn't debug a sign-in failure that's really a "stack not up" failure.

---

## File Structure

New files:

```
appview/internal/api/
  whoami_response.go              (new — WhoAmIResponse struct)
  handle_resolver.go              (new — identity.Directory wrapper)
  handle_resolver_test.go         (new)

app/lib/shared/device/
  device_id_provider.dart         (new — UUID-generating Riverpod provider)
  device_id_provider.g.dart       (generated)
```

Modified files (server):

```
appview/internal/auth/handlers_session.go
  - JSON tags snake_case → camelCase on loginRequest / loginResponse
  - writeJSONError call sites switch to envelope.WriteError
  - delete writeJSONError helper from handlers_render.go
appview/internal/auth/handlers_render.go
  - delete writeJSONError (only JSON emitter left here; rest is HTML)
appview/internal/auth/handlers_test.go
  - test fixtures flip snake → camel; expectJSONError → expectEnvelopeError
appview/internal/api/whoami.go
  - rewritten: depends on HandleResolver; returns {did, handle}; 502 on failure
appview/internal/api/whoami_test.go
  - assertions for envelope shape + handle + 502 path
appview/internal/middleware/device_id.go
  - add regex validation; distinguish missing_device_id vs invalid_device_id
appview/internal/middleware/device_id_test.go
  - new case: malformed-but-nonempty → 400 invalid_device_id
appview/internal/app/deps.go
  - construct identity.DefaultDirectory; add HandleResolver (typed as
    api.HandleResolverFunc interface, so routes_test can inject a stub)
    to Deps; pass into handler factory
appview/internal/routes/routes.go
  - WhoAmIHandler now takes a HandleResolver arg
appview/README.md
  - replace snake_case curl examples with camelCase
```

Modified files (client):

```
app/pubspec.yaml
  - add `uuid: ^4.5.1` to dependencies
app/lib/shared/api/craftsky_api_client.dart
  - handoff_mode → handoffMode on login body
app/lib/shared/api/providers/session_auth_interceptor.dart
  - attach X-Craftsky-Device-Id on every request (authenticated and anonymous)
app/lib/shared/api/providers/api_client_provider.dart
  - handoffApiClient family: add deviceId parameter; bake into BaseOptions
app/lib/auth/providers/auth_controller.dart
  - pre-resolve deviceId before constructing handoff client
app/lib/bootstrap.dart
  - eagerly resolve deviceIdProvider alongside dioProvider
app/test/shared/api/providers/session_auth_interceptor_test.dart
  - assert X-Craftsky-Device-Id attached on both anonymous + authed paths
app/test/shared/api/craftsky_api_client_test.dart
  - login body uses handoffMode; assertion updates
app/test/shared/api/handoff_api_client_test.dart
  - handoff client carries both headers
app/test/auth/providers/auth_controller_test.dart
  - handoff provider override takes deviceId parameter
```

Docs:

```
AGENTS.md
  - one-line convention entry under "Coding Conventions"
docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md
  - errata at top of §6 pinning camelCase
```

---

## Chunk 1: Server — kill snake_case on /v1/auth/login + use the envelope

Replace `handlers_session.go`'s snake_case JSON tags and its local `writeJSONError` helper. At the end of this chunk the `/v1/auth/login` endpoint emits the canonical error envelope and consumes camelCase request bodies. No behavioural change to success paths except field renames.

### Task 1: Flip request/response JSON tags on /v1/auth/login

**Files:**
- Modify: `appview/internal/auth/handlers_session.go:17-25`
- Test: `appview/internal/auth/handlers_test.go`

- [ ] **Step 1: Read the current state**

Read `appview/internal/auth/handlers_session.go` lines 17–25 and `handlers_test.go` entirely to understand the existing test harness (`postLogin`, `handlersFixture`, `expectJSONError`).

- [ ] **Step 2: Write the failing test for camelCase request decoding**

Append to `appview/internal/auth/handlers_test.go` (place near the other login-decode tests):

```go
func TestLogin_AcceptsCamelCaseBody(t *testing.T) {
	rr := postLogin(t, handlersFixture(t, ""),
		`{"handle":"alice.example","handoffMode":"deep_link"}`)
	if rr.Code != http.StatusBadGateway {
		// We expect StartAuthFlow to fail because the test fixture has no
		// real PDS; the important assertion is that the request decoded
		// and validation PASSED (otherwise we'd get 400 invalid_handoff_mode).
		t.Fatalf("got %d, want 502 (body decoded, reached StartAuthFlow)", rr.Code)
	}
}

func TestLogin_RejectsSnakeCaseBody(t *testing.T) {
	rr := postLogin(t, handlersFixture(t, ""),
		`{"handle":"alice.example","handoff_mode":"deep_link"}`)
	// handoffMode absent → invalid_handoff_mode 400.
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", rr.Code)
	}
}
```

- [ ] **Step 3: Run to verify they fail**

```bash
just test -run 'TestLogin_AcceptsCamelCaseBody|TestLogin_RejectsSnakeCaseBody' ./internal/auth/...
```

Expected: both fail — `TestLogin_AcceptsCamelCaseBody` fails because the current server rejects camel `handoffMode` as invalid; `TestLogin_RejectsSnakeCaseBody` passes accidentally only if the current server already rejects snake (it doesn't — it accepts snake, so this test will fail because it gets a 502 instead of 400).

- [ ] **Step 4: Flip the JSON tags**

In `appview/internal/auth/handlers_session.go`, change:

```go
type loginRequest struct {
	Handle              string `json:"handle"`
	HandoffMode         string `json:"handoff_mode"` // "deep_link" | "loopback"
	LoopbackRedirectURI string `json:"loopback_redirect_uri,omitempty"`
}

type loginResponse struct {
	AuthURL string `json:"auth_url"`
}
```

to:

```go
type loginRequest struct {
	Handle              string `json:"handle"`
	HandoffMode         string `json:"handoffMode"` // "deep_link" | "loopback"
	LoopbackRedirectURI string `json:"loopbackRedirectUri,omitempty"`
}

type loginResponse struct {
	AuthURL string `json:"authUrl"`
}
```

- [ ] **Step 5: Update existing login tests to send camelCase**

In `appview/internal/auth/handlers_test.go`, replace every snake_case body literal used by `postLogin(...)` with its camelCase equivalent. Examples (grep for `handoff_mode`):

```go
// line ~107 (before):
rr := postLogin(t, handlersFixture(t, ""), `{"handle":"alice.example","handoff_mode":"wat"}`)
// after:
rr := postLogin(t, handlersFixture(t, ""), `{"handle":"alice.example","handoffMode":"wat"}`)

// line ~112:
rr := postLogin(t, handlersFixture(t, ""), `{"handle":"alice.example","handoff_mode":"loopback"}`)
// after:
rr := postLogin(t, handlersFixture(t, ""), `{"handle":"alice.example","handoffMode":"loopback"}`)

// line ~118:
`{"handle":"alice.example","handoff_mode":"loopback","loopback_redirect_uri":"https://evil.example/"}`
// after:
`{"handle":"alice.example","handoffMode":"loopback","loopbackRedirectUri":"https://evil.example/"}`

// line ~124:
`{"handle":"alice.example","handoff_mode":"loopback","loopback_redirect_uri":"javascript:alert(1)"}`
// after:
`{"handle":"alice.example","handoffMode":"loopback","loopbackRedirectUri":"javascript:alert(1)"}`
```

Also if any success-path test decodes the response body and checks for `auth_url`, rename to `authUrl`.

- [ ] **Step 6: Run the full auth test suite**

```bash
just test ./internal/auth/...
```

Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add appview/internal/auth/handlers_session.go appview/internal/auth/handlers_test.go
git commit -m "feat(appview): camelCase JSON on /v1/auth/login"
```

### Task 2: Migrate /v1/auth/login off writeJSONError to envelope.WriteError

**Files:**
- Modify: `appview/internal/auth/handlers_session.go` (every `writeJSONError` call)
- Modify: `appview/internal/auth/handlers_render.go:10-15` (delete `writeJSONError`)
- Modify: `appview/internal/auth/handlers_test.go` (add `expectEnvelopeError` helper; update assertions)

- [ ] **Step 1: Write the failing envelope assertion helper + test**

Add to `appview/internal/auth/handlers_test.go` (near existing `expectJSONError`):

```go
// expectEnvelopeError asserts the response body is a canonical
// envelope.Error with the given status and code, and that requestId
// is non-empty.
func expectEnvelopeError(t *testing.T, rr *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rr.Code != status {
		t.Fatalf("status = %d, want %d; body: %s", rr.Code, status, rr.Body.String())
	}
	var env envelope.Error
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode envelope: %v; body: %s", err, rr.Body.String())
	}
	if env.Error != code {
		t.Errorf("error = %q, want %q", env.Error, code)
	}
	if env.Message == "" {
		t.Errorf("message is empty")
	}
	// requestId may be "" if Logging middleware didn't run in the test
	// harness; we don't assert presence here. Contract tests in
	// routes_test.go cover the end-to-end propagation.
}
```

Then pick ONE existing negative test (e.g. `TestLogin_RejectsSnakeCaseBody` from Task 1, or any current `invalid_handoff_mode` test) and add an `expectEnvelopeError(t, rr, 400, "invalid_handoff_mode")` alongside the existing `expectJSONError` assertion. Add the import for `envelope` if not already present: `"social.craftsky/appview/internal/api/envelope"`.

- [ ] **Step 2: Run — verify it fails**

```bash
just test -run 'TestLogin_RejectsSnakeCaseBody' ./internal/auth/...
```

Expected: fails because current `writeJSONError` emits `{"error":"invalid_handoff_mode"}` with no `message` field → `env.Message == ""` fires.

- [ ] **Step 3: Replace writeJSONError calls with envelope.WriteError in handlers_session.go**

Import once at the top of `handlers_session.go`:

```go
import (
	// ...existing imports...
	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)
```

Replace every `writeJSONError(w, status, code)` call with `envelope.WriteError(w, status, code, msg, middleware.GetRunID(r.Context()), nil)`. Use these messages (keep them short — the `message` field is diagnostic, not user-facing):

| Code | Message |
|---|---|
| `invalid_body` | `"request body could not be decoded"` |
| `handle_required` | `"handle is required"` |
| `invalid_handoff_mode` | `"handoffMode must be deep_link or loopback"` |
| `loopback_redirect_uri_required` | `"loopbackRedirectUri is required when handoffMode is loopback"` |
| `loopback_redirect_uri_invalid` | `"loopbackRedirectUri must match http://127.0.0.1:<port>[/path]"` |
| `authorization_server_unavailable` | `"could not reach the authorization server"` |
| `internal` | `"internal error"` |

For each callsite, pass `r.Context()` to `middleware.GetRunID` — the `r` is in scope inside every `http.HandlerFunc`.

- [ ] **Step 4: Delete writeJSONError from handlers_render.go**

Remove lines 10–15 of `handlers_render.go` (the `writeJSONError` helper and its comment). Leave the HTML helpers (`renderErrorHTML`, templates) untouched.

- [ ] **Step 5: Update remaining test assertions**

Grep for `expectJSONError` in `handlers_test.go`. For every callsite that is asserting a response from a `/v1/auth/login` path (or any other handler we just migrated), replace:

```go
expectJSONError(t, rr, status, code)
```

with:

```go
expectEnvelopeError(t, rr, status, code)
```

Keep `expectJSONError` defined if there are still callsites elsewhere that exercise the callback HTML path or other non-envelope responses — delete it only if it becomes unused.

- [ ] **Step 6: Run the full server test suite**

```bash
just test ./...
```

Expected: all pass. If a test asserted against `writeJSONError`'s `{"error":"code"}` shape with no `message`, it needs to be updated — the migration is incomplete.

- [ ] **Step 7: Commit**

```bash
git add appview/internal/auth/handlers_session.go \
        appview/internal/auth/handlers_render.go \
        appview/internal/auth/handlers_test.go
git commit -m "feat(appview): emit envelope.Error from /v1/auth/login"
```

### Task 3: Update appview README snippets to camelCase

**Files:**
- Modify: `appview/README.md:136`
- Modify: `appview/README.md:176` (only if it shows a JSON body; SQL column stays snake_case)

- [ ] **Step 1: Inspect the README**

Read lines 130–180. The curl example at line ~136 uses `handoff_mode` and `auth_url`. The SQL example at line ~176 uses the DB column `handoff_mode` which stays snake_case — do not touch SQL.

- [ ] **Step 2: Edit the curl examples**

Change:

```
-d '{"handle":"YOUR_HANDLE","handoff_mode":"deep_link"}' | jq -r .auth_url
```

to:

```
-d '{"handle":"YOUR_HANDLE","handoffMode":"deep_link"}' | jq -r .authUrl
```

If the README also shows a `/whoami` response (grep for `"did":"did:plc`), flip it from `{"did":"did:plc:..."}` to `{"did":"did:plc:...","handle":"..."}` — that matches Task 6's new shape.

Leave the `psql` / SQL example alone — it queries the `oauth_auth_requests.handoff_mode` column, which is a DB-side name.

- [ ] **Step 3: Commit**

```bash
git add appview/README.md
git commit -m "docs(appview): update README curl examples to camelCase"
```

---

## Chunk 2: Server — /v1/whoami returns {did, handle}

This is the main feature of the whole spec: resolve the handle on every whoami call via the indigo identity directory, return 502 `identity_unavailable` on failure.

### Task 4: Construct HandleResolver + identity.BaseDirectory in Deps

**Files:**
- Modify: `appview/internal/app/deps.go` (add `HandleResolver` field to `Deps`, construct in `newDeps`)

**Rationale.** Rather than store the raw `identity.Directory` on `Deps` and ask route wiring to wrap it in a resolver, we hoist the whole `HandleResolver` onto `Deps`. Two wins: (1) `routes_test.go` can replace the resolver with a stub without ever touching indigo's real directory, and (2) route wiring becomes a one-liner. (Task 5 creates the resolver type; this task just reserves a field and commits the identity-directory construction so the field is populated.)

- [ ] **Step 1: Read the relevant indigo package**

The directory interface + constructor lives at `github.com/bluesky-social/indigo/atproto/identity`. Verify the exact type by running:

```bash
cd appview && grep -r "identity\.Base\|identity\.Directory" $(go env GOPATH)/pkg/mod/github.com/bluesky-social/indigo*/atproto/identity/ 2>/dev/null | head -20
```

You should find `identity.BaseDirectory` (concrete) satisfying `identity.Directory` (interface). Use `identity.DefaultDirectory()` as the simplest constructor — it returns a `*BaseDirectory` with sensible defaults (in-process cache + HTTP PLC lookups). If `DefaultDirectory` is not available in the vendored version, fall back to `&identity.BaseDirectory{}` with whatever fields the version exposes.

- [ ] **Step 2: Add HandleResolver to Deps**

**Note to implementer:** Task 5 creates `api.HandleResolver`. If Task 5 hasn't been merged yet when you're working on this plan sequentially, swap the order: do Task 5 first (create the type + its tests in isolation with no Deps wiring), then this task. The plan text keeps the current order because the Deps scaffolding is the smaller diff.

In `appview/internal/app/deps.go`, add to the `Deps` struct (after `CraftskySessionStore`):

```go
	// Identity resolution for /v1/whoami. Stubbable in tests by
	// overriding Deps.HandleResolver before wiring routes.
	HandleResolver api.HandleResolver
```

Add imports:

```go
	"github.com/bluesky-social/indigo/atproto/identity"

	"social.craftsky/appview/internal/api"
```

- [ ] **Step 3: Initialise it in newDeps**

Inside `newDeps`, after the existing `craftskyStore := ...` line, add:

```go
	identityDir := identity.DefaultDirectory()
```

And populate it in the returned `Deps` literal:

```go
	deps := &Deps{
		// ...existing fields...
		CraftskySessionStore: craftskyStore,
		HandleResolver:       api.HandleResolver{Directory: identityDir},
		// ...
	}
```

- [ ] **Step 4: Compile check**

```bash
cd appview && go build ./...
```

Expected: if Task 5 hasn't landed yet this will fail on `api.HandleResolver` undefined — do Task 5 first and return to this step. Otherwise no errors.

- [ ] **Step 5: Commit**

```bash
git add appview/internal/app/deps.go
git commit -m "feat(appview): add HandleResolver to Deps"
```

### Task 5: Write HandleResolver

**Files:**
- Create: `appview/internal/api/handle_resolver.go`
- Create: `appview/internal/api/handle_resolver_test.go`

- [ ] **Step 1: Write the failing tests**

Create `appview/internal/api/handle_resolver_test.go`:

```go
package api

import (
	"context"
	"errors"
	"testing"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

type fakeDirectory struct {
	identity *identity.Identity
	err      error
}

func (f *fakeDirectory) LookupDID(ctx context.Context, did syntax.DID) (*identity.Identity, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.identity, nil
}

// Stubs for the rest of the identity.Directory interface. We only
// care about LookupDID; everything else panics if exercised.
func (f *fakeDirectory) LookupHandle(context.Context, syntax.Handle) (*identity.Identity, error) {
	panic("unexpected LookupHandle")
}
func (f *fakeDirectory) Lookup(context.Context, syntax.AtIdentifier) (*identity.Identity, error) {
	panic("unexpected Lookup")
}
func (f *fakeDirectory) Purge(context.Context, syntax.AtIdentifier) error {
	panic("unexpected Purge")
}

func TestHandleResolver_HappyPath(t *testing.T) {
	r := HandleResolver{Directory: &fakeDirectory{
		identity: &identity.Identity{Handle: syntax.Handle("alice.bsky.social")},
	}}
	h, err := r.ResolveHandle(context.Background(), "did:plc:abc")
	if err != nil {
		t.Fatalf("ResolveHandle: %v", err)
	}
	if h != "alice.bsky.social" {
		t.Errorf("handle = %q, want %q", h, "alice.bsky.social")
	}
}

func TestHandleResolver_MalformedDID(t *testing.T) {
	r := HandleResolver{Directory: &fakeDirectory{}}
	_, err := r.ResolveHandle(context.Background(), "not-a-did")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHandleResolver_DirectoryError(t *testing.T) {
	r := HandleResolver{Directory: &fakeDirectory{err: errors.New("network down")}}
	_, err := r.ResolveHandle(context.Background(), "did:plc:abc")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHandleResolver_EmptyHandle(t *testing.T) {
	r := HandleResolver{Directory: &fakeDirectory{
		identity: &identity.Identity{Handle: syntax.HandleInvalid},
	}}
	_, err := r.ResolveHandle(context.Background(), "did:plc:abc")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
```

Note: `identity.Directory` is an interface in indigo. Our `fakeDirectory` must satisfy it. If the interface has *additional* methods beyond the ones stubbed here, `go build` will fail — add panicking stubs for each additional method. The compile error message tells you which.

- [ ] **Step 2: Run to verify it fails to compile**

```bash
just test ./internal/api/...
```

Expected: compile error — `HandleResolver` undefined.

- [ ] **Step 3: Write handle_resolver.go**

Create `appview/internal/api/handle_resolver.go`:

```go
package api

import (
	"context"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

// HandleResolver resolves a DID to its current handle via indigo's
// identity.Directory. v1 does no caching beyond what the directory
// provides internally — every /v1/whoami call pays one lookup.
//
// A nil Directory is a programmer error and will panic on use.
type HandleResolver struct {
	Directory identity.Directory
}

// ErrHandleUnavailable wraps every failure mode (malformed DID,
// directory error, empty handle) into a single sentinel. Handlers
// convert this to 502 identity_unavailable.
var ErrHandleUnavailable = errors.New("handle unavailable")

// ResolveHandle returns the handle for did.
func (r HandleResolver) ResolveHandle(ctx context.Context, did string) (string, error) {
	parsed, err := syntax.ParseDID(did)
	if err != nil {
		return "", fmt.Errorf("%w: parse did: %v", ErrHandleUnavailable, err)
	}
	id, err := r.Directory.LookupDID(ctx, parsed)
	if err != nil {
		return "", fmt.Errorf("%w: lookup: %v", ErrHandleUnavailable, err)
	}
	h := id.Handle.String()
	if h == "" || h == "handle.invalid" {
		// "handle.invalid" is the indigo sentinel for DIDs with no
		// valid handle (deactivated, mid-migration, etc.).
		return "", fmt.Errorf("%w: empty handle for %s", ErrHandleUnavailable, did)
	}
	return h, nil
}
```

- [ ] **Step 4: Run tests**

```bash
just test ./internal/api/...
```

Expected: the four resolver tests pass. If the `identity.Directory` interface has more methods than stubbed, add panicking stubs in the test until compile is clean.

- [ ] **Step 5: Commit**

```bash
git add appview/internal/api/handle_resolver.go appview/internal/api/handle_resolver_test.go
git commit -m "feat(appview): add HandleResolver over identity.Directory"
```

### Task 6: Rewrite /v1/whoami to return {did, handle}

**Files:**
- Modify: `appview/internal/api/whoami.go`
- Create: `appview/internal/api/whoami_response.go`
- Modify: `appview/internal/api/whoami_test.go`

- [ ] **Step 1: Read the existing whoami tests**

Read `appview/internal/api/whoami_test.go` completely. Note the fixture pattern (how it seeds a DID into the context before calling the handler).

- [ ] **Step 2: Write the failing tests**

Replace `appview/internal/api/whoami_test.go` with a version that covers:

1. Happy path — resolver returns handle → 200, body `{"did":"...","handle":"..."}`.
2. Resolver error → 502 with envelope `{"error":"identity_unavailable",...}`.
3. No DID in context → 500 with envelope `{"error":"internal_error",...}`.

Use the existing `middleware.WithDID(ctx, ...)` helper to seed DID. Example skeleton:

```go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

type stubResolver struct {
	handle string
	err    error
}

func (s stubResolver) ResolveHandle(ctx context.Context, did string) (string, error) {
	return s.handle, s.err
}

func TestWhoAmI_HappyPath(t *testing.T) {
	h := WhoAmIHandler(stubResolver{handle: "alice.example"})
	req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
	req = req.WithContext(middleware.WithDID(req.Context(), "did:plc:abc"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}
	var body WhoAmIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.DID != "did:plc:abc" {
		t.Errorf("did = %q", body.DID)
	}
	if body.Handle != "alice.example" {
		t.Errorf("handle = %q", body.Handle)
	}
}

func TestWhoAmI_DirectoryUnavailable(t *testing.T) {
	h := WhoAmIHandler(stubResolver{err: errors.New("plc down")})
	req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
	req = req.WithContext(middleware.WithDID(req.Context(), "did:plc:abc"))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rr.Code)
	}
	var env envelope.Error
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Error != "identity_unavailable" {
		t.Errorf("code = %q", env.Error)
	}
}

func TestWhoAmI_NoDIDInContext(t *testing.T) {
	h := WhoAmIHandler(stubResolver{handle: "unused"})
	req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rr.Code)
	}
	var env envelope.Error
	if err := json.Unmarshal(rr.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if env.Error != "internal_error" {
		t.Errorf("code = %q", env.Error)
	}
}
```

- [ ] **Step 3: Run — expect compile failure**

```bash
just test ./internal/api/...
```

Expected: `WhoAmIResponse` undefined, `WhoAmIHandler` signature mismatch.

- [ ] **Step 4: Write WhoAmIResponse**

Create `appview/internal/api/whoami_response.go`:

```go
package api

// WhoAmIResponse is the 200 body for GET /v1/whoami.
type WhoAmIResponse struct {
	DID    string `json:"did"`
	Handle string `json:"handle"`
}
```

- [ ] **Step 5: Rewrite the handler**

Replace `appview/internal/api/whoami.go` entirely:

```go
package api

import (
	"context"
	"encoding/json"
	"net/http"

	"social.craftsky/appview/internal/api/envelope"
	"social.craftsky/appview/internal/middleware"
)

// HandleResolverFunc is the minimal interface WhoAmIHandler needs.
// HandleResolver from handle_resolver.go satisfies it; tests stub it.
type HandleResolverFunc interface {
	ResolveHandle(ctx context.Context, did string) (string, error)
}

// WhoAmIHandler returns the caller's DID and current handle.
//
// The DID is read from the request context (injected by the
// Authenticated middleware). The handle is resolved on every call via
// the identity directory.
//
// Errors collapse:
//   - DID missing from context → 500 internal_error (routing bug).
//   - Directory lookup failure (unknown DID, empty handle, network) →
//     502 identity_unavailable.
func WhoAmIHandler(resolver HandleResolverFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		did, ok := middleware.GetDID(r.Context())
		if !ok {
			envelope.WriteError(w, http.StatusInternalServerError,
				"internal_error", "no did in context",
				middleware.GetRunID(r.Context()), nil)
			return
		}
		handle, err := resolver.ResolveHandle(r.Context(), did)
		if err != nil {
			envelope.WriteError(w, http.StatusBadGateway,
				"identity_unavailable", "could not resolve handle for did",
				middleware.GetRunID(r.Context()), nil)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(WhoAmIResponse{DID: did, Handle: handle})
	})
}
```

- [ ] **Step 6: Run tests**

```bash
just test ./internal/api/...
```

Expected: three whoami tests + four handle-resolver tests pass.

- [ ] **Step 7: Commit**

```bash
git add appview/internal/api/whoami.go appview/internal/api/whoami_response.go appview/internal/api/whoami_test.go
git commit -m "feat(appview): /v1/whoami returns {did, handle}"
```

### Task 7: Wire HandleResolver into routes

**Files:**
- Modify: `appview/internal/routes/routes.go:39`
- Modify: `appview/internal/routes/routes_test.go`

- [ ] **Step 1: Update WhoAmIHandler call site**

Change line 39 of `appview/internal/routes/routes.go`:

```go
mux.Handle("GET /v1/whoami", authN(deviceID(api.WhoAmIHandler())))
```

to:

```go
mux.Handle("GET /v1/whoami", authN(deviceID(api.WhoAmIHandler(deps.HandleResolver))))
```

- [ ] **Step 2: Stub the resolver in routes_test.go**

Existing route tests for `/v1/whoami` (such as `TestAddRoutes_V1WhoAmIAuthenticatedReturnsDID`) seed a fake DID via the `X-Dev-DID` header and assert a 200 with the DID in the body. After Task 6 these tests MUST seed a stub handle resolver too — otherwise the real `identity.DefaultDirectory()` makes network calls for a nonexistent DID and the test either hangs or returns 502.

In `routes_test.go`, find where the test builds `*app.Deps` (or a `Deps`-shaped literal). Replace `HandleResolver: api.HandleResolver{Directory: identity.DefaultDirectory()}` (or the equivalent construction) with a stub. The simplest path:

```go
// Stub resolver so route tests don't depend on the real PLC directory.
type stubResolver struct{ handle string }

func (s stubResolver) ResolveHandle(ctx context.Context, did string) (string, error) {
    return s.handle, nil
}
```

And in the Deps construction:

```go
HandleResolver: api.HandleResolver{},  // zero value — will not be called
```

This works because the test's seeded DID (`"did:plc:from-header"`) has no real PLC entry and the test happens to have been asserting `{did}` only. After Task 6 the handler returns `{did, handle}`; update the assertion:

```go
// before:
if got := body["did"]; got != "did:plc:from-header" { ... }
// after:
if got := body["did"]; got != "did:plc:from-header" { ... }
if got := body["handle"]; got != "stub-handle.example" { ... }
```

Using a **real** stub resolver is cleaner. Change `HandleResolver: api.HandleResolver{}` in the test fixture to go through a function the test injects — but `api.HandleResolver` is a struct, not an interface. Two options:

1. **Add a `HandleResolverFunc` interface to `Deps` instead of the struct.** `Deps.HandleResolver` is typed as `api.HandleResolverFunc` (an interface with the single `ResolveHandle` method — which Task 6 already introduces). Then the test can pass a function-adapter stub. This is the cleaner path; recommend taking it.

2. **Keep the struct; wrap the `identity.Directory` field with a fake.** In the test, implement a `fakeDirectory` (like the one in `handle_resolver_test.go` from Task 5) and set `HandleResolver: api.HandleResolver{Directory: fakeDirectory{identity: &identity.Identity{Handle: syntax.Handle("stub-handle.example")}}}`.

**Pick option 1:** retype `Deps.HandleResolver` as `api.HandleResolverFunc` (the interface). `api.HandleResolver` (the struct) satisfies the interface; Task 4 already populates Deps with a `HandleResolver` struct value which is accepted by the interface field.

Concretely, in `appview/internal/app/deps.go`, change the field type to the interface:

```go
// before (from Task 4):
HandleResolver api.HandleResolver
// after:
HandleResolver api.HandleResolverFunc
```

Then in `routes_test.go` add the stub struct as above and set it in the Deps fixture.

- [ ] **Step 3: Compile + test**

```bash
just test ./...
```

Expected: all tests pass. If any other route test touches `/v1/whoami` and still expects a DID-only body, update its assertions the same way.

- [ ] **Step 4: Commit**

```bash
git add appview/internal/routes/routes.go \
        appview/internal/routes/routes_test.go \
        appview/internal/app/deps.go
git commit -m "feat(appview): wire HandleResolver into /v1/whoami route; stub in tests"
```

---

## Chunk 3: Server — tighten device-id validation

Currently the device-id middleware rejects only missing / over-length. The spec asks for a regex on allowed characters and a distinct `invalid_device_id` code for malformed-but-nonempty values. Small, self-contained change.

### Task 8: Regex-validate X-Craftsky-Device-Id + distinct error codes

**Files:**
- Modify: `appview/internal/middleware/device_id.go`
- Modify: `appview/internal/middleware/device_id_test.go`

- [ ] **Step 1: Write the failing tests**

Add to `appview/internal/middleware/device_id_test.go` (near existing negative cases):

```go
func TestDeviceID_MalformedHeader_ReturnsInvalidCode(t *testing.T) {
	mw := DeviceID(nil, testLogger())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not run")
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
	req.Header.Set("X-Craftsky-Device-Id", "has spaces")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	var env envelope.Error
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Error != "invalid_device_id" {
		t.Errorf("code = %q, want invalid_device_id", env.Error)
	}
}

func TestDeviceID_AcceptsUUIDWithDashes(t *testing.T) {
	mw := DeviceID(nil, testLogger())
	ran := false
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ran = true
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
	req.Header.Set("X-Craftsky-Device-Id", "01925c42-fe83-7d51-a7f8-5e2e9b1c8d0f")
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if !ran {
		t.Fatal("next handler did not run")
	}
}

func TestDeviceID_RejectsOverLength(t *testing.T) {
	mw := DeviceID(nil, testLogger())
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not run")
	}))
	req := httptest.NewRequest(http.MethodGet, "/v1/whoami", nil)
	req.Header.Set("X-Craftsky-Device-Id", strings.Repeat("a", 129))
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
	var env envelope.Error
	_ = json.Unmarshal(rr.Body.Bytes(), &env)
	if env.Error != "invalid_device_id" {
		t.Errorf("code = %q, want invalid_device_id", env.Error)
	}
}
```

Add needed imports: `"strings"`, `"encoding/json"`, `"social.craftsky/appview/internal/api/envelope"`.

If `testLogger()` doesn't exist in the test file, either find the existing helper used by current tests or inline `slog.New(slog.NewTextHandler(io.Discard, nil))`.

The existing missing-header test should keep expecting `missing_device_id`. Verify by reading the existing file before editing.

- [ ] **Step 2: Run to verify failures**

```bash
just test ./internal/middleware/...
```

Expected: the new malformed + over-length tests fail — current middleware returns `missing_device_id` (not `invalid_device_id`) for both.

- [ ] **Step 3: Implement regex validation**

Edit `appview/internal/middleware/device_id.go`. Change:

```go
const maxDeviceIDLen = 256
```

to:

```go
const maxDeviceIDLen = 128
```

(Aligns with the spec's §3.2 regex bound.)

Add at file scope:

```go
var deviceIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
```

Add the import: `"regexp"`.

Inside the handler function, replace the existing validation block:

```go
if id == "" || len(id) > maxDeviceIDLen {
    logger.Warn("device-id: missing or invalid header", ...)
    envelope.WriteError(w, http.StatusBadRequest,
        "missing_device_id",
        "X-Craftsky-Device-Id header is required",
        GetRunID(r.Context()),
        nil)
    return
}
```

with:

```go
if id == "" {
    logger.Warn("device-id: missing header",
        slog.String("run_id", GetRunID(r.Context())))
    envelope.WriteError(w, http.StatusBadRequest,
        "missing_device_id",
        "X-Craftsky-Device-Id header is required",
        GetRunID(r.Context()),
        nil)
    return
}
if !deviceIDPattern.MatchString(id) {
    logger.Warn("device-id: malformed header",
        slog.Int("len", len(id)),
        slog.String("run_id", GetRunID(r.Context())))
    envelope.WriteError(w, http.StatusBadRequest,
        "invalid_device_id",
        "X-Craftsky-Device-Id is malformed",
        GetRunID(r.Context()),
        nil)
    return
}
```

- [ ] **Step 4: Prune now-redundant test**

If an existing test named along the lines of `TestDeviceID_TooLongHeaderReturns400` exists and sends a 257-byte header, it now asserts behaviour identical to the new `TestDeviceID_RejectsOverLength` (both → 400 `invalid_device_id` after this task). Delete the old test to avoid redundancy. Keep the new one since it additionally verifies the error code.

- [ ] **Step 5: Run tests**

```bash
just test ./internal/middleware/...
```

Expected: all device-id tests pass.

- [ ] **Step 6: Run the full test suite**

```bash
just test ./...
```

Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add appview/internal/middleware/device_id.go appview/internal/middleware/device_id_test.go
git commit -m "feat(appview): regex-validate X-Craftsky-Device-Id with distinct invalid code"
```

---

## Chunk 4: Client — deviceId generation and header attachment

This is the Flutter side. It's a chain: add the `uuid` dep, write the provider, route it through the interceptor, update the handoff client, gate on bootstrap. Tests at each step.

### Task 9: Add uuid dependency

**Files:**
- Modify: `app/pubspec.yaml`

- [ ] **Step 1: Add the dep**

In `app/pubspec.yaml`, under `dependencies:`, add (alphabetical sort — likely before `url_launcher` if present):

```yaml
  uuid: ^4.5.1
```

Then from `app/`:

```bash
flutter pub get
```

Expected: resolves cleanly. Version `^4.5.1` is the latest stable at plan write time; bump if a newer latest is current at implementation time.

- [ ] **Step 2: Commit**

```bash
git add app/pubspec.yaml app/pubspec.lock
git commit -m "chore(app): add uuid dependency"
```

### Task 10: Write deviceIdProvider

**Files:**
- Create: `app/lib/shared/device/device_id_provider.dart`
- Create: `app/test/shared/device/device_id_provider_test.dart`

- [ ] **Step 1: Write the failing tests**

Create `app/test/shared/device/device_id_provider_test.dart`:

```dart
import 'package:craftsky_app/shared/device/device_id_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:flutter_test/flutter_test.dart';

class _FakeSecureStorage implements FlutterSecureStorage {
  _FakeSecureStorage();
  final Map<String, String> _map = <String, String>{};

  @override
  Future<String?> read({
    required String key,
    IOSOptions? iOptions,
    AndroidOptions? aOptions,
    LinuxOptions? lOptions,
    WindowsOptions? wOptions,
    WebOptions? webOptions,
    MacOsOptions? mOptions,
  }) async => _map[key];

  @override
  Future<void> write({
    required String key,
    required String? value,
    IOSOptions? iOptions,
    AndroidOptions? aOptions,
    LinuxOptions? lOptions,
    WindowsOptions? wOptions,
    WebOptions? webOptions,
    MacOsOptions? mOptions,
  }) async {
    if (value == null) {
      _map.remove(key);
    } else {
      _map[key] = value;
    }
  }

  // Other methods throw — we exercise only read/write in the provider.
  @override
  dynamic noSuchMethod(Invocation inv) => super.noSuchMethod(inv);
}

void main() {
  test('generates and persists a UUID when storage is empty', () async {
    final storage = _FakeSecureStorage();
    final container = ProviderContainer.test(
      overrides: [deviceIdSecureStorageProvider.overrideWithValue(storage)],
    );

    final id = await container.read(deviceIdProvider.future);
    expect(id, isNotEmpty);
    expect(id.length, greaterThanOrEqualTo(32));
    expect(await storage.read(key: 'craftsky_device_id'), id);
  });

  test('returns the existing ID when storage already has one', () async {
    final storage = _FakeSecureStorage();
    await storage.write(key: 'craftsky_device_id', value: 'pre-existing-id');
    final container = ProviderContainer.test(
      overrides: [deviceIdSecureStorageProvider.overrideWithValue(storage)],
    );

    final id = await container.read(deviceIdProvider.future);
    expect(id, 'pre-existing-id');
  });

  test('two reads return the same ID (keep-alive cached)', () async {
    final storage = _FakeSecureStorage();
    final container = ProviderContainer.test(
      overrides: [deviceIdSecureStorageProvider.overrideWithValue(storage)],
    );

    final first = await container.read(deviceIdProvider.future);
    final second = await container.read(deviceIdProvider.future);
    expect(first, second);
  });
}
```

(The `_FakeSecureStorage.noSuchMethod` pattern is deliberate — `FlutterSecureStorage`'s interface has many methods we don't exercise; matching them all would bloat the fake. If `dart analyze` complains about missing overrides, add `@override` stubs that throw `UnimplementedError` for `delete`, `deleteAll`, `readAll`, `containsKey`, `isCupertinoProtectedDataAvailable`, `registerListener`, `unregisterListener`, `unregisterAllListenersForKey`, `unregisterAllListeners`.)

- [ ] **Step 2: Run tests — expect failure**

From `app/`:

```bash
flutter test test/shared/device/device_id_provider_test.dart
```

Expected: import errors — `deviceIdProvider`, `deviceIdSecureStorageProvider` don't exist.

- [ ] **Step 3: Write the provider**

Create `app/lib/shared/device/device_id_provider.dart`:

```dart
import 'package:flutter/services.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:logging/logging.dart';
import 'package:riverpod_annotation/riverpod_annotation.dart';
import 'package:uuid/uuid.dart';

part 'device_id_provider.g.dart';

final _log = Logger('DeviceIdProvider');

/// Secure-storage key for the per-install device identifier. Separate
/// from `craftsky_session` so sign-out does NOT clear the device ID.
const deviceIdStorageKey = 'craftsky_device_id';

/// Injection seam: production uses a shared `FlutterSecureStorage`
/// instance; tests override with a fake.
@Riverpod(keepAlive: true)
FlutterSecureStorage deviceIdSecureStorage(Ref ref) =>
    const FlutterSecureStorage();

/// Returns this install's stable device identifier. On first access,
/// generates a v4 UUID and writes it to secure storage. Subsequent
/// accesses return the persisted value.
///
/// Platform-error tolerant: if secure storage fails, we return a fresh
/// in-memory UUID for this session and attempt to persist it. On a
/// persistence failure, future launches may generate a different ID —
/// acceptable because device-id is correlation data, not a security
/// primitive.
@Riverpod(keepAlive: true)
Future<String> deviceId(Ref ref) async {
  final storage = ref.watch(deviceIdSecureStorageProvider);
  try {
    final existing = await storage.read(key: deviceIdStorageKey);
    if (existing != null && existing.isNotEmpty) return existing;
  } on PlatformException catch (e, st) {
    _log.warning('device-id read failed; will mint a fresh one', e, st);
  }

  final fresh = const Uuid().v4();
  try {
    await storage.write(key: deviceIdStorageKey, value: fresh);
  } on PlatformException catch (e, st) {
    _log.warning('device-id write failed; using in-memory only', e, st);
  }
  return fresh;
}
```

- [ ] **Step 4: Run build_runner**

From `app/`:

```bash
dart run build_runner build --delete-conflicting-outputs
```

Expected: generates `device_id_provider.g.dart`.

- [ ] **Step 5: Run tests**

```bash
flutter test test/shared/device/device_id_provider_test.dart
```

Expected: all three tests pass.

- [ ] **Step 6: Analyze + commit**

```bash
dart analyze
```

Expected: no errors.

```bash
git add app/lib/shared/device/ app/test/shared/device/
git commit -m "feat(app): add deviceIdProvider with UUID + secure storage"
```

### Task 11: Attach X-Craftsky-Device-Id in SessionAuthInterceptor

**Files:**
- Modify: `app/lib/shared/api/providers/session_auth_interceptor.dart`
- Modify: `app/test/shared/api/providers/session_auth_interceptor_test.dart`

- [ ] **Step 1: Read the existing test**

Read `app/test/shared/api/providers/session_auth_interceptor_test.dart` entirely (note the `providers/` subdirectory). The existing tests all use `SessionAuthInterceptor.withReader(() => ...)` — a one-arg constructor. After this task, the interceptor gains a device-id reader; we keep `.withReader` working as a backward-compat alias that injects a fixed dummy device-id, so existing tests keep compiling without modification. New tests use `.withReaders(...)` (plural) to exercise device-id attachment explicitly.

- [ ] **Step 2: Write the failing tests**

Append to `app/test/shared/api/providers/session_auth_interceptor_test.dart` (inside the existing `main()` body, after the existing tests):

```dart
  // --- Device-ID header attachment ---

  test('attaches X-Craftsky-Device-Id on anonymous requests', () async {
    final container = ProviderContainer.test(
      overrides: [authSessionProvider.overrideWith(_SignedOutFake.new)],
    );
    await seed(container);

    final options = RequestOptions(path: '/v1/auth/login');
    SessionAuthInterceptor.withReaders(
      readAuth: () => container.read(authSessionProvider),
      readDeviceId: () async => 'device-abc',
    ).onRequest(options, _CapturingHandler());

    // onRequest is `void`-returning but internally awaits; poll until
    // the handler is notified the chain continued.
    await _pumpEventLoop();

    expect(options.headers['X-Craftsky-Device-Id'], 'device-abc');
    expect(options.headers.containsKey('Authorization'), isFalse);
  });

  test('attaches BOTH Authorization and X-Craftsky-Device-Id on authed requests', () async {
    final container = ProviderContainer.test(
      overrides: [
        authSessionProvider.overrideWith(() => _SignedInFake('tok')),
      ],
    );
    await seed(container);

    final options = RequestOptions(path: '/v1/whoami');
    SessionAuthInterceptor.withReaders(
      readAuth: () => container.read(authSessionProvider),
      readDeviceId: () async => 'device-abc',
    ).onRequest(options, _CapturingHandler());

    await _pumpEventLoop();

    expect(options.headers['Authorization'], 'Bearer tok');
    expect(options.headers['X-Craftsky-Device-Id'], 'device-abc');
  });
```

Add a tiny helper at the bottom of the file (above `main`'s closing brace or as a top-level function):

```dart
/// Flushes microtasks so async gaps inside a `void`-returning async
/// interceptor settle before assertions run.
Future<void> _pumpEventLoop() async {
  for (var i = 0; i < 5; i++) {
    await Future<void>.delayed(Duration.zero);
  }
}
```

The pump-loop idiom is the same one prescribed in `.claude/rules/riverpod.md` for fire-and-forget async work.

- [ ] **Step 3: Run — expect failure**

```bash
flutter test test/shared/api/providers/session_auth_interceptor_test.dart
```

Expected: compile error on `SessionAuthInterceptor.withReaders` (doesn't exist yet).

- [ ] **Step 4: Extend the interceptor**

Rewrite `app/lib/shared/api/providers/session_auth_interceptor.dart`:

```dart
import 'package:craftsky_app/auth/models/auth_state.dart';
import 'package:craftsky_app/auth/providers/auth_session_provider.dart';
import 'package:craftsky_app/shared/device/device_id_provider.dart';
import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

/// Paths on which the Authorization header should never be attached.
/// X-Craftsky-Device-Id is sent on ALL paths including these — the
/// server treats device-id as install-identity, not user-identity.
const _anonymousPaths = <String>{'/v1/auth/login'};

class SessionAuthInterceptor extends Interceptor {
  /// Production constructor.
  SessionAuthInterceptor.fromRef(Ref ref)
      : _readAuth = (() => ref.read(authSessionProvider)),
        _readDeviceId = (() => ref.read(deviceIdProvider.future));

  /// Back-compat test constructor — injects a fixed dummy device-id
  /// so the existing 4 tests that only care about the Authorization
  /// header keep working without a rewrite.
  SessionAuthInterceptor.withReader(
    AsyncValue<AuthState> Function() readAuth,
  )   : _readAuth = readAuth,
        _readDeviceId = (() async => 'test-device-id');

  /// Full test constructor — both readers explicit.
  SessionAuthInterceptor.withReaders({
    required AsyncValue<AuthState> Function() readAuth,
    required Future<String> Function() readDeviceId,
  })  : _readAuth = readAuth,
        _readDeviceId = readDeviceId;

  final AsyncValue<AuthState> Function() _readAuth;
  final Future<String> Function() _readDeviceId;

  // Dio's base signature is `void onRequest(...)`. We declare `void`
  // with an `async` body so we can await the device-id future; Dio
  // continues the chain when we call `handler.next(options)`, not
  // when our future resolves. Do NOT change the return type to
  // Future<void> — it's an invalid override.
  @override
  void onRequest(
    RequestOptions options,
    RequestInterceptorHandler handler,
  ) async {
    final deviceId = await _readDeviceId();
    options.headers['X-Craftsky-Device-Id'] = deviceId;

    if (!_anonymousPaths.contains(options.path)) {
      final auth = _readAuth().value;
      if (auth is SignedIn) {
        options.headers['Authorization'] = 'Bearer ${auth.token}';
      }
    }
    handler.next(options);
  }
}
```

Two things this shape locks in:

1. `withReader` is preserved for the four existing tests that pass only an auth reader. They inject a fixed `'test-device-id'` automatically and their assertions (about `Authorization` only) stay valid.
2. `onRequest` stays `void`-returning to satisfy Dio's `Interceptor` base. The async body is legal; Dio continues the chain on `handler.next(options)` regardless of when the method's async work completes.

- [ ] **Step 5: Run tests**

```bash
flutter test test/shared/api/session_auth_interceptor_test.dart
```

Expected: new tests pass. If the existing `_CaptureHandler` didn't exist, match the pattern the existing tests use instead and adapt.

- [ ] **Step 6: Run full app test suite + analyze**

```bash
flutter test
dart analyze
```

Expected: no regressions. The existing `SessionAuthInterceptor` callsites compile because `fromRef` still exists.

- [ ] **Step 7: Commit**

```bash
git add app/lib/shared/api/providers/session_auth_interceptor.dart \
        app/test/shared/api/providers/session_auth_interceptor_test.dart
git commit -m "feat(app): SessionAuthInterceptor attaches X-Craftsky-Device-Id"
```

### Task 12: Flip login body to camelCase + drop snakeCase override on LoginResponse

**Files:**
- Modify: `app/lib/shared/api/craftsky_api_client.dart:25`
- Modify: `app/lib/shared/api/models/login_response.dart:5`
- Modify: `app/test/shared/api/craftsky_api_client_test.dart`
- Modify: `app/test/shared/api/models/login_response_test.dart` (if it asserts snake_case)

**Why both files.** The client currently sends `{handoff_mode: ...}` AND decodes `{auth_url: ...}` because `LoginResponse` has `@MappableClass(caseStyle: CaseStyle.snakeCase)`. After Task 1 the server emits `{authUrl: ...}` — if we flip only the send body and leave the decoder on snake_case, `authUrl` arrives but the mapper reads `auth_url`, sees null, decode fails, sign-in breaks. Both flips must land together.

- [ ] **Step 1: Update the failing test**

Find the test in `app/test/shared/api/craftsky_api_client_test.dart` that asserts the `POST /v1/auth/login` body shape. Change the expected body from `{'handle': ..., 'handoff_mode': 'deep_link'}` to `{'handle': ..., 'handoffMode': 'deep_link'}`.

Add a NEW test that asserts snake_case is NOT sent (guard against regression):

```dart
test('login sends handoffMode (not handoff_mode)', () async {
  final captured = <Map<String, dynamic>>[];
  final dio = _dioWithCapture(captured);
  dio.httpClientAdapter.fetch = (options, _, __) async {
    return ResponseBody.fromString(
      jsonEncode({'authUrl': 'https://example/auth'}),
      200,
      headers: {'content-type': ['application/json']},
    );
  };
  final client = CraftskyApiClient(dio);
  await client.login(handle: 'alice.example');
  expect(captured.single.containsKey('handoffMode'), isTrue);
  expect(captured.single.containsKey('handoff_mode'), isFalse);
});
```

Adapt `_dioWithCapture` to the existing fixture pattern in the file if a different pattern is in use.

- [ ] **Step 2: Run — expect failure**

```bash
flutter test test/shared/api/craftsky_api_client_test.dart
```

Expected: fails because the client still sends `handoff_mode`.

- [ ] **Step 3: Flip the client**

In `app/lib/shared/api/craftsky_api_client.dart` line 25, change:

```dart
data: {'handle': handle, 'handoff_mode': 'deep_link'},
```

to:

```dart
data: {'handle': handle, 'handoffMode': 'deep_link'},
```

- [ ] **Step 4: Drop the snakeCase override on LoginResponse**

In `app/lib/shared/api/models/login_response.dart`, change:

```dart
@MappableClass(caseStyle: CaseStyle.snakeCase)
class LoginResponse with LoginResponseMappable {
```

to:

```dart
@MappableClass()
class LoginResponse with LoginResponseMappable {
```

Then regenerate:

```bash
dart run build_runner build --delete-conflicting-outputs
```

The generated `login_response.mapper.dart` should now list the key as `'authUrl'` (not `'auth_url'`).

- [ ] **Step 5: Update LoginResponse model test**

If `app/test/shared/api/models/login_response_test.dart` asserts decode from a snake_case JSON literal (`'{"auth_url": "..."}'`), flip it to camelCase (`'{"authUrl": "..."}'`). Add a case that decodes from camelCase and passes; drop any case asserting snake_case is the wire shape.

- [ ] **Step 6: Run tests**

```bash
flutter test test/shared/api/
```

Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add app/lib/shared/api/craftsky_api_client.dart \
        app/lib/shared/api/models/login_response.dart \
        app/lib/shared/api/models/login_response.mapper.dart \
        app/test/shared/api/craftsky_api_client_test.dart \
        app/test/shared/api/models/login_response_test.dart
git commit -m "feat(app): send + decode camelCase on /v1/auth/login"
```

### Task 13: Handoff client carries deviceId

**Files:**
- Modify: `app/lib/shared/api/providers/api_client_provider.dart`
- Modify: `app/lib/auth/providers/auth_controller.dart`
- Modify: `app/test/shared/api/handoff_api_client_test.dart`
- Modify: `app/test/auth/providers/auth_controller_test.dart`

- [ ] **Step 1: Extend the family signature**

Edit `app/lib/shared/api/providers/api_client_provider.dart`. The current `handoffApiClient` takes `(Ref ref, String token)`; extend to `(Ref ref, String token, String deviceId)`:

```dart
@riverpod
HandoffApiClient handoffApiClient(Ref ref, String token, String deviceId) {
  final base = baseDioOptions();
  final dio = Dio(
    base.copyWith(
      headers: {
        ...base.headers,
        'Authorization': 'Bearer $token',
        'X-Craftsky-Device-Id': deviceId,
      },
    ),
  )..interceptors.add(const ErrorMappingInterceptor());
  return HandoffApiClient(dio);
}
```

- [ ] **Step 2: Update AuthController.completeFromDeepLink to pre-resolve device-id**

In `app/lib/auth/providers/auth_controller.dart`, in `completeFromDeepLink`, find where `handoffApiClientProvider(token)` is read and change to:

```dart
final deviceId = await ref.read(deviceIdProvider.future);
if (!ref.mounted) return;
final handoff = ref.read(handoffApiClientProvider(token, deviceId));
```

Add the import: `import 'package:craftsky_app/shared/device/device_id_provider.dart';`.

The `if (!ref.mounted) return;` guard after the await is required per [.claude/rules/riverpod.md](../../../.claude/rules/riverpod.md) (Lifecycle & Safety). Match the existing pattern in the controller.

- [ ] **Step 3: Regenerate**

```bash
dart run build_runner build --delete-conflicting-outputs
```

- [ ] **Step 4: Update existing handoff tests**

In `app/test/shared/api/handoff_api_client_test.dart`, wherever the test constructs or reads `handoffApiClientProvider`, pass a second argument (e.g. `'test-device-id'`). Add an assertion that `BaseOptions.headers` includes both `Authorization: Bearer <token>` and `X-Craftsky-Device-Id: <deviceId>`.

In `app/test/auth/providers/auth_controller_test.dart`, any override of `handoffApiClientProvider(token)` becomes `handoffApiClientProvider(token, 'test-device-id')`. Any test that overrides `deviceIdProvider` should do so with `overrideWith((ref) => Future.value('test-device-id'))`.

- [ ] **Step 5: Run full test suite**

```bash
flutter test
```

Expected: all pass. If a test is now flaky because it didn't override `deviceIdProvider` and hits real secure storage, add the override.

- [ ] **Step 6: Commit**

```bash
git add app/lib/shared/api/providers/api_client_provider.dart \
        app/lib/shared/api/providers/api_client_provider.g.dart \
        app/lib/auth/providers/auth_controller.dart \
        app/test/shared/api/handoff_api_client_test.dart \
        app/test/auth/providers/auth_controller_test.dart
git commit -m "feat(app): handoff client carries X-Craftsky-Device-Id"
```

### Task 14: Gate bootstrap on deviceIdProvider

**Files:**
- Modify: `app/lib/bootstrap.dart`

- [ ] **Step 1: Extend the probe**

In `app/lib/bootstrap.dart`, find the existing block around lines 68–73:

```dart
final probe = ProviderContainer();
try {
  probe.read(dioProvider);
} finally {
  probe.dispose();
}
```

Change to eagerly resolve `deviceIdProvider`:

```dart
final probe = ProviderContainer();
try {
  probe.read(dioProvider);
  // Eagerly resolve device-id so the first /v1/* request sees it.
  // Strict server-side enforcement means a missed header → 400.
  await probe.read(deviceIdProvider.future);
} finally {
  probe.dispose();
}
```

Add the import near the top: `import 'package:craftsky_app/shared/device/device_id_provider.dart';`.

- [ ] **Step 2: Analyze**

```bash
cd app && dart analyze
```

Expected: clean.

- [ ] **Step 3: Manual boot check**

Run `flutter run` (Android emulator) with the server offline. The bootstrap should complete and `ProviderScope` should build. (We can't smoke-test secure-storage failure here — this step is just a compile/boot sanity check.)

- [ ] **Step 4: Commit**

```bash
git add app/lib/bootstrap.dart
git commit -m "feat(app): bootstrap resolves deviceIdProvider before runApp"
```

---

## Chunk 5: Docs, AGENTS, spec errata

### Task 15: AGENTS.md line + spec errata

**Files:**
- Modify: `AGENTS.md`
- Modify: `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md`

- [ ] **Step 1: Update AGENTS.md**

In `AGENTS.md`, under the "Coding Conventions" section, append a bullet about the API convention. Locate the existing `API:` line (line ~40 at time of plan writing — adjust if moved):

```markdown
- **API:** The HTTP surface between the Flutter app and the AppView is governed by the API architecture spec ...
```

Append (or add a sibling bullet):

```markdown
- **JSON casing:** Every `/v1/*` JSON body uses camelCase keys. `/oauth/*` keeps whatever the atproto OAuth spec dictates. See [2026-04-22-api-wire-alignment-design.md](docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md) §1.
```

- [ ] **Step 2: Add errata to the API architecture spec**

In `docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md` §6 ("Error envelope"), immediately under the heading, insert an errata block:

```markdown
> **Errata (2026-04-22):** This spec's error envelope uses camelCase
> (`error`, `message`, `requestId`). The project-wide JSON key convention
> is formally camelCase across the entire `/v1/*` surface, codified in
> [2026-04-22-api-wire-alignment-design.md](./2026-04-22-api-wire-alignment-design.md) §1.
```

- [ ] **Step 3: Commit**

```bash
git add AGENTS.md docs/superpowers/specs/2026-04-21-appview-api-architecture-design.md
git commit -m "docs: codify camelCase JSON convention in AGENTS + spec errata"
```

---

## Chunk 6: End-to-end smoke test

No automated test; follow the spec's §5 checklist manually against `just dev`.

### Task 16: Run the 11-step smoke test

**Files:** none (manual verification).

- [ ] **Step 1: Clean state**

```bash
just down
just dev
```

From a second shell:

```bash
just psql -c "DELETE FROM craftsky_sessions;"
```

- [ ] **Step 2: Run smoke test steps 1–11 from the spec**

Follow [docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md §5.2](../specs/2026-04-22-api-wire-alignment-design.md).

For steps 3, 4, 6 (curl-based checks), keep the terminal output pasted into the PR description as evidence.

For step 11 (device-ID persistence across sign-out), the concrete assertion is: after sign-out, re-sign-in, then `just psql -c "SELECT last_device_id FROM craftsky_sessions;"` should show the SAME id value as before sign-out. The Flutter app reuses the stored UUID.

- [ ] **Step 3: If any step fails**

Do not paper over. File a new TODO in the plan, fix the root cause, re-run from step 1. The smoke test is the acceptance bar.

- [ ] **Step 4: Record smoke-test evidence in the PR**

When opening the PR, paste the curl outputs from smoke-test steps 3, 4, and 6 into the PR body under a `## Smoke test` heading.

- [ ] **Step 5: Final commit and push**

All work should already be committed at this point. Push the branch and open a PR:

```bash
git push -u origin feat/api-wire-alignment
gh pr create --title "feat: align /v1/* wire protocol between Flutter and AppView" \
             --body "$(cat <<'EOF'
## Summary
- camelCase JSON on /v1/* + all auth endpoints
- /v1/whoami returns {did, handle}; handle resolved via indigo identity directory
- X-Craftsky-Device-Id: regex-validated on server, generated + attached on client
- Bootstrap ordering: device-id provider resolved before runApp

Implements [docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md](docs/superpowers/specs/2026-04-22-api-wire-alignment-design.md).

## Smoke test
(paste step 3 / 4 / 6 curl outputs here)

## Test plan
- [ ] `just test` passes
- [ ] `flutter test` passes
- [ ] `dart analyze` clean
- [ ] Manual sign-in flow on Android emulator against `just dev`

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

---

## Done criteria (mirror of spec §5.3)

- [ ] All 11 smoke-test steps pass.
- [ ] `just test` green.
- [ ] `flutter test` green.
- [ ] `dart analyze` clean.
- [ ] `dart run build_runner build --delete-conflicting-outputs` clean.
- [ ] AGENTS.md has the camelCase convention line.
- [ ] API architecture spec §6 has the camelCase errata.
- [ ] PR is open with smoke-test evidence.

## Post-merge follow-up tickets (not blocking this plan)

These are tracked here for discoverability. Do NOT do them in this plan.

- Full-surface contract test that walks every `/v1/*` route and asserts envelope shape (waits for OpenAPI).
- Per-process handle cache in `HandleResolver` if `whoami` latency shows up in dashboards.
- `ApiException.requestId` field on the Flutter side when the first support workflow needs it.
- Active-sessions UI reading `last_device_id` + `device_label`.
- Per-device rate limiting keyed on `(device_id, endpoint)`.
- Struct-tag linter for `/v1/*` handlers if camelCase drift becomes a review pattern.
