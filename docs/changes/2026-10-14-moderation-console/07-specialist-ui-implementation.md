# Custom console stage 4: safety, intake, operations and operators

Status: Complete locally, 15 October 2026. Production rollout is separate.

## Delivered

The console navigation now includes incident review, external intake, specialist
workflows, account standing/history and eligibility, image scans, operator
administration and safe command operations. Direct resource links, filters and
opaque pagination are retained. Helpers see only their assigned safe incidents.

| Workspace | Available work |
|---|---|
| Incidents | Safe metadata, subjects, owner/cover/helper assignment and release, detection confirmation, classification/deadline, external report records, information requests/responses and resolution |
| Intake | Message metadata, attachment disposition, follow-up correspondence, indexed-subject case creation, verified appeal linking and intimate-image workflow creation |
| Specialist workflows | Intimate-image standing/declarations, judgment/search/action/outcome; credible-threat assessment/outcome; authority verification, minimized disclosure-field recording and closure |
| Accounts | Standing, linked case history, evidenced eligibility restriction and reasoned restoration |
| Scans | Queue/failure metadata and dead-letter retry with both job revision and scan version |
| Operators | DID allowlisting, preset permission changes, session revocation, deactivation and audit timeline |
| Operations | Bounded safe command receipt metadata and direct detail |

Binary evidence, preservation transport and legal holds remain stage 5. External
report/disclosure forms record actions and references; they do not transmit
anything to authorities or other recipients. Moderation remains AppView state;
no PDS record deletion or consumer OAuth credential handoff was added.

## Command and authorization behavior

The stage 3 confirmation/recovery component is shared across workspaces. It
captures the exact route, serialized body, reviewed revision and idempotency key.
Uncertain results retain that command for retry. Conflicts refresh data and
require renewed review with a new key. Known committed results remain committed
when a subsequent read fails. Navigation and conflicting controls are locked
while review/submission/recovery is pending. New case creation reads its recorded
case back before reporting refreshed data. Appeal linking exposes its returned
correspondence reference for case follow-up.

Background identity refreshes with unchanged capabilities preserve specialist
form drafts. Completing a final workflow action preserves committed feedback
after its available actions disappear. Unknown response fields are not rendered
as raw JSON. Restricted bytes, object keys, session cookies, CSRF tokens and
request bodies are excluded from the curated metadata views.

New operator/assignment HTTP commands reuse the audited registry service and
trusted server actor, inside the command receipt transaction. Owner/cover slots
replace previous assignments; helper assignments are additive. Invalid or
inactive targets are rejected. Operator management is admin-only; helpers cannot
list queues, subjects, assignments, operators or other private workspaces.

Migration 99 adds operator revisions. Registry writes share the maintenance
serialization lock and prevent browser removal/demotion of the last usable
administrator, including concurrent requests. The privileged CLI remains the
controlled recovery path. Session revocation bumps the operator revision.
Route/access/diagnostic inventories, OpenAPI and the capability map are aligned.

## Deadline configuration gap repaired

The real browser walkthrough exposed that app composition always constructed an
empty intimate-image deadline policy. Such workflows could never start.
`SAFETY_INTIMATE_IMAGE_RESPONSE_WITHIN` now wires an explicitly configured target
into the service. There is no production default; unset/zero rejects creation
with HTTP 422 `deadline_target_not_configured` before committing. The console
explains this definite rejection without leaving an uncertain command locked.
Malformed or negative durations fail configuration loading.

Only this worktree's gitignored `.env.local` was given a labelled synthetic `24h`
target for verification. It is not a legal or production policy. Production
configuration and operator grants were not changed. The runbook describes the
reviewed configuration required before rollout.

## Verification

Tests use the agreed rendered UI/HTTP boundary, real registered Go HTTP routes
with isolated migrated PostgreSQL schemas, and the shared T3 browser against the
local Docker API with benign synthetic records. Development seeds are in
`tools/moderation-console/seed-stage4.sql`; rerunning them preserves walkthrough
results and creates no operators or grants.

Observed passing checks:

- 34 frontend tests across five files, including the 15 existing case tests;
  TypeScript checking, production frontend build and npm audit (zero findings).
