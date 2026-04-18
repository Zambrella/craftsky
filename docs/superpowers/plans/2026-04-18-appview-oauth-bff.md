# AppView OAuth (BFF v1) Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `NotImplementedAuthService` with a real OAuth implementation. AppView acts as a confidential Backend-for-Frontend (BFF) OAuth client against user PDSes, using indigo's `atproto/auth/oauth` library with a Postgres-backed `ClientAuthStore`. Issue Craftsky bearer tokens to clients (Flutter/CLI) that map to OAuth sessions.

**Architecture:** Three tables (`oauth_sessions`, `oauth_auth_requests`, `craftsky_sessions`). Five HTTP endpoints (`/oauth/client-metadata.json`, `/oauth/jwks.json`, `/oauth/callback`, `/auth/login`, `/auth/logout`). All PDS tokens (access, refresh, DPoP key) stay server-side inside indigo-owned JSONB blobs; the client only ever holds an opaque Craftsky bearer token.

**Tech Stack:** Go 1.25, `pgx/v5`, stdlib `net/http`, `slog`, indigo's `atproto/auth/oauth` + `atproto/atcrypto` + `atproto/syntax`, `golang-migrate/v4`.

**Spec:** [docs/superpowers/specs/2026-04-18-appview-oauth-bff-design.md](../specs/2026-04-18-appview-oauth-bff-design.md) — read before starting. All "per spec §N" references point there.

**Note on env var naming.** The spec's prose refers to `CLIENT_HOSTNAME` in one place (§2.6) matching the cookbook. This plan uses the consistent `OAUTH_HOSTNAME` naming throughout — that matches the spec's §5 summary table and keeps every OAuth-related env var under the `OAUTH_*` prefix. If you see `CLIENT_HOSTNAME` in the spec, read it as `OAUTH_HOSTNAME`.

**Working directory:** `/Users/douglastodd/Projects/craftsky` (repo root). Go commands run from `appview/` unless noted. Tests run on the host against the compose Postgres at `localhost:5433` (see `just test`).

---

## Background for a fresh contributor

If you're picking this up cold:

