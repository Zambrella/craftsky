# Document Review: Pinned Profile Posts

## Verdict

Status: Approved
Reviewer: Codex
Date: 2026-08-05
Risk level: Medium

## Summary

The requirements and acceptance-test specification are consistent with the confirmed AppView-only direction and are ready for coding-plan work. They preserve the two independent owner slots, universal current-member access, profile-only promotion and annotation, existing visibility policy, server-confirmed Flutter behavior, target-specific unpinning, and pin-bound pagination. They do not introduce payment, free/paid-user, plan, tier, entitlement, or access-gating work.

All 25 Must requirements link to acceptance criteria and tests. The one Should requirement, NFR-005, is also covered. All 20 acceptance criteria have concrete acceptance, unit, integration, regression, or justified manual verification. DR-001–DR-003 were resolved in `01-requirements.md` and `02-acceptance-tests.md`: the feature-specific no-body/authoritative-response API contract is explicit, absent `pinnedPostUri` has one exact server encoding, and pending mutations are scoped by active account and slot. No blocking question, missing Must coverage, unresolved finding, or contradiction remains.

The optional `00-initial-prompt.md` is absent, but the initial request, discovery context, confirmed decisions, and scope corrections are preserved in Sections 1–5 of `01-requirements.md`; no information required for this review is missing.

## Findings

None identified.

## Resolved Findings

| ID | Original Severity | Resolution | Updated References |
|---|---|---|---|
| DR-001 | Important | The pin routes are now explicitly path-complete, no-body operations. Requirements and tests identify their authoritative `200 OK` mutation bodies as a deliberate feature-specific exception to the API architecture's generic PUT-body and DELETE-204 examples, with route-policy and Flutter contract protection. | `01-requirements.md` Q11, FR-013, AC-017, RISK-008, UI/API impact; `02-acceptance-tests.md` strategy, IT-005, IT-009, IT-015, handoff |
| DR-002 | Suggestion | The server now has one exact absent-state contract: include `pinnedPostUri` only for a visible promoted pin on page one; omit it when no pin is promoted and on every later page. Flutter tolerates absent or explicit null on decode and omits the key when encoding no pin. | `01-requirements.md` Q11, FR-006, FR-013, AC-009, AC-012, AC-017, RISK-009; `02-acceptance-tests.md` UT-001, IT-006, IT-007, REG-001, TD-006 |
| DR-003 | Suggestion | Pending mutation state is now keyed by active account and inferred slot. All same-slot Pin/Unpin actions are disabled while pending; the independent slot remains usable; active-account fencing remains mandatory. | `01-requirements.md` Q8, FR-008, AC-003, AC-005, AC-018, RISK-010; `02-acceptance-tests.md` AT-003, AT-004, AT-008, UT-003, UT-005, IT-016, MAN-001 |

## Traceability Review

- Planning to requirements: Complete. The recommended dedicated AppView relation is carried into the data, API, privacy, lifecycle, and non-goal sections. Every grilling decision and DR-001–DR-003 resolution is recorded in requirements or acceptance criteria. Former assumptions are explicitly resolved and there are no open questions.
- Requirements to acceptance criteria: Complete. All 25 Must requirements reference at least one acceptance criterion. NFR-005, the single Should requirement, references AC-019. Each acceptance criterion is externally verifiable and uses stable AC IDs.
- Acceptance criteria to tests: Complete. All AC-001–AC-020 appear in test cases. Twelve Gherkin scenarios cover user-visible workflows; nine unit designs, sixteen integration designs, eight regressions, and two justified manual checks cover lower-level and cross-cutting behavior. Every AT, UT, IT, REG, and MAN case references both requirement and acceptance-criteria IDs.

## Coverage Review

- Must requirements covered: 25 of 25.
- Should requirements covered: 1 of 1.
- Acceptance criteria covered: 20 of 20.
- Missing or weak coverage: No missing behavioral coverage. GAP-001 correctly records the absence of a full Flutter-to-AppView device E2E harness and compensates with real-Postgres Go tests plus Flutter API/provider/widget tests. GAP-003 correctly requires deterministic transaction barriers rather than timing-based concurrency tests. GAP-004 adds query-count and side-effect sentinels rather than relying only on response assertions.
- Manual-only coverage: No requirement is entirely manual-only. MAN-001 and MAN-002 supplement automated semantics/layout tests for physical VoiceOver/TalkBack behavior and final visual parity at real device rendering/text scales.
- Automation targets: Practical and consistent with the repository. Proposed Go tests sit beside existing migration, saved-post, profile-list, route, query-plan, lifecycle, and observability suites. Proposed Flutter tests extend existing post API, page model, account fencing, profile provider, profile tab, and shared `PostCard` suites.

## Risk And Approval Review

- Risk level: Medium. The material risks are transaction ordering, profile cursor correctness, viewer-policy ordering, lifecycle cleanup after indexed structural changes, shared-card presentation leakage, active-account races, and future drift from the explicit feature-specific API/metadata contracts.
- Review requirement: Document review was recommended and is now complete. Explicit approval beyond the normal coding-plan review is not required before implementation planning.
- Approval notes: The user approved all product and API decisions during grilling and requested that DR-001–DR-003 be incorporated into the documents. IT-002–IT-003 address atomicity and last-committed-wins behavior; IT-005, IT-009, and IT-015 protect the deliberate route exception; IT-006–IT-008 address exact metadata omission plus limit/cursor/policy risks; IT-010–IT-014 address lifecycle, isolation, privacy, query plans, and observability; UT-003–UT-007 and IT-016 address non-optimistic per-account-and-slot UI, accessibility, and account fencing.
- Remaining gates: MAN-001 and MAN-002 remain UI verification gates after implementation. They are not blockers for coding planning.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Design the private profile-pin schema and start implementation with the proposed failing `TestProfilePinsMigration`, then follow the store-first test order in `02-acceptance-tests.md`.
- Blocking issues: None.
- Settled contract for the planner: Preserve bodyless target-specific PUT/DELETE routes and authoritative `200 OK` bodies; include `pinnedPostUri` only for a visible promoted first-page pin and omit it otherwise/later; keep Flutter absent/null tolerant on decode and omission-based on encode; scope pending mutations by account and slot while leaving the other slot usable; keep public `PostResponse`, PDS/lexicon/Tap data, and non-profile ranking free of pin state.

## Notes For Next Stage

- Read the AppView API architecture spec before designing the new routes, and carry forward DR-001's resolved feature-specific no-body/authoritative-response exception rather than silently changing the agreed contract.
- Re-check the highest migration number before assigning the profile-pin migration.
- Use the private saved-post migration/store/route suites as persistence and authorization precedents, while keeping pin target validation stricter because only owned, currently returnable top-level posts qualify for new pins.
- Keep pin-aware profile traversal in set-based queries. The cursor must encode enough starting pin state to reject a later page after a pin change without exposing implementation details to Flutter.
- Keep `pinnedPostUri` in the profile page envelope and out of the canonical `Post`/`PostResponse` model; omit it for no visible pin and every later page.
- Reuse the existing active-account operation guard for mutation completion and account-scoped cache invalidation; key pending actions by account and slot.
- Preserve the repost attribution row's layout and styling, but make `Pinned post` informational and non-interactive.
- Do not introduce payment, free/paid-user, plan, tier, entitlement, or access-gating models, dependencies, fixtures, or tests.
