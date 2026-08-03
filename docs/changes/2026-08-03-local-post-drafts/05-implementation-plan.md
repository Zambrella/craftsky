# TDD Implementation Plan: Local Post Drafts And Submit-Time Media Uploads

## Inputs

- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Document review: `03-document-review.md` — Approved
- Coding plan: `04-coding-plan.md`

## Implementation Rules

- Do not implement behavior without a linked requirement ID.
- Write or update one focused failing test before its implementation.
- Run the smallest relevant test first and confirm a meaningful red result.
- Refactor only after the focused and nearby tests pass.
- Keep traceability and actual command outcomes updated in this document.
- Preserve the existing AppView routes, database, worker, and lexicon surfaces.
- Do not upload or stage media before an explicit Post, Reply, Publish now, or Schedule action.
- Keep local draft content, identifiers, paths, and media out of diagnostics.

## Test Order

The steps mirror section 9 of `04-coding-plan.md`. Within a step containing several IDs, execute the IDs one at a time in the listed order and record each red-green result before moving on.

| Step | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|
| 1 | AT-009, UT-008, IT-010 | BR-003, FR-010, FR-021, RULE-001 | AC-009, AC-025 | Fails because image selection still uploads eagerly and has no local-ready phase |
| 2 | AT-001, AT-021, UT-001 | BR-001, FR-001, RULE-002, RULE-004 | AC-001, AC-021 | Fails because no reusable draft eligibility policy exists |
| 3 | AT-004, AT-015, UT-002, UT-003, UT-015 | FR-002, FR-003, FR-015, NFR-002 | AC-004, AC-006, AC-015, AC-019, AC-025 | Fails because draft domain, manifest codec, media validation, and redacted diagnostics do not exist |
| 4 | AT-007, AT-016, AT-017, UT-006, UT-007, IT-002, IT-003 | FR-006, FR-009, FR-015, FR-016, NFR-001 | AC-007, AC-014–AC-017 | Fails because immutable update planning and fault-safe storage behavior do not exist |
| 5 | AT-002, AT-008, AT-015, AT-023, IT-001, IT-004, IT-005 | BR-002, FR-003, FR-007, FR-014–FR-017, FR-020, RULE-001, RULE-006 | AC-002, AC-004, AC-008, AC-015, AC-016, AC-018, AC-023 | Fails because the account-scoped file repository and retention boundary do not exist |
| 6 | AT-005, AT-008, AT-018, AT-020, UT-004, UT-014, UT-016, UT-017, IT-006, IT-015, IT-018 | FR-004, FR-007, FR-014, NFR-004, RULE-007 | AC-005, AC-007, AC-008, AC-018, AC-020 | Fails because row projection, account-bound providers, async states, and Drafts page do not exist |
| 7 | AT-005, IT-007 | FR-004, FR-017 | AC-005 | Fails because Settings and the typed Drafts route do not exist |
| 8 | AT-002, AT-003, AT-004, AT-006, AT-024, UT-005, IT-008, IT-009 | BR-001, FR-001, FR-002, FR-005, FR-019, RULE-004 | AC-002–AC-004, AC-006, AC-024 | Fails because standard/project draft snapshots cannot hydrate and schedule intent cannot be restored |
| 9 | AT-003, AT-025, IT-008, IT-009 | FR-003, FR-010, FR-018, FR-021, RULE-003 | AC-003, AC-025 | Fails because eligible composers have no explicit save or three-choice close lifecycle |
| 10 | AT-010, AT-011, UT-009, IT-017 | BR-003, FR-011, FR-012 | AC-010, AC-011 | Fails because target materialization and scheduled client cancellation seams do not exist |
| 11 | AT-026, UT-010, UT-011 | FR-009, FR-011, FR-012, FR-023 | AC-014, AC-026 | Fails because transient immediate reference reuse and independent image timeouts do not exist |
| 12 | AT-010, AT-011, AT-012, AT-022, AT-026, IT-011, IT-012 | BR-003, FR-008, FR-009, FR-011, FR-012, FR-023, RULE-005 | AC-010–AC-012, AC-014, AC-022, AC-026 | Fails because submission remains duplicated in widgets rather than coordinated |
| 13 | AT-013, AT-014, UT-012, IT-013 | BR-004, FR-009, FR-013, NFR-003, NFR-005 | AC-013, AC-014, AC-026 | Fails because no common full-screen overlay or screen-awake lifecycle exists |
| 14 | AT-012, AT-027, UT-013, IT-014 | FR-008, FR-022, RULE-003, RULE-004 | AC-012, AC-027 | Fails because existing-draft snapshot-before-network and new-origin policy do not exist |
| 15 | AT-018, UT-016, IT-015 | FR-014 | AC-018 | Fails because the new local/remote operations are not yet fenced to captured account leases |
| 16 | AT-019, UT-015, IT-016 | BR-002, NFR-002 | AC-019 | Fails until all new models, errors, diagnostics, and failure paths pass privacy canaries |
| 17 | REG-001–REG-010 | All linked requirements | All linked criteria | Existing regression suites initially encode eager upload and inline submission assumptions |

