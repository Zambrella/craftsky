# TDD Implementation Plan: Pinned Profile Posts

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` — `Approved`, no blocking findings.
- Coding plan: `04-coding-plan.md`
- Implementation approval: the user explicitly invoked `implement-tdd` on 2026-08-05, approving the planned migration and private AppView-state changes.

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update one focused failing test before its implementation.
- Run the smallest relevant test first and record the meaningful red result.
- Refactor only while tests are green.
- Keep traceability and the execution log updated after each loop.
- Keep pin state in AppView Postgres; do not modify PDS records, lexicons, Tap, or public post models.
- Do not add payment, plan, tier, entitlement, access-gating models, dependencies, fixtures, telemetry, or tests.
- Preserve bodyless target-specific PUT/DELETE with authoritative `200` responses.
- Preserve exact `pinnedPostUri` omission and active-account-plus-slot pending semantics.
- Do not commit or push during this stage unless the user explicitly requests it.

## Test Order

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---:|---|---|---|---|
| 1 | IT-001 | FR-005, NFR-006 | AC-007, AC-013, AC-020 | Migration files and table absent. |
| 2 | UT-008 | RULE-001, RULE-002 | AC-001–AC-006 | Slot classifier absent. |
| 3 | IT-002 | BR-001, FR-002, NFR-001, RULE-001 | AC-003, AC-004, AC-014 | Pin store/upsert absent. |
| 4 | IT-003 | FR-002, FR-003, NFR-001, RULE-001 | AC-004, AC-005, AC-014 | Mutation serialization absent. |
| 5 | IT-004 | FR-004, FR-010, RULE-002 | AC-006, AC-015, AC-016 | Target policy enforcement absent. |
| 6 | IT-005 | FR-003, FR-013 | AC-005, AC-006, AC-017 | Pin handlers absent. |
| 7 | IT-009 | FR-013 | AC-017 | Route policies and registrations absent. |
| 8 | UT-002 | FR-013 | AC-017 | Flutter pin-state model absent. |
| 9 | IT-015 | FR-013 | AC-006, AC-017 | Flutter pin API calls absent. |
| 10 | UT-009 | FR-013 | AC-006, AC-017 | Pin error contract assertions absent. |
| 11 | IT-006 | BR-001, FR-006, NFR-002 | AC-008–AC-010, AC-012 | Pin-aware first page absent. |
| 12 | IT-007 | FR-007, NFR-002 | AC-009 | Pin-bound traversal absent. |
| 13 | IT-008 | FR-006, FR-011 | AC-012 | Policy-aware promotion absent. |
| 14 | IT-013 | NFR-002 | AC-009, AC-019 | Query-plan proof absent. |
| 15 | IT-010 | FR-005, FR-012 | AC-013, AC-016 | Structural cleanup absent. |
| 16 | IT-011 | FR-004, FR-010, FR-012, NFR-001, NFR-003 | AC-015, AC-016 | Cross-account isolation coverage absent. |
| 17 | IT-012 | BR-003, FR-005, NFR-003, RULE-003 | AC-007, AC-011, AC-016, AC-019 | Privacy side-effect sentinel absent. |
| 18 | IT-014 | NFR-003, NFR-005 | AC-019 | Pin observability absent. |
| 19 | UT-001 | FR-006, FR-013 | AC-008, AC-010, AC-017 | Page metadata model absent. |
| 20 | UT-003 | FR-008 | AC-003, AC-005, AC-018 | Pin controller absent. |
| 21 | UT-006 | FR-008, NFR-001 | AC-013, AC-014, AC-016 | Mutation lease fencing absent. |
| 22 | UT-004 | FR-007 | AC-009 | Invalid-cursor restart absent. |
| 23 | IT-016 | FR-007, FR-008 | AC-003, AC-005, AC-009, AC-013, AC-018 | Provider coordination absent. |
| 24 | UT-005 | FR-001, NFR-004 | AC-001, AC-002, AC-005, AC-006, AC-018 | Pin menu behavior absent. |
| 25 | UT-007 | BR-002, FR-009, NFR-004, RULE-004 | AC-010, AC-018 | Pinned attribution absent. |
| 26 | AT-001 | BR-001, FR-001, RULE-002 | AC-001 | Standard profile workflow absent. |
| 27 | AT-002 | BR-001, FR-001, RULE-002 | AC-001, AC-002 | Approved-surface workflow absent. |
| 28 | AT-003 | BR-001, FR-002, FR-008, RULE-001 | AC-003, AC-004 | Confirmed replacement workflow absent. |
| 29 | AT-004 | FR-001, FR-003, FR-008, RULE-001 | AC-005, AC-014 | Target-specific unpin workflow absent. |
| 30 | AT-005 | BR-001, BR-002, FR-006, FR-009, RULE-004 | AC-008, AC-010, AC-018 | Profile pin presentation absent. |
| 31 | AT-006 | FR-001, FR-004, FR-011, RULE-002 | AC-006, AC-012 | Cross-layer invalid-target coverage absent. |
| 32 | AT-007 | FR-006, FR-007, NFR-002 | AC-008, AC-009 | Cross-layer restart workflow absent. |
| 33 | AT-008 | FR-008, NFR-004 | AC-018 | Failure recovery workflow absent. |
| 34 | AT-009 | FR-008, FR-012, NFR-001 | AC-013, AC-016, AC-018 | Account switch workflow absent. |
| 35 | AT-010 | BR-003, FR-010, RULE-004 | AC-015, AC-016 | Universal-member workflow absent. |
| 36 | AT-011 | FR-005, FR-011, FR-012 | AC-012, AC-013, AC-016 | Full lifecycle workflow absent. |
| 37 | AT-012 | BR-003, FR-009, NFR-003, RULE-003 | AC-007, AC-011 | Default-off surface coverage absent. |
| 38 | REG-001 | FR-006, FR-007, RULE-003 | AC-008, AC-009 | No-pin metadata guard absent. |
| 39 | REG-002 | BR-003, FR-009, RULE-003 | AC-011 | Presentation leakage guard absent. |
| 40 | REG-003 | FR-011 | AC-012 | Policy-before-limit guard absent. |
| 41 | REG-004 | FR-006, NFR-003 | AC-007, AC-017 | Canonical JSON guard absent. |
| 42 | REG-005 | BR-003, FR-010 | AC-015, AC-016 | Universal authorization regression guard absent. |
| 43 | REG-006 | RULE-003 | AC-007, AC-011 | Unrelated-side-effect guard absent. |
| 44 | REG-007 | RULE-002 | AC-001, AC-002, AC-006 | Profile membership guard absent. |
| 45 | REG-008 | FR-005, NFR-006 | AC-007, AC-020 | Migration preservation guard absent. |
| 46 | MAN-001 | BR-002, NFR-004 | AC-018 | Physical accessibility verification pending. |
| 47 | MAN-002 | BR-002, FR-009, NFR-004 | AC-010, AC-018 | Device visual verification pending. |

## Implementation Steps

Each automated entry below is updated only after its focused red-green-refactor loop. Manual entries remain pending unless actually performed.

### Step 1: IT-001

- Write failing test: Added `TestProfilePinsMigration` for up/down/up, owner/slot uniqueness, slot values, owner/target cascades, required constraints/index, and preservation of chronology/saved-post sentinels.
- Run command: `cd appview && TEST_DATABASE_URL=postgres://craftsky:dev@localhost:16430/craftsky_dev?sslmode=disable go test ./internal/db -run TestProfilePinsMigration -count=1`
- Confirmed failure: `read up migration: open ../../migrations/000035_profile_pins.up.sql: no such file or directory`.
- Implement: Added reversible `000035_profile_pins` migrations with private owner/slot rows, post FK, state token, timestamps, and target index.
- Green result: Focused test passed.
- Refactor / nearby checks: `TestProfilePinsMigration|TestMigrationVersionsAreUniqueAndPaired` passed.
- Notes: Highest migration was still `000034`. The worktree compose stack publishes Postgres on port `16430`, so execution used that current port rather than the example `5433` port. An initial sandbox/cache failure and a pre-stack connection refusal were infrastructure failures, not accepted red results.

