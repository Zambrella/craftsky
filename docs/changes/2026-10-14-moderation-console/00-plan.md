# CraftSky moderation console plan

Created: 14 October 2026

Revised: 16 October 2026

Status: Local implementation, validation and rollout preparation complete with
documented limits; production rollout pending

## Decision and architecture

Build a private React/TypeScript moderation web app in this repository.
Operators sign in with an existing Atmosphere
(AT Protocol) account. A private AppView table maps approved account DIDs to a
permission set, such as admin or mod. Accounts absent from that table, or marked
inactive, cannot obtain an operator session.

CraftSky remains responsible for authorisation, decisions, audit history,
retention and enforcement. The console calls authenticated AppView APIs; it has
no direct database access or parallel moderation store.

Create a separate `appview/admin/` frontend project. The initial deployment design is
to build its static assets into AppView and serve the console at `/admin/` on
the AppView origin. Browser requests use that origin's `/v1/admin/*` routes,
without another application server or cross-origin credential handling.
Use a development proxy for the local frontend; AppView continues to run in
Docker. Scope admin assets and SPA fallback explicitly.

Early integration experiments established useful command, error and evidence
behaviour. Their prototype code and resource/authentication adapters were removed
on 16 October. The original reports remain archived evidence:
[stage 1](01-integration-proof.md) and [stage 2](02-operator-access.md).

The login and standalone shell are implemented and validated in development.
Evidence and verification limits are recorded in
[03-console-login-implementation.md](03-console-login-implementation.md).
No production access has changed. Stage 2 API implementation is recorded in
[04-admin-api-implementation.md](04-admin-api-implementation.md), with its
[capability map](05-api-capability-map.md). Stage 3 case, decision, appeal and
restoration workflows are implemented and validated locally; see
[06-case-ui-implementation.md](06-case-ui-implementation.md) for browser evidence,
release checks and verification limits. Safety/intake, specialist workflows and
operator administration are recorded in [stage 4](07-specialist-ui-implementation.md);
restricted preservation, access, holds and retention in
[stage 5](08-evidence-ui-implementation.md). Final validation and production
handoff are recorded in [stage 6](09-final-validation-rollout.md).
The [prototype cleanup](10-prototype-cleanup.md) records removal of the retired
integration and verification of the standalone console after migration 100.
[Rich content previews](11-rich-content-previews.md) add account, post and event
cards with ordinary images and on-demand video.
[Context, business and version completion](12-preview-context-business-versions.md)
adds bounded quote/thread cards, account lookup previews and explicit case/current
record comparison.

## Atmosphere login and a simple operator table

Reuse `safety_operators` as the allowlist rather than creating a competing
registry. Add a unique, validated DID for each human operator. Keep its existing
operator UUID, stable audit actor reference, active/deactivated state and
timestamps. A handle is used for login/display; the DID controls access.

| Logical field | Meaning |
|---|---|
| `did` | Unique, server-verified AT Protocol account identity |
| `permissionSet` | `admin`, `mod` or `helper` |
| `active` | Whether this identity may log in or continue using the console |
| Operator/audit metadata | Existing stable ID, actor reference and timestamps |

`permissionSet` can map onto the existing `role` column:
`admin` → `safetyAdministrator`, `mod` → `moderator`, `helper` → `helper`.
It does not require a second role column or a policy editor.

Define sets centrally and translate them to the explicit capability map already
checked by admin routes and domain services:

| Set | Capabilities |
|---|---|
| `admin` | All defined operator capabilities, including operator management, eligibility review and specialist evidence/workflow controls |
| `mod` | `moderation.read`, `moderation.manage`, `safety.read`, `intake.manage`, `incident.readSafe`, `incident.confirm`, `scan.retry` |
| `helper` | `incident.readSafe` for explicitly assigned incidents only |

Admin includes the existing `evidence.access`, `evidence.preserve`,
`hold.manage`, `authority.report`, `disclosure.approve`, `workflow.manage`,
`eligibility.review` and `operator.manage` capabilities. Incident assignments
and domain-specific preconditions continue to apply. Unknown sets fail closed.

