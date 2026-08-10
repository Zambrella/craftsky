# TDD Implementation Plan: Public Profile Customisation

## Status

Stage: Implementation
Started: 2026-08-10
Current loop: Implementation-review corrections complete
Current result: IR-001 through IR-005 have focused passing correction evidence and IR-006 traceability is corrected below. IR-007's Should-level custom signal is explicitly deferred; generic HTTP instrumentation remains. The full AppView race suite and static analysis pass. The broad Flutter run passes 1,409 tests and reproduces only the two pre-existing untouched `auth_complete_page_test.dart` failures caused by its `MaterialApp` harness lacking `GoRouter`.

GAP-001 through GAP-003 closed on 2026-08-10 before their dependent exact assertions. `01-requirements.md` Q11–Q13 record all stable colour, texture, and feedback constants.

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved with notes`)
- Coding plan: `04-coding-plan.md`

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update one focused failing test before implementation.
- Run the smallest relevant test first and confirm a meaningful failure.
- Implement only enough to make the active test green.
- Refactor only while tests are green.
- Keep traceability and actual command evidence updated here after every loop.
- Preserve the dedicated AppView boundary: no Lexicon, PDS, Tap, profile/blob, or indexing-convergence change.
- Keep public response additions tolerant/additive while the current mutation request remains strict.
- Never perform per-avatar customisation reads; hydrate one deduplicated DID set per response.
- Keep draft preview state local until an authoritative save succeeds.
- Use the shared avatar seam and the exact 36/48/96 px width table only.
- Do not start palette-sensitive, texture-style-sensitive, or exact failure-copy tests until GAP-001, GAP-002, or GAP-003 respectively is approved and recorded.
- Stop for explicit approval before migration, auth/permission, privacy, or security-sensitive implementation.

## Approval Gates

| Gate | Required before | Status | Closure evidence |
|---|---|---|---|
| GAP-001: five non-cobalt stable keys and audited base/foreground/hover/pressed/soft-container values | Palette-sensitive part of UT-001; UT-006; AT-005/AT-010 exact colour checks | Approved 2026-08-10 | `01-requirements.md` Q11 records `cobalt`, `orchid`, `rose`, `amber`, `lime`, and `teal` plus exact bundle constants. |
| GAP-002: per-colour texture tint and opacity | Exact UT-007 painter/style tests; AT-005 visual baselines | Approved 2026-08-10 | Each bundle uses its foreground tint at 18% opacity; recorded in `01-requirements.md` Q11–Q12. |
| GAP-003: exact save-failure feedback | Exact copy assertion in UT-009/AT-006 | Approved 2026-08-10 | Exact localized copy is `Couldn't save your profile customisation.`; recorded in `01-requirements.md` Q13. |
| High-risk implementation approval | IT-001 migration and later authenticated/privacy-sensitive AppView work | Approved 2026-08-10 | User instructed the implementation to proceed to completion after the explicit high-risk approval request. |

## Test Order

