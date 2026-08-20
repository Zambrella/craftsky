# Flutter Auth — verified-link handoff design

**Date:** 2026-04-21

**Security revision:** 2026-08-20

**Status:** implemented, with release-verification gates still open

**Scope:** Flutter authentication against the AppView BFF on iOS and Android.

This revision replaces the original custom-scheme, bearer-in-URL design. The
authoritative remediation and acceptance detail is in
[`AV-008_AV-018_AV-019-oauth-handoff.md`](../../2026-appview-code-audit-plan/AV-008_AV-018_AV-019-oauth-handoff.md).

## 1. Security contract

- The Flutter app holds only an opaque CraftSky session bearer. It never holds
  an OAuth access token, refresh token, or DPoP private key.
- The browser callback carries only a short-lived, single-use handoff code:
  `https://app.craftsky.social/auth/complete?code=...`.
- Production authentication uses verified HTTPS Android App Links and iOS
  Universal Links. No custom URL scheme is registered for authentication.
- A CraftSky bearer never appears in a browser URL, route parameter, provider
  key, referrer, callback HTML, or diagnostic string.
- The code is not a session. The initiating installation must redeem it with
  the same `X-Craftsky-Device-Id` over the direct AppView API.
- Redemption creates one pending local/server handoff. Only confirmation after
  durable client storage activates the parent OAuth session and child bearer.
- The server owns all expiry decisions. Client wall-clock state is not an
  authorization boundary.

These requirements are breaking and replace every earlier reference to a
custom-scheme token callback or a one-call `whoami` handoff.

## 2. End-to-end login flow

1. `AuthController.signIn` normalizes the handle and calls:

   ```http
   POST /v1/auth/login
   X-Craftsky-Device-Id: <stable installation id>
   Content-Type: application/json

   {"handle":"alice.example","handoffMode":"verified_link"}
   ```

2. AppView returns `authUrl`; Flutter opens it in the system browser.
3. The PDS authorization server redirects to AppView's `/oauth/callback`.
4. AppView completes the upstream exchange into a `pending_handoff` parent,
   performs the bounded onboarding initialization, creates a random handoff
   code, and redirects the browser to:

   ```text
   https://app.craftsky.social/auth/complete?code=<single-use code>
   ```

5. `AuthCompleteRoute` accepts `code` (or a redacted error), and
   `AuthController.completeFromDeepLink` calls:

   ```http
   POST /v1/auth/handoffs/exchange
   X-Craftsky-Device-Id: <same installation id>
   Content-Type: application/json

   {"code":"<single-use code>"}
   ```

6. AppView returns a `PendingHandoff` containing the pending CraftSky bearer,
   DID, canonical handle, opaque receipt ID, and `confirmBy` deadline. A replay
   of the same code from the same device within the server window returns the
   same pending result; another device is rejected.
7. Flutter writes the complete pending snapshot to the secure session registry
   before making it visible as a signed-in account.
8. Flutter confirms the receipt directly:

   ```http
   POST /v1/auth/handoffs/confirm
   Authorization: Bearer <pending CraftSky bearer>
   X-Craftsky-Device-Id: <same installation id>
   Content-Type: application/json

   {"receiptId":"<opaque receipt id>"}
   ```

9. AppView atomically activates the parent and child. Flutter atomically moves
   the locally staged handoff into the active session registry. Only then may
   ordinary authenticated API calls use the bearer.

If the exchange response is lost, the same device retries the code. If the app
is killed after staging, cold-start recovery retries confirmation from the
stored receipt. If the confirmation response is lost, confirmation is
idempotent. If the server declares the receipt invalid or expired, Flutter
discards the pending snapshot and asks the user to sign in again.

## 3. Client state and storage

### 3.1 Active sessions

`sessionRegistryProvider` owns the encrypted-at-rest platform storage snapshot
for active CraftSky sessions, the active DID, and at most one pending handoff.
The pending bearer is never published through `authSessionProvider` until the
confirm transaction succeeds.

An active entry contains the minimum canonical identity required by the app:

```json
{
  "token": "<opaque CraftSky bearer>",
  "did": "did:plc:...",
  "handle": "alice.example"
}
```

The registry supports multiple accounts. Account switching, removal, and
pending-handoff promotion are serialized so a late callback cannot overwrite a
newer active-account choice.

### 3.2 Pending browser intent

