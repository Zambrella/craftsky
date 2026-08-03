# Document Review: Scheduled Posts

## Verdict

Status: Approved

Reviewer: Codex

Date: 2026-07-31

Risk level: High

## Summary

The revised requirements and acceptance-test specification are ready for coding-plan work. The agreed product scope is preserved, all 39 formal requirements map to acceptance criteria and defined tests, all 33 acceptance criteria are covered, manual checks are limited to genuine deployment/device boundaries, and the proposed first failing test is practical.

The previous review's four findings are resolved without renumbering existing IDs. The migration now has an explicit schema contract test; queue observability and deployed alerts have a formal Must requirement, acceptance criterion, automated test, and manual production check; the partial-PDS-media crash boundary is explicit; and the workflow status accurately records authorization through re-review while preserving the implementation approval gate.

## Findings

None identified.

### Prior Findings Resolution

| ID | Resolution | Evidence |
|---|---|---|
| DR-001 | Resolved | IT-026 applies and inspects the scheduled-post migration, including private tables, lifecycle/reference constraints, owner/idempotency uniqueness, required query indexes, and the absence of public post/lexicon changes. |
| DR-002 | Resolved | NFR-006 and AC-033 formalize safe queue/latency/outcome signals and deployed alerts; IT-027 automates signal semantics/redaction and MAN-006 verifies production alert configuration. |
| DR-003 | Resolved | AT-011 explicitly includes a crash after one or more PDS media uploads; IT-009 and TD-007 define deterministic recovery, no premature record, safe content-addressed re-upload/reuse, stable body/identity, and no Craftsky PDS blob deletion. |
| DR-004 | Resolved | Requirements status is `Revised; pending re-review`, records the user's authorization through test design/review, and retains explicit approval before implementation. This approved re-review supersedes that pending status for coding-plan readiness. |

## Traceability Review

- Planning to requirements: Pass. The initial request and discovery context are embedded in `01-requirements.md`; the recommended AppView/Postgres/in-process-worker/S3-compatible-storage direction is reflected in the requirements, risks, data impact, and non-goals. There is no separate `00-initial-prompt.md`, but no planning decision is lost.
- Requirements to acceptance criteria: Pass. All 39 requirements have acceptance-criteria links, including all 37 Must requirements. No duplicate or undefined requirement/acceptance-criteria IDs were found.
- Acceptance criteria to tests: Pass for AC-001 through AC-033. All are referenced by defined acceptance, unit, integration, regression, or manual tests; all 77 matrix test IDs are defined and each test case links back to requirement and acceptance-criteria IDs.
- Prior traceability weakness: Resolved. Operational observability now has the NFR-006 → AC-033 → IT-027/MAN-006 chain.

## Coverage Review

- Must requirements covered: 37 of 37 formal Must requirements have linked tests.
- Should requirements covered: FR-019 and NFR-005 are covered by automated and/or manual targets.
- Acceptance criteria covered: 33 of 33 current acceptance criteria have linked tests.
- Defined test inventory: 14 acceptance scenarios, 20 unit cases, 27 integration cases, 10 regressions, and 6 manual checks, plus 11 test-data definitions and 4 explicit gaps.
- Missing or weak coverage: None identified for coding planning.
- Manual-only coverage: Production managed-storage and alert controls remain correctly isolated in MAN-005/MAN-006 and GAP-001/GAP-002. Real deployment latency, real PDS cleanup observation, and a live process-kill drill remain justified release checks because deterministic automated substitutes are also specified.

## Risk And Approval Review

- Risk level: High, unchanged from requirements and test design.
- Review requirement: This formal document review is approved for coding-plan work. Explicit user approval remains required before implementation begins.
- Approval notes: The user approved the product decisions and explicitly advanced the workflow through test design, revision, and re-review. That authorizes coding planning; it does not remove the separate implementation approval gate.
- Risk handling that is already strong: transactional three-item capacity, last-write-wins member edits with internal worker fencing, no-early publication, fixed retry window, active-session rules, stable PDS identity/body, owner-scoped private media, bounded retention, privacy canaries, no-notification scope, and honest external release gaps.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Write `04-coding-plan.md`, preserving `UT-002` as the first failing implementation test and the dependency-ordered sequence in `02-acceptance-tests.md` §11.
- Blocking issues: None for coding planning.
- Release gates: GAP-001 and the applicable live checks in GAP-002 through GAP-004 remain explicit pre-production work; they do not block coding-plan design.

## Notes For Next Stage

- Preserve `UT-002` as the recommended first failing implementation test; it remains a small, deterministic starting point.
- Include IT-026 early in the database phase so the migration shape is fixed before store behavior depends on it.
- Include IT-027 with worker observability and retain MAN-006 as a deployed alert release gate.
- Keep real Postgres coverage based on `internal/testdb.WithSchema`, matching existing migration/store conventions.
- Keep time and crash tests deterministic through injected clocks and barriers; do not replace them with wall-clock waits.
- Keep MinIO as the local S3-compatible contract target and managed-provider verification as a production release gate.
- Do not add a lexicon change, notification behavior, history screen, custom application-level encryption, client-side scheduler, or separate worker service during coding planning or implementation.
