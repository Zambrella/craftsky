# Implementation Review: Search Refinements Before UI Slice

## Verdict
Status: Changes required
Reviewer: OpenAI gpt-5.5 implementation reviewer
Date: 2026-06-21
Risk level: Medium

## Summary
The implementation covers most of the planned AppView and Flutter non-UI search refinement surface: new suggestion and hashtag result contracts, disjoint relevance-first submitted Posts/Projects searches, exact hashtag sort parsing, project browse filters under `/v1/projects`, canonical craft tokens, refined recent-search payloads, Flutter models/clients/providers, and no rendered UI changes. The implementation log records focused and broader AppView/Flutter verification passing.

One blocking issue remains: the existing facet hashtag suggestion path still has its own SQL and does not apply the same visible/top-level count rules as the unified search suggestion/hashtag query path. That means hidden/takedown posts can still affect composer/profile hashtag suggestions while search suggestions do not, which violates the shared suggestion/count contract and leaves a missing regression test.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Behavior / Tests | `/v1/facets/hashtags` still uses separate count logic from the unified search suggestion/hashtag result path and omits the standard post visibility predicate. `SearchStore.SearchHashtags` filters visible top-level posts with `postVisibleModerationPredicate`, but `FacetStore.SearchHashtagSuggestions` counts recent root posts without that visibility filter, so hidden/takedown posts can affect facet autocomplete counts/ranking and search/facet suggestions can drift. Existing tests cover ranking but not hidden/takedown parity. | `01-requirements.md` FR-003, FR-004, AC-004, AC-005, AC-016; `02-acceptance-tests.md` IT-003, REG-001; `appview/internal/api/search_store.go`; `appview/internal/api/facet_store.go`; `appview/internal/api/search_store_test.go` | Align facet hashtag suggestions with the shared visible count logic used by search suggestions/hashtag results, preferably by extracting/reusing one helper or by applying equivalent predicates. Add a regression test with hidden/takedown rows proving `/v1/facets/hashtags` and search hashtag suggestions/counts exclude non-visible posts consistently. |

## Requirement And Test Traceability
- Requirements implemented: Most planned Must requirements are represented in code and tests: `FR-001` unified suggestions, `FR-005` hashtag query results, `FR-006`/`FR-007` relevance-first disjoint submitted post/project search, `FR-008` exact hashtag sort parsing/results, `FR-009` through `FR-011` project browse/filter boundary, `FR-014`/`FR-015` craft-token canonicalization, `FR-016` refined recents, and `NFR-001` no rendered UI changes.
- Tests implemented: The implementation added AppView parser/store/route tests and Flutter model/client/provider/project/facet tests matching the approved implementation plan, with manual checks recorded for no UI changes, AppView-only/private recents, and bounded query review.
- Unplanned behavior: None identified beyond the facet hashtag suggestion divergence in IR-001.
- Remaining gaps: IR-001 must be fixed before approval. Accepted non-blocking gaps from the test spec remain: no rendered UI E2E in this slice, production-scale hashtag/project performance profiling deferred until needed, and profile recent display freshness as future work.

## Test Evidence
- Commands reviewed:
  - Implementation log focused AppView commands for steps 1-25.
  - Implementation log focused Flutter model/client/provider/project/facet commands for steps 26-35.
  - `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api ./internal/routes -count=1`.
  - `dart run build_runner build --delete-conflicting-outputs`.
  - `flutter test test/search test/shared/rich_text test/projects`.
  - `flutter analyze`.
- Passing evidence: The implementation log records all broader commands above passing after import fixes, plus manual checks `MAN-001`, `MAN-002`, and `MAN-003` passing.
- Failing or skipped tests: No failing command is documented in the final implementation log. The review identified a missing regression around facet hashtag visibility/count parity.

## Risk Review
- Risk level: Medium.
- Risk notes: The slice changes AppView API contracts, query semantics, generated Flutter models/providers, a migration, and project/search boundaries. Most risks are covered by focused tests. IR-001 is a moderation/visibility and shared-contract risk because suggestion counts can leak or rank hidden content differently between search and facet autocomplete.
- Approval notes: Not approved until IR-001 is corrected and verified.

## UI Polish Recommendation
- Recommendation: Not needed
- Reason: This is a non-UI slice. The implementation log records no diffs under `app/lib/search/pages`, `app/lib/projects/pages`, or router/navigation files, and no rendered search/project UI was added.
- Suggested polish notes: None.

## Handoff Back To TDD Builder
- Required fixes: Address IR-001.
- Suggested next failing test: Add an AppView DB-backed regression test for `FacetStore.SearchHashtagSuggestions` that seeds visible, hidden/takedown, old, reply/comment, and duplicate hashtag rows, then asserts facet hashtag suggestions use the same visible 28-day distinct top-level post counts as the search suggestion/hashtag query path.
- Verification to rerun:
  - `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:5433/craftsky_dev?sslmode=disable' go test ./internal/api ./internal/routes -count=1`
  - `flutter test test/search test/shared/rich_text test/projects`
  - `flutter analyze`