## Implementation Steps

### Step 1: AT-009 / UT-008 / IT-010

- Write failing test: make `composer_images_provider_test.dart` require a locally-ready prepared image and zero blob calls after selection; update state tests for readiness/save gating.
- Run command: `just app-test test/feed/providers/composer_images_provider_test.dart`
- Confirmed failure: `ImageReady` was undefined, proving the provider exposed only eager-upload completion. The first wrapper invocation also split a quoted plain name; subsequent single-test runs use `flutter test` directly from `app/`.
- Implement: remove eager upload ownership from `ComposerImages`, retain prepared bytes/dimensions/digest, and expose local readiness.
- Run command: same focused test, then nearby media/state tests.
- Refactor: only after green.
- Notes: Green. Selection now finishes as `ImageReady` with prepared app-owned bytes, MIME, dimensions, and SHA-256; `PostApiClient.uploadImage` receives zero calls. `canSaveDraftMedia` is true only for all-ready attachments. Obsolete selection-owned upload progress/retry tests were removed; those behaviors move to steps 10–12. Focused provider/state/media group passed (26 tests).

### Step 2: AT-001 / AT-021 / UT-001

- Write failing test: add the eligibility matrix for standard/project versus quote/reply and meaningful versus default-only state.
- Run command: `just app-test test/drafts/composer/draft_save_eligibility_test.dart`
- Confirmed failure: the test could not import `draft_save_eligibility.dart` and all intended policy types were undefined.
- Implement: add the pure draft eligibility policy.
- Run command: same focused test.
- Refactor: only after green.
- Notes: Green. The pure policy permits only meaningful top-level standard/project state, treats each approved edit category as meaningful, excludes quote/reply, and blocks saving until all retained media is locally ready. Focused file passed (2 tests).

### Step 3: AT-004 / AT-015 / UT-002 / UT-003 / UT-015

- Write failing tests: introduce manifest round-trip first, then media descriptor validation, then privacy-safe model/error output one ID at a time.
- Run command: focused files under `test/drafts/models` and `test/drafts/privacy`.
- Confirmed failure: the first test could not import any draft model/codec types; subsequent tests exposed ignored future schema versions, missing descriptor validation, accepted traversal references, duplicate media identities, and unvalidated writes.
- Implement: add draft domain objects, explicit v1 codec, descriptor validation, and content-free error types.
- Run command: each focused test, then the small draft-model group.
- Refactor: only after green.
- Notes: Green. Added an explicit hand-written schema-v1 codec for standard and incomplete whitelisted project content, schedule intent, timestamps, account ownership, and ordered immutable-media metadata. Unknown fields are tolerated, future/corrupt input maps to typed content-free errors, unsafe media is rejected on read and write, and model diagnostics redact authored/account/file values. Draft test group passed (10 tests).

### Step 4: AT-007 / AT-016 / AT-017 / UT-006 / UT-007 / IT-002 / IT-003

- Write failing tests: start with pure update planning, then safe error mapping, then inject one filesystem boundary at a time.
- Run command: focused update-plan, storage-error, recovery, and failure test files.
- Confirmed failure: immutable update planning and the file-store boundary were absent; injected manifest-switch and cleanup failures exposed that a partially written replacement could otherwise displace the last valid bundle.
- Implement: add file-store abstraction, immutable update plan, atomic manifest switch, rollback, and cleanup ordering.
- Run command: each focused test and the repository failure group.
- Refactor: only after green.
- Notes: Green. Updates write immutable media, switch the manifest atomically, preserve the previous readable version until commit, and recover or roll back after injected write, rename, and cleanup failures.

