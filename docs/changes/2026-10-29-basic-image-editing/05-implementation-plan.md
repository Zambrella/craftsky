# TDD Implementation Plan: Basic Image Editing and Avatar Preview

## Inputs
- Requirements: `01-requirements.md`
- Tests: `02-acceptance-tests.md`
- Review: `03-document-review.md` (approved with notes)
- Coding plan: `04-coding-plan.md`

## Implementation Rules
- Link every behavior to approved requirements.
- Write one failing behavior test, run it, implement minimally, then verify green.
- Refactor only while green; retain existing integration tests unchanged.
- No commits, push, production actions, or new integration infrastructure.

## Test Order
| Step | Test ID | Requirement IDs | Acceptance Criteria | Status |
|---|---|---|---|---|
| 1 | AT-001, UT-001 | FR-001, RULE-001 | AC-001, AC-008 | Completed |
| 2 | AT-002, AT-005 | FR-002, FR-005 | AC-002, AC-005 | Completed |
| 3 | UT-002, UT-003, UT-005, AT-003, AT-004, AT-008, AT-009 | FR-003, FR-004, RULE-001–003 | AC-003, AC-004, AC-008–010 | Completed |
| 4 | AT-005, AT-006, AT-009, REG-003 | FR-004–006, RULE-003 | AC-004–006, AC-010 | Completed |
| 5 | AT-007, UT-007, REG-002 | FR-007 | AC-007 | Completed |
| 6 | UT-004, UT-006, UT-008, REG-004 | RULE-002, NFR-003 | AC-009, AC-013 | Completed |
| 7 | AT-010, REG-001 | NFR-002, FR-001 | AC-001, AC-012 | Completed |

## Implementation Steps
Execution evidence is added after each focused red/green loop.

## Completion Checklist
- [x] All Must requirements covered by tests or documented gaps
- [x] All planned automated Must tests passing; manual checks recorded below
- [x] Relevant regression tests passing
- [x] No unlinked behavior implemented
- [x] Docs updated
- [x] Implementation review explicitly deferred to next stage

### Step 1: AT-001 / UT-001
- Red: focused attachment suite failed compilation because the optional `onEditImage` boundary did not exist.
- Green: `just app-test test/feed/widgets/composer_image_attachment_section_test.dart`: 10 passed after adding independent overlay actions on ready images, summary and details. Added eligibility regression for unavailable/reference images; existing restrictions already satisfy it.
- Test setup required shared SDK/cache access; no source behavior altered for setup.
- Optional callback retains business form behavior and existing summary-to-details navigation.

### Step 2: AT-002 / AT-005
- Red: shared editor test failed for missing wrapper/dependency.
- Resolved pinned `pro_image_editor` 14.21.1; only one dependency added.
- Green: restricted real-package smoke widget passes after allowing its async decode and draining its entrance animation timer.
- Adapter uses encoded local memory, rotate-only tools, disabled built-in transform shortcuts, free composer crop and fixed-square oval avatar preview, square PNG export bounded to 4000 pixels.
- No package geometry/export suite added.

### Step 3: UT-002 / UT-003 / UT-005 / AT-001 / AT-003–004 / AT-008–009
- UT-002 red: missing `editImage`; green after atomic preparation/replacement. Target lookup uses stable ID and latest metadata, clearing old origin.
- UT-005 red: pending edit still allowed submit; green after transient busy state and source/operation/lease guards. Removed images reject late output. Cancellation/failure preservation test passes without additional implementation.
- Full provider suite: 21 passed.
- AT-001 owning widgets red: post/project callback null; wired both scoped launchers and imperative submission/draft/schedule guards.
- Implementation uses one transient boolean rather than an editing-ID set because only one editor can be active per composer. Draft snapshots still consume only accepted images.
- Scoped build_runner removed unrelated generated outputs; restored only its deleted tracked outputs from HEAD (initial worktree was clean), retaining the changed composer mapper.

### Step 4: AT-005 / AT-006 / AT-009 / REG-003
- Avatar picker red: missing optional editor/currentness boundary. Green: cancellation opens avatar mode before preview/upload and makes zero requests.
- Onboarding UI red: editor callback absent. Green: both camera/gallery pass scoped launcher.
- Edit Profile red: missing callback/currentness guard; green with preserved original error owner and prior avatar restoration.
- Onboarding imperative save red: saved previous blob during pending selection. Green: save/next/back/completion respect pending avatar work; no profile mutation while pending.
- Focused avatar/profile regression run: 51 passed. Separate onboarding page suite: 6 passed.
- Edited avatar previews remain on the last accepted selection until upload succeeds, preventing failed uploads from displaying an unsaved replacement. Existing no-editor business callback timing is unchanged.

### Step 5: AT-007 / UT-007 / REG-002
- Repeated edit/reopen test passes against current operation: accepted current bytes enter each next request, and draft media adapter receives fresh bytes/dimensions/alt/ID.
- No storage implementation change required. Existing snapshot/repository suites remain authoritative for persistence mechanics.