`pendingAuthProvider` is UX state only: it records that the user launched a
browser sign-in and supports error copy. It is not relied upon to validate a
callback or to impose a client-side freshness window. The server-side code,
device binding, auth epoch, parent lifecycle, receipt state, and deadlines are
authoritative.

### 3.3 Diagnostics

`PendingHandoff.toString`, auth-controller logging, router diagnostics, and
provider keys must remain content-free. They may log a generated run ID and a
typed outcome, but not codes, receipts, bearers, callback query strings, OAuth
state, PDS tokens, or full upstream URLs.

## 4. Router and UI behavior

- `AuthCompleteRoute` is a root-navigator route at `/auth/complete` with an
  optional `code` and redacted `error`.
- Signed-out users may remain on this route while exchange/confirmation runs.
- `AuthCompletePage` invokes completion once, shows bounded progress, and
  offers retry for recoverable network/storage failures.
- A confirmed first account routes to onboarding or home according to the
  per-DID onboarding state. Adding an account preserves existing accounts.
- Cold start resumes a durably staged receipt before declaring the sign-in
  abandoned.
- Server `401` for an active CraftSky bearer removes only the matching account.
  Infrastructure errors such as `503` preserve the account and propagate as a
  retryable error.

## 5. Account-deletion reauthentication

Deletion reauthentication is intentionally separate from ordinary login. Its
verified link is:

```text
https://app.craftsky.social/account-deletion/reauth-complete?job-id=...&proof=...
```

The proof is single-use and bound to the deletion intent/job. It is not an
ordinary CraftSky bearer and cannot restore membership. The callback creates a
`deletion_only` server credential, emits no ordinary handoff child, and the
client immediately completes or retries the pre-acceptance deletion flow. No
post-acceptance deletion status credential, recovery route, or polling UI is
introduced.

## 6. API clients

`AuthApiClient` uses the normal Dio stack for login, `whoami`, and logout.
`HandoffApiClient` is credential-free for exchange and accepts the pending
bearer only as the `Authorization` header of the confirm request. The bearer is
passed as a method argument, not used as a Riverpod family key or embedded in a
base URL.

Anonymous paths are limited to the explicit login and handoff endpoints. The
global sign-out interceptor reacts only to an authenticated `401`; it must not
sign out on `5xx`, network cancellation, or a handoff failure.

## 7. Platform verification

### 7.1 Android

`AndroidManifest.xml` contains two exact HTTPS `android:autoVerify="true"`
filters for host `app.craftsky.social`:

- `/auth/complete`
- `/account-deletion/reauth-complete`

`https://app.craftsky.social/.well-known/assetlinks.json` must authorize package
`social.craftsky.app` using the real release-signing certificate SHA-256. A
debug certificate must never be published as production authority.

### 7.2 iOS

`Runner.entitlements` contains only the required Associated Domain entry:
`applinks:app.craftsky.social`. The AASA document must authorize Apple App ID
`B6YZZCUZWS.social.craftsky.app` and only the two completion paths above. The
release provisioning profile must include Associated Domains.

### 7.3 Web fallback

Both HTTPS paths must render a safe fallback when the app is absent. The page
must not echo secrets, add third-party scripts, permit framing, or leak callback
queries through referrers.

## 8. Required tests

Automated coverage must prove:

- login sends `handoffMode: verified_link` and the stable device ID;
- the auth route accepts a code, never a bearer query parameter;
- exchange, durable staging, confirm, and local promotion occur in that order;
- a lost exchange response, lost confirm response, final local-write failure,
  app restart, and concurrent deep-link/cold-start recovery converge safely;
- pending sessions remain invisible to ordinary requests;
- another device cannot exchange or confirm the handoff;
- invalid/expired receipts are discarded without affecting existing accounts;
- code, receipt, and bearer values are redacted from diagnostics;
- a `401` invalidates only its account and a `503` preserves it;
- Android and iOS release artifacts contain exactly the intended verified-link
  declarations and no custom authentication scheme.

## 9. Release gates

Code completion does not prove domain ownership or OS verification. Before
release, operators must:

1. confirm `app.craftsky.social` is the canonical controlled callback host;
2. publish and validate `assetlinks.json` with the real Android release
   certificate fingerprint;
3. publish and validate the AASA document and Apple provisioning capability;
4. exercise both paths on release-signed physical Android and iOS devices;
5. test both web fallbacks with the app absent.

Until these gates pass, the implementation is suitable for deterministic test
and review but not a production OAuth handoff.
