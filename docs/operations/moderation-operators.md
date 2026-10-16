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
Incident assignment and domain preconditions still apply. Admins manage assignments and the DID registry in the console. Helpers can
open only safe metadata for their current incident assignments.

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
not be used for credential recovery. Earlier external-UI experiments remain
archived integration evidence, not supported deployment instructions.
Migration 95 adds the DID/state index required for bounded terminal-attempt purge.

Migration 100 removes the unused external UI identity column. Operator UUIDs,
DIDs, permission sets, browser sessions and audit history remain intact. Its down
migration restores an empty optional column for schema compatibility; it cannot
recover discarded external bindings or revive credentials. The retired proof
server, generated UI snapshot and database bearer authenticator are removed.

The old admin commands in `scripts/moderation-dev` now explain that browser
login is required. Its ordinary development reporting/history paths remain
available. `CRAFTSKY_OPERATOR_TOKEN_FILE` is no longer a supported login path.

## Console API commands

The [API contract](../../appview/internal/api/moderation_console.openapi.json) and
[capability map](../changes/2026-10-14-moderation-console/05-api-capability-map.md)
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
and share a direct case link using `/admin/#/cases/MOD-…`. The content preview shows indexed account cards (bio, avatar, banner and crafts),
posts (author, text, photos, project/link cards and on-demand video), and events
(image, dates and venue). Images can be enlarged; videos load only when requested
and never autoplay. Media failures show an unavailable state. Quoted posts and
reply parent/root cards are bounded to one level and each receive separate safety
and version checks. Record-version details compare the case's captured CID with
the current index. Changed captured content is unavailable; safe current content
is shown in a separate card. Older content is not reconstructed. Account cases
without a captured CID are explicitly labelled as current indexed profiles.
Current personal/business record CIDs are listed separately. Blocked subjects and
terminal owners remain unavailable. Incident-linked content remains restricted. Use the separate
restricted evidence workspace for explicit, audited access; ordinary previews do
not fetch evidence bytes or contact a PDS. Public media uses the existing image
CDN and configured video service with case URLs suppressed from referrers.
The Accounts tab also has a current profile preview, including business types,
offerings, tagline, hours, service area, location, primary action and product
cards/prices/images. Business records are visible for moderation independently of
subscription eligibility; their own image-safety projection must be clear.
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


## Specialist and operator workspaces

The navigation exposes only workspaces allowed by the current permission set.
Incident owners/covers/helpers are assigned from active DID-bound operators.
Changing an owner or cover replaces that slot; releasing a helper assignment
removes its incident access on the next request. Queue and history pages use
opaque pagination. Account lookup accepts an Atmosphere DID and shows standing,
moderation case history and, for admins, age eligibility review/restoration.

External intake records message references and attachment disposition without
uploading media. Follow-up correspondence stays on the intake. Case creation
checks the indexed subject against the intake; verified appeal linking also
checks the case owner. A new case does not itself impose consequences.

Specialist authority forms record verification, a lawful basis, minimized field
names and closure. Incident reporting forms record external report references,
information request/response tracking and resolution. These forms do not send
reports, responses or disclosed data to third parties. Use the separate restricted evidence workspace for preservation, audited access
and legal holds.

Intimate image workflows require an explicitly approved response target:
`SAFETY_INTIMATE_IMAGE_RESPONSE_WITHIN`, expressed as a positive Go duration
such as `24h`. The example is a synthetic test value, not a legal or production
policy. There is no default. An unset/zero target rejects creation with HTTP 422
`deadline_target_not_configured` before committing. Configure a reviewed target
through the normal production configuration controls before enabling that work.
For local synthetic walkthroughs only, set the variable in gitignored `.env.local`
and rebuild the worktree stack. Production configuration was not changed.