- The appview is a Go HTTP service that indexes atproto records and serves JSON to a Flutter client. It currently has only stub auth (`NotImplementedAuthService` in prod, `MockAuthService` in dev).
- atproto OAuth is a profiled OAuth 2.0 flow with three mandatory extensions: **PAR** (Pushed Authorization Requests — client pushes auth params to the AS before the browser redirect), **DPoP** (proof-of-possession tokens bound to a client-held key, signed per-request), and **`private_key_jwt`** client authentication (confidential clients sign assertions with an ES256 key whose public half lives in a served JWKS).
- indigo (`github.com/bluesky-social/indigo/atproto/auth/oauth`) implements all three. We're not implementing OAuth; we're implementing a **storage backend** (indigo's `ClientAuthStore` interface) and **HTTP handlers** that call into indigo's `ClientApp`.
- The reference implementation is the cookbook example: https://github.com/bluesky-social/cookbook/tree/main/go-oauth-web-app. Its `main.go` and `sqlitestore.go` are the prior art this plan mirrors — with two differences: we use Postgres (not SQLite), and we layer our own **Craftsky session token** on top (their example just uses a signed browser cookie; we need a real bearer token for a mobile/CLI client).
- **Authorization code ≠ access token.** The `code` that lands at `/oauth/callback` is single-use and useless without our client private key. Only indigo's `ProcessCallback` exchanges it for real tokens.
- **BFF means the AppView holds all PDS tokens.** The Flutter client and CLI never talk to the PDS directly in v1. We make PDS calls server-side via `oauthSess.APIClient()`; indigo handles DPoP signing and token refresh. TMB is future work.

Project-wide rules in [AGENTS.md](../../../AGENTS.md) that bind this work:

- `slog` for logging, stdlib `net/http` for routing, `pgx/v5` for Postgres.
- **Integration tests must hit a real database, not mocks** — binding user-memory rule. The `store` tests use the same per-test-schema pattern `BlueskyPostsSample` uses.
- No generic OAuth libraries. indigo's `atproto/auth/oauth` is the only acceptable OAuth dependency.
- `sqlc` is NOT used in the existing codebase and we won't introduce it here — the three tables have narrow, focused queries and raw `pgx` is the project's current standard for non-generated SQL.

### indigo API signatures (pinned for this plan)

Verified against the current indigo source as of 2026-04-18. If the pinned version in `go.mod` diverges, reconcile against that version — but don't silently adapt the plan to a mismatch.

| Symbol | Signature |
|---|---|
| `atcrypto.GeneratePrivateKeyP256` | `() (*PrivateKeyP256, error)` |
| `(*PrivateKeyP256).Multibase` | `() string` (no error) |
| `atcrypto.ParsePrivateMultibase` | `(string) (PrivateKeyExportable, error)` |
| `oauth.NewClientApp` | `(*ClientConfig, ClientAuthStore) *ClientApp` |
| `oauth.NewLocalhostConfig` | `(callbackURL string, scopes []string) ClientConfig` |
| `oauth.NewPublicConfig` | `(clientID, callbackURL string, scopes []string) ClientConfig` |
| `(*ClientConfig).SetClientSecret` | `(atcrypto.PrivateKey, string) error` |
| `(*ClientConfig).IsConfidential` | `() bool` |
| `(*ClientConfig).PublicJWKS` | `() JWKS` |
| `(*ClientConfig).ClientMetadata` | `() ClientMetadata` |
| `(*ClientApp).StartAuthFlow` | `(ctx, identifier string) (string, error)` — returns auth URL |
| `(*ClientApp).ProcessCallback` | `(ctx, url.Values) (*ClientSessionData, error)` — **note: not `CallbackData`** |
| `(*ClientApp).ResumeSession` | `(ctx, syntax.DID, sessionID string) (*ClientSession, error)` |
| `(*ClientApp).Logout` | `(ctx, syntax.DID, sessionID string) error` |
| `syntax.ParseDID` | `(string) (syntax.DID, error)` — use for validated conversion, not bare cast |

---

## File structure

Files this plan creates or modifies, with responsibilities:

**Create:**
- `appview/migrations/000002_oauth_tables.up.sql` + `.down.sql` — Creates/drops `oauth_sessions`, `oauth_auth_requests`, `craftsky_sessions` as a single logical unit (FK makes them inseparable).
- `appview/migrations/000003_oauth_auth_requests_handoff.up.sql` + `.down.sql` — **Conditional.** Only created if Task 2.2's probe concludes indigo's serializer drops unknown JSONB fields. Adds `handoff_mode` and `loopback_redirect_uri` columns to `oauth_auth_requests`.
- `appview/internal/auth/config.go` — Loads OAuth-related env vars; builds indigo's `oauth.ClientConfig` with `NewLocalhostConfig` (dev) or `NewPublicConfig` + `SetClientSecret` (prod).
- `appview/internal/auth/config_test.go`
- `appview/internal/auth/store.go` — Postgres implementation of `oauth.ClientAuthStore` (6 methods). Treats indigo's `ClientSessionData` / `AuthRequestData` as opaque JSONB. Contains `StoreConfig` and the `handoffStorage` interface (see Task 2.2).
- `appview/internal/auth/store_test.go` — Integration tests against compose Postgres via per-test schemas. Includes the mandatory `SaveSession`-on-upsert test (spec §4.5; verifies *our* `updated_at` bump — whether indigo calls `SaveSession` on refresh is validated by the Task 4.3 smoke test).
- `appview/internal/auth/craftsky_session.go` — Generates/hashes/looks up Craftsky bearer tokens. Throttled `last_seen_at` updates. Soft-revoke via `revoked_at`.
- `appview/internal/auth/craftsky_session_test.go`
- `appview/internal/auth/handlers_oauth.go` — `ClientMetadataHandler`, `JWKSHandler`, `CallbackHandler`. The atproto-spec-facing endpoints plus callback handoff.
- `appview/internal/auth/handlers_session.go` — `LoginHandler`, `LogoutHandler`. The Craftsky-client-facing endpoints.
- `appview/internal/auth/handlers_render.go` — Shared response helpers (`writeJSONError`, `renderErrorHTML`, deep-link/loopback callback templates via `html/template`).
- `appview/internal/auth/handlers_test.go` — Shared test fixtures + exported handler-behavior tests (status codes, response bodies, handoff branching). Kept in one file because the fixtures are shared.
- `appview/cmd/cli/oauth_keygen.go` — `cli oauth-keygen` subcommand that generates a P-256 private key in multibase form.
- `appview/cmd/cli/oauth_keygen_test.go`

**Modify:**
- `appview/internal/auth/oauth.go` — Replace `NotImplementedAuthService` with `CraftskyAuthService` (resolves a Bearer token → DID via `craftsky_sessions`).
- `appview/internal/auth/service.go` — Change `AuthService.Authenticate` return type to `AuthInfo{DID, SessionID}`.
- `appview/internal/auth/mock.go` — Return `AuthInfo`.
- `appview/internal/app/config.go` — Add OAuth-related fields (see Task 1.2). Validate in prod that the confidential-client fields are set.
- `appview/internal/app/config_test.go`
- `appview/internal/app/deps.go` — Construct `oauth.ClientApp`, `PostgresAuthStore`, `CraftskySessionStore`; wire into `Deps`. `NewProdDeps` uses `CraftskyAuthService`; `NewDevDeps` keeps `MockAuthService`.
- `appview/internal/app/deps_test.go`
- `appview/internal/middleware/auth.go` — Add `WithOAuthSessionID` / `GetOAuthSessionID` context helpers; inject the session ID returned by the new `AuthService.Authenticate` shape.
- `appview/internal/middleware/auth_test.go`
- `appview/internal/routes/routes.go` — Register the 5 new routes.
- `appview/cmd/cli/main.go` — Register `oauth-keygen` subcommand.
- `appview/environments/dev.env` — Add `OAUTH_*` keys with localhost-mode defaults.
- `appview/environments/prod.env.example` — Add prod-only required `OAUTH_*` keys.
- `justfile` — Add `oauth-keygen` recipe. Also modify the existing `psql` recipe to accept passthrough args (see Task 1.1).
- `appview/README.md` — Document the 5 endpoints, env vars, dev key generation.
- `AGENTS.md` — Add a footnote near rule #2 flagging that TMB will require a wording amendment.

**No deletions.** The sample posts indexer stays; this plan does not touch it.

### Key-source scope note

Spec §2.6 mentions two prod key sources — inline value vs file path. **This plan implements only the inline variant** (`OAUTH_CLIENT_SECRET_KEY` holds a multibase-encoded P-256 key directly). The file-path variant (`OAUTH_CLIENT_SECRET_KEY_PATH`) is deferred to a deploy-time follow-up.

---

## Chunk 1: Migration, config, and client-private-key generation

Goal: Land the three tables, the env vars, and the dev key-generation recipe. At the end of this chunk, `just oauth-keygen` produces a usable ES256 key, `just migrate up` creates the three tables, and `app.LoadConfig` returns OAuth fields. No HTTP handlers yet.

Create a feature branch at the start of the chunk: `git checkout -b feat/oauth-bff`.

### Task 1.1: Write the migration and make `just psql` accept passthrough args

**Files:**
- Create: `appview/migrations/000002_oauth_tables.up.sql`
- Create: `appview/migrations/000002_oauth_tables.down.sql`
- Modify: `justfile`

- [ ] **Step 1: Update the `psql` recipe to accept args**

In `/Users/douglastodd/Projects/craftsky/justfile`, replace the existing `psql` recipe with:

```makefile
# Open a psql session against the dev database, or run one-off commands.
#   just psql                 # interactive shell
#   just psql -c '\d'         # pass -c / other args through to psql
psql *ARGS:
    docker compose exec postgres psql -U craftsky craftsky_dev {{ARGS}}
```

Verify:

```bash
just psql -c '\dt'
```

Expected: lists tables in the dev DB (will include `schema_migrations` plus the `bluesky_posts_sample` table from migration 1).

- [ ] **Step 2: Write the up migration**

Create `appview/migrations/000002_oauth_tables.up.sql`:

```sql
-- OAuth tables for BFF v1. See:
-- docs/superpowers/specs/2026-04-18-appview-oauth-bff-design.md §2.

CREATE TABLE oauth_sessions (
    account_did  TEXT        NOT NULL,
    session_id   TEXT        NOT NULL,
    data         JSONB       NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (account_did, session_id)
);
CREATE INDEX oauth_sessions_updated_at_idx ON oauth_sessions (updated_at);
CREATE INDEX oauth_sessions_created_at_idx ON oauth_sessions (created_at);

CREATE TABLE oauth_auth_requests (
    state       TEXT        NOT NULL PRIMARY KEY,
    data        JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX oauth_auth_requests_created_at_idx ON oauth_auth_requests (created_at);

CREATE TABLE craftsky_sessions (
    token_hash        BYTEA       NOT NULL PRIMARY KEY,
    account_did       TEXT        NOT NULL,
    oauth_session_id  TEXT        NOT NULL,
    device_label      TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at        TIMESTAMPTZ,
    FOREIGN KEY (account_did, oauth_session_id)
        REFERENCES oauth_sessions (account_did, session_id)
        ON DELETE CASCADE
);
CREATE INDEX craftsky_sessions_did_idx ON craftsky_sessions (account_did);
CREATE INDEX craftsky_sessions_last_seen_idx ON craftsky_sessions (last_seen_at);
```

- [ ] **Step 3: Write the down migration**

Create `appview/migrations/000002_oauth_tables.down.sql`:

```sql
-- Drop in reverse dependency order: craftsky_sessions has FK → oauth_sessions.
-- oauth_auth_requests has no dependents; drop it alongside for a clean rollback.
DROP TABLE IF EXISTS craftsky_sessions;
DROP TABLE IF EXISTS oauth_auth_requests;
DROP TABLE IF EXISTS oauth_sessions;
```

- [ ] **Step 4: Apply the migration**

Ensure the compose stack is running, then:

```bash
cd /Users/douglastodd/Projects/craftsky
just dev-d
just migrate up
```

Expected: migration `000002_oauth_tables` reports applied.

- [ ] **Step 5: Verify tables exist**

```bash
just psql -c '\d oauth_sessions'
just psql -c '\d oauth_auth_requests'
just psql -c '\d craftsky_sessions'
```

Expected: three tables shown with the columns/indexes above, and `craftsky_sessions` shows the FK to `oauth_sessions`.

- [ ] **Step 6: Verify rollback works**

```bash
just migrate down 1
just psql -c '\dt'
just migrate up
```

Expected: after `down 1`, the three new tables are gone (only `bluesky_posts_sample` + `schema_migrations` remain). After `up`, tables are back.

- [ ] **Step 7: Commit**

```bash
git add appview/migrations/000002_oauth_tables.up.sql appview/migrations/000002_oauth_tables.down.sql justfile
git commit -m "feat(migrations): add OAuth BFF tables; justfile psql passthrough"
```

### Task 1.2: Add OAuth config fields to `app.Config`

**Files:**
- Modify: `appview/internal/app/config.go`
- Modify: `appview/internal/app/config_test.go`

- [ ] **Step 1: Write failing tests for the new fields**

Add to `appview/internal/app/config_test.go` three new subtests under the existing `TestLoadConfig`:

```go
t.Run("oauth fields dev defaults", func(t *testing.T) {
    // Only required dev vars set; OAUTH_* left at their defaults.
    // Assert cfg.OAuthHostname == "", cfg.OAuthScopes == []string{"atproto","transition:generic"},
    // cfg.OAuthSessionExpiry == 180*24*time.Hour,
    // cfg.OAuthSessionInactivity == 30*24*time.Hour,
    // cfg.OAuthAuthRequestExpiry == 30*time.Minute,
    // cfg.CraftskySessionLastSeenThrottle == 5*time.Minute,
    // cfg.OAuthClientKeyID == "primary".
})

t.Run("oauth required in prod", func(t *testing.T) {
    // env = prod, OAUTH_HOSTNAME set, OAUTH_CLIENT_SECRET_KEY unset.
    // Assert LoadConfig returns an error that mentions OAUTH_CLIENT_SECRET_KEY.
})

t.Run("oauth custom values parsed", func(t *testing.T) {
    // Set all OAUTH_* vars to non-default values.
    // Assert each lands on cfg verbatim.
})
```

- [ ] **Step 2: Run tests, verify failure**

```bash
cd appview && go test ./internal/app/... -run TestLoadConfig -v
```

Expected: new subtests fail.

- [ ] **Step 3: Add fields and parse them**

Extend `Config` in `appview/internal/app/config.go`:

```go
// OAuth-related.
OAuthHostname                   string        // empty in dev (localhost mode)
OAuthClientSecretKey            string        // multibase-encoded P-256 private key; empty in dev
OAuthClientKeyID                string        // default "primary"
OAuthScopes                     []string      // default ["atproto", "transition:generic"]
OAuthSessionExpiry              time.Duration // default 180d
OAuthSessionInactivity          time.Duration // default 30d
OAuthAuthRequestExpiry          time.Duration // default 30m
CraftskySessionLastSeenThrottle time.Duration // default 5m
```

In `LoadConfig`, parse each env var:

```go
cfg.OAuthHostname = os.Getenv("OAUTH_HOSTNAME")
cfg.OAuthClientSecretKey = os.Getenv("OAUTH_CLIENT_SECRET_KEY")
cfg.OAuthClientKeyID = getEnvWithDefault("OAUTH_CLIENT_SECRET_KEY_ID", "primary")

scopesStr := getEnvWithDefault("OAUTH_SCOPES", "atproto transition:generic")
cfg.OAuthScopes = strings.Fields(scopesStr) // space-split, dropping empties

var err error
if cfg.OAuthSessionExpiry, err = durationEnv("OAUTH_SESSION_EXPIRY", 180*24*time.Hour); err != nil {
    return Config{}, err
}
if cfg.OAuthSessionInactivity, err = durationEnv("OAUTH_SESSION_INACTIVITY", 30*24*time.Hour); err != nil {
    return Config{}, err
}
if cfg.OAuthAuthRequestExpiry, err = durationEnv("OAUTH_AUTH_REQUEST_EXPIRY", 30*time.Minute); err != nil {
    return Config{}, err
}
if cfg.CraftskySessionLastSeenThrottle, err = durationEnv("CRAFTSKY_SESSION_LAST_SEEN_THROTTLE", 5*time.Minute); err != nil {
    return Config{}, err
}

// Prod validation: if hostname is set, confidential client requires the key.
if cfg.Env == EnvProd && cfg.OAuthHostname != "" && cfg.OAuthClientSecretKey == "" {
    return Config{}, fmt.Errorf("OAUTH_CLIENT_SECRET_KEY is required in prod when OAUTH_HOSTNAME is set")
}
```

Introduce `getEnvWithDefault(key, def string) string` and `durationEnv(key string, def time.Duration) (time.Duration, error)` as private helpers at the bottom of the file, mirroring the existing `TAP_ACK_TIMEOUT` pattern but DRY. (The current code inlines duration parsing three times — factor it here.) Verify the existing `TAP_*` parsing still passes.

- [ ] **Step 4: Run tests, verify pass**

```bash
cd appview && go test ./internal/app/... -v
```

- [ ] **Step 5: Commit**

```bash
git add appview/internal/app/config.go appview/internal/app/config_test.go
git commit -m "feat(config): add OAuth-related config fields and duration-parsing helper"
```

### Task 1.3: Update env files

**Files:**
- Modify: `appview/environments/dev.env`
- Modify: `appview/environments/prod.env.example`

- [ ] **Step 1: Update `dev.env`**

Append:

```
# OAuth (localhost mode — public client, no client secret).
# OAUTH_HOSTNAME intentionally unset in dev; this triggers oauth.NewLocalhostConfig.
OAUTH_HOSTNAME=
OAUTH_SCOPES=atproto transition:generic
OAUTH_SESSION_EXPIRY=4320h
OAUTH_SESSION_INACTIVITY=720h
OAUTH_AUTH_REQUEST_EXPIRY=30m
CRAFTSKY_SESSION_LAST_SEEN_THROTTLE=5m
```

- [ ] **Step 2: Update `prod.env.example`**

Append:

```
# OAuth (confidential client — required in prod).
# Generate the key locally with `just oauth-keygen` and paste the multibase output.
# Never commit real prod.env.
OAUTH_HOSTNAME=appview.craftsky.social
OAUTH_CLIENT_SECRET_KEY=
OAUTH_CLIENT_SECRET_KEY_ID=primary
OAUTH_SCOPES=atproto transition:generic
OAUTH_SESSION_EXPIRY=4320h
OAUTH_SESSION_INACTIVITY=720h
OAUTH_AUTH_REQUEST_EXPIRY=30m
CRAFTSKY_SESSION_LAST_SEEN_THROTTLE=5m
```

- [ ] **Step 3: Commit**

```bash
git add appview/environments/dev.env appview/environments/prod.env.example
git commit -m "feat(env): add OAuth env keys to dev and prod examples"
```

### Task 1.4: Add the `cli oauth-keygen` subcommand

**Files:**
- Create: `appview/cmd/cli/oauth_keygen.go`
- Create: `appview/cmd/cli/oauth_keygen_test.go`
- Modify: `appview/cmd/cli/main.go`
- Modify: `justfile`

- [ ] **Step 1: Write the failing test**

Create `appview/cmd/cli/oauth_keygen_test.go`:

```go
package main

import (
    "bytes"
    "strings"
    "testing"

    "github.com/bluesky-social/indigo/atproto/atcrypto"
)

func TestOAuthKeygenRoundTrip(t *testing.T) {
    var buf bytes.Buffer
    if err := runOAuthKeygen(&buf); err != nil {
        t.Fatalf("runOAuthKeygen: %v", err)
    }
    out := strings.TrimSpace(buf.String())
    if out == "" {
        t.Fatal("runOAuthKeygen produced empty output")
    }
    priv, err := atcrypto.ParsePrivateMultibase(out)
    if err != nil {
        t.Fatalf("output did not parse as multibase private key: %v", err)
    }
    if _, ok := priv.(*atcrypto.PrivateKeyP256); !ok {
        t.Fatalf("expected P-256 private key, got %T", priv)
    }
}
```

- [ ] **Step 2: Run the test, verify failure**

```bash
cd appview && go test ./cmd/cli/ -run TestOAuthKeygen -v
```

Expected: compile error — `runOAuthKeygen` not defined.

- [ ] **Step 3: Implement `runOAuthKeygen`**

Create `appview/cmd/cli/oauth_keygen.go`:

```go
package main

import (
    "fmt"
    "io"

    "github.com/bluesky-social/indigo/atproto/atcrypto"
    "github.com/spf13/cobra"
)

// oauthKeygenCmd generates a P-256 private key and prints its multibase
// encoding to stdout. Paste the output into your prod-style .env as
// OAUTH_CLIENT_SECRET_KEY. Never commit the key.
func oauthKeygenCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "oauth-keygen",
        Short: "Generate a P-256 private key for OAuth confidential-client auth",
        RunE: func(cmd *cobra.Command, _ []string) error {
            return runOAuthKeygen(cmd.OutOrStdout())
        },
    }
}

func runOAuthKeygen(w io.Writer) error {
    priv, err := atcrypto.GeneratePrivateKeyP256()
    if err != nil {
        return fmt.Errorf("generate key: %w", err)
    }
    // Note: Multibase() returns a single string (no error) per indigo API.
    _, err = fmt.Fprintln(w, priv.Multibase())
    return err
}
```

- [ ] **Step 4: Register the subcommand**

Edit `appview/cmd/cli/main.go` and add beside the other `rootCmd.AddCommand(...)` calls:

```go
rootCmd.AddCommand(oauthKeygenCmd())
```

- [ ] **Step 5: Resolve the indigo dependency**

`indigo/atproto/atcrypto` is already a transitive dependency of `indigo/atproto/auth/oauth` (and will be a direct dep once Chunk 2 lands). Pin it now:

```bash
cd appview && go mod tidy
```

If `atcrypto` doesn't resolve yet (it has to be a direct import), add it explicitly using the same indigo module version already listed in `go.mod`. Do **not** use `@latest` — pin to the version already present, e.g.:

```bash
cd appview && go get github.com/bluesky-social/indigo/atproto/atcrypto
go mod tidy
```

`go get` without a version takes whatever is already referenced indirectly, promoting it to a direct dep without bumping.

- [ ] **Step 6: Run the test, verify pass**

```bash
cd appview && go test ./cmd/cli/ -run TestOAuthKeygen -v
```

- [ ] **Step 7: Add the justfile recipe**

Append to `/Users/douglastodd/Projects/craftsky/justfile`:

```makefile
# Generate a P-256 private key for OAUTH_CLIENT_SECRET_KEY. Prints to stdout.
# Paste into your local prod-style .env; never commit.
oauth-keygen:
    cd appview && go run ./cmd/cli oauth-keygen
```

- [ ] **Step 8: Smoke-test the recipe**

```bash
cd /Users/douglastodd/Projects/craftsky
just oauth-keygen
```

Expected: a single-line multibase string (starts with `z`) prints.

- [ ] **Step 9: Commit**

```bash
git add appview/cmd/cli/oauth_keygen.go appview/cmd/cli/oauth_keygen_test.go appview/cmd/cli/main.go appview/go.mod appview/go.sum justfile
git commit -m "feat(cli): add oauth-keygen subcommand"
```

---

## Chunk 2: `ClientAuthStore` + Craftsky session store

Goal: Implement the three storage layers (indigo's OAuth sessions, indigo's auth requests, our Craftsky sessions). Integration tests against compose Postgres. At the end of this chunk, the stores exist and round-trip real indigo types — but they're not wired into `Deps` yet.

### Task 2.1: Extend the `AuthService` interface and middleware to carry session ID

**Rationale:** The existing `AuthService.Authenticate` returns only a DID. Handlers that need to make PDS calls need `(did, oauth_session_id)`. Rather than add a second lookup step, extend the interface to return both.

**Files:**
- Modify: `appview/internal/auth/service.go`
- Modify: `appview/internal/auth/mock.go`
- Modify: `appview/internal/auth/oauth.go` (the existing stub — rewritten fully in Chunk 3 Task 3.1)
- Modify: `appview/internal/middleware/auth.go`
- Modify: `appview/internal/middleware/auth_test.go`

- [ ] **Step 1: Write a failing middleware test**

Add to `appview/internal/middleware/auth_test.go`:

```go
func TestAuthenticatedInjectsOAuthSessionID(t *testing.T) {
    svc := &fakeAuthSvc{did: "did:plc:xyz", sessID: "sess-123"}
    var gotDID, gotSID string
    next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        gotDID, _ = middleware.GetDID(r.Context())
        gotSID, _ = middleware.GetOAuthSessionID(r.Context())
    })
    h := middleware.Authenticated(svc, noopLogger())(next)
    req := httptest.NewRequest("GET", "/x", nil)
    req.Header.Set("Authorization", "Bearer t")
    h.ServeHTTP(httptest.NewRecorder(), req)
    if gotDID != "did:plc:xyz" || gotSID != "sess-123" {
        t.Fatalf("ctx mismatch: did=%q sid=%q", gotDID, gotSID)
    }
}
```

Define `fakeAuthSvc` in this test file: a type implementing the new `Authenticate(ctx, token) (AuthInfo, error)` signature.

- [ ] **Step 2: Run, verify failure**

```bash
cd appview && go test ./internal/middleware/... -v
```

Expected: compile failures referencing `AuthInfo`, `GetOAuthSessionID`.

- [ ] **Step 3: Change the interface in `service.go`**

Replace:

```go
type AuthService interface {
    Authenticate(ctx context.Context, token string) (did string, err error)
}
```

with:

```go
// AuthInfo carries the authenticated identity and its OAuth session ID.
// Dev/mock implementations return SessionID = "" and consumers must tolerate that.
type AuthInfo struct {
    DID       string
    SessionID string
}

type AuthService interface {
    Authenticate(ctx context.Context, token string) (AuthInfo, error)
}
```

- [ ] **Step 4: Update existing implementations**

- `mock.go`: return `AuthInfo{DID: did}` (SessionID empty).
- `oauth.go` (the current `NotImplementedAuthService`): return `AuthInfo{}, ErrAuthNotImplemented`. This type is removed entirely in Chunk 3 Task 3.1; for now it just has to compile.

- [ ] **Step 5: Add middleware context helpers**

In `appview/internal/middleware/auth.go`, add alongside `didKey`:

```go
const oauthSessionIDKey contextKey = "oauth_session_id"

func GetOAuthSessionID(ctx context.Context) (string, bool) {
    sid, ok := ctx.Value(oauthSessionIDKey).(string)
    return sid, ok
}
```

Update the `authService.Authenticate` call site:

```go
info, err := authService.Authenticate(ctx, token)
if err != nil {
    logger.Warn("auth: Authenticate returned error",
        slog.String("err", err.Error()),
        slog.String("run_id", GetRunID(r.Context())))
    http.Error(w, "Unauthorized", http.StatusUnauthorized)
    return
}
ctx = context.WithValue(ctx, didKey, info.DID)
if info.SessionID != "" {
    ctx = context.WithValue(ctx, oauthSessionIDKey, info.SessionID)
}
next.ServeHTTP(w, r.WithContext(ctx))
```

- [ ] **Step 6: Run middleware tests, verify pass**

```bash
cd appview && go test ./internal/middleware/... -v
```

- [ ] **Step 7: Build the full repo to confirm no other compile breakage**

```bash
cd appview && go build ./...
```

`middleware/auth.go` is the only caller of `Authenticate` outside the auth package. No other changes should be needed. If any new callers appeared since this plan was written, update them.

- [ ] **Step 8: Commit**

```bash
git add appview/internal/auth/service.go appview/internal/auth/mock.go appview/internal/auth/oauth.go appview/internal/middleware/auth.go appview/internal/middleware/auth_test.go
git commit -m "refactor(auth): carry OAuth session ID through AuthService and middleware"
```

### Task 2.2: Probe — decide where `handoff_mode` lives

**Rationale:** Per spec §5 open question #4, we need to decide whether `handoff_mode` and `loopback_redirect_uri` live inside indigo's opaque `AuthRequestData` JSONB or as sibling columns. The decision heuristic from the spec: round-trip and see if indigo preserves unknown keys.

**Files:** No code committed in this task — the output is a decision recorded in Appendix A.

- [ ] **Step 1: Write a temporary probe**

Create `appview/internal/auth/handoff_probe_test.go` (deleted in Step 3):

```go
package auth_test

import (
    "encoding/json"
    "strings"
    "testing"

    "github.com/bluesky-social/indigo/atproto/auth/oauth"
)

func TestProbe_AuthRequestDataPreservesUnknownFields(t *testing.T) {
    raw := `{"state":"probe-state","extra_field":"keep-me"}`
    var d oauth.AuthRequestData
    if err := json.Unmarshal([]byte(raw), &d); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    out, err := json.Marshal(d)
    if err != nil {
        t.Fatalf("marshal: %v", err)
    }
    t.Logf("probe result: %s", string(out))
    if strings.Contains(string(out), "keep-me") {
        t.Log("DECISION: extra_field survived; handoff fields can live inline in JSONB")
    } else {
        t.Log("DECISION: extra_field was dropped; use sibling columns")
    }
}
```

```bash
cd appview && go test ./internal/auth/... -run TestProbe_AuthRequestData -v
```

Read the `t.Log` output.

- [ ] **Step 2: Record the decision in Appendix A**

Scroll to the bottom of this plan file and fill in Appendix A with one of the two templates below.

**Inline variant** (if `keep-me` survived):

```markdown
## Appendix A: handoff storage decision

**Decision:** Inline in JSONB. Extra top-level keys survived indigo's AuthRequestData (un)marshal round-trip.

**Implication:** Task 3.0 is SKIPPED. Handoff fields are stored in the JSONB `data` column via a small wrapper.

**Implementation:** In `appview/internal/auth/store.go`, define adjacent to `SaveAuthRequestInfo`:

    type authRequestWithHandoff struct {
        oauth.AuthRequestData
        HandoffMode         string `json:"handoff_mode,omitempty"`
        LoopbackRedirectURI string `json:"loopback_redirect_uri,omitempty"`
    }

`recordHandoff` in Task 3.5 reads the row (by state), unmarshals into `authRequestWithHandoff`, sets the two fields, re-marshals, UPDATEs. `loadHandoff` in Task 3.6 does the same read + unmarshal path.
```

**Sibling-column variant** (if `keep-me` was dropped):

```markdown
## Appendix A: handoff storage decision

**Decision:** Sibling columns. Extra top-level keys did NOT survive indigo's AuthRequestData round-trip.

**Implication:** Task 3.0 MUST be executed. Migration 000003 adds `handoff_mode TEXT NOT NULL DEFAULT 'deep_link'` and `loopback_redirect_uri TEXT` columns to `oauth_auth_requests`.

**Implementation:** `recordHandoff` in Task 3.5 runs `UPDATE oauth_auth_requests SET handoff_mode=$1, loopback_redirect_uri=$2 WHERE state=$3`. `loadHandoff` in Task 3.6 runs `SELECT handoff_mode, loopback_redirect_uri FROM oauth_auth_requests WHERE state=$1`.
```

- [ ] **Step 3: Delete the probe file and commit the decision**

```bash
rm appview/internal/auth/handoff_probe_test.go
git add docs/superpowers/plans/2026-04-18-appview-oauth-bff.md
git commit -m "plan: record handoff_mode storage decision"
```

### Task 2.3: Implement `PostgresAuthStore`

**Files:**
- Create: `appview/internal/auth/store.go`
- Create: `appview/internal/auth/store_test.go`

- [ ] **Step 1: Add the indigo OAuth dependency**

```bash
cd appview && go get github.com/bluesky-social/indigo/atproto/auth/oauth
go get github.com/bluesky-social/indigo/atproto/syntax
go mod tidy
```

Use bare `go get` (no `@latest`) so the module version matches whatever is already referenced transitively.

- [ ] **Step 2: Write the first failing test — SaveSession + GetSession round-trip**

Create `appview/internal/auth/store_test.go`. Start with shared fixtures:

```go
package auth_test

import (
    "context"
    "fmt"
    "log/slog"
    "math/rand/v2"
    "os"
    "testing"
    "time"

    "github.com/bluesky-social/indigo/atproto/auth/oauth"
    "github.com/bluesky-social/indigo/atproto/syntax"
    "github.com/jackc/pgx/v5/pgxpool"

    "social.craftsky/appview/internal/auth"
)

// withAuthSchema creates a private schema, runs the OAuth DDL inside it,
// and returns a pool scoped to that schema via search_path. Dropped via
// t.Cleanup. Mirrors the withSchema helper in internal/index tests.
func withAuthSchema(t *testing.T) *pgxpool.Pool {
    t.Helper()
    url := os.Getenv("TEST_DATABASE_URL")
    if url == "" {
        url = os.Getenv("DATABASE_URL")
    }
    if url == "" {
        t.Skip("TEST_DATABASE_URL and DATABASE_URL both unset")
    }
    ctx := context.Background()
    bootstrap, err := pgxpool.New(ctx, url)
    if err != nil {
        t.Fatalf("bootstrap pool: %v", err)
    }
    schema := fmt.Sprintf("test_auth_%d", rand.Uint32())
    if _, err := bootstrap.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
        t.Fatalf("create schema: %v", err)
    }
    t.Cleanup(func() {
        _, _ = bootstrap.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
        bootstrap.Close()
    })

    // Full DDL matching migration 000002 (and 000003 if Appendix A chose sibling columns).
    ddl := `
        CREATE TABLE ` + schema + `.oauth_sessions (
            account_did TEXT NOT NULL,
            session_id  TEXT NOT NULL,
            data        JSONB NOT NULL,
            created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
            updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
            PRIMARY KEY (account_did, session_id)
        );
        CREATE TABLE ` + schema + `.oauth_auth_requests (
            state      TEXT NOT NULL PRIMARY KEY,
            data       JSONB NOT NULL,
            created_at TIMESTAMPTZ NOT NULL DEFAULT now()
            /* SIBLING_COLUMNS_PLACEHOLDER */
        );
        CREATE TABLE ` + schema + `.craftsky_sessions (
            token_hash        BYTEA NOT NULL PRIMARY KEY,
            account_did       TEXT NOT NULL,
            oauth_session_id  TEXT NOT NULL,
            device_label      TEXT,
            created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
            last_seen_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
            revoked_at        TIMESTAMPTZ,
            FOREIGN KEY (account_did, oauth_session_id)
                REFERENCES ` + schema + `.oauth_sessions (account_did, session_id)
                ON DELETE CASCADE
        );`
    // If Appendix A chose sibling columns, replace the placeholder with:
    //     , handoff_mode TEXT NOT NULL DEFAULT 'deep_link', loopback_redirect_uri TEXT
    if _, err := bootstrap.Exec(ctx, ddl); err != nil {
        t.Fatalf("create tables: %v", err)
    }

    cfg, _ := pgxpool.ParseConfig(url)
    cfg.ConnConfig.RuntimeParams["search_path"] = schema
    pool, err := pgxpool.NewWithConfig(ctx, cfg)
    if err != nil {
        t.Fatalf("scoped pool: %v", err)
    }
    t.Cleanup(pool.Close)
    return pool
}

func testStoreConfig() auth.StoreConfig {
    return auth.StoreConfig{
        SessionExpiry:     180 * 24 * time.Hour,
        SessionInactivity: 30 * 24 * time.Hour,
        AuthRequestExpiry: 30 * time.Minute,
        Logger:            slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})),
    }
}
```

Then the first test:

```go
func TestStore_SaveGetSession(t *testing.T) {
    pool := withAuthSchema(t)
    store := auth.NewPostgresAuthStore(pool, testStoreConfig())
    ctx := context.Background()

    sess := oauth.ClientSessionData{
        AccountDID: syntax.DID("did:plc:abc"),
        SessionID:  "sess-1",
        HostURL:    "https://pds.example.com",
    }
    if err := store.SaveSession(ctx, sess); err != nil {
        t.Fatalf("SaveSession: %v", err)
    }
    got, err := store.GetSession(ctx, sess.AccountDID, sess.SessionID)
    if err != nil {
        t.Fatalf("GetSession: %v", err)
    }
    if got.HostURL != sess.HostURL {
        t.Fatalf("HostURL: got %q want %q", got.HostURL, sess.HostURL)
    }
}
```

- [ ] **Step 3: Run, verify failure**

```bash
cd appview && go test ./internal/auth/... -run TestStore -v
```

Expected: compile error.

- [ ] **Step 4: Implement the store**

Create `appview/internal/auth/store.go`:

```go
// Package auth contains OAuth storage (oauth.ClientAuthStore impl) and
// Craftsky bearer-token session management.
package auth

import (
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "log/slog"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"

    "github.com/bluesky-social/indigo/atproto/auth/oauth"
    "github.com/bluesky-social/indigo/atproto/syntax"
)

// ErrOAuthSessionNotFound is returned by GetSession / GetAuthRequestInfo
// when the requested row doesn't exist. Callers that need to distinguish
// not-found from other errors use errors.Is.
var ErrOAuthSessionNotFound = errors.New("oauth session/auth-request not found")

// StoreConfig carries TTLs and the logger used for lazy-cleanup errors.
type StoreConfig struct {
    SessionExpiry     time.Duration
    SessionInactivity time.Duration
    AuthRequestExpiry time.Duration
    Logger            *slog.Logger
}

// PostgresAuthStore is a Postgres-backed implementation of
// oauth.ClientAuthStore. The ClientSessionData / AuthRequestData blobs
// are round-tripped as opaque JSONB; this code never inspects them
// beyond what indigo's serializer provides.
//
// Cleanup is lazy, inside the Get methods, matching the indigo cookbook
// example. No separate sweeper in v1.
type PostgresAuthStore struct {
    pool *pgxpool.Pool
    cfg  StoreConfig
}

var _ oauth.ClientAuthStore = (*PostgresAuthStore)(nil)

func NewPostgresAuthStore(pool *pgxpool.Pool, cfg StoreConfig) *PostgresAuthStore {
    if cfg.Logger == nil {
        cfg.Logger = slog.Default()
    }
    return &PostgresAuthStore{pool: pool, cfg: cfg}
}

func (s *PostgresAuthStore) SaveSession(ctx context.Context, sess oauth.ClientSessionData) error {
    data, err := json.Marshal(sess)
    if err != nil {
        return fmt.Errorf("marshal session: %w", err)
    }
    const q = `
        INSERT INTO oauth_sessions (account_did, session_id, data, created_at, updated_at)
        VALUES ($1, $2, $3, now(), now())
        ON CONFLICT (account_did, session_id) DO UPDATE SET
            data = EXCLUDED.data,
            updated_at = now()
    `
    if _, err := s.pool.Exec(ctx, q, sess.AccountDID.String(), sess.SessionID, data); err != nil {
        return fmt.Errorf("upsert session: %w", err)
    }
    return nil
}

func (s *PostgresAuthStore) GetSession(ctx context.Context, did syntax.DID, sessionID string) (*oauth.ClientSessionData, error) {
    s.cleanupSessions(ctx)
    var data []byte
    err := s.pool.QueryRow(ctx,
        `SELECT data FROM oauth_sessions WHERE account_did = $1 AND session_id = $2`,
        did.String(), sessionID).Scan(&data)
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, ErrOAuthSessionNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("select session: %w", err)
    }
    var sess oauth.ClientSessionData
    if err := json.Unmarshal(data, &sess); err != nil {
        return nil, fmt.Errorf("unmarshal session: %w", err)
    }
    return &sess, nil
}

func (s *PostgresAuthStore) DeleteSession(ctx context.Context, did syntax.DID, sessionID string) error {
    _, err := s.pool.Exec(ctx,
        `DELETE FROM oauth_sessions WHERE account_did = $1 AND session_id = $2`,
        did.String(), sessionID)
    return err
}

func (s *PostgresAuthStore) SaveAuthRequestInfo(ctx context.Context, info oauth.AuthRequestData) error {
    data, err := json.Marshal(info)
    if err != nil {
        return fmt.Errorf("marshal auth request: %w", err)
    }
    _, err = s.pool.Exec(ctx,
        `INSERT INTO oauth_auth_requests (state, data) VALUES ($1, $2)`,
        info.State, data)
    if err != nil {
        return fmt.Errorf("insert auth request: %w", err)
    }
    return nil
}

func (s *PostgresAuthStore) GetAuthRequestInfo(ctx context.Context, state string) (*oauth.AuthRequestData, error) {
    s.cleanupAuthRequests(ctx)
    var data []byte
    err := s.pool.QueryRow(ctx,
        `SELECT data FROM oauth_auth_requests WHERE state = $1`, state).Scan(&data)
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, ErrOAuthSessionNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("select auth request: %w", err)
    }
    var info oauth.AuthRequestData
    if err := json.Unmarshal(data, &info); err != nil {
        return nil, fmt.Errorf("unmarshal auth request: %w", err)
    }
    return &info, nil
}

func (s *PostgresAuthStore) DeleteAuthRequestInfo(ctx context.Context, state string) error {
    _, err := s.pool.Exec(ctx,
        `DELETE FROM oauth_auth_requests WHERE state = $1`, state)
    return err
}

// cleanupSessions deletes rows older than SessionExpiry by created_at
// or untouched for SessionInactivity by updated_at. Best-effort; errors
// are logged at WARN and otherwise ignored so cleanup doesn't mask the
// caller's real query.
func (s *PostgresAuthStore) cleanupSessions(ctx context.Context) {
    expiry := time.Now().Add(-s.cfg.SessionExpiry)
    inactivity := time.Now().Add(-s.cfg.SessionInactivity)
    if _, err := s.pool.Exec(ctx,
        `DELETE FROM oauth_sessions WHERE created_at < $1 OR updated_at < $2`,
        expiry, inactivity); err != nil {
        s.cfg.Logger.Warn("oauth_sessions cleanup failed", slog.String("err", err.Error()))
    }
}

func (s *PostgresAuthStore) cleanupAuthRequests(ctx context.Context) {
    cutoff := time.Now().Add(-s.cfg.AuthRequestExpiry)
    if _, err := s.pool.Exec(ctx,
        `DELETE FROM oauth_auth_requests WHERE created_at < $1`, cutoff); err != nil {
        s.cfg.Logger.Warn("oauth_auth_requests cleanup failed", slog.String("err", err.Error()))
    }
}
```

**If Appendix A chose inline JSONB**, also add the `authRequestWithHandoff` wrapper struct described there. If sibling columns, leave `store.go` as above — the handoff UPDATE/SELECT lives in the handlers.

- [ ] **Step 5: Run the first test, verify pass**

```bash
cd appview && just dev-d
cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/auth/... -run TestStore -v
```

- [ ] **Step 6: Write remaining tests**

Add to `store_test.go`:

1. `TestStore_DeleteSession` — save; delete; `GetSession` returns `ErrOAuthSessionNotFound`.
2. `TestStore_SaveGetAuthRequest` — analogous to `TestStore_SaveGetSession`.
3. `TestStore_DeleteAuthRequest` — analogous.
4. `TestStore_ExpiredSessionsCleanedUp` — insert via raw SQL with `created_at = now() - interval '200 days'` (bypass `SaveSession` so we can forge the timestamp). Call `GetSession`; assert `ErrOAuthSessionNotFound` and that the row is gone via `SELECT COUNT(*)`.
5. `TestStore_InactiveSessionsCleanedUp` — insert with `updated_at = now() - interval '60 days'`.
6. `TestStore_ExpiredAuthRequestsCleanedUp` — analogous.
7. `TestStore_SaveSessionUpdatesTimestamp` — save initial session; record `updated_at`; sleep 50ms; `SaveSession` again for the same `(did, sid)`; assert `updated_at` advanced. This validates the upsert's `updated_at = now()` clause. (Whether indigo itself calls `SaveSession` on refresh is verified by the end-to-end smoke test in Task 4.3 — not here.)

- [ ] **Step 7: Run all, verify pass**

```bash
cd appview && TEST_DATABASE_URL=... go test ./internal/auth/... -v
```

- [ ] **Step 8: Commit**

```bash
git add appview/internal/auth/store.go appview/internal/auth/store_test.go appview/go.mod appview/go.sum
git commit -m "feat(auth): Postgres-backed oauth.ClientAuthStore"
```

### Task 2.4: Implement `CraftskySessionStore`

**Files:**
- Create: `appview/internal/auth/craftsky_session.go`
- Create: `appview/internal/auth/craftsky_session_test.go`

- [ ] **Step 1: Write failing tests**

Tests (all use `withAuthSchema`):

1. `TestCraftskySession_Create_ReturnsTokenAndRow` — `Create(ctx, did, sessID, "")` returns a non-empty opaque token; SELECT row exists with matching `token_hash = SHA256(token)`, matching `account_did` + `oauth_session_id`, null `revoked_at`.
2. `TestCraftskySession_Lookup_HappyPath` — Create, then `Lookup(token)` returns the correct `AuthInfo`.
3. `TestCraftskySession_Lookup_Unknown` — `Lookup("never-issued")` returns `ErrCraftskySessionNotFound`.
4. `TestCraftskySession_Lookup_Revoked` — Create, `Revoke(token)`, then `Lookup(token)` returns `ErrCraftskySessionNotFound`.
5. `TestCraftskySession_RevokeAll_SetsRevokedAtOnAllRowsForDID` — seed two `oauth_sessions` rows for the same DID, create one Craftsky token per row, `RevokeAll(did)`, assert both have non-null `revoked_at` and `Lookup` returns not-found for each.
6. `TestCraftskySession_LastSeenThrottled` — Construct the store with `lastSeenThrottle = 1 * time.Hour` so the second Lookup is always within the window. Record `last_seen_at` via raw SQL after Lookup #1 and after Lookup #2; assert they're equal (i.e. the second call did NOT write).
7. `TestCraftskySession_FKCascadeFromOAuthSessionDelete` — Create token for `(did, sessID)`; `DELETE FROM oauth_sessions WHERE ...`; assert via `SELECT COUNT(*) FROM craftsky_sessions WHERE account_did=$1` that the row is gone (don't route through `Lookup` — we want to verify the FK, not infer it).

Each test seeds `oauth_sessions` rows manually via raw SQL when needed (don't require a `PostgresAuthStore` instance to seed dependencies).

- [ ] **Step 2: Run, verify failure**

```bash
cd appview && go test ./internal/auth/... -run TestCraftskySession -v
```

- [ ] **Step 3: Implement `CraftskySessionStore`**

Create `appview/internal/auth/craftsky_session.go`:

```go
package auth

import (
    "context"
    "crypto/rand"
    "crypto/sha256"
    "encoding/base64"
    "errors"
    "fmt"
    "sync"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

var ErrCraftskySessionNotFound = errors.New("craftsky session not found")

type CraftskySessionStore struct {
    pool             *pgxpool.Pool
    lastSeenThrottle time.Duration

    mu             sync.Mutex
    lastSeenMemory map[string]time.Time
}

func NewCraftskySessionStore(pool *pgxpool.Pool, lastSeenThrottle time.Duration) *CraftskySessionStore {
    return &CraftskySessionStore{
        pool:             pool,
        lastSeenThrottle: lastSeenThrottle,
        lastSeenMemory:   make(map[string]time.Time),
    }
}

// Create generates a fresh opaque bearer token, stores its SHA-256 in
// the DB, and returns the plaintext token (shown to the client exactly
// once). deviceLabel is optional; pass "" if none.
func (s *CraftskySessionStore) Create(ctx context.Context, did, oauthSessionID, deviceLabel string) (string, error) {
    raw := make([]byte, 32)
    if _, err := rand.Read(raw); err != nil {
        return "", fmt.Errorf("rand: %w", err)
    }
    token := base64.RawURLEncoding.EncodeToString(raw)
    hash := sha256.Sum256([]byte(token))
    _, err := s.pool.Exec(ctx,
        `INSERT INTO craftsky_sessions (token_hash, account_did, oauth_session_id, device_label) VALUES ($1, $2, $3, $4)`,
        hash[:], did, oauthSessionID, nullableString(deviceLabel))
    if err != nil {
        return "", fmt.Errorf("insert craftsky session: %w", err)
    }
    return token, nil
}

func (s *CraftskySessionStore) Lookup(ctx context.Context, token string) (AuthInfo, error) {
    hash := sha256.Sum256([]byte(token))
    var did, sessID string
    err := s.pool.QueryRow(ctx,
        `SELECT account_did, oauth_session_id FROM craftsky_sessions WHERE token_hash = $1 AND revoked_at IS NULL`,
        hash[:]).Scan(&did, &sessID)
    if errors.Is(err, pgx.ErrNoRows) {
        return AuthInfo{}, ErrCraftskySessionNotFound
    }
    if err != nil {
        return AuthInfo{}, fmt.Errorf("lookup craftsky session: %w", err)
    }
    s.maybeTouchLastSeen(ctx, hash[:])
    return AuthInfo{DID: did, SessionID: sessID}, nil
}

func (s *CraftskySessionStore) Revoke(ctx context.Context, token string) error {
    hash := sha256.Sum256([]byte(token))
    _, err := s.pool.Exec(ctx,
        `UPDATE craftsky_sessions SET revoked_at = now() WHERE token_hash = $1 AND revoked_at IS NULL`,
        hash[:])
    return err
}

func (s *CraftskySessionStore) RevokeAll(ctx context.Context, did string) error {
    _, err := s.pool.Exec(ctx,
        `UPDATE craftsky_sessions SET revoked_at = now() WHERE account_did = $1 AND revoked_at IS NULL`,
        did)
    return err
}

func (s *CraftskySessionStore) maybeTouchLastSeen(ctx context.Context, hash []byte) {
    key := fmt.Sprintf("%x", hash)
    s.mu.Lock()
    last, ok := s.lastSeenMemory[key]
    now := time.Now()
    if ok && now.Sub(last) < s.lastSeenThrottle {
        s.mu.Unlock()
        return
    }
    s.lastSeenMemory[key] = now
    s.mu.Unlock()
    _, _ = s.pool.Exec(ctx,
        `UPDATE craftsky_sessions SET last_seen_at = now() WHERE token_hash = $1`, hash)
}

func nullableString(s string) any {
    if s == "" {
        return nil
    }
    return s
}
```

- [ ] **Step 4: Run tests, verify pass**

```bash
cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test ./internal/auth/... -v
```

- [ ] **Step 5: Commit**

```bash
git add appview/internal/auth/craftsky_session.go appview/internal/auth/craftsky_session_test.go
git commit -m "feat(auth): CraftskySessionStore (opaque bearer tokens, soft revoke, throttled last_seen)"
```

---

## Chunk 3: Real `AuthService`, indigo wiring, HTTP handlers

Goal: Replace `NotImplementedAuthService` with the real implementation, construct `oauth.ClientApp` in `Deps`, and add the five HTTP handlers. At the end of this chunk, GET `/oauth/client-metadata.json` serves a valid document and an authenticated request with a Craftsky bearer token resolves to a DID.

**Pre-chunk orientation:** Skim Appendix A before starting. If it says "inline JSONB", Task 3.0 is SKIPPED and `recordHandoff`/`loadHandoff` implementations go through the JSONB wrapper struct. If "sibling columns", execute Task 3.0 first.

### Task 3.0 (conditional): sibling-column migration

Execute only if Appendix A says "sibling columns". Otherwise skip to Task 3.1.

**Files:**
- Create: `appview/migrations/000003_oauth_auth_requests_handoff.up.sql`
- Create: `appview/migrations/000003_oauth_auth_requests_handoff.down.sql`
- Modify: `appview/internal/auth/store_test.go` (update `withAuthSchema` to include the new columns)

- [ ] **Step 1: Write migrations**

Up:

```sql
ALTER TABLE oauth_auth_requests
    ADD COLUMN handoff_mode TEXT NOT NULL DEFAULT 'deep_link',
    ADD COLUMN loopback_redirect_uri TEXT;
-- Drop the transient default so new rows must supply a value explicitly.
ALTER TABLE oauth_auth_requests ALTER COLUMN handoff_mode DROP DEFAULT;
```

Down:

```sql
ALTER TABLE oauth_auth_requests
    DROP COLUMN handoff_mode,
    DROP COLUMN loopback_redirect_uri;
```

- [ ] **Step 2: Apply**

```bash
just migrate up
```

- [ ] **Step 3: Update `withAuthSchema`**

In `store_test.go`, replace the `/* SIBLING_COLUMNS_PLACEHOLDER */` comment in the DDL with `, handoff_mode TEXT NOT NULL DEFAULT 'deep_link', loopback_redirect_uri TEXT`.

- [ ] **Step 4: Re-run all auth tests**

```bash
cd appview && TEST_DATABASE_URL=... go test ./internal/auth/... -v
```

Expected: all pass (no behavioural change — the new columns have defaults or nullability).

- [ ] **Step 5: Commit**

```bash
git add appview/migrations/000003_*.sql appview/internal/auth/store_test.go
git commit -m "feat(migrations): add handoff_mode + loopback_redirect_uri columns to oauth_auth_requests"
```

### Task 3.1: Replace `NotImplementedAuthService` with `CraftskyAuthService`

**Files:**
- Modify: `appview/internal/auth/oauth.go` (rewrite)
- Create: `appview/internal/auth/oauth_test.go`

- [ ] **Step 1: Write failing tests**

Create `appview/internal/auth/oauth_test.go`:

```go
package auth_test

import (
    "context"
    "errors"
    "testing"

    "social.craftsky/appview/internal/auth"
)

func TestCraftskyAuthService_HappyPath(t *testing.T) {
    pool := withAuthSchema(t)
    // seed oauth_sessions row so FK holds
    if _, err := pool.Exec(context.Background(),
        `INSERT INTO oauth_sessions (account_did, session_id, data) VALUES ('did:plc:a', 's1', '{}')`); err != nil {
        t.Fatal(err)
    }
    store := auth.NewCraftskySessionStore(pool, 5*60*1e9) // 5m as time.Duration
    token, err := store.Create(context.Background(), "did:plc:a", "s1", "")
    if err != nil { t.Fatal(err) }
    svc := &auth.CraftskyAuthService{Store: store}
    info, err := svc.Authenticate(context.Background(), token)
    if err != nil { t.Fatalf("Authenticate: %v", err) }
    if info.DID != "did:plc:a" || info.SessionID != "s1" {
        t.Fatalf("unexpected: %+v", info)
    }
}

func TestCraftskyAuthService_EmptyToken(t *testing.T) {
    svc := &auth.CraftskyAuthService{Store: nil} // Store not touched for empty
    _, err := svc.Authenticate(context.Background(), "")
    if !errors.Is(err, auth.ErrAuthTokenInvalid) {
        t.Fatalf("want ErrAuthTokenInvalid, got %v", err)
    }
}

func TestCraftskyAuthService_RevokedOrUnknownToken(t *testing.T) {
    pool := withAuthSchema(t)
    store := auth.NewCraftskySessionStore(pool, 5*60*1e9)
    svc := &auth.CraftskyAuthService{Store: store}
    _, err := svc.Authenticate(context.Background(), "never-issued")
    if !errors.Is(err, auth.ErrAuthTokenInvalid) {
        t.Fatalf("want ErrAuthTokenInvalid, got %v", err)
    }
}
```

- [ ] **Step 2: Run, verify failure**

```bash
cd appview && go test ./internal/auth/... -run TestCraftskyAuthService -v
```

- [ ] **Step 3: Rewrite `oauth.go`**

Replace the entire file with:

```go
package auth

import (
    "context"
    "errors"
)

// ErrAuthTokenInvalid is returned by CraftskyAuthService.Authenticate
// when the presented bearer token is empty, unknown, or revoked.
// Middleware surfaces it as 401.
var ErrAuthTokenInvalid = errors.New("invalid craftsky session token")

// CraftskyAuthService is the real AuthService used in production. It
// resolves a bearer token to (DID, oauth_session_id) by looking it up
// in the craftsky_sessions table via CraftskySessionStore.
type CraftskyAuthService struct {
    Store *CraftskySessionStore
}

var _ AuthService = (*CraftskyAuthService)(nil)

func (s *CraftskyAuthService) Authenticate(ctx context.Context, token string) (AuthInfo, error) {
    if token == "" {
        return AuthInfo{}, ErrAuthTokenInvalid
    }
    info, err := s.Store.Lookup(ctx, token)
    if errors.Is(err, ErrCraftskySessionNotFound) {
        return AuthInfo{}, ErrAuthTokenInvalid
    }
    if err != nil {
        return AuthInfo{}, err
    }
    return info, nil
}
```

`NotImplementedAuthService` and `ErrAuthNotImplemented` are deleted. `deps.go` still references the old type — that's fixed in Task 3.3.

- [ ] **Step 4: Run tests, verify pass (auth package only)**

```bash
cd appview && TEST_DATABASE_URL=... go test ./internal/auth/... -v
```

- [ ] **Step 5: Commit (the build for `./...` will fail until Task 3.3)**

```bash
git add appview/internal/auth/oauth.go appview/internal/auth/oauth_test.go
git commit -m "feat(auth): CraftskyAuthService backed by craftsky_sessions

internal/app/deps.go still references NotImplementedAuthService and
will not compile until Task 3.3 lands."
```

### Task 3.2: Build the `auth.Config` layer

**Files:**
- Create: `appview/internal/auth/config.go`
- Create: `appview/internal/auth/config_test.go`

- [ ] **Step 1: Write failing tests**

Create `config_test.go`:

```go
package auth_test

import (
    "strings"
    "testing"

    "social.craftsky/appview/internal/auth"
)

func TestBuildClientConfig_Localhost(t *testing.T) {
    cfg, err := auth.BuildClientConfig("", "", "", []string{"atproto"})
    if err != nil { t.Fatal(err) }
    if cfg.IsConfidential() {
        t.Fatal("localhost config should not be confidential")
    }
    if !strings.HasPrefix(cfg.ClientID, "http://localhost?") {
        t.Fatalf("unexpected client_id: %q", cfg.ClientID)
    }
}

func TestBuildClientConfig_Confidential(t *testing.T) {
    // Generate a key inline for the test.
    // Use auth.generateTestKeyMultibase() helper (to be added in config.go
    // test-support section) or just call atcrypto directly here.
    // The point is: hostname + valid multibase key → IsConfidential() true,
    // client_id has the expected shape, PublicJWKS returns one key.
    // See config.go implementation for the expected ClientID format.
}
```

Fill in the confidential test body using `atcrypto.GeneratePrivateKeyP256().Multibase()` directly in the test.

- [ ] **Step 2: Run, verify failure**

- [ ] **Step 3: Implement `BuildClientConfig`**

Create `appview/internal/auth/config.go`:

```go
package auth

import (
    "fmt"

    "github.com/bluesky-social/indigo/atproto/atcrypto"
    "github.com/bluesky-social/indigo/atproto/auth/oauth"
)

// BuildClientConfig produces an indigo oauth.ClientConfig.
//
//   - hostname == ""            → localhost/public client via NewLocalhostConfig
//                                 (callback http://127.0.0.1:8080/oauth/callback).
//   - hostname != "" and key == ""  → public client at that hostname (test scenario).
//   - hostname != "" and key != ""  → confidential client via SetClientSecret.
//
// The key is always multibase-encoded P-256 (matching what `cli oauth-keygen` emits).
func BuildClientConfig(hostname, clientSecretKey, clientKeyID string, scopes []string) (oauth.ClientConfig, error) {
    if hostname == "" {
        return oauth.NewLocalhostConfig("http://127.0.0.1:8080/oauth/callback", scopes), nil
    }
    clientID := fmt.Sprintf("https://%s/oauth/client-metadata.json", hostname)
    callback := fmt.Sprintf("https://%s/oauth/callback", hostname)
    cfg := oauth.NewPublicConfig(clientID, callback, scopes)

    if clientSecretKey == "" {
        return cfg, nil
    }
    priv, err := atcrypto.ParsePrivateMultibase(clientSecretKey)
    if err != nil {
        return oauth.ClientConfig{}, fmt.Errorf("parse OAUTH_CLIENT_SECRET_KEY: %w", err)
    }
    if err := cfg.SetClientSecret(priv, clientKeyID); err != nil {
        return oauth.ClientConfig{}, fmt.Errorf("set client secret: %w", err)
    }
    return cfg, nil
}
```

- [ ] **Step 4: Run tests, verify pass**

- [ ] **Step 5: Commit**

```bash
git add appview/internal/auth/config.go appview/internal/auth/config_test.go
git commit -m "feat(auth): BuildClientConfig helper for localhost and confidential modes"
```

### Task 3.3: Wire `oauth.ClientApp`, stores, and `CraftskyAuthService` into `Deps`

**Files:**
- Modify: `appview/internal/app/deps.go`
- Modify: `appview/internal/app/deps_test.go`

**Approach:** Move `AuthService` construction out of `newDeps` entirely. `NewDevDeps` and `NewProdDeps` each build their own `AuthService` *after* calling `newDeps`, since both now need access to the already-built `CraftskySessionStore`.

- [ ] **Step 1: Add fields to `Deps`**

```go
type Deps struct {
    Config      Config
    Logger      *slog.Logger
    DB          *pgxpool.Pool
    AuthService auth.AuthService

    // OAuth subsystem.
    OAuthApp             *oauth.ClientApp
    OAuthStore           *auth.PostgresAuthStore
    CraftskySessionStore *auth.CraftskySessionStore

    Consumer tap.Consumer
    Indexer  index.Indexer
}
```

Add imports: `"github.com/bluesky-social/indigo/atproto/auth/oauth"`.

- [ ] **Step 2: Change `newDeps` to not take an `AuthService`**

New signature:

```go
func newDeps(ctx context.Context, cfg Config, level slog.Level) (*Deps, func(), error)
```

Inside, after the `db.Connect`, before the indexer wiring, add:

```go
oauthCfg, err := auth.BuildClientConfig(
    cfg.OAuthHostname,
    cfg.OAuthClientSecretKey,
    cfg.OAuthClientKeyID,
    cfg.OAuthScopes,
)
if err != nil {
    pool.Close()
    return nil, nil, fmt.Errorf("build oauth client config: %w", err)
}
oauthStore := auth.NewPostgresAuthStore(pool, auth.StoreConfig{
    SessionExpiry:     cfg.OAuthSessionExpiry,
    SessionInactivity: cfg.OAuthSessionInactivity,
    AuthRequestExpiry: cfg.OAuthAuthRequestExpiry,
    Logger:            logger,
})
oauthApp := oauth.NewClientApp(&oauthCfg, oauthStore)
craftskyStore := auth.NewCraftskySessionStore(pool, cfg.CraftskySessionLastSeenThrottle)
```

And wire them into `deps` (leave `AuthService` nil here — the caller sets it):

```go
deps := &Deps{
    Config:               cfg,
    Logger:               logger,
    DB:                   pool,
    OAuthApp:             oauthApp,
    OAuthStore:           oauthStore,
    CraftskySessionStore: craftskyStore,
    Indexer:              indexerImpl,
    Consumer:             tap.NotImplemented{},
}
```

- [ ] **Step 3: Update `NewDevDeps` and `NewProdDeps`**

```go
func NewDevDeps(ctx context.Context, cfg Config) (*Deps, func(), error) {
    deps, cleanup, err := newDeps(ctx, cfg, slog.LevelDebug)
    if err != nil {
        return nil, nil, err
    }
    deps.AuthService = &auth.MockAuthService{DefaultDID: cfg.DevDID}
    return deps, cleanup, nil
}

func NewProdDeps(ctx context.Context, cfg Config) (*Deps, func(), error) {
    deps, cleanup, err := newDeps(ctx, cfg, slog.LevelInfo)
    if err != nil {
        return nil, nil, err
    }
    deps.AuthService = &auth.CraftskyAuthService{Store: deps.CraftskySessionStore}
    return deps, cleanup, nil
}
```

- [ ] **Step 4: Run tests**

```bash
cd appview && TEST_DATABASE_URL=... go test ./internal/app/... -v
```

If `deps_test.go` asserts `AuthService` type directly, it should already pass — the dev variant still uses `MockAuthService`. Update any assertions on `NotImplementedAuthService` (which no longer exists) to `CraftskyAuthService`.

- [ ] **Step 5: Build the full module**

```bash
cd appview && go build ./...
```

Expected: clean. If anything else referenced `NotImplementedAuthService`, fix now.

- [ ] **Step 6: Commit**

```bash
git add appview/internal/app/deps.go appview/internal/app/deps_test.go
git commit -m "feat(app): wire oauth.ClientApp, PostgresAuthStore, CraftskySessionStore into Deps"
```

### Task 3.4: Shared handler scaffolding + client-metadata and JWKS handlers

**Files:**
- Create: `appview/internal/auth/handlers_render.go`
- Create: `appview/internal/auth/handlers_oauth.go`
- Create: `appview/internal/auth/handlers_test.go`
- Modify: `appview/internal/routes/routes.go`

- [ ] **Step 1: Create `handlers_render.go` with shared helpers**

```go
package auth

import (
    "encoding/json"
    "html/template"
    "log/slog"
    "net/http"
)

// writeJSONError writes a JSON body `{"error":"<code>"}` with the given status.
func writeJSONError(w http.ResponseWriter, status int, code string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)
    _ = json.NewEncoder(w).Encode(map[string]string{"error": code})
}

// renderErrorHTML shows a minimal HTML error page. Used by the OAuth
// callback since it's loaded in a browser, not by a programmatic client.
func renderErrorHTML(w http.ResponseWriter, logger *slog.Logger, status int, userMessage string) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(status)
    _ = errorPageTmpl.Execute(w, errorPageData{Message: userMessage})
}

type errorPageData struct{ Message string }

var errorPageTmpl = template.Must(template.New("err").Parse(`<!doctype html>
<html><head><title>Craftsky — error</title></head><body>
<h1>Sign-in failed</h1>
<p>{{.Message}}</p>
</body></html>`))

// deepLinkCallbackData / loopbackCallbackData / devCallbackData drive
// callback rendering templates declared below. Using html/template gives
// us contextual escaping — important for loopbackURI which comes from
// client input.

type callbackPageData struct {
    Token       string
    DeepLinkURL string // for deep-link mode
    LoopbackURI string // for loopback mode
    DevMode     bool   // if true, also shows the token in plaintext for manual debugging
}

var callbackTmpl = template.Must(template.New("cb").Parse(`<!doctype html>
<html><head><title>Craftsky — signed in</title></head><body>
<p>Signed in. {{if .DeepLinkURL}}Return to the Craftsky app.{{else}}You can close this tab.{{end}}</p>
{{if .DevMode}}<p><strong>Dev-mode token (do not show in prod):</strong> <code id="devtok">{{.Token}}</code></p>{{end}}
<script>
{{if .DeepLinkURL}}
window.location.replace({{.DeepLinkURL}});
{{else if .LoopbackURI}}
fetch({{.LoopbackURI}}, {
    method: "POST",
    headers: {"Content-Type": "application/json"},
    body: JSON.stringify({token: {{.Token}}})
}).finally(function(){ document.body.insertAdjacentHTML("beforeend", "<p>Done.</p>"); });
{{end}}
</script>
</body></html>`))
```

**Why `html/template`:** contextual autoescaping means `{{.LoopbackURI}}` inside a JS string literal gets `js` context escaping (the stdlib applies it automatically when the template is inside a `<script>`). A hostile `loopback_redirect_uri` like `"; evil();//` cannot break out of the JS string.

- [ ] **Step 2: Validate `loopback_redirect_uri` at ingress** (documented here, implemented in Task 3.5)

Defense in depth: even with contextual escaping, we should reject obviously malicious URIs at `/auth/login`. Define in `handlers_render.go`:

```go
import "regexp"

// loopbackRedirectPattern matches the only URI shape our CLI uses:
// http://127.0.0.1:<port>/<path>. Reject anything else at ingress.
var loopbackRedirectPattern = regexp.MustCompile(`^http://127\.0\.0\.1:\d{1,5}(/[A-Za-z0-9._~\-/]*)?$`)
```

- [ ] **Step 3: Define the handlers container in `handlers_oauth.go`**

```go
package auth

import (
    "encoding/json"
    "fmt"
    "log/slog"
    "net/http"

    "github.com/bluesky-social/indigo/atproto/auth/oauth"
    "github.com/jackc/pgx/v5/pgxpool"
)

// HTTPHandlers bundles the OAuth-related HTTP handlers. Construct via
// NewHTTPHandlers; wire the resulting methods into routes.AddRoutes.
type HTTPHandlers struct {
    OAuth            *oauth.ClientApp
    CraftskySessions *CraftskySessionStore
    Pool             *pgxpool.Pool // for handoff read/write
    Logger           *slog.Logger
    DevMode          bool // emits the session token in the callback HTML when true
}

func NewHTTPHandlers(oauthApp *oauth.ClientApp, craftskyStore *CraftskySessionStore, pool *pgxpool.Pool, logger *slog.Logger, devMode bool) *HTTPHandlers {
    return &HTTPHandlers{
        OAuth:            oauthApp,
        CraftskySessions: craftskyStore,
        Pool:             pool,
        Logger:           logger,
        DevMode:          devMode,
    }
}

// ClientMetadataHandler serves /oauth/client-metadata.json — the
// discovery document Authorization Servers fetch to learn about our
// client.
func (h *HTTPHandlers) ClientMetadataHandler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        cfg := &h.OAuth.Config
        meta := cfg.ClientMetadata()
        if cfg.IsConfidential() {
            jwksURL := fmt.Sprintf("https://%s/oauth/jwks.json", r.Host)
            meta.JWKSURI = &jwksURL
        }
        if err := meta.Validate(cfg.ClientID); err != nil {
            h.Logger.Error("client metadata validation failed", slog.String("err", err.Error()))
            http.Error(w, "internal error", http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(meta)
    })
}

// JWKSHandler serves /oauth/jwks.json — the public keys for confidential
// client auth. In dev (public client) this is an empty keys array.
func (h *HTTPHandlers) JWKSHandler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(h.OAuth.Config.PublicJWKS())
    })
}
```

**Note on `&h.OAuth.Config`:** per indigo source, `ClientApp.Config` is a value field of type `ClientConfig`, not a method. We take the address because `ClientMetadata()` and `PublicJWKS()` are defined with pointer receivers in indigo. Confirm this on the pinned version and adjust if needed.

- [ ] **Step 4: Write failing tests in `handlers_test.go`**

```go
package auth_test

