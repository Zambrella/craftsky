# Moderation operator access

The moderation console uses Atmosphere OAuth and a private DID allowlist.
`MODERATION_ADMIN_ENABLED` controls the console/admin route registration;
`MODERATION_SOURCE_DID` must still be valid. Route enablement alone grants no
access. Ordinary CraftSky sessions, development DID headers and legacy operator
bearers cannot authenticate production admin routes.

The standalone frontend lives in `appview/admin/` and is built into the AppView
image, served at `/admin/`. Deployment follows the existing AppView exact-tag
release process. Production configuration, provisioning and deployment remain
separately authorised operations under the [production runbook](production-render.md).

## Provision and maintain the allowlist

Use the companion CLI through a privileged database connection. This is
maintainer/database-administrator access; no operator bearer or browser cookie
is needed or issued by these commands. Do not grant database access to ordinary
console operators.

From the repository root, start the worktree's local Docker stack:

```sh
./scripts/compose-dev up -d --build
./scripts/compose-dev exec -T appview /app/cli --env dev operators allow \
  did:plc:ACCOUNT --permission-set admin
./scripts/compose-dev exec -T appview /app/cli --env dev operators list
```

Replace `did:plc:ACCOUNT` with the intended account's verified DID. The example
is illustrative and is not a real account. The CLI accepts DIDs, not handles:
confirm the account before provisioning it. Commands emit metadata only, never
credentials. The returned `operatorId` is used for changes:

```sh
./scripts/compose-dev exec -T appview /app/cli --env dev operators set \
  OPERATOR_ID --permission-set mod
./scripts/compose-dev exec -T appview /app/cli --env dev operators disable OPERATOR_ID
./scripts/compose-dev exec -T appview /app/cli --env dev operators revoke-sessions OPERATOR_ID
./scripts/compose-dev exec -T appview /app/cli --env dev operators audit OPERATOR_ID
```

The first admin is allowlisted using the same controlled maintainer operation.
There is no first-visitor promotion or public registration. Duplicate DIDs are
rejected rather than silently changing an existing account's permissions.
`list` shows active DID-bound operators. `audit` shows their latest 100 lifecycle
events in descending sequence order, with maintenance operations explicitly
attributed as `maintenance:operator-cli`. It does not claim an OAuth-verified
identity for the maintainer process.

A disabled identity cannot log in; disabling also deletes its current browser
sessions. `revoke-sessions` logs out all its browser sessions while leaving the
allowlist entry active. Set changes apply on the next admin request without
requiring another login. Already-running commands retain their transaction
semantics. There is no CLI re-enable command in this stage: restoring a disabled
row requires a reviewed maintenance operation preserving its DID and history.
Globally terminal DIDs are also denied even if an allowlist row remains active.
This does not require ordinary CraftSky membership. The operator/audit identity
is retained as private security history; pending attempts join terminal purge.

## Permission sets

| Set | Access |
|---|---|
| `admin` | All 15 defined operator capabilities, including operator management, restricted evidence and specialist workflows |
| `mod` | Moderation reads/commands, safety status, intake management, safe incident reads/confirmation and scan retry |
| `helper` | Safe detail for assigned incidents only; no global queues or moderation commands |

The existing `safety_operators.role` stores the set using its established
`safetyAdministrator`/`moderator`/`helper` names. The identity API exposes
`permissionSet` as `admin`/`mod`/`helper`. The set determines the explicit
capability map; legacy per-operator grant rows are not combined with it.
Incident assignment and domain preconditions still apply. Assignment management
and browser operator administration are later console slices.

DIDs are stable allowlist keys. Handles are login/display conveniences: changing
a handle does not change access, and handle reuse by another DID grants none.
Replacing an account means a new DID-bound operator and disabling the old one;
do not rewrite historical actor references.

## Browser login and sessions

Open the local AppView URL printed by `compose-dev`, with `/admin/` appended.
Enter the account handle or DID and complete authentication at its provider.
CraftSky requests only the identity scope `atproto`. Operator login does not
create consumer membership, initialise profiles or write PDS records.

AppView verifies the account/server authority, rechecks the allowlist at callback
and creates a local browser session. The session is in a host-only HttpOnly,
SameSite=Lax cookie, Secure on HTTPS. Exact HTTP `127.0.0.1` origins are permitted
only for local development. Session expiry is eight hours absolute or 30 minutes
idle. Use Sign out to revoke that session; it does not sign out the PDS or the
consumer CraftSky app.

Cookie-authenticated changes require the configured Origin and browser CSRF
token. Attempts are browser-bound, one-use, expire after ten minutes, and have a
global capacity limit. Expired attempts are reclaimed on admission. OAuth tokens
are discarded after identity verification; they are not stored in consumer
OAuth tables or delivered to the browser.

The isolated client endpoints are:

