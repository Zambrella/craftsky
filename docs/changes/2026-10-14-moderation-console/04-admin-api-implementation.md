# Custom console stage 2: admin API

Status: Complete and validated locally on 15 October 2026

This implements revised stage 2 of the approved console plan. The existing case,
strike, appeal, enforcement, eligibility and specialist services remain the
policy boundary. No production changes or frontend workflow screens are included.

## HTTP contract and permissions

The [capability map](05-api-capability-map.md) covers every moderation/safety
operation, its required capability and its planned console destination. The
[OpenAPI contract](../../../appview/internal/api/moderation_console.openapi.json)
documents requests, responses, cookies, CSRF, revisions and replay behavior.
Atmosphere authentication remains in the separate operator-login contract.

Added bounded opaque-cursor reads for incidents, workflows and their timelines,
external intakes and correspondence, evidence metadata, holds, scans, account
moderation history, case provenance and console command audit. Pages default to
50 and cannot exceed 100. Cursors bind the resource, parent and filters. Stable
time/UUID ordering handles equal timestamps. Unsupported/duplicate query keys
and malformed cursors are rejected. Lists return an empty `items` array and
omit the final cursor.

Workflow/intake/incident lists support `kind`; scans support `jobState`; evidence
and holds support `incidentId`. Resource detail returns 404 for a missing record.
Child collections return an empty page when no matching records exist. New
queries select explicit camelCase wire fields and have supporting page indexes.
They do not serialize entire storage rows.

Helpers retain only assigned safe incident detail. They cannot browse global
queues, subject references, workflows, evidence, account history or operations.
CSEA classification/reporting/resolution now declares `authority.report`, matching
the existing specialist service. Ordinary detection confirmation remains a mod
capability, `incident.confirm`.

## Commands and provenance

Safety/intake/eligibility commands require an `Idempotency-Key` of 16–128 printable
ASCII characters. Commands acting on an existing resource additionally require
its current `expectedRevision`. Eligibility uses zero before its first review.
A successful command returns the current revision; state changes that previously
returned 204 now return a JSON result with HTTP 200.

The operation and its replay receipt share a SQL transaction. A transaction-level
advisory lock serializes concurrent uses of one actor/key. Resource locks and
revision checks prevent stale changes, including concurrent first eligibility
reviews. An exact retry returns the original status/body and
`Idempotent-Replayed: true`, even after the resource's revision changes. Different
input with the same actor/key returns 409. Current authentication and capability
checks always run before replay. Case adjudication retains its existing
revision/idempotency implementation.

Receipts contain only a key digest, request digest, safe response, trusted actor,
route pattern and non-capability resource UUID. Raw requests, restricted bytes,
private object keys, cookies and OAuth credentials are not retained. The
admin-only operations reader exposes audit metadata, without digests or stored
response bodies. Existing domain timelines remain the detailed action history.

Added:

- Human confirmation of a detected incident into the existing moderation-case
  lifecycle, without manufacturing a user report.
- Operator-created and external-intake cases, retaining actor, source, intake
  linkage, creation reason and time. External case subjects must match the
  intake's canonical AT URI/profile subject; unrelated account linkage fails.
  Website-only intake URLs need canonical subject resolution before this path.
- Dead-lettered scan retry using both job revision and scan generation.
- Credible-threat outcome completion and audited legal-hold release.
- Eligibility status/revision reads and guarded review/reversal commands.
- Legal classification on the existing moderation decision request.

Actor, source, approval identity and replay identity are derived from trusted
operator context/header boundaries. Safety bodies no longer accept arbitrary
`sourceSystem`, `replayId` or `approvedBy`. External appeal-link source/replay
identity follows the same rule. Existing real provider message references remain
metadata, distinct from authenticated operator identity.

The work summary includes active `caseCreated` and `reporting` incidents, workflow
and unlinked-intake work, owners, covers, priorities and deadlines. It returns at
most 100 work items, with `workTruncated` when more exist; callers use the paged
resource lists for the remainder.

## Safe previews and evidence boundary