import (
    "encoding/json"
    "log/slog"
    "net/http"
    "net/http/httptest"
    "os"
    "strings"
    "testing"

    "github.com/bluesky-social/indigo/atproto/auth/oauth"

    "social.craftsky/appview/internal/auth"
)

// handlersFixture builds a test HTTPHandlers backed by a real
// oauth.ClientApp built from BuildClientConfig, and the Postgres
// test-schema stores.
func handlersFixture(t *testing.T, hostname string) *auth.HTTPHandlers {
    t.Helper()
    pool := withAuthSchema(t)
    cfg, err := auth.BuildClientConfig(hostname, "", "", []string{"atproto", "transition:generic"})
    if err != nil {
        t.Fatal(err)
    }
    store := auth.NewPostgresAuthStore(pool, testStoreConfig())
    oauthApp := oauth.NewClientApp(&cfg, store)
    craftsky := auth.NewCraftskySessionStore(pool, 5*60*1e9)
    logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
    return auth.NewHTTPHandlers(oauthApp, craftsky, pool, logger, true /* devMode */)
}

func TestClientMetadata_Localhost(t *testing.T) {
    h := handlersFixture(t, "")
    rr := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/oauth/client-metadata.json", nil)
    h.ClientMetadataHandler().ServeHTTP(rr, req)
    if rr.Code != 200 {
        t.Fatalf("status: %d body=%s", rr.Code, rr.Body.String())
    }
    var meta oauth.ClientMetadata
    if err := json.NewDecoder(rr.Body).Decode(&meta); err != nil {
        t.Fatal(err)
    }
    if !strings.HasPrefix(meta.ClientID, "http://localhost?") {
        t.Fatalf("client_id: %q", meta.ClientID)
    }
    if !meta.DPoPBoundAccessTokens {
        t.Fatal("DPoPBoundAccessTokens must be true per atproto spec")
    }
}