The order mirrors `04-coding-plan.md`. Acceptance and regression IDs that close a slice are placed after their underlying focused unit/integration loops.

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---:|---|---|---|---|
| 1 | UT-001 | FR-011, FR-012, RULE-001–RULE-003 | AC-005, AC-011–AC-013 | Expected to fail; gate closed |
| 2 | UT-002 | FR-003, RULE-001–RULE-003 | AC-005 | Fails |
| 3 | IT-001 | BR-001, FR-004, NFR-002 | AC-003, AC-016, AC-020 | Fails; high-risk approval required |
| 4 | IT-002 | BR-004, BR-005, FR-002–FR-004, RULE-004 | AC-003, AC-004, AC-014 | Fails |
| 5 | IT-003 | FR-002, FR-003 | AC-003, AC-005, AC-017 | Fails |
| 6 | IT-007 / REG-009 | BR-005, FR-004, RULE-005 | AC-003, AC-016 | Fails |
| 7 | IT-004 | FR-001, FR-012 | AC-001, AC-002, AC-013, AC-017 | Fails |
| 8 | IT-006 | FR-001, FR-013, NFR-004, RULE-004 | AC-015 | Fails |
| 9 | IT-005 | NFR-001 | AC-002, AC-019 | Fails |
| 10 | IT-008 / UT-003 | FR-002, FR-012, NFR-004 | AC-013, AC-017 | Fails |
| 11 | UT-008 | FR-005 | AC-006, AC-007 | Fails |
| 12 | UT-009 | FR-006 | AC-007, AC-014 | Fails after GAP-003 closes |
| 13 | UT-010 | FR-006, FR-010 | AC-014 | Fails |
| 14 | AT-006 / AT-007 | BR-004, FR-004–FR-006, FR-010 | AC-006, AC-007, AC-014 | Fails |
| 15 | REG-003 / REG-007 / REG-008 | FR-005, FR-006, FR-010, NFR-003 | AC-006, AC-007, AC-014, AC-018 | Fails |
| 16 | UT-004 | FR-007, RULE-002 | AC-009 | Fails |
| 17 | UT-005 | FR-007 | AC-008–AC-010 | Fails |
| 18 | AT-002 / AT-004 / REG-004 | BR-002, FR-001, FR-007, FR-008, NFR-001, RULE-002 | AC-002, AC-008–AC-010 | Fails |
| 19 | UT-007 structural / IT-010 | FR-009, FR-011, RULE-003 | AC-011, AC-019 | Fails after GAP-002 closes |
| 20 | UT-006 | FR-009–FR-011, RULE-001 | AC-012 | Fails after GAP-001 closes |
| 21 | AT-005 / REG-005 | BR-003, FR-009–FR-011, RULE-001, RULE-003 | AC-011, AC-012, AC-019 | Fails after GAP-001/GAP-002 close |
| 22 | UT-011 / AT-010 | FR-005, NFR-003 | AC-018 | Fails |
| 23 | IT-009 | BR-005, NFR-005 | AC-016, AC-019 | Fails |
| 24 | AT-001 / AT-003 / AT-008 / AT-009 | BR-001, BR-004, BR-005, FR-001–FR-004, FR-012, FR-013, NFR-004, RULE-004, RULE-005 | AC-001–AC-005, AC-013, AC-015–AC-017 | Fails until underlying slices are complete |
| 25 | REG-001 / REG-002 / REG-006 | NFR-002, NFR-004, NFR-006, FR-013 | AC-015, AC-017, AC-020 | Existing suites must remain green |
| 26 | MAN-001–MAN-003 | BR-003, FR-005, FR-009, FR-011, NFR-003, RULE-003 | AC-006, AC-007, AC-011, AC-012, AC-018, AC-019 | Manual supplements after automated green |

## Implementation Steps

### Step 1: UT-001 — Effective values, defaults, and closed catalogues

- Requirement IDs: FR-011, FR-012, RULE-001, RULE-002, RULE-003.
- Acceptance criteria: AC-005, AC-011, AC-012, AC-013.
- Write failing tests:
  - Go public API catalogue/default test under `appview/internal/api/`.
  - Dart value/catalogue/default test at `app/test/profile/models/profile_customisation_test.dart`.
  - Assert matching stable wire keys, cobalt `#1535D6`, default `medium`/`none`, exactly three border keys, and `none` plus six named background keys.
  - Assert independent per-field fallback in the value policy without depending on parent response models yet.
- Focused commands:
  - `cd appview && go test ./internal/api -run ProfileCustomisationCatalogue`
  - `cd app && flutter test test/profile/models/profile_customisation_test.dart`
- Confirmed failure: Both focused suites failed to compile because the intended Go and Dart catalogue/value APIs did not exist.
- Implement: Added only the standalone Go catalogue/effective-value policy and Dart immutable value/local theme-bundle catalogue; no parent response wiring, persistence, route, or widgets.
- Refactor: Kept the Go policy independent from storage/handlers and the Dart value independent from Flutter rendering types.
- Notes: Use only the six approved Q11 keys and bundle constants.

### Step 2: UT-002 — Strict full-replacement request validation

