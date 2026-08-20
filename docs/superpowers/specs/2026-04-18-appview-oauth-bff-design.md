# AppView OAuth BFF — durable handoff and session design

**Date:** 2026-04-18

**Security revision:** 2026-08-20

**Status:** implemented, with release-verification gates still open

**Scope:** atproto OAuth, server-side credentials, CraftSky sessions, callback
finalization, client handoff, refresh coordination, and revocation.

This revision replaces the original one-step callback handoff. The authoritative
finding-level plan is
[`AV-008_AV-018_AV-019-oauth-handoff.md`](../../2026-appview-code-audit-plan/AV-008_AV-018_AV-019-oauth-handoff.md).

## 1. Core rules

1. AppView is the confidential atproto OAuth client. Access tokens, refresh
   tokens, DPoP private keys, nonces, and authorization-server session data are
   stored and used only by AppView.
2. Flutter and the CLI receive only opaque CraftSky session bearers.
3. A browser callback never transports a CraftSky bearer. It transports only a
   short-lived, hash-only, device-bound handoff code.
4. First persistence creates a `pending_handoff` OAuth parent. Ordinary session
   resume and authenticated routes accept only `active` parents/children.
5. Direct code redemption creates one pending child and a sealed receipt.
   Confirmation after client persistence atomically activates both.
6. Account-deletion OAuth creates a childless `deletion_only` parent bound to
   one operation and positive credential generation. It never restores ordinary
   membership or emits a normal bearer.
7. Owner lifecycle state, owner generation, auth epoch, parent version, and the
   canonical lock order are authorization inputs, not advisory metadata.

## 2. Components

- **Indigo OAuth client:** performs discovery, PAR, callback exchange, DPoP
  signing, nonce handling, refresh, and revocation.
- **Hardened federated HTTP boundary:** validates HTTPS origins/endpoints,
  blocks private/special-use addresses and DNS rebinding, disables ambient
  proxies, caps redirects/bodies, and applies purpose-specific timeouts.
- **Postgres auth store:** persists typed authorization-request metadata,
  exchange attempts, OAuth parents, CraftSky children, handoff exchanges and
  receipts, cleanup jobs, lifecycle versions, and deadlines.
- **Owner lifecycle/session coordinator:** holds canonical owner and parent
  fences while resuming or refreshing a session and CAS-persists every Indigo
  mutation before allowing the remote operation to finish.
- **Cleanup processors:** expire abandoned handoffs, revoke unusable parents,
  clean auxiliary device state, and retain honest ambiguity evidence when an
  upstream token cannot be reconstructed.

Raw `oauth.ClientApp.ResumeSession` and uncoordinated `SaveSession` callbacks are
not application dependencies. PDS callers receive narrow coordinated
capabilities.

## 3. Public endpoint surface

| Method | Path | Access | Purpose |
|---|---|---|---|
| `GET` | `/oauth/client-metadata.json` | public AS-facing | Immutable client metadata derived from the configured public origin. |
| `GET` | `/oauth/jwks.json` | public AS-facing | Immutable public JWKS for the configured ES256 key. |
| `GET` | `/oauth/callback` | browser/AS | Finalize the upstream exchange and render a code-only handoff. |
| `POST` | `/v1/auth/login` | anonymous, device-bound | Start login or the server-derived deletion-replacement flow. |
| `POST` | `/v1/auth/handoffs/exchange` | anonymous, code + device protected | Redeem a code for one pending bearer and sealed receipt. |
| `POST` | `/v1/auth/handoffs/confirm` | pending bearer + device | Confirm durable client persistence and activate the session. |
| `POST` | `/v1/auth/logout` | authenticated recovery | Revoke one child or, with the explicit all-devices option, advance the owner auth epoch and invalidate all ordinary auth artifacts. |

The `/oauth/*` paths are protocol-facing and unversioned. CraftSky client APIs
are under `/v1/` and use camelCase JSON.

## 4. Login start

Flutter sends:

```json
{"handle":"alice.example","handoffMode":"verified_link"}
```

The CLI uses `handoffMode: "loopback"` plus an exact
`http://127.0.0.1:<port>/<path>` URI. No other loopback host, scheme, or missing
port is accepted.

Before Indigo stores the authorization request, AppView:

1. resolves the entered identifier to one canonical DID;
2. takes that owner's auth fence;
3. creates the first-login lifecycle row if necessary;
4. records purpose, mode, validated loopback URI, stable device ID, owner DID,
   owner generation, and auth epoch in the same insert as Indigo state/data;
5. starts Indigo with the resolved identity and rejects any callback DID drift.

There is no follow-up metadata update that a fast callback can race.

## 5. Callback and exchange durability

The callback reads complete typed metadata before logical consumption. Under the
same bounded owner/parent operation it marks a unique `exchange_started`
attempt, performs the upstream token exchange, and attempts the first parent
save.

- Before the first parent save, a process crash can lose an upstream token that
  AppView cannot reconstruct. The durable non-secret attempt becomes
  `exchange_ambiguous`; it is retained and alerted honestly rather than called
  revoked or recoverable.
- If token material is still in memory but the first save fails, AppView makes
  a bounded revocation attempt. Unconfirmed revocation remains ambiguous.
- After the parent save succeeds, the durable parent owns cleanup. Callback or
  onboarding failure moves it to `revocation_pending` for the worker.