- `GET /oauth/admin/client-metadata.json`
- `GET /oauth/admin/jwks.json`
- `GET /oauth/admin/callback`
- `GET /v1/admin/auth/csrf`
- `POST /v1/admin/auth/login` with `{identifier}`
- `POST /v1/admin/auth/logout`
- `GET /v1/admin/identity`

The production operator client derives its own client ID/callback paths from
the configured OAuth public origin. It initially shares the server signing-key
configuration with the consumer client; key rotation must update both published
client documents. No new production client configuration was applied during
development. See [ADR 018](../../adr/018-atmosphere-operator-console-sessions.md).

If sign-in fails, return to `/admin/` and begin again. Do not copy callback URLs,
cookies, CSRF tokens or provider request URIs into logs or support messages.
Inspect protected diagnostics using the returned request reference. Access
failures stay closed when the registry or session database is unavailable.

## Development and migration

For frontend type/build checks:

```sh
cd appview/admin
npm ci --no-audit --no-fund
npm run build
```

For the local Vite UI, set `APPVIEW_ORIGIN` to the worktree's loopback AppView
origin and run `npm run dev`. Its proxy translates only the exact
`http://127.0.0.1:5173` Origin into the configured AppView origin. OAuth returns
to AppView's built shell; ordinary verification uses that built shell directly.

Migration 94 adds DID binding, browser sessions and isolated pending attempts;
it revokes existing active human bearer tokens. It leaves historic actors,
grants and audits intact. Its down migration does not revive tokens and should
not be used for credential recovery. The old Retool resource binding is no
longer part of the production authentication path. The earlier Retool reports
remain historical integration evidence.
Migration 95 adds the DID/state index required for bounded terminal-attempt purge.

The old admin commands in `scripts/moderation-dev` now explain that browser
login is required. Its ordinary development reporting/history paths remain
available. `CRAFTSKY_OPERATOR_TOKEN_FILE` is no longer a supported login path.

## Console API commands

The [API contract](../../appview/internal/api/moderation_console.openapi.json) and
[capability map](../changes/2026-10-14-retool-moderation-console/05-api-capability-map.md)
cover the browser moderation surface. Use the operator cookie and CSRF token;
consumer/legacy bearer credentials do not authorize these routes.

Safety, intake and eligibility mutations require `Idempotency-Key` (16–128
printable ASCII characters). Existing-resource mutations also need the current
`expectedRevision` from a read response. Eligibility's initial revision is zero;
scan retry additionally requires the current `expectedVersion` scan generation.
After an uncertain response, retry the exact route, payload and key. A changed
payload with the same actor/key returns 409. After a revision conflict, refresh,
review again and submit a new key. Successful safety commands return JSON with
`revision`; former 204 transitions now return 200. A replay returns its original
status/body with `Idempotent-Replayed: true`.

Migration 96 adds revisions, safe command receipts and hold-release audit.
Migration 97 adds explicit operator/external case provenance. Its downgrade
refuses to discard real operator/external cases. Apply these through the existing
migration/release process; this implementation has not changed production.

The operations API is admin-only and contains safe console command audit
metadata. Detailed case, incident, workflow and eligibility histories remain in
their domain audit records. Recording an authority reference or disclosure
approval does not transmit a report or data to an external party.

## Case review in the console

Admins and mods can use the case queue at `/admin/`, filter open/resolved cases,
and share a direct case link using `/admin/#/cases/MOD-…`. The workspace presents
safe text only; restricted incident review and binary evidence are later slices.
Reports are allegations, and account standing counts cases rather than reports.

Review the decision summary before confirming it. Internal evidence notes and
owner-visible explanations are separate. Appeal reversal, effect changes and
severe-suspension restoration also have their own confirmations. Other case
restrictions can remain after one effect is lifted.

Keep the page open while an outcome is uncertain. Use **Retry exact command**;
it retains the original body, revision and idempotency key. Do not recreate the
command in another tab. On conflict, review fresh case data and confirm again.
A successful command followed by a failed refresh remains committed; refresh
before acting again. Closing the page or losing the session loses in-memory
recovery, so inspect case history before creating a new command after sign-in.

Migration 98 retains internal appeal-resolution rationale, separately from owner
history. It recovers earlier changed-appeal rationale where effect history holds
it; earlier upheld rationale may be unavailable. Downgrading refuses to discard
retained rationale and needs an explicit reviewed data decision if any exists.

For local synthetic review only, seed the migrated development database with:

```sh
./scripts/compose-dev exec -T postgres psql -v ON_ERROR_STOP=1 \
  -U craftsky -d craftsky_dev < tools/moderation-console/seed-stage3.sql
```

The seed adds benign fixtures without operators or grants. Repeating it preserves
existing decisions and history. Frontend checks run from `appview/admin` with
`npm ci`, `npm test` and `npm run build`.