### Steps 2–25: Completed Automated Tests

- Status: Completed in the exact order above; evidence is recorded in the TDD Execution Log.

### Steps 26–45: Completed Acceptance And Regression Tests

- Status: Completed in the exact order above using the unit/integration seams established by steps 1–25 plus focused profile, timeline, and thread-root workflow tests.
- Evidence and requirement composition are recorded below; no test is claimed beyond the automated seam it actually exercises.

### Step 46: MAN-001

- Status: Pending external physical-device verification after automation.
- Do not report as passed unless VoiceOver/TalkBack and applicable keyboard/focus checks are actually run.

### Step 47: MAN-002

- Status: Pending device/emulator visual verification after automation.
- Do not report as passed unless light/dark, narrow-width, and supported maximum text-scale checks are actually run.

## TDD Execution Log

### IT-001 — completed

- Requirements / criteria: FR-005, NFR-006 / AC-007, AC-013, AC-020.
- Red: missing `000035_profile_pins.up.sql`.
- Green: reversible private table with exact capacity, lifecycle, internal-state, and index contract.
- Verification: focused migration and migration-file-pair tests passed against real Postgres.

### UT-008 — completed

- Requirements / criteria: RULE-001, RULE-002 / AC-001–AC-006.
- Red: the focused test failed to compile because the slot, target-shape, and classifier API did not exist.
- Green: `ClassifyProfilePinSlot` accepts top-level standard and quote shapes, accepts fully materialized top-level projects, and rejects replies plus inconsistent project/quote shapes.
- Verification: `go test ./internal/api -run TestClassifyProfilePinSlot -count=1` passed after formatting the focused files.