Case previews use an explicit allowlist of indexed text/title fields. Post/event
previews require the captured CID and clear image-safety state. Missing or changed
versions return `unavailable`; restricted incident subjects return `restricted`.
Account previews use the captured profile version where available. No raw PDS
record, remote media fetch, image URL or restricted bytes enter these previews.

Evidence preservation now returns an ID without an object key. Restricted byte
access remains a separate permission/reason/audit path with `no-store`; it is
never buffered in a replay receipt. Known SQL rollback removes objects newly
written by that attempted preservation. Object storage is a separate system: an
ambiguous commit keeps bytes intact for safe receipt reconciliation. Production
object-store failure recovery and direct browser binary transport remain part
of the planned restricted-evidence stage; this stage uses benign in-memory
objects for API tests.

## Reproduced integration defects

The complete migration stack exposed a stale `ON CONFLICT` target in machine
match recording: migration 84 had removed the named uniqueness constraint. The
recorder now locks the incident and checks for an existing detection event,
working with both historical and current schemas without duplicating detection.

Profile and business-profile detections map to account cases; business-event
subjects map to event cases. Profile account cases retain canonical account
identity rather than invalid record-only fields. A fully migrated integration
test records a profile match twice, confirms it through HTTP, reads the account
case and checks that the active incident remains in the work summary.

## Validation

The approved seam is the registered HTTP API, backed by real migrated PostgreSQL
and synthetic benign data. Operator context is supplied by the test boundary;
Atmosphere callback/cookie behavior was verified in stage 1 and is unchanged.
No external service or authority is contacted by these acceptance tests.

Observed passing checks:

- Workflow pagination with equal timestamps, page limits, invalid cursors and
  denied helper access.
- Scan metadata canaries excluding blob and lease details.
- Bounded empty collections and missing detail responses.
- Exact/changed/concurrent command replay, stale workflow review and completion.
- Profile detection recording, account-case confirmation and active work.
- Scan retry and hold release, stale release rejection and denied mod release.
- Evidence preservation responses/replay excluding object keys and bytes.
- Operator/external case provenance, unrelated-intake rejection, forged actor
  rejection and strict duplicate/trailing JSON rejection.
- Eligibility review/reversal replay and stale-update protection.
- Existing affected API, domain and composition suites; focused race suites;
  route inventory, access classes, diagnostic manifest and terminal DID inventory.
- OpenAPI 3.0 validation with `openapi-spec-validator`.

The initial broad focused run flagged the deliberately exact route/access/
diagnostic inventories; they were updated and pass. An initial full gate was
stopped during ordinary tests when contract review identified missing eligibility
protection. The final `./scripts/appview-check` run used frozen Go source and
exited successfully with all release gates passed: 49 ordinary package suites,
49 race package suites, Go vet/Staticcheck, generated-source checks, Docker and
packaged media-memory checks, migrations, API smoke checks and vulnerability
scans. No affected code or imported-package vulnerability was found; one
required-module vulnerability is outside the code's call paths. The pinned Tap
handshake passed; deterministic event injection is unsupported by that version
and was not claimed.

The final log is `/private/tmp/craftsky-console-api-verified.log`, with artifacts
in `/private/tmp/craftsky-console-api-verified`.

The worktree's Docker development stack was rebuilt and migrated to version 97
with `dirty=false`. Both existing local operator rows remain active and no
operator session was created by this stage. The shell returns HTTP 200 and a
new workflow API request without a session returns HTTP 401 with
`operator_authentication_required` and a request ID. Authenticated command
coverage comes from the real-PostgreSQL HTTP integration tests; no fresh provider
login or browser workflow screen verification is claimed for this backend stage.

## Migrations and handoff

Migration 96 adds resource revisions, replay receipts, hold-release audit and page
indexes. Migration 97 adds explicit operator/external origins and case provenance.
Downgrading 97 refuses to discard attributed operator/external cases; it requires
an explicit reviewed data decision if such cases exist.

Case/appeal/restoration screens are the next approved stage. Specialist screens,
operator administration, browser evidence transport and end-to-end rollout
remain the later stages. This work does not provision production operators,
change Render configuration, deploy, publish, commit or push.
