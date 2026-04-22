# AppView OAuth (BFF v1) — design

**Date:** 2026-04-18
**Status:** proposed
**Scope:** OAuth data model + flow shape for the AppView. Replaces `NotImplementedAuthService`.

## Summary

Wire real atproto OAuth into the AppView as a confidential Backend-for-Frontend (BFF) client, backed by [`indigo/atproto/auth/oauth`](https://github.com/bluesky-social/indigo/tree/main/atproto/auth/oauth) and a Postgres implementation of indigo's `ClientAuthStore`. AppView holds all PDS tokens; the Flutter client and the ops CLI present an opaque Craftsky bearer token that maps to an OAuth session row.

This spec covers the **OAuth handshake and storage** only. The write-proxy endpoint that will eventually let clients create `social.craftsky.feed.post` records is deliberately a separate, follow-up spec.

## Goals

1. Replace `NotImplementedAuthService` with a real implementation backed by Craftsky bearer tokens.
2. Persist atproto OAuth state (sessions + in-flight auth requests) in Postgres via indigo's `ClientAuthStore` interface.
3. Serve the five HTTP endpoints required to run the OAuth flow and let a client authenticate against the AppView.
4. Keep all PDS tokens (access, refresh, DPoP keypair) server-side. The client only ever holds an opaque Craftsky token.
5. Leave a clean on-ramp for the Token-Mediating Backend (TMB) upgrade later.

## Non-goals

- **TMB pattern.** v1 is pure BFF — every PDS call is proxied through the AppView. TMB is tracked as future work (§6).
- **Write-proxy endpoints.** Creating `social.craftsky.feed.post` through the AppView is its own spec. This one only guarantees that authenticated handlers can obtain a DPoP-signing indigo `APIClient`.
- **Blob upload proxying.** `com.atproto.repo.uploadBlob` is explicit future work; it's the most likely trigger for upgrading to TMB.
- **Client-key rotation.** v1 supports one ES256 private key. Rotating it forces users to re-authenticate (§6).
- **App-layer encryption** of `oauth_sessions.data`. Deferred (§4, §6).
- **Admin UI / active-session management.** No "signed in on these devices" endpoints yet.
- **CLI `login`/`logout` subcommands.** The HTTP endpoints are in scope; the CLI commands follow in a separate spec once we know the deep-link and loopback details.
- **Rate limiting on the auth endpoints.** Called out here but not designed.

## 1. Architecture

AppView is a confidential OAuth client with a stable `client_id` URL that serves its own metadata document. It holds a single ES256 private key (the "client secret" in atproto parlance) used to sign `private_key_jwt` client assertions against PDS Authorization Servers.

```
┌─────────────────┐                         ┌───────────────────────────┐
│  Flutter / CLI  │ ─── Craftsky session ──▶│          AppView          │
│                 │     token (Bearer)      │                           │
│  holds ONLY:    │                         │  ┌─────────────────────┐  │
│  - Craftsky     │ ◀── JSON responses      │  │  indigo oauth       │  │
│    session tok  │                         │  │  ClientApp          │  │
│                 │                         │  │  (StartAuthFlow,    │  │
│                 │                         │  │   ProcessCallback,  │  │
│                 │                         │  │   ResumeSession,    │  │
│                 │                         │  │   Logout)           │  │
└─────────────────┘                         │  └──────────┬──────────┘  │
                                            │             │             │
                                            │  ┌──────────▼──────────┐  │
                                            │  │ ClientAuthStore     │  │
                                            │  │ (our Postgres impl) │  │
                                            │  └─────────────────────┘  │
                                            │  ┌─────────────────────┐  │
                                            │  │ craftsky_sessions   │  │
                                            │  │ (our own table)     │  │
                                            │  └─────────────────────┘  │
                                            │             │             │
                                            │  (APIClient(), DPoP-signed)
                                            │             ▼             │
                                            └─────────────┼─────────────┘
                                                          ▼
                                                  ┌───────────────┐
                                                  │  user's PDS   │
                                                  └───────────────┘
```

Three logical components, all inside `appview/internal/auth/`:

1. **indigo OAuth subsystem.** `oauth.ClientApp` drives the state machine: PAR, authorization-code exchange, DPoP nonce handling, token refresh, logout. It takes a `ClientConfig` (our client metadata) and a `ClientAuthStore` (our Postgres impl).
2. **Postgres `ClientAuthStore`.** Our 6-method implementation of indigo's interface, backed by `oauth_sessions` and `oauth_auth_requests`. The `data` columns hold indigo's `ClientSessionData` / `AuthRequestData` as JSONB. **We never interpret the contents of `data` in v1.**
3. **Craftsky session layer.** Our own `craftsky_sessions` table mapping an opaque bearer token (SHA-256 of it, actually) to an OAuth session. This is what `Authenticated` middleware resolves on every authenticated request.

### Confirmation of AGENTS.md rule #2

The project rule "Flutter app never holds PDS tokens … OAuth access/refresh tokens live in the App View's sessions table" is literally satisfied by BFF. No doc reconciliation is needed now. If and when we upgrade to TMB (§6), the rule's wording will need a small amendment because short-lived access tokens and DPoP material will be handed down to clients.

## 2. Data model

Three tables, all in the default `public` schema, created by one migration.

### 2.1 `oauth_sessions` — indigo-opaque session blobs

```sql
CREATE TABLE oauth_sessions (
    account_did  TEXT        NOT NULL,
    session_id   TEXT        NOT NULL,
    data         JSONB       NOT NULL,  -- serialized oauth.ClientSessionData
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (account_did, session_id)
);
CREATE INDEX oauth_sessions_updated_at_idx ON oauth_sessions (updated_at);
CREATE INDEX oauth_sessions_created_at_idx ON oauth_sessions (created_at);
```

- `data` holds the refresh token, DPoP keypair, access token, PDS URL, scopes, DPoP nonce, `kid`, etc., encoded by indigo's serializer. The column is opaque to our code.
- Cleanup is **lazy, inside `GetSession`**, matching the cookbook example: rows older than `OAUTH_SESSION_EXPIRY` by `created_at`, or untouched for `OAUTH_SESSION_INACTIVITY` by `updated_at`, are deleted before the query runs. No separate sweeper process in v1.
- Defaults: `OAUTH_SESSION_EXPIRY=180d` (matches the atproto-spec refresh-token cap), `OAUTH_SESSION_INACTIVITY=30d`.

### 2.2 `oauth_auth_requests` — in-flight PAR state

```sql
CREATE TABLE oauth_auth_requests (
    state       TEXT        NOT NULL PRIMARY KEY,
    data        JSONB       NOT NULL,  -- serialized oauth.AuthRequestData
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX oauth_auth_requests_created_at_idx ON oauth_auth_requests (created_at);
```

- `state` is the OAuth `state` parameter; indigo generates it in `StartAuthFlow` and looks it up in `ProcessCallback`.
- Default: `OAUTH_AUTH_REQUEST_EXPIRY=30m`. Cleanup is lazy inside `GetAuthRequestInfo`.

### 2.3 `craftsky_sessions` — our opaque-token bearer

```sql
CREATE TABLE craftsky_sessions (
    token_hash           BYTEA       NOT NULL PRIMARY KEY,  -- SHA-256 of the bearer token
    account_did          TEXT        NOT NULL,
    oauth_session_id     TEXT        NOT NULL,
    device_label         TEXT,                              -- optional, user-facing
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at           TIMESTAMPTZ,
    FOREIGN KEY (account_did, oauth_session_id)
        REFERENCES oauth_sessions (account_did, session_id)
        ON DELETE CASCADE
);
CREATE INDEX craftsky_sessions_did_idx ON craftsky_sessions (account_did);
CREATE INDEX craftsky_sessions_last_seen_idx ON craftsky_sessions (last_seen_at);
```

- **Token format:** 32 random bytes, base64url-encoded. Returned to the client once at creation. We store only `SHA-256(token)` in `token_hash`; the plaintext token is never persisted.
- **Lookup:** `SELECT … WHERE token_hash = $1 AND revoked_at IS NULL`.
- **`last_seen_at`:** updated opportunistically, throttled to at most one write per row per `CRAFTSKY_SESSION_LAST_SEEN_THROTTLE` (default `5m`), so write load is proportional to real activity rather than request rate.
- **`revoked_at`:** soft-delete. Logout sets this. Rows stay for audit; a later spec can add hard-delete after a retention window.
- **Revocation decoupling:** revoking one `craftsky_sessions` row (one device) does **not** invalidate the OAuth session — other devices keep working. Full logout (see §3.5) takes a different path: it deletes the OAuth session via `oauth.ClientApp.Logout`, and the FK's `ON DELETE CASCADE` removes the dependent `craftsky_sessions` rows.

### 2.4 What deliberately is **not** in the schema

- **No separate columns for access token, refresh token, DPoP key, PDS URL.** They live inside `oauth_sessions.data`, owned by indigo.
- **No `oauth_client_keys` table.** v1 uses one ES256 key read from config (see §2.6). Multi-key support is future work (§6).
- **No audit log.** Deferred.
- **No "active sessions" view.** Deferred until the Flutter app has a profile/settings screen.

### 2.5 Encryption at rest

**Not implemented at the application layer in v1.** Rationale:

- The most sensitive item is the refresh token; it already has a 180-day cap.
- Compromise requires both DB access *and* the client private key. The client key is config (file / secret source), not in Postgres. A DB dump alone is not catastrophic.
- App-layer encryption of `oauth_sessions.data` would require us to unmarshal and re-marshal indigo's blob, crossing an abstraction we currently treat as opaque.
- Volume-level or transparent DB encryption belongs to ops/infra and is deferred.

Tracked in §6 as future work.

### 2.6 Client private key — config, not DB

Environment-specific:

- **Dev / localhost mode** (`CLIENT_HOSTNAME` unset): public client via `oauth.NewLocalhostConfig`. No client secret. No `/oauth/jwks.json` endpoint (or it returns an empty JWKS). Loopback callback `http://127.0.0.1:8080/oauth/callback`.
- **Prod** (`CLIENT_HOSTNAME=appview.craftsky.social`): confidential client via `oauth.NewPublicConfig` + `ClientConfig.SetClientSecret`. Key source is one of:
  - PEM content via env var (`OAUTH_CLIENT_SECRET_KEY`), or
  - File path (`OAUTH_CLIENT_SECRET_KEY_PATH`) — the path itself comes from config, the file is mounted from a secret volume.

The secret-source decision is a deployment/ops concern and is explicitly deferred. The spec only requires that **one of the two variants** is implemented in v1.

- **Key type:** P-256 (ES256). Non-negotiable per atproto spec.
- **`OAUTH_CLIENT_SECRET_KEY_ID`:** string, arbitrary, default `primary`. Appears as `kid` in the served JWKS and in client assertion JWTs.

## 3. OAuth flow & endpoints

### 3.1 Endpoint surface

> **Errata (2026-04-21):** The Craftsky-internal auth endpoints
> moved under `/v1/` as part of the API architecture spec
> ([`2026-04-21-appview-api-architecture-design.md`](./2026-04-21-appview-api-architecture-design.md)).
> OAuth AS-facing endpoints (`/oauth/*`) are unchanged.

| Method | Path | Audience | Purpose |
|---|---|---|---|
| GET | `/oauth/client-metadata.json` | Authorization Server | Serves the client metadata doc. URL must equal `client_id`. |
| GET | `/oauth/jwks.json` | Authorization Server | Public JWKS (prod/confidential only; empty in dev). |
| GET | `/oauth/callback` | Authorization Server → user browser | Final leg of auth flow; exchanges code for tokens. |
| POST | `/v1/auth/login` | Craftsky client (Flutter/CLI) | Starts the OAuth flow for a given handle. Returns the authorization URL. |
| POST | `/v1/auth/logout` | Craftsky client (Flutter/CLI) | Revokes the Craftsky session; optionally full logout. |

The `/oauth/*` paths are the public atproto-spec surface; `/v1/auth/*` paths are Craftsky-internal.

### 3.2 Login flow

```
┌─────────────┐      ┌─────────────┐     ┌──────────────────┐     ┌──────────┐
│ Flutter/CLI │      │  AppView    │     │  User's Browser  │     │   PDS    │
│             │      │             │     │                  │     │   (AS)   │
└──────┬──────┘      └──────┬──────┘     └────────┬─────────┘     └────┬─────┘
       │                    │                     │                    │
       │ 1. POST /auth/login                      │                    │
       │    {handle, handoff_mode,                │                    │
       │     loopback_redirect_uri?}              │                    │
       ├───────────────────▶│                     │                    │
       │                    │ 2. StartAuthFlow()                       │
       │                    │    - resolves handle → DID → PDS         │
       │                    │    - fetches AS metadata                 │
       │                    │    - PAR (private_key_jwt + DPoP)        │
       │                    │    - stores oauth_auth_requests row      │
       │                    ├─────────────────────────────────────────▶│
       │                    │◀─────────────── {request_uri} ───────────┤
       │                    │                                          │
       │ 3. 200 {auth_url}  │                                          │
       │◀───────────────────┤                                          │
       │                    │                                          │
       │ 4. open auth_url in system browser                            │
       ├───────────────────────────────────▶│                          │
       │                                    │ 5. user authenticates    │
       │                                    ├─────────────────────────▶│
       │                                    │◀────── redirect to AppView
       │                                    │                          │
       │                    │ 6. GET /oauth/callback?code=…&state=…    │
       │                    │◀───────────────┤                          │
       │                    │ 7. ProcessCallback() → SaveSession()     │
       │                    │    - code → tokens (DPoP-bound)          │
       │                    ├─────────────────────────────────────────▶│
       │                    │◀──── {access, refresh, DPoP-bound} ──────┤
       │                    │                                          │
       │                    │ 8. create craftsky_sessions row           │
       │                    │ 9. callback page hands token to client   │
       │                    │    (deep link or loopback — see §3.3)    │
       │                    │───────────────▶│                          │
       │                    │                                          │
       │ 10. client persists Craftsky token                             │
       │                                                                │
       │ 11. subsequent requests with Bearer <craftsky token>           │
       ├───────────────────▶│                                          │
```

Scopes requested in v1: `atproto` (mandatory) plus whatever scope covers creating `social.craftsky.feed.post` records. The exact scope string is a deploy-time config value (`OAUTH_SCOPES`) because `transition:generic` is deprecated per the atproto-spec and the final scope for custom lexicons is still being nailed down across the ecosystem. Default for v1: `atproto transition:generic`.

### 3.3 The callback handoff

The callback lands in the **user's system browser**, not the Flutter/CLI process. How the Craftsky token reaches the client depends on the client type. The AppView supports both handoff modes; which one to use is selected by a value recorded in the `oauth_auth_requests` row when `/auth/login` is called (`handoff_mode: "deep_link" | "loopback"`).

- **Deep link** (Flutter mobile / desktop): callback page redirects to `craftsky://auth/complete?token=<craftsky_token>`. The OS hands it to the registered Craftsky app.
- **Loopback** (CLI, dev localhost mode): before opening the browser, the CLI starts `http://127.0.0.1:<random_port>/` and passes that URL in `/auth/login`. The callback page submits the token to that loopback URL via JS fetch and then shows a "you can close this tab" message.

Exact scheme registration, loopback port allocation, and callback-page HTML/JS belong to the client-integration specs that will land later. The AppView's responsibility in v1 is:

1. Accept `handoff_mode` + (for loopback) `loopback_redirect_uri` in the `/auth/login` request.
2. Persist them in the `oauth_auth_requests` row (note: inside the `data` blob or a sibling column — TBD when implementing; must not break indigo's serializer).
3. Branch on them when rendering the callback page.

### 3.4 Authenticated-request flow

The existing `Authenticated` middleware in `internal/middleware/` keeps its shape. The `AuthService.Authenticate` call now does:

```
1. Extract Bearer token from Authorization header.
2. SHA-256 hash it.
3. SELECT account_did, oauth_session_id FROM craftsky_sessions
   WHERE token_hash = $1 AND revoked_at IS NULL.
4. If found: write (did, oauth_session_id) into context, call next handler.
5. If not: 401.
6. (Async, throttled) UPDATE last_seen_at.
```

A new context helper exposes `oauth_session_id` to handlers that need to make PDS calls:

```go
sess, err := deps.OAuth.ResumeSession(ctx, did, oauthSessionID)
client := sess.APIClient()  // DPoP-signing, auto-refresh
client.Post(ctx, "com.atproto.repo.createRecord", body, &resp)
```

indigo handles DPoP signing, nonce rotation (`DPoP-Nonce` header retry on HTTP 400), and token refresh transparently. Refresh mutates `oauth_sessions.data` via our `SaveSession`.

### 3.5 Logout

- `POST /auth/logout`: sets `revoked_at` on the Craftsky session row identified by the Bearer token. OAuth session untouched. Other devices keep working.
- `POST /auth/logout?all=true`: calls `oauth.ClientApp.Logout(did, session_id)` first (indigo attempts AS-side revocation if supported, then deletes the `oauth_sessions` row). The `ON DELETE CASCADE` on the FK removes the dependent `craftsky_sessions` rows. Note: this intentionally **drops** those rows rather than marking `revoked_at` — full logout prioritises cleanup over audit. The audit trail via `revoked_at` applies to single-device logout only.

### 3.6 Error paths worth naming

| Failure | Response |
|---|---|
| Unknown handle in `/auth/login` | 400 `{"error": "handle_not_found"}` |
| PDS unreachable during PAR / AS metadata fetch | 502 `{"error": "authorization_server_unavailable"}` |
| Missing / unknown `state` at callback | HTML error page. Usually means the `oauth_auth_requests` row was cleaned up (30 min). Don't 500. |
| Token exchange fails | HTML error page. Don't create a `craftsky_sessions` row. |
| DPoP nonce rejection on a downstream request | indigo retries internally with the new nonce. Transparent to handlers. |
| Refresh token rejected (180-day hit, or AS revoked) | `APIClient` surfaces an auth error. Handler returns 401. The `craftsky_sessions` row is effectively orphaned (OAuth session dead); lazy cleanup handles it. |

### 3.7 What this section deliberately does not cover

- **Rate limiting** on the auth endpoints — future work.
- **CSRF on the callback** — the `state` parameter already provides it; indigo validates.
- **IP binding / device fingerprinting** — not doing it.
- **"Logout from other devices" UX** — `?all=true` plumbing is there, but no listing/selection endpoint yet.

## 4. Implementation map

### 4.1 Package layout — `appview/internal/auth/`

```
internal/auth/
├── service.go          (exists) AuthService interface, ctx helpers — unchanged
├── mock.go             (exists) dev mock — unchanged
├── oauth.go            (exists, rewritten) replaces NotImplementedAuthService
│                       with real Craftsky-token → DID resolver
├── config.go           (new)    Loads hostname, client key, scopes, TTLs;
│                                builds indigo oauth.ClientConfig
├── store.go            (new)    Postgres impl of oauth.ClientAuthStore
├── store_test.go       (new)    Integration tests against compose Postgres
├── craftsky_session.go (new)    Generates/revokes/looks up bearer tokens
├── handlers.go         (new)    HTTP handlers for the 5 endpoints
└── handlers_test.go    (new)
```

Nothing else restructures; we expand `auth/`.

### 4.2 Touch points outside `internal/auth/`

| File | Change |
|---|---|
| `internal/app/config.go` | Add `OAuth*` config keys (hostname, key source, session TTLs, auth-request TTL, scopes, JWT/token defaults). Dev defaults wire `oauth.NewLocalhostConfig`. |
| `internal/app/deps.go` | Construct `oauth.ClientApp`, Postgres store, Craftsky-session store; wire into `Deps`. Replace `NotImplementedAuthService` with the real impl in `NewProdDeps`. |
| `internal/routes/routes.go` | Register the 5 new routes. |
| `internal/middleware/` | No structural change. `Authenticated` continues to call `AuthService.Authenticate`; the new impl writes the `oauth_session_id` into ctx via a new helper. |
| `environments/dev.env` | Add `OAUTH_*` keys with localhost-mode defaults. |
| `environments/prod.env.example` | Add `OAUTH_HOSTNAME`, `OAUTH_CLIENT_SECRET_KEY_PATH`, `OAUTH_CLIENT_SECRET_KEY_ID` as required-in-prod. |
| `justfile` | New recipes: `just oauth-keygen` (generate ES256 dev key), optional helper recipes for exercising the flow locally. |
| `AGENTS.md` | No rule rewrite in v1. Add a short note (inline near rule #2 or in a footnote) flagging that the TMB upgrade will require amending the wording. |
| `appview/README.md` | Document the 5 endpoints, env vars, dev key generation. |

The `cmd/cli` CLI gets `login`/`logout` subcommands in a **separate, later spec** — not here.

### 4.3 Migration

One migration, creating all three tables together (the FK makes them a single logical unit):

```
appview/migrations/000002_oauth_tables.up.sql
appview/migrations/000002_oauth_tables.down.sql
```

The existing `000001_bluesky_posts_sample` stays. It'll be removed when the first Craftsky indexer lands (separate spec).

### 4.4 Dependency additions

All from `indigo`, pulled transitively by the first import:

- `github.com/bluesky-social/indigo/atproto/auth/oauth`
- `github.com/bluesky-social/indigo/atproto/atcrypto`
- `github.com/bluesky-social/indigo/atproto/identity`
- `github.com/bluesky-social/indigo/atproto/syntax`

`go mod tidy` handles the rest.

### 4.5 Testing posture

- **`store_test.go`:** integration tests against compose Postgres. All 6 `ClientAuthStore` methods, plus lazy cleanup (session expiry, session inactivity, auth-request expiry). Use real `oauth.ClientSessionData` / `oauth.AuthRequestData` values from indigo to ensure the JSONB round-trip is stable across library versions. **Must include a test that confirms `SaveSession` is called on token refresh and updates `updated_at`** — if it isn't, inactivity cleanup will evict live sessions. If indigo does not call `SaveSession` on refresh, the inactivity semantics in §2.1 need revisiting (drive cleanup from `oauth_sessions.data` contents instead, or ship a different accounting strategy).
- **`craftsky_session_test.go`:** integration tests for token generation, hash-based lookup, `last_seen_at` throttling, revocation, FK cascade on `oauth_sessions` delete.
- **`handlers_test.go`:** unit tests for endpoint shape (status codes, error responses, handoff branching) against a mocked `oauth.ClientApp`.
- **No end-to-end tests against a real PDS.** indigo is the library handling the atproto-facing correctness; the value we add is our store + handlers + Craftsky session layer, all of which are tested directly.

`just test` stays the single test runner, running on the host against compose Postgres.

## 5. Open questions flagged, not resolved

1. **Prod client-key source.** PEM in env var? File path mounted from a secret volume? KMS-signed assertions later? The spec requires v1 to implement at least one variant; operators/deployment decide which.
2. **Deep-link URL scheme** for Flutter, **loopback strategy** for CLI. Both belong to client-integration specs.
3. **Scope string for the Craftsky custom lexicon.** v1 defaults to `atproto transition:generic`; this may change once the atproto ecosystem settles on a non-transitional scope syntax for custom record types. Configurable via `OAUTH_SCOPES`.
4. **Where `handoff_mode` lives on the auth-request row.** Inside the opaque `data` JSONB (risks indigo re-serializing and dropping unknown fields) or as a sibling column. **Decision heuristic:** write a unit test that round-trips an `oauth.AuthRequestData` with an extra top-level JSONB field through indigo's serializer; if the extra field survives, inline in `data` is fine. If it does not, add a sibling column. Do this check before committing either direction.

## 6. Future work

Explicitly out of scope for v1. Listed here so they're discoverable.

1. **TMB upgrade.** Add `/auth/session/exchange` and `/auth/session/refresh` endpoints that hand down short-lived access tokens + DPoP JWKs to clients for direct-to-PDS calls. Requires reading fields from indigo's `ClientSessionData`; may need an indigo PR or a bounded unmarshal. Primary motivator: avoid proxying blob uploads through the AppView.
2. **Write-proxy endpoint.** `POST /xrpc/com.atproto.repo.createRecord` proxied for `social.craftsky.feed.post`. Own spec.
3. **Blob upload proxying.** `com.atproto.repo.uploadBlob`. The bandwidth cost of doing this through the AppView is what will likely motivate TMB.
4. **Client-key rotation.** v1 has one key; rotating it re-authenticates all users. True rotation (sessions survive rotation) needs either indigo exposing per-session signing-key selection, or us running multiple `ClientApp` instances keyed by `kid` and routing refreshes.
5. **App-layer encryption of `oauth_sessions.data`.** Envelope encryption with a KMS-held key. Depends on an indigo serialization hook or us taking over serialization.
6. **Active-session management UI.** "Signed in on these devices" + remote sign-out. Depends on Flutter app profile/settings screens.
7. **Dedicated sweeper process** for revoked Craftsky sessions. v1 relies on lazy cleanup (same pattern as the cookbook's OAuth store); a sweeper becomes worth it at scale.
8. **CLI `login` / `logout` subcommands.** Depends on the loopback handoff spec.
9. **Rate limiting** on `/auth/login`, `/oauth/callback`.