### IT-002 — completed

- Requirements / criteria: BR-001, FR-002, NFR-001, RULE-001 / AC-003, AC-004, AC-014.
- Red: the focused real-Postgres test failed to compile because the pin store, authoritative state, mutation result, and operation types did not exist.
- Green: the store reads empty/two-slot state, inserts independent slots, keeps same-target token/timestamps unchanged, and atomically replaces one slot with a rotated opaque token while preserving the other.
- Verification: the focused store test and adjacent slot-classifier test passed.

### IT-003 — completed

- Requirements / criteria: FR-002, FR-003, NFR-001, RULE-001 / AC-004, AC-005, AC-014.
- Red: the deterministic transaction-barrier test failed to compile because target-specific `Unpin` did not exist.
- Green: mutation commits are serialized by owner after target validation; controlled B/C replacements produce last-committed-wins state; stale URI deletion is a no-op; current URI deletion clears only that pin.
- Verification: the focused transaction-barrier test passed without sleeps, then all profile-pin policy/store tests passed together.

### IT-004 — completed

- Requirements / criteria: FR-004, FR-010, RULE-002 / AC-006, AC-015, AC-016.
- Red: the policy test failed to compile because the explicit forbidden error did not exist.
- Green: new pins require owner/target DID equality, current target membership, visible moderation state, top-level structure, and consistent standard/project materialization. Missing/hidden targets are indistinguishable, while target-specific unpin bypasses target visibility and removes a retained hidden pin.
- Verification: the focused policy test passed, then all profile-pin policy/store tests passed together against real Postgres.

### IT-005 — completed

- Requirements / criteria: FR-003, FR-013 / AC-005, AC-006, AC-017.
- Red: the handler contract test failed to compile because the private read and target-specific mutation handlers did not exist.
- Green: narrow reader/mutator handlers parse identifiers at the boundary, enforce owner targeting, map exact 403/404/422/500 envelopes, construct canonical unpin URIs, and return only nullable `standardPostUri`/`projectPostUri` in authoritative `200` JSON bodies.
- Verification: the focused handler test passed, then all profile-pin API tests passed together. Request-body rejection remains intentionally assigned to IT-009 route middleware.

