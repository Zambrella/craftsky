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

## Implementation Review Correction Pass

Correction source: `06-implementation-review.md` — Changes required on 2026-08-04.

The correction pass uses one focused red-green loop per review finding. The order follows risk: account ownership, recoverability, retry safety, transfer validation, eligibility, filesystem containment, then traceability completion.

| Step | Review Finding | Test IDs | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|---|
| C1 | IR-001 | IT-015, AT-018 | FR-014 | AC-018 | Fails because immediate upload resolves the globally active client after submission starts |
| C2 | IR-002 | IT-004, AT-015 | FR-015, FR-016 | AC-015 | Fails because a readable draft with unavailable media cannot open and has no Replace action |
| C3 | IR-003 | IT-014, AT-027 | FR-009, FR-022, FR-023 | AC-026, AC-027 | Fails because retry reuses the origin seed's stale expected revision |
| C4 | IR-004 | IT-011, IT-012 | FR-010, FR-011, FR-012 | AC-010, AC-011 | Fails because immediate/scheduled transfer trusts stale prepared metadata without revalidation |
| C5 | IR-005 | AT-021, REG-007 | RULE-002, RULE-003 | AC-021 | Fails because ineligible origins render a disabled local-draft action |
| C6 | IR-006 | IT-004 | FR-015, NFR-002 | AC-015, AC-019 | Fails because lexical path validation does not reject filesystem links escaping the draft root |
| C7 | IR-007 | IT-005, IT-008–IT-010, IT-014–IT-018 | Linked Must requirements | Linked criteria | Fails because the original record claims integration coverage that does not exist |

### Correction Step C1: IR-001 / IT-015 / AT-018

- Write failing tests: add stale-ownership coordinator and sequential-upload cases, then a real session-registry create-provider boundary.
- Run command: `flutter test test/feed/composer/composer_submission_coordinator_test.dart`; `flutter test test/feed/composer/composer_media_uploader_test.dart`; focused create/composer widget tests.
- Confirmed failure: coordinator compilation rejected the missing ownership callback; uploader compilation exposed constructor-owned global upload resolution.
- Implement: bind immediate upload/create orchestration to the lease captured before overlay presentation and check it at delayed boundaries.
- Run command: focused coordinator, uploader, create-provider, standard-composer, and project-composer tests.
- Notes: Green. Both composers capture the active lease and upload client before the blocking operation, upload checks ownership before each transfer and after completion, and create rejects a stale captured lease before repository work. Isolated widget fixtures remain supported only when no upload client is needed; production authenticated flows always carry a lease.

### Correction Step C2: IR-002 / IT-004 / AT-015

- Write failing test: require unavailable draft media to be replaced in place while retaining media identity and alt text.
- Run command: `flutter test test/feed/providers/composer_images_provider_test.dart --plain-name 'replaces unavailable draft media in place and preserves alt text'`.
- Confirmed failure: `ComposerImages.replaceUnavailable` did not exist; review evidence also showed `DraftsPage._openDraft` returned early for a readable manifest whose media was damaged.
- Implement: open readable damaged drafts, hydrate unavailable attachments, and support stable-identity replacement.
- Run command: focused replacement test, `test/drafts/drafts_page_test.dart`, then the complete draft/composer correction group.
- Notes: Green. Only revision-zero unreadable manifests are blocked. Readable damaged drafts reopen with aspect ratio, order, identity, and alt text intact, and unavailable attachments expose Replace image.

### Correction Step C3: IR-003 / IT-014 / AT-027

- Write failing test: add `draft_submission_origin_test.dart` requiring successive accepted snapshots to advance revision and reject identity changes.
- Run command: `flutter test test/drafts/composer/draft_submission_origin_test.dart`.
- Confirmed failure: the shared revision tracker did not exist; the reviewed composer code retained the immutable seed revision after a successful snapshot, so an unchanged retry would conflict.
- Implement: retain the latest origin revision after every exact pre-network snapshot.
- Run command: focused origin test plus standard/project composer lifecycle tests.
- Notes: Green. Both composers now use `DraftSubmissionOrigin`; the exact saved snapshot becomes the next authoritative expected revision and the latest origin is deleted only after confirmed success. Create returns its local result so listener reset cannot erase success detection.