### Step 5: AT-002 / AT-008 / AT-015 / AT-023 / IT-001 / IT-004 / IT-005

- Write failing tests: save/restart/read prepared bytes, idempotent delete, damaged-bundle recovery, account namespaces, and sign-out retention one behavior at a time.
- Run command: focused file repository, recovery, and retention tests.
- Confirmed failure: no account-scoped repository could persist, restart, reconcile, or delete a complete draft bundle.
- Implement: complete `FileLocalPostDraftRepository`, reconciliation, account namespaces, read/delete, and retention behavior.
- Run command: repository test group.
- Refactor: only after green.
- Notes: Green. `FileLocalPostDraftRepository` stores versioned JSON manifests and app-owned media beneath an account namespace, reopens prepared bytes after restart, reconciles incomplete bundles, deletes idempotently, and deliberately retains drafts across sign-out and time.

### Step 6: AT-005 / AT-008 / AT-018 / AT-020 / UT-004 / UT-014 / UT-016 / UT-017 / IT-006 / IT-015 / IT-018

- Write failing tests: row sort/projection, provider list/delete/error states, account fencing, async progress, then Drafts page behavior.
- Run command: focused row/provider/page/async tests.
- Confirmed failure: draft list projections, account-bound async controllers, and the management page did not exist.
- Implement: add repository providers, list/save controllers, row/thumbnail widgets, and Drafts page.
- Run command: draft provider/page group.
- Refactor: only after green.
- Notes: Green. Added account-scoped repository providers, sorted list/save/delete controllers, damaged-draft rows, thumbnails, loading/error/progress states, and the Drafts management page. Captured account leases prevent stale completions from mutating the newly active account.

### Step 7: AT-005 / IT-007

- Write failing test: Settings exposes Drafts and `/profile/settings/drafts` opens on the expected navigator.
- Run command: `just app-test test/settings/settings_page_test.dart test/router/settings_routes_test.dart`
- Confirmed failure: Settings had no Drafts row and the typed `/profile/settings/drafts` destination was missing.
- Implement: add localized Settings row, typed route, and generated routing/localization output.
- Run command: same focused tests.
- Refactor: only after green.
- Notes: Green. Settings now exposes Drafts without a count badge and opens the full-screen typed Drafts route; routing and localization output were regenerated.

### Step 8: AT-002 / AT-003 / AT-004 / AT-006 / AT-024 / UT-005 / IT-008 / IT-009

- Write failing tests: schedule restoration first, then standard snapshot hydration, then project partial-field hydration.
- Run command: focused schedule restoration and standard/project draft composer tests.
- Confirmed failure: standard and project composer state had no persisted snapshot adapters or origin-aware hydration, and saved schedule intent could not be restored.
- Implement: add snapshot adapters, hydrator, draft seeds/origins, and invalid-Later explanation.
- Run command: focused hydration/composer tests.
- Refactor: only after green.
- Notes: Green. Both composer types hydrate supported text, media, project fields, and valid schedule intent. Expired or otherwise invalid `Later` intent resets to the safe immediate option with an explanation. Project snapshots use the approved field whitelist.

### Step 9: AT-003 / AT-025 / IT-008 / IT-009

- Write failing tests: new/existing close choices, local readiness gating, save success, and save failure one behavior at a time.
- Run command: focused standard/project discard and draft composer tests.
- Confirmed failure: existing composer tests encoded a two-choice discard dialog and there was no explicit save action or save-progress lifecycle.
- Implement: add explicit save actions, save controller integration, common close dialog, and save feedback.
- Run command: focused widget tests.
- Refactor: only after green.
- Notes: Green. Eligible standard/project composers provide Save draft or Save changes, use the common Save/Discard/Keep editing close dialog, block save until retained media is locally ready, and preserve editable state when a save fails.

### Step 10: AT-010 / AT-011 / UT-009 / IT-017

- Write failing tests: target selection without side effects, then immediate/scheduled client wire contract tests.
- Run command: focused materialization and API-client tests.
- Confirmed failure: immediate materialization still depended on selection-time uploads and scheduled staging had no caller cancellation seam.
- Implement: add materialization plan and thread cancellation through existing scheduled staging APIs without changing wire shape.
- Run command: same focused tests.
- Refactor: only after green.
- Notes: Green. Immediate submission uploads prepared media only after submit; scheduled submission stages the same prepared media only after Schedule. Existing scheduled wire shapes are unchanged and staging accepts a cancellation token.