Successful login finalization creates a random 32-byte handoff code, stores
only its SHA-256 hash, and records the exact device, owner, auth epoch, parent,
and short expiry. The browser receives only:

```text
https://app.craftsky.social/auth/complete?code=<single-use code>
```

For CLI loopback, the callback page posts the same code—not a bearer—to the
prevalidated loopback listener. Callback HTML uses restrictive cache, referrer,
framing, MIME, and script policies.

## 6. Exchange and confirmation

`POST /v1/auth/handoffs/exchange` hashes the supplied code, verifies the stable
device ID, owner epoch, parent state, and deadline, then creates at most one
`pending_confirmation` child. The plaintext bearer is sealed under a dedicated
handoff-receipt AEAD key; code and receipt replay by the same device return the
same pending result during the window.

`POST /v1/auth/handoffs/confirm` authenticates only the exact pending bearer and
requires its device and receipt ID. In one ordered transaction it changes the
parent to `active`, the child to active, consumes the code/receipt, and destroys
sealed pending secrets. A pending bearer cannot call ordinary APIs.

The row-lock order after canonical advisory locks is:

1. owner lifecycle/auth epoch;
2. authorization requests;
3. account-deletion operation when relevant;
4. OAuth parents by session ID;
5. CraftSky children;
6. handoff exchanges/receipts;
7. cleanup/outbox rows.

Every callback, exchange, confirmation, logout, expiry, cleanup, and deletion
credential path follows the same relative order.

## 7. Account-deletion branch

Deletion start records an intent but does not grant permanent-delete authority.
After fresh OAuth and exact-handle proof, the callback converts only the exact
pending parent into `deletion_only`, binds it atomically to the operation and a
positive credential generation, emits no child, and redirects to:

```text
https://app.craftsky.social/account-deletion/reauth-complete?job-id=...&proof=...
```

Only the claimed matching deletion worker authority may resume that parent.
Cancel or expiry queues it for revocation; acceptance preserves only the exact
bound parent; completion preserves none. If an accepted job later needs fresh
authority, the normal public sign-in entry point detects the same deleting DID
and server-side job state and derives the replacement purpose. No caller flag,
status bearer, post-acceptance recovery route, or membership restoration is
allowed.

## 8. Session use, refresh, and logout

Authenticated requests resolve a hash of the opaque CraftSky bearer to an
active child and active parent whose owner generation/auth epoch still match.
Database or lifecycle failures return a retryable infrastructure response; they
are not misreported as invalid credentials.

Every complete PDS method runs inside the session coordinator. It holds a shared
owner fence plus the parent-session lock, reloads the latest version, and
CAS-persists refresh/nonce changes on the same connection. A persistence
failure is indeterminate and the raw client does not escape.

Single-device logout revokes the selected child locally first and queues parent
revocation when appropriate. All-devices logout advances the DID-wide auth
epoch under the owner fence and invalidates ordinary parents, children,
authorization requests, handoffs, and exchanges. The sole narrow exception is
an exact accepted `deletion_only` credential, which is rebased only when the
account-deletion operation proves that exemption.

## 9. Storage and retention properties

The schema is defined by the paired migrations rather than copied into this
design. It enforces:

- typed request purposes, modes, lifecycle states, deadlines, and generations;
- immutable owner/session identity and monotonic versions;
- hash-only bearer/code lookup;
- one child/receipt per handoff exchange;
- bounded pending-capacity and sweeper indexes;
- leased, fenced revocation and auxiliary cleanup work;
- no durable plaintext handoff bearer;
- terminal cleanup of local credentials while retaining only the minimal
  non-secret ambiguity evidence required by the provider guarantee.

## 10. Required security tests

The implementation must cover real-store and concurrency barriers for:

- atomic auth-request metadata versus a fast callback;
- handle rebinding/DID mismatch;
- private-address and rebinding attempts during discovery, PAR, token,
  revocation, refresh, and PDS use;
- first-save failure and the pre-save ambiguous crash window;
- callback-only pending onboarding and rejection by ordinary resume;
- same-device exchange replay, wrong-device rejection, lost response, storage
  failure, confirmation retry, expiry, and process restart;
- callback versus single/all logout and auth-epoch changes;
- concurrent refresh across two AppView instances and stale version CAS;
- exact deletion-only binding, cancellation, acceptance, replacement, and
  completion;
- cleanup lease expiry, stale-token finalization, retry ceiling, and secret
  destruction;
- high-cardinality owner logout without process-local lock maps.

## 11. Configuration and release gates

Production derives all OAuth URLs and Expected Host behavior from one canonical
HTTPS public origin. Client metadata/JWKS are immutable artifacts. Legacy
hostname/callback variables are rejected in production, secrets are redacted,
and every operation duration is positive and bounded.

Before release, operators must also:

1. confirm `app.craftsky.social` as the controlled verified-link origin;
2. publish Android `assetlinks.json` with the real release certificate SHA-256;
3. publish the Apple AASA document and enable Associated Domains for
   `B6YZZCUZWS.social.craftsky.app`;
4. provision the rotatable handoff-receipt AEAD key in secret storage;
5. verify login and deletion links on release-signed physical devices and test
   the web fallback with the app absent.

No custom authentication URL scheme or bearer-in-browser compatibility path is
permitted.