- Requirement IDs: FR-003, RULE-001, RULE-002, RULE-003.
- Acceptance criterion: AC-005.
- Write failing test: `appview/internal/api/profile_customisation_request_test.go` covering valid complete body, missing/extra/non-string fields, arbitrary colours, unsupported keys, nested resource/URL input, malformed/duplicate/oversized JSON.
- Run command: `cd appview && go test ./internal/api -run ProfileCustomisationRequest`.
- Confirmed failure: `go test ./internal/api -run ProfileCustomisationRequest` failed because the decoder API did not exist.
- Implement: Exact bounded decoder plus validation errors only; no store/route.
- Refactor: Reused the UT-001 catalogue membership policy; kept persistence and HTTP handling out of this loop.
- Notes: This loop may begin only after UT-001 is green because it consumes the approved colour catalogue.

### Step 3: IT-001 — Reversible AppView persistence

- Requirement IDs: BR-001, FR-004, NFR-002.
- Acceptance criteria: AC-003, AC-016, AC-020.
- Approval: Stop and obtain explicit migration approval before editing migration/test files.
- Write failing test: real-Postgres up/down/up, schema/FK/PK/timestamps, membership cascade, preservation of prior data.
- Run command: focused `go test` for `internal/db` with the repository database environment.
- Confirmed failure: The real-Postgres migration test failed because migration `000036` did not exist.
- Implement: Migration `000036` with explicit columns, DID primary key/membership cascade, timestamps, and no catalogue `CHECK` constraints.
- Refactor: None while migration test is red.
- Notes: Re-check the migration head immediately before allocation.

### Steps 4–9: AppView store, route, boundaries, response hydration

- Tests: IT-002, IT-003, IT-007/REG-009, IT-004, IT-006, IT-005.
- Requirements: BR-004, BR-005, FR-001–FR-004, FR-012, FR-013, NFR-001, NFR-004, RULE-004, RULE-005.
- Implement one test ID at a time after the high-risk approval.
- Store: missing-row defaults, atomic upsert, retry, isolation, concurrency, lifecycle.
- Route: authenticated/device/current-member `PUT /v1/profiles/me/customisation`, exact camelCase body and standard envelope.
- Boundary: zero PDS/Tap/blob/profile writer calls.
- Hydration: pure defaulted builders plus one deduplicated batch read per assembled response, with moderation shells preserved.
- Query evidence: statement count independent of page length and an indexed `ANY`/PK plan.
- Notes: Update this section with exact red/green commands and failure messages after each loop rather than batching evidence.

### Steps 10–15: Flutter compatibility, repository, editor, and Settings flow

- Tests: IT-008, UT-003, UT-008–UT-010, AT-006, AT-007, REG-003, REG-007, REG-008.
- Requirements: BR-004, FR-002, FR-005, FR-006, FR-010, FR-012, NFR-003, NFR-004.
- Add tolerant nested decoding before API/editor consumers.
- Keep `ProfileCustomisationEditorState` to `confirmed`, `draft`, and derived `isDirty`; use `AsyncValue` for initial load, retained-value save loading, and retained-value save error.
- Reconcile public/session caches only after the authoritative response and fence every continuation by the initiating account/session generation.
- Add typed Settings route, fixed localized controls, live local preview, explicit Save, value-based discard confirmation, exact success feedback, and exact retryable failure feedback.
- Notes: Never write draft choices into global profile/session state optimistically.

### Steps 16–18: Shared avatar geometry and propagation

- Tests: UT-004, UT-005, AT-002, AT-004, REG-004.
- Requirements: BR-002, FR-001, FR-007, FR-008, NFR-001, RULE-002.
- Lock all nine size/level widths first, then image/fallback/clipping/shadow behavior, then migrate each avatar consumer.
- Remove decorative frames/second rings only after the shared renderer is green.
- Use 36 px for navigation/account adapters rather than inventing a 32 px width.

### Steps 19–21: Local textures and profile-local theme scope

- Tests: structural UT-007/IT-010, then UT-006, AT-005, REG-005 after relevant gates.
- Requirements: BR-003, FR-009–FR-011, RULE-001, RULE-003.
- Acquire only the six approved Ribo assets and record provenance/license/attribution before asset-dependent code.
- Structural asset mapping, no-network, tiling, clipping, and the approved foreground-at-18% treatment are covered together.
- Exact fixed bundles/contrast and material-state tests may not precede GAP-001.
- Compact theme covers the entire card; full theme stops before the tab bar.