### IT-009 — completed

- Requirements / criteria: FR-013 / AC-017.
- Red: the focused route test reported all three profile-pin policies missing.
- Green: GET state and PUT/DELETE mutation routes are registered with authenticated device and current-member middleware, `BodyNoBody`, read/write rate classes, and one shared AppView store.
- Verification: the focused route test passed, followed by the policy registry and all-v1-policy middleware regression tests.

### UT-002 — completed

- Requirements / criteria: FR-013 / AC-017.
- Red: the focused Flutter test failed because the authoritative pin-state model and generated mapper did not exist.
- Green: `ProfilePinState` decodes and re-encodes both nullable authoritative slots and exposes slot-based URI lookup without adding payment or access-gating state.
- Verification: `flutter test test/feed/models/profile_pin_state_test.dart` passed, followed by the adjacent `profile_pin_state_test.dart` and `post_page_test.dart` pair.
- Refactor note: code generation rewrote unrelated mapper whitespace and provider hashes; those generated-only diffs were removed, leaving only the new pin mapper and its bootstrap registration.

### IT-015 — completed

- Requirements / criteria: FR-013 / AC-006, AC-017.
- Red: the focused client-contract test failed to compile because private-state GET and target-specific pin/unpin methods did not exist.
- Green: the Flutter client and repository expose exact GET, bodyless PUT, and bodyless DELETE operations, decode every successful response as the authoritative two-slot state, and use the shared `ApiException` path.
- Verification: the focused profile-pin client group passed, followed by the complete existing `post_api_client_test.dart` suite plus the pin-state model tests (41 total passing tests).
- Refactor note: the programmable fake repository now exposes matching optional callbacks for later provider TDD; unstubbed calls still fail loudly.

### UT-009 — completed

- Requirements / criteria: FR-013 / AC-006, AC-017.
- Red: the focused Flutter test failed to compile because structured API failure details did not retain a safe AppView 4xx message.
- Green: pin errors preserve status, code, 4xx developer message, request ID, and a redacted endpoint category; internal failures preserve safe diagnostics but intentionally discard potentially sensitive 5xx server text in favor of `http_500`.
- Verification: the focused five-case error test passed, followed by the shared error-mapping and complete post-client suites (48 total passing tests).
- Refactor note: added the bounded `appview.posts.pin` endpoint category; dynamic DID/rkey values remain absent from diagnostics.

### IT-006 — completed

- Requirements / criteria: BR-001, FR-006, NFR-002 / AC-008–AC-010, AC-012.
- Red: the real-Postgres first-page test failed to compile because profile list handlers accepted only language preferences and had no pin-aware read dependency.
- Green: standard/project page one reads a structurally compatible, moderation-visible pin; promotes it before chronology; excludes it from the remainder; preserves limits 1, 2, and 10; and emits `pinnedPostUri` only when promotion occurs.
- Verification: the focused real-Postgres pagination test passed, followed by all profile-pin, existing profile-post/project-list, and route regressions.
- Refactor note: post-row scanning now supports bounded extra query columns so the list-pin query hydrates the post and private state token in one set-based row read.

### IT-007 — completed

- Requirements / criteria: FR-007, NFR-002 / AC-009.
- Red: the fixed-state traversal returned the promoted pin again on the final page (9 rows for an 8-row dataset), demonstrating an actual duplicate.
- Green: profile cursors wrap the existing chronological boundary with list kind and current private state token; later pages unwrap and validate that binding, over-fetch by one while a visible pin exists, and exclude the pin without gaps or page-size drift.
- Verification: the focused real-Postgres traversal test passed with tied timestamps, unique full traversal, later-page metadata omission, wrong-list-kind rejection, and replacement/clear invalidation; adjacent profile-list and route regressions also passed.

### IT-008 — completed