### Correction Step C4: IR-004 / IT-011 / IT-012

- Write failing tests: mutate an `ImageReady` byte buffer after preparation for immediate upload and scheduled staging.
- Run command: focused mutation cases in `composer_media_uploader_test.dart` and `scheduled_post_edit_test.dart`.
- Confirmed failure: immediate retry emitted the cached reference after mutation instead of throwing; scheduled staging likewise had no digest verification.
- Implement: revalidate current prepared bytes and derive retry identity from verified bytes immediately before transfer.
- Run command: complete uploader tests plus the scheduled mutation case and scheduled-edit suite.
- Notes: Green. Both paths verify current digest, size, MIME, dimensions, and alt-text bounds at the explicit transfer boundary. Immediate reuse keys use the verified digest; mismatched or invalid bytes fail before transfer.

### Correction Step C5: IR-005 / AT-021 / REG-007

- Write failing assertions: require no Save draft action in reply, quote, standard scheduled-edit, and project scheduled-edit composers.
- Run command: focused `post_composer_languages_test.dart` and scheduled project editor coverage.
- Confirmed failure: implementation review recorded the visible disabled actions. No separate pre-fix command was retained for this presentation-only correction.
- Implement: omit local-draft actions from quote/reply/comment and scheduled-edit origins.
- Run command: focused reply/quote test and complete scheduled-edit suite.
- Notes: Green. Ineligible origins omit the action entirely; eligible top-level standard/project origins retain Save draft or Save changes.

### Correction Step C6: IR-006 / IT-004

- Write failing tests: detect links without following them, then place a valid external draft behind a symlink at an otherwise valid draft ID.
- Run command: focused `draft_file_store_test.dart` and `file_local_post_draft_repository_test.dart` symlink cases.
- Confirmed failure: the repository followed the link and returned the escaped external draft.
- Implement: reject filesystem links at generated operation targets before direct read, reuse, upload, or delete work.
- Run command: all file-store and file-repository tests.
- Notes: Initial correction green for direct generated targets. The later R1 pass extends this to every protected ancestor and all enumerated reconciliation/cleanup children.

### Correction Step C7: IR-007 / Missing Integration Evidence

- Write/expand integration evidence: real session-registry stale-create test; local preparation with zero blob calls; draft provider account-family isolation; storage restart/update/delete; damaged-media recovery; origin revision lifecycle; immediate and scheduled transfer mutation; reply/quote/scheduled action eligibility; symlink escape; privacy canaries.
- Run commands: complete `test/drafts`, composer/provider/widget correction files, project discard coverage, and `scheduled_post_edit_test.dart` in one targeted invocation.
- Confirmed failures: the missing account, mutation, replacement, origin, and link boundaries failed during C1-C6. Tests whose behavior was already implemented were consolidated under the exact observable target rather than duplicated under speculative filenames.
- Implement: add the missing observable integration coverage or record an exact equivalent target for every planned Must test.
- Run commands: targeted correction invocation completed with 97 passing tests; final full-suite and analyzer commands are recorded below.
- Notes: This first correction consolidated equivalent seam evidence. R2-R9 below supersede the remaining equivalence claims with direct public-flow, retention, privacy, async, and API-client integration targets. The Instagram migration fixture adjustment remains compatibility-only.

### Pre-existing Regression Fixture Compatibility

The full application suite exposed one unrelated Instagram migration assertion that searched for an `InkWell` ancestor around the Notification settings link. The current UI implements that inline link with a `TextSpan` recognizer inside `RichText`, so the baseline assertion fails even though the link remains interactive. The test-only adjustment scrolls the notice into view and verifies the recognizer directly. It does not change production behavior and is not evidence for any local-drafts requirement.

## Second Implementation Review Correction Pass

Correction source: `06-implementation-review.md` — Changes required on 2026-08-04.

