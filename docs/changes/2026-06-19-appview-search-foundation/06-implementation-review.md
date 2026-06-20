# Implementation Review: AppView Search Foundation

## Verdict
Status: Changes required
Reviewer: OpenAI gpt-5.5 implementation reviewer
Date: 2026-06-20
Risk level: High

## Summary
The implementation adds the planned AppView search route family, request parsers, response wrappers, store methods, recent-search persistence, and migration support. `just test` passes, and the work stays AppView-only with no Flutter UI or lexicon changes.

However, several Must requirements are not yet satisfied or protected by the planned acceptance tests. Project search accepts `sort=popular` but still returns chronological results, profile search does not implement cursor pagination, recent-search payloads are not type-validated or semantically normalized, keyword search does not use the added FTS/indexed path, and the planned seeded store/integration tests for the most important behavior are missing. These gaps require another TDD pass before the implementation is ready.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Behavior | `GET /v1/search/projects?sort=popular` is parsed as valid, but `SearchProjects` always decodes a chronological cursor, selects `0::double precision AS popularity_score`, orders by `p.created_at DESC, p.uri DESC`, and never applies the required decayed popularity formula. Browse-all project search with `sort=popular` therefore returns chronological results. | `01-requirements.md` BR-006, FR-007, FR-017, FR-020, AC-013, AC-023, AC-026; `04-coding-plan.md` §§5.4, 6.4; `appview/internal/api/search_store.go` lines 101-160 | Add a project popularity query path using the centralized formula, active likes/reposts and visible replies, stable popularity cursors, and tests for IT-011/AT-008 including browse-all projects. |
| IR-002 | Important | Behavior / Pagination | Profile search parses cursors but ignores them in the store, uses `LIMIT req.Limit` instead of `limit + 1`, returns an empty next cursor, and has no profile cursor encoder/decoder despite the coding plan. Multi-page profile search cannot work and following a cursor cannot continue the result order. | `01-requirements.md` FR-004, FR-005, FR-016, AC-004, AC-015; `04-coding-plan.md` §6.5; `appview/internal/api/search_store.go` lines 35-78; `appview/internal/api/search_cursor.go` lines 1-109 | Implement deterministic profile seek cursors over followed rank, relevance rank, handle, and DID; fetch `limit + 1`; return next cursors; add profile pagination tests. |
| IR-003 | Important | Behavior / Privacy | Recent-search saves only validate top-level `type`, `displayLabel`, and raw JSON size. The payload is canonical JSON but not type-validated or semantically normalized, so invalid/non-rerunnable payloads such as `null` or `{}` can be saved, and equivalent searches with different casing/defaults may not de-duplicate as required. | `01-requirements.md` FR-014, FR-021, FR-022, AC-005, AC-006, AC-019, AC-027, EC-005; `02-acceptance-tests.md` UT-004, IT-008, IT-014; `appview/internal/api/search_request.go` lines 76-117 | Add type-specific recent payload validation/normalization for hashtag, profile, post, and project searches; ensure normalized de-duplication preserves the existing display label; add the missing unit and store lifecycle/privacy tests. |
| IR-004 | Important | Tests / Traceability | The implementation plan documents that dedicated seeded integration fixtures for exact hashtag equality, keyword search, project filters, top hashtags, popularity ordering, recent-search lifecycle, and moderation remain gaps. The repository currently has only route auth plus helper/unit tests for the search files, while the acceptance spec marked those Must behaviors as automated. | `02-acceptance-tests.md` AT-002 through AT-010, IT-002 through IT-014; `05-implementation-plan.md` lines 172-175; current test files under `appview/internal/api/search*_test.go` | Add the planned `search_store_test.go`, `search_recent_store_test.go`, top-hashtag, moderation, response, and handler/store fixtures so Must behavior is proven through public/store interfaces before approval. |
| IR-005 | Important | Risk / Performance | The migration adds FTS/search-vector indexes, but post and project keyword search use raw `lower(...) LIKE '%q%'` predicates and array `unnest` scans, so the documented indexed local FTS path is not used. `MAN-001` EXPLAIN review was also not run. | `01-requirements.md` NFR-002, NFR-006, AC-020, AC-022; `04-coding-plan.md` §§7.2, 8.2, 8.3; `05-implementation-plan.md` lines 153-157, 175; `appview/internal/api/search_store.go` lines 125-130, 272-277, 302-307; `appview/migrations/000019_search_foundation.up.sql` lines 18-32 | Switch keyword search to the intended PostgreSQL FTS/local indexed strategy or revise the implementation evidence with concrete indexed-query tests and EXPLAIN review. Keep API shape unchanged. |