### Step 6: UT-004 / UT-006 / UT-008 / REG-004
- UT-004 red: final validation override rejected output but edit still applied; green after explicitly validating prepared edited bytes before commit.
- UT-006 red: consumed editor failure emitted no issue; green after one private failure owner using supported reporting entry points. Serialized local/Sentry canaries retain operation, cause type and supplied stack; protected image prose/bytes/path absent.
- Existing avatar reporter remains the consuming owner for avatar failures; no core diagnostic/privacy configuration changed.
- Local avatar handoff/cancellation tests intercept network boundaries; cancellation/stale result makes no request, accepted preparation reaches only existing output boundary. Business callers opt out by default.

### Step 7: AT-010 / REG-001 and remaining lifecycle checks
- Restricted real-package widget coverage runs in both composer/avatar modes: free versus fixed-square aspect, rectangular versus oval UI, square export configuration, rotate-only tools and disabled unrestricted shortcuts.
- AT-009 red: duplicate completion replaced the first output; green after first-result guarding and single route pop ownership.
- AT-010 verifies enlarged text, labeled screen-reader/button semantics, minimum target height, keyboard Cancel and reduced-motion configuration. The app currently supports English only; no unsupported locale was invented for testing.
- Package wall-clock entrance animations did not cooperate with the widget fake clock. Tests allow async image decode across frames and use reduced-motion configuration; this changes test setup, not package geometry.
- AT-004 verifies export-error route handoff preserves the original cause and stack. A small localized toolbar awaits package export, catches errors and releases its loading overlay; the consuming composer/profile flow retains error ownership.
- UT-005 activation-away/back test passes using existing registry session/activation generations. Removed and stale output are inert; previously accepted previews remain mounted while conflicting controls are unavailable.

## Verification and Scope Notes
- `just app-analyze`: passed, no issues.
- Full `just app-test`: passed on final rerun. The first run found a new restored-draft test lacked a listener across an async operation; the corrected test retains the provider, matching mounted production ownership.
- Relevant code changes remain in the Flutter client, dependency lock, localization and unit/widget tests. No backend/API/Lexicon, permissions, auth implementation, production, integration-test files, drivers, or durable editor-history changes.
- Existing post/project local-draft dirty checks already include accepted content hashes. Published-image editing remains limited to new local attachments, whose identity differs from the retained base; no remote acquisition or new dirty-state model is needed.
- The package worker asset is bundled under `assets/packages/pro_image_editor/...`, not a third-party editor endpoint. The adapter passes local encoded bytes only.
- MAN-001–MAN-003 are not performed in this stage: no claim of physical iOS/Android, browser appearance/export, gallery interaction, screen-reader, live publication, or manual disk save/reopen validation. Package functionality/platform support is trusted per maintainer direction; existing storage/preparation suites and application mapping tests supply automated coverage.
- Implementation review is deferred to the next workflow stage. No commits or push authorized or created.

## Requirement Traceability
| Requirement | Implemented behavior and verification |
|---|---|
| BR-001 | Shared crop/rotate editor plus scoped composer/avatar entry; AT-001/002/005 |
| FR-001 | Ready local summary/details Edit actions and post/project opt-in; attachment and owning-widget suites |
| FR-002 | Actual package widget restricted to rotate/free crop; AT-002; core gestures/export trusted |
| FR-003 | Atomic prepared bytes/hash/dimensions replacement using latest ID/order/alt; UT-002 |
| FR-004 | Cancel/error preserves accepted state; export failure returns original cause to owner; UT-003, AT-004 |
| FR-005 | Avatar mode uses fixed 1:1 oval UI with mask export disabled; AT-005; visual appearance remains MAN-001 |
| FR-006 | Avatar edit precedes preview/upload and existing save/completion semantics; picker, Edit Profile, onboarding lifecycle/publication suites |
| FR-007 | Repeated reopen uses current pixels; restored edited media loses old origin and maps to fresh media; AT-007/UT-007 plus existing draft/snapshot suites |
| RULE-001 | Optional scoped entry and ImageReady-only eligibility; UT-001/AT-008 |
| RULE-002 | Existing normalized/capped input and final preparation/upload validation; UT-004 plus existing media/profile tests |
| RULE-003 | Single active composer edit, source identity and whole active lease guards, route invalidation, duplicate-result suppression, command guards; UT-005/AT-009 |
| NFR-001 | Package memory editor is shared on supported app platforms; platform support trusted, MAN-001 remains unperformed |
| NFR-002 | Localized labels, accessible minimum target, actual button semantics, keyboard Cancel, reduced motion and enlarged text; AT-010; physical accessibility remains MAN-003 |
| NFR-003 | Memory-only editor boundary, unchanged permissions/gallery/upload architecture, protected serialized local/Sentry failures; UT-006/008, existing business/diagnostic suites |

