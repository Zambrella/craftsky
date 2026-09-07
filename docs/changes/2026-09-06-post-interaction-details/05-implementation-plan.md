# TDD Implementation Plan: Post Interaction Details

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved with notes`)
- Coding plan: `04-coding-plan.md`

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update a failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability updated.
- Do not change lexicons or add a migration. If IT-008 demonstrates an index deficiency, stop for separately reviewed migration approval.
- Do not commit or push unless explicitly requested.

## Test Order

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | IT-001 | FR-003, RULE-001, RULE-003 | AC-003, AC-010 | Fails: liker-list store contract absent |
| 2 | IT-002, UT-003 | BR-002, FR-004, FR-010, RULE-001, RULE-002, RULE-003 | AC-003, AC-004, AC-010, AC-011 | Fails: repost membership path absent |
| 3 | IT-003 | BR-002, FR-006, RULE-001, RULE-002, RULE-003 | AC-004, AC-010, AC-011 | Fails: quote-list path absent |
| 4 | UT-002, UT-008, IT-010 | FR-003, FR-004, FR-006, FR-009, NFR-002 | AC-010, AC-016 | Fails: bound cursor/traversal absent |
| 5 | IT-004, IT-011, IT-012 | FR-005, FR-009, FR-010 | AC-012, AC-016, AC-017 | Fails: counts use narrower policy |
| 6 | IT-013 | NFR-001 | AC-019 | Fails: bounded hydration unproven |
| 7 | IT-008 | NFR-001 | AC-019 | Fails: list query plans absent |
| 8 | IT-005, IT-006 | FR-008, FR-009, NFR-002 | AC-014, AC-015, AC-016 | Fails: GET handlers absent |
| 9 | IT-014 | NFR-005 | AC-022 | Fails: list observability absent |
| 10 | IT-007, REG-004 | FR-002, FR-008, NFR-004 | AC-009, AC-014, AC-021 | Fails: routes and policies absent |
| 11 | UT-007 | FR-005, FR-008 | AC-006, AC-015 | Fails: endpoint-shape reuse unpinned |
| 12 | UT-005, IT-009, REG-004 | FR-008, FR-009, NFR-004 | AC-014, AC-015, AC-016, AC-021 | Fails: Flutter methods absent |
| 13 | UT-004 | FR-007 | AC-008, AC-013 | Fails: providers/state absent |
| 14 | AT-003, AT-006 | BR-001, BR-002, BR-003, FR-003, FR-004, FR-005, FR-007 | AC-003, AC-006, AC-008, AC-013 | Fails: account destination absent |
| 15 | AT-004 | BR-001, BR-002, BR-003, FR-006 | AC-004, AC-007 | Fails: quote destination absent |
| 16 | UT-001 | FR-001, RULE-004 | AC-001 | Fails: summary widget absent |
| 17 | AT-001, AT-002, AT-008 | BR-001, BR-002, FR-001, FR-002, FR-011, FR-012, RULE-004 | AC-001, AC-002, AC-005, AC-018 | Fails: thread summary absent |
| 18 | UT-006, AT-005 | BR-004, FR-008, FR-013, RULE-004, RULE-005 | AC-024 | Fails: response liker menu absent |
| 19 | AT-009 | FR-002, NFR-004 | AC-009, AC-021 | Fails: typed routes absent |
| 20 | AT-007 | NFR-003 | AC-020 | Fails: semantics/layout unproven |
| 21 | REG-001, REG-002 | FR-012, FR-013, RULE-005 | AC-018, AC-024 | Fails until shared-card scope is verified |
| 22 | REG-003, REG-005 | FR-006, FR-010, RULE-002, RULE-003 | AC-004, AC-007, AC-011, AC-017 | Fails until quote behavior is verified |
| 23 | REG-006 | FR-012, NFR-003 | AC-018, AC-020 | Fails until thread regressions are verified |
| 24 | MAN-001 | NFR-003 | AC-020 | Pending manual responsive smoke |
| 25 | MAN-002 | NFR-003 | AC-020 | Pending manual assistive-technology smoke |

## Implementation Steps

### Step 1: IT-001

- Write failing test: `TestPostStore_ListPostInteractionAccounts_Likes` for roots, comments, nested replies, active/deleted records, tied ordering, total, and cursor exhaustion.
- Run command: `go test ./internal/api -run 'TestPostStore_ListPostInteractionAccounts_Likes'` from `appview/`.
- Confirmed failure: Build failed because `PostInteractionLikes`, `ResolveInteractionTarget`, and `ListPostInteractionAccounts` did not exist.
- Implement: Added the typed interaction target, viewer-aware target resolution, set-wise active-liker account query, stable seek pagination, exact total, cached-handle hydration, and `limit + 1` exhaustion handling.
- Run command: Passed (`go test ./internal/api -run 'TestPostStore_ListPostInteractionAccounts_Likes'`).
- Refactor: Kept interaction rows separate from follow-specific `ProfileAccountRow` ordering fields; moved total calculation before the seek filter so every page retains the authoritative total.
- Notes: Start with typed identifiers and an interaction-specific account row. Use `limit + 1`; do not copy follow-list cursor behavior.

### Steps 2-10: Remaining AppView Tests

- Write failing test: One focused test ID/group at a time in the coding-plan order.
- Run command: Smallest matching `go test ./internal/api` or `go test ./internal/routes` filter.
- Confirmed failure: Recorded per step below.
- Implement: Completed per step below.
- Run command: Focused tests and nearby regressions passed per step below.
- Refactor: Performed only after each focused test was green.
- Notes: Preserve shared eligibility between lists/counts, bound cursors to kind/target, and avoid per-item hydration. IT-008 cannot introduce a migration without approval.

- Step 2 red: `PostInteractionReposts` was undefined.
- Step 2 green: Added the closed Reposts kind and selected only the fixed `craftsky_reposts` table. `TestPostStore_ListPostInteractionAccounts_Reposts` proves active/deleted membership, quote-only exclusion, dual-actor independence, order, total, and exhaustion. Both focused Reposts and nearby Likes tests pass.
- Step 3 red: `PostInteractionQuotes` and `ListQuotePosts` were undefined.
- Step 3 green: Added the closed Quotes kind and root-only quote-row list using `postSelectColumns`, existing moderation/block predicates, newest-first order, repeated-author preservation, and `limit + 1` exhaustion. Focused Quotes and nearby Likes/Reposts tests pass.
- Step 4 red: Generic seek cursors were accepted across interaction kinds and target posts; tied traversal and exact-size exhaustion already passed.
- Step 4 green: Added strict `PostInteractionCursor` encoding/decoding with kind, target, `createdAt`, and URI validation, then bound all three list paths to exact kind/target. UT-002, UT-008, IT-010, and IT-001 through IT-003 pass against PostgreSQL.
- Step 5 red: Account policy returned five rows instead of three, Reposts accepted a reply target, and quote list/count APIs lacked content-language input.
- Step 5 green: Centralized viewer-aware account and quote eligibility for lists and engagement summaries, added current-member/terminal/block/moderation/language filtering while retaining mute/warnings, protected unavailable targets and unsupported reply shares, and updated engagement-summary callers. Focused tests and `go test ./internal/api ./internal/routes` pass.
- Step 6 red: The bounded quote-response hydration entry point did not exist.
- Step 6 green: Added page-bounded quote response hydration through batch engagement summaries, batch relationship states, distinct-author handle resolution, and one page-level quote-preview attachment. IT-013 proves full-page query counts remain bounded and account hydration makes no external identity calls; the full internal API suite passes.

- Step 7 red: Query text was not exposed for focused EXPLAIN verification.
- Step 7 green: Added representative Likes/Reposts subject and Quotes `quote_uri` plan tests. Existing indexes support the required scans and joins, so no migration is needed; the full internal API suite passes.
- Step 8 red: GET interaction-list handlers and the typed identity-unavailable error did not exist.
- Step 8 green: Added the three handler contracts with typed identifier validation, bounded limits, cursor/target/identity error mapping, bare camelCase account/post pages, and omitted exhausted cursors. Focused IT-005/IT-006 and the internal API suite pass.
- Step 9 red: None of the Likes/Reposts/Quotes handler cases emitted bounded operation logs.
- Step 9 green: Added sanitized handler result logs and existing `ObserveDB` instrumentation. Handler and real-PostgreSQL store observability tests pass using the active compose database; canary DIDs, handles, URIs, cursors, and items are absent.
- Step 10 red: All three GET registrations, route policies, inventory entries, and read-only constructor assertions were absent.
- Step 10 green: Registered method-distinct Likes/Reposts/Quotes reads as current-member, read-rate, no-body routes; middleware rejection, valid access, inventory, architecture, and no-PDS-effects coverage pass with `go test ./internal/api ./internal/routes`.

### Steps 11-23: Flutter And Regression Tests

- Write failing test: One focused test ID/group at a time in the coding-plan order.
- Run command: Smallest matching `flutter test` target from `app/`.
- Confirmed failure: Recorded per step below.
- Implement: Completed per step below.
- Run command: Focused tests and nearby regressions passed per step below.
- Refactor: Performed only after each focused test was green.
- Notes: Reuse existing models and public UI interfaces. Generate Riverpod, router, and localization artifacts only after their corresponding red tests.

### Steps 24-25: Manual Gates

- MAN-001: Blocked. No seeded app instance from this worktree is connected; automated AT-007 covers the four approved viewport/text-scale combinations, but that is not a substitute for visual inspection.
- MAN-002: Blocked. No mobile screen-reader/device session is available; automated semantics and keyboard-route coverage pass, but real spoken phrasing and platform focus restoration remain a handoff check.

## Execution Notes

- 2026-09-06: Loaded all workflow documents. No blocking gaps or questions were present.
- 2026-09-06: Existing worktree was clean at stage start. Commits are not enabled.
- 2026-09-06: Inspection found no prior `05-implementation-plan.md` and no existing interaction-list implementation.
- 2026-09-06: `ProfileAccountRow` is follow-specific; interaction pages need a separate ordering row contract with set-wise cached-handle hydration.
- 2026-09-06: `parseLimit` currently defaults malformed values while IT-006 mentions invalid-limit failures. Follow the authoritative coding plan's instruction to reuse bounded `parseLimit` unless the focused handler test exposes a direct contract conflict.
- 2026-09-06: IT-001 completed red-green-refactor. Focused test passes for roots, comments, and nested replies.

## Flutter And Regression Outcomes

- Step 11 initial state: UT-007 passed immediately because existing `ProfileAccountPage` and `PostPage` already decode the approved endpoint shapes. Added focused contract coverage only; no production model or interaction DTO was needed.
- Step 12 red: Flutter API, repository, and fake contracts lacked all three interaction-list methods.
- Step 12 green: Added exact encoded AppView GET paths, optional pagination parameters, existing model decoding/error mapping, repository delegation, and programmable fake callbacks. All 50 focused data tests pass with clean analysis.
- Step 13 red: Interaction-list state and providers did not exist.
- Step 13 green: Added immutable account/quote state and generated notifier families with retained-data pagination errors, exact-cursor retry, invalid-cursor restart, DID/URI deduplication, refresh generation and active-account fencing, language watching, and quote replace/remove. All 12 UT-004 tests pass; new files analyze cleanly.
- Step 14 red: `PostInteractionAccountsPage` did not exist.
- Step 14 green: Added the shared Likes/Reposts page, extracted reusable profile account tile, preserved Follow-list behavior, and generated localized loading/title/empty/error states. Eight focused page tests and six Follow-list regressions pass with clean scoped analysis.
- Step 15 red: `PostQuotesPage` did not exist.
- Step 15 green: Added the plain-title Quotes page using normal `PostCard` composition, preserving repeated authors and server order while replacing/removing provider items after mutations/deletion. Twenty-three focused/nearby interaction tests pass; scoped analysis is clean.
- Step 16 red: `PostInteractionSummary` did not exist.
- Step 16 green: Added the model-derived nonzero-only wrapping summary with fixed order, exact localized plurals, independent callbacks, 48-point targets, and explicit value/type/destination semantics. All nine UT-001 tests pass.
- Step 17 red: The root summary and its three navigation links were absent, and successful interaction model replacements could not render updated summary values.
- Step 17 green: Inserted the summary only beneath the root `PostCard`, added minimal typed destination routes, preserved mutation controls/shared surfaces, and verified model-backed like/repost updates. Twenty-five thread/summary and five relevant card tests pass.
- Step 18 red: `PostCard.onViewLikes` did not exist.
- Step 18 green: Added the localized first standalone non-destructive menu group only for positive-like responses, wired direct/nested response DID+rkey routes, and kept root/zero/repost/quote/inline actions absent. UT-006, AT-005, and all thread tests pass.
- Step 19 initial state: Direct route production behavior already worked after step 17. Four cases passed immediately; the remaining failures were test-harness assumptions and an await-before-pop deadlock.
- Step 19 green: Added eight AT-009 route acceptance tests covering typed DID/rkey parsing, no `$extra`, signed-out redirects, compact/large shell placement, and push/back thread-state preservation. Twenty-two router regression tests also pass; no production change was needed.
- Step 20 red: Account rows lacked explicit profile-action semantics, empty states lacked standalone status labels, and error states were not live regions.
- Step 20 green: Added those semantics and verified summary/menu/page behavior plus full and pagination layouts at all four approved viewport/text-scale combinations. Twelve AT-007 and 30 nearby tests pass serially.
- Step 21 initial/green: All REG-001/REG-002 assertions passed immediately; no production regression existed. Added nine regression tests proving mutation/list separation, combined share behavior, root-only summary scope, response-only liker scope, and no leakage across other card surfaces.
- Step 22 initial/green: REG-003/REG-005 passed without production fixes. Added AppView and Flutter assertions for independent repost/quote state, repeated quotes, viewer repost state, muted placeholders, unavailable nested targets, and one-level non-recursive quote hydration; 43 focused Flutter tests and the Go regression suite pass.
- Step 23 initial/green: The serial regression suite passed 233 tests with one unrelated baseline author-profile-route failure. Added/retagged nine REG-006 cases for sorting, replies, pin/report/delete, interaction navigation/back, and compact/large shells; 150 thread/comment/router and 22 relevant card tests pass.

## Final Verification

- `just fmt`: Passed.
- `TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test ./internal/api ./internal/routes -count=1`: Passed.
- Focused Flutter feature suite across data, providers, pages, accessibility, summary, and routes: 107 tests passed.
- `just appview-check`: Passed during implementation review. Two later correction reruns reached the full suite but failed only `TestS3ObjectStoreAgainstMinIO` with `private object store unavailable`; no changed post-interaction package failed.
- `just app-analyze`: Reports five diagnostics in unchanged native-video source/tests and `composer_video_attachment_card_test.dart`. A detached clean `HEAD` worktree reproduces the same five diagnostics. All changed feature files report no diagnostics through Dart MCP analysis.
- `just app-test`: 2,199 tests passed and three tests failed. A detached clean `HEAD` worktree reproduces the same two business-provider `Bad state: No element` failures and the same `PostCard TDD-005A` author-route failure. All feature-focused suites, including the corrected 12-test production-router suite, pass.
- `git diff --check`: Passed.
- No migration, lexicon, dependency, PDS-read, or OAuth changes were made.

## Completion Checklist

- [x] All Must requirements covered by tests or documented gaps
- [x] All planned automated Must tests passing
- [x] Relevant feature regression tests passing
- [x] No unlinked behavior implemented
- [x] Generated files are current
- [x] `just fmt` passes
- [ ] `just appview-check` passes
- [ ] `just app-analyze` passes
- [ ] `just app-test` passes
- [x] MAN-001 recorded as blocked
- [x] MAN-002 recorded as blocked
- [x] Docs updated and read back
- [ ] Review completed or explicitly skipped

## Implementation Review Corrections

The 2026-09-07 implementation review in `06-implementation-review.md` returned `Changes required`. The user approved a correction pass. Corrections retain the original requirements and use the review findings as the ordered red-green-refactor inputs.

| Step | Review ID | Requirement IDs | Acceptance Criteria / Tests | Expected Initial State |
|---|---|---|---|---|
| 26 | IR-001 | NFR-004 | AC-021, EC-011, UT-004, IT-009 | Fails: old account rows/cursor remain after activation changes |
| 27 | IR-002 | FR-010 | AC-017, EC-007, IT-004, IT-012 | Fails: detail counts ignore viewer language policy |
| 28 | IR-003 | FR-006, FR-007 | AC-007, AC-013, UT-004, REG-003 | Fails: late continuation overwrites quote mutation/removal |
| 29 | IR-004 | FR-006, FR-010 | AC-007, AC-017, EC-007, REG-005 | Fails: revealable muted quote has no Reveal action |
| 30 | IR-005 | FR-005, FR-010 | AC-012, AC-013, AC-017, IT-001, IT-010, IT-012 | Fails: empty continuation reports zero total |
| 31 | IR-006 | FR-008, FR-009, FR-010 | AC-015, AC-017, AC-024, IT-006, IT-011 | Fails: response Likes target does not apply conversation visibility |
| 32 | IR-007 | NFR-001 | AC-019, IT-013 | Fails: handle resolution scales with distinct quote authors |
| 33 | IR-008 | NFR-004, NFR-006 | AC-014, AC-021, AC-023, AT-006, IT-004, IT-007, IT-011, REG-002, REG-004 | Fails: planned acceptance matrices are incomplete |
| 34 | IR-009 | FR-002, FR-007 | AC-008, AC-009, AT-006, AT-009 | Fails: direct unavailable route Back can be a no-op |
| 35 | IR-010 | NFR-006 | AC-023 | Fails: repository Flutter gates are not established green or baseline |
| 36 | IR-011 | NFR-003 | AC-020, MAN-001, MAN-002 | Blocked until visual/device sessions are available |
| 37 | IR-012 | FR-008, NFR-002 | IT-006 | Documentation clarification: coding plan intentionally reuses bounded `parseLimit` |
| 38 | IR-013 | FR-012, FR-013, RULE-005 | AC-018, AC-024, REG-002 | Fails until representative production wiring is pinned |
| 39 | IR-014 | NFR-003 | AC-020, MAN-001, MAN-002 | Blocked until seeded visual and screen-reader sessions are available |

### Correction Execution Notes

- Step 26 red: Account and quote provider tests timed out after activation changed during continuation because the stale completion was rejected while old state/cursor remained and no cursorless reload started.
- Step 26 green: Both provider builds now watch the session registry future and bind their initial request to its active lease. Activation invalidates the provider, discards old rows/cursors, and starts page one for the new account. Two IR-001 tests and all 13 provider tests pass.
- Step 27 red: Direct post, root-thread/comment, and reply-list handler tests failed to compile because their constructors accepted no language preference reader; all three passed an empty language list to engagement summaries.
- Step 27 green: All post-detail hydration handlers now load authoritative content languages and pass them to engagement summaries; production routes inject the private language store. Three focused IR-002 tests pass.
- Step 28 red: A pending quote continuation restored a removed post and discarded a replacement because it appended to state captured before the request.
- Step 28 green: Continuation now appends into the latest generation-compatible state, preserving local replacements/removals. The IR-003 race test and all 14 provider tests pass.
- Step 29 red: A revealable muted quote rendered no established `Show post` action because Quotes supplied no reveal callback.
- Step 29 green: Quotes now supplies a reveal callback that refetches the selected post through AppView and replaces it in provider state. The focused IR-004/REG-005 widget test passes.
- Step 30 red: After deleting the only interaction behind a valid continuation cursor, PostgreSQL returned an empty page with `totalCount: 0` while one newer eligible liker remained.
- Step 30 green: The account query now computes totals and the seek page in one SQL snapshot and left-joins the page to its total, preserving the count even with no page rows. IR-005 and nearby pagination/account-list tests pass.
- Step 31 red: A response whose author blocked its parent still resolved as a Likes target; after fixing the target itself, a nested reply under that protected ancestor still resolved.
- Step 31 green: Eligible post reads now apply conversation-equivalent reply-parent and mention block predicates, and recursive context validation applies them to every ancestor. IR-006, IT-011, and saved-context regression tests pass.
- Step 32 red: IT-013 observed one external resolver call per distinct quote/preview author despite complete local identity-cache fixtures.
- Step 32 green: Quotes now batch-load all outer and target-author handles from the local identity cache in one set-based query and use that fixed map for outer cards and one-level previews. IT-013 reports zero external resolver calls and both bounded hydration and preview regression tests pass.
- Step 33 red: Quotes lacked its full loading/error/retry/continuation matrix, target policy tests omitted missing and reverse-block cases, and production `AddRoutes` coverage did not execute successful reads with a fail-on-use PDS dependency.
- Step 33 green: Added the Quotes page-state matrix, missing targets across all kinds, both quote-author block directions, blocked targets across all kinds, shared-card leakage assertions, and successful production-mux Likes/Reposts/Quotes reads with a PDS factory that fails if constructed. Focused Flutter and PostgreSQL-backed API/routes suites pass.
- Step 34 red: Both destination pages used `Navigator.maybePop` alone, so direct-entry unavailable routes had no working exit.
- Step 34 green: Both pages now pop an existing stack or navigate to Feed, with direct-entry account and Quotes tests. The page suites and corrected 12-test production-router suite pass.
- Step 35: `just app-test` now reaches 2,199 passes with only three failures that reproduce identically in a detached clean `HEAD` worktree. `just app-analyze` reports the same five diagnostics on the feature worktree and detached clean `HEAD`; scoped analysis of all correction files is clean.
- Step 36: MAN-001 and MAN-002 remain blocked because no seeded app/device and real screen-reader session are available.
- Step 37: Clarified IT-006 to match the approved coding plan and established `parseLimit` contract: malformed/nonpositive limits use the default and oversized limits use the cap.
- Step 38 planned: Replace the label-only standalone-card REG-002 loop with an architecture regression over every production `PostCard` consumer. Pin feed, profile, search, project, notification, standalone-list, root-detail, comment, and nested-reply wiring; permit summaries and response-list callbacks only in the thread implementation. Run the new test first, then the existing thread runtime REG-002 test.
- Step 39 planned: Attempt MAN-001 and MAN-002 only against a seeded app built from this worktree. Record a blocker rather than treating automated viewport, semantics, or keyboard tests as substitutes.