## Requirement And Test Traceability
- Requirements implemented: Route registration/authentication scaffolding for `/v1/search/*`; exact hashtag normalization and result shape; profile relevance ranking helpers; post/project/hashtag/top-hashtag/recent handler and store scaffolding; recent-search table; response wrappers that omit `popularityScore`; bounded limits and basic validation.
- Tests implemented: Route auth/device tests; parser tests for post/profile/project query validation and hashtag normalization; cursor round-trip tests for chronological/popular post cursors; popularity formula unit tests; profile rank tuple tests; post response wrapper JSON test.
- Unplanned behavior: None identified outside the AppView slice. No Flutter UI, lexicon, or PDS-write changes were found.
- Remaining gaps: Project popularity sort, profile pagination, typed recent-search payload validation/normalization, seeded store/handler behavior tests, indexed keyword search evidence, and manual query-plan review.

## Test Evidence
- Commands reviewed:
  - `git status --short` and `git diff`: clean working tree before review; implementation inspected from `56fea97 feat: implement appview search foundation` / `b196ab4..HEAD`.
  - `git show --stat --name-status 56fea97`: confirmed changed implementation, tests, migrations, routes, and `05-implementation-plan.md` files.
  - `just test` from the repo root during review.
  - `05-implementation-plan.md` final evidence reporting `go test ./...`, `just fmt`, and `just test` passed.
- Passing evidence:
  - `just test` passed during review (`go test -race ./...` under `appview/`).
- Failing or skipped tests:
  - No failing tests observed.
  - Planned seeded store/integration tests and `MAN-001` EXPLAIN review were skipped or documented as gaps, and those gaps expose missing Must behavior.

## Risk Review
- Risk level: High
- Risk notes: The broad API/store surface compiles and passes current tests, but current tests are not strong enough for the requirements. Missing project popularity ordering and profile pagination are direct user-visible API behavior gaps. Recent-search validation gaps can persist private but invalid/non-rerunnable state. Keyword-search implementation does not match the indexed-path plan.
- Approval notes: Not ready to approve until the blocking findings above are fixed and covered by the planned TDD tests.

## UI Polish Recommendation
- Recommendation: Not needed
- Reason: This stage changed AppView API, storage, migrations, and tests only; no user-facing Flutter UI was changed.
- Suggested polish notes: None.

## Handoff Back To TDD Builder
- Required fixes:
  1. Implement and test project `sort=popular` for filtered and browse-all project search.
  2. Implement and test profile search cursor pagination.
  3. Implement and test type-specific recent-search payload validation/normalization, de-duplication, pruning, and DID-scoped hard delete.
  4. Add the planned seeded store/handler tests for hashtag equality, keyword search, project filters, top hashtags, popularity, moderation, response contracts, and recent-search privacy.
  5. Align keyword search with the documented FTS/indexed local path and run/document `MAN-001` query-plan review.
- Suggested next failing test: Start with `IT-011` / `AT-008` for `GET /v1/search/projects?sort=popular`, because the endpoint currently accepts the parameter but demonstrably cannot satisfy the required order.
- Verification to rerun: Focused package tests for the new failing tests, then `just test`; run/document `MAN-001` EXPLAIN checks before returning to implementation review.