## Final Verification Evidence
- Corrected restored-draft provider lifetime: listener retained across the await, matching the mounted composer.
- Post-correction focused command: `just app-test test/feed/providers/composer_images_provider_test.dart test/profile/edit_profile_dialog_test.dart test/shared/image/basic_image_editor_test.dart` — **63 passed**.
- Final `just app-analyze` — **No issues found**.
- Final full `just app-test` — **3,523 passed, 38 skipped, no failures**, completed in 4m12s. Existing skipped tests remain skipped; no integration runner invoked.
- `git diff --check` — passed.
- Test logs for this local session: `/tmp/craftsky-basic-image-editing-tests-final.log` and `/tmp/craftsky-basic-image-editing-focused.log`.
- Final diff inspected: only intended Flutter sources/generated state and localization, the single dependency addition/lock, unit/widget tests, and this execution record. Existing workflow inputs remain unchanged.
- Manual checks and implementation review remain explicitly unperformed, as recorded above. No commits or push created.

## Correction Pass: IR-001

The maintainer selected Address required changes after `06-implementation-review.md`. This authorizes the focused correction; it does not reopen scope or enable commits.

| Order | Test IDs | Requirements | Acceptance criteria | Status |
|---|---|---|---|---|
| 1 | AT-009, UT-005 | RULE-003, FR-004, FR-006 | AC-004/006/010 | Completed: reproduced disabled Save after stale completion; focused correction test passes |
| 2 | REG-003 | FR-004, FR-006, RULE-003 | AC-004/006/010 | Completed: prior avatar/fields and save payload retained, retry succeeds, disposal ignores late completion |
| 3 | Verification | Same scoped requirements | Same criteria | Completed: 393 relevant regression tests pass; analysis and diff checks clean |

- Keep the existing worktree and implementation intact outside IR-001.
- First failing test: hold an avatar operation pending in the real Edit Profile widget harness; change activation away/back without unmounting the form; complete the old result; assert previous avatar/typed fields remain and Save/reselection recover.
- Intended implementation boundary: the immutable temporary avatar draft identifies its pending operation. Cleanup may restore only that still-current temporary draft, and must never update disposed state or overwrite a later accepted selection.
- Review artifact remains the original review verdict until the next review stage. Manual checks remain unperformed and no new integration coverage is planned.

### IR-001 loop 1: AT-009 / UT-005 (FR-004/006, RULE-003; AC-004/006/010)
- Setup corrections: constrained the registry-provider import to avoid a name clash; scrolled the avatar control into view after typing.
- Meaningful red: `just app-test test/profile/edit_profile_dialog_test.dart --plain-name IR-001` failed because Save remained disabled after an away/back activation invalidated the pending avatar and its future completed.
- Minimum fix: retain the immutable temporary draft as the operation identity; in `finally`, restore the prior accepted draft only if the form is still mounted and that exact temporary draft remains active. Ordinary success/cancel/failure paths already replace it, so cleanup cannot overwrite accepted output.
- Green: the same focused test passes, including preserved typed name and a subsequent avatar selection.

### IR-001 loop 2: REG-003 (FR-004/006, RULE-003; AC-004/006/010)
- Added an owning-widget regression with a previously accepted avatar, unsaved name, a held later selection, away/back activation invalidation, and a late successful response. The previously accepted preview/blob survive; no profile write occurs before Save; retry/cancel works; Save submits the accepted blob rather than stale output.
- Added disposal coverage: unmounting the form while a selection is held makes its guard false; late completion produces no framework exception or issue and cannot restore the closed UI.
- All three IR-001 focused widget tests pass without further production changes.
- Broader correction gate: `just app-test test/profile test/onboarding test/observability/profile_image_error_reporting_test.dart` — **393 passed**, no failures. Log: `/tmp/craftsky-image-editing-ir001-tests.log`.
- The prior full-suite result (3,523 passed, 38 skipped) is historical evidence from before this correction. The correction uses the relevant broader profile/onboarding/diagnostic gate; it does not claim that the entire application suite was rerun.

### IR-001 correction completion
- `just app-analyze`: **No issues found** after mechanical test formatting/import fixes.
- `git diff --check`: passed.
- Correction source scope: only `_pickProfileImage` temporary-draft identity and guarded `finally` cleanup in `edit_profile_dialog.dart`; three observable widget regressions and their controlled picker fixtures in its existing test file; this execution record. No additional dependencies/generated/API/auth implementation changes.
- Correction checklist: focused red/green confirmed; previous accepted avatar, unrelated fields, retry/save and disposal verified; relevant broader gate passed; no integration tests added; no commits or push created.
- The implementation review in `06-implementation-review.md` was performed after the original implementation. Its original Changes required verdict is retained as historical correction input; IR-001 is now implemented and ready for re-review. Manual checks remain unperformed.
