# TDD Implementation Plan: Flutter Business Profiles

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` (`Approved`)
- Coding plan: `04-coding-plan.md`

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update one focused failing test before implementation.
- Run the smallest relevant test first.
- Refactor only after tests pass.
- Keep traceability and red-green evidence updated after every stage.
- Do not alter lexicons, migrations, indexers, mutation routes, public eligibility, or unfiltered owner-event semantics.
- Do not compare or infer ordering from CID values.

## Test Order

### Implementation Review Corrections

| Step | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| R1 | UT-018, IT-011, AT-003, AT-009 | FR-004, FR-007, FR-018, RULE-005 | AC-011, AC-014, AC-028, AC-029 | Fails: Products lacks the regular-account redirect and owner setup actions target generic Settings |
| R2 | AT-011, IT-010, REG-008 | NFR-001 | AC-036 | Fails: a dirty Products draft is not registered with the production unsaved-work guard |
| R3 | UT-015, IT-013 | FR-022, NFR-001 | AC-034, AC-036 | Fails: projection read fences are captured after requests and mounted profile aliases remain stale |
| R4 | UT-014, UT-015, IT-007, IT-013 | FR-020, FR-022 | AC-031, AC-034 | Fails: accepted replacement uploads lose their local image during projection lag |
| R5 | AT-006, AT-007, AT-008, IT-008 | FR-014, FR-015, FR-016 | AC-005, AC-023, AC-025, AC-026 | Fails: unknown event values are not safely editable and conflict reload closes without retry |
| R6 | IT-002, AT-013 | FR-021 | AC-033 | Fails: shared post image JSON emits empty URL keys and malformed present alt is accepted |
| R7 | IT-003 | FR-014, FR-023 | AC-023, AC-041 | Fails: filtered real-Postgres traversal matrix omits tied starts, multi-page History, and changed-limit continuation |
| R8 | AT-012, REG-010 | NFR-003 | AC-038 | Fails: the required viewport/text-scale/semantics catalog covers only a subset of business surfaces |
| R9 | IT-010, REG-008 | NFR-001 | AC-036 | Fails: route/system Back bypasses the dirty Products discard decision |
| R10 | UT-015, IT-013 | FR-022, NFR-001 | AC-034, AC-036 | Fails: stale read rejection publishes an authoritative-absence state |
| R11 | AT-008, IT-008 | FR-016 | AC-026 | Fails: lifecycle conflict retry reapplies a stale complete draft onto the new CID |
| R12 | AT-012, REG-010 | NFR-003 | AC-038 | Fails: business Edit Profile and ordered keyboard traversal remain outside the accessibility matrix |
| R13 | IT-003 | FR-014, FR-023 | AC-023, AC-041 | Fails: equal-start URI tie-breakers do not straddle a filtered cursor boundary |
| R14 | UT-015, IT-013 | FR-022, NFR-001 | AC-034, AC-036 | Fails: record-stale list reconciliation publishes incomplete rows and an obsolete cursor after overlay settlement |

The R1-R8 loops were added after the implementation review in `06-implementation-review.md`. They supersede the previously checked final-completion claims until every correction and repository gate is green.

#### R1: UT-018, IT-011, AT-003, AT-009

- Confirmed failure: The new real-router suite showed regular Products deep links remained on `/profile/settings/products`, while both owner profile setup actions opened generic Settings.
- Implement: Added the same authoritative fail-closed redirect used by Events to `BusinessProductsRoute`; wired Products and Upcoming Events owner setup actions directly to their corresponding manager routes.
- Run command: `cd app && flutter test test/router/business_settings_routes_test.dart` passed 9 tests; the nearby router/settings/profile command passed 39 tests; focused analysis and `git diff --check` passed.
- Notes: Coverage now includes canonical locations, regular/business route admission, compact/wide real Settings-row navigation, and integrated owner profile setup actions.

#### R2: AT-011, IT-010, REG-008

- Confirmed failure: A production Products manager with a dirty controller registered no unsaved work, so account activation proceeded without the discard decision.
- Implement: Converted the manager shell to account-lease-aware state, registered dirty Products work with `unsavedWorkGuardProvider`, rejected stale lease callbacks, and cleared registration after save/clean transitions and disposal.
- Run command: The focused production guard test passed; nearby Products, guard, account-boundary, and account-switch suites passed 25 tests; `just app-analyze` and `git diff --check` passed.
- Notes: Account activation is covered directly. The shared guard remains the route/back integration point; no duplicate page-local navigation policy was added.

#### R3: UT-015, IT-013

- Confirmed failure: A detail read started before mutation completion was accepted as current when it returned afterward, and a mounted profile alias was not invalidated after Products save.
- Implement: Captured projection fences before repository awaits across profile/detail/public-list/owner-list reads, carried the exact fence through reconciliation/failure handling, and invalidated active handle/DID profile aliases after accepted Products writes.
- Run command: Both focused regressions passed; the affected reconciliation/Products/profile/list/detail/account-boundary run passed 41 tests; analysis and `git diff --check` passed.
- Notes: Old reads can no longer acquire a post-mutation generation after completion, and mounted profile aliases rebuild through the accepted declaration overlay.

#### R4: UT-014, UT-015, IT-007, IT-013

- Confirmed failure: Accepted product and event replacement/create projections had no local image, and product presentation fell back to network rendering rather than uploaded preview bytes.
- Implement: Added non-wire local preview bytes to hydrated image views, composed them from accepted upload drafts, and taught product/event card and detail presentation to render local memory images before AppView URLs.
- Run command: Focused model/controller/reconciliation/widget coverage passed 46 tests; the lagging profile integration regression, app analysis, and `git diff --check` passed.
- Notes: Mutation serialization remains blob/alt/aspect-only. Accepted-CID settlement and third-CID divergence replace previews with authoritative AppView images.

#### R5: AT-006, AT-007, AT-008, IT-008

- Confirmed failure: Unknown status/mode values triggered Flutter dropdown assertions, unknown roles were invisible, and conflict Reload closed the editor while discarding the authoritative result.
- Implement: Added bounded visible open-value choices and removable unknown-role chips while retaining first-party submission validation; conflict Reload now refreshes the editor baseline/current CID in place and permits retry.
- Run command: Focused event model/editor/mutation/manager coverage passed 35 tests; full app analysis and `git diff --check` passed.
- Notes: Independently authored values can be understood and corrected but cannot be newly submitted as unsupported first-party values.

#### R6: IT-002, AT-013

- Confirmed failure: JSON-level post coverage exposed newly emitted empty image keys, while present null/non-string independent-image `alt` values hydrated as safe empty strings.
- Implement: Restored ordinary post `omitempty` behavior, introduced an exact required-key `BusinessImageView` projection, and made independent-image alt extraction presence/type aware.
- Run command: Focused post/business API projection tests and the complete `internal/business` suite passed; `gofmt` and scoped `git diff --check` passed.
- Notes: Absent alt remains valid as authored empty text; present malformed alt omits only the unsafe image. Ordinary post production code has no net R6 diff.

#### R7: IT-003

- Confirmed failure: Pre-existing green; the expanded real-Postgres traversal matrix revealed no production defect.
- Implement: Added deterministic three-page Upcoming and History traversals with equal-start URI tie-breakers, changed second-page limits that retain another cursor, frozen independent cutoffs, exact directional order, and no duplicate/omission assertions.
- Run command: Focused traversals and full `go test ./internal/api -count=1` passed; `gofmt` and `git diff --check` passed.
- Notes: No production code changed in R7.

#### R8: AT-012, REG-010

- Confirmed failure: At 320x568 and text scale 2.0, delete confirmation and the event editor action overflowed; product reorder/remove actions lacked explicit semantic labels.
- Implement: Made destructive confirmation content scrollable, added localized button semantics to product actions, replaced the narrow event submit text action with a tooltip-labeled icon, and expanded the required business surface catalog.
- Run command: The catalog now has 28 matrix cases; 80 focused/nearby widget tests and 11 event-editor regressions passed; `flutter analyze` and `git diff --check` passed.
- Notes: Automated coverage now includes account selector, summary/actions, dynamic tabs, Products manager/editor, Events manager/editor, pagination/retry, detail/report, and delete confirmation. MAN-001 and MAN-002 remain manual-only.

#### R9: IT-010, REG-008

- Confirmed failure: A dirty Products route popped immediately on system Back without presenting the registered discard decision.
- Implement: Added owner-scoped `PopScope` handling that retains the dirty route on cancel, unregisters and performs one allowed pop on confirmation, and leaves clean Back unprompted.
- Run command: Focused Products/router/guard/account-boundary coverage passed 26 tests; the complete business settings route suite passed 10 tests; analysis and `git diff --check` passed.
- Notes: The allowed-pop rebuild avoids recursive or double navigation.

#### R10: UT-015, IT-013

- Confirmed failure: Stale detail success/error, public-list cursor, and owner-list cursor completions replaced current state because stale reconciliation was represented as ordinary null/absence.
- Implement: Added an explicit stale reconciliation result and made profile/detail/list success, not-found, and error paths retain accepted/current state and pagination when their pre-request fence is superseded.
- Run command: Focused reconciliation/profile/list/account-boundary coverage passed 35 tests; detail passed 13 and nearby profile providers passed 7; analysis and `git diff --check` passed.
- Notes: Legitimate authoritative absence remains distinct and still settles delete/create overlays correctly.

#### R11: AT-008, IT-008

- Confirmed failure: Lifecycle Retry submitted the stale event name and other complete-draft fields with only the newly loaded CID.
- Implement: Lifecycle conflict Retry now rebuilds from the authoritative event and applies only the intended status change; regular editor conflict retry retains its submitted draft behavior.
- Run command: Focused mutation, manager, and editor suites passed 29 tests; analysis and `git diff --check` passed.
- Notes: Authoritative name, dates, links, summary, venue, image, and new CID survive lifecycle retry.

#### R12: AT-012, REG-010

- Confirmed failure: Business Edit Profile discard confirmation overflowed by 197 pixels at 320x568 and text scale 2.0.
- Implement: Added the four-case business Edit Profile matrix, made shared dialog content scrollable, and added explicit keyboard order/activation assertions for Edit Profile, Products, and Events controls.
- Run command: Seven focused R12 tests and 78 accessibility/Edit Profile/nearby widget tests passed; `just app-analyze` and `git diff --check` passed.
- Notes: Real VoiceOver/TalkBack and physical desktop keyboard/device inspection remain MAN-001/MAN-002.

#### R13: IT-003

- Confirmed failure: Pre-existing green; forcing tied records across cursor boundaries revealed no production defect.
- Implement: Changed Upcoming limits to 2/1/4 and History to 3/1/3 so equal-start records straddle pages, with exact per-page URI order, continuation cursors after changed limits, complete traversal, and duplicate checks.
- Run command: Strengthened focused real-Postgres tests and full `go test ./internal/api -count=1` passed; `gofmt` and scoped `git diff --check` passed.
- Notes: No production code changed in R13.

#### R14: UT-015, IT-013

- Confirmed failure: Public and owner list requests that became record-stale after overlay settlement published empty rows instead of retaining their confirmed row/cursor.
- Implement: List reconciliation now reports aggregate stale status, and public/owner providers retain current rows and cursors atomically whenever any returned record is stale.
- Run command: Reconciliation/public/owner coverage passed 29 tests; account-boundary passed 6; full app analysis and `git diff --check` passed.
- Notes: Accepted, absent, delete, dedupe, ordering, and account-generation behavior remains covered.

| Step | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | IT-002, AT-013 | FR-021, NFR-005 | AC-032, AC-033, AC-040 | Fails: declaration CID absent and images use raw record shape |
| 2 | UT-007, IT-003, AT-006, REG-006 | BR-003, FR-014, FR-023 | AC-005, AC-023, AC-041 | Fails: owner endpoint has no filtered traversals |
| 3 | IT-014, REG-011 | FR-009, FR-012 | AC-018 | Fails: both replacement modes are not covered |
| 4 | UT-001, IT-001 | FR-001, FR-002, FR-009, FR-012, FR-015, FR-016, FR-019, FR-021, FR-023 | AC-001, AC-008, AC-017, AC-018, AC-032, AC-033 | Fails: Flutter has no business wire models or API client |
| 5 | UT-011, IT-004, AT-001 | BR-001, FR-002, FR-003, FR-004, RULE-002, RULE-005 | AC-001, AC-009, AC-010, AC-011 | Fails: no account-type controller or conditional settings |
| 6 | UT-002, UT-003, AT-002, AT-003, REG-001, REG-005 | BR-002, BR-004, FR-005, FR-006, FR-007, FR-013, RULE-001, RULE-003, RULE-004, NFR-002, NFR-004 | AC-002, AC-003, AC-006, AC-008, AC-012, AC-013, AC-014, AC-015, AC-022, AC-035, AC-037, AC-039 | Fails: profile has fixed ordinary-only presentation |
| 7 | UT-004, UT-005, IT-005, AT-004, REG-002 | FR-008, FR-009, FR-010, FR-012 | AC-016, AC-017, AC-018, AC-019 | Fails: editor supports one ordinary mutation only |
| 8 | UT-006, UT-014, IT-006, IT-007, AT-005 | BR-003, FR-011, FR-012, FR-020, FR-021 | AC-004, AC-018, AC-020, AC-021, AC-031, AC-033 | Fails: no product draft, controller, or manager |
| 9 | UT-010, IT-009, AT-009 | BR-004, FR-017, FR-018, FR-019, RULE-004 | AC-007, AC-028, AC-029, AC-030 | Fails: no public event state, tab, detail, or route |
| 10 | UT-008, UT-009, IT-008, AT-007 | FR-015, FR-016, FR-020 | AC-024, AC-025, AC-026, AC-027, AC-031 | Fails: no event draft, time conversion, API, or editor |
| 11 | UT-016, AT-006, AT-008 | FR-014, FR-016, FR-022, NFR-002 | AC-005, AC-023, AC-026, AC-027, AC-034, AC-037 | Fails: no owner event manager or diagnostics presentation |
| 12 | UT-015, IT-013 | FR-022, NFR-001 | AC-034, AC-036 | Fails: no CID-identity accepted-state overlays |
| 13 | AT-010, IT-009, IT-012 | FR-001, FR-019, RULE-005, NFR-004 | AC-008, AC-030, AC-039 | Fails: report flow has no event subject or unavailable detail |
| 14 | AT-011, IT-010, REG-008 | FR-003, NFR-001 | AC-010, AC-036 | Fails: business async state is not account-fenced |
| 15 | UT-012, UT-013, UT-017, AT-012, AT-014, REG-009, REG-010 | RULE-001, RULE-003, RULE-004, NFR-002, NFR-003, NFR-005 | AC-003, AC-012, AC-013, AC-022, AC-035, AC-037, AC-038, AC-040 | Fails: business localization, formatting, launcher, privacy, and layout contracts are absent |
| 16 | REG-003, REG-004, REG-007 | FR-004, FR-016, NFR-004 | AC-011, AC-027, AC-030, AC-039 | Existing baselines pass; business-aware regressions absent |

## Implementation Steps

### Step 1: IT-002, AT-013

- Write failing test: assert declaration CID and exact normalized product/event image JSON on every approved surface, including omission of unsafe images.
- Run command: `cd appview && go test ./internal/api/... -run 'Business(Profile|Event).*Image|BusinessProfile.*CID'`.
- Confirmed failure: Real-Postgres profile test lacked `business.cid` and returned raw lexicon images; direct, public-list, and owner-list event loops each failed because normalized source metadata lacked display URLs.
- Implement: Added validated `business.HydratedImage`, declaration CID selection, API-only profile/event response builders, and shared exact `PostImageView` projection with required zero-valued `size` support.
- Run command: `TEST_DATABASE_URL=... TEST_DATABASE_REQUIRED=true go test ./internal/api -count=1` and `go test ./internal/business -count=1` passed.
- Refactor: Centralized source-image validation and response conversion; kept `internal/business` independent of `internal/api`.
- Notes: Completed profile, event detail, public upcoming, and owner list as separate red-green loops. Existing unsafe profile/event hydration tests prove unsafe images become nil; API response `omitempty` then omits them.

### Step 2: UT-007, IT-003, AT-006, REG-006

- Write failing test: cover one owner filtered traversal before extending cursor/filter matrices.
- Run command: focused `go test` for the added filter test.
- Confirmed failure: The pure filter symbols were absent; `filter=upcoming` and `filter=history` returned the full descending list; filtered continuation used the new clock; invalid filters returned 200.
- Implement: Added owner partition types/classification, explicit upcoming/history store queries, exact filter admission, owner-filter cursor kinds with frozen cutoff, and kind-bound continuation.
- Run command: Focused classification/cursor tests and real-Postgres owner filter tests passed; full `internal/api` and `internal/business` suites passed.
- Refactor: Kept the original unfiltered query/cursor kind as the default branch and shared only response hydration/scanning.
- Notes: Upcoming includes scheduled/default-status records with `endsAt > cutoff` regardless of public suppression. History is the exact complement for that traversal cutoff. Unknown parameters remain ignored.

### Step 3: IT-014, REG-011

- Write failing test: add distinct detail-only and product-only replacement preservation cases.
- Run command: `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:16050/craftsky_dev?sslmode=disable' TEST_DATABASE_REQUIRED=true go test ./internal/business -run 'TestProfileReplacement(MergeAndSafeHydration|PreservesUnknownExtensionForCompleteKnownReplacements)$' -count=1` and `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:16050/craftsky_dev?sslmode=disable' TEST_DATABASE_REQUIRED=true go test ./internal/api -run 'TestBusinessProfile(ConditionalCreateReplaceAndDelete|CompleteKnownReplacementsPreserveUnknownExtension)$' -count=1` from `appview/`.
- Confirmed failure: Pre-existing green. Both new replacement modes passed on the first run, so no production defect was revealed.
- Implement: Added focused merge and handler coverage only; no production behavior changed.
- Run command: Both focused commands passed (`internal/business` in 0.413s; `internal/api` in 0.494s).
- Refactor: Updated the handler test executor to decode stored test records with `json.Decoder.UseNumber`, preventing the fake from introducing float64 precision loss.
- Notes: Detail-only and product-only complete-known replacements each preserve the other known area, every submitted known field, and a nested unknown top-level extension containing `9007199254740993`; an omitted known `serviceArea` is removed under replacement semantics. Existing lifecycle and merge regression tests remain green. Unknown top-level extension preservation remains AppView responsibility.

### Step 4: UT-001, IT-001

- Write failing test: decode a typed event page that reuses the exact normalized business image view, then add account/declaration and event route adapter tests as separate loops.
- Run command: `cd app && flutter test test/business/models/business_wire_models_test.dart`, followed by `flutter test test/business/data/business_api_client_test.dart` for each API loop.
- Confirmed failure: The event model test failed to compile because `business_event.dart` and `BusinessEventPageMapper` did not exist. The first API loop failed to compile because `BusinessApiClient` did not exist. The event API loop then failed to compile because the profile/owner list, detail, CRUD, report methods, and `OwnerEventFilter` did not exist.
- Implement: Added typed business event/page and mutation-result wire models; shared `BusinessImageView` reuse; typed `AtIdentifier`; exact account-type, declaration PUT, profile events, filtered owner events, event detail/CRUD, and event-report Dio operations; and the minimal repository interface/API delegate. Complete declaration/event write maps cross the API boundary unchanged, leaving draft construction and validation to later steps.
- Run command: `cd app && dart run build_runner build --delete-conflicting-outputs`; `dart format lib/business lib/shared/atproto/identifiers.dart lib/bootstrap.dart test/business`; `flutter test test/business/models/business_wire_models_test.dart test/business/data/business_api_client_test.dart`; focused Dart MCP analysis of the touched Flutter paths.
- Refactor: Kept all authenticated traffic on relative `/v1/` AppView routes, centralized standard Dio error unwrapping, retained opaque cursors, and used one `RecordMutationResult` for declaration and event responses without introducing providers or UI.
- Notes: Final focused result is 11 tests passed and no analyzer diagnostics. Repository-wide `git diff --check` remains blocked by pre-existing EOF blank-line changes in unrelated generated mapper files; the Step 4 path-only diff check passes. Read models and complete write maps remain separate, and no PDS URL/token is constructed.

### Step 5: UT-011, IT-004, AT-001

- Write failing test: account-type controller serializes one mutation, publishes the exact returned type, and retains the prior confirmed type on failure; Account and Settings widget tests cover pending/error behavior and authoritative conditional rows.
- Run command: `cd app && flutter test test/business/providers/account_type_controller_test.dart`, then `flutter test test/settings/account_page_test.dart test/settings/settings_page_test.dart`.
- Confirmed failure: The provider test failed to compile because the repository provider and account-type controller did not exist. The widget tests then failed because Account had no selector and Settings had no Business section.
- Implement: Added the generated business repository provider and lease-fenced account-type controller; reconciled loaded handle/DID profile aliases from the exact mutation response; invalidated active profile/settings reads; registered repository/controller account-boundary invalidation; added the localized pending-safe selector above deletion and authoritative Business > Events/Products rows.
- Run command: `cd app && flutter test test/business test/settings` passed 90 tests; focused `account_boundary_provider_test.dart` passed; `flutter analyze` reported no issues; Step 5 path-scoped `git diff --check` passed.
- Refactor: Replaced use of Riverpod's internal previous-state API with explicit confirmed-state restoration and page-owned localized failure feedback. Removed unrelated generated-file churn after required code generation.
- Notes: Switching to regular has no confirmation and the recording repository proves zero declaration replacement and zero event deletion. The immediate cached profile projection is cleared when regular, then aliases are refreshed. No Flutter event providers exist yet to invalidate; account-boundary and reconciliation hooks are ready for those later providers. Events/Products rows intentionally have no navigation callback in Step 5 because typed guarded owner routes and managers belong to later UT-018/IT-011/product/event steps.

### Step 6: UT-002, UT-003, AT-002, AT-003, REG-001, REG-005

- Write failing test: derive stable regular/business/blocked tab lists, then cover catalog labels, business summary/About presentation, product cards, empty/loading tab shells, and ordinary/customization regressions as separate loops.
- Run command: `cd app && flutter test test/profile/models/profile_tabs_test.dart`, followed by the focused business model/widget and profile business-tab tests.
- Confirmed failure: The tab policy test initially failed to compile because `ProfileTabPolicy` and the business tab identities did not exist. Subsequent widget loops failed because the profile had no business summary, hydrated About sections, product card/list, or Upcoming Events shell.
- Implement: Added the stable ordinary/business tab policy and logical selection remapping; localized business catalogs and bounded unknown fallback; plain Business summary, tagline, primary action, and hydrated About sections; ordered product cards with exact external URI handoff; owner/visitor product empties; and a provider-free Upcoming Events shell.
- Run command: `cd app && flutter test test/profile` passed 170 tests; `flutter test test/business` passed 17 tests; the final focused regression command passed 28 tests; `flutter analyze` reported no issues; repository-wide `git diff --check` passed.
- Refactor: Drove the full profile controller, labels, children, and storage keys from one logical tab list; preserved the compact profile card's five-tab controller contract; omitted absent business sections instead of adding placeholders.
- Notes: Business tabs remain present independent of product/event state. Blocked profiles initialize no tab body. Upcoming Events intentionally has no provider or event data in Step 6; those remain assigned to Step 9. Managers, editors, and settings navigation remain assigned to later steps.

### Step 7: UT-004, UT-005, IT-005, AT-004, REG-002

- Write failing test: complete declaration draft preserves products during detail-only change.
- Run command: `cd app && flutter test test/business/models/business_declaration_draft_test.dart`, followed by the save-result/provider and edit-dialog suites as separate vertical loops.
- Confirmed failure: The draft test failed to compile because `BusinessDeclarationDraft` did not exist. The provider loop then failed because `SaveProfile.save` accepted only one ordinary mutation and returned `void`. The business widget loop initially exposed an untouched declaration as dirty because the parent evaluated before nested FormBuilder fields registered.
- Implement: Added a complete known-field declaration draft with CID/create precondition state, open-value/product/image preservation, and mutation-only image serialization; explicit per-record combined outcomes with conflict classification; non-fail-fast concurrent ordinary/business dispatch through the existing repositories; per-section cache/baseline reconciliation; localized business fields and exact partial-failure feedback in the existing dialog.
- Run command: `cd app && dart run build_runner build --delete-conflicting-outputs`; `flutter gen-l10n`; `dart format ...`; focused Step 7 tests passed 25 tests; `flutter test test/profile test/business` passed 202 tests; `flutter analyze` reported no issues; `git diff --check` passed.
- Refactor: Kept business fields in one bounded widget, retained the existing dialog's image and unsaved-work ownership, and made initial business dirtiness wait for field registration. Successful portions become retry baselines before feedback or close decisions.
- Notes: No `Future.wait` is used. Ordinary-only, business-only, both, both partial-success directions, failed-only retry, complete product retention, unknown open-value retention, current CID/create state, conflict classification, regular editor regression, and close-on-full-success are covered. Full conflict reload/product UX remains later; no product or event manager was added.

### Step 8: UT-006, UT-014, IT-006, IT-007, AT-005

- Write failing test: validate ordered product drafts and the four-product maximum.
- Run command: focused product draft test.
- Confirmed failure: The product/image model suites failed to compile because no product draft or explicit image mutation states existed. The provider suite then failed because there was no products controller, and the page/editor suites failed because no management widgets or route existed.
- Implement: Added bounded ordered product drafts, exact-URI uniqueness, credential-free HTTPS/title/image/canonical-price validation, explicit missing/removed/existing/uploaded image states, and exact blob/alt/aspect mutation reconstruction. Added a current-CID complete-declaration products controller with active-business ownership checks, conflict blocking plus explicit complete-profile reload, and upload failure/cancel preservation. Added the localized responsive manager/editor, authenticated image preparation/upload path, loading/error/empty/conflict states, add/edit/remove/drag and semantic move controls, four-card cap, Settings row, and typed Products route.
- Run command: `cd app && dart run build_runner build --delete-conflicting-outputs`; `flutter gen-l10n`; `dart format ...`; focused Step 8 tests passed 25 tests; `flutter test test/business test/profile test/settings/settings_page_test.dart` passed 224 tests; `flutter analyze` reported no issues; Dart MCP analysis reported no errors; `git diff --check` passed.
- Refactor: Kept products as a mutation-draft layer over the existing complete declaration so every known non-product field remains intact, moved reorder handling to Flutter's adjusted `onReorderItem`, and avoided publishing stale read-model cards before the later CID-overlay step.
- Notes: Display URLs never enter declaration mutation JSON. Untouched source blob metadata, alt, and aspect round-trip exactly; failed/cancelled replacement leaves the saved image intact. A conflict cannot retry until Reload has fetched and displayed the exact owner's complete current declaration. Events and Step 12 projection overlays remain out of scope.

### Step 9: UT-010, IT-009, AT-009

- Write failing test: public upcoming state preserves server order and confirmed rows after incremental failure.
- Run command: focused event list provider test.
- Confirmed failure: The provider test failed to compile because the account/target-scoped public event provider and list state did not exist. The tab test then failed because the placeholder accepted no target or event rows, the detail test failed because no detail provider/page existed, and the route test failed because no typed event route existed.
- Implement: Added account/target-keyed public pagination and account/DID/rkey-keyed detail state; stable initial/retry/empty/loading-more/incremental-error/refresh-error/end states; record-identity dedupe preserving first-seen server order; unchanged opaque cursor forwarding; localized cards and complete responsive detail; normalized event images; exact hydrated external actions; safe `event_not_found`; and the authenticated typed `/events/:did/:rkey` route with exact DID/rkey validation and compact/wide back behavior.
- Run command: `flutter gen-l10n`; `dart run build_runner build --delete-conflicting-outputs`; `dart format ...`; focused Step 9 plus nearby profile/product/router regressions passed 62 tests; `flutter analyze` reported no issues; Dart MCP analysis reported no errors; `git diff --check` passed.
- Refactor: Centralized exact hydrated HTTPS/mailto validation and localized launcher failure handling, reused it for business summary, products, and event detail, and kept event-list state immutable while retaining confirmed rows and cursors across failures.
- Notes: Flutter renders every AppView-returned public event without lifecycle/status filtering. Owner event management/editing remains assigned to Steps 10-11, and event reporting remains assigned to Step 13; no placeholder owner/report actions were added to detail.

### Step 10: UT-008, UT-009, IT-008, AT-007

- Write failing test: convert one timed IANA-zone draft to whole-second UTC.
- Run command: focused event draft test.
- Confirmed failure: `flutter test test/business/models/event_draft_test.dart test/business/models/event_mutation_test.dart` failed because `BusinessEventDraft` and `BusinessTimeZoneService` did not exist; the controller and editor tracer tests then failed on their missing public interfaces.
- Implement: Added `timezone ^0.11.1`, bootstrap-initialized/injected IANA data, timed and exclusive all-day local-boundary conversion, whole-second UTC serialization, DST/range/duration/field validation, and complete writable create/update drafts with exact image reconstruction. Added typed event repository writes, current-CID create/update/delete orchestration, cancellation/postponement PUT edits, confirmed deletion, conflict reload, account fencing, and current profile/detail cache invalidation without Step 12 overlays. Added the localized responsive full-screen event form/editor with explicit IANA selection, required/optional fields, existing image upload path, and unsaved-work guard.
- Run command: `cd app && flutter gen-l10n`; `dart run build_runner build --delete-conflicting-outputs`; `dart format ...`; focused Step 10 suite passed 25 tests; `flutter test test/business` passed 68 tests; focused Dart MCP analysis reported no errors; `flutter analyze` passed after lint cleanup; `git diff --check` passed after generated-output normalization.
- Refactor: Kept UTC conversion behind injected `BusinessTimeZoneService`, confined raw mutation JSON to `ApiBusinessRepository`, and exposed the editor independently so Step 11 can open it without adding owner Upcoming/History management early.
- Notes: Create/update bodies omit `did`, `rkey`, `uri`, `cid`, `createdAt`, `past`, and diagnostics. Untouched images retain exact blob/alt/aspect data and never serialize hydrated URLs. Step 11 diagnostics/manager, Step 12 overlays, and Step 13 reporting were not implemented.

### Step 11: UT-016, AT-006, AT-008

- Write failing test: map the closed owner diagnostic catalog to localized bounded copy.
- Run command: `cd app && flutter test test/business/models/event_diagnostics_test.dart`, followed by focused owner-provider, manager, mutation, API, settings, and route suites.
- Confirmed failure: UT-016 failed to compile because the diagnostic catalog was absent. The owner-provider suite then failed to compile because `ownerBusinessEventsProvider(filter)` was absent. The manager suite failed because no Events settings page or route existed.
- Implement: Added the closed localized seven-code diagnostic catalog with duplicate/empty/unknown omission; account-fenced independent owner Upcoming/History provider instances with confirmed rows, opaque cursor, load-more/refresh state, incremental/refresh errors, refresh generation, identity-only dedupe, and one cursorless restart for `invalid_cursor`; added the Events manager with retained tab selection, owner cards, diagnostics, pagination/retry/refresh, create/edit/cancel/postpone/delete paths, destructive confirmation, conflict reload/retry using the newly fetched CID, and business-only route guard; successful event mutations now restart both owner traversals in addition to existing profile/detail reads.
- Run command: `flutter test test/business` passed 83 tests; focused business API tests passed 10 tests; focused Settings/router tests passed 9 tests; `flutter analyze` passed with no issues; Step 11 path-scoped `git diff --check` passed.
- Refactor: Reused `BusinessEventListState`, existing event editor/mutation controller/card, exact repository filters, and localized shared retry copy. Kept list admission/order server-authoritative and kept all accepted-CID overlay behavior deferred to Step 12.
- Notes: Upcoming and History retain independent provider state and server cutoffs. Flutter forwards `filter=upcoming|history` and opaque cursors exactly, never repartitions or status-filters rows, and dedupes only repeated DID/rkey identity. A lifecycle mutation restarts both potentially affected owner views so a moved/deleted event cannot remain in a stale loaded partition. `just app-test` reached 1,583 passing tests but was not a green gate: five earlier-scope link/profile tests failed before the command was terminated by the 120-second timeout (`faceted_text_actions_test.dart` x2, `profile_bio_test.dart`, `profile_tab_bar_business_test.dart` x2). Repository-wide `git diff --check` remains blocked only by previously documented blank EOF lines in two unrelated generated mapper files; the Step 11 path-scoped check is clean.

### Step 12: UT-015, IT-013

- Write failing test: retain an accepted update while reads return the exact pre-write CID.
- Run command: focused reconciliation provider test.
- Confirmed failure: `business_reconciliation_test.dart` failed to compile because the projection overlay state/provider did not exist.
- Implement: Added an account/owner/record-keyed accepted projection registry with opaque-CID identity transitions, per-record request and account-read generations, session-lease fencing, retry metadata, and confirmed explicit-discard decisions. Declaration/product and event mutations now install accepted upserts or delete tombstones before invalidating reads; profile declarations, product state, public and owner event lists, event detail, and combined profile saves reconcile through the registry. Event-list reconciliation replaces exact stale rows before identity dedupe and filters only accepted local movement, preventing a moved event from remaining in both owner partitions without re-filtering authoritative AppView rows.
- Run command: Focused UT-015/IT-013 reconciliation tests passed 10 tests; the final focused reconciliation/management/profile-save run passed 19 tests. Existing product, profile-save, public/owner event pagination, mutation, and detail suites passed, and `flutter test test/business` passed all 93 tests. `flutter analyze` passed with no issues. `dart run build_runner build --delete-conflicting-outputs` completed (the installed build runner warned that the flag is now ignored) and regenerated six outputs.
- Refactor: Centralized event-list overlay replacement/injection and failure annotation while retaining each provider's existing pagination and refresh generation controls.
- Notes: CIDs are compared only for equality and are never sorted or treated chronologically. Exact pre-write CID/creation absence retains accepted state; accepted CID/absence settles; any third CID adopts authoritative divergence. Delete tombstones hide only the exact deleted CID and accept different-CID recreation. Failures never time-discard overlays, explicit discard requires confirmation, and wrong account/session or superseded request/read generations cannot publish.

### Step 13: AT-010, IT-009, IT-012

- Write failing test: a non-owner event detail exposes the existing report reasons.
- Run command: `cd app && flutter test test/business/pages/event_detail_page_test.dart --plain-name 'AT-010 visitor reports a visible event with existing reasons'`, followed by focused not-found loops and the business/moderation suites.
- Confirmed failure: The visitor test found no Report action. After the first green slice, the report-time `event_not_found` test failed because the report route retained generic retry feedback over stale event detail instead of replacing it with unavailable.
- Implement: Added an account-keyed, active-operation-fenced `reportBusinessEventProvider` over the existing repository report route; event subject title/type in the unchanged shared reason/details form; non-owner Report detail action with exact owner DID/rkey submission; owner self-report suppression; and established pending, accepted, validation, and retryable failure behavior. Load/refresh/report `event_not_found` now makes detail unconditionally unavailable, dismisses the report route when necessary, and removes event, external, and report controls without exposing suppression data.
- Run command: `flutter gen-l10n`; `dart run build_runner build --delete-conflicting-outputs`; `dart format ...`; `flutter test test/business test/moderation` passed 110 tests; `just app-analyze` passed with no issues.
- Refactor: Reused `ReportSubjectSheet` and the existing accepted-report handler rather than duplicating reasons, validation, pending, success, or generic failure UI. Kept not-found handling bounded to the exact account/DID/rkey detail provider and retained standard API exception mapping for all other failures.
- Notes: Existing post/profile report-flow and subject-sheet regressions remain green. Repository-wide `git diff --check` remains blocked only by the previously documented blank EOF lines in `instagram_verification_provider.mapper.dart` and `post_summary.mapper.dart`; Step 13 path-scoped diff checks are clean. Step 14 account-boundary mega coverage and Step 15 quality/privacy sweep were not implemented.

### Step 14: AT-011, IT-010, REG-008

- Write failing test: a late account-A business completion cannot publish into account B.
- Run command: `cd app && flutter test test/business/providers/business_account_boundary_test.dart --plain-name 'AT-011 account boundary advances overlays before invalidation'`, followed by one focused test per read, mutation, and upload slice.
- Confirmed failure: The first test failed to compile because the overlay registry had no account-boundary generation advance. The controlled upload slice then failed behaviorally because Alice's late uploaded blob populated the still-mounted editor after Bob became active.
- Implement: Added a synchronous overlay boundary advance that clears accepted views, advances every known request/read generation, and removes request leases before provider invalidation. Account invalidation now covers the business repository, account type, product drafts, owner/public lists, details, event mutations, reports, and the existing combined profile save path while retaining every unrelated invalidation. Product and event upload callbacks now reject stale-account previews, results, and errors and clear stale progress. Controlled Alice/Bob tests cover profile/declaration/product/event owner/public list/detail/upload/mutation/report operations, unsaved draft decisions, CIDs, cursors, overlays, feedback state, requests, and navigation resets; switch-back loads only authoritative first-page/detail/profile/product state.
- Run command: `dart run build_runner build --delete-conflicting-outputs`; path formatting; `flutter test test/business test/profile/providers/save_profile_provider_test.dart` passed 106 tests; existing account boundary, activation, routing, and shell suites passed 21 tests; the final focused boundary/editor/guard run passed 20 tests; `flutter analyze` passed with no issues.
- Refactor: Kept one account-boundary operation on the existing overlay controller, reused the existing active session lease identity for upload fencing, and added `ProviderScope` to the pre-existing isolated product-editor test rather than weakening production ownership checks.
- Notes: Overlay/business generations advance before invalidations. The existing unsaved-work decision still completes before invalidation/activation, account Home reset occurs once per accepted switch, and unrelated account-scoped invalidations remain unchanged. One parallel pair of `flutter test` commands raced over Flutter's shared native-assets build and reported a missing `objective_c.dylib`; the same business command passed serially. No localization, privacy, accessibility, or layout sweep was performed.

### Step 15: UT-012, UT-013, UT-017, AT-012, AT-014, REG-009, REG-010

- Write failing test: external actions hand off only exact hydrated destinations and surface false/throw safely.
- Run command: `flutter test test/business/models/business_formatters_test.dart test/business/models/business_action_test.dart`, followed by focused localization, privacy, policy, accessibility, widget, and named regression tests.
- Confirmed failure: The first red run failed because no shared business action/formatter contracts existed. Follow-up red runs exposed locale-specific currency/time output, the duplicate all-day form/display label, missing generated product hint and busy semantics, absent localized launcher failure handling, isolated link tests without localization delegates, and profile business-tab tests waiting on a real repository-backed animated load.
- Implement: Added generated business display, validation, diagnostic, external-confirmation, and busy-state localization coverage; centralized locale-aware seller-authored money, timed/all-day exclusive-end date ranges, localized country/location, and known event catalog formatting without changing wire values. The shared outbound helper now validates exact hydrated HTTPS/mailto actions before confirmation, launches only after user approval, preserves the exact URI, and handles launcher false/throw with bounded localized feedback and no preview/fetch/rewrite. Added recording-Dio/launcher/observability privacy canaries, source-boundary checks, policy-neutrality assertions, semantics/focus/touch checks, and the exact 320x568/800x600 at 1.0/2.0 summary/About/editor matrix. Added localized live busy semantics to business page, list, save, and image-upload progress. The linked earlier-scope fixtures now install localization delegates and an empty business repository rather than reaching runtime infrastructure.
- Run command: focused Step 15 tests passed; named `faceted_text_actions_test.dart`, `profile_bio_test.dart`, and `profile_tab_bar_business_test.dart` passed 12 tests; `flutter test test/business test/profile test/settings test/router test/moderation test/l10n/business_profile_l10n_test.dart test/shared/rich_text/faceted_text_actions_test.dart` passed 514 tests; `go test ./internal/api ./internal/business` passed; Dart analysis reported no errors; `git diff --check` passed.
- Refactor: Kept canonical model/API values untouched, reused `l10n_countries`, `intl`, the existing launcher abstraction, AppView Dio path, and app messenger, and removed generator-only blank EOF churn from the two unrelated mapper files with `apply_patch`.
- Notes: Authored destination/email/free text/title/price/location/alt/DID/rkey canaries cause only the expected AppView detail request; no destination/PDS/preview request or business observability call exists, and bounded sink fixtures contain operation/result values only. Automated semantics cannot replace the MAN-001 VoiceOver/TalkBack and desktop keyboard pass, and OS chooser/no-handler behavior plus final visual inspection remain MAN-002.

### Step 16: REG-003, REG-004, REG-007

- Write failing test: assert the complete Settings row order for regular and business accounts, including conditional Business insertion only; then assert a blocked profile containing hidden business details/products remains the reduced shell and never calls the profile-event repository.
- Run command: `cd app && flutter test test/settings/settings_page_test.dart`, followed by `flutter test test/settings/settings_page_test.dart test/profile/widgets/profile_tab_bar_business_test.dart`.
- Confirmed failure: The first Settings run initially observed only the lazily built visible rows, so the fixture was corrected to a bounded tall viewport before evaluating order. The next red result showed the expected list omitted the existing `signOut` descriptor; adding it made the assertion protect sign-out rather than accidentally excluding it. With the assertion complete, both regular/business order and blocked no-fetch behavior passed without a production-code change.
- Implement: Added only REG-003 descriptor-order assertions and REG-004 hidden business detail/product/tab plus zero-event-fetch assertions. Existing Settings production routing tests and Account deletion test already prove reachability, while existing AppView event suites already cover REG-007, so no duplicate production or Go test code was added.
- Run command: `cd app && flutter test test/settings test/profile test/router` passed 376 tests. `TEST_DATABASE_URL='postgres://craftsky:dev@localhost:16050/craftsky_dev?sslmode=disable' TEST_DATABASE_REQUIRED=true go test ./internal/api ./internal/business -count=1` from `appview/` passed (`internal/api` 12.547s, `internal/business` 0.488s). `just app-analyze` passed with no issues.
- Refactor: Renamed the focused assertions to REG-003/REG-004 and kept the recording repository inside the existing profile-tab test fixture.
- Notes: REG-003 now fixes the exact existing row order and permits only Events/Products insertion between Discovery and General for business accounts; existing production-router cases cover every prior disclosure and `account_page_test.dart` covers permanent deletion. REG-004 proves the blocked reduced shell hides label, tagline, About details, products, both business tabs, and provider loading. The real-Postgres API/business run covers existing event CRUD/CID conflict, request and domain validation, report/moderation, blocked not-found, direct reads, public upcoming eligibility, and the unfiltered owner default/max/descending/opaque-cursor behavior. No production fix was required. Final repository gates subsequently passed as recorded below.

## Completion Checklist

- [x] All Must requirements covered by tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] `just test` passing with database-backed tests executed
- [x] `just app-test` passing
- [x] `just app-analyze` passing
- [x] `git diff --check` passing
- [x] MAN-001 completed or explicitly recorded as outstanding
- [x] No unlinked behavior implemented
- [x] Docs updated and read back
- [x] Review completed or explicitly skipped

## Execution Notes

- 2026-08-30: Loaded approved workflow documents and inspected the clean worktree. Stages 1-2 and 4-15 are absent; stage 3 and baseline regressions are partial. The approved first failing test remains IT-002 on the AppView business profile projection.
- 2026-08-30: Completed Step 1 with real-Postgres evidence. `internal/api` and `internal/business` are green after exact CID/image projection changes.
- 2026-08-30: Completed Step 2. Filtered owner traversals, admission, cutoff/cursor binding, and unfiltered regression coverage are green against real Postgres.
- 2026-08-30: Completed Step 3. Detail-only and product-only complete-known declaration replacements preserve the other known area and nested unknown extensions, including large integers, with real-Postgres handler/domain coverage.
- 2026-08-30: Completed Step 4. Typed Flutter event/page wire models and exact Dio/repository operations are green in 11 focused tests; focused analysis is clean. No provider or UI work was added.
- 2026-08-30: Completed Step 5. UT-011, IT-004, and the in-scope AT-001 Account/Settings scenarios are green; generated Riverpod/localization output is current, all business/settings tests pass, account-boundary regressions pass, and analysis is clean. Owner deep-link guards and profile business tabs remain assigned to later steps.
- 2026-08-30: Completed Step 6. UT-002, UT-003, AT-002, AT-003, REG-001, and REG-005 are green across the full profile and business suites; analysis and diff checks are clean. Event providers/data and all manager/editor flows remain deferred to their planned steps.
- 2026-08-30: Completed Step 7. UT-004, UT-005, IT-005, AT-004, and REG-002 are green. Complete declaration drafts preserve products and unknown open values; combined saves reconcile and retry per record without fail-fast behavior; active business accounts receive the bounded prefilled business section while regular editing remains unchanged. Focused tests, full profile/business suites, analysis, generation, formatting, and diff checks are clean.
- 2026-08-30: Completed Step 8. Product draft, image round-trip, declaration replacement, conflict reload, manager/editor, route, and settings coverage are green with complete known-field preservation and the four-product bound.
- 2026-08-30: Completed Step 9. UT-010, IT-009, and AT-009 are green for public upcoming pagination/state, profile cards, complete event detail, exact external actions, unavailable handling, and typed compact/wide routing. Owner management/editing/reporting remain deferred to their assigned steps.
- 2026-08-30: Completed Step 10. IANA/DST event drafting, exact create/update serialization, current-CID mutation handling, conflict state, image preservation/upload, editor validation, and unsaved-work coverage are green.
- 2026-08-30: Completed Step 11. UT-016, AT-006, and the Step 11 portion of AT-008 are green across diagnostics, independent owner pagination/restart state, management UI, lifecycle movement, deletion confirmation, settings routing, and exact filtered API requests. Step 12 CID overlays and Step 13 reporting remain unimplemented by design. Full analysis passes; the full Flutter test gate remains blocked by five earlier-scope failures and the command timeout recorded above.
- 2026-08-30: Completed Step 12. CID-identity accepted projection overlays, tombstones, failure retention, explicit discard, divergence handling, and account/request generation fencing are green across declarations, products, profile reads, event lists, and event detail.
- 2026-08-30: Completed Step 13. AT-010, the remaining event-report portion of IT-009, and IT-012 are green for visitor reporting, owner self-report suppression, shared moderation feedback, and load/refresh/report not-found replacement. Business plus moderation tests pass 110 tests and analysis is clean. Step 14 account-boundary mega coverage and the Step 15 quality sweep remain intentionally unimplemented.
- 2026-08-30: Completed Step 14. AT-011, IT-010, and REG-008 are green for controlled Alice/Bob reads, pagination, drafts, CIDs, overlays, uploads, mutations, reports, feedback, requests, and navigation. Business tests pass 106 tests, existing account-boundary/routing tests pass 21, and full Flutter analysis is clean. Step 15 remains intentionally unimplemented.
- 2026-08-30: Completed Step 15. UT-012, UT-013, UT-017, AT-012, AT-014, REG-009, and REG-010 are green for generated localization, locale-aware authored display, exact confirmed outbound handoff, privacy canaries, policy neutrality, busy/destructive semantics, keyboard/touch checks, and responsive constraints. The requested Flutter suites pass 514 tests, AppView API/business tests pass, analysis and diff checks are clean, and MAN-001/MAN-002 remain explicitly outstanding.
- 2026-08-30: Completed Step 16. REG-003 and REG-004 gained minimum explicit assertions for exact Settings preservation/conditional insertion and the blocked reduced shell/no-fetch boundary. The focused Settings/profile/router run passes 376 tests; the existing AppView API/business suites pass against required real Postgres and retain REG-007 CRUD, validation, moderation, direct/public, and unfiltered-owner behavior. Analysis and diff checks are clean; `just test` and `just app-test` were not run.
- 2026-08-30: Completed final automated verification. `just test` passed every AppView package with database-backed tests running against the required real PostgreSQL service; `just app-test` passed all 1,742 Flutter tests; `just app-analyze` reported no issues; and `git diff --check` passed. All Must requirements and planned automated tests are covered. MAN-001 (real VoiceOver/TalkBack and desktop keyboard traversal) and MAN-002 (operating-system external handoff/no-handler behavior and final device visual inspection) remain explicitly outstanding release checks. Implementation review has not yet been performed.
- 2026-08-30: Completed implementation review with `Changes required` in `06-implementation-review.md`, then completed correction loops R1-R8 for every Important finding: owner route/CTA behavior, Products unsaved-work protection, pre-request projection fences/profile aliases, accepted upload previews, unknown event editing/conflict retry, exact business versus existing post image JSON, complete filtered pagination evidence, and the full responsive/accessibility catalog. Removed unrelated generated churn and reconciled missing stage ledger entries.
- 2026-08-30: Completed post-review final verification. `just test` passed every AppView package with required real-Postgres coverage; `just app-test` passed all 1,785 Flutter tests; `just app-analyze` reported no issues; and `git diff --check` passed. MAN-001 and MAN-002 remain the only outstanding release checks. A fresh implementation review is the next recommended stage.
- 2026-08-30: A second implementation review retained `Changes required` for IR-015 through IR-019. Completed correction loops R9-R13: dirty Products route Back now shares the discard guard; stale projection reads publish no absence/old state; lifecycle conflict Retry merges only status onto the authoritative event; business Edit Profile and ordered keyboard activation are in the accessibility matrix; and equal-start records straddle filtered cursor boundaries.
- 2026-08-30: Completed second post-review verification. `just test` passed every AppView package with required real-Postgres coverage; `just app-test` passed all 1,798 Flutter tests; `just app-analyze` reported no issues; and `git diff --check` passed. MAN-001 and MAN-002 remain the only outstanding release checks. A fresh implementation review is recommended.
- 2026-08-30: A third focused implementation review closed IR-015 through IR-019 but found IR-020, where record-generation staleness after overlay settlement could publish an incomplete event list and obsolete cursor. Completed R14 by returning aggregate stale status from list reconciliation and retaining current public/owner rows and cursors atomically.
- 2026-08-30: Completed R14 final verification. `just test` passed every AppView package with required real-Postgres coverage; `just app-test` passed all 1,801 Flutter tests; `just app-analyze` reported no issues; and `git diff --check` passed. MAN-001 and MAN-002 remain the only outstanding release checks.
- 2026-08-31: Completed R15 for the runtime `!semantics.parentDataDirty` assertion when account-type changes replaced the five-tab profile composition with seven tabs or vice versa. A semantics-enabled widget regression reproduced the assertion before the fix. The profile now retains and remaps logical selection in an outer state while a keyed inner state owns each fixed-length `TabController`, allowing Flutter to replace the controller and tab render subtree atomically. The focused regression covers both transition directions, retained `Comments` selection, and `Products` fallback to `About`; `just app-test` passed all 1,802 Flutter tests, `just app-analyze` reported no issues, `git diff --check` passed, and the final hot reload reported no runtime errors.