### Steps 22–26: Accessibility, observability, acceptance closure, and regression

- Tests: UT-011, AT-010, IT-009, AT-001, AT-003, AT-008, AT-009, REG-001, REG-002, REG-006, MAN-001–MAN-003.
- Requirements: all remaining linked Must requirements plus NFR-005.
- Complete automated semantics, focus, text-scale, contrast, response-inventory, moderation, compatibility, and full migration/regression assertions.
- NFR-005 customisation-specific result/error-class instrumentation is Should priority and is deferred after implementation review. Existing generic HTTP instrumentation remains; no DID, catalogue choice, asset name, or URL is introduced as a metric label. IT-009 is therefore not claimed as executed.
- Run manual assistive-technology, texture-balance, and colour-vision checks only after the automated suite is green and visual gates are closed.

## Codebase Inspection Notes

- Go API tests commonly use external package `api_test` and exercise exported public interfaces; new catalogue tests should follow that pattern.
- Existing request decoders use `FieldError` and table-driven validation tests; UT-002 should reuse the established error vocabulary rather than add a second envelope model.
- Flutter profile models use `dart_mappable`, but `ProfileCustomisation` needs manual/tolerant per-field parsing so one unknown value cannot fail the parent identity.
- Flutter tests use `flutter_test` expectations and focused model/widget files. Generated mapper outputs are committed.
- `ProfileAvatar` is the intended common renderer; `ProfileFramedAvatar` and `AccountAvatar` are parallel legacy seams that must eventually delegate to or be replaced by it.
- The current branch was clean at implementation-stage entry. The four approved workflow documents are already committed and pushed; implementation changes start with this file.

## Verification Commands

Focused commands will be recorded per loop. Planned broad verification:

```text
just test
just app-test
just app-analyze
git diff --check
```

Run `flutter gen-l10n` after ARB edits and `dart run build_runner build` after mapper/provider/router edits, then inspect generated changes for scope. Run Go formatting through the repository's standard command and Dart formatting only on touched source/test files before final verification.

## Execution Log

