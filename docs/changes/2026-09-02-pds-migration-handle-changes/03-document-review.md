# Document Review: PDS Migration And Handle Change Resilience

## Verdict
Status: Approved
Reviewer: OpenCode
Date: 2026-09-03
Risk level: High

## Summary

The requirements and acceptance-test specification consistently describe the approved lean hybrid: DID remains canonical, Tap remains the primary ingestion path, uncached authority verification protects each authenticated PDS effect and ordinary OAuth callback, same-DID reauthorization activates a new grant, and a narrow verified repository sweep repairs missed state. Scope boundaries, security constraints, operational requirements, and the lack of production compatibility obligations are preserved.

Every Must requirement links to acceptance criteria and at least one automated test. Every acceptance criterion has a concrete verification path. No blocking product question or missing Must coverage remains. The earlier review findings have been incorporated into `01-requirements.md` and `02-acceptance-tests.md` without renumbering stable IDs.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Important | Tests | Resolved: IT-007 and AC-018 now enumerate onboarding, ordinary sign-in, and reauthentication as separate durable Tap-tracking entry points, including remote failure, restart, recovery, and duplicate-trigger behavior. | `01-requirements.md` FR-010, AC-018; `02-acceptance-tests.md` IT-007 | None. Preserve the explicit table-driven trigger matrix in the coding plan. |
| DR-002 | Important | Tests | Resolved: AC-048, AT-008, and UT-015 now define exactly 30 minutes as the final eligible attempt. Success may publish; failure transitions to `needs_attention`; later authorization cannot automatically publish. | `01-requirements.md` FR-025, AC-048; `02-acceptance-tests.md` AT-008, UT-015, TD-008 | None. Preserve this boundary in implementation tests. |
| DR-003 | Suggestion | Tests | Resolved: UT-005 now targets a pure identity-event policy unit test, while IT-014 retains transactional persistence, restart, duplicate, and out-of-order coverage. | `02-acceptance-tests.md` UT-005, IT-014 | None. |
| DR-004 | Suggestion | Risk | Resolved for current known surfaces: UT-018 now inventories each known identity surface and requires paired focused provider/widget coverage. GAP-005 remains as an honest residual risk for future dynamically constructed navigation. | `01-requirements.md` RISK-006; `02-acceptance-tests.md` UT-018, GAP-005 | None before coding planning. Preserve inventory review as new identity surfaces are added. |

## Traceability Review

- Planning to requirements: Complete. Option A's authority checks, same-DID reauthorization, Tap-first ingestion, verified repair sweep, incremental post-verification projection, existing scheduler lifecycle, `handle.invalid`, DID-first navigation, and full-DID deletion confirmation are represented. Options B and C remain excluded by the non-goals and explicit simplifications.
- Requirements to acceptance criteria: Complete. Every Must BR, FR, NFR, and RULE has at least one linked acceptance criterion. Criteria are externally verifiable at the HTTP, persistence, protocol-adapter, lifecycle, or Flutter UI boundary.
- Acceptance criteria to tests: Complete. AC-001 through AC-055 as defined in the requirements document all appear in the coverage matrix and concrete automated cases. The intentionally unused numeric IDs in the AC sequence are not missing criteria.

## Coverage Review

- Must requirements covered: All 42 Must requirements are covered: BR-001 through BR-003; FR-001 through FR-021; FR-023, FR-025, FR-027 through FR-031; NFR-001, NFR-002, NFR-003, NFR-005; and RULE-001 through RULE-007.
- Missing or weak coverage: None identified. DR-001 through DR-004 are resolved. GAP-001 through GAP-005 remain documented limitations or deferred boundary values rather than missing Must behavior.
- Manual-only coverage: None of the product correctness or credential-safety requirements relies only on manual testing. MAN-001 supplements automated Flutter presentation checks. MAN-002 covers deployment-level dashboard and alert delivery while UT-016 and IT-017 automate signal-content and secret-redaction behavior.

## Risk And Approval Review

- Risk level: High, unchanged. OAuth authority confusion, stale-token forwarding, exact-parent fencing, incomplete-snapshot deletes, DID lifecycle preservation, and broad Flutter identity changes remain high-impact failure modes.
- Review requirement: Satisfied for progression to coding planning by this cross-document review and resolution pass. Implementation must still follow the test-first sequence and retain explicit review before merge or handoff.
- Approval notes: Approval is limited to coding-plan work. Preserve the resolved trigger matrix and scheduled cutoff, the HTTP-fake credential-capture tests, and verified-complete-snapshot proof ahead of inferred deletes. No implementation approval is implied until the coding plan and later implementation review are complete.

## Coding Plan Readiness

- Ready for coding planning: Yes
- Recommended first step: Design the uncached authority-verification interface at the shared OAuth-session coordinator and ordinary callback boundary around UT-001 and UT-002, including typed outcomes for current, proven stale, and retryable/unverified authority. The first red test should prove a verified PDS/authorization-server mismatch prevents the protected effect before any stale credential-bearing request.
- Blocking issues: None.

## Notes For Next Stage

- Keep one shared authority-verification path for foreground and background effects; do not distribute migration checks across handlers.
- Make exact-parent fencing transactional and version/lifecycle-fenced so concurrent checks cannot invalidate a corrected or independent parent.
- Model old-provider cleanup as asynchronous and bounded against the original issuer only. Reauthorization must not wait for it.
- Separate complete repository acquisition and cryptographic/root verification from projection comparison. No indexer delete may run before the complete snapshot is trusted.
- Derive repair collection coverage from the same registry used to register indexers; do not introduce a second NSID list.
- Reuse the existing durable repository-job system where its lease and retry semantics suffice. Define CAR size, batch, lease, retry, and alert limits in the coding plan and add corresponding boundary tests.
- Expand IT-007 across onboarding, ordinary sign-in, and reauthentication as required by DR-001.
- Define exactly 30 minutes as the final eligible scheduled publication attempt, as required by DR-002.
- Keep Flutter route, provider, cache, ownership, mention, recent-search, and deletion identities DID-first. Handles remain presentation or explicit external alias input only.
- No lexicon change is expected. If implementation discovers one, stop and use the lexicon/ADR workflow before proceeding.
- Preserve the test commands and release gates listed in `02-acceptance-tests.md`; use `just appview-check` for release-equivalent AppView evidence and focused Flutter suites before `just app-test`.