| Step | Finding | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|---|
| R1 | IR-009 | IT-004 | FR-015, NFR-002 | AC-015, AC-019 | Linked ancestors and reconciliation directories can escape the trusted draft root |
| R2 | IR-010 | IT-015 | FR-014 | AC-018 | Account seams are tested separately rather than through standard/project submission |
| R3 | IR-010 | IT-004 | FR-016 | AC-015 | Replacement is tested below the Drafts-page-to-composer boundary |
| R4 | IR-010 | IT-014 | FR-008, FR-022 | AC-012, AC-027 | Revision advancement is tested without failed-submit retry through a composer lifecycle |
| R5 | IR-010 | IT-010 | FR-010, RULE-001 | AC-009 | No single recording boundary covers every composer origin before submission |
| R6 | IR-010 | IT-016 | BR-002, NFR-002 | AC-019 | Privacy canaries cover model strings but not repository/submission failures |
| R7 | IR-010 | IT-005 | FR-014, FR-020, RULE-006 | AC-018, AC-023 | Account families exist, but sign-out/re-sign-in retention is not exercised against files |
| R8 | IR-010 | IT-018 | NFR-004 | AC-020 | Async state is tested with one draft rather than maximum-media save/open work |
| R9 | IR-010 | IT-017 | FR-011, FR-012 | AC-010, AC-011 | The scheduled-media and schedule HTTP wire contract lacks a direct client test |

### Correction Step R1: IR-009 / IT-004

- Write failing tests: link the `v1` storage ancestor to an outside draft tree; replace a valid draft's media directory with a link to an outside canary; then cover linked media files and a linked draft-directory delete target.
- Confirmed failures: the linked ancestor allowed an external manifest to be read, and reconciliation deleted a canary through the linked media directory.
- Implement: treat the application-documents directory as the trusted root, validate every existing component between it and each account/draft/media/manifest target without following links, recheck targets after directory creation and before manifest replacement, validate direct children returned by enumeration, and use the same guard for reconciliation, pending cleanup, media reuse, and deletion.
- Run command: complete draft path, file-store, repository, and repository-failure tests.
- Notes: Green. Linked ancestors, draft/media directories, media files, cleanup targets, and delete targets are rejected or skipped. Outside canaries remain unchanged.

### Correction Step R2: IR-010 / IT-015

- Write failing tests: hold standard and project image upload open, activate a different retained account, release upload, and observe the real composer submission result.
- Implement: no further production change was required after C1; these tests exercise the completed lease fence through both public composer flows.
- Run command: focused standard and project account-switch widget cases.
- Notes: Green. Each flow starts one upload and performs zero post-create calls after ownership changes.

### Correction Step R3: IR-010 / IT-004

- Write failing test: open a readable damaged row from the real Drafts page and require preserved text/alt text plus visible unavailable/replacement controls.
- Confirmed failure: routing and hydration succeeded, but `Image unavailable` was not visibly rendered.
- Implement: render the content-free unavailable status beside the existing Replace image action.
- Run command: `flutter test test/drafts/drafts_page_test.dart --reporter compact`.
- Notes: Green. The route reaches `PostComposerSheet`, preserves authored fields, and exposes both recovery controls.

### Correction Step R4: IR-010 / IT-014

- Write failing tests: submit an originating standard/project draft, fail the first create, retry in the same mounted composer, and enforce repository revision conflicts.
- Implement: no further production change was required after C3; the public-flow tests now prove the shared origin tracker is used by both composers.
- Run command: focused standard and project retry widget cases.
- Notes: Green. Snapshot expectations advance from revision 1 to 2, the successful first project upload is reused, and origin deletion happens only after the second create succeeds.

### Correction Step R5: IR-010 / IT-010

- Write failing boundary harness: mount standard, quote, reply, project, new-schedule, and scheduled-edit composers with fail-fast public-upload/private-staging clients, then mutate local alt text and order without submitting.
- Implement: no production change was required; every origin already shares the local-only composer image state and submit-time materializers.
- Run command: `flutter test test/drafts/network/draft_network_boundary_test.dart --reporter compact`.
- Notes: Green. Six public origins perform zero blob uploads, staging transfers, or schedule mutations before explicit submission.

