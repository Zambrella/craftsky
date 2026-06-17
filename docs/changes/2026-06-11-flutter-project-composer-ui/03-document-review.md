# Document Review: Flutter Project Composer UI

## Verdict
Status: Approved with notes
Reviewer: Document-review agent
Date: 2026-06-11
Risk level: Medium

## Summary

The requirements and acceptance-test specification are consistent enough to proceed to coding planning. The selected direction is clear: a separate project composer MVP with a reusable Craftsky FormBuilder field kit, UI-facing option catalogs, a responsive post-type chooser, and no AppView, lexicon, migration, dependency or DTO-enum changes. Must requirements have acceptance criteria and mapped tests, and the recommended first failing test is explicit: `UT-006` for project option catalogs.

No blocking gaps were identified. The notes below should be carried into the coding-plan stage so implementation tests stay focused and traceability remains easy to audit.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Traceability | A few coverage-matrix links are broader than the test metadata they point to. For example, `BR-001` lists `IT-004`, but `IT-004` names `FR-016`, `FR-017` and `RULE-001`; `FR-009` lists `AT-007`, but `AT-007` names `FR-019`, `FR-020`, `FR-023` and `RULE-004`. The acceptance-criteria links still provide practical coverage, so this is not blocking. | `02-acceptance-tests.md` §2 rows for `BR-001`, `FR-009`; §3 `AT-007`; §5 `IT-004` | In coding planning, preserve AC-based coverage and optionally normalise test metadata if the test specification is revised later. |
| DR-002 | Suggestion | Tests | Some acceptance scenarios intentionally bundle several behaviours into one test target, especially optional metadata visibility, count limits and payload serialization. This is acceptable for planning, but could produce brittle widget tests if implemented as one large test. | `02-acceptance-tests.md` §3 `AT-007`, §4 `UT-004`, `UT-007` | Coding planner should split broad scenarios into focused failing tests where useful while preserving the listed test IDs/coverage intent. |
| DR-003 | Important | Requirements | Text-length validation is required but exact limits are not enumerated in the workflow docs. The requirements point to existing composer and lexicon/practical limits, which is enough to proceed but requires an implementation-time lookup before writing assertions. | `01-requirements.md` Q5, `FR-021`, `AC-023`; `02-acceptance-tests.md` `AT-006` | Coding planner should make the first validation tests reference concrete existing constants or lexicon-derived limits rather than inventing new limits. |
| DR-004 | Suggestion | Risk | Accessibility, responsive visual placement and dense-form usability have justified manual checks. These are appropriate because automated widget tests cannot fully validate platform assistive technology behaviour or visual balance. | `02-acceptance-tests.md` §8 `MAN-001` through `MAN-004`; §9 `GAP-001`, `GAP-002` | Keep these checks in the implementation/review checklist before release; do not treat the automated suite as the only acceptance signal for NFR-001 and UX density. |

## Traceability Review

- Planning to requirements: The confirmed Option A direction is reflected throughout the requirements: separate project composer, shared FormBuilder field kit, UI option catalogs, responsive context-menu chooser, all known craft variants, UI-safe validation, and no backend/lexicon/dependency changes. Superseded entry-picker decisions are explicitly captured in Q7/Q8 and carried into `FR-006`, `FR-025`, `AC-001`, and `AC-028`.
- Requirements to acceptance criteria: Every Must `BR`, `FR`, `RULE`, and `NFR` in `01-requirements.md` links to at least one acceptance criterion. `NFR-004` is a Should and is covered by regression/acceptance criteria where it affects risk. No Must requirement lacks AC coverage.
- Acceptance criteria to tests: Every acceptance criterion `AC-001` through `AC-028` appears in the acceptance-criteria coverage index and has at least one automated or justified mixed/manual test path. Minor metadata mismatches are noted in `DR-001`, but they do not remove practical coverage.

## Coverage Review

- Must requirements covered: Yes. Core user flows, reusable field components, option catalogs, payload mapping, responsive chooser, regular-composer regression, craft-specific details, validation, discard, feedback states, localization, no dependency changes and generated-code/static checks are covered.
- Missing or weak coverage: No blocking gaps. Weak spots are text-length specificity (`DR-003`), broad bundled acceptance scenarios (`DR-002`), and representative rather than exhaustive token-catalog drift checks, which is already documented as `GAP-003`.
- Manual-only coverage: Manual checks are justified for accessibility smoke review, responsive visual placement, copy/tone review and dense-form usability (`MAN-001` through `MAN-004`). Automated tests still cover the underlying semantics, presentation type, localization resources and key UI states.

## Risk And Approval Review

- Risk level: Medium.
- Review requirement: Document review recommended and now completed.
- Approval notes: The main risks are breadth of UI surface, regular-composer regression, token catalog drift, density of craft-specific details and image/body/facet reuse. The test plan addresses these with separate option-catalog tests, component tests, project payload/validation tests, chooser tests, provider/image integration tests and regression tests for existing composer/model/context-menu/profile behaviour.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Start with `UT-006` for project option catalogs, especially the `social.craftsky.feed.defs#finished` status token and representative craft/detail option values, because field controls, payload builders and composer tests depend on stable option values.
- Blocking issues: None.

## Notes For Next Stage

- Treat `01-requirements.md`, `02-acceptance-tests.md`, and this review as source of truth.
- Keep the regular composer and reply flows protected; avoid a broad shared-composer refactor unless a focused extraction clearly reduces risk.
- Resolve concrete validation constants before asserting text-length failures.
- Prefer smaller TDD increments even when an acceptance scenario groups several behaviours.
- Preserve the documented non-goals: no AppView/API/lexicon/migration/dependency changes, no DTO enum conversion, no generated token pipeline, no project rendering/editing scope, and no project replies.