- Requirements / criteria: FR-006, FR-011 / AC-012.
- Initial result: the new focused policy test passed on its first run because IT-006's shared list-pin query already reused the existing moderation and language predicates, while the handler already applied bidirectional block policy before listing. No additional production behavior was needed.
- Characterization proof: language mismatch, temporary moderation, and a viewer block each suppress promotion and omit metadata; the page fills from allowed chronology where applicable; the private pin remains stored; and reversing each temporary exclusion restores promotion.
- Verification: the focused real-Postgres policy test and the combined first-page/traversal/policy suite passed.

### IT-013 — completed

- Requirements / criteria: NFR-002 / AC-009, AC-019.
- Initial result: after correcting a missing timestamp in the test fixture, both performance proofs passed without a production change; the fixture error was not counted as a behavioral red result.
- Green: `EXPLAIN` uses `profile_pins_pkey` for owner/slot lookup and `craftsky_posts_pkey` for URI hydration; handler pin/list/engagement calls stay fixed at 1/1/1 for page sizes 1, 10, and 50.
- Verification: the focused real-Postgres query-plan and bounded-call tests passed.

### IT-010 — completed

- Requirements / criteria: FR-005, FR-012 / AC-013, AC-016.
- Red: after an indexed standard post changed into a reply, both `standard` and `project` pin rows remained; only `project` should have survived.
- Green: the post indexer now deletes a pin in the same upsert transaction when the final indexed/materialized row no longer satisfies its stored slot, without moving it or touching the independent slot. Target and membership foreign keys provide permanent-delete cleanup.
- Verification: the focused structural-transition test passed, the real-Postgres target/membership cascade test passed, and the complete `internal/index` suite passed.

### IT-011 — completed (backend portion)

- Requirements / criteria: FR-004, FR-010, FR-012, NFR-001, NFR-003 / AC-015, AC-016.
- Initial result: the real-Postgres owner-isolation test passed on its first run because owner/slot keys, handler-authenticated ownership, and owner foreign-key cleanup were already established in earlier loops.
- Characterization proof: two owners retain distinct state across a new store instance; one owner's replacement cannot change the other; membership removal clears only the removed owner.
- Remaining linked proof: active-device/account completion fencing and client reload behavior remain assigned to UT-006 and IT-016 rather than being claimed here.

### IT-012 — completed

- Requirements / criteria: BR-003, FR-005, NFR-003, RULE-003 / AC-007, AC-011, AC-016, AC-019.
- Initial result: the privacy sentinel passed on its first run because `ProfilePinStore` depends only on Postgres and earlier mutation code writes only `profile_pins`.
- Characterization proof: database triggers observed no post, like, repost, or saved-state writes across pin/replace/unpin; another owner's URI never appeared in mutation responses; the other owner's private row remained unchanged.
- Boundary note: no PDS client, OAuth token, Tap, lexicon, notification, timeline/search ranking, or interaction service is accepted by the mutation store.

### IT-014 — completed

- Requirements / criteria: NFR-003, NFR-005 / AC-019.
- Red: the focused telemetry test failed to compile because `ProfilePinStoreOptions` had no observer seam.
- Green: each mutation emits one bounded duration observation with operation (`pin|replace|unpin`), slot (`standard|project|unknown` for an unclassifiable no-op/rejection), result (`success|noop|rejected|error`), and error class (`none|forbidden|not_found|policy|store`).
- Verification: seven success, replacement, no-op, rejection, and internal-error cases passed; metric validation and redaction sentinels passed; the full observability package and focused profile-pin API/routes suites passed.

### UT-001 — completed

- Requirements / criteria: FR-006, FR-013 / AC-008, AC-010, AC-017.
- Red: the focused Flutter test failed to compile because `PostPage` had no page-level `pinnedPostUri` field.
- Green: visible metadata round-trips; absent and explicit-null input both decode as no pin; encoding omits null metadata; canonical item maps do not gain `isPinned`.
- Verification: all four `post_page_test.dart` cases passed. Code generation changed the intended mapper plus two unrelated files; the two unrelated diffs were removed.

