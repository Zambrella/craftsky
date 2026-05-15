# Implementation Review: Reply Comment Section

## Verdict
Status: Approved with notes
Reviewer: OpenAI gpt-5.5 implementation reviewer
Date: 2026-05-15
Risk level: Medium

## Summary
The implementation covers much of the requested comment-section replacement: the `/comments` read surface exists, `/thread` route usage is removed from the active client/API surface, comment/reply models and providers were added, focus status handling exists, reply parentage is preserved, and the focused test suites reviewed here pass. This review originally identified two Must-level behavior/doc gaps. Follow-up work accepted IR-001 as a legitimate bug and fixed it with target-including focused branch hydration. Follow-up work resolved IR-002 by updating `02-requirements.md` and `03-acceptance-tests.md` to make recursive flattened branch replies the authoritative behavior.

`04-document-review.md` was not present in this workflow folder; `05-implementation-plan.md` records that it was absent at implementation start.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Behavior / Tests | Follow-up addressed the original bug where focused replies outside the first reply page were not guaranteed to be returned. | `02-requirements.md` FR-004, RULE-007, AC-003, AC-021, EC-011; `03-acceptance-tests.md` AT-002, IT-006, IT-011; `appview/internal/api/post.go`, `appview/internal/api/post_store.go`, `appview/internal/api/post_test.go`, `appview/internal/api/post_store_test.go` | Completed: added target-including bounded focused branch hydration and tests. |
| IR-002 | Suggestion | Behavior / Traceability | User feedback reclassified the original direct-only concern as incorrect documentation. `/replies` now intentionally returns visual comment-branch replies, including flattened descendants. | `02-requirements.md` FR-010, RULE-003, ASM-002, AC-006, AC-010, AC-026; `03-acceptance-tests.md` IT-007, REG-005; `app/test/feed/data/post_api_client_test.dart` | Completed: updated requirements, acceptance tests, implementation notes, and relevant client decode coverage. |
| IR-003 | Suggestion | Traceability | Follow-up updated the implementation plan to record IR-001/IR-002 decisions, test order, red/green notes, and verification evidence. | `05-implementation-plan.md` | Completed. |

## Requirement And Test Traceability
- Requirements implemented:
  - Root comment-section API and client state: BR-002, FR-002, FR-005, FR-006, FR-022, FR-023, NFR-001.
  - Focus parameter/status contract for malformed, missing, mismatched, comment, and reply focus: BR-001, FR-003, FR-019, FR-020, FR-021.
  - Focused branch promotion and no duplicate focused comments in covered cases: FR-004, FR-022, NFR-003/NFR-004.
  - Comment sort options and viewer-authored grouping: BR-003, FR-007, FR-008, RULE-005, RULE-006.
  - Two visible levels and flattened metadata for covered focused/deep reply cases: FR-013, FR-014, FR-024.
  - Composer mention and parent/root ref preservation for reply-to-reply creation: FR-014, FR-015, RULE-001.
  - `/thread` route/API/model/provider removal from active surfaces: FR-001.
  - No `lexicon/` changes were present in `main...HEAD`.
- Tests implemented:
  - Backend handler/store/route tests for `/comments`, focus status, focus promotion, reply metadata, comment paging, sorting/grouping, route removal, and branch reply pagination.
  - Flutter model/provider/page/client tests for comment-section decoding, route focus, lazy loading, branch controls, sort changes, creation insertion, localization, and stale-thread replacement.
- Unplanned behavior:
  - None identified after follow-up documentation alignment.
- Remaining gaps:
  - OS-level deep-link launch/push notification delivery and visual copy clarity for the stubbed `follows` sort remain manual/out of scope as documented in `03-acceptance-tests.md`.

## Test Evidence
- Commands reviewed:
  - `cd appview && go test ./internal/api ./internal/routes`
  - `cd app && flutter test test/feed/models/post_comment_section_test.dart test/feed/models/post_comment_section_state_test.dart test/feed/providers/post_comment_section_provider_test.dart test/feed/pages/post_comment_section_page_test.dart test/feed/data/post_api_client_test.dart test/feed/widgets/post_composer_sheet_test.dart test/router/router_redirect_test.dart`
  - `git status --short`
  - `git diff --stat main...HEAD`
  - `git diff --name-only main...HEAD -- lexicon`
- Passing evidence:
  - Backend API/routes tests passed.
  - Focused Flutter comment-section/client/provider/widget/router tests passed.
  - `git diff --name-only main...HEAD -- lexicon` returned no lexicon changes.
- Failing or skipped tests:
  - No tests failed in the verification commands run during review.
  - The passing suite does not cover the focused-target-after-first-10 branch case required by AC-021/IT-011.
  - `04-document-review.md` was absent; review used `02-requirements.md`, `03-acceptance-tests.md`, and `05-implementation-plan.md` plus implementation diffs.

## Risk Review
- Risk level: Medium
- Risk notes:
  - Deep-link reliability remains a core Must requirement; follow-up tests now cover the previously missing off-page focused reply case.
  - Normal recursive reply expansion is now documented as intentional visual branch behavior.
- Approval notes:
  - Follow-up work addressed the blocking findings. Manual OS-level deep-link launch checks remain as documented manual coverage.

## Handoff Back To TDD Builder
- Required fixes:
  - None remaining from this review after follow-up.
- Suggested next failing test:
  - None; proceed to final review/manual checks as desired.
- Verification to rerun:
  - `cd appview && go test ./internal/api ./internal/routes`
  - `cd app && flutter test test/feed/models/post_comment_section_test.dart test/feed/models/post_comment_section_state_test.dart test/feed/providers/post_comment_section_provider_test.dart test/feed/pages/post_comment_section_page_test.dart test/feed/data/post_api_client_test.dart test/feed/widgets/post_composer_sheet_test.dart test/router/router_redirect_test.dart`

## Follow-up Resolution
- IR-001: Addressed in follow-up TDD loop. Added a failing handler test for a focused reply after the first branch page, added `ListCommentBranchRepliesAround`, verified the focused branch includes the target while remaining bounded, and tightened focused-slice cursor emission so the cursor only appears when later branch replies can be loaded.
- IR-002: Reclassified by user feedback as incorrect documentation, not an implementation bug. Updated `02-requirements.md`, `03-acceptance-tests.md`, and relevant client decode coverage so `/replies` means visual comment-branch replies with flattened descendants.
- IR-003: Addressed by updating `05-implementation-plan.md` with follow-up decisions, test order, red/green notes, and verification commands.