### Step 11: AT-026 / UT-010 / UT-011

- Write failing tests: same-composer digest-keyed reuse/invalidation, then independent 60-second transfer budgets.
- Run command: focused retry-state and timeout tests.
- Confirmed failure: no digest-keyed retry ledger existed and uploads did not receive an independent one-minute budget per actual transfer.
- Implement: add transient retry ledger and media uploader timeout primitive.
- Run command: same focused tests.
- Refactor: only after green.
- Notes: Green. The composer-local retry ledger reuses successful unchanged blob references, invalidates changed work, preserves display order, and gives every upload or staging transfer its own one-minute timeout.

### Step 12: AT-010 / AT-011 / AT-012 / AT-022 / AT-026 / IT-011 / IT-012

- Write failing tests: immediate ordering/retry first, then scheduled staging/create-update/capacity behavior.
- Run command: focused coordinator and scheduled submission tests.
- Confirmed failure: focused coordinator tests could not import a shared submission coordinator; standard/project widgets still owned duplicated overlay, lifecycle, origin, and cleanup sequencing.
- Implement: add shared submission controller and adapt immediate/scheduled materializers.
- Run command: focused coordinator/scheduled tests.
- Refactor: remove duplicated widget submission logic only while green.
- Notes: Green. A shared coordinator now owns presentation gating, screen-awake lifecycle, originating-draft snapshot, submission execution, success-only cleanup, failure routing, and running state for both composer types. Immediate and scheduled materializers remain separate because their network contracts intentionally differ.

### Step 13: AT-013 / AT-014 / UT-012 / IT-013

- Write failing tests: overlay state lifecycle, presentation-before-work, exact copy/semantics/blocking, then terminal wakelock cleanup.
- Run command: focused overlay state/widget tests.
- Confirmed failure: no common blocking overlay or testable screen-awake lifecycle existed; a disposal test initially showed that route disposal during submission needed an explicit release path.
- Implement: add screen-awake adapter, blocking overlay, presentation gate, and controller lifecycle cleanup.
- Run command: focused overlay and composer tests.
- Refactor: only after green.
- Notes: Green. The common full-screen overlay has exact localized copy, spinner, modal barrier, pop prevention, and semantics. Android/iOS screen-awake acquisition occurs after overlay presentation and releases once on success, failure, cancellation, or owner disposal.

### Step 14: AT-012 / AT-027 / UT-013 / IT-014

- Write failing tests: origin policy, exact save-before-network ordering, success-only delete, and never-saved no-write behavior.
- Run command: focused draft submission policy/lifecycle tests.
- Confirmed failure: submission orchestration had no origin policy or enforceable snapshot-before-network/success-only-delete ordering.
- Implement: integrate originating-draft snapshot and cleanup into the coordinator.
- Run command: same focused tests.
- Refactor: only after green.
- Notes: Green. Existing drafts are overwritten with an exact current snapshot before any network work and deleted only after complete success. Composers without a saved origin do not create a recovery draft during submission.

### Step 15: AT-018 / UT-016 / IT-015

- Write failing tests: switch/sign-out at each delayed local/remote completion boundary.
- Run command: focused account-boundary and account-switch routing tests.
- Confirmed failure: new repository and submission completion paths were not all bound to the account lease captured when the operation began.
- Implement: complete captured-lease checks and stale success suppression.
- Run command: same focused tests.
- Refactor: only after green.
- Notes: Green. Repository providers, management mutations, and both composer submission flows check the captured active-account lease before surfacing success or applying follow-up state.

### Step 16: AT-019 / UT-015 / IT-016

- Write failing test: place privacy canaries in every new model, path, error, event, and submission failure recorder.
- Run command: `just app-test test/drafts/privacy/draft_privacy_test.dart`
- Confirmed failure: the initial `ComposerImageDraft.toString()` exposed authored media detail to diagnostics.
- Implement: remove/redact any remaining sensitive diagnostics without hiding actionable coarse state.
- Run command: same focused test, then draft suite.
- Refactor: only after green.
- Notes: Green. New draft models, paths, typed errors, and submission diagnostics expose only coarse, content-free state; privacy canaries cover authored text, account identifiers, paths, names, and media bytes.