### UT-003 — completed

- Requirements / criteria: FR-008 / AC-003, AC-005, AC-018.
- Red: the focused Flutter provider test failed to compile because the account-family controller, presentation state, and bounded mutation outcomes did not exist.
- Green: confirmed two-slot state remains unchanged while a slot is pending; duplicate same-slot requests are suppressed; the independent slot can proceed; authoritative full-state responses reconcile on success; failures preserve the latest confirmed state; exact success/error copy is exposed through bounded outcomes.
- Verification: the focused deferred standard/project success/failure test passed. Generated output was limited to the new provider family with no unrelated churn.

### UT-006 — completed

- Requirements / criteria: FR-008, NFR-001 / AC-013, AC-014, AC-016.
- Initial result: the focused late-completion test passed on its first run because UT-003's controller already fenced mutation completion with `ActiveAccountLease`; the account-boundary invalidator did not yet include the new provider family.
- Green: switching the active account while account A's mutation is pending returns a bounded stale-completion outcome, preserves A's last confirmed state, and cannot publish into account B; account teardown now explicitly invalidates every live profile-pin family entry.
- Verification: the focused account-switch test passed after adding the account-boundary wiring.

### UT-004 — completed

- Requirements / criteria: FR-007 / AC-009.
- Red: both focused provider tests failed to compile because accumulated standard/project state had no `pinnedPostUri`, and neither provider could represent a restarted first page.
- Green: initial builds retain page-level pin metadata; normal appends retain first-page metadata; `ApiBadRequest('invalid_cursor')` discards the stale traversal, performs exactly one cursorless page-one request, and publishes only that restarted page and metadata after the existing active-account lease check.
- Verification: both focused restart tests passed, followed by the complete standard/project provider suites (17 passing tests). Unrelated generator-only churn was removed.

### IT-016 — completed

- Requirements / criteria: FR-007, FR-008 / AC-003, AC-005, AC-009, AC-013, AC-018.
- Red: the focused coordination test failed to compile because pin mutations accepted no author cache IDs and could not refresh live profile-list families.
- Green: a current successful mutation invalidates only live profile-list families matching the affected slot for the author's DID/handle IDs; failures and stale completions invalidate nothing. Earlier provider loops already prove authoritative reconciliation, same-slot suppression, independent-slot availability, account isolation, and invalid-cursor restart.
- Verification: the focused coordination test passed, followed by the combined pin/standard/project provider suites (20 passing tests).

### UT-005 — completed

- Requirements / criteria: FR-001, NFR-004 / AC-001, AC-002, AC-005, AC-006, AC-018.
- Red: the widget test failed to compile because `PostCard` had no explicit pin-action surface opt-in.
- Green: authoritative loaded state controls `Pin post` versus `Unpin post`; default-off, non-owner, reply, loading, and mismatched active-account cards expose no pin action; an affected pending slot retains its confirmed label with disabled semantics while the independent slot remains enabled.
- Refinement red/green: a pure client-visible-shape test first failed because no shared classifier existed; `classifyProfilePinSlot` now rejects replies and inconsistent project-plus-quote shapes before exposing an action, matching the AppView policy boundary.
- Verification: the pure classifier test and all three focused menu eligibility/current-pin/pending tests passed. Exact action copy is localized in the English ARB and generated localization output.

### UT-007 — completed

- Requirements / criteria: BR-002, FR-009, NFR-004, RULE-004 / AC-010, AC-018.
- Red: the attribution test failed to compile because `PostCard` had no explicit profile-only annotation input.
- Green: the explicit opt-in renders an outlined pin plus exact `Pinned post` text in the repost attribution slot, uses one labelled non-button semantics node, and remains readable in the focused narrow-width/2x-text widget test.
- Verification: the focused attribution test passed after its semantics assertion was corrected to inspect the dedicated node rather than the enclosing tappable card surface.

