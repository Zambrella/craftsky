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
against `http://127.0.0.1:8080/oauth/callback` and does not require a
client secret.

### Running the OAuth flow manually (dev)

There is no `cli login` subcommand yet (tracked as future work in the
[OAuth spec §6](../docs/superpowers/specs/2026-04-18-appview-oauth-bff-design.md)).
For now, the flow is driven by `curl` plus a real browser:

**1. Get an authorize URL.** Replace `YOUR_HANDLE` with any atproto
handle you have credentials for:

```bash
curl -s -X POST http://localhost:8080/auth/login \
  -H 'Content-Type: application/json' \
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
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8080/whoami | jq .
# {"did":"did:plc:...","handle":"..."}
```

**5. Logout (single device):**

```bash
curl -is -X POST -H "Authorization: Bearer $TOKEN" http://localhost:8080/auth/logout
```

**Or all devices** (revokes the underlying OAuth session, cascades
through all bearer tokens for the DID):

```bash
curl -is -X POST -H "Authorization: Bearer $TOKEN" \
  'http://localhost:8080/auth/logout?all=true'
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
real Craftsky session. Useful for testing `/whoami`-style endpoints
without doing OAuth first:

```bash
curl -s -H 'Authorization: Bearer ignored' \
       -H 'X-Dev-DID: did:plc:example' http://localhost:8080/whoami
```

The fallback is **dev-only** and only fires when the bearer token is
invalid — a real OAuth-issued token always takes precedence.

## Why Go

See the reference doc's "Tech Stack" section. TL;DR: contributor accessibility, ecosystem maturity, single static binary deploys, alignment with the atproto Go ecosystem (`indigo`).
