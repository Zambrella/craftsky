# Implementation Review: Lifecycle Ingestion Reset Safety

## Verdict

Status: Changes required
Reviewer: Codex
Date: 2026-08-21
Risk level: High

## Summary

The Tap-reset correction fixes the observed profile lifecycle failure: record
winner selection no longer treats Tap's database-local event ID as a durable
repository order, and eligible effect-backed sources now carry the current
lifecycle generation. The focused ingestion, Tap, and database suites pass.

The review found two remaining source-winner correctness gaps. Repository
revisions are compared as arbitrary strings without first proving that they are
sortable AT Protocol TIDs, and the new duplicate test ignores record bytes even
though the incoming CID is not validated. Either condition can cause a delivery
to be ACKed while retaining the wrong source. Identity refresh scheduling also
still assumes Tap event IDs remain monotonic across a Tap reset. These should be
fixed before treating reset recovery as complete.

The main simplification opportunity is to stop making one
`owner_generation` column mean both effect-attempt provenance and current
projection authority. Effect provenance already has a stable operation ID;
projection authority should have a separately named, consistently current
generation. A shared source-version classifier can then replace the duplicated
generic/profile comparison branches.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-004 | Important | Behavior | Source ordering now depends on lexical revision comparison, but the WebSocket boundary accepts any non-blank string up to 128 bytes and `tap.Event.Rev` remains an untyped `string`. A malformed high-sorting value can become the durable winner and make all later valid TID revisions look stale. | AV-004/AV-005 source CAS contract; `appview/internal/tap/consumer.go:36-46,562-566`; `appview/internal/ingestion/store.go:128-150`; `appview/internal/ingestion/service.go:312-333` | Parse the revision with `syntax.ParseTID` at the Tap boundary, carry `syntax.TID` internally, and defensively reject invalid revisions in the durable ingestor. Add invalid-revision and valid-TID ordering tests. |
| IR-005 | Important | Behavior | A delivery is considered the same durable source when revision, action, and CID match, even if its record bytes differ. Because CID is only a passthrough string, an upstream defect or malformed event can be ACKed as a duplicate while the retained source and projection contain different content. The existing stored fingerprint cannot close this gap because it includes Tap event ID. | AV-004 duplicate/idempotency contract; `appview/internal/ingestion/store.go:137-148,436-449`; `appview/internal/ingestion/service.go:319-330`; `appview/internal/tap/consumer.go:556-561` | Introduce a canonical repository-source fingerprint that excludes Tap event ID and includes URI, revision, action, CID, and record bytes; use it for duplicate/conflict classification. Keep a separate delivery fingerprint for receipts. Validate canonical CIDs at the network boundary or otherwise make the content fingerprint authoritative. Add same-tuple/different-body regressions for profile and ordinary records. |
| IR-006 | Important | Behavior | Ordinary identity refresh scheduling still uses a greater-than comparison on Tap event ID. If retained state is retrying event 700 and a rebuilt Tap emits the current identity as event 1, the new delivery gets a receipt but does not move the refresh forward, so authoritative identity convergence waits for the old retry schedule. | Reset-safety rationale in `05-implementation-plan.md`; `appview/internal/ingestion/store.go:504-516`; `appview/internal/ingestion/identity_refresh_trigger_integration_test.go:69-121` | Replace Tap event ID as the identity refresh version fence with a reset-safe database-owned token or Tap-instance epoch plus event ID. Preserve the in-flight-finalization CAS using that token and add a lower-ID-after-reset regression. |
| IR-007 | Suggestion | Code Quality | Migration 56 documents `tap_source_records.owner_generation` as current projection authority, but ingestion still stores the historical effect-attempt generation for non-eligible dispositions, and rejoin preparation relies on equality with that historical value. The column therefore remains conditionally overloaded, making future lifecycle changes easy to get wrong. | `appview/migrations/000056_tap_source_projection_generation.up.sql:10-22`; `appview/internal/ingestion/store.go:205-224,318-338`; `appview/internal/ingestion/service.go:390-405` | Rename/split the field as `projection_generation` and make it consistently represent the lifecycle generation used by projection checks. Treat `effect_operation_id` as provenance and obtain the attempt's historical generation through that relation. This also makes the new composite owner constraint easier to reassess. |
| IR-008 | Important | Tests | Migration 56 has no dedicated data migration or up/down/up regression. The full migration suite proves that the migration files execute in sequence, but does not assert the generation backfill, FK ownership, blocked-job wake, or rollback restoration that this fix depends on. | `appview/migrations/000056_tap_source_projection_generation.{up,down}.sql`; `appview/internal/db/tap_ingestion_migration_test.go:13-43` | Add a migration-56 integration test seeded with an old-generation eligible effect source and blocked job. Assert up migration ownership/backfill/wake behavior, down restoration, and a second up. |
| IR-009 | Suggestion | Code Quality | Generic ingestion and profile ingestion implement nearly identical source comparison, durable-source construction, effect classification, source upsert, and job upsert logic; reconciliation contains a third variant. The reset fix already had to be applied in three places, which is a strong signal that the invariant lacks one owner. | `appview/internal/ingestion/store.go:125-315`; `appview/internal/ingestion/service.go:307-470`; `appview/internal/ingestion/reconciliation.go:215-329` | Extract a pure source-version comparator and one transaction-scoped source/job installation routine. Keep only lifecycle-transition orchestration in the profile service and authoritative-read orchestration in reconciliation. |