### Correction Step R6: IR-010 / IT-016

- Write failing test: exercise real save/list/get/read/delete failures and a shared submission failure with canaries in content, project values, owner, path, file, bytes, and upstream error text.
- Confirmed failure: `ComposerSubmissionCoordinator` passed the raw upstream exception, including its private failure canary, to the UI callback.
- Implement: replace raw exceptions at the shared UI boundary with content-free `ComposerSubmissionFailure`.
- Run command: privacy integration plus coordinator lifecycle tests.
- Notes: Green. Repository and submission diagnostics contain no canary.

### Correction Step R7: IR-010 / IT-005

- Write lifecycle test: save Alice's text and media, drop session-owned objects to model sign-out, open Bob's isolated repository, then recreate Alice's repository and explicitly delete the retained draft.
- Implement: no production change was required; account-hashed persistent roots already have the required retention semantics.
- Run command: `flutter test test/drafts/data/draft_retention_test.dart --reporter compact`.
- Notes: Green. Sign-out does not delete local work, Bob cannot see Alice's draft, sign-in restores it, and explicit deletion removes it.

### Correction Step R8: IR-010 / IT-018

- Expand async evidence: issue one controller save with four prepared media items, hold the repository future, verify loading state and event-loop progress, then complete it.
- Implement: keep the provider listened to for the duration of the async assertion; no production change was required.
- Run command: `flutter test test/drafts/providers/draft_save_controller_test.dart --reporter compact`.
- Notes: Green for the provider-save half. Maximum-media work remains asynchronous and exposes bounded progress state, but the planned widget save-and-open path and visible progress assertion remain the Should-level gap recorded as IR-012.

### Correction Step R9: IR-010 / IT-017

- Add client contract test: record create, update, publish-now, and raw media-stage requests including methods, paths, camelCase bodies, UTC time conversion, bytes, and content type.
- Implement: no production change was required; the direct client test pins the preserved AppView contract.
- Run command: `flutter test test/scheduled_posts/scheduled_post_api_client_test.dart --reporter compact`.
- Notes: Green. No AppView route or payload change was introduced.

## Third Implementation Review Correction Pass

Correction source: `06-implementation-review.md` — Changes required on 2026-08-04.

| Step | Finding | Test ID | Requirement IDs | Acceptance Criteria | Expected Initial State |
|---|---|---|---|---|---|
| S1 | IR-011 | IT-013 | FR-009, NFR-005 | AC-014, AC-026 | A screen-awake disable exception masks the submission result and leaves the lifecycle running |

### Correction Step S1: IR-011 / IT-013

- Write failing tests: make the first screen-awake disable call fail after a successful submission and after a failed submission; require the original result or operation error to remain authoritative, the lifecycle to become idle, and a later run or disposal to retry release.
- Confirmed failure: the disable exception escaped from `run`, masking the successful terminal result and preventing the lifecycle from becoming idle.
- Implement: make release best-effort, retain release ownership after a failed disable so a later run or disposal can retry it, and clear the running guard in an inner `finally` regardless of release outcome.
- Run command: `flutter test test/feed/composer/submission_screen_awake_test.dart --reporter compact`, followed by the coordinator, overlay, standard composer, project composer, and scheduled-submission regression group.
- Notes: Green. Four lifecycle tests preserve success and operation failures, unblock retries, and retry retained release; the 22-test submission regression group also passes.

### Non-blocking Review Note: IR-012 / IT-018

- `NFR-004` is a Should requirement. The four-media controller test proves asynchronous persistence and bounded provider progress, but it does not execute the full Drafts-page widget save-and-open path or assert its visible progress treatment.
- Keep the existing provider evidence, do not claim complete `IT-018` coverage, and carry the widget-level scenario as a remaining Should/manual gap tied to `MAN-003`.

## Post-review Runtime Correction

Runtime evidence from the iOS simulator showed that draft manifests were written successfully while the composer still reported `Could not save draft`. The composer invoked the auto-dispose `draftSaveControllerProvider` through `ref.read` without retaining a widget-tree listener, allowing the mutation provider to dispose before its asynchronous completion could close the composer and report success. The provider-level test kept an explicit subscription and therefore did not reproduce production usage.

