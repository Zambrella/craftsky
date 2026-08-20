# Implementation Review: AppView Code-Audit Remediation

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
| Push | AV-025 | Bounded claims/concurrency, exact lease/finalization CAS, lifecycle and current preference/follow rechecks, installation/token rotation fencing, honest ambiguity observations, unique delivery semantics, and durable client stage deduplication. |
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
