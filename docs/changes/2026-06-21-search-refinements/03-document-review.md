# Document Review: Search Refinements Before UI Slice

## Verdict
Status: Approved with notes
Reviewer: OpenAI gpt-5.5 document reviewer
Date: 2026-06-21
Risk level: Medium

## Summary
The requirements and acceptance-test documents are consistent enough for coding-plan work. The recommended Option A direction is carried through the requirements, the Must requirements all have acceptance criteria, and the acceptance-test matrix traces those requirements into concrete unit, integration, acceptance, regression, and manual checks. The test specification also preserves the key scope boundaries: no rendered UI, no PDS recent-search persistence, AppView-authenticated reads, facet compatibility, disjoint submitted Posts/Projects tabs, exact hashtag feeds, and project browsing under the Projects API/data layer.

No blocking issues were found. The notes below should be handled by the coding planner or by small document clarifications if the team wants stricter implementation guidance before coding.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Requirements / Tests | Empty unified-suggestion query behavior is slightly loose in the requirements. `FR-001` says the endpoint accepts a non-empty `q`, `EC-001` allows either an empty suggestions response or validation behavior, while `UT-001` expects invalid input to map to standard validation errors. The test design is specific enough to proceed, but the requirement edge case leaves room for a different implementation. | `01-requirements.md` FR-001, EC-001; `02-acceptance-tests.md` UT-001, AT-009 | Prefer the stricter validation behavior already described by `UT-001` and `/v1/` validation tests, or revise both documents if an empty-response contract is intended. |
| DR-002 | Suggestion | API Contract | The `/v1/search/projects` rich-filter removal language is mostly clear but has one soft phrase: `AC-019` allows unsupported browse filters to be “rejected or otherwise documented as removed.” Other criteria and tests point toward standard validation rejection. | `01-requirements.md` FR-010, AC-014, AC-019, EC-015; `02-acceptance-tests.md` AT-006, UT-008, IT-009 | Coding plan should choose the validation-error path unless requirements are revised. Avoid silent ignore behavior for unsupported browse filters on `/v1/search/projects`. |
| DR-003 | Suggestion | Requirements | The requirements artifact still lists `Status: Draft` and `Reviewer: Unassigned` in its own review-status section. This review artifact supplies the formal stage verdict, so this is not blocking. | `01-requirements.md` §22; this document §Verdict | Treat `03-document-review.md` as the review decision for moving forward. If desired, update prior document status in a separate requirements-maintenance step, not as part of implementation. |
| DR-004 | Suggestion | Risk / Tests | Relevance ordering is intentionally requirement-level, but the exact scoring implementation and final tie-breakers are left to implementation design. Tests require deterministic relevance-first ordering, so the coding plan must pin this down before store tests are written. | `01-requirements.md` FR-006, RULE-008, RISK-008; `02-acceptance-tests.md` AT-004, UT-006, IT-005, AC-018 | In the coding plan, define the scoring helper or PostgreSQL ranking approach and stable tie-breakers before implementing `UT-006` / `IT-005`. |

## Resolution Notes
- DR-001 addressed in `01-requirements.md` by making empty or whitespace-only unified suggestion `q` a standard validation error, and in `02-acceptance-tests.md` by covering empty required queries in API validation.
- DR-002 addressed in `01-requirements.md` and `02-acceptance-tests.md` by requiring standard validation errors for rich project browse filters sent to `/v1/search/projects`.
- DR-003 addressed in `01-requirements.md` §22 by setting the review status to `Approved with notes` and assigning this reviewer.
- DR-004 addressed in `01-requirements.md` and `02-acceptance-tests.md` by pinning submitted post/project text search to relevance score descending, then `createdAt` descending, then URI descending.

## Traceability Review
- Planning to requirements: The requirements preserve the confirmed Option A direction: shared suggestion core, unified search typeahead contract, separate paginated result APIs, `/v1/projects` for project browse/filtering, disjoint submitted Posts/Projects tabs, exact hashtag combined feeds, and recent/saved searches as one private AppView-backed surface.
- Requirements to acceptance criteria: All Must business, functional, non-functional, and rule requirements in `01-requirements.md` link to at least one acceptance criterion. Should-level `NFR-004` and `NFR-005` are also represented through `AC-016`.
- Acceptance criteria to tests: `02-acceptance-tests.md` maps every requirement to test IDs in the coverage matrix, and each acceptance, unit, integration, regression, and manual test references requirement IDs and acceptance criteria. No orphan critical acceptance criteria were identified.

## Coverage Review
- Must requirements covered: Yes. `BR-001` through `BR-006`, `FR-001` through `FR-016`, `NFR-001` through `NFR-003`, and `RULE-001` through `RULE-009` all have mapped acceptance criteria and tests.
- Missing or weak coverage: No blocking missing coverage. The main weak spots are clarifications rather than coverage holes: empty typeahead query behavior, strict handling of removed `/v1/search/projects` browse filters, and final relevance scoring/tie-breakers.
- Manual-only coverage: Appropriate and justified. Manual checks are limited to source-diff review for no rendered UI, architecture/privacy inspection for AppView-only/private recents behavior, bounded/index-aware query review, and this document review.

## Risk And Approval Review
- Risk level: Medium.
- Review requirement: Review recommended and now completed. The risk level is appropriate because the slice crosses AppView routes/stores, Flutter clients/providers, ranking, pagination, recent-search payloads, and project/search boundaries.
- Approval notes: Approved with notes. The coding planner can proceed if it treats the notes as implementation-design constraints and does not silently choose behavior that conflicts with the stricter tests.

## Coding Plan Readiness
- Ready for coding planning: Yes.
- Recommended first step: Start with `UT-001` for parsing and validating `GET /v1/search/suggestions` as a bounded, non-paginated, authenticated top-N suggestion request, as recommended in `02-acceptance-tests.md` §11.
- Blocking issues: None identified.

## Notes For Next Stage
- Keep `01-requirements.md`, `02-acceptance-tests.md`, and this review document as the source of truth for the coding plan.
- Treat this as a non-UI slice. Do not add rendered search/project UI, visual navigation changes, or UI-specific widgets beyond compile compatibility.
- Preserve `/v1/facets/*` compatibility while introducing shared suggestion logic.
- Keep project browsing/filtering under `/v1/projects` and the Flutter projects feature boundary; keep `/v1/search/projects` text-search-only.
- Define deterministic relevance scoring/tie-breakers before implementing AppView store tests for submitted post/project search.
- Prefer validation errors for empty unified-suggestion queries and unsupported browse filters on `/v1/search/projects`, unless the requirements and tests are revised together.