## Requirement And Test Traceability

- Requirements implemented: lower Tap event IDs no longer lose against newer
  repository revisions; lower repository revisions remain stale; equal durable
  tuples delivered under a new Tap ID are treated idempotently; eligible
  effect-backed sources are assigned the current lifecycle generation.
- Tests implemented: the profile lifecycle test covers newer-revision/lower-ID,
  same-tuple/new-ID, and older-revision/higher-ID cases; the effect reconciliation
  test covers current-generation promotion after authoritative reconciliation.
- Unplanned behavior: none identified.
- Remaining gaps: revision syntax/order validation, content-sensitive durable
  duplicate identity, reset-safe ordinary identity scheduling, and direct
  migration-56 data/rollback coverage.

## Test Evidence

- Command reviewed and rerun:
  `TEST_DATABASE_URL='postgres://craftsky:dev@127.0.0.1:15747/craftsky_dev?sslmode=disable' TEST_DATABASE_REQUIRED=true go test ./internal/ingestion ./internal/tap ./internal/db -count=1`.
- Passing evidence: all three packages passed against the running local
  PostgreSQL container after rerunning outside the filesystem/network sandbox.
- Failing or skipped tests: the first sandboxed run could not connect to local
  PostgreSQL or bind local test listeners (`operation not permitted`); this was
  an environment restriction, and the authorized rerun passed. No full
  repository gate was rerun for this read-only follow-up review.

## Risk Review

- Risk level: High.
- Risk notes: profile lifecycle transitions are an authorization boundary, and
  malformed winner classification can reactivate or depart an owner incorrectly.
  The current patch fixes the concrete reset incident but does not yet validate
  every value on which its new ordering and duplicate decisions depend.
- Approval notes: retain the successful recovery and migration work, but address
  IR-004 through IR-006 and IR-008 before merge or handoff. IR-007 and IR-009 are
  the recommended simplification direction and can be combined with that pass.

## UI Polish Recommendation

- Recommendation: Not needed.
- Reason: this change has no user-interface surface.
- Suggested polish notes: None.

## Handoff Back To TDD Builder

- Required fixes: IR-004, IR-005, IR-006, and IR-008.
- Suggested next failing test: first add a profile source event with a valid
  current TID followed by an invalid lexically larger revision and prove the
  invalid delivery cannot alter source or lifecycle; then add the same
  revision/CID/action with different record bytes and prove it is not accepted
  as a duplicate.
- Verification to rerun: focused new regressions, migration-56 up/down/up,
  `go test ./internal/ingestion ./internal/tap ./internal/db -count=1`, and the
  full required-database repository gate.

# Previous Review Snapshot: AppView Code-Audit Remediation

## Verdict

Status: Approved with notes  
Reviewer: Codex  
Date: 2026-08-20  
Risk level: Medium

## Summary

The implementation satisfies the approved AV-001 through AV-037 remediation
contracts in the current worktree. The final independent code review found no
remaining blocking correctness or security issue after the last Tap source,
push cancellation, PostgreSQL iterator, upload-admission, OAuth endpoint, and
capability-surface corrections. The repository-wide automated release gate
also passes against disposable PostgreSQL 16 and MinIO state in normal and race
modes, including migration failure/round-trip behavior, static analysis,
source and binary vulnerability policy, exact release-image startup, generated
code drift, and release-container media-memory evidence.