### Runtime Correction Step P1: FR-018 / AC-025

- Write failing widget test: start a standard draft save through the real composer, hold the repository future across multiple frames without an external provider subscription, then require completion to close the composer and emit `Draft saved` without an error.
- Confirmed failure: the repository completed, but the composer remained open because the auto-dispose save controller was no longer retained.
- Implement: listen to the account-family `draftSaveControllerProvider` from both standard and project composer widget trees while their active account is mounted.
- Run command: focused failing test, then draft-controller, standard-composer, and project-composer regression suites.
- Notes: Green. The mutation remains alive for the owning composer lifecycle and remains auto-disposable after the composer unmounts.

Runtime evidence from a subsequent immediate-post attempt showed a valid 7,251,552-byte JPEG reaching the AppView, resuming the PDS session, and then being canceled after approximately 15 seconds. The composer uploader had the approved independent one-minute timeout, but the shared Dio client's shorter `receiveTimeout` still applied to the blob request and ended it first. The AppView correctly propagated that client cancellation as HTTP 499.

### Runtime Correction Step P2: FR-011, FR-012 / AC-010, AC-011, AC-014, AC-026

- Write failing tests: send an immediate upload through a real loopback Dio server whose response exceeds an inherited short `receiveTimeout`, and inspect the scheduled staging request options under the shared client timeout.
- Confirmed failure: the immediate request failed at the inherited response timeout and the scheduled request retained the shared 15-second value, so neither path honored its existing one-minute transfer budget.
- Implement: override `receiveTimeout` with `Duration.zero` only for immediate blob uploads and scheduled media staging. Keep the existing per-transfer one-minute outer timers and cancellation tokens authoritative.
- Run command: the two focused API client files, followed by uploader, coordinator, scheduled edit, and client regressions.
- Notes: Green. The real-network immediate regression and scheduled request-option regression pass; the focused 58-test submission/media group and complete 1,287-test Flutter suite also pass.

Runtime testing of a reopened project draft exposed a display-only round-trip bug in the pattern tag/name field. The normalized form value already retained its leading `#`, but composer initialization unconditionally prepended another one. The form state itself retained the original value unless the field was edited, which is why the extra prefix appeared only once rather than accumulating on every reopen.

### Runtime Correction Step P3: FR-005 / AC-004, AC-006

- Write failing widget test: open a project draft whose stored pattern tag/name is `#SockKAL`, then require the visible editor value to contain exactly one leading hash and the dependent Pattern info fields to remain visible.
- Confirmed failure: the reopened editor displayed `##SockKAL` because initialization combined `#` with an already-prefixed stored value.
- Implement: normalize only the display value during initialization—retain one existing prefix, add one for legacy/unprefixed values, and keep empty values as the single `#` placeholder. Do not rewrite the persisted draft value.
- Run command: focused project-draft snapshot, project composer, metadata, and submission suites.
- Notes: Green. The regression passes and the 16-test focused project group remains green.

Runtime testing on page two of a reopened project draft exposed a second project-field hydration bug. JSON decoding produced `List<dynamic>` values for the saved colour and design-tag arrays, and the snapshot adapter passed those runtime types into `FormBuilderField<List<String>>`. Materials did not fail because their adapter already reconstructed typed `ProjectMaterial` values explicitly.

### Runtime Correction Step P4: FR-005 / AC-004, AC-006

