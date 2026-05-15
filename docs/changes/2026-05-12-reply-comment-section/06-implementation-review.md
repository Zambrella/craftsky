# Implementation Review: Reply Comment Section

## Verdict
Status: Changes required
Reviewer: OpenAI gpt-5.5 implementation reviewer
Date: 2026-05-15
Risk level: High

## Summary
The implementation covers much of the requested comment-section replacement: the `/comments` read surface exists, `/thread` route usage is removed from the active client/API surface, comment/reply models and providers were added, focus status handling exists, reply parentage is preserved, and the focused test suites reviewed here pass. However, two Must-level behavior gaps remain against the persisted requirements and acceptance tests. Focused replies outside the first reply page are not guaranteed to be included in the response, and normal action-expanded reply loading now returns recursive branch descendants even though the requirements and acceptance spec say normal expansion should load direct replies only, with deeper descendants appearing only for focus/newly-created flattened cases.

`04-document-review.md` was not present in this workflow folder; `05-implementation-plan.md` records that it was absent at implementation start.

## Findings
| ID | Severity | Area | Finding | References | Required Action |
|---|---|---|---|---|---|
| IR-001 | Important | Behavior / Tests | Focused replies outside the first reply page are not guaranteed to be returned. `GetPostCommentsHandler` resolves the focused reply's comment ancestor, then calls `ListCommentBranchReplies(..., limit=10, cursor="")` and renders that first oldest-first branch page. If the focused reply is after the first 10 branch replies, the target will not be in `replies.items`, despite `focus.status = "included"`. The current handler test uses a fake store that already returns the focused reply, so it does not exercise the real store ordering/pagination case described by the acceptance tests. | `02-requirements.md` FR-004, RULE-007, AC-003, AC-021, EC-011; `03-acceptance-tests.md` AT-002, IT-006, IT-011; `appview/internal/api/post.go:531-535`, `appview/internal/api/post_store.go:563-570`, `appview/internal/api/post_test.go:1450-1499` | Add a store-backed or handler+store test with more than 10 earlier replies and a focused reply after the first page. Then implement a bounded target-including focused slice, or update the persisted requirements/acceptance tests if the product decision has changed. The API must not report an included focused reply while omitting the target from the loaded focused branch. |
| IR-002 | Important | Behavior / Traceability | The actioned `/v1/posts/{commentDid}/{commentRkey}/replies` loader now returns recursive comment-branch descendants, not only direct replies. The persisted requirements specify that activating “view replies” loads the first 10 direct replies oldest-first, and the acceptance spec explicitly says deeper replies only appear through focused/flattened cases handled elsewhere. Returning all descendants during normal expansion changes visible behavior by surfacing reply-to-reply records without a focus or newly-created insertion path. | `02-requirements.md` FR-010, RULE-003, ASM-002, AC-006, AC-010, AC-014; `03-acceptance-tests.md` IT-007, REG-005; `appview/internal/api/post.go:331`, `appview/internal/api/post_store.go:543-563`, `appview/internal/api/post_store_test.go:489-529`, `app/lib/feed/data/post_api_client.dart:42-43`, `app/lib/feed/providers/post_comment_section_provider.dart:139-146` | Restore direct-only action expansion for `/replies` and keep recursive/flattened branch hydration limited to focus/newly-created cases, or update `02-requirements.md`, `03-acceptance-tests.md`, and `05-implementation-plan.md` to make recursive visual branch loading the new source of truth before approval. Add tests that distinguish normal direct expansion from focused branch hydration. |
| IR-003 | Important | Traceability | The implementation plan is stale relative to the final behavior. It still records the focused reply slice as a single focused item in Steps 8-9 and describes the direct-replies regression path in REG-005, while the latest implementation hydrates first-page branch replies and uses recursive branch pagination. This makes the persisted source of truth ambiguous for the next TDD loop. | `05-implementation-plan.md:130-146`, `05-implementation-plan.md:490-497`, `appview/internal/api/post.go:531-535`, `appview/internal/api/post_store.go:543-563` | After deciding whether IR-001/IR-002 should be fixed in code or accepted as changed product behavior, update the workflow documents so requirements, tests, and implementation notes agree. |

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
  - Recursive branch replies are now returned by normal `/replies` action expansion; this is not reflected in `02-requirements.md`/`03-acceptance-tests.md`.
  - Focused reply hydration now loads the first bounded branch page rather than a target-centered bounded slice; this does not satisfy the target-outside-first-page case in the docs.
- Remaining gaps:
  - Missing automated coverage proving a focused reply after the first 10 branch replies is included.
  - Missing coverage that normal action expansion excludes deeper descendants if the persisted direct-only requirement remains valid.

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
- Risk level: High
- Risk notes:
  - Deep-link reliability is a core Must requirement. Reporting `focus.status = "included"` while omitting an off-page focused reply would make share/push links land in the wrong context.
  - Normal recursive reply expansion may change conversation visibility and pagination semantics beyond what the approved documents describe.
- Approval notes:
  - The implementation is close, but the remaining gaps affect Must acceptance criteria and need a new TDD loop or explicit document update before approval.

## Handoff Back To TDD Builder
- Required fixes:
  1. Decide whether the source of truth remains direct-only normal reply expansion plus target-centered focused slices. If yes, implement those behaviors and add the missing tests.
  2. If the desired product behavior has changed to recursive visual branch loading and first-page focused branch hydration, update `02-requirements.md`, `03-acceptance-tests.md`, and `05-implementation-plan.md` before re-review.
- Suggested next failing test:
  - Add a backend store-backed focus test where a comment branch has at least 11 replies and the focus URI is the 11th or later reply. Assert the response includes the focused reply in the loaded focused branch while remaining bounded and exposing predictable pagination state.
- Verification to rerun:
  - `cd appview && go test ./internal/api ./internal/routes`
  - `cd app && flutter test test/feed/models/post_comment_section_test.dart test/feed/models/post_comment_section_state_test.dart test/feed/providers/post_comment_section_provider_test.dart test/feed/pages/post_comment_section_page_test.dart test/feed/data/post_api_client_test.dart test/feed/widgets/post_composer_sheet_test.dart test/router/router_redirect_test.dart`
