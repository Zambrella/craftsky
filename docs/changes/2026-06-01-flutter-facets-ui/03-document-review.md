# Document Review: Flutter Facets UI

## Verdict
Status: Approved with notes
Reviewer: gpt-5.5 document reviewer
Date: 2026-06-01
Risk level: High

## Summary

The requirements and acceptance-test documents are consistent and sufficiently traceable for coding planning to begin. The confirmed product direction from `01-requirements.md` is preserved in the test strategy: Flutter-only rich-text facets, mock-backed mention/hashtag autocomplete, post facet payload propagation, intentional profile `descriptionFacets` send ahead of AppView support, and safeguards against direct PDS/external identity calls.

No blocking documentation gaps were found. The feature remains high risk because profile `descriptionFacets` are knowingly incompatible with the current AppView profile endpoint and because implementation must use any atproto.dart ecosystem helper without accidentally performing external handle resolution. Those risks are explicitly captured in requirements, acceptance tests, and test gaps, so they do not block moving to coding planning.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Risk | The profile `descriptionFacets` live-save incompatibility is intentionally accepted and well documented, but it should remain visible as a first-class implementation-review item because it can break live profile saves until AppView/API support lands. | `01-requirements.md` Q2/Q4, `RULE-004`, `RISK-001`, `Open Questions`; `02-acceptance-tests.md` `GAP-001`, `IT-006`, `MAN-003` | Coding plan should isolate this behavior in API/repository tests and explicitly call out the follow-up backend slice. |
| DR-002 | Suggestion | Risk | The tests correctly identify that they cannot fully prove a third-party helper will never perform external identity resolution unless implementation avoids those API paths. | `01-requirements.md` `RULE-001`, `RISK-003`, `ASM-001`; `02-acceptance-tests.md` `GAP-002`, `UT-017`, `IT-007` | Coding plan should require dependency/API inspection and prefer local byte-index/entity APIs plus injected Craftsky resolver seams. |
| DR-003 | Suggestion | Tests | Accessibility coverage for autocomplete is appropriately partial/manual because full assistive-technology validation is outside widget-test scope. | `01-requirements.md` `NFR-003`; `02-acceptance-tests.md` `MAN-001`, `GAP-004` | Coding plan should keep visible labels/semantics testable and leave full accessibility certification as a follow-up/manual check. |

## Traceability Review

- Planning to requirements: The initial request and clarified decisions are carried through: pass facets now (`Q1`), intentionally send profile `descriptionFacets` despite current AppView rejection (`Q2`, `Q4`), use a shared Flutter rich-text/facet module with mock-backed autocomplete (`Q3`), resolve manually typed known handles (`Q5`), preserve hashtag casing and parsing rules (`Q9`, `Q13`, `Q14`), handle URL normalization/trimming (`Q10`-`Q12`), and tolerate malformed incoming facets (`Q21`).
- Requirements to acceptance criteria: Every Must `BR`, `FR`, `NFR`, and `RULE` in `01-requirements.md` links to at least one acceptance criterion. The only Should requirement, `NFR-003`, is covered with partial automated/manual accessibility checks.
- Acceptance criteria to tests: `02-acceptance-tests.md` maps every acceptance criterion to one or more automated acceptance, unit, integration, or regression test IDs, with explicit manual checks and gaps where automation is not complete.

## Coverage Review

- Must requirements covered: `BR-001`, `BR-002`, `FR-001` through `FR-013`, `NFR-001`, `NFR-002`, `NFR-004`, `NFR-005`, and `RULE-001` through `RULE-009` are covered in the requirement coverage matrix and detailed test cases.
- Missing or weak coverage: No blocking missing coverage found. Non-blocking weak spots are already documented as `GAP-001` for live AppView profile support and `GAP-002` for implementation-dependent third-party helper behavior.
- Manual-only coverage: Manual checks are limited and justified: autocomplete accessibility (`MAN-001`), visual theme-color review (`MAN-002`), live profile-save compatibility-risk smoke check (`MAN-003`), and real-device link launch smoke check (`MAN-004`).

## Risk And Approval Review

- Risk level: High.
- Review requirement: Satisfied for the documentation stage. High-risk areas are explicitly called out in both workflow documents.
- Approval notes: Coding planning may proceed, but the plan should keep the high-risk work isolated and reviewable: byte-safe facet generation first, renderer normalization second, autocomplete/provider seams third, then post/profile payload propagation and widget flows.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Start with the recommended first failing test from `02-acceptance-tests.md`: `UT-002` in `app/test/shared/rich_text/facet_generator_test.dart` for UTF-8 byte offsets with emoji/multibyte text before mention/link/hashtag facets.
- Blocking issues: None for this Flutter-only coding plan. Live profile usability remains blocked on a future AppView/API slice, but that is an accepted out-of-scope risk rather than a blocker for Flutter implementation planning.

## Notes For Next Stage

- Treat `01-requirements.md`, `02-acceptance-tests.md`, and this review as source of truth.
- Preserve the Flutter-only boundary: no AppView implementation, migrations, lexicon changes, PDS calls, or external identity lookup calls in this slice.
- Use the acceptance-test suggested order unless coding-planner finds a stronger dependency ordering.
- Explicitly include implementation-review checks for `descriptionFacets` compatibility handling and third-party helper usage.