Admins can list/add DIDs, change preset permission sets, disable access and revoke
browser sessions in **Operators**. These commands use the existing audited
registry service with a trusted browser actor and fresh resource revision.
The browser prevents disabling or demoting the last usable administrator, even
for concurrent requests. The privileged CLI remains a controlled recovery path;
it can override that browser guard. It does not offer a browser re-enable flow.
Do not disable the last administrator through maintenance tooling except as an
explicitly reviewed recovery operation. Migration 99 adds operator revisions.

Seed the stage 4 local fixtures after migrations with:

```sh
./scripts/compose-dev exec -T postgres psql -v ON_ERROR_STOP=1 \
  -U craftsky -d craftsky_dev < tools/moderation-console/seed-stage4.sql
```

The seed adds a benign indexed post, detection metadata and a failed scan job.
It contains no restricted media, operators or grants and preserves existing
walkthrough results. The scan fixture uses a synthetic scanner identity; a real
provider scan is not claimed by the retry walkthrough.


## Restricted evidence workspace

Admins can open safe evidence metadata and legal holds from navigation or an
incident. Ordinary moderators and helpers cannot preserve or fetch evidence
bytes. No bytes are loaded when a metadata page opens. Use **Review evidence
access**, provide a reason, check the incident/evidence identifiers and confirm.
The dedicated same-origin POST streams binary bytes with `no-store` and
`nosniff`; it requires the matching `incidentReference` and browser CSRF token.
It has no command replay key: each attempted access is audited separately.

Images are shown only in an in-memory object URL. **Hide evidence**, navigation,
page hiding, focus loss or access removal clears the view and revokes the URL.
A new access requires a new review. Browser responses are size-bounded, and
private bytes do not enter local/session storage, general metadata, receipts,
analytics or diagnostics. Avoid screenshots or recordings while evidence is
open. Retain an authorised workflow reference when seeking support, not content,
credentials or private storage paths.

Preserve a local image only after obtaining it through the approved evidence
procedure. Choose a JPEG/PNG/GIF/WebP file, record a reason and a future retention
deadline, then confirm its metadata. Supported signatures are checked, original
bytes are retained with a server digest, and the current console limit is
512 KiB or the smaller allowance returned by the capability endpoint. Larger
files need a separately approved preservation procedure; this console does not
silently resize or recompress evidence. Uncertain preservation results retain
the exact submission in memory for **Retry exact command**.

The console explains when private storage is not configured. This is configured
capability reporting, not a live provider-health guarantee. Metadata and existing
hold administration remain separate from storage access. Production evidence
storage configuration is still a controlled production operation.

Create a hold by choosing retained evidence within its incident, entering the
approved basis and expiry, and confirming. Creation/approval use the trusted
administrator actor. Release is revision-protected and requires a reason. Other
active holds can still prevent deletion. Evidence metadata shows active coverage,
retention deadline and deletion state; separate cards show access attempts,
retention events and bounded job status without object keys or leases. The
existing worker deletes due unheld objects and retains metadata/audit. Deleted
evidence access returns 404. No generic manual evidence-delete or PDS-delete
operation was added.

## Production rollout

The checked-in Blueprint keeps `MODERATION_ADMIN_ENABLED=false`. Local
allowlisting and a successful local build do not enable production access.
Review the intended production workspace/service, configuration and operator
DIDs before applying changes through the existing
[Render runbook](production-render.md). Do not copy development fixtures,
synthetic deadline targets or local allowlist entries into production.

