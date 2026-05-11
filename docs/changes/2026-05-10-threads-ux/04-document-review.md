# Document Review: Threads UX

## Verdict
Status: Approved with notes
Reviewer: Document reviewer
Date: 2026-05-10
Risk level: Medium

## Summary
The discovery, requirements, and acceptance-test documents are consistent and ready for TDD implementation. The confirmed product direction is traceable from discovery through requirements and tests: keep the work Flutter-focused, anchor the selected thread post just below the app bar, keep ancestors above in scrollback, disable self-navigation only for the selected post, and add a compact three-line reply-target preview in the composer. All Must requirements have acceptance criteria and test coverage. Notes below are non-blocking implementation/test-design cautions.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Tests | The phrase "just below the app bar" is intentionally tolerance-based, but the implementation agent should choose a stable widget-test assertion strategy rather than exact pixels. | `02-requirements.md` ASM-002; `03-acceptance-tests.md` GAP-001, AT-001, AT-002 | Use relative positioning/tolerances in widget tests and keep MAN-001 for visual confirmation. |
| DR-002 | Suggestion | Tests | AT-005 combines ancestor, reply, and continuation navigation in one scenario. This is acceptable, but implementation may be easier if split into separate widget tests or subcases. | `03-acceptance-tests.md` AT-005, REG-002 | TDD builder may split AT-005 into smaller tests while preserving the same test ID coverage. |
| DR-003 | Suggestion | Risk | NFR-002 theme consistency is manual-only, which is reasonable for visual polish but should be explicitly checked before considering implementation complete. | `02-requirements.md` NFR-002, AC-010; `03-acceptance-tests.md` MAN-002, GAP-002 | Perform MAN-002 or equivalent visual review after the UI is implemented. |

## Traceability Review
- Discovery to requirements: Good. The selected-at-top clarification in discovery Q1 is reflected in BR-001, BR-002, FR-001, FR-002, FR-003, and ASM-002. The compact-preview decision in discovery Q2 is reflected in BR-004, FR-007, FR-008, and AC-006/AC-007. The Flutter-only/API-preserving direction in discovery Q3 is reflected in NG-001, NG-002, RULE-002, and AC-011.
- Requirements to acceptance criteria: Good. Every Must BR, FR, and RULE in `02-requirements.md` links to one or more acceptance criteria. Should-level NFRs have acceptance criteria and manual/regression verification paths.
- Acceptance criteria to tests: Good. AC-001 through AC-011 are covered by AT, UT, REG, IT, or MAN entries in `03-acceptance-tests.md`. Must functional behavior has automated widget-test targets.

## Coverage Review
- Must requirements covered: Yes. BR-001 through BR-004, FR-001 through FR-009, RULE-001, and RULE-002 all have linked acceptance criteria and test IDs.
- Missing or weak coverage: None blocking. The only weaker areas are visual/tolerance concerns for NFR-001/NFR-002, already documented as GAP-001/GAP-002 with manual checks.
- Manual-only coverage: NFR-002/AC-010 is manual-only via MAN-002. NFR-001/AC-009 has partial automated regression coverage plus MAN-001.

## Risk And Approval Review
- Risk level: Medium, consistent across discovery, requirements, and tests.
- Review requirement: Review recommended, not required. The user approved requirements and test plan; Plannotator folder review was attempted/opened as part of the workflow.
- Approval notes: Approved to proceed with TDD implementation. The implementation should stay within Flutter UI/tests and docs unless a documented reason emerges to reopen requirements.

## Implementation Readiness
- Ready for TDD implementation: Yes.
- Recommended first step: Start with the first failing widget test for `AT-002` in `app/test/feed/pages/post_thread_page_test.dart`: a selected reply opens anchored below the app bar while ancestors remain above in scrollback and replies remain below.
- Blocking issues: None identified.

## Notes For Next Stage
- Keep source changes scoped to the Flutter thread page/composer and their tests unless a requirement is reopened.
- Prefer small failing tests in the order listed in `03-acceptance-tests.md` §11.
- Preserve existing reply reference semantics; the composer preview is display-only.
- Do not alter AppView routes, thread response shape, lexicons, migrations, or PDS write behavior for this change.