func TestJWKS_LocalhostEmpty(t *testing.T) {
    h := handlersFixture(t, "")
    rr := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/oauth/jwks.json", nil)
    h.JWKSHandler().ServeHTTP(rr, req)
    if rr.Code != 200 {
        t.Fatal(rr.Code)
    }
    var jwks oauth.JWKS
    if err := json.NewDecoder(rr.Body).Decode(&jwks); err != nil {
        t.Fatal(err)
    }
    if len(jwks.Keys) != 0 {
        t.Fatalf("expected 0 keys in localhost mode, got %d", len(jwks.Keys))
    }
}
```

- [ ] **Step 5: Run, verify failure, fix, verify pass**

- [ ] **Step 6: Register routes**

In `appview/internal/routes/routes.go`, add after the existing `/healthz` registration:

```go
oauthHandlers := auth.NewHTTPHandlers(
    deps.OAuthApp,
    deps.CraftskySessionStore,
    deps.DB,
    deps.Logger,
    deps.Config.Env == app.EnvDev, // devMode
)
mux.Handle("GET /oauth/client-metadata.json", oauthHandlers.ClientMetadataHandler())
mux.Handle("GET /oauth/jwks.json", oauthHandlers.JWKSHandler())
```

Import `"social.craftsky/appview/internal/auth"` if not already.

- [ ] **Step 7: Smoke-test**

```bash
just dev-d
sleep 2  # let appview restart
curl -fsS http://localhost:8080/oauth/client-metadata.json | jq .
curl -fsS http://localhost:8080/oauth/jwks.json | jq .
```

Expected: metadata with `client_id` starting with `http://localhost?`, `dpop_bound_access_tokens: true`. JWKS returns `{"keys":[]}`.

