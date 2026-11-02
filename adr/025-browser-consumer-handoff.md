# ADR 025: Browser consumer OAuth handoff and HttpOnly client sessions

Date: 2026-11-02
Status: Accepted for the requested full web client implementation

## Context

The public SvelteKit client now needs authenticated mobile feature parity.
Mobile authentication callbacks are verified app links and intentionally render
static, content-free browser fallbacks. Reusing those routes for a browser login
would send browser sign-ins to an installed mobile app and weaken the existing
callback contract. Operator-console sessions are purpose-isolated and must not
authenticate consumer activity.

## Decision

Add the explicit consumer handoff mode `browser` to login, provider-first
registration, optional chat reconsent, and owner-approved deletion
reauthentication. PDS OAuth credentials and DPoP keys remain in AppView.
Browser mode reuses the existing device-bound exchange, pending child session,
durable receipt, confirmation, owner lifecycle fencing and revocation paths.

The Svelte server mediates browser API requests. It stores only opaque CraftSky
child-session credentials in host-only, HttpOnly, SameSite=Lax cookies, Secure
outside explicitly local development. It must not expose them to page data,
browser JavaScript, local storage, URLs, diagnostics or third-party origins.
Separate account cookies and selected-account metadata permit account switching
without merging identities. AppView remains the session and membership authority.
Unsafe BFF requests and login initiation require same-origin/CSRF checks. A 401
removes only the matching browser account; transient infrastructure failures retain
it. Pending exchange/confirmation must be durable before activation becomes visible.

Android's previous host-wide verified intent filter would also intercept the new
browser callback URLs. Its path alternatives now match the existing iOS
association contract: `/auth/complete`, `/account-deletion/reauth-complete`,
`/posts/` and `/profiles/`. Browser-only `/app/*`, `/auth/browser/*` and
`/account-deletion/browser/*` paths remain in the browser. This narrows the older
Android all-path decision and requires an Android release before browser OAuth is
reliable with that app installed. Android combines the path alternatives within
the same host/scheme filter, as documented in [Android deep-link filters](https://developer.android.com/training/app-links/create-deeplinks).

Browser login starts include `browserRedirectUri`, with exactly the configured
`WEB_APP_ORIGIN`, the path `/auth/browser/complete`, and one cryptographically
random 32-byte base64url `state` query. Deletion uses
`/account-deletion/browser/reauth-complete`. No arbitrary host, path, fragment,
credentials, extra query key or redirect target is accepted. The BFF stores the
intent nonce in an HttpOnly cookie and compares it before exchanging a callback
code. The existing installation/device binding remains independently enforced.

`WEB_APP_ORIGIN` defaults to `https://app.craftsky.social`. Production only permits
that controlled HTTPS origin. Development may explicitly configure an HTTP IPv4
loopback origin with a valid port. Request Host/Origin headers never establish
backend redirect authority. Worktree tooling supplies the selected local origin.

The OAuth callback returns HTTP 303 to the stored browser target, retaining the
intent nonce and appending only the one-time consumer handoff code. Registration
failures append only their reviewed error code. Deletion appends only its bound
job ID and single-use proof and never mints an ordinary child handoff. Responses
are no-store and no-referrer; no callback capability is rendered into HTML.

For storage, the existing private `oauth_auth_requests.loopback_redirect_uri`
column retains its legacy name and carries either a loopback target or a browser
target, distinguished by the explicit handoff mode. Migration 000112 expands the
constraint without rewriting existing mobile requests or sessions.

## Consequences

The public read endpoints remain intact and existing mobile handoff behavior is
unchanged. The Svelte BFF needs a complete per-account session/CSRF boundary and
recovery tests; accepting browser mode alone does not provide authenticated UI.
No generic OAuth library, reusable PDS delete endpoint, OAuth-token browser
handoff, new lexicon, production mutation or domain reassignment is introduced.
Browser deployment needs the normal local exact-tag AppView release, followed by
reviewed web publication and live OAuth checks. Tests of mocked providers or local
callbacks are not proof of production PDS consent or device link resolution.