Approval is subject to the documented deployment and destructive-cutover
notes below. The authoritative automated evidence was produced from the
current dirty worktree, not a committed release candidate, so it must be rerun
from the eventual clean commit before release.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Suggestion | Traceability | The complete gate passed, but its tool record identifies baseline commit `2b5d5efa7fee4fa803a2ed64e6a684efb8abad92` and `repository_state=dirty`. This is valid evidence for the reviewed worktree, not for an eventual release commit. | `05-implementation-plan.md` Step 15; `/private/tmp/craftsky-appview-final-gate-20260820-4/tool-identities.txt` | Commit the intended patch, then rerun `scripts/appview-check` from a clean tree before release. |
| IR-002 | Suggestion | Risk | Verified links, provider/device behavior, production network policy, and destructive retained-data/Tap cutover require credentials, infrastructure, devices, or explicit authorization unavailable to the automated implementation pass. | `05-implementation-plan.md` Step 15 and completion checklist | Complete and retain evidence for every external/destructive gate before production release. |
| IR-003 | Suggestion | Risk | The exact built-binary scan has one narrow all-symbol false-positive exception for `GO-2026-5932`. Source scanning is clean and neither binary links an OpenPGP dependency or symbol, but the decision is intentionally time-bounded. | `scripts/appview-check`; artifact `appview-binary-exception.txt` and `cli-binary-exception.txt` | Remove or renew the reviewed exception before its hard expiry on 2026-09-20; never broaden it to another finding or linked symbol. |

No blocking implementation finding was identified.

## Requirement And Test Traceability

| Contract group | Findings | Implemented and reviewed evidence |
|---|---|---|
| Federated HTTP boundary | AV-001, AV-017 | Purpose-scoped clients, destination/DNS/IP validation, redirect and response limits, transient-vs-policy error classification, endpoint-family terminalization, and real-listener zero-connection proofs. |
| Owner lifecycle and private effects | AV-002, AV-003, AV-006, AV-007 | Canonical generation fences, terminal purge catalogue/workers, durable ordinary PDS/object effects, scheduled generation binding, exact-key cleanup, accepted-deletion settlement, and private Instagram suggestions with explicit member acceptance only. |
| Tap ingestion and projection | AV-004, AV-005 | Commit-before-ACK receipts, durable retry/quarantine/replay, exact bounded frame payloads, preserving PostgreSQL `JSON` source records, transaction-scoped projection, effect provenance/reconciliation, same-DID rejoin, and order-independent dependency recovery. |
| OAuth handoff | AV-008, AV-018, AV-019 | Code-only verified HTTPS and loopback handoff, exact loopback CSP, durable server/device-bound exchange and confirm, restart/lost-response recovery, and atomic session promotion. |
| Session integrity | AV-009, AV-010, AV-011, AV-020, AV-021, AV-035 | Cross-instance refresh serialization, versioned persistence, local-first logout, deletion-only credential binding, explicit access classes, dependency-vs-auth response semantics, expiry/revocation/auxiliary workers, and bounded in-memory state. |
| Verification and release policy | AV-012, AV-013, AV-033, AV-036 | Fail-closed migration inspection and Compose dependency behavior, required-database skip detection, normal/race suites, pinned formatting/vet/Staticcheck/govuln policy, module and Lexicon drift checks, release artifacts, and health smoke. |
| HTTP admission and routing | AV-014, AV-015, AV-031, AV-032 | Connection/request/body admission, pre-body rejection, trusted-proxy client identity, deadlines and write-budget geometry, upload concurrency before buffering, explicit access classes, CORS, and canonical JSON errors. |
| Media | AV-016 | Header-first bounded decoding, process-wide pre-body upload admission, durable staged-object cleanup, and exact 512 MiB release-container baseline/per-codec/maximum-admitted-concurrency budgets. |
| Configuration | AV-022, AV-023, AV-024, AV-030 | Canonical public URLs, production-required secrets/endpoints, safe development binding and impersonation rules, bounded durations/counts, and cross-setting geometry validation. |
| Push | AV-025 | Bounded claims/concurrency, exact lease/finalization CAS, lifecycle and current preference/follow rechecks, installation/token rotation fencing, honest ambiguity observations, and standard notification-plus-data delivery with explicitly accepted OS collapse/late/duplicate residuals. |
| PostgreSQL iterator handling | AV-026, AV-027 | Acquisition errors and terminal iterator errors propagate before dependent mutations or commit, including notification preference patching. |
| Index maintenance | AV-028, AV-034 | Required FK-support paths, duplicate-index removal, catalogue/plan assertions, and PostgreSQL 16 up/down/up evidence. |
| Moderation | AV-029 | Atomic output/outbox/receipt persistence, source-vs-target restoration ownership, canonical transition fences, accepted-deletion cleanup, and bounded DID-bearing/DID-free retention. |
| Capability architecture | AV-037 | Narrow route/store/worker bundles, named dependency constructors with reverse cleanup, transactional-only projection, removal of dead forwarding facades, and executable ownership/inventory rules. |