- [ ] **Step 8: Commit**

```bash
git add appview/internal/auth/handlers_render.go appview/internal/auth/handlers_oauth.go appview/internal/auth/handlers_test.go appview/internal/routes/routes.go
git commit -m "feat(auth): /oauth/client-metadata.json and /oauth/jwks.json handlers"
```

### Task 3.5: Handler — `POST /auth/login`

**Files:**
- Create: `appview/internal/auth/handlers_session.go`
- Modify: `appview/internal/auth/handlers_test.go`
- Modify: `appview/internal/routes/routes.go`

**Appendix A branch notes:**
- If Appendix A says **inline JSONB**: `recordHandoff` reads the auth-request row by `state`, unmarshals into `authRequestWithHandoff`, sets the fields, UPDATEs the JSONB column.
- If Appendix A says **sibling columns**: `recordHandoff` runs an `UPDATE oauth_auth_requests SET handoff_mode=$1, loopback_redirect_uri=$2 WHERE state=$3`.

Implementation below shows the sibling-column branch (simpler); adjust to the JSONB wrapper if your Appendix A said inline.

**Race note:** `oauth.ClientApp.StartAuthFlow` INSERTs the auth-request row and returns the authorization URL. We then parse `state` out of that URL and UPDATE the row to set handoff fields. A parallel callback landing between the INSERT and the UPDATE would see the row with defaults. This is acceptable for v1: the default handoff mode is `deep_link`, and a callback arriving within milliseconds of `StartAuthFlow` returning would require the user to authenticate nearly instantly — not realistic. Document in a code comment.

