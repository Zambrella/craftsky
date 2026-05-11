# TDD Implementation Plan: Threads UX

## Inputs
- Requirements: `02-requirements.md`
- Tests: `03-acceptance-tests.md`
- Document review: `04-document-review.md`

## Implementation Rules
- Do not implement behavior without a linked requirement ID.
- Write or update a failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | AT-002 / UT-001 | BR-001, BR-002, FR-002, FR-003, FR-004 | AC-002, AC-003 | Fails because ancestors render before the selected reply at initial offset. |
| 2 | AT-001 / AT-009 | BR-001, FR-001, FR-004 | AC-001, AC-002 | May pass or need empty/top-level anchor assertions after Step 1. |
| 3 | AT-004 / UT-002 | BR-003, FR-005, RULE-001 | AC-004 | Fails because selected post is currently tappable. |
| 4 | AT-005 | FR-006, RULE-001 | AC-005 | Should continue to pass; add focused coverage for non-selected navigation. |
| 5 | AT-006 / UT-003 | BR-004, FR-007, FR-008 | AC-006 | Fails because composer lacks compact preview. |
| 6 | AT-007 | BR-004, FR-008 | AC-007 | Fails until preview text is capped at three lines. |
| 7 | AT-008 / UT-004 / REG-003 | FR-009 | AC-008 | Existing reply ref behavior should remain green. |
| 8 | REG-004 | NFR-001 | AC-009 | Existing text-scale regression should remain green with anchoring. |

## Implementation Steps
### Step 1: AT-002 / UT-001
- Write failing test: Added `anchors selected reply below app bar when opened` to `post_thread_page_test.dart`.
- Run command: `flutter test test/feed/pages/post_thread_page_test.dart --plain-name 'anchors selected reply below app bar when opened'`.
- Confirmed failure: Failed because selected reply text top was `476.5`, outside the tolerated below-app-bar region (`<216.0`).
- Implement: Replaced the ancestor-first `ListView` with a centered `CustomScrollView` so the selected post sliver is the initial anchor and ancestors remain above in reverse-growth scrollback.
- Run command: `dart format lib/feed/pages/post_thread_page.dart && flutter test test/feed/pages/post_thread_page_test.dart --plain-name 'anchors selected reply below app bar when opened'`.
- Refactor: Reversed ancestors in the pre-center sliver so visual scrollback remains root/top-most parent before immediate parent.
- Notes: Green. Focused test passes.

### Step 2: AT-001 / AT-009
- Write failing test: Added `anchors top-level post below app bar with replies below` and extended `shows empty reply state` with anchor/ordering assertions.
- Run command: `flutter test test/feed/pages/post_thread_page_test.dart --plain-name 'anchors top-level post below app bar with replies below'`; `flutter test test/feed/pages/post_thread_page_test.dart --plain-name 'shows empty reply state'`.
- Confirmed failure: No additional red after Step 1 because centered thread layout already satisfied top-level and empty anchoring.
- Implement: No code change needed for this step.
- Run command: Same focused commands.
- Refactor: None.
- Notes: Green. Both focused tests pass.

### Step 3: AT-004 / UT-002
- Write failing test: Added `tapping selected post does not push same thread route` to `post_thread_page_test.dart`.
- Run command: `flutter test test/feed/pages/post_thread_page_test.dart --plain-name 'tapping selected post does not push same thread route'`.
- Confirmed failure: Failed because tapping selected post caused a second same-route load (`['target', 'target']`).
- Implement: Set `_ThreadPostCard` `PostCard.onTap` to `null` when `isAnchor` is true.
- Run command: `dart format lib/feed/pages/post_thread_page.dart && flutter test test/feed/pages/post_thread_page_test.dart --plain-name 'tapping selected post does not push same thread route'`.
- Refactor: None.
- Notes: Green. Focused test passes.

### Step 4: AT-005
- Write failing test: Added `tapping non-selected reply navigates to that reply thread`; after code review, added `tapping non-selected ancestor navigates to ancestor thread` to cover ancestor navigation explicitly.
- Run command: `flutter test test/feed/pages/post_thread_page_test.dart --plain-name 'tapping non-selected reply navigates to that reply thread'`; later `flutter test test/feed/pages/post_thread_page_test.dart`.
- Confirmed failure: No failure after Step 3 because non-selected navigation remained intact.
- Implement: No code change needed for this step.
- Run command: Same focused command.
- Refactor: None.
- Notes: Green. Existing continuation test still remains as broader coverage for continuation navigation; added ancestor test closes AT-005 coverage gap.

### Step 5: AT-006 / UT-003
- Write failing test: Added `reply mode shows compact target preview above input` to `post_composer_sheet_test.dart`.
- Run command: `flutter test test/feed/widgets/post_composer_sheet_test.dart --plain-name 'reply mode shows compact target preview above input'`.
- Confirmed failure: Failed because `@alice.craftsky.social` was not rendered in reply mode.
- Implement: Added `_ReplyTargetPreview` above `BrandTextField` when `replyTarget` is present, using theme spacing/swatch tokens and showing author handle and target text.
- Run command: `dart format lib/feed/widgets/post_composer_sheet.dart && flutter test test/feed/widgets/post_composer_sheet_test.dart --plain-name 'reply mode shows compact target preview above input'`.
- Refactor: None.
- Notes: Green. Focused test passes.

### Step 6: AT-007
- Write failing test: Added `reply target preview limits long text to three lines` and parameterized `_replyTarget` text fixture.
- Run command: `flutter test test/feed/widgets/post_composer_sheet_test.dart --plain-name 'reply target preview limits long text to three lines'`.
- Confirmed failure: No additional red after Step 5 because `_ReplyTargetPreview` already used `maxLines: 3` and ellipsis for FR-008.
- Implement: No additional code change needed.
- Run command: Same focused command after formatting.
- Refactor: None.
- Notes: Green. Focused test passes.

### Step 7: AT-008 / UT-004 / REG-003
- Write failing test: Existing `reply mode shows reply copy and forwards reply refs` already covers nested reply refs.
- Run command: `flutter test test/feed/widgets/post_composer_sheet_test.dart --plain-name 'reply mode shows reply copy and forwards reply refs'`.
- Confirmed failure: No red after preview UI change; reply ref behavior remained intact.
- Implement: No code change needed.
- Run command: Same focused command.
- Refactor: None.
- Notes: Green. Existing reply ref regression passes.

### Step 8: REG-004
- Write failing test: Existing `avoids overflow at narrow width and text scale 2` regression covers this.
- Run command: `flutter test test/feed/pages/post_thread_page_test.dart --plain-name 'avoids overflow at narrow width and text scale 2'`.
- Confirmed failure: No red after thread layout changes.
- Implement: No code change needed.
- Run command: Same focused command.
- Refactor: None.
- Notes: Green. Text-scale regression passes.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Review completed or explicitly skipped

## Final Verification
- Focused thread tests: `flutter test test/feed/pages/post_thread_page_test.dart` — passed.
- Focused composer tests: `flutter test test/feed/widgets/post_composer_sheet_test.dart` — passed.
- Broader feed tests: `flutter test test/feed` — passed after code-review coverage fix.
- Manual checks not run: MAN-001 visual scroll-jank check and MAN-002 visual theme-fit check require interactive app review.
- Coverage gaps: None blocking; manual visual checks remain as documented.