| Date | Test ID | Phase | Command / evidence | Result |
|---|---|---|---|---|
| 2026-08-10 | Setup | Contract | Read `01`–`04` completely and confirmed `03` is Approved with notes. | Pass |
| 2026-08-10 | Setup | Inspection | Inspected Go API test/request patterns, Flutter profile models/tests, shared avatar test seam, package manifests, and current branch status. | Pass |
| 2026-08-10 | UT-001 | Gate | GAP-001 still lacks five stable non-cobalt keys and audited bundle constants. | Paused before red test |
| 2026-08-10 | UT-001 | Gate | User approved the proposed six-key palette; constants recorded across workflow documents before test code. | Pass |
| 2026-08-10 | UT-001 | Red | `go test ./internal/api -run ProfileCustomisationCatalogue` failed on undefined catalogue/value APIs; `flutter test test/profile/models/profile_customisation_test.dart` failed because the intended model file and APIs did not exist. | Expected failure |
| 2026-08-10 | UT-001 | Green | `go test ./internal/api -run ProfileCustomisationCatalogue` and `flutter test test/profile/models/profile_customisation_test.dart`. | Pass |
| 2026-08-10 | UT-002 | Red | `go test ./internal/api -run ProfileCustomisationRequest` failed because `DecodeProfileCustomisationPut` did not exist. | Expected failure |
| 2026-08-10 | UT-002 | Green | `go test ./internal/api -run 'ProfileCustomisation(Catalogue|Request)'`. | Pass |
| 2026-08-10 | IT-001 | Approval | User approved proceeding to completion after the explicit migration/auth/privacy gate. | Pass |
| 2026-08-10 | IT-001 | Red | `go test ./internal/db -run ProfileCustomisationMigration` failed because migration `000036` did not exist. | Expected failure |
| 2026-08-10 | IT-001 | Green | `go test ./internal/db -run 'ProfileCustomisationMigration|MigrationVersions'`. | Pass |
| 2026-08-10 | IT-002 | Red | `go test ./internal/api -run ProfileCustomisationStore` failed because the store API did not exist. | Expected failure |
| 2026-08-10 | IT-002 | Green | `go test ./internal/api -run ProfileCustomisationStore`. | Pass |
| 2026-08-10 | IT-003 | Red | Handler contract tests failed because `PutProfileCustomisationHandler` did not exist; route registration/policy remained absent. | Expected failure |
| 2026-08-10 | IT-003 | Green | Focused handler, route-policy, auth/device/current-member, and persistence tests passed. | Pass |
| 2026-08-10 | IT-004 | Red | Batch reader/hydrator APIs and typed response customisation fields were absent. | Expected failure |
| 2026-08-10 | IT-004 | Green | Focused typed-default, nested batch hydration, and middleware tests passed; AppView API/route/db regression packages passed (real-Postgres cases skip without an exported database URL). | Pass |
| 2026-08-10 | IT-006 | Red | Unconditional hydrator test showed an unavailable actor with an empty retained handle regained default customisation. | Expected failure |
| 2026-08-10 | IT-006 | Green | Focused hydrator tests preserve blocked/unavailable shells while decorating valid Craftsky identities. | Pass |
| 2026-08-10 | IT-001–IT-004 | Integration | `TEST_DATABASE_URL=postgres://craftsky:dev@localhost:15787/craftsky_dev?sslmode=disable go test -v ./internal/api ./internal/db -run ProfileCustomisation` exercised migration up/down/up, cascade, default reads, upsert, batch hydration, route, and handler behavior against PostgreSQL. | Pass |
| 2026-08-10 | IT-008 / UT-003 | Green | Added tolerant nested Dart mapping to profile, author, actor, account-summary, and search identity models plus exact repository request/response mapping. | Pass |
| 2026-08-10 | UT-008–UT-010 | Green | Provider tests cover local draft state, retained `AsyncValue` loading/error values, duplicate-save suppression, authoritative response reconciliation, and retryable failure. | Pass |
| 2026-08-10 | UT-009 / UT-010 / REG-008 | Green | Confirmed values update alive self-profile caches, invalidate embedded identity collections, persist per-session switcher customisation, and fence a late completion so it updates only its retained initiating account. | Pass |
| 2026-08-10 | AT-006 / AT-007 | Green | Settings widget tests cover all three localized choices, live preview, exact success copy, exact failure copy, and retained retryable draft. | Pass |
| 2026-08-10 | UT-004 / UT-005 | Green | Shared-avatar tests cover all nine inside-border widths, selected colour, image/fallback paths, and preserved shadow behavior. | Pass |
| 2026-08-10 | AT-002 / AT-004 / REG-004 | Green | Profile, post, notification, search, edit-profile, navigation, and account-switcher focused suites pass through the shared avatar seam. | Pass |
| 2026-08-10 | UT-006 / UT-007 | Green | Exact fixed colour bundles pass AA contrast tests; six local Ribo PNG mappings pass no-network, tiling, clipping, tint, opacity, and `none` tests. | Pass |
| 2026-08-10 | AT-005 / REG-005 | Green | Compact profile tests confirm whole-view theme scope; full profile tests confirm the scope stops above the tab bar. | Pass |
| 2026-08-10 | Static | Green | `dart analyze`. | Pass, no issues |
| 2026-08-10 | AppView regression | Green | `just test` with local PostgreSQL and MinIO, including `-race` across all AppView packages. | Pass |
| 2026-08-10 | Flutter regression | Broad | Final `flutter test` after account-scoped cache integration. | 1,398 pass; two untouched `auth_complete_page_test.dart` tests fail because their `MaterialApp` harness has no `GoRouter`. The same two fail alone and neither the production page nor its test changed in this implementation. |
| 2026-08-10 | Flutter regression | Green | Corrected post and saved-post round-trip fixtures to include the newly required default nested customisation, then reran those focused suites. | Pass |
| 2026-08-10 | IR-001 / UT-006 / AT-005 | Red | `flutter test test/profile/widgets/profile_customisation_controls_test.dart` failed because the actual secondary Chunky control did not paint the selected bundle's soft-container colour. | Expected failure |
| 2026-08-10 | IR-001 / UT-006 / AT-005 | Green | `flutter test test/profile/widgets/profile_customisation_controls_test.dart test/profile/widgets/profile_card_test.dart` exercises real primary and secondary Chunky surfaces at rest, hover, focus, and full press; the existing compact profile suite also passes. | Pass |
| 2026-08-10 | IR-002 / UT-009 / REG-008 | Red | `flutter test test/profile/providers/profile_identity_cache_invalidator_test.dart` failed because no centralized identity-cache invalidator or auditable family inventory existed. | Expected failure |
| 2026-08-10 | IR-002 / UT-009 / REG-008 | Green | The focused invalidator and editor-provider suites pass. The inventory now includes single post, thread/comments, pins, timeline, notifications, owner collections, project feed, searches/suggestions/recent results, saved posts, and relationship lists; the authoritative-save and late-account fencing tests remain green. | Pass |
| 2026-08-10 | IR-003 / IT-005 | Red | The focused Go test failed to compile because the production batch query had no single named contract that a PostgreSQL plan test could explain. | Expected failure |
| 2026-08-10 | IR-003 / IT-005 | Green | With local PostgreSQL on port 5433, the focused test proves one hydrator batch call for page sizes 1, 25, and 250 with repeated DIDs, and `EXPLAIN` contains both `craftsky_profiles_pkey` and `profile_customisations_pkey`. | Pass |
| 2026-08-10 | IR-004 / AT-002 / REG-004 | Coverage | Non-default colour/thickness assertions were added to profile card, feed root/thread plus quote, notification actor, search profile result, post summary, edit preview, and navigation/account-switcher suites. Existing propagation passed on first execution; one notification assertion was narrowed from the first page avatar to the matching non-default actor avatar. | Pass |
| 2026-08-10 | IR-005 / UT-011 / AT-006 / AT-010 / REG-007 | Red | The expanded Settings suite initially proved the missing ordered focus structure; lifecycle test setup also required a rebuild before simulating dirty Back. | Expected failure |
| 2026-08-10 | IR-005 / UT-011 / AT-006 / AT-010 / REG-007 | Green | `flutter test test/settings/profile_customisation_page_test.dart` passes nine tests covering clean/dirty/reverted/saved Back, branded discard/cancel, duplicate pending activation, initial-load retry, selected semantics, explicit focus order, and 2x text in light/dark themes. | Pass |
| 2026-08-10 | IR-007 / NFR-005 / IT-009 | Decision | Customisation-specific bounded observability is deferred as a Should-level follow-up. Existing generic HTTP instrumentation remains; IT-009 is not marked executed. | Explicit deferment |
| 2026-08-10 | Correction verification | Green | The focused Flutter correction matrix passes 128 tests; the focused real-Postgres AppView API/routes suites pass; `dart analyze` reports no issues; `just test` passes every AppView package under the race detector; and `git diff --check` passes. | Pass |
| 2026-08-10 | Flutter regression | Broad | Final `flutter test` after review corrections. | 1,409 pass; the same two untouched `auth_complete_page_test.dart` tests fail because their `MaterialApp` harness has no `GoRouter`. No feature test fails. |

## Completion Checklist

- [x] All Must requirements covered by passing tests or an explicit documented gap.
- [x] All implemented Must unit/integration/acceptance tests passing.
- [x] Relevant profile/feed/notification/search/settings/navigation regressions passing.
- [x] Manual supplements explicitly deferred to implementation review/device QA because assistive-technology and subjective texture-balance checks require an interactive target.
- [x] No unlinked behavior implemented.
- [x] No Lexicon/PDS/Tap/blob change introduced.
- [x] GAP-001, GAP-002, and GAP-003 closed before dependent assertions.
- [x] High-risk implementation approval recorded before migration/auth/privacy changes.
- [x] Durable implementation plan created and initialized.
- [x] Implementation notes updated and read back.
- [x] IR-001 through IR-006 correction evidence records only assertions and commands that actually ran.
- [x] NFR-005 / IR-007 explicitly deferred without claiming IT-009 evidence.
- [ ] Implementation review completed or explicitly skipped.