- [ ] **Step 1: Write failing tests**

Add to `handlers_test.go`:

1. `TestLogin_HappyPath_DeepLink` — mock/stub a fake `oauth.ClientApp` somehow... **problem:** `oauth.ClientApp` is a concrete type, not an interface. For login tests we have two options:

   - **Option A:** Test with a real `oauth.ClientApp` pointed at a stub HTTP server impersonating a PDS. Complex.
   - **Option B:** Accept that `/auth/login`'s happy path requires network to a real AS, and cover it only in the end-to-end smoke test (Task 4.3). In handler tests, only cover the ingress validation (400s) and the response shape given a preseeded `oauth_auth_requests` row.

   Pick **Option B**. Write these tests:

```go
func TestLogin_MissingHandle(t *testing.T) { /* POST {} → 400 handle_required */ }
func TestLogin_InvalidHandoffMode(t *testing.T) { /* POST {"handle":"x","handoff_mode":"wat"} → 400 */ }
func TestLogin_LoopbackMissingRedirect(t *testing.T) { /* → 400 */ }
func TestLogin_LoopbackRedirectRejectsNonLoopback(t *testing.T) { /* redirect_uri="https://evil.example/" → 400 */ }
func TestLogin_LoopbackRedirectRejectsJavaScript(t *testing.T) { /* redirect_uri="javascript:alert(1)" → 400 */ }
```

