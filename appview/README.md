# appview

The Craftsky App View — a Go service that consumes the atproto firehose via a [Tap](https://github.com/bluesky-social/indigo/tree/main/cmd/tap) sidecar, indexes Craftsky records into Postgres, and serves a JSON/HTTP API to the Flutter client.

Also acts as the **Token Mediating Backend (TMB)** for OAuth with user PDSes.

## Layout

```
appview/
├── cmd/
│   ├── appview/             # server binary (main + NewServer)
│   └── cli/                 # ops & smoke-test CLI (cobra)
├── internal/
│   ├── app/                 # Config, Deps, NewDevDeps/NewProdDeps
│   ├── auth/                # AuthService interface + mock / not-implemented impls
│   ├── db/                  # pgxpool connection wrapper
│   ├── middleware/          # Logging, CORS, Authenticated
│   ├── routes/              # HTTP route registration
│   ├── api/                 # HTTP handler factories
│   ├── tap/                 # WebSocket-with-acks consumer for the Tap sidecar
│   ├── index/               # Indexer interface + BlueskyPostsSample (throwaway)
│   └── models/              # reserved for sqlc-generated types
├── environments/
│   ├── dev.env              # checked in; no secrets
│   └── prod.env.example     # template; real prod.env is gitignored
├── migrations/              # SQL files consumed by golang-migrate/v4
└── queries/                 # SQL files consumed by sqlc
```

## Binaries

### `cmd/appview` — the HTTP server

```
appview dev    # loads environments/dev.env, debug logging, mock auth
appview prod   # loads environments/prod.env, info logging, (future) real OAuth
```

### `cmd/cli` — ops and smoke-test CLI

```
cli ping --env dev              # pings the DB, prints pool stats
cli migrate up|down|status|redo # wraps golang-migrate/v4
cli request GET /whoami --env dev  # hits the running server as the dev DID
cli tap status --env dev           # prints tap connection state (exit 0 connected, 1 disconnected, 2 transport)
cli did-resolve alice.bsky.social --env dev       # stub until the identity resolver lands
```

Exit codes for `cli request`:
- `0` — 2xx response from the server
- `1` — 4xx/5xx response
- `2` — transport error (couldn't reach server)

## Key Dependencies

- [`github.com/bluesky-social/indigo`](https://github.com/bluesky-social/indigo) — source of the Tap sidecar image; indigo Go SDK will be adopted for OAuth (TMB) when that lands
- [`github.com/coder/websocket`](https://github.com/coder/websocket) — WebSocket client for the Tap `/channel` stream
- [`pgx/v5`](https://github.com/jackc/pgx) — Postgres driver + pool
- [`sqlc`](https://sqlc.dev) — SQL → Go codegen — to be adopted once first queries land
- [`cobra`](https://github.com/spf13/cobra) — CLI framework
- [`golang-migrate/v4`](https://github.com/golang-migrate/migrate) — migrations, wrapped by `cmd/cli`
- [`godotenv`](https://github.com/joho/godotenv) — env file loader
- [`uuid`](https://github.com/google/uuid) — per-request run IDs
- `slog` — standard library logging
- `net/http` — standard library router (Go 1.22+ method/path routing is sufficient)

## Development

Local development runs through Docker Compose, driven by the `justfile` at the repo root. See the root `README.md` for a getting-started walkthrough; the compose stack (`postgres`, `migrate`, `tap`, `tap-bootstrap`, `appview`) is brought up with:

```
just dev      # foreground
just dev-d    # detached
just down     # stop (volumes preserved)
```

Common recipes:

```
just test         # go test -race ./... on the host against the compose Postgres
just fmt          # gofmt -w . && go vet ./... on the host
just ping         # cli ping inside the appview container
just tap-status   # cli tap status inside the appview container
just psql         # psql shell against the dev database
just migrate up   # wraps golang-migrate/v4 via the CLI
```

Tests run on the **host** (the appview image has no Go toolchain), so Go must be installed locally and `just dev-d` must already be running — the integration tests connect to the compose Postgres at `localhost:5433` via the `TEST_DATABASE_URL` the recipe sets. (Host port 5433 maps to the container's 5432; this avoids a collision with any native Postgres already bound to 5432.)

## Observability

AppView emits structured JSON logs with a per-request `run_id`, safe route-pattern fields, and no request or response bodies by default. Sensitive headers such as `Authorization`, `Cookie`, `DPoP`, and Craftsky device/session token headers are redacted in request logs.

AppView does not expose a local `/metrics` endpoint. Metrics are recorded through AppView-domain methods and are sent to Sentry Application Metrics only when Sentry metrics are explicitly enabled. With no Sentry DSN, or with metrics disabled, runtime metric calls use a no-op recorder; tests can inject an in-memory recorder.

Metric names are internal ops details and use the `craftsky_appview` prefix where Sentry permits. Current metric families include:

- `craftsky_appview_http_requests_total` counter.
- `craftsky_appview_http_request_duration_seconds` histogram.
- `craftsky_appview_http_response_size_bytes` histogram.
- `craftsky_appview_http_requests_in_flight` gauge.
- `craftsky_appview_db_operation_duration_seconds` histogram for bounded DB operations such as `search.posts`.
- `craftsky_appview_pds_write_duration_seconds` histogram for bounded PDS/OAuth write-proxy operations.
- `craftsky_appview_tap_last_event_age_seconds` gauge for firehose freshness.
- Tap and indexer counters/histograms for bounded firehose processing stages.

Sentry export is disabled unless `SENTRY_DSN` is set. With `SENTRY_DSN` alone, AppView sends only classified errors and recovered panics. Higher-volume pillars are independently gated:

- `SENTRY_LOGS_ENABLED=true` sends safe filtered AppView log entries to Sentry logs. Local stdout JSON logs remain available either way.
- `SENTRY_TRACING_ENABLED=true` exports bounded HTTP, DB, PDS/OAuth, and work-boundary transactions/spans. `SENTRY_TRACES_SAMPLE_RATE` controls normal trace sampling; production defaults conservatively when omitted.
- `SENTRY_METRICS_ENABLED=true` exports AppView metrics to Sentry metrics.
- `SENTRY_TAP_TRACING_ENABLED=true` enables sampled Tap/indexer success-path tracing. `SENTRY_TAP_TRACES_SAMPLE_RATE` controls that volume; forced error/panic Tap spans still bypass the success sampler when tracing is enabled.

Sentry-bound events, logs, spans, and metrics use bounded route patterns, operation names, status/result classes, and sentinel error codes. Panic event values stay redacted. Local stdout logs may retain existing raw error strings where the current local logging policy already allowed them, but Sentry-bound logs and events do not export arbitrary raw error text. This implementation does not configure OpenTelemetry/OTLP export, Prometheus, or breadcrumbs as a primary log stream.

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
| POST | `/auth/logout` | Craftsky clients (behind `Authenticated`) |

### Dev key generation

```
just oauth-keygen
```

Prints a multibase-encoded P-256 private key to stdout. Paste into your
local prod-style `.env` as `OAUTH_CLIENT_SECRET_KEY`. Never commit.

In dev (`OAUTH_HOSTNAME` unset) the appview runs as a public client
against `http://127.0.0.1:18080/oauth/callback` and does not require a
client secret.

The compose stack publishes that callback on host port `18080`. Android
emulators cannot otherwise reach the host through their own loopback address,
so `just app-run-android` installs `adb reverse tcp:18080 tcp:18080` before
launching Flutter. Keep the OAuth callback on `127.0.0.1`: atproto's localhost
client profile does not permit `10.0.2.2` as a redirect host.

### Running the OAuth flow manually (dev)

There is no `cli login` subcommand yet (tracked as future work in the
[OAuth spec §6](../docs/superpowers/specs/2026-04-18-appview-oauth-bff-design.md)).
For now, the flow is driven by `curl` plus a real browser:

**1. Get an authorize URL.** Replace `YOUR_HANDLE` with any atproto
handle you have credentials for:

```bash
curl -s -X POST http://localhost:8080/v1/auth/login \
  -H 'Content-Type: application/json' \
  -H 'X-Craftsky-Device-Id: curl-dev' \
  -d '{"handle":"YOUR_HANDLE","handoffMode":"deep_link"}' | jq -r .authUrl
```

**2. Open the printed URL in a browser.** Sign in at the PDS, approve
the requested scopes (`atproto`, `transition:generic`).

**3. The PDS redirects to `/oauth/callback`.** The appview exchanges
the code for tokens, mints a Craftsky bearer token, and renders an HTML
page. `window.location.replace("craftsky://...")` will silently fail
(no app registered for the scheme) — **expected**. Because dev mode is
on, the page also displays the token in plaintext under
`<code id="devtok">...</code>`. Copy it.

**4. Use the token:**

```bash
TOKEN='<paste-here>'
curl -s \
  -H "Authorization: Bearer $TOKEN" \
  -H 'X-Craftsky-Device-Id: curl-dev' \
  http://localhost:8080/v1/whoami | jq .
# {"did":"did:plc:...","handle":"..."}
```

**5. Logout (single device):**

```bash
curl -is -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H 'X-Craftsky-Device-Id: curl-dev' \
  http://localhost:8080/v1/auth/logout
```

**Or all devices** (revokes the underlying OAuth session, cascades
through all bearer tokens for the DID):

```bash
curl -is -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H 'X-Craftsky-Device-Id: curl-dev' \
  'http://localhost:8080/v1/auth/logout?all=true'
```

### Inspecting OAuth state

```bash
just psql -c 'SELECT account_did, session_id, updated_at FROM oauth_sessions;'
just psql -c 'SELECT encode(token_hash, '\''hex'\''), account_did, oauth_session_id, last_seen_at, revoked_at FROM craftsky_sessions;'
just psql -c "SELECT state, handoff_mode, data->>'request_uri' AS request_uri, age(now(), created_at) AS age FROM oauth_auth_requests;"
```

### Dev-only auth shortcut

For non-OAuth smoke tests, the dev appview also accepts an
`X-Dev-DID` header as a fallback when the bearer token doesn't match a
real Craftsky session. Useful for testing `/v1/whoami`-style endpoints
without doing OAuth first. Note: `X-Craftsky-Device-Id` is required on
every `/v1/*` authenticated call — any value matching `[A-Za-z0-9_-]{1,128}`
is fine.

```bash
curl -s \
  -H 'Authorization: Bearer ignored' \
  -H 'X-Dev-DID: did:plc:example' \
  -H 'X-Craftsky-Device-Id: curl-dev' \
  http://localhost:8080/v1/whoami
```

Note `/v1/whoami` does a live PLC-directory lookup for the handle, so a
made-up DID like `did:plc:example` returns `502 identity_unavailable`.
Use a real DID (e.g. `did:plc:ewvi7nxzyoun6zhxrhs64oiz` → `atproto.com`)
to exercise the happy path.

The fallback is **dev-only** and only fires when the bearer token is
invalid — a real OAuth-issued token always takes precedence.

The `cli request` subcommand sends both `X-Dev-DID` and
`X-Craftsky-Device-Id: cli-dev` automatically in dev mode, so
`cli request GET /v1/whoami` Just Works without extra flags.

## Smoke testing the indexer

End-to-end sanity check: write a record to your real PDS and confirm it
lands in `craftsky_posts` via the firehose → Tap → indexer pipeline. We
use [`goat`](https://github.com/bluesky-social/indigo/tree/main/cmd/goat),
indigo's atproto CLI, to do the PDS write — there is no built-in `cli`
subcommand for this yet.

### Prerequisites

- `goat` installed: `brew install goat` on macOS, or
  `go install github.com/bluesky-social/indigo/cmd/goat@latest`.
- An **app password** for your atproto account. From bsky.social:
  Settings → App Passwords → Add. Make a dedicated one for craftsky-dev
  so you can revoke it independently. App passwords are required by
  goat — it uses the legacy session API, not OAuth.
- You must be **onboarded** to the local appview, i.e. have a row in
  `craftsky_profiles` for your DID. The post indexer drops events from
  non-members silently. If your dev DB has been reset, re-run the OAuth
  flow above (your `social.craftsky.actor.profile` record on the PDS
  triggers Tap's `TAP_SIGNAL_COLLECTION` discovery on first event, but
  the membership row only re-materialises when the OAuth callback fires
  `InitializeProfile`).

### One-time goat login

```bash
goat account login -u YOUR_HANDLE -p YOUR_APP_PASSWORD
```

Goat caches the session under `~/.config/goat/` (or your XDG-config
equivalent). Re-run only if you log out or rotate the app password.

### Write a test post

Put the record in a file rather than echoing inline. `goat record create`
reads `$type` from the JSON to derive the collection — and dotted NSIDs
like `social.craftsky.feed.post` are eagerly auto-linked into markdown
`[text](url)` syntax by many chat UIs and "smart-paste" tools, which
silently corrupts `$type` and produces a record under a malformed
collection name. A single-quoted heredoc bypasses every layer of shell
substitution and "helpful" linkification:

```bash
cat > /tmp/post.json <<'EOF'
{
  "$type": "social.craftsky.feed.post",
  "text": "smoke test",
  "createdAt": "2026-05-04T17:00:00Z"
}
EOF

goat record create --no-validate /tmp/post.json
```

`--no-validate` is required against bsky.social: the PDS doesn't bundle
`social.craftsky.*` lexicons and refuses the write under strict
validation. `--no-validate` sets `validate=false` on the
`com.atproto.repo.createRecord` call so the PDS skips schema validation
for unknown NSIDs.

`goat record create` prints the new record's `at://` URI and CID on
success. The PDS mints a TID-shaped rkey when `--rkey` is omitted — pass
`-r myrkey` if you want a stable key for repeated overwrite tests.

### Verify the row landed

The firehose round-trip is usually under a second:

```bash
just psql -c "
SELECT uri, text, created_at, indexed_at
FROM craftsky_posts
ORDER BY indexed_at DESC
LIMIT 5;"
```

You should see the row at the top. If it doesn't appear within a few
seconds:

```bash
# Confirm the appview's Tap consumer is connected.
just tap-status

# Confirm you're a member.
just psql -c "SELECT did FROM craftsky_profiles WHERE did = 'YOUR_DID';"

# Confirm the record really landed at the canonical NSID on your PDS
# (not a markdown-corrupted variant).
goat record ls --collection social.craftsky.feed.post YOUR_HANDLE
```

If `goat record ls` shows the record but `craftsky_posts` is empty, the
membership gate is the most likely culprit — the indexer drops
non-member posts silently. Re-onboard via the OAuth flow.

### Editing and deleting test records

```bash
# Replace existing record (same rkey, new content).
goat record update --no-validate -r RKEY /tmp/post.json

# Delete.
goat record delete -c social.craftsky.feed.post -r RKEY
```

Records on your real PDS persist across dev DB resets, so periodically
prune old smoke-test records or use a stable rkey (e.g. `-r smoketest`)
that you keep overwriting.

### Why not a `cli` subcommand?

A future `cli post-create` could resume the user's existing OAuth
session via `Deps.NewPDSClient` and write the record without an app
password. That's a better long-term home — same tokens as the rest of
the dev stack, no separate session file. Tracked as future work; until
then, goat is the path of least resistance.

## Why Go

See the reference doc's "Tech Stack" section. TL;DR: contributor accessibility, ecosystem maturity, single static binary deploys, alignment with the atproto Go ecosystem (`indigo`).