### AT-001–AT-012 — completed automated coverage

- AT-001: a focused own-profile widget test opens the standard-post menu, calls target-specific pin, and shows localized `Post pinned`; AppView store/handler tests prove the corresponding authoritative slot write.
- AT-002: focused timeline and thread-root widget tests prove both approved non-profile surfaces; project-slot eligibility is covered by the shared card matrix and project profile provider tests. Comments/replies remain default-off.
- AT-003 and AT-004: deferred provider tests prove server-confirmed replacement, same-slot suppression, authoritative menu reconciliation, target-specific unpin, exact feedback, and independent-slot availability; real-Postgres tests prove atomic replacement and stale-unpin safety.
- AT-005: standard and project profile-tab tests render only the URI identified by page-one metadata with the exact accessible attribution; real-Postgres pagination tests prove ordering and limit behavior.
- AT-006: widget eligibility omits non-owner/reply actions; real-Postgres policy and handler tests prove forbidden/not-found/not-allowed outcomes without state change.
- AT-007: server traversal tests prove state-bound cursor invalidation and uniqueness; both Flutter profile providers prove one cursorless restart that discards stale traversal state.
- AT-008: provider failure paths preserve confirmed state, clear pending state, keep the independent slot usable, and expose exact localized retry copy.
- AT-009: active-account lease tests prove a late account-A completion cannot mutate state, caches, or feedback for account B; account-boundary wiring invalidates the provider family.
- AT-010: owner-isolation/current-member route and store tests use the ordinary authenticated-member path with no additional account classification.
- AT-011: target deletion, membership deletion, structural transition, temporary policy suppression, and restoration are covered by real-Postgres lifecycle/policy tests.
- AT-012: privacy sentinels and default-off card tests prove no public/PDS write, no non-profile attribution, and no added ranking/count/notification/search behavior.
- Verification: the consolidated pinned-post Flutter suite passed 176 tests; the full Flutter suite passed 1,311 tests; the full AppView `go test ./... -count=1` suite passed against real Postgres.

### REG-001–REG-008 — completed

- REG-001/REG-003: no-pin chronology and policy-before-limit behavior passed in the profile pagination/policy suites.
- REG-002/REG-006/REG-007: `PostCard` inputs remain default-off; only profile, timeline, and thread-root call sites opt into actions, and only profile metadata opts into attribution. Existing search, project discovery, notification, saved-post, comment, and reply suites passed in the full Flutter run.
- REG-004: page metadata is separate from canonical `Post` JSON, uses camelCase, omits null, and passed model/API tests.
- REG-005: authenticated current-member route/store tests contain no additional authorization category.
- REG-008: migration up/down/up and unrelated-state preservation passed against real Postgres.
- Scope scan: the implementation contains no payment, plan-tier, entitlement, free/paid-user, or access-gating model, dependency, fixture, telemetry, or test.

## Final Verification

- Focused AppView suites passed throughout the TDD loops against `localhost:16430`.
- Full AppView passed: `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:16430/craftsky_dev?sslmode=disable go test ./... -count=1`.
- Consolidated pinned-post Flutter suite passed: 176 tests.
- Full Flutter passed: `flutter test` with 1,311 tests.
- Static analysis passed: `dart analyze` reported no issues.
- Formatting/scope checks passed: touched Go files were run through `gofmt`; touched Dart files were formatted; `git diff --check` passed; unrelated generator churn was removed.
- Manual gates MAN-001 and MAN-002 were not run and remain external release verification.

## Completion Checklist

- [x] All Must requirements covered by passing tests or documented manual gaps.
- [x] All planned automated Must tests passing.
- [x] Relevant regression tests passing.
- [x] No unlinked behavior implemented.
- [x] No payment/access-gating or public PDS/lexicon behavior added.
- [x] Generated files inspected for unrelated churn.
- [x] `05-implementation-plan.md` updated and read back.
- [ ] Implementation review completed or explicitly skipped.