The happy path is covered by Task 4.3.

- [ ] **Step 2: Implement `LoginHandler`**

Create `appview/internal/auth/handlers_session.go`:

```go
package auth

import (
    "context"
    "encoding/json"
    "log/slog"
    "net/http"
    "net/url"
    "strings"

    "github.com/bluesky-social/indigo/atproto/syntax"

    "social.craftsky/appview/internal/middleware"
)

type loginRequest struct {
    Handle              string `json:"handle"`
    HandoffMode         string `json:"handoff_mode"` // "deep_link" | "loopback"
    LoopbackRedirectURI string `json:"loopback_redirect_uri,omitempty"`
}

type loginResponse struct {
    AuthURL string `json:"auth_url"`
}

// LoginHandler starts the OAuth flow and returns the authorization URL.
// The client (Flutter/CLI) opens this URL in the user's system browser.
func (h *HTTPHandlers) LoginHandler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req loginRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            writeJSONError(w, http.StatusBadRequest, "invalid_body")
            return
        }
        req.Handle = strings.TrimPrefix(strings.TrimSpace(req.Handle), "@")
        if req.Handle == "" {
            writeJSONError(w, http.StatusBadRequest, "handle_required")
            return
        }
        if req.HandoffMode != "deep_link" && req.HandoffMode != "loopback" {
            writeJSONError(w, http.StatusBadRequest, "invalid_handoff_mode")
            return
        }
        if req.HandoffMode == "loopback" {
            if req.LoopbackRedirectURI == "" {
                writeJSONError(w, http.StatusBadRequest, "loopback_redirect_uri_required")
                return
            }
            if !loopbackRedirectPattern.MatchString(req.LoopbackRedirectURI) {
                writeJSONError(w, http.StatusBadRequest, "loopback_redirect_uri_invalid")
                return
            }
        }

        authURL, err := h.OAuth.StartAuthFlow(r.Context(), req.Handle)
        if err != nil {
            h.Logger.Warn("StartAuthFlow failed",
                slog.String("handle", req.Handle),
                slog.String("err", err.Error()))
            writeJSONError(w, http.StatusBadGateway, "authorization_server_unavailable")
            return
        }

        state, err := extractState(authURL)
        if err != nil {
            h.Logger.Error("extractState from StartAuthFlow URL", slog.String("err", err.Error()))
            writeJSONError(w, http.StatusInternalServerError, "internal")
            return
        }
        if err := h.recordHandoff(r.Context(), state, req.HandoffMode, req.LoopbackRedirectURI); err != nil {
            // Race note: StartAuthFlow may have INSERTed the row but we
            // can't find it via state — extremely unlikely in practice.
            // Log and continue; fallback handoff in CallbackHandler will
            // take over.
            h.Logger.Error("recordHandoff failed",
                slog.String("state", state),
                slog.String("err", err.Error()))
        }

        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(loginResponse{AuthURL: authURL})
    })
}

func extractState(authURL string) (string, error) {
    u, err := url.Parse(authURL)
    if err != nil {
        return "", err
    }
    s := u.Query().Get("state")
    if s == "" {
        return "", errStateMissing
    }
    return s, nil
}

var errStateMissing = errStrip("authorization URL missing state parameter")
type errStrip string
func (e errStrip) Error() string { return string(e) }

// recordHandoff persists the handoff mode + loopback URI on the
// oauth_auth_requests row identified by state. Sibling-column variant.
// If Appendix A chose inline JSONB, replace this implementation with
// the wrapper-struct-update approach.
func (h *HTTPHandlers) recordHandoff(ctx context.Context, state, mode, loopbackURI string) error {
    _, err := h.Pool.Exec(ctx,
        `UPDATE oauth_auth_requests SET handoff_mode = $1, loopback_redirect_uri = $2 WHERE state = $3`,
        mode, nullableString(loopbackURI), state)
    return err
}

// loadHandoff is the counterpart used by CallbackHandler. Sibling-column variant.
func (h *HTTPHandlers) loadHandoff(ctx context.Context, state string) (mode string, loopbackURI string, err error) {
    var uri *string
    err = h.Pool.QueryRow(ctx,
        `SELECT handoff_mode, loopback_redirect_uri FROM oauth_auth_requests WHERE state = $1`,
        state).Scan(&mode, &uri)
    if uri != nil {
        loopbackURI = *uri
    }
    return
}

// used by LogoutHandler
func (h *HTTPHandlers) oauthLogout(ctx context.Context, did, sessionID string) error {
    parsed, err := syntax.ParseDID(did)
    if err != nil {
        return err
    }
    return h.OAuth.Logout(ctx, parsed, sessionID)
}

func bearerToken(r *http.Request) string {
    h := r.Header.Get("Authorization")
    const p = "Bearer "
    if !strings.HasPrefix(h, p) {
        return ""
    }
    return strings.TrimSpace(strings.TrimPrefix(h, p))
}

// authInfoFromCtx pulls DID and session ID off the context. Assumes
// the request has passed through Authenticated middleware.
func authInfoFromCtx(ctx context.Context) (string, string, bool) {
    did, ok := middleware.GetDID(ctx)
    if !ok {
        return "", "", false
    }
    sid, _ := middleware.GetOAuthSessionID(ctx)
    return did, sid, true
}
```

**For the JSONB-inline variant** (if Appendix A said inline), replace `recordHandoff` and `loadHandoff` with:

```go
func (h *HTTPHandlers) recordHandoff(ctx context.Context, state, mode, loopbackURI string) error {
    var blob []byte
    if err := h.Pool.QueryRow(ctx,
        `SELECT data FROM oauth_auth_requests WHERE state = $1`, state).Scan(&blob); err != nil {
        return err
    }
    var wrap authRequestWithHandoff
    if err := json.Unmarshal(blob, &wrap); err != nil {
        return err
    }
    wrap.HandoffMode = mode
    wrap.LoopbackRedirectURI = loopbackURI
    updated, err := json.Marshal(wrap)
    if err != nil {
        return err
    }
    _, err = h.Pool.Exec(ctx,
        `UPDATE oauth_auth_requests SET data = $1 WHERE state = $2`, updated, state)
    return err
}

func (h *HTTPHandlers) loadHandoff(ctx context.Context, state string) (string, string, error) {
    var blob []byte
    if err := h.Pool.QueryRow(ctx,
        `SELECT data FROM oauth_auth_requests WHERE state = $1`, state).Scan(&blob); err != nil {
        return "", "", err
    }
    var wrap authRequestWithHandoff
    if err := json.Unmarshal(blob, &wrap); err != nil {
        return "", "", err
    }
    return wrap.HandoffMode, wrap.LoopbackRedirectURI, nil
}
```

- [ ] **Step 3: Register**

```go
mux.Handle("POST /auth/login", oauthHandlers.LoginHandler())
```

- [ ] **Step 4: Run tests**

- [ ] **Step 5: Commit**

```bash
git add appview/internal/auth/handlers_session.go appview/internal/auth/handlers_test.go appview/internal/routes/routes.go
git commit -m "feat(auth): POST /auth/login handler with loopback URI validation"
```

### Task 3.6: Handler — `GET /oauth/callback`

**Files:**
- Modify: `appview/internal/auth/handlers_oauth.go`
- Modify: `appview/internal/auth/handlers_test.go`
- Modify: `appview/internal/routes/routes.go`

- [ ] **Step 1: Write failing tests**

For callback, we have the same constraint as login: full happy path requires a real AS. Cover the paths we can:

1. `TestCallback_UnknownState` — no auth-request row for the given state → `ProcessCallback` returns error → HTML error page (status 400).
2. `TestCallback_HandoffFallback` — seed a completed auth-request whose `handoff_mode` is empty, insert a "fake" session into oauth_sessions directly, then call `CallbackHandler` via httptest — **but this requires stubbing ProcessCallback**. Skip this case; cover it in the smoke test.

Practically: only `TestCallback_UnknownState` is a meaningful unit test. The rest is Task 4.3.

- [ ] **Step 2: Implement `CallbackHandler`**

Add to `handlers_oauth.go`:

```go
// CallbackHandler receives the user browser after the PDS authentication
// step. It completes the OAuth dance via indigo, issues a Craftsky
// bearer token, and renders an HTML page that hands the token to the
// client (deep link for mobile/desktop; loopback POST for CLI/dev).
func (h *HTTPHandlers) CallbackHandler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        state := r.URL.Query().Get("state")
        sessData, err := h.OAuth.ProcessCallback(r.Context(), r.URL.Query())
        if err != nil {
            h.Logger.Warn("ProcessCallback failed",
                slog.String("state", state),
                slog.String("err", err.Error()))
            renderErrorHTML(w, h.Logger, http.StatusBadRequest, "Sign-in could not be completed. Please try again.")
            return
        }

        mode, loopbackURI, herr := h.loadHandoff(r.Context(), state)
        if herr != nil {
            // Best-effort: ProcessCallback may already have deleted the
            // auth-request row (it's single-use). Fall back to deep_link.
            h.Logger.Warn("loadHandoff failed; defaulting to deep_link",
                slog.String("state", state),
                slog.String("err", herr.Error()))
            mode = "deep_link"
        }
        if mode == "" {
            mode = "deep_link"
        }

        token, err := h.CraftskySessions.Create(r.Context(), sessData.AccountDID.String(), sessData.SessionID, "")
        if err != nil {
            h.Logger.Error("CraftskySessions.Create failed", slog.String("err", err.Error()))
            renderErrorHTML(w, h.Logger, http.StatusInternalServerError, "Internal error. Please try again.")
            return
        }

        data := callbackPageData{Token: token, DevMode: h.DevMode}
        switch mode {
        case "loopback":
            if loopbackURI == "" {
                renderErrorHTML(w, h.Logger, http.StatusInternalServerError, "Missing loopback redirect URI.")
                return
            }
            // Re-validate at egress (defence in depth; the ingress check
            // in /auth/login is the primary guard).
            if !loopbackRedirectPattern.MatchString(loopbackURI) {
                h.Logger.Error("loopback_redirect_uri failed egress validation",
                    slog.String("uri", loopbackURI))
                renderErrorHTML(w, h.Logger, http.StatusInternalServerError, "Invalid loopback redirect URI.")
                return
            }
            data.LoopbackURI = loopbackURI
        default: // deep_link
            data.DeepLinkURL = "craftsky://auth/complete?token=" + url.QueryEscape(token)
        }

        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        if err := callbackTmpl.Execute(w, data); err != nil {
            h.Logger.Error("callback template", slog.String("err", err.Error()))
        }
    })
}
```

Imports: add `"net/url"` to `handlers_oauth.go`.

- [ ] **Step 3: Register**

```go
mux.Handle("GET /oauth/callback", oauthHandlers.CallbackHandler())
```

- [ ] **Step 4: Run tests, verify pass**

- [ ] **Step 5: Commit**

```bash
git add appview/internal/auth/handlers_oauth.go appview/internal/auth/handlers_test.go appview/internal/routes/routes.go
git commit -m "feat(auth): GET /oauth/callback with deep-link and loopback handoffs"
```

### Task 3.7: Handler — `POST /auth/logout`

**Files:**
- Modify: `appview/internal/auth/handlers_session.go`
- Modify: `appview/internal/auth/handlers_test.go`
- Modify: `appview/internal/routes/routes.go`

**Invariant (clarified from reviewer feedback):** For `?all=true`:
1. Call `oauth.ClientApp.Logout` FIRST. On success, indigo deletes the `oauth_sessions` row, and the FK's `ON DELETE CASCADE` removes all `craftsky_sessions` rows for that `(did, session_id)` automatically.
2. Then call `RevokeAll(did)` as a defensive step. This covers the partial-failure case where `Logout` failed to delete the OAuth session (e.g. AS-side revocation failed) and the cascade therefore didn't fire. If `Logout` succeeded, `RevokeAll` operates on an already-empty result set — a harmless no-op.

- [ ] **Step 1: Write failing tests**

