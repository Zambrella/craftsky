# Custom console stage 1: Atmosphere operator login

Status: Complete and validated locally on 15 October 2026

The approved test boundaries are the admin HTTP login/callback/identity/logout
routes, maintainer CLI and browser shell. Migrated PostgreSQL is real; external
AT Protocol identity/authorization servers are controlled fixtures. Tests cover
AUTH-01 through AUTH-10 in the accepted plan. Implement one observable slice at
a time, starting with DID provisioning and permission-set resolution, followed
by OAuth/browser binding, request-time authorization, and the UI.

This stage does not deploy, publish or provision production access. A real
development account check requires the user's chosen handle/DID and interactive
authentication at its provider; no password is collected by CraftSky.

## Implemented

- Added the standalone React/TypeScript project in `appview/admin/`, packaged by
  the existing AppView Docker build and scoped to `/admin/`. This placement fits
  the existing `appview` Docker/Render build context without new infrastructure.
- Added a separate identity-only Indigo OAuth client and browser-bound, one-use
  pending attempts. The console does not use consumer onboarding, profile
  initialisation, mobile handoff, consumer session tables or PDS record writes.
- Added DID allowlisting and central admin/mod/helper sets over the existing
  operator registry. Browser requests resolve current sets and assignments;
  the legacy per-operator grant map is not combined with the selected set.
- Added host-only HttpOnly cookies, exact-origin/CSRF checks, eight-hour absolute
  and 30-minute idle expiry, logout and maintainer session revocation. OAuth
  tokens are discarded after identity verification rather than stored/refreshed.
- Production admin composition now uses the operator cookie boundary and has no
  legacy bearer fallback. The synthetic Retool fixture retains its isolated
  test transport. Migration 94 revokes pre-existing active integration tokens.
- Replaced routine credential issuance with privileged maintainer CLI commands:
  `allow`, `list`, `set`, `disable`, `revoke-sessions`, `audit`. Changes record
  maintenance attribution and preserve stable operator/audit references.
- Classified the new DID columns in the terminal-lifecycle inventory. Globally
  terminal DIDs cannot log in, receive a session or authenticate an existing
  session. Pending attempts join bounded terminal purge; private operator/audit
  identity is retained. Migration 95 supplies its required DID/state index.
- Recorded [ADR 018](../../../adr/018-atmosphere-operator-console-sessions.md),
  amended both API/auth architecture documents, added the
  [OpenAPI contract](../../../appview/internal/operatorconsole/openapi.json)
  and replaced the [operations runbook](../../operations/moderation-operators.md).

## Observed verification

The requested two Atmosphere DIDs were provisioned as admins in this worktree's
local development database only. The real account identifiers and allowlist rows
are not seeded into application source or production configuration.

Using the shared preview, the first requested account completed authentication
at Bluesky's official provider. Its consent screen requested identity only.
The callback returned to `/admin/` with the correct DID and administrator set.
Database inspection showed one local session and zero remaining pending
attempts. The real Sign out control returned to the login screen and left zero
local sessions. The second account is allowlisted but has not been exercised
through its own provider session.

The desktop authenticated shell was visually inspected and captured at:
`/Users/douglastodd/.t3/userdata/browser-artifacts/browser-screenshot-127-0-0-1-mv9fwgep-7ae51fd5.png`.
Preview viewport-resize calls timed out; mobile-size browser inspection is not
claimed. Responsive CSS is present, but that is not equivalent to a verified
mobile browser result.

Focused race tests pass for:

- Listed/unlisted accounts, handle changes/reuse, forged login bodies and
  identity-only scope checks.
- Wrong subject/issuer, missing/extra scopes, wrong browser, duplicate state,
  expired attempts and callback replay.
- Permission changes, disablement, explicit revocation, idle/absolute expiry,
  global terminal identity denial and helper assignment/release boundaries.
- Cross-origin/CSRF rejection, logout, legacy/consumer bearer rejection,
  fail-closed database behavior and safe identity output.
- Maintainer CLI provisioning/set changes/disablement and route policies.

Existing consumer OAuth tests pass. Frontend type/production build, operator
OpenAPI validation, Go vet, shell syntax and whitespace checks pass. npm's
runtime dependency audit reports zero vulnerabilities.
The local Vite proxy was started and exercised over HTTP: CSRF bootstrap returned
200 and a malformed identifier returned the expected 400 `invalid_request`,
confirming the proxy's scoped Origin translation. That temporary dev server was
then stopped; the Docker-served console remains available.

The release gate initially caught a test-style issue and the missing terminal
DID inventory classification. Both were corrected; the terminal regression and
inventory/index checks pass with race detection. The final full
`./scripts/appview-check` run exited successfully with all release gates passed:
ordinary and race suites, static analysis, Docker build, migrations, API smoke
checks and vulnerability scanning. Go's scan reports no affected or imported
package vulnerabilities; one required-module vulnerability is outside the code's
call paths. The pinned Tap handshake passed; deterministic event injection is
not supported by the pinned Tap version and was not claimed.

The final log is `/private/tmp/craftsky-console-release-check.log`, with artifacts
at `/private/tmp/craftsky-console-release-check`. An intervening run was
invalidated by a test-file edit during its build; the successful final run used
frozen source.

## Scope and remaining work

This is the authenticated console shell. Case review, safety workflows and
browser operator administration remain the later approved slices. Assignment
administration is not exposed in this shell yet; existing assignments are
enforced by the backend.

No production allowlisting, environment updates, deployment, Retool publishing,
commit or push was performed. The existing other worktree's Docker stack was
left untouched.