- Requirements implemented: every AV-001 through AV-037 contract is mapped in
  the table above and in `05-implementation-plan.md`.
- Tests implemented: focused red/green regressions, PostgreSQL integration and
  migration tests, race tests, socket/listener tests, release-container probes,
  Flutter unit/widget/platform tests, and executable architecture checks.
- Unplanned behavior: none identified. Contract changes made during review
  (notably preserving source JSON and pre-body upload admission) directly close
  approved boundedness and durability requirements.
- Remaining gaps: no application-code gap. Only the release/cutover actions in
  IR-001 through IR-003 remain.

## Test Evidence

- Authoritative command reviewed:
  `APPVIEW_CHECK_ARTIFACT_DIR=/private/tmp/craftsky-appview-final-gate-20260820-4 ./scripts/appview-check`.
- Passing evidence: Go 1.26.6; Staticcheck 2026.1; govulncheck 1.7.0; module,
  format, vet, Staticcheck, and Lexicon drift checks; all required-PostgreSQL and
  MinIO normal/race packages with real-database skip detection; PostgreSQL 16
  migration up/down-to-zero/up; empty-bundle Compose failure with AppView
  blocked; exact AppView/CLI binaries; release-image health smoke; and source
  scan with zero reachable vulnerabilities.
- Release-container media evidence: 536,870,912-byte cgroup limit;
  134,217,728-byte safety margin; 15,474,688-byte AppView baseline; budget totals
  of 164,511,744 bytes (JPEG), 165,318,656 bytes (PNG), 220,352,512 bytes
  (WebP), and 272,515,072 bytes (maximum admitted concurrent upload).
- Client evidence reviewed: all 1,489 Flutter tests pass and `dart analyze`
  reports no issues; focused release Android/iOS builds and platform-contract
  checks are recorded in the implementation plan and product workflow reviews.
- Failing or skipped tests: none relevant remain. The gate's JSON stream uses
  Go's package-level `skip` action for packages with no test files; its required
  real-database skip detector reported no skipped database case.

## Risk Review

- Risk level: Medium.
- Risk notes: the patch spans authentication, lifecycle, persistence,
  federation, deletion, migrations, transport admission, push, and client
  handoff behavior. Automated coverage is correspondingly broad, but production
  association/provider/network checks and destructive data cutovers cannot be
  simulated by repository tests alone.
- Approval notes: implementation is approved for handoff and clean-commit
  verification. Production release remains contingent on IR-001 through IR-003
  and the explicit external/destructive checklist in `05-implementation-plan.md`.

## UI Polish Recommendation

- Recommendation: Not needed.
- Reason: this is primarily security, durability, lifecycle, routing, and
  architecture remediation. The user-visible OAuth and notification surfaces
  already have exact copy/routing/platform tests; no independent visual polish
  issue was found.
- Suggested polish notes: none.

## Handoff Back To TDD Builder

- Required fixes: none.
- Suggested next failing test: none; do not add implementation work solely to
  satisfy this review.
- Verification to rerun: after committing the intended tree, rerun the complete
  `scripts/appview-check` gate with a fresh artifact directory and the full
  Flutter test/analyzer gate. Then execute the authorized production and
  destructive-cutover checklist and attach its evidence to the release record.