1. Prepare the reviewed application on clean, synchronized `main`. Create the
   next explicit AppView version/tag, run `just appview-check` on that exact
   commit, then use the documented release-push/exact-tag deployment sequence
   in [Releases](releases.md#appview-release). Keep the migration and health
   receipts. The image includes the console; there is no separate hosting step.
2. Review `MODERATION_SOURCE_DID` and the HTTPS `OAUTH_PUBLIC_ORIGIN`. The
   operator client derives its discovery/JWKS/callback paths from that origin
   and uses the existing confidential-client signing configuration. Do not add
   broader consumer OAuth scopes to the identity-only operator client.
3. Provision the first independently verified DID with
   `/app/cli --env prod operators allow VERIFIED_DID --permission-set admin`
   through the controlled maintainer shell/database connection. Review `list`
   and the returned identity's `audit`. Keep a second usable administrator
   before performing deactivation/demotion exercises. Never use direct SQL to
   manufacture browser sessions, credentials or per-user grant rows.
4. Review and apply route enablement separately from the application release.
   Blueprint synchronization remains manual; validate it before an approved
   sync. Ensure the reviewed configuration survives future syncs. Verify the
   public operator metadata has the expected client ID, callback and `atproto`
   scope, and its JWKS matches the configured signing key.
5. If restricted preservation/access is required, configure the private
   `SAFETY_EVIDENCE_S3_ENDPOINT`, `_REGION`, `_BUCKET`, `_ACCESS_KEY_ID` and
   `_SECRET_ACCESS_KEY` variables through the secrets/configuration process.
   Keep the bucket private and separate from public media, with reviewed access
   and retention controls. Configure an approved
   `SAFETY_INTIMATE_IMAGE_RESPONSE_WITHIN` before enabling that workflow.
   Otherwise, verify the console's explicit unavailable-operation states.
6. At the deployed `/admin/`, perform identity-only consent and verify the
   independently checked DID and intended permission set. Use a designated
   benign case to check command attribution, owner-visible standing/content
   changes, reversal and refresh. Test mod/helper boundaries, assignment
   release, CSRF denial and session revocation with designated test operators.
   If evidence storage is configured, use approved benign media to check
   preservation, audited viewing, hide, hold/release and actual retention.
7. Check desktop and narrow-screen layouts, keyboard confirmation/cancellation,
   session expiry/re-login and unavailable-dependency behavior. Save only safe
   metadata screenshots and bounded operational references. Inspect classified
   diagnostics without logging payloads, storage keys or credentials.

Record the exact tag/commit, migration state, configuration review, operator
permission decisions, browser results and remaining limitations in the rollout
record. Health and a rendered login page are insufficient evidence of permitted
actions or correct enforcement. Any production mutation still requires its
existing explicit authorization; this procedure is preparation, not a deploy.

### Lost access and last-administrator recovery

For an ordinary lost browser session, return to `/admin/` and sign in again.
For a suspected compromised session, use a second admin's **Revoke browser
sessions** action or the privileged CLI's `operators revoke-sessions
OPERATOR_ID`. Verify the old session is rejected; revocation alone permits a
new login while the identity stays active. Use **Disable operator access** or
`operators disable OPERATOR_ID` to remove sign-in access as well.

Browser management protects the last usable admin. Privileged maintenance
commands can bypass this guard, so check current usable admins before changing
one. If every admin was demoted but an approved identity remains active, restore
its `admin` set with the privileged `operators set OPERATOR_ID --permission-set
admin` command and review its audit. If no active approved identity remains,
independently verify a new recovery DID and allowlist it using the same controlled
first-admin procedure. There is no supported re-enable command for a disabled
DID: restoring that exact row requires a separately reviewed maintenance change
preserving its actor/history. Do not delete it and re-create its identity.

### Console rollback

For a console exposure/authentication incident, disable admin route registration
through the approved configuration process and verify both `/admin/` and admin
APIs become unavailable. Disabling routes does not erase existing sessions;
revoke affected sessions before any re-enablement. Consumer routes retain their
separate authentication contract.

If an application rollback is required, use the previous verified exact tag
under the Render runbook after checking schema compatibility. Migration 94
revocation is intentional; down migrations do not restore former credentials.
Migrations 97/98 protect retained case provenance/rationale, so do not use schema
down as a routine rollback. Preserve operator and command audits, holds and
private evidence. If storage access fails, repair configuration/provider access
and verify retention convergence rather than removing holds or evidence rows.
