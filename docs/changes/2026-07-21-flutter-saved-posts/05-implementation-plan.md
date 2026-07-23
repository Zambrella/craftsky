# TDD Implementation Plan: Flutter Saved Posts

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved with notes`)
- Coding plan: `04-coding-plan.md`
- Risk: Medium
- Blocking issues: None
- Commit authorization: Not granted; do not stage, commit, push, or create a PR.

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Use the detailed test rows in `02-acceptance-tests.md` as authoritative traceability.
- Write or update one focused failing test before implementation.
- Run the smallest relevant test first and record a meaningful red result.
- Refactor only after the focused test is green.
- Keep saves and folder data account-scoped and private.
- Never enumerate hydrated saved rows to delete a folder's saves.
- Do not edit generated files manually; run the repository generators.
- Keep traceability and actual command evidence updated below.

## Test Order

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | UT-001 | FR-001 | AC-001, AC-024 | Fails: `Post` lacks saved viewer fields. |
| 2 | UT-002 | FR-007 | AC-027 | Fails: typed saved models do not exist. |
| 3 | UT-003 | FR-004, FR-005, FR-013, FR-026, RULE-007 | AC-004, AC-005, AC-017 | Fails: folder validation and ID identity do not exist. |
| 4 | IT-001 | FR-004, FR-007, FR-015, FR-017, FR-026 | AC-003, AC-004, AC-014, AC-015, AC-016, AC-017, AC-027 | Fails: saved API client does not exist. |
| 5 | IT-002 | FR-007 | AC-027 | Fails: saved repository seam does not exist. |
| 6 | UT-004 | FR-006, FR-018, NFR-003 | AC-006, AC-018, AC-024, AC-029 | Fails: account URI state seam does not exist. |
| 7 | UT-013 | FR-004, FR-005, RULE-001, RULE-004 | AC-003, AC-005, AC-030 | Fails: dialog controller does not exist. |
| 8 | IT-004 | FR-005, FR-006, FR-011, NFR-003, RULE-001, RULE-004, RULE-005 | AC-005, AC-006, AC-012, AC-018, AC-030 | Fails: confirmation mutation flow does not exist. |
| 9 | IT-005 | FR-003, FR-006, FR-018, NFR-003 | AC-006, AC-018, AC-024, AC-029 | Fails: optimistic unsave seam does not exist. |
| 10 | AT-001 | BR-001, FR-001, FR-002, FR-003, FR-024, RULE-002 | AC-001, AC-002, AC-023, AC-032 | Fails: full post bookmark is absent. |
| 11 | AT-002 | BR-001, FR-003, FR-004, FR-005, FR-006, FR-024, FR-026, NFR-003, RULE-001, RULE-004, RULE-007 | AC-002, AC-003, AC-004, AC-005, AC-006, AC-017, AC-023, AC-030 | Fails: chooser UI is absent. |
| 12 | AT-003 | BR-001, FR-003, FR-006, FR-018, NFR-003, RULE-002 | AC-006, AC-018, AC-024, AC-029 | Fails: one-shot optimistic unsave is absent. |
| 13 | UT-005 | FR-010, FR-013, FR-026, NFR-002 | AC-009, AC-010, AC-017, AC-031 | Fails: pagination reconciliation does not exist. |
| 14 | IT-003 | FR-009, FR-010, FR-013, FR-026, NFR-002, RULE-008 | AC-008, AC-009, AC-010, AC-017, AC-028, AC-031 | Fails: independent collection state does not exist. |
| 15 | UT-006 | FR-009, RULE-008, RULE-010, RULE-011 | AC-008, AC-028, AC-031 | Fails: overview projection does not exist. |
| 16 | UT-007 | FR-011, FR-012, FR-022 | AC-011, AC-021 | Fails: exact saved destination helper does not exist. |
| 17 | UT-014 | FR-011, RULE-005, RULE-008 | AC-008, AC-012 | Fails: server-confirmed folder changes do not reconcile shared chronology state. |
| 18 | UT-011 | FR-008, FR-013 | AC-007, AC-031 | Fails: canonical and redacted routes do not exist. |
| 19 | AT-004 | BR-002, FR-008, FR-009, FR-010, FR-013, RULE-005, RULE-006, RULE-008, RULE-010, RULE-011 | AC-007, AC-008, AC-028, AC-031 | Fails: Settings saved overview is absent. |
| 20 | AT-005 | FR-009, FR-010, FR-013, FR-025, FR-026, NFR-002, RULE-008, RULE-010 | AC-009, AC-010, AC-017, AC-019, AC-028, AC-031 | Fails: collection pages and states are absent. |
| 21 | AT-006 | BR-002, FR-006, FR-011, FR-012, FR-018, NFR-003, RULE-005 | AC-011, AC-012, AC-018, AC-030 | Fails: saved row actions and navigation are absent. |
| 22 | AT-007 | BR-003, FR-005, FR-013, FR-014, FR-024, FR-026, RULE-003, RULE-007, RULE-011 | AC-005, AC-013, AC-017, AC-023, AC-031 | Fails: folder management UI is absent. |
| 23 | IT-007 | FR-008, FR-013, RULE-006 | AC-007, AC-026, AC-031 | Fails: full-screen typed route stack is absent. |
| 24 | IT-008 | FR-009, FR-010, FR-011, FR-012, FR-013, FR-014, FR-023, RULE-010, RULE-011 | AC-008–AC-013, AC-017, AC-019, AC-022, AC-023, AC-028, AC-030, AC-031 | Fails: collection screens are absent. |
| 25 | UT-009 | FR-015 | AC-016 | Fails: strict delete query parser is absent. |
| 26 | UT-012 | FR-015 | AC-015, AC-016 | Fails: delete-mode error mapping is absent. |
| 27 | IT-009 | FR-015 | AC-014, AC-015, AC-016 | Fails: handler cannot select delete modes. |
| 28 | IT-010 | BR-003, FR-015, RULE-003, RULE-005 | AC-013, AC-014, AC-015 | Fails: preserve-mode signature/coverage is absent. |
| 29 | IT-011 | BR-003, FR-015, FR-016, FR-017, RULE-003 | AC-014, AC-015, AC-025 | Fails: remove mode and rollback are absent. |
| 30 | IT-012 | BR-005, FR-015, FR-016, FR-025, FR-026, NFR-001 | AC-014, AC-015, AC-019, AC-025 | Fails: privacy/redaction proof is incomplete. |
| 31 | AT-008 | BR-003, BR-005, FR-015, FR-016, FR-017, NFR-001, RULE-003, RULE-006 | AC-014, AC-015, AC-016, AC-025 | Fails: atomic end-to-end acceptance is unproven. |
| 32 | UT-008 | FR-020, FR-021, FR-023, NFR-004 | AC-020, AC-021, AC-022 | Fails: shared summary does not exist. |
| 33 | AT-010 | BR-004, FR-002, FR-020, FR-021, FR-024, NFR-004 | AC-020, AC-023, AC-032 | Fails: quote still uses its private preview. |
| 34 | AT-011 | BR-004, FR-020, FR-022, FR-024, NFR-004 | AC-021, AC-023 | Fails: notification subject still uses inline text. |
| 35 | AT-012 | BR-004, FR-020, FR-023, FR-024, NFR-004 | AC-022, AC-023 | Fails: saved row does not exist. |
| 36 | IT-006 / AT-009 | BR-005, FR-006, FR-008, FR-018, FR-019, NFR-001, RULE-009 | AC-018, AC-024, AC-025, AC-026 | Fails: saved providers are not account-guarded. |
| 37 | UT-010 | FR-025, NFR-001 | AC-019, AC-025 | Fails: saved failure projection is absent. |
| 38 | AT-013 | FR-002, FR-007, FR-014, FR-020, FR-024, FR-025, NFR-001, NFR-005, NFR-006 | AC-019, AC-023, AC-025, AC-027 | Fails: quality/privacy/accessibility boundary is incomplete. |
| 39 | REG-001–REG-005 | Detailed regression-row requirement IDs | AC-001, AC-007, AC-011, AC-020, AC-021, AC-023, AC-025, AC-027, AC-032 | Existing behavior may regress after Flutter changes. |
| 40 | REG-006–REG-009 | Detailed regression-row requirement IDs | AC-009, AC-010, AC-011, AC-014, AC-016, AC-025, AC-026, AC-027 | Existing AppView/account behavior may regress. |
| 41 | MAN-001 / MAN-002 | FR-024 | AC-023 | Platform behavior requires manual evidence. |

## Implementation Steps

Each step follows: write one focused failing test; run the smallest command; confirm a meaningful failure; implement the minimum behavior; rerun green; run nearby tests; refactor only while green; record evidence here.

### Implementation-review correction pass (2026-07-22)

The implementation review in `06-implementation-review.md` returned `Changes required`. Address IR-001 through IR-005 in this order, preserving the original approved contracts and strict red-green-refactor loop:

| Order | Review finding | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|---|
| C1 | IR-001 | IT-008 / UT-014 | FR-011, FR-018, RULE-005 | AC-012, AC-013, AC-018 | Fails: confirmed moves and keep-mode folder deletion do not reconcile an already-loaded destination collection. |
| C2 | IR-002 | IT-003 / IT-008 | FR-010, FR-025 | AC-010, AC-019, AC-031 | Fails: Retry after a confirmed folder mutation's failed page-one restart cannot issue a new page-one request. |
| C3 | IR-003 | UT-010 / AT-003 | FR-025 | AC-019 | Fails: production unsave UI presents `ApiCanceled`, and retry presentation is not derived from the bounded failure policy. |
| C4 | IR-004 | UT-010 / AT-013 | NFR-001 | AC-025 | Fails: private saved DTO/page `toString` output contains sentinel values. |
| C5 | IR-005 | IT-008 | FR-009–FR-014, FR-023, NFR-005 | AC-008–AC-013, AC-017, AC-019, AC-022, AC-023, AC-027, AC-028, AC-030, AC-031 | Fails: the page integration suite does not exercise the approved success/failure/action/empty/restoration matrix. |

Correction-pass rules:

- C1 must use the exact server-confirmed folder assignment and `savedAt`; update only provider instances that already exist, and let AppView remain authoritative for folder deletion.
- C2 must distinguish page-one restart retry from cursor-based load-more and retain the confirmed mutation entity while recovery is pending.
- C3 must route actual mutation errors through `SavedPostFailure`; canceled work remains silent and only safely retryable classes expose recovery.
- C4 must remove generated value-bearing stringification from private DTO/page types and verify URI, DID, content, folder, and cursor sentinels are absent.
- C5 expands IT-008 through public widget/provider interfaces and corrects the earlier evidence claim; it does not add new product behavior.
- After C1–C5, rerun generators, focused saved-post suites, canonical Flutter/Go gates, dependency diff, diagnostic sentinel audit, and `git diff --check`.

Correction evidence:

- C1 move red: With folder and Unfiled resources both mounted, the server-confirmed move removed the source row but left the loaded Unfiled destination empty.
- C1 move green: `moveSavedPost` now derives an exact `SavedPostItem` from the server-confirmed presentation and updates only existing source/destination resources. The focused IT-008 case passed without a third list request, preserving the returned `savedAt`.
- C1 folder-delete red: A successful keep-mode folder deletion restarted the folder list but left an already-loaded Unfiled resource empty.
- C1 folder-delete green: Confirmed deletion now invalidates only existing affected folder resources and, for keep mode, existing Unfiled resources for both sorts. AppView refetch remains authoritative and no inactive list or per-row delete is created. Both focused cases and the neighboring folder-page/provider suites passed (5 tests).
- C2 provider red: After confirmed folder creation discarded its unsafe cursor, a failed page-one restart retained the entity and error but had no operation capable of retrying page one.
- C2 provider green: Added one folder-resource `retry` dispatcher: incremental failures with a cursor retry load-more, while restart failures with a discarded cursor retry page one. The focused IT-003 test passed and retained the confirmed created folder across failure and recovery.
- C2 UI red: The chooser's visible Retry still called cursor-only load-more, and the overview rendered no folder restart error control.
- C2 UI green: Chooser and overview Retry controls now use the retry dispatcher. Focused chooser/overview cases and the neighboring folder provider/widget/page suite passed (13 tests).
- C3 unsave red: `ApiCanceled` restored the selected bookmark but the real bookmark UI still showed the generic failure snackbar.
- C3 unsave green: Bookmark and saved-row unsave presentation now derive from `SavedPostFailure`; cancellation remains silent while presentable failures retain localized bounded feedback. Focused AT-003 cases passed.
- C3 folder/save red: Inline create, rename, and save/move controllers discarded the error class, so canceled work produced generic inline errors. Folder-list UI also exposed Retry for a non-retryable validation failure.
- C3 folder/save green: Folder state retains only a bounded `SavedPostFailure` classification, create/rename/delete presentation consults it, save/move derives the same policy from its URI presentation, and folder error controls render feedback/Retry only when `shouldPresent`/`canRetry` permit it. Focused cancellation and retry-policy cases plus neighboring suites passed (14 tests in the combined folder/mutation run, with AT-003 focused separately).
- C4 red: Generated `toString` output exposed private folder identifiers immediately and could recursively expose saved post/author/content and cursor values.
- C4 green: Disabled generated stringify methods for private saved state/item/page/folder DTOs while retaining their approved decode, encode, copy, equality, and custom folder-ID identity behavior. Regenerated mapper output through build runner. The focused sentinel test and all saved model tests passed (5 tests).
- C5 empty/dangling red: The folder screen rendered no empty state, and a chooser retained a folder ID after that exact folder was confirmed deleted.
- C5 empty/dangling green: Added a localized empty-folder presentation and a bounded confirmed-deletion marker in the redacted folder collection state. The chooser clears only the matching deleted selection, so a valid folder temporarily absent from page one is retained. Focused and neighboring provider tests passed.
- C5 collection-error red: Folder and Unfiled screens exposed Retry for every incremental error class and did not render the bounded failure message for non-retryable validation failures.
- C5 collection-error green: Initial/incremental list controls now derive localized presentation and Retry visibility from `SavedPostFailure`. Focused folder and overview cases passed while confirmed rows remained visible.
- C5 expanded IT-008 evidence: Added public-interface cases for loaded move destination reconciliation, move failure selection/source retention, keep/remove folder deletion, successful and canceled row unsave, folder and Unfiled sort/resource behavior, folder pagination above Unfiled, initial and incremental recovery, invalid-cursor page-one restart, pull-to-refresh, folder mutation restart recovery in chooser/overview, empty folder UI, scroll retention, and deleted-selection cleanup. Remove-mode, incremental recovery, invalid-cursor, refresh, initial retry, unsave success/cancellation, move failure, overview sort, and later-folder-page cases were initially green and document existing behavior; the red cases and their fixes are recorded above.
- C5 traceability correction: IT-008 is now a collection of focused page/provider/widget integration cases rather than the previous single scroll test. Supporting UT/AT/IT cases still own exact navigation, typed back-stack restoration, summary ownership, semantics/focus, and account-boundary behavior; the final evidence no longer attributes those behaviors to the scroll test alone.

### Second implementation-review correction pass (2026-07-22)

The implementation re-review in `06-implementation-review.md` accepted IR-002 through IR-004, found IR-001 only partially resolved, and returned two remaining Must-level findings. Address them in this order:

| Order | Review finding | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|---|
| D1 | IR-006 | IT-008 | FR-011, FR-018, RULE-005 | AC-012, AC-018, AC-031 | Fails: a confirmed move updates only the destination using the source screen's sort, leaving an existing destination with another sort stale. |
| D2 | IR-007 | IT-008 | FR-010, NFR-005 | AC-010, AC-027 | Fails: the fully empty overview returns before its refreshable scroll surface and cannot pull to refresh. |

Second-pass rules:

- D1 must reconcile every already-existing affected destination sort from the exact server-confirmed folder assignment and `savedAt`, without fetching inactive resources or changing AppView authority.
- D1 must use a public widget/provider path with different source and destination sorts; the same-sort regression remains as neighboring coverage.
- D2 must keep the localized full-empty state inside an always-scrollable overview refresh surface and prove refresh can replace it with server-returned hierarchy content.
- After D1/D2, rerun the focused collection suite, scoped analysis, canonical Flutter/Go gates, dependency/privacy/diff audits, and correct the final traceability claim again.

Second-pass evidence:

- D1 red: With an Oldest folder resource and Newest Unfiled resource both mounted, the confirmed move removed the source item but the differently sorted destination remained empty.
- D1 green: Move reconciliation now derives one exact item from the server-confirmed assignment and `savedAt`, removes the visible source, and upserts that item into every already-existing destination-sort resource. It never creates or fetches an inactive resource. The focused cross-sort case and all 15 folder-page tests passed; the repository observed only the two initial list requests.
- D2 red: A fully empty overview rendered the localized copy but contained no `RefreshIndicator` or scrollable collection surface, so the refresh gesture and server refetch were unavailable.
- D2 green: The full-empty presentation now uses `SliverFillRemaining` inside the same always-scrollable `CustomScrollView` and `RefreshIndicator` as populated overviews. The focused test pulled to refresh from empty, replaced it with one folder plus one Unfiled item, preserved folders-before-Unfiled ordering, and observed exactly two folder and two Unfiled page-one requests. All 9 overview-page tests passed.

### Step 1: UT-001

- Write failing test: Added one focused case covering saved/unfiled/unsaved decode, protected omission defaults, copy preservation, explicit nullable clear, and malformed field types.
- Run command: `cd app && flutter test test/feed/models/post_test.dart --plain-name "UT-001 saved viewer state"`
- Confirmed failure: Compile failed because `Post` had no `viewerHasSaved` / `viewerSavedFolderId` getters or copy parameters. The first malformed scalar fixture was corrected before further production work because `dart_mappable` deliberately coerces scalars.
- Implement: Added required canonical `viewerHasSaved`, nullable `viewerSavedFolderId`, strict hook type checks, protected-placeholder false/null defaults, and generated mapper/copy support. Updated neighboring canonical fixtures to include the now-required viewer field.
- Green verification: Focused command passed (1 test). `cd app && flutter test test/feed/models/post_test.dart` passed (9 tests).
- Refactor: No production refactor required; `dart format` reported no changes.
- Notes: Generated only `lib/feed/models/post.mapper.dart` through a scoped build filter; no unrelated generated file changed.

### Steps 2–5: UT-002, UT-003, IT-001, IT-002

- UT-002 red: The focused test failed to compile because the saved state/item/page and folder/page model files and mappers did not exist.
- UT-002 green: Added camelCase `dart_mappable` models for `SavedPostState`, `SavedPostItem`, `SavedPostPage`, `SavedPostFolder`, and `SavedPostFolderPage`, including nullable assignment/cursor copy clearing. The focused test passed (1 test); `flutter test test/saved_posts/models test/feed/models/post_test.dart` passed (10 tests).
- UT-002 refactor: Formatted sources/tests; only the two scoped mapper outputs were generated.
- UT-003 red: The focused test failed to compile because server-parity folder-name validation/error types were absent; existing generated equality also did not express opaque-ID identity.
- UT-003 green: Added trim, Unicode-rune-count, slash/backslash, and Unicode-control validation; introduced bounded validation error kinds; made folder equality/hash depend only on opaque ID while preserving generated decode/encode/copy/string output. The focused test passed (1 test); `flutter test test/saved_posts/models` passed (2 tests).
- UT-003 refactor: Formatted sources/tests; regenerated only `saved_post_folder.mapper.dart`.
- IT-001 red: The focused Dio contract test failed to compile because `SavedPostApiClient`, saved list scope, and sort types did not exist.
- IT-001 green: Added redacted unfiled/folder scopes, newest/oldest sort, and typed methods for save/move, unsave, scoped lists, folder paging/create/rename/delete. Tests prove explicit nullable `folderId`, opaque query round-trip, no-body deletes, absent destructive query for keep mode, and exactly one `deleteSaves=true` request for remove mode. The focused test passed (1 test); all saved model/data tests passed (3 tests).
- IT-001 refactor: Formatted the new data/model/test files; no generated output changed.
- IT-002 red: The focused repository test failed to compile because the typed repository, API abstraction, and production delegation implementation did not exist.
- IT-002 green: Added `SavedPostApi`, `SavedPostRepository`, and `ApiSavedPostRepository`; a recording fake verifies exact argument/result forwarding for every operation and unchanged error propagation. The focused test passed (1 test); all saved model/data tests passed (4 tests).
- IT-002 refactor: Formatted the new interfaces, implementation, and test; no behavior change was needed.

### Steps 6–12: UT-004, UT-013, IT-004, IT-005, AT-001, AT-002, AT-003

- UT-004 red: The focused provider test failed to compile because account/URI keys, presentation/map state, repository provider, account notifier, and selector did not exist.
- UT-004 green: Added redacted immutable keys/state; account-keyed URI maps; public `whenData` projection that relies on Riverpod's automatic previous-state preservation; seed-if-absent; confirmation-driven save/move; immediate optimistic unsave; duplicate guards; exact rollback; revision checks; and per-account isolation. After correcting two test setup issues (an explicit non-transitive model import and one microtask before the async repository call count), the focused test passed (1 test); `flutter test test/saved_posts` passed (5 tests).
- UT-004 refactor: Added immutability annotations, removed an unnecessary import, formatted files, and ran `dart analyze lib/saved_posts` with no issues.
- UT-013 red: The focused dialog-controller test failed to compile because the redacted dialog key/state and keyed controller did not exist.
- UT-013 green: Added account/URI/initial-folder dialog identity, default/current folder selection, ID-only selection, normalized inline folder creation, independent create error state, editable recovery, cancellation without a save mutation, and bounded redacted diagnostics. The focused test passed (1 test); `flutter test test/saved_posts` passed (6 tests).
- UT-013 refactor: Kept the auto-disposed provider listened to for the full async lifecycle in the test, simplified validation branching, formatted the files, and ran `dart analyze lib/saved_posts` with no issues.
- IT-004 red: The focused mutation integration test failed to compile because dialog save/move confirmation and confirmed state did not exist.
- IT-004 green: Added one-shot save/move confirmation through the account state notifier. Duplicate submissions are ignored, the dialog remains pending until completion, success publishes server-returned placement/timestamp state, and failure retains the attempted selection while the shared state keeps its prior placement. The focused test passed (1 test).
- IT-004 refactor: Centralized save and move completion handling in one private confirmation method and formatted the implementation/test.
- IT-005 red: The new multi-consumer test initially observed a stale subscription callback immediately after a synchronous derived-provider update. A direct selector read already held the correct optimistic value, identifying a test scheduling boundary rather than a production failure.
- IT-005 green: After flushing the derived-provider subscription microtask at each observation boundary, two consumers saw the same optimistic unsave, exact rollback, successful commit, duplicate suppression, and stale-post seed protection. The focused test passed (1 test); the existing account reducer required no production change.
- IT-005 refactor: Reused the controlled mutation repository and retained explicit immediate-read coverage so synchronous optimistic state and deferred listener delivery are both documented.
- AT-001 red: Four focused `PostCard` scenarios failed because the bookmark and chooser widget did not exist; the saved model also had no full-post UI consumer.
- AT-001 green: Added a fixed 48-pixel account-scoped bookmark immediately before overflow, localized selected semantics, seed-if-absent projection, unsaved chooser launch without mutation, and one-shot optimistic unsave without confirmation or Undo. Protected cards still return before the bookmark builds. All four focused `AT-001` widget scenarios passed.
- AT-001 refactor: Extracted the bookmark and chooser shell into saved-post widgets, generated localization output, formatted the touched files, and kept the full `PostCard` action row otherwise unchanged.
- AT-002 red: The first chooser widget scenario rendered only `No folder`; no folder resource, opaque paging, distinct ID options, loading/error UI, or inline creation controls existed.
- AT-002 green: Added an account-keyed folder resource and bounded chooser UI. It pages using the opaque cursor, deduplicates solely by ID, preserves duplicate/case-variant names, leaves `No folder` savable on initial failure, confirms exactly once while busy, closes silently on server confirmation, and supports independent normalized inline creation with safe editable failure. All four focused `AT-002` scenarios passed.
- AT-002 refactor: Kept folder mutation in the shared folder notifier, retained it for the dialog-controller lifecycle, generated Riverpod/localization output, and restored the UT-013 controller suite after identifying auto-dispose during an unlistened async create.
- AT-003 red: The failed-unsave widget scenario restored the bookmark through existing reducer behavior but exposed no retryable user feedback.
- AT-003 green: The bookmark now awaits its one-shot command only to project completion, showing localized failure guidance without raw error/folder data and no success snackbar or Undo. The focused widget test passed; the exact multi-consumer rollback remains covered by IT-005.
- AT-003 refactor: Kept success silent and error presentation in the button surface while mutation truth remains solely in the account URI notifier.

### Steps 13–24: UT-005, IT-003, UT-006, UT-007, UT-014, UT-011, AT-004, AT-005, AT-006, AT-007, IT-007, IT-008

- UT-005 red: The focused pagination test failed to compile because saved-list keys/state, canonical-URI merging, cursor classification, and folder mutation reconciliation helpers did not exist.
- UT-005 green: Added redacted account/scope/sort keys; immutable saved/folder collection states; stable URI/ID deduplication; opaque cursor retention; incremental error retention; sealed `invalid_cursor` detection; page-one restart projection; and retained confirmed folder entities across partial cursor restarts. The focused test passed (1 test).
- UT-005 refactor: Kept retained mutation entities separate from server-ordered page items and exposed a combined chooser view without parsing or diagnosing private identifiers.
- IT-003 red: The provider integration test failed to compile because shared folder rename/delete commands did not exist.
- IT-003 green: Added independent account/scope/sort saved-list families and one account folder family with guarded load-more, safe invalid-cursor page-one restart, mutation-triggered folder restart, ID/URI deduplication, and confirmed create/rename/delete retention. The focused test passed (1 test), including a controlled duplicate load and independent newest/oldest/folder resources.
- IT-003 refactor: Centralized restart behavior within each notifier, reused immutable merge projections, and changed the invalid-cursor fixture to complete its error only after a listener attached.
- UT-006 red: The overview projection test failed to compile because no common folders/Unfiled projection existed.
- UT-006 green: Added an immutable redacted overview projection that preserves server folder order, excludes foldered rows, sorts Unfiled solely by `savedAt`, hides an empty Unfiled section, and reports full empty only when both sections are empty. The focused test passed (1 test).
- UT-006 refactor: Corrected the test to assert immutable list equality rather than source-list identity.
- UT-007 red: The exact saved-post destination test failed to compile because no destination helper existed.
- UT-007 green: Added a redacted destination value that opens top-level posts directly and comments/nested replies through their root with exact-post focus. The focused test and unchanged notification destination suite passed (4 tests total).
- UT-014 red: The focused chronology-reconciliation test failed to compile because the account URI state had no server-item or folder-delete reconciliation methods.
- UT-014 green: Added canonical saved-item reconciliation on every loaded page and confirmed folder-delete reconciliation on the shared account URI state. Move and keep-saves mode preserve the exact server `savedAt`; remove-saves mode clears saved state and chronology. The focused test passed (1 test), and the neighboring saved provider/page/row run passed (17 tests).
- UT-014 refactor: Centralized the immutable folder-deletion projection in the redacted account map and made stale async mutations fail their revision check after a confirmed delete.
- UT-011 red: The route test failed to compile because canonical saved overview/folder routes and redacted route data did not exist.
- UT-011 green: Added `/profile/settings/saved` and static `/profile/settings/saved/folder` typed routes; opaque folder data travels only in nullable redacted `$extra`. The focused route test passed (1 test).
- UT-011 refactor: Generated router output and cleared route/saved static analysis.
- AT-004 red: The overview test failed to compile because the page did not exist; the Settings test then failed because no Saved posts entry existed.
- AT-004 green: Added a stateful overview with retained scroll identity, folders before Unfiled, saved-time sorting, full-empty behavior, refresh/error shells, and typed folder navigation. Added the Settings-only entry and removed the obsolete profile Saved route/tab/content so owner and visitor profiles share five non-private tabs. The focused Settings/profile/route/overview tests passed (5 tests).
- AT-004 refactor: Localized saved collection copy, regenerated localization/router output, and ran scoped analysis with no issues.
- AT-005 red: The folder-page test failed to compile because the independently keyed folder screen did not exist.
- AT-005 green: Added folder-scope paging and Newest/Oldest resources, Refresh/Retry/Load-more UI, and equivalent Unfiled incremental controls. Provider invalid-cursor restart retains confirmed rows and restarts only the affected scope/sort. The focused folder page plus collection/provider suite passed.
- AT-005 refactor: Localized paging copy and reused the same `SavedPosts` notifier family for overview and folder scopes.
- AT-006 red: The saved-row test failed to compile because no parent-owned compact row existed.
- AT-006 green: Added a compact saved row with parent-owned Open/Move/Unsave callbacks and no bookmark/engagement logic. Both pages derive exact root/focus routes, use the common move chooser, reconcile source lists after confirmed moves, and use the account overlay for immediate unsave/rollback. The focused row test passed (1 test).
- AT-006 refactor: Centralized page row actions while leaving compact content surface-agnostic for the later `PostSummary` extraction.
- AT-007 red: The overview create test failed because its Add folder action was inert; the folder action test then failed because rename/delete UI did not exist.
- AT-007 green: Added normalized create/rename dialogs, adaptive folder actions, confirmed title reconciliation, and an explicit Cancel/Keep saved posts/Delete saved posts delete dialog that sends one boolean-scoped command. Focused create and rename/delete tests passed (2 tests).
- AT-007 refactor: Reused shared folder notifier mutation/restart behavior and kept destructive dialog focus on Cancel.
- IT-007 green (unexpected initial green): The new typed-stack integration test passed without another production change after UT-011/AT-004 routing work: Settings → overview → static generic folder, then two back operations, preserved matched locations and screens.
- IT-008 partial evidence (superseded by the correction pass): The original collection screen test confirmed only that overview scroll offset survives a confirmed folder mutation while scrolled. It did not by itself prove the approved interaction matrix. The correction-pass evidence above now supplies focused page/provider/widget cases for the missing success, failure, retry, empty-state, mutation-reconciliation, and restoration paths.

### Steps 25–31: UT-009, UT-012, IT-009, IT-010, IT-011, IT-012, AT-008

- UT-009 red: The strict delete-query test failed to compile because the mode enum and parser did not exist.
- UT-009 green: Added strict absent/false preserve mode and true remove mode parsing; empty, mixed-case, repeated, invalid, and unknown shapes return `validation_failed`. The focused parser test passed (8 cases).
- UT-009 refactor: Reused existing single-value query parsing and standard field errors.
- UT-012 green (existing coverage): The operation-specific error test already proved missing/cross-owner deletes map indistinguishably to 204 while unrelated store failures remain available for bounded 500 mapping. No production change was needed.
- IT-009 red: The handler-mode test failed to compile after the recording fake adopted the planned mode-aware store signature.
- IT-009 green: Updated the handler/store interface; authenticated deletes parse before storage, record preserve/remove mode, and reject invalid/unknown queries with the standard 422 envelope without a store call. The focused handler test passed.
- IT-010 red: The real-Postgres mode test showed remove mode still unfiled the save because storage ignored the new mode.
- IT-010 green: Added an owner-scoped transaction that deletes assigned saves only in remove mode and always deletes the folder; preserve, remove, repeat, missing, malformed, and cross-owner cases passed under `-race` with real Postgres.
- IT-011 red: Added a deterministic trigger failure after the remove-mode save-row statement.
- IT-011 green: The real-Postgres test proved both the folder and its save rows remain after the later folder delete fails; the transaction rolls back atomically.
- IT-012 green: Extended the real-Postgres privacy regression through folder assignment and remove-mode deletion; the author/public engagement projection remains unchanged and only the owner's private save disappears.
- AT-008 green: `TEST_DATABASE_URL=... go test -race ./internal/api ./internal/routes -run 'SavedPost|SavedPostFolder'` passed, composing strict handler modes, transactional storage, rollback, privacy, and unchanged saved APIs/routes.

### Steps 32–38: UT-008, AT-010, AT-011, AT-012, IT-006 / AT-009, UT-010, AT-013

- UT-008 red: The focused shared-widget test failed to compile because `PostSummary`, its immutable redacted data projection, and visible/policy adapters did not exist.
- UT-008 green: Added an action-free bounded summary with post/quote adapters, optional author/text/project/first-image data, visible/muted/hidden/unavailable states, and parent-owned post/author/reveal callbacks. The focused test passed (1 test) and proves no bookmark or engagement controls are rendered.
- UT-008 refactor: Kept surface-specific chrome, mutation controls, and navigation outside the shared widget; formatted the implementation and test.
- AT-010 red: The focused quote test found zero `PostSummary` instances because `PostCard` still rendered its private quote preview implementation.
- AT-010 green: Replaced the private quote content with `PostSummaryData.fromQuoteView` and `PostSummary` while retaining quote card chrome, cached first-image behavior, project title, author/post taps, bounded text, all hidden/muted/blocked/unavailable copy, and reveal policy. The focused test passed (1 test).
- AT-010 refactor: Moved the existing compact author/image primitives into the shared widget and removed the now-unused private quote helpers. The complete `post_summary_test.dart` plus `post_card_test.dart` suite passed (55 tests).
- AT-011 red: After bringing the canonical notification fixture forward with required saved viewer state, the focused test found zero `PostSummary` instances across like, repost, reply, mention, and quote rows.
- AT-011 green: Added a text-only notification subject adapter and replaced all five post-bearing inline subtitles with `PostSummary`, retaining the parent-owned actor/action title, avatar, category treatment, timestamp, filtering, follow control, row tap, and exact destination/focus behavior. The focused test passed (1 test).
- AT-011 refactor: Added surface-supplied summary padding so notification rows keep their compact existing height while quote chrome retains inset content. The complete notification page suite passed (13 tests), including destination and filtering regressions.
- AT-012 red: The focused saved-row test found no `PostSummary` or parent-owned saved timestamp because the row still used a text-only `ListTile`.
- AT-012 green: Added a saved-item adapter that omits post chronology, composed the row with `PostSummary`, and rendered `savedAt` alongside parent-owned Move/Unsave controls. The focused test passed (1 test).
- AT-012 refactor: Kept exact open navigation on the summary and all saved-specific metadata/mutations as siblings. Shared-summary, saved-row, and saved-page suites passed (9 tests).
- IT-006 / AT-009 red: A delayed account-race test showed Alice's late list page appending after the account boundary; the saved feature was absent from the central account invalidator and async commands checked only `ref.mounted`, which remains true when a dependency rebuilds the notifier.
- IT-006 / AT-009 green: Registered every saved repository/state/list/folder/dialog family at the account boundary, added one shared generation seam, and required list, dialog load/create, save, move, unsave, rename, and delete completions to match their captured generation. The focused delayed-operation test passed and proves a post-switch Bob state remains untouched.
- IT-006 / AT-009 refactor: Centralized generation capture/current checks and retained account-keyed repository ownership. All saved provider tests plus the existing account-boundary regression passed after updating its canonical Post fixture (10 tests total across the combined run and focused rerun).
- UT-010 red: The focused projection test failed to compile because no bounded saved-operation failure model existed. A second focused endpoint test mapped private saved routes to `appview.unknown` because their allowlisted categories were absent.
- UT-010 green: Added redacted error kind/operation/retry/presentation projection using only existing localized copy; canceled work stays silent and validation is not blindly retried. Added bounded saved-post, folder-list, and folder-detail endpoint categories without dynamic IDs. Model, interceptor, and Sentry sentinel tests passed (13 tests combined).
- UT-010 refactor: Reused the existing API exception taxonomy and Sentry allowlist; no raw exception, folder, URI, owner-target, cursor, or server message enters the projection.
- AT-013 red: The narrow 320-pixel/2×-text saved-row test reported an 80-pixel horizontal overflow. A destructive-dialog semantics assertion then showed the remove-saves action had no explicit hint.
- AT-013 green: Replaced the fixed metadata/action row with a wrapping layout that keeps both 48-pixel actions reachable and added an explicit localized destructive semantic node while retaining Cancel autofocus. The saved-row and folder-page quality suites passed (5 tests).
- AT-013 refactor: Reused existing localized destructive copy, kept summary text bounded, and verified the shared summary remains action-free under semantics.
- REG-001–REG-005 green: The curated full-post action/layout, canonical post decode, profile/tab privacy, Settings ownership, quote policy, notification context/destination, and protected-placeholder suites passed (104 tests). Removed the obsolete Saved profile page and its stale direct widget test as required by the approved surface removal.
- REG-006–REG-008 green: `TEST_DATABASE_URL=... go test -race ./internal/api ./internal/routes -run 'SavedPost|SavedPostFolder'` passed against real Postgres. The suites cover absent-query unfile behavior, response/cursor/policy/reply stability, both folder delete modes, rollback, privacy, unchanged indexed/public state, and no PDS collaboration.
- REG-009 green: `flutter test test/auth/providers/account_boundary_provider_test.dart test/router/router_redirect_test.dart test/router/saved_posts_route_test.dart` passed (16 tests), retaining account-specific cancellation, switching, callback, redirect, and saved-route behavior.

### Steps 39–41: REG-001–REG-009, MAN-001, MAN-002

- Automated regression evidence: REG-001–REG-009 passed in the focused runs above and again within the full Flutter/Go gates.
- MAN-001 pending: This environment has no attached iOS/VoiceOver or Android/TalkBack target and no hardware-keyboard accessibility session, so native announcement/traversal evidence cannot be claimed. Widget semantics and Cancel autofocus are covered automatically.
- MAN-002 pending: This environment has no real narrow device/simulator visual session. The 320-pixel, 2×-text widget coverage is green, but native font/platform rendering still requires the approved manual check.

## Final Verification

- Generate mappers/providers/router output: `cd app && dart run build_runner build --delete-conflicting-outputs` passed; generated outputs are current. The tool reported its existing analyzer-language warning and that the now-removed option was ignored, but completed successfully.
- Generate localization output: `cd app && flutter gen-l10n` passed using `l10n.yaml`.
- `just app-analyze`: Passed with no issues.
- `just app-test`: Passed (977 tests after UT-014 in the original implementation pass).
- `just fmt`: Passed (`gofmt -w .` and `go vet ./...`).
- `just test`: Passed with `-race` across every AppView package against compose Postgres; database-backed tests did not skip.
- Focused real-Postgres `-race` evidence with `TEST_DATABASE_URL`: Passed for `./internal/api ./internal/routes -run 'SavedPost|SavedPostFolder'`.
- `git diff --check -- app appview docs/changes/2026-07-21-flutter-saved-posts`: Passed.
- Dependency diff (`app/pubspec.yaml`, `app/pubspec.lock`, `appview/go.mod`, `appview/go.sum`): Empty; no runtime or test dependency changed.
- Original traceability audit (superseded by the correction pass): All approved automated ID sets had at least one passing reference, but implementation review correctly found that the original IT-008 evidence did not exercise its full approved interaction matrix. The correction-pass traceability and verification below replace that claim. MAN-001/MAN-002 remain explicitly pending for a real device/assistive-technology session.

### Post-review correction verification (2026-07-22)

- Regenerated private DTO mappers with `cd app && dart run build_runner build --delete-conflicting-outputs`; output completed successfully and removed generated value-bearing stringification for saved state/item/page/folder DTOs.
- `cd app && flutter gen-l10n`: Passed using `l10n.yaml`.
- Focused saved-post and PostCard regression run: Passed (107 tests).
- `cd app && dart analyze lib/saved_posts test/saved_posts`: Passed with no issues.
- `just app-analyze`: Passed with no issues.
- `just app-test`: Passed (1,002 tests).
- `just fmt`: Passed (`gofmt -w .` and `go vet ./...`).
- `just test`: Passed with `-race` across every AppView package against compose Postgres; database-backed tests did not skip.
- Dependency diff (`app/pubspec.yaml`, `app/pubspec.lock`, `appview/go.mod`, `appview/go.sum`): Empty; no runtime or test dependency changed.
- Private DTO sentinel test: Passed for owner DID, post URI/content, folder ID/name, and cursor. Generated saved-model mapper sources contain no `stringifyValue` implementation.
- `git diff --check -- app appview docs/changes/2026-07-21-flutter-saved-posts`: Passed.
- First correction traceability audit (superseded by the second correction pass): C1–C5 added meaningful red/green evidence, but implementation re-review found the cross-sort destination and fully empty overview refresh cases still missing from IT-008. D1/D2 and the verification below replace the earlier completeness claim. Supporting UT/AT/IT cases retain ownership of exact routing, shared-summary, accessibility, and account-boundary contracts. MAN-001/MAN-002 remain the only external manual gaps.

### Second correction verification (2026-07-22)

- D1 focused cross-sort move: Passed, followed by all 15 folder-page tests.
- D2 focused fully empty overview refresh: Passed, followed by all 9 overview-page tests.
- `cd app && flutter test test/saved_posts`: Passed (54 tests).
- `cd app && dart analyze lib/saved_posts test/saved_posts`: Passed with no issues.
- `cd app && flutter gen-l10n`: Passed using `l10n.yaml`; no localization source change was required.
- `just app-analyze`: Passed with no issues.
- `just app-test`: Passed (1,004 tests).
- `just fmt`: Passed (`gofmt -w .` and `go vet ./...`).
- `just test`: Passed with `-race` across every AppView package against compose Postgres; database-backed tests did not skip.
- Dependency diff (`app/pubspec.yaml`, `app/pubspec.lock`, `appview/go.mod`, `appview/go.sum`): Empty.
- Private generated-stringify audit: Saved model mapper sources contain no `stringifyValue`; the 54-test saved suite reran the full private DTO sentinel test.
- `git diff --check -- app appview docs/changes/2026-07-21-flutter-saved-posts`: Passed for tracked changes; a separate trailing-whitespace scan covers the feature's new/untracked files.
- Final second-pass traceability audit: IR-006 and IR-007 each have meaningful red evidence and passing public-interface regressions. Together with C1–C5 and the supporting UT/AT/IT suites, IT-008 now covers the approved collection interaction matrix. MAN-001/MAN-002 remain the only external manual gaps; implementation re-review remains pending.

### User-feedback convention pass (2026-07-22)

- Read and applied `app/.agents/rules/riverpod.md`. Removed every saved-feature use of Riverpod's internal `copyWithPrevious` API and the associated `invalid_use_of_internal_member` analyzer suppression. `projectSavedPostPresentation` now maps with the public `AsyncValue.whenData` API and leaves previous-state preservation to Riverpod.
- Replaced hand-written nullable-sentinel `copyWith` implementations on `SavedPostPresentation`, `SavePostDialogState`, `SavedPostFolderListState`, and `SavedPostListState` with DartMappable-generated copy/equality support. Generated stringification remains disabled and each class retains a redacted custom `toString`. Wire DTOs continue to own generated JSON serialization; transient provider state containing private values and arbitrary `Object` failures is deliberately not made serializable.
- Regenerated Riverpod and DartMappable outputs with `cd app && flutter packages pub run build_runner build --delete-conflicting-outputs`. The installed build runner reported that the removed option was ignored and completed successfully.
- Forbidden-API audit: `rg -n "copyWithPrevious|invalid_use_of_internal_member" app/lib app/test` returned no matches.
- `cd app && flutter test test/saved_posts`: Passed (54 tests).
- `cd app && flutter analyze lib/saved_posts test/saved_posts`: Passed with no issues.
- `just app-analyze`: Passed with no issues.
- `cd app && flutter test`: Passed (1,004 tests).
- `just fmt`: Passed (`gofmt -w .` and `go vet ./...`). The existing second-correction real-Postgres `just test` evidence remains current because this feedback pass changed only Flutter source/tests/generated output and this implementation artifact.

## Completion Checklist

- [x] All Must requirements covered by tests or documented gaps.
- [x] All planned automated Must tests passing.
- [x] Relevant regression tests passing.
- [x] Real-Postgres integration tests ran and did not skip.
- [x] Manual checks completed or explicitly recorded as blocked with reason.
- [x] No unlinked behavior implemented.
- [x] No private values added to diagnostics.
- [x] No unapproved runtime dependency added.
- [x] Generated files updated through generators.
- [x] Docs updated and this file read back.
- [x] Implementation review completed (the first correction re-review returned IR-006/IR-007; both are implemented above and await final re-review).

## Deviations And Gaps

- The original `05-implementation-plan.md` test-order table accidentally omitted approved case UT-014 even though `02-acceptance-tests.md` and `04-coding-plan.md` required it. The final traceability audit caught the omission; UT-014 then received its own red-green-refactor cycle and is now recorded in the corrected order and evidence above.
- Making `viewerHasSaved` required for canonical visible-post JSON exposed older test fixtures that predated the wire field. Those fixtures now provide the canonical boolean; protected placeholders remain the only intentional omission path.
- The full Flutter gate exposed six existing auto-disposed provider tests that read `.future` without retaining a listener. Their tests now keep the provider alive for the async assertion; production behavior was unchanged.
- MAN-001 and MAN-002 require real platform/device evidence and remain pending for the reasons recorded above. They are the only external manual verification gaps; implementation re-review remains pending.
