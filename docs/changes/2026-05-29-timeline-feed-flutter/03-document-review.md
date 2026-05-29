# Document Review: Timeline Feed Flutter

## Verdict
Status: Approved with notes
Reviewer: gpt-5.5 document reviewer
Date: 2026-05-29
Risk level: Medium

## Summary

The workflow documents are consistent and ready for coding planning. `01-requirements.md` captures the confirmed Flutter-side scope: AppView timeline consumption, a paginated Feed tab, top-level compose integration, and optimistic insertion into live timeline state, while explicitly excluding AppView, PDS, generic feed-framework, discovery, ranking, and durable-cache work. `02-acceptance-tests.md` provides practical API-client, repository, provider, widget/acceptance, regression, test-data, and manual-check coverage for the Must requirements and the main risks around cursor pagination, optimistic dedupe, shared post interactions, and generated-code drift.

No blocking gaps were found. The only notes are non-blocking implementation-planning considerations around keeping the repository plumbing test practical and preserving shared profile/feed cache behavior.

## Findings

| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| DR-001 | Suggestion | Tests / Implementation planning | `IT-004` asks for a production `ApiPostRepository` delegation test using a fake/stub `PostApiClient` or a repository-targeted seam. The current app code may not have a convenient seam for stubbing `PostApiClient` directly, so the coding planner should choose the least invasive verification path, such as compile-time repository/fake coverage plus provider tests, rather than adding abstractions solely for this test. | `02-acceptance-tests.md` IT-004, UT-001; `01-requirements.md` FR-002, RULE-001 | During coding planning, keep `IT-004` flexible: verify no handle/DID input and repository method plumbing without over-engineering the API client boundary. |
| DR-002 | Suggestion | Risk / Cache consistency | The documents correctly identify shared cache-update risk for timeline, profile lists, likes/reposts, replies, create, and delete. Implementation planning should explicitly decide whether to extract shared helpers or update each live provider directly, then place tests accordingly. | `01-requirements.md` RISK-001 through RISK-003, FR-011, FR-012, FR-015; `02-acceptance-tests.md` UT-006 through UT-010, REG-003 through REG-005 | Carry this into the coding plan as an implementation design choice; no document changes required. |

## Traceability Review

- Planning to requirements:
  - The confirmed recommended scope is preserved in `01-requirements.md` Q1, Recommended Direction, Goals, and Desired Behavior.
  - Scope boundaries are clear in Non-Goals NG-001 through NG-008, especially excluding AppView changes, PDS reads, generic feed framework work, recommendations/discovery, durable cache, and dependency/lexicon changes.
  - Risks from discovery are carried into requirements as RISK-001 through RISK-007, with generated-code and optimistic-dedupe concerns explicitly captured.
  - Open questions are correctly marked non-blocking and future-facing.
- Requirements to acceptance criteria:
  - Every Must `BR`, `FR`, `NFR`, and `RULE` has at least one linked acceptance criterion.
  - Acceptance criteria are externally verifiable through API/client tests, provider tests, widget tests, command checks, or review/manual checks.
  - Should requirements `FR-008`, `FR-013`, and `NFR-003` are also covered.
- Acceptance criteria to tests:
  - Every acceptance criterion `AC-001` through `AC-021` has at least one linked test, regression command, or justified manual check in `02-acceptance-tests.md`.
  - Gherkin scenarios cover the user-visible Feed tab behavior, while tables cover API/client, provider, regression, and test-data needs.

## Coverage Review

- Must requirements covered:
  - `BR-001`, `FR-001`, `FR-002`, `NFR-001`, `NFR-002`, `RULE-001`, and `RULE-002` are covered by `IT-001` through `IT-004` and related provider tests.
  - Timeline state, pagination, load-more failure, and opaque cursor behavior (`FR-003` through `FR-007`) are covered by `UT-002` through `UT-005`, `AT-005`, and `AT-006`.
  - Feed UI and post-card behavior (`FR-009`, `FR-010`) are covered by `AT-002`, `AT-007`, and regression tests.
  - Like/repost, reply/comment, compose, optimistic prepend, and dedupe behavior (`FR-011`, `FR-012`, `FR-014`, `FR-015`, `RULE-003`, `RULE-004`) are covered by `AT-008` through `AT-012` and `UT-006` through `UT-009`.
  - Localization/accessibility and generated-code requirements (`NFR-004`, `NFR-005`) are covered by `AT-003`, `AT-004`, `REG-006`, and `MAN-001`.
- Missing or weak coverage:
  - None blocking. Performance/laziness and accessibility are appropriately listed as partial/manual coverage in `GAP-002` and `MAN-001`.
  - `IT-004` may need a pragmatic implementation-specific test shape, noted in DR-001.
- Manual-only coverage:
  - `MAN-001` for visual/accessibility smoke checking and lazy-scroll feel is justified.
  - `MAN-002` for no speculative feed framework is a reasonable review check for `BR-002`/`NFR-002`.

## Risk And Approval Review

- Risk level: Medium.
- Review requirement: Review recommended, not required, before coding planning.
- Approval notes:
  - The medium-risk areas—user-visible feed UI, cursor pagination, optimistic cache updates, generated files, and shared post interaction behavior—have concrete test paths.
  - Plannotator review for the requirements/test folder returned approved before this document-review stage.
  - No high-risk auth, security, migration, dependency, or lexicon changes are in scope.

## Coding Plan Readiness

- Ready for coding planning: Yes.
- Recommended first step: Start with `IT-001` in `app/test/feed/data/post_api_client_test.dart`, adding the failing `PostApiClient.listTimeline` no-cursor timeline parsing test for `GET /v1/feed/timeline`.
- Blocking issues: None.

## Notes For Next Stage

- Keep the coding plan test-first and follow the suggested order in `02-acceptance-tests.md` §11.
- Prefer reusing or lightly extracting existing profile-list/provider patterns instead of building a generic feed framework.
- Decide early how live timeline, profile post lists, and interaction/create/delete providers share cache updates; then ensure `UT-006` through `UT-010` and `REG-003` through `REG-005` cover that choice.
- Run generated-code tooling after provider/model/l10n changes and include generated outputs in the implementation stage if needed.