### Step 17: REG-001–REG-010

- Write/update failing regression expectations one existing suite at a time where eager-upload assumptions changed.
- Run command: focused composer, project, scheduled, Settings, router, media, and API suites.
- Confirmed failure: 12 existing tests initially expected eager-upload states, the previous close dialog, or inline submission timing.
- Implement: compatibility fixes and generated output only; no new behavior.
- Run command: `just app-test`, `just app-analyze`, generation, formatter, and `git diff --check`.
- Refactor: only while the complete relevant suite is green.
- Notes: Green. Existing composer, project, scheduled-post, Settings, router, media, and API expectations were updated only where the approved behavior changed. The complete Flutter suite passes.

## Verification Log

| Step | Command | Result |
|---|---|---|
| Setup | Contract and package documentation review | Passed; no blocking gaps |
| 1 | `flutter test test/feed/providers/composer_images_provider_test.dart --plain-name "prepares accepted images locally without uploading"` | Red: missing `ImageReady`; green after local-only pipeline change |
| 1 | `flutter test test/feed/providers/composer_image_state_test.dart --plain-name "requires every retained image to be locally ready"` | Red: missing `canSaveDraftMedia`; green after readiness policy |
| 1 | `flutter test test/feed/providers/composer_images_provider_test.dart test/feed/providers/composer_image_state_test.dart test/feed/media/composer_image_media_service_test.dart` | Passed: 26 tests |
| 2 | `flutter test test/drafts/composer/draft_save_eligibility_test.dart` | Red: missing eligibility policy; green after implementation, 2 tests passed |
| 3 | `flutter test test/drafts/models/draft_manifest_codec_test.dart --plain-name "round-trips every version 1 standard draft field"` | Red: missing draft model and codec; green after explicit v1 schema implementation |
| 3 | `flutter test test/drafts/models/draft_media_descriptor_test.dart --plain-name "rejects unsafe or inconsistent persisted media metadata"` | Red: missing validation; green after bounded leaf-path/MIME/digest/dimension/ID checks |
| 3 | `flutter test test/drafts` | Passed: 10 tests including project round-trip, future-version handling, duplicate rejection, and privacy canaries |
| 4–6 | `flutter test test/drafts` | Passed: 48 draft model, repository, provider, controller, page, recovery, account-boundary, and privacy tests |
| 7–9 | Focused Settings, router, standard composer, project composer, hydration, schedule-restoration, and close-dialog tests | Passed |
| 10–11 | Focused immediate uploader and scheduled staging/timeout tests | Passed; 3 uploader tests cover order, retry reuse, and independent timeout tokens |
| 12–14 | `flutter test test/feed/composer/composer_submission_coordinator_test.dart test/feed/composer/submission_screen_awake_test.dart test/feed/composer/submission_overlay_test.dart test/feed/widgets/standard_post_composer_test.dart test/projects/project_composer_test.dart test/scheduled_posts/scheduled_submission_test.dart` | Red: shared coordinator missing and disposal release absent; green: 22 tests passed |
| Generation | `dart run build_runner build` | Passed; 136 outputs written |
| Generation | `flutter gen-l10n` | Passed |
| Static analysis | `just app-analyze` | Passed; no issues found |
| Full regression | `just app-test --reporter compact` | Passed; 1,254 tests |
| Diff hygiene | `git diff --check` | Passed |

## Remaining Manual Release Checks

- `MAN-001`: physical iOS/Android persistence, source loss, restart, and process interruption.
- `MAN-002`: physical-device screen-awake behavior and release on every terminal path.
- `MAN-003`: themes, large text, accessibility semantics, and maximum-media responsiveness.
- `MAN-004`: sign-out retention, explicit deletion, time retention, and app-data removal.

## Completion Checklist

- [x] All Must requirements covered by passing tests or documented gaps
- [x] All planned Must tests passing
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Generated providers, routes, mappers, and localizations updated
- [x] `just app-test` passing
- [x] `just app-analyze` passing
- [x] `git diff --check` passing
- [x] Implementation notes updated and read back
- [x] Manual release checks remain explicitly handed off
- [ ] Implementation review completed or explicitly skipped