The row's selected set is authoritative for human access. Do not require a
duplicate list of individual grants, or union legacy grants with the selected
set. Resolve the stage 2 `safety_operator_permissions` path during migration
so there is one human permission model. Per-person overrides and custom set
editing are outside this plan.

Initially, use the companion CLI to provision DIDs, change sets and disable
operators, recording attributable audit events. Install the first admin DID
through a controlled maintainer/database-administrator operation. Login never
creates a row or promotes the first visitor. Browser operator administration,
added in revised stage 4, requires `operator.manage` and the same service/audit
boundary.

The initial registry CLI is maintainer tooling authorised by privileged database
access, not a reason to retain console bearer credentials. Identify its audit
source as a maintenance operation; do not present a supplied actor label as an
OAuth-verified DID. Preserve the domain service's transactional audit writes.
Account replacement creates a new DID-bound operator row and disables the old
one; it does not relabel an existing operator's history or sessions.

A handle change preserves access for the same DID. Another account acquiring
an old handle must not inherit that account's permissions.

## OAuth and session design

Use AT Protocol OAuth through the existing Indigo library and CraftSky's
identity/authorization-server validation patterns. Request the identity-only
`atproto` scope for the operator client, without CraftSky record-write,
blob-upload or account-management authority. Verify the authenticated DID and
the authority of its server. The official
[OAuth specification](https://atproto.com/specs/oauth#identity-authentication)
supports this identity-only use case.

Give operators a separate OAuth client metadata document, callback/state context
and local session purpose. Reuse validated protocol machinery rather than the
consumer login completion workflow. Current
[callback handlers](../../../appview/internal/auth/handlers_oauth.go) initialise
profiles and create mobile handoffs; the
[flow coordinator](../../../appview/internal/auth/oauth_flow.go) couples ordinary
login to consumer lifecycle/onboarding. Operator login must not enrol a CraftSky
member, initialise a profile or write PDS records. An approved Atmosphere account
need not already be a CraftSky consumer account.

Planned flow:

1. Accept a handle or DID and resolve it server-side, including handle verification.
2. Check for an active allowlist row before starting OAuth. Return a generic
   access-denied result for absent/inactive accounts. This early check saves an
   external round trip; it does not authenticate the account.
3. Start a browser-bound OAuth attempt with PKCE, PAR, state, DPoP and the
   expected DID/issuer. Bind it to the operator client and initiating browser.
4. Complete protocol/authority checks at callback and verify the returned account
   DID. Re-read the allowlist using that verified DID. Removal/deactivation during
   the flow must prevent session issuance.
5. Only then issue a local operator session and redirect to a fixed console
   destination. OAuth tokens and operator bearers never enter redirect URLs or
   frontend response bodies.
6. On every admin request, validate the session, re-read the current active row,
   derive its set and assignments, and construct the trusted moderator context.
   Each route independently checks its required capability. The browser cannot
   select an actor, DID or role through request fields.

There is no automatic registration, password collection, app-password flow or
shared operator account. Consumer login or PDS authentication alone grants no
operator access.

Use a separate opaque operator session with a server-side verifier, bounded
absolute/idle expiry and explicit revocation. Deliver it in a host-only,
`Secure`, `HttpOnly`, `SameSite=Lax` cookie. OAuth credentials and DPoP keys
remain server-side, isolated from consumer client/session context. Denied and
abandoned attempts receive bounded cleanup and leave no operator session.

Validate request origins and apply CSRF protection to cookie-authenticated state
changes, including login initiation and logout. OAuth callbacks use their own
browser/state binding: a callback copied into another browser cannot establish a
session. Use fixed redirect destinations and reject replayed/invalid attempts.
Return private data with `no-store` and exclude it from offline/service-worker
caches and diagnostic payloads. Do not put credentials in browser storage.

Disabling/removing an operator, changing their set or releasing an assignment
affects subsequent requests, including existing sessions. Do not cache grants
for the session lifetime. Logout revokes only the local operator session and
clears its cookie; it does not log the person out of their PDS or consumer app.
Revocation does not cancel an already-committed command.

Keep `GET /v1/admin/identity` for verified DID, permission-set identity,
effective capabilities and assignment context; remove legacy external-UI fields
from the console contract. Plan login/logout under `/v1/admin/auth/*` and
separate public operator OAuth discovery/callback endpoints.

Before implementing routes, record the operator client/session decision and
amend the API/auth architecture documents. Admin cookies are an explicit scoped
addition to today's bearer/device-header contract. Consumer auth keeps its
existing contract; operator sessions cannot access consumer endpoints. Existing
OAuth URL validation assumes consumer paths: extend it with exact operator
endpoints rather than weakening validation.

## Full console scope

| Area | Operator functionality | Backend work needed |
|---|---|---|
| Cases | Report queue, filters, subject review, decisions and audit timeline | Safe content previews, subject search and richer queue context |
| Enforcement | Formal warnings, visibility warnings, hiding, takedowns, strikes, severe suspensions and restoration | Reuse commands and expose permitted actions |
| Appeals | Correspondence linking, confirmation, reconsideration and reversals | Browse correspondence and complete appeal details |
| Safety incidents | Prioritisation, assignment, deadlines, confirmation and case creation | Complete incident reads and expose the confirmation service |
| External complaints | Intake, correspondence and case/workflow linkage | List/detail APIs and complaint-to-case linkage |
| Specialist workflows | Intimate-image complaints, credible threats, authority requests and child-safety reporting records | Workflow reads, missing transitions and consistent command protection |
| Evidence | Preservation, audited access, holds and retention status | Metadata reads, hold release and dedicated restricted access |
| Accounts and operations | Account history, eligibility review, scanner failures/retry and operator administration | Account reads, retry route and DID/set management tooling |

Normal posts, project posts, accounts and business events need useful review
presentations. Distinguish reported and current record versions, explicitly
represent unavailable content, and preserve image-safety/evidence boundaries.

Private case, strike, appeal and enforcement state remains authoritative in
AppView Postgres. Moderation does not delete a user's PDS records. Public labels
and Ozone integration remain separate extensions. `prepared_not_submitted`
in the existing report forwarder is not an external submission.

## Retained work and migration

Retain stage 2's identity endpoint, route permission checks, role ceilings,
assignment checks, lifecycle audit and tests. Adapt the registry/CLI to
DID-bound sets and OAuth sessions. Preserve historical actor references; do not
infer a DID from an external UI identifier or bind synthetic operators to real accounts.

CLI-issued bearer credentials are not the console login mechanism. Retire the
legacy external UI binding dependency and routine human token
issuance during migration. Revoke superseded human tokens rather than leaving
a parallel bearer path around the DID allowlist and permission sets. Future
service credentials need a separately defined use case and are not required here.

Useful components, types and the tested action state machine now live in
`appview/admin/`, using a typed HTTP client. The old generated UI snapshot,
standalone proof server and proof-only adapters have been removed.

Use normal HTTP status handling and the canonical
`{error, message, requestId}` envelope. Send `Idempotency-Key` and expected
revision directly to real command routes. Do not promote proof-only JSON
adapters or a base64 evidence envelope into production contracts.

The original backend proof passed on 14 October. It is historical evidence;
Atmosphere login and the standalone UI have separate verification in revised
stages 1 and 3 below.
External prototype verification is no longer a gate. Local prototype code is
retired; migration 100 removes the unused external identity column while
preserving DID bindings, actor/audit history and sessions. Applied migrations and
dated reports retain their historical identifiers. Remote resource cleanup and
production mutations remain separate operations.

## Revised implementation stages

### 1. Atmosphere operator login and console shell

Record the scoped API/auth architecture change. Add DID-bound sets, audited
provisioning/disable/set-change tooling, isolated OAuth and cookie sessions.
Reuse the route/domain permission boundary. Add the `appview/admin/` build, scoped
static serving and minimal login/identity/logout screen.

Start with synthetic operators/test identity providers, then verify an explicitly
allowlisted real Atmosphere account end to end in development.

Completion evidence: listed accounts reach the shell with the correct DID/set;
unlisted accounts cannot establish sessions; changes and disabling are enforced
on subsequent requests. No consumer onboarding or PDS write occurs.
This milestone is complete and passed the full local release gate. The
admin API stage is also complete and has passed its full local release gate.

### 2. Complete the admin API

Add bounded opaque-cursor list/detail APIs for workflows, intakes, evidence
metadata, scans and account moderation history. Add safe previews,
system-detected incident confirmation, manual scan retry and attributable
external/operator case provenance. Confirm profile-image incident mapping.
Include active incident states, owners, covers, priorities and deadlines in
application work readers.

Complete missing workflow and hold-release transitions. Give safety commands
revision/replay protection where duplicate or stale execution could produce
incorrect transitions or repeated records. Derive actor/provenance from trusted
context. Align camelCase JSON, strict decoding and OpenAPI with the real routes.

Completion evidence: an endpoint-to-console capability map and integration tests
for reads, commands, sets and assignment boundaries.
Implementation and verification are recorded in [stage 2](04-admin-api-implementation.md)
and the [capability map](05-api-capability-map.md).
This milestone is complete and passed the full local release gate.

### 3. Cases, decisions, appeals and restoration

Port the prototype queue/detail layout and useful components. Add content
presentations, filters, timelines and consequence-aware confirmation dialogs.
Keep internal evidence notes separate from user-visible explanations. Use
CraftSky styling with loading, empty, unavailable and error states, keyboard
navigation and direct detail links.

Pending commands show a spinner on the actioned control and disable conflicting
controls. Uncertain outcomes retain the exact submitted payload, expected
revision and idempotency key for retry. Refresh after success; refresh failure
must not turn a committed success into a failed command. Conflicts require
refreshed data and renewed review with a new key.

Completion evidence: report → decision → enforcement → appeal → reversal works
through the browser, with correct attribution, effects, strikes and standing.

This milestone is complete and passed the full local release gate. Browser
evidence, migration 98 and remaining verification limits are recorded in
[stage 3](06-case-ui-implementation.md).

### 4. Safety, intake, specialist workflows and operations

Build incident queues/details, assignments, deadlines, confirmation, complaints/
correspondence, specialist workflows, eligibility review, scan retry and account
history. Reach every planned action with its appropriate permission set.
Helpers see only assigned safe incident detail.

Add a simple admin-only operator screen to list approved DIDs, add an account,
select its permission set, disable access and revoke sessions. Use the same
registry and audited services as the maintainer tooling; do not expose raw
credentials or turn it into a custom permission-policy builder.

Recording an external report/disclosure reference must remain distinct from
actually transmitting it to an authority or another service.

Completion evidence: synthetic workflows with matching persisted state,
attribution and denied-access tests, including stale/repeated commands.

This milestone is complete locally. Browser workflows, authorization checks,
migration 99, explicit deadline configuration and verification limits are
recorded in [stage 4](07-specialist-ui-implementation.md).

### 5. Restricted evidence handling

Display safe metadata separately from bytes. Reuse dedicated AppView evidence
access with explicit capability, incident boundaries, reason, audit and
`no-store`. Validate direct browser binary transport. Keep private object keys
and credentials out of responses and restricted bytes out of general caches,
analytics, diagnostics, source fixtures and developer tool output.

Expose preservation, access, hold creation/release and retention status.
Explain unavailable operations when private storage is not configured.

Completion evidence: authorised/denied access, attributable audit, hold/retention
transitions and protected-value canary checks.

This milestone is complete locally. Preservation, scoped binary access, hold
creation/release and observed worker deletion, 42 frontend tests and the full
AppView release gate are recorded in [stage 5](08-evidence-ui-implementation.md).
The upload allowance is 512 KiB; narrow-screen visual verification remains open.
Final validation and rollout preparation are next.

### 6. Validate and roll out

Exercise complete workflows with synthetic cases and benign media:

- Report → decision → enforcement → appeal → reversal.
- Incident → confirmation → case → reporting record → resolution.
- External complaint → correspondence → case/workflow linkage.
- Evidence preservation → access → hold → release → retention handling.
- Eligibility restriction → review → restoration.
- Dead-lettered scan → manual retry → resulting state.
- DID allowlisting → OAuth login → permitted action → disable → denied access.

Check actual standing/content visibility and restoration convergence alongside
HTTP bodies. Run focused Go integration/race tests, `./scripts/appview-check`,
Docker smoke checks, OpenAPI validation, frontend type/build checks, meaningful
component tests, browser workflow checks and `git diff --check`.

The checked-in Render configuration disables moderation admin routes. Live
configuration has not been verified for this plan. Production allowlisting,
OAuth configuration, route enablement, infrastructure changes and deployment
remain explicit production operations under the exact-tag release process.
Add the frontend build to that process and update operational guidance for
sessions, permission changes, revocation and last-admin recovery before rollout.

Local validation and rollout preparation are complete. [Stage 6](09-final-validation-rollout.md)
records the fresh real consumer HTTP visible → hidden → restored check, keyboard
confirmation/cancellation, workflow coverage, Docker frontend tests and the full
AppView release gate. The release image now tests/builds the frontend, and the
gate checks its packaged shell/assets and unauthenticated identity boundary.
The operator/release runbooks cover production setup, recovery and rollback.
Narrow-screen visual acceptance remains open after repeated preview resize
timeouts. Production inspection, provisioning, configuration and exact-tag
deployment remain separate explicitly authorized operations.

## Authentication acceptance checks

| ID | Required result |
|---|---|
| AUTH-01 | An active listed DID authenticates through its authoritative server and receives only the selected set's capabilities |
| AUTH-02 | Unlisted/inactive accounts, forged DID/role fields and mismatched callback identities cannot establish operator sessions |
| AUTH-03 | Removal/deactivation during callback prevents issuance; changes after login affect the next request |
| AUTH-04 | A handle change preserves the same DID's access; another DID acquiring that handle gains none |
| AUTH-05 | Mod cannot call admin-only operations; helper cannot browse global queues/unassigned incidents; direct API calls match UI restrictions |
| AUTH-06 | Consumer sessions, legacy human bearers and external-resource credentials cannot substitute for operator login |
| AUTH-07 | Logout, expiry and revocation reject sessions; browser storage/URLs/responses contain no OAuth tokens or reusable operator bearers |
| AUTH-08 | Invalid state/issuer/account, cross-browser callbacks, replay, CSRF and arbitrary redirects are rejected |
| AUTH-09 | Login needs no consumer membership and creates no onboarding/profile state or PDS record |
| AUTH-10 | Registry/session lookup failure denies access; actions and set/assignment changes retain stable actor attribution |

## Governing references

- [Architecture reference](../../../atproto-craft-social-app-reference.md).
- [API architecture](../../superpowers/specs/2026-04-21-appview-api-architecture-design.md).
- [OAuth BFF design](../../superpowers/specs/2026-04-18-appview-oauth-bff-design.md).
- [AT Protocol OAuth specification](https://atproto.com/specs/oauth).
- [Product vision](https://docs.google.com/document/d/11wu5ZFifrhx3HwdqOR5-7WQiESq5MUKk7vTa_U8fl-c/edit).
- [Logging and error reporting](../../development/logging-error-reporting.md).
- [Moderation strikes requirements](../2026-09-09-moderation-strikes/01-requirements.md).
- [Online safety readiness requirements](../2026-09-16-online-safety-launch-readiness/01-requirements.md).
- [Current CLI runbook](../../operations/moderation-operators.md) — documents Atmosphere login, DID provisioning, permission sets and session revocation.
- [Production Render runbook](../../operations/production-render.md).
- [Release operations](../../operations/releases.md).
