# Document Review: Public Profile Customisation

## Verdict

Status: Approved with notes\
Reviewer: Codex\
Date: 2026-08-09\
Risk level: Medium

## Summary

The requirements and acceptance-test specification are ready for coding-plan work. They preserve the confirmed AppView-only ownership boundary, nested additive public identity contract, authenticated full-replacement mutation, set-based response hydration, shared avatar seam, exact border geometry, fixed local texture catalogue, Settings editing lifecycle, compact/full theme boundaries, backwards-compatible defaults, moderation policy, accessibility expectations, and active-account fencing.

All four findings from the first review are resolved:

- The non-goal now agrees with the confirmed seven-colour and `none` plus six-texture catalogues. The seventh theme-Ink bundle was approved as a later additive catalogue extension on 2026-08-10 and does not change the reviewed architecture.
- The three unresolved exact inputs use one consistent gate: they do not block coding planning, but must close before their affected implementation tests are written.
- Public response acceptance allows additive future customisation fields while the current mutation request remains strict.
- Every approved texture is now exercised in both compact and full profile presentations at acceptance level.

Traceability remains complete. All 28 Must requirements link to acceptance criteria and planned tests, all AC-001 through AC-020 have planned coverage, and every referenced AT, UT, IT, REG, MAN, and GAP ID resolves to a definition.

No `00-initial-prompt.md` exists in this workflow folder. This is acceptable because `01-requirements.md` contains the initial request, codebase discovery, candidate approaches, recommended direction, and the complete grilling decision record.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-005 | Suggestion | Risk / Coding plan | Three exact inputs were deliberately gated: the non-default theme bundles, per-colour texture tint/opacity, and save-failure copy. GAP-001 through GAP-003 closed on 2026-08-10 before their dependent exact assertions. | `01-requirements.md` §§3 Q9/Q11–Q13, 19 RISK-006, 21, 23; `02-acceptance-tests.md` §§1, 9 GAP-001–GAP-003, 11, 12 | Preserve the recorded constants and exact copy in implementation tests. |

## Traceability Review

- Planning to requirements: The initial request and every confirmed grilling decision are carried into goals, non-goals, requirements, acceptance criteria, edge cases, risks, and handoff notes. The recommended dedicated AppView resource remains consistent throughout.
- Requirements to acceptance criteria: Every Must BR, FR, NFR, and RULE links to at least one AC. All 28 Must requirements are represented across AC-001 through AC-020. NFR-005 is Should priority and also has test coverage.
- Acceptance criteria to tests: Every AC appears in `02-acceptance-tests.md`; every referenced test/gap ID has a definition. Public-response additivity and the complete six-texture × two-presentation behavior are now explicit.

## Coverage Review

- Must requirements covered: 28 of 28 have planned acceptance, unit, integration, regression, or explicit gated-gap coverage.
- Missing or weak coverage: None identified. GAP-001 through GAP-003 are closed.
- Manual-only coverage: None. MAN-001 through MAN-003 supplement automated semantics, focus, layout, contrast, bounds, local-asset, network-boundary, and rendering assertions with platform speech and perceptual review.
- Test practicality: Proposed targets match the repository's real-Postgres migration/query-plan patterns and focused Flutter model, provider, widget, router, and account-switch suites. The suggested red-green order is coherent.

## Risk And Approval Review

- Risk level: Medium. The feature adds durable AppView state and a mutation, enriches every public identity response shape, and changes all avatar-bearing Flutter surfaces plus profile-local theming.
- Review requirement: This document review satisfies the pre-coding-plan review recommendation. Implementation review remains important and should inspect response-shape completeness, query behavior, no-PDS/no-network guarantees, account fencing, avatar geometry, scoped theming, asset provenance, accessibility, and actual test evidence.
- Approval notes: Architecture and product behavior are approved for coding planning. DR-005 is a non-blocking sequencing note. The gated constants require approval only at their stated pre-implementation points.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Inspect the current persistence, route, response-builder/hydration, generated model, account-state, shared-avatar, and profile presentation seams closely enough to produce a file-specific implementation plan. Preserve the test order beginning with catalogue/default/validation policy.
- Blocking issues: None for coding planning.

## Notes For Next Stage

- Keep customisation in a dedicated AppView-owned DID-scoped resource and authenticated/device-bound `PUT /v1/profiles/me/customisation`; do not merge it into the PDS-backed profile mutation.
- Choose a persistence representation that supports one atomic complete record, membership cascade, per-field fallback, indexed set-based hydration, and future fields without weakening current validation.
- Inventory every full and embedded identity builder/query before planning response changes so hydration is centralized and bounded rather than added as per-surface lookups.
- Preserve additive public response decoding while keeping the current mutation request strict to exactly the three current keys.
- Centralize Flutter catalogue/default mapping, active-account state, avatar border rendering, texture rendering, and profile theme scoping; remove obsolete experimental frame/background/route-extra paths deliberately.
- Preserve the recorded palette, texture tint/opacity, and failure copy in dependent tests.
- Carry the acceptance specification's TDD order and regression matrix into the coding plan, including generated-output commands and full-suite verification.
