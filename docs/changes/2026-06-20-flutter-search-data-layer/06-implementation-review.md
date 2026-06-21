# Implementation Review: Flutter Search Data Layer

## Verdict
Status: Changes required
Reviewer: GPT-5.5 implementation-reviewer
Date: 2026-06-21
Risk level: Medium

## Summary
The Flutter search data layer implementation is well scoped to the requested non-UI slice and the focused search/regression tests pass. The implementation adds the expected search models, Dio-backed API client, repository boundary, Riverpod providers, generated files, and mapper registration without changing AppView, lexicon, route, dependency, or rendered search-page behavior.

However, two planned Must acceptance-test areas are not fully covered by the committed tests: all four recent-search typed payload variants, and load-more behavior for every result provider family. Because the workflow requires planned Must tests to be implemented or explicitly documented as gaps, this stage is not ready to approve until those coverage gaps are addressed.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Tests / Traceability | Recent-search typed payload coverage is incomplete. The acceptance spec requires list/save coverage for hashtag, profile, post, and project payloads, but the committed API-client/model tests only list-decode hashtag/profile payloads, save a project payload, serialize profile/project payloads, and deserialize one hashtag item. This leaves required post payload behavior and several save/list variants unprotected. | `02-acceptance-tests.md` `AT-006`, `IT-007`, `IT-008`, `UT-005`; `05-implementation-plan.md` steps 10-11 and 21; `app/test/search/data/search_api_client_test.dart`; `app/test/search/models/recent_search_test.dart` | Add or expand tests so recent list decoding and save serialization cover all supported payload types: `hashtag`, `profile`, `post`, and `project`. Ensure profile payloads still exclude `sort`, project filters use only supported keys, and all payloads are rerunnable typed representations. |
| IR-002 | Important | Tests / Traceability | Result-provider pagination coverage is incomplete. The acceptance spec targets hashtag, profile, post, and project providers for `loadMore()` cursor pass-through, accumulation, previous-data preservation, duplicate suppression, and concurrency/no-more guards, but the committed tests exercise `loadMore()` only through `hashtagSearchProvider`; profile/post/project provider tests cover initial fetch only. | `02-acceptance-tests.md` `AT-007`, `IT-012`, `IT-013`; `04-coding-plan.md` §4.5 and §5.2; `05-implementation-plan.md` steps 16-18; `app/test/search/providers/hashtag_search_provider_test.dart`; `app/test/search/providers/profile_search_provider_test.dart`; `app/test/search/providers/post_search_provider_test.dart`; `app/test/search/providers/project_search_provider_test.dart` | Add focused `loadMore()` tests for profile, post, and project providers, or explicitly document a justified test gap. Coverage should include opaque cursor pass-through, append behavior, no-more guard, and duplicate suppression by stable identity where applicable. |

## Requirement And Test Traceability
- Requirements implemented: The code implements the requested Flutter-only, non-UI search data layer (`BR-001` through `BR-003`, `FR-001` through `FR-016`, `NFR-001` through `NFR-003`, `RULE-001` through `RULE-004`) via `app/lib/search/data`, `app/lib/search/models`, `app/lib/search/providers`, generated mapper/provider files, and `app/lib/bootstrap.dart` mapper registration.
- Tests implemented: Focused search data/model/provider tests exist under `app/test/search/`, plus the search-page and facet autocomplete regressions. Implemented areas include endpoint paths/query/body shapes, `unwrapApi` error mapping, repository/provider construction, initial provider loads, representative hashtag pagination, top hashtags, recent save/delete refresh, and non-UI regression checks.
- Unplanned behavior: None identified. The implementation commit did not modify `appview/`, `lexicon/`, `app/pubspec.yaml`, `app/pubspec.lock`, `app/lib/search/pages`, or `app/lib/router`.
- Remaining gaps: `IR-001` and `IR-002` are required test coverage gaps against the approved acceptance-test specification.

## Test Evidence
- Commands reviewed:
  - `git status --short` — clean before review artifact creation.
  - `git show --stat --name-status --oneline 7d89ddb` — implementation scope reviewed.
  - `git diff -- appview lexicon app/pubspec.yaml app/pubspec.lock app/lib/search/pages app/lib/router` — no forbidden scoped working-tree diff.
  - `flutter test test/search test/shared/rich_text/facet_suggestion_repository_test.dart` from `app/` — passed.
  - `flutter analyze` from `app/` — passed, no issues.
  - `flutter test` from `app/` — failed in two existing non-search feed composer tests.
- Passing evidence:
  - Focused search/facet regression command passed all reported tests.
  - Analyzer passed with no issues.
- Failing or skipped tests:
  - Full `flutter test` currently fails in `test/feed/pages/feed_page_composer_entry_test.dart` and `test/feed/widgets/post_type_chooser_test.dart`, both expecting `Craft type`. These failures are outside this search-data-layer diff and were already documented in `05-implementation-plan.md`; no feed/project composer files were changed by the implementation commit.
  - No additional tests were skipped during this review.

## Risk Review
- Risk level: Medium
- Risk notes: The production code aligns with AppView JSON/HTTP conventions, uses the shared authenticated Dio provider, keeps cursors/recent IDs opaque, keeps recent searches private to AppView calls, and avoids local/PDS persistence. Remaining risk is primarily test traceability for recent payload variants and provider pagination across all result families.
- Approval notes: The implementation should be straightforward to approve after the required test coverage gaps are closed and focused tests/analyzer are rerun.

## UI Polish Recommendation
- Recommendation: Not needed
- Reason: The implementation does not introduce rendered user-facing search UI changes; `SearchPage`/route behavior remains unchanged.
- Suggested polish notes: None.

## Handoff Back To TDD Builder
- Required fixes:
  - Address `IR-001` by expanding recent-search model/API-client tests to cover all supported typed payload variants for list and save flows.
  - Address `IR-002` by expanding provider pagination tests beyond hashtag to profile, post, and project providers, or by documenting a justified test gap in the workflow artifact.
- Suggested next failing test:
  - Start with `app/test/search/models/recent_search_test.dart` / `app/test/search/data/search_api_client_test.dart` for missing post/hashtag/profile/project recent payload variants, then add `loadMore()` tests in `app/test/search/providers/profile_search_provider_test.dart`, `post_search_provider_test.dart`, and `project_search_provider_test.dart`.
- Verification to rerun:
  - From `app/`: `flutter test test/search test/shared/rich_text/facet_suggestion_repository_test.dart`
  - From `app/`: `flutter analyze`
  - From `app/`: `flutter test` and document any unrelated failures.