1. `TestLogout_SingleDevice_SetsRevokedAt` — seed oauth_sessions + a Craftsky token, POST with the token, assert `revoked_at IS NOT NULL` on the Craftsky row and the OAuth session still exists.
2. `TestLogout_AllDevices_CascadesRows` — seed two Craftsky tokens for same `(did, sid)`, POST `?all=true` with one of them, assert `oauth.ClientApp.Logout` was effective (oauth_sessions row gone) and both Craftsky rows are gone.
3. `TestLogout_Unauthorized` — no Bearer → 401 (because we run behind `Authenticated`).
4. `TestLogout_AllDevices_OAuthLogoutFailsButRevokeAllRuns` — stub/mock not straightforward with a concrete `*oauth.ClientApp`; cover this manually in smoke test instead, or skip.

For test 2, the `oauth.ClientApp.Logout` call will probably fail in-test because there's no AS — that's fine, it just means the cascade doesn't fire but `RevokeAll` does. Adjust the assertion: "all Craftsky rows are revoked OR gone".

- [ ] **Step 2: Implement `LogoutHandler`**

Add to `handlers_session.go`:

```go
// LogoutHandler revokes the presented Craftsky session. With ?all=true,
// revokes every session for the caller's DID and deletes the underlying
// OAuth session (subject to AS-side revocation success).
func (h *HTTPHandlers) LogoutHandler() http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        did, sid, ok := authInfoFromCtx(r.Context())
        if !ok {
            // Authenticated middleware should have rejected already;
            // defensive 401 here means routing bug.
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        if r.URL.Query().Get("all") == "true" {
            // Step 1: delete the OAuth session. Cascade removes
            // craftsky_sessions rows on success.
            if sid != "" {
                if err := h.oauthLogout(r.Context(), did, sid); err != nil {
                    h.Logger.Warn("oauth.Logout failed; revoke-all will cover",
                        slog.String("did", did),
                        slog.String("session_id", sid),
                        slog.String("err", err.Error()))
                }
            }
            // Step 2: belt-and-braces. If Logout succeeded, the cascade
            // already deleted these rows and RevokeAll is a no-op. If
            // Logout failed, this at least invalidates local tokens.
            if err := h.CraftskySessions.RevokeAll(r.Context(), did); err != nil {
                h.Logger.Error("RevokeAll failed", slog.String("did", did), slog.String("err", err.Error()))
                writeJSONError(w, http.StatusInternalServerError, "internal")
                return
            }
        } else {
            token := bearerToken(r)
            if err := h.CraftskySessions.Revoke(r.Context(), token); err != nil {
                writeJSONError(w, http.StatusInternalServerError, "internal")
                return
            }
        }
        w.WriteHeader(http.StatusNoContent)
    })
}
```

- [ ] **Step 3: Register behind `Authenticated`**

```go
authN := middleware.Authenticated(deps.AuthService, deps.Logger)
mux.Handle("POST /auth/logout", authN(oauthHandlers.LogoutHandler()))
```

(The `authN` middleware already exists above the `/whoami` registration — reuse the same variable.)

- [ ] **Step 4: Run tests**

- [ ] **Step 5: Commit**

```bash
git add appview/internal/auth/handlers_session.go appview/internal/auth/handlers_test.go appview/internal/routes/routes.go
git commit -m "feat(auth): POST /auth/logout handler with single-device and all-devices paths"
```

---

## Chunk 4: Docs, AGENTS.md touch, smoke tests, PR

Goal: Document what landed. Manual smoke test against a real PDS.

### Task 4.1: Update `appview/README.md`

**Files:**
- Modify: `appview/README.md`

- [ ] **Step 1: Add an "OAuth" section**

Under the existing "Binaries" section, add the OAuth block from the previous draft (endpoints table, dev key generation, note on dev vs prod). Full text below for clarity:

```markdown
## OAuth

The appview acts as a confidential Backend-for-Frontend (BFF) OAuth client
against users' PDSes, using [indigo's `atproto/auth/oauth`](https://github.com/bluesky-social/indigo/tree/main/atproto/auth/oauth).
All PDS tokens (access, refresh, DPoP key) stay server-side; clients
present an opaque Craftsky bearer token on every authenticated request.

See [docs/superpowers/specs/2026-04-18-appview-oauth-bff-design.md](../docs/superpowers/specs/2026-04-18-appview-oauth-bff-design.md)
for the full design rationale.

### Endpoints

| Method | Path | Audience |
|---|---|---|
| GET | `/oauth/client-metadata.json` | Authorization Servers |
| GET | `/oauth/jwks.json` | Authorization Servers (empty in dev) |
| GET | `/oauth/callback` | Authorization Server → user browser |
| POST | `/auth/login` | Craftsky clients (Flutter/CLI) |
| POST | `/auth/logout` | Craftsky clients |

### Dev key generation

```
just oauth-keygen
```

Prints a multibase-encoded P-256 private key to stdout. Paste into your
local prod-style `.env` as `OAUTH_CLIENT_SECRET_KEY`. Never commit.

In dev (`OAUTH_HOSTNAME` unset) the appview runs as a public client
against `http://127.0.0.1:8080/oauth/callback` and does not require a
client secret.
```

- [ ] **Step 2: Commit**

```bash
git add appview/README.md
git commit -m "docs(appview): document OAuth endpoints and dev key generation"
```

### Task 4.2: Update `AGENTS.md`

**Files:**
- Modify: `AGENTS.md`

- [ ] **Step 1: Locate rule #2**

Run:

```bash
grep -n "Flutter app never holds PDS tokens" AGENTS.md
```

It's under the "## Architectural Rules" heading, currently numbered "2." Open the file at that line.

- [ ] **Step 2: Add a footnote directly under rule #2**

Insert (immediately after the line ending "private data in Postgres."):

```markdown
   > **Note on the TMB upgrade path:** the current BFF design
   > is consistent with this rule as written. Upgrading to the
   > Token-Mediating Backend pattern (tracked as future work in the
   > OAuth spec) will require amending the rule to distinguish
   > *refresh* tokens (server-only) from short-lived access tokens
   > + DPoP keys (may be handed down to clients).
```

- [ ] **Step 3: Commit**

```bash
git add AGENTS.md
git commit -m "docs(agents): note TMB-upgrade amendment to rule #2"
```

### Task 4.3: End-to-end smoke test (manual)

**Files:** None — output is the smoke-test log appended to Appendix B.

- [ ] **Step 1: Fresh stack**

```bash
cd /Users/douglastodd/Projects/craftsky
just down
just dev-d
just migrate up
sleep 3  # wait for appview to come up
```

- [ ] **Step 2: Curl the discovery endpoints**

```bash
curl -fsS http://localhost:8080/oauth/client-metadata.json | jq .
curl -fsS http://localhost:8080/oauth/jwks.json | jq .
```

Expected: client-metadata with `client_id` starting with `http://localhost?`, `dpop_bound_access_tokens: true`. JWKS: `{"keys":[]}`.

- [ ] **Step 3: Kick off login**

```bash
curl -fsS -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"handle":"bsky.app","handoff_mode":"deep_link"}' | jq .
```

Expected: `{"auth_url":"https://..."}` pointing at `bsky.app`'s PDS authorize endpoint.

- [ ] **Step 4: Complete the browser flow**

Open the `auth_url` in a browser (a real user-agent — not curl). Sign in at the PDS with a real account. You'll be redirected to `/oauth/callback`.

Because dev mode is on, the callback HTML will include a line like `<code id="devtok">...</code>` showing the Craftsky bearer token. Copy it.

(It will also attempt `window.location.replace("craftsky://...")`, which fails silently in the browser because no app is registered for the `craftsky` scheme. That's expected.)

- [ ] **Step 5: Check DB state**

```bash
just psql -c 'SELECT account_did, session_id, updated_at FROM oauth_sessions;'
just psql -c 'SELECT account_did, oauth_session_id, last_seen_at, revoked_at FROM craftsky_sessions;'
just psql -c 'SELECT state, created_at FROM oauth_auth_requests;'
```

Expected:
- One row in `oauth_sessions` with `account_did` matching the DID you signed in with.
- One row in `craftsky_sessions` pointing at the same `(account_did, oauth_session_id)`, `revoked_at` null.
- `oauth_auth_requests` is empty (indigo deletes the row on successful callback).

- [ ] **Step 6: Test authenticated request**

```bash
TOKEN='...'  # paste from Step 4
curl -fsS -H "Authorization: Bearer $TOKEN" http://localhost:8080/whoami | jq .
```

Expected: `{"did":"did:plc:..."}` matching your DID.

- [ ] **Step 7: Test single-device logout**

```bash
curl -is -X POST -H "Authorization: Bearer $TOKEN" http://localhost:8080/auth/logout
# expect: 204
curl -is -H "Authorization: Bearer $TOKEN" http://localhost:8080/whoami
# expect: 401
just psql -c 'SELECT revoked_at IS NOT NULL AS revoked FROM craftsky_sessions;'
# expect: revoked=t, and the oauth_sessions row still exists
```

- [ ] **Step 8: Test `?all=true` logout**

Complete another login dance (Steps 3–4) to get a fresh `TOKEN2`. Then:

```bash
curl -is -X POST -H "Authorization: Bearer $TOKEN2" 'http://localhost:8080/auth/logout?all=true'
# expect: 204
just psql -c 'SELECT COUNT(*) FROM oauth_sessions;'
# expect: 0 (indigo.Logout deleted, cascade fired)
just psql -c 'SELECT COUNT(*) FROM craftsky_sessions;'
# expect: 0
```

- [ ] **Step 9: Validate the `SaveSession`-on-refresh assumption**

To test that indigo bumps `updated_at` on refresh, force a refresh. Options:

- Trigger an authenticated call that exercises `oauthSess.APIClient()` (no such handler exists in v1 — defer to next spec that adds a write proxy).
- Manually: after login, let time pass (~30 min — access token lifetime), then issue an authenticated call that uses the APIClient. Check `updated_at` on the `oauth_sessions` row.

For v1 smoke test, record this as **"deferred to write-proxy spec"**: we won't have a handler exercising `APIClient()` until the write proxy lands.

- [ ] **Step 10: Record results in Appendix B**

Update Appendix B with:
- date, test handle used
- pass/fail for each step above
- any anomalies (e.g., indigo version mismatch, unexpected field names)

```bash
git add docs/superpowers/plans/2026-04-18-appview-oauth-bff.md
git commit -m "plan: record OAuth smoke-test results"
```

### Task 4.4: Open PR

**Files:** None.

- [ ] **Step 1: Run the full suite**

```bash
cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable go test -race ./...
cd appview && gofmt -l .
cd appview && go vet ./...
```

All clean.

- [ ] **Step 2: Push and open PR**

```bash
git push -u origin feat/oauth-bff
gh pr create --title "AppView OAuth (BFF v1)" --body "$(cat <<'EOF'
## Summary
- Real OAuth against user PDSes via indigo's `atproto/auth/oauth`
- Postgres-backed `ClientAuthStore` + Craftsky bearer-token store
- Five HTTP endpoints (client-metadata, jwks, callback, login, logout)
- Replaces `NotImplementedAuthService` with `CraftskyAuthService`

Spec: docs/superpowers/specs/2026-04-18-appview-oauth-bff-design.md
Plan: docs/superpowers/plans/2026-04-18-appview-oauth-bff.md

## Test plan
- [ ] All unit + integration tests pass: `just test`
- [ ] `curl /oauth/client-metadata.json` returns a valid discovery doc
- [ ] `curl /oauth/jwks.json` returns `{"keys":[]}` in dev
- [ ] `POST /auth/login` with `{"handle":"bsky.app","handoff_mode":"deep_link"}` returns a PDS authorize URL
- [ ] Full OAuth dance against `bsky.app` completes; rows appear in `oauth_sessions` and `craftsky_sessions`
- [ ] `/whoami` with the issued Craftsky bearer token returns the DID
- [ ] Single-device `/auth/logout` revokes the token; subsequent `/whoami` returns 401
- [ ] `?all=true` logout deletes the OAuth session and cascades Craftsky sessions
- [ ] Loopback URI validation rejects non-localhost schemes at `/auth/login`

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

Mark the boxes during review as each item is verified.

---

## Appendix A: handoff storage decision

*Populated by Task 2.2.*

- **Decision:** _TBD_
- **Probe output:**
- **Implementation branch:** _see Task 2.2 step 2 for the two templates_

## Appendix B: smoke-test log

*Populated by Task 4.3.*

- **Date:**
- **Stack:**
- **Test handle used:**
- **Results:**

---

## Future work (carried forward from the spec §6)

Not in this plan — listed so implementers don't accidentally build them.

1. TMB upgrade (access-token + DPoP-key handoff to clients).
2. Write-proxy endpoint for `social.craftsky.feed.post`.
3. Blob upload proxying.
4. Client-key rotation.
5. App-layer encryption of `oauth_sessions.data`.
6. Active-session management UI.
7. Sweeper process for revoked sessions.
8. CLI `login` / `logout` subcommands.
9. Rate limiting on auth endpoints.
10. `OAUTH_CLIENT_SECRET_KEY_PATH` file-mounted variant (deferred deploy-time concern).