- Write failing tests: decode JSON-shaped dynamic lists through the project snapshot adapter and require typed string lists; open a project draft with those values, navigate to page two, and require both selections to render without a framework exception.
- Confirmed failure: the adapter returned `List<dynamic>`, and mounting the first multi-select field threw `type 'List<dynamic>' is not a subtype of type 'List<String>?'`.
- Implement: reconstruct immutable `List<String>` values for the known colour and design-tag fields at the draft decode boundary, retaining only valid string entries. Keep other scalar and material decoding unchanged.
- Run command: focused manifest codec, project snapshot adapter, composer, metadata, and submission suites.
- Notes: Green. The production-shaped page-two regression renders both selections, the adapter proves the restored runtime types, and the 23-test focused group passes.

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
| 12–14 | Focused coordinator, screen-awake, overlay, standard/project composer, and scheduled submission tests at their real paths | Red: shared coordinator missing and disposal release absent; green after shared lifecycle implementation |
| Generation | `dart run build_runner build` | Passed; 136 outputs written |
| Generation | `flutter gen-l10n` | Passed |
| Review corrections | Targeted drafts/composer/provider/widget/scheduled invocation | Passed; 97 tests |
| Review fixture | `flutter test test/instagram_migration/instagram_migration_page_test.dart --reporter compact` | Passed; 11 tests; compatibility-only evidence, not linked to a draft requirement |
| Second review filesystem | `flutter test test/drafts/data/draft_storage_paths_test.dart test/drafts/data/draft_file_store_test.dart test/drafts/data/file_local_post_draft_repository_test.dart test/drafts/data/file_local_post_draft_repository_failure_test.dart --reporter compact` | Passed; ancestor, directory, file, reconciliation, cleanup, and delete link boundaries covered |
| Second review public flows | Standard/project account-switch and same-composer retry widget cases; damaged Drafts row-to-composer case | Passed; delayed ownership, recovery UI, revision advancement, upload reuse, and success-only cleanup observed |
| Second review boundary evidence | `draft_network_boundary_test.dart`, `draft_privacy_test.dart`, `draft_retention_test.dart`, `draft_save_controller_test.dart`, `scheduled_post_api_client_test.dart` | Passed; zero pre-submit transfer, diagnostics redaction, retention/isolation, four-media async state, and wire contract observed |
| Second review focused regression | 15 correction-related files in one `flutter test ... --reporter compact` invocation | Passed; 45 tests |
| Third review lifecycle | `flutter test test/feed/composer/submission_screen_awake_test.dart --reporter compact` | Red: disable failure masked the terminal result and retained the running guard; green after best-effort release with retained retry ownership, 4 tests passed |
| Third review submission regression | Coordinator, screen-awake, overlay, standard composer, project composer, and scheduled-submission tests | Passed; 22 tests |
| Post-review runtime correction | Draft save controller plus standard/project composer regression suites | Red: delayed production-shaped save left the composer open; green after widget-tree listeners, 13 tests passed |
| Post-review media-timeout correction | `flutter test test/feed/data/post_api_client_test.dart test/scheduled_posts/scheduled_post_api_client_test.dart --reporter expanded` | Red: immediate upload inherited a 20 ms Dio response timeout and scheduled staging retained the shared timeout; green after request-scoped overrides, 37 tests passed |
| Post-review media-timeout regression | Immediate API client/uploader/coordinator and scheduled client/edit suites | Passed; 58 tests |
| Post-review project pattern hydration | Project draft pattern regression, snapshot adapter, composer, metadata, and submission suites | Red: `#SockKAL` reopened as `##SockKAL`; green after display-prefix normalization, 16 tests passed |
| Post-review project list hydration | Manifest codec, JSON-shaped list adapter regression, page-two draft widget regression, and project composer suites | Red: decoded lists remained `List<dynamic>` and page two threw during FormBuilder registration; green after typed reconstruction, 23 tests passed |
| Static analysis | `just app-analyze` | Passed; no issues found |
| Full regression | `just app-test --reporter compact` | Passed; 1,287 tests |
| Diff hygiene | `git diff --check` | Passed |

## Remaining Manual Release Checks

- `MAN-001`: physical iOS/Android persistence, source loss, restart, and process interruption.
- `MAN-002`: physical-device screen-awake behavior and release on every terminal path.
- `MAN-003`: themes, large text, accessibility semantics, and maximum-media responsiveness.
- `MAN-004`: sign-out retention, explicit deletion, time retention, and app-data removal.

`IR-012` remains an explicit Should-level coverage gap under `MAN-003`: exercise a four-media save through the Drafts-page widget, verify visible progress and continued rendering responsiveness, then reopen the saved draft and verify all four media items.

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