- Affected API, routes and operator suites against local PostgreSQL. Additional
  checks cover strict forged-actor rejection with a fresh key, exact replay,
  stale revisions, trusted attribution, admin-only commands and concurrent
  last-admin protection. Config/composition checks exercise explicit and unset
  workflow targets through a persisted workflow and registered HTTP request.
- OpenAPI 3.0 validation with `openapi-spec-validator`: 67 operations.
- Final `./scripts/appview-check`: 49 ordinary package suites and 49 race package
  suites, vet/Staticcheck, generated-source checks, Docker, packaged media-memory
  checks, migrations, API smoke checks and vulnerability scans all passed.
  Zero affected-code/imported-package vulnerabilities; one required-module
  advisory is outside the code's call paths. Pinned Tap handshake passed;
  deterministic event injection is unsupported by that Tap version.
- Final frontend-only feedback/readback/display refinements were followed by the
  complete frontend checks and a fresh local Docker build/browser verification.
  Go source stayed fixed during the successful full release gate.
- `git diff --check` passed. Local database migration 99 is clean.

The first release-gate run was stopped after the browser exposed the missing
deadline wiring. The final successful log is
`/private/tmp/craftsky-stage4-final.log`; artifacts are in
`/private/tmp/craftsky-stage4-final-artifacts`. Frontend evidence is in
`/private/tmp/craftsky-stage4-ui-tests.log`. No private browser credentials are
included in these reports.

## Browser and persisted-state evidence

The existing approved Atmosphere account renewed its identity-only consent and
used the real operator cookie/CSRF flow. Form interactions through T3 preview
were followed by API readback and bounded database checks.

| Walkthrough | Observed persisted result |
|---|---|
| Complaint, correspondence, case | Intake revision 1; correspondence attached; case origin `externalIntake` and linked provenance attributed to the signed-in operator |
| Intake to intimate-image review | Linked intake ID; configured deadline; all standing/declaration/search/judgment/action/outcome events; workflow `resolved`, revision 1 |
| Uncertain intimate-image creation | Original exact command recovered after local target configuration; one workflow created, then completed |
| Credible threat | Preservation assessment/outcome audit; workflow `resolved`, revision 1 |
| Authority request | Verified requester, lawful-process approval, selected `accountDid` field, disclosure record and post-incident review; workflow `resolved`, revision 3 |
| Incident | Human confirmation created the system case; owner/helper assignments plus cover assignment/release; classification/deadline; initial report within deadline; information request `responded`; incident `resolved` |
| Global case and operations | Operator-created account case read back successfully; its operation detail showed the trusted actor, case reference and HTTP 201 |
| Appeal intake | Canonical case subject and owner-verified appeal link persisted with trusted actor attribution; repeating the link returned the same correspondence reference and opened the linked case |
| Eligibility | Restriction revision 1, restoration to `eligible` revision 2; account history showed both related cases and zero active strikes |
| Scan retry | Job changed from `dead_letter` revision 0 to `queued` revision 1, preserving scan version 1 |
| Operator lifecycle | Synthetic DID added as helper, changed to mod, sessions revoked, then disabled at revision 3; all four audit entries use the real signed-in actor |
| Assigned helper boundary | Temporarily changed only the signed-in local operator's role; shell exposed only assigned safe incident detail and no actions; global queue, unassigned incident and registry requests returned 403 |
| Moderator boundary | Operator navigation hidden; forced registry workspace unavailable; registry and eligibility API requests returned 403 |

Both real allowlisted DIDs were verified active with administrator roles after
the temporary role checks. The synthetic operator remains disabled. The seed
contains no restricted media; no real provider scan, emergency contact, report
submission or disclosure delivery is claimed.

The T3 preview's screenshot capture failed and resize timed out despite retries.
Browser DOM/form/API verification passed, but no new screenshot or narrow-screen
visual verification is claimed. Native pointer automation also became unreliable,
so later form interactions used DOM events within the same shared T3 preview.
The previous stage's keyboard checks and current component confirmation tests
remain the available keyboard evidence.

No production configuration, production grants, deployment, commit or push was
performed. Restricted evidence handling is the next stage.
