# Coding Plan: Basic Image Editing and Avatar Preview

## 1. Inputs

- Requirements: `01-requirements.md` (BR-001; FR-001–FR-007; RULE-001–RULE-003; NFR-001–NFR-003; AC-001–AC-013).
- Tests: `02-acceptance-tests.md` (AT-001–AT-010; UT-001–UT-008; REG-001–REG-004; MAN-001–MAN-003).
- Document review: `03-document-review.md`, **Approved with notes**, Medium risk. DR-001–DR-004 are resolved or superseded as recorded in the revised review. The maintainer approved this plan with new integration tests excluded.
- Maintainer decisions: `pro_image_editor` selected; composer Edit action over each eligible thumbnail; A1–A4 explicitly confirmed. The maintainer also confirmed a unit/widget-only automated test scope; no new integration tests or test harnesses.
- Read-only implementation inspection: shared composer UI/state/preparation, avatar picker and onboarding/Edit Profile callers, draft media write/update adapters, account lease models, localization/configuration, tests, integration README, architecture reference, and mandatory diagnostic guide.
- Stage boundary: This artifact is design only. No dependencies, application/test sources, generated files, commits, or production resources changed. No builds/tests or dependency resolution performed.

## 2. Implementation Strategy

Introduce one shared, memory-input crop/rotate screen using the package's standalone editor, with two modes: composer (free crop) and avatar (fixed square crop, circular preview only). Keep navigation in the widget layer. Pass a narrow editor callback into existing provider/service operations so orchestration can be tested without a `BuildContext` or package state in providers.

Reuse `ComposerImageMediaService` for validation, native HEIC conversion, orientation, metadata removal and size policy. Composer edits start with existing `ImageReady` bytes. Avatars first follow current bounded read/preparation, then open the editor on normalized JPEG/PNG bytes, and finally prepare/validate the accepted export before upload. This preserves supported input formats without asking the editor to decode a raw 48 MP HEIC source. The pre-editor avatar preparation retains today's limits/quality policy; zoom operates on those prepared pixels, not an indefinitely retained original. Do not add persistent originals or a new processing pipeline.

Keep accepted attachment state intact while editing/exporting. Separate transient edit-operation state prevents submission and supplies progress without replacing the image with a placeholder. Commit accepted output atomically by image identity; cancel/failure/stale completion leaves prior accepted bytes and metadata intact. Existing upload, record-write, local-draft and scheduling contracts stay authoritative.

### Package/API evidence and initial dependency choice

Use `pro_image_editor` **14.21.1 as the initial pinned dependency candidate**, matching the inspected upstream stable manifest. It declares Dart >=3.12 and Flutter >=3.44; CraftSky declares Dart ^3.13 and Flutter >=3.47.7, pinned to 3.47.7. Its `material_ui`/`cupertino_ui` dependencies fit the same package families already used by CraftSky. This is manifest-level compatibility, not proven dependency resolution or a reason to upgrade Flutter. Resolve/lock the candidate during the first authorized TDD package step; do not silently upgrade unrelated dependencies. More recent pub.dev metadata and cached API pages differ in version, so inspect the resolved package source before finalizing exact constructor arguments.

Primary sources consulted:

- [Upstream stable manifest](https://raw.githubusercontent.com/hm21/pro_image_editor/stable/pubspec.yaml): initial candidate and SDK/dependency constraints.
- [Standalone CropRotateEditor](https://pub.dev/documentation/pro_image_editor/latest/features_crop_rotate_editor_crop_rotate_editor/CropRotateEditor-class.html): memory input and standalone init configuration.
- [Init configuration](https://pub.dev/documentation/pro_image_editor/latest/core_models_init_configs_crop_rotate_editor_init_configs/CropRotateEditorInitConfigs-class.html): byte-export mode, callbacks and pop ownership.
- [Crop configuration](https://pub.dev/documentation/pro_image_editor/latest/core_models_editor_configs_crop_rotate_editor_configs/CropRotateEditorConfigs-class.html): tools, oval preview and `exportOvalMask`, aspect ratio and animation settings.
- [Upstream crop editor source](https://github.com/hm21/pro_image_editor/blob/stable/lib/features/crop_rotate_editor/crop_rotate_editor.dart): byte-export completion differs from transform-only completion. Do not assume Done returns bytes through `Navigator.pop` automatically.

Verify the resolved API while implementing the wrapper and its widget smoke checks. Trust package geometry/export; do not build a real-codec export/conformance suite.

## 3. Affected Areas

| Area | Existing Pattern | Planned Change | Requirement IDs | AC IDs | Test IDs |
|---|---|---|---|---|---|
| Shared editor | `shared/image` widgets, `material_ui`, localized theme | Standalone memory editor and typed callback contract, restricted tools | BR-001, FR-002, FR-005, NFR-001, NFR-002 | AC-002, AC-005, AC-011, AC-012 | AT-002/005/010, AT-002, MAN-001–003 |
| Composer UI | Shared attachment summary and details route | Independent overlay Edit target; pass edit callback only from post/project composers | FR-001, FR-003, RULE-001 | AC-001, AC-003, AC-008 | AT-001/003/008, REG-001/002/004 |
| Composer orchestration | Riverpod family notifier, local ready bytes | Transient edit token/busy state, validation/atomic replacement and lease fencing | FR-003, FR-004, FR-007, RULE-001–003 | AC-003, AC-004, AC-007–010 | UT-001–005, AT-003/004/007–009, UT-007 |
| Avatar orchestration | Shared profile picker plus onboarding notifier/Edit Profile state | Optional editing callback used only by avatar flows; upload after acceptance; restore prior avatar on failure | FR-004–006, RULE-002/003 | AC-004–006, AC-009/010 | AT-004–006/009, AT-006, UT-004, REG-003/004 |
| Persistence/submission | Existing snapshot/media write adapters and clients | Edited bytes get new hash and no stale stored origin; existing save/publish/schedule paths consume them | FR-003, FR-007 | AC-003, AC-007 | AT-007, UT-007, REG-002 |
| Diagnostics/privacy | Existing private failure reporting and serialized sink tests | Single consuming owner; expected cancel/rejection classification; no private image material | NFR-003 | AC-013 | UT-006, UT-008, REG-004 |

## 4. Files And Modules

Proposed filenames may be adjusted to nearby conventions without changing responsibility or coverage. Avoid creating parallel abstractions if an existing helper provides the same seam.

| Path / Module | Create / Change | Purpose | Requirement IDs | AC IDs | Test IDs |
|---|---|---|---|---|---|
| `app/pubspec.yaml`, `app/pubspec.lock` | Change during TDD | Add/lock selected editor only; inspect platform registration/build effects | FR-002, NFR-001 | AC-002, AC-011 | AT-002, AT-002 |
| `app/lib/shared/image/basic_image_editor.dart` | Create | Mode/request/output/callback contract and scoped route launcher | FR-002, FR-005, NFR-001 | AC-002, AC-005, AC-011 | AT-002/005, AT-002 |
| `app/lib/shared/image/basic_image_editor_page.dart` | Create | Package UI adapter, local export state, Done/Cancel/Back and accessibility | FR-002/004/005, NFR-002/003 | AC-002, AC-004/005, AC-012/013 | AT-002/004/005/010, AT-002, UT-008 |
| `app/lib/feed/providers/composer_images_provider.dart` | Change | `editImage` orchestration, token lifetime and atomic accepted replacement | FR-003/004/007, RULE-001–003 | AC-003/004/007–010 | UT-001–005, AT-003/004/009, UT-007 |
| `app/lib/feed/providers/composer_image_state.dart` and its generated mapper | Change only as needed | Transient editing IDs/state, readiness guard, bounded failure notice | FR-004, RULE-001/003 | AC-004/008/010 | AT-008/009, UT-001/003/005 |
| `app/lib/feed/widgets/composer_image_attachment_section.dart` | Change | Overlay Edit on summary thumbnails and details previews; nullable opt-in callback | FR-001/003, RULE-001, NFR-002 | AC-001/003/008/012 | AT-001/003/008/010, REG-001/004 |
| `app/lib/feed/widgets/post_composer_sheet.dart`, `app/lib/projects/widgets/project_composer_sheet.dart` | Change | Supply scoped launcher and lease, wire edit readiness into save/publish/schedule guards and dirty checks | FR-001/003/007, RULE-003 | AC-001/003/007/010 | AT-001/003/007/009, UT-007 |
| `app/lib/profile/providers/profile_image_picker_provider.dart` | Change | Optional callback/currentness checks between selection, preparation, editor, upload | FR-004–006, RULE-002/003 | AC-004–006/009/010 | AT-004–006/009, AT-006, UT-004 |
| `app/lib/profile/pages/edit_profile_dialog.dart` | Change | Avatar callback and activation lease, previous-avatar restoration | FR-004–006, RULE-003 | AC-004–006/010 | AT-004–006/009, REG-003 |
| `app/lib/onboarding/pages/onboarding_page.dart`, `app/lib/onboarding/providers/onboarding_flow_provider.dart` | Change | Pass launcher callback into `pickAvatar`, preserve existing lease ownership and picker subscription | FR-004–006, RULE-003 | AC-004–006/010 | AT-004–006/009, AT-006 |
| `app/lib/keys.dart` and owning key helper | Change | Stable per-image edit and editor action keys | FR-001, NFR-002 | AC-001/012 | AT-001/010 |
| `app/lib/l10n/app_en.arb`, generated localization files | Change | Accessible localized actions/errors; package i18n adapter | NFR-002, FR-004 | AC-004/012 | AT-004/010 |
| `app/lib/drafts/composer/draft_media_write_adapter.dart` and existing snapshot adapters | Inspect; change only if failing test requires it | Existing `storedOrigin`/hash decision already selects fresh media when origin cleared | FR-007 | AC-007 | UT-007 |
| Existing diagnostic owner helpers | Change only where required | Classify/report new consumed failures through established entry points; no second reporter | NFR-003 | AC-013 | UT-006 |
| `app/test/shared/image/basic_image_editor_test.dart` | Create | Restricted-control widget smoke checks and deterministic output/cancel tests | FR-002/004/005, NFR-002 | AC-002/004/005/012 | AT-002/004/005/010 |
| Existing composer/profile/onboarding/draft/scheduling/business/observability suites | Extend | Application orchestration, regression and serialization coverage | FR-001–007, RULE-001–003, NFR-003 | AC-001–010/013 | AT-001/003–009, UT-001–006, UT-007/008, REG-001–004 |

No AppView, API route, Lexicon, server schema, account permissions, or desktop platform folders need changing. No ADR is needed for this local UI integration unless implementation discovers a broader architectural change.

## 5. Services, Interfaces, And Data Flow

### Narrow editing boundary

Illustrative signatures only; request/output types contain bytes and actual MIME, with redacted `toString` and no serialization/history:

```text
enum BasicImageEditMode { composer, avatar }
BasicImageEditRequest(bytes, mimeType, mode)
BasicImageEditOutput(bytes, mimeType)
typedef BasicImageEditor = Future<BasicImageEditOutput?> Function(request)
showBasicImageEditor(context, request) -> Future<BasicImageEditOutput?>
ComposerImages.editImage(imageId, editor, initiatingLease) -> Future<void>
ProfileImagePicker.pickAndPrepare(..., editImage?, isCurrent?)
ProfileImagePicker.pickAndUpload(..., editImage?, isCurrent?)
OnboardingFlow.pickAvatar(source, editImage: editorCallback)
```

`null` means user cancellation, not an error. Exceptions preserve original typed causes/stacks for the consuming owner. Dimensions are obtained from validated exported bytes, not trusted from a package callback. No path, remote URL, session credential or raw original is passed to the editor. When an editor callback is supplied, require a currentness guard at the avatar orchestration boundary; the optional no-editor path keeps existing business callers unchanged.

### Shared editor configuration

- Instantiate `CropRotateEditor.memory` directly; do not show the full multi-tool editor or add a second menu before cropping.
- Composer: rectangular/free crop, rotate tool only; no aspect-ratio chooser, flip, tilt, paint, text, filters or advanced module entry. Pan/zoom/crop handles remain interactive.
- Avatar: `initAspectRatio = 1`, `initialCropMode = oval`, `exportOvalMask = false`; rotate tool only, no ratio-changing controls. Verify gestures and keyboard shortcuts cannot bypass fixed-square or quarter-turn-only policy. Disable unsupported transform shortcuts/gestures or supply restricted shortcuts rather than only hiding buttons.
- Use byte export (`convertToUint8List`) and the documented image-completion callback. Set one route-result owner: the wrapper completes once and pops its own route; do not also rely on transform-only `onDone`/automatic pop. Inspect resolved API before wiring `enablePopWhenDone` and callbacks.
- Export lossless PNG as the intermediate so alpha is retained and an extra lossy JPEG encode is avoided; existing preparation selects final supported JPEG/PNG according to its current policy. Confirm the resolved package's generation-size option caps export at input dimensions, with no upscaling and at most current 4000-by-4000 bounds. If that option is insufficient, bound the export through its supported generation API before materializing unbounded pixels. Do not reimplement crop geometry in a second Dart image editor.
- Map package strings and icons through CraftSky localization/theme and disable nonessential motion using existing MediaQuery reduced-motion signals. Inspect actual control semantics and shortcuts, not just config fields.

### Composer transaction

```text
validate ready/local + enabled + initiating active lease
reserve edit token for image ID and accepted source revision; mark transient busy
await editor(current accepted prepared bytes)
check token + image existence/source revision + active lease after every await
if cancelled: release token; keep accepted image
prepare + validate accepted export using existing media service
check token/currentness again
replace ready payload/preview/dimensions/hash in place; clear storedOrigin
finally release only this token's busy state
```

Read the latest image at commit time so unrelated alt-text changes/order are preserved. Capture a source revision/token, not a list index; reordering does not invalidate the edit, replacement/removal does. Do not overwrite the whole captured list. Set the filename extension and MIME consistently with final prepared output. Do not reuse uploaded blob metadata or old origin after pixel changes.

### Avatar transaction

Existing bounded `XFile` read/source validation -> existing `prepareImage` (normalized/capped) -> editor callback in avatar mode -> currentness check -> prepare/validate export -> final `onPreviewReady` -> currentness check -> managed blob upload -> currentness check -> accepted preview/blob in owning flow. Only Save/onboarding completion writes the profile.

For edited avatar flows, suppress intermediate pre-editor `onPreviewReady`; no unconfirmed selection replaces the previous avatar. Preserve previous avatar-related draft state until success and restore it on cancel/export/preparation/upload failure. Merge restoration into current form state so unrelated typing is retained. Existing no-editor business flows keep their callback timing. Ignore late results using activation/session generation, not DID equality alone; a switch away and back must invalidate the original operation.

### Persistence and scheduling

`draftMediaWritesFromComposer` uses `ExistingStoredMedia` only when stored origin/hash/length agree. On accepted edit construct a fresh `ImageReady` with `storedOrigin = null` and new hash; it naturally becomes `PreparedDraftMedia`. Existing repository creates a new storage revision. Verify mapping after repeated edits with focused provider/adapter unit tests; briefly check save/reopen manually. Do not add a migration or edit history.

Audit composer dirty checks as well as snapshot adapters: post/project published-edit comparisons currently include IDs/alt text in some paths. A newly selected local replacement can keep its ID across an edit, so pixel changes must count as dirty. Use accepted content hash/revision where applicable; do not treat transient editing/cancellation as a persisted change. Existing immediate and scheduled preparation remains separate and applies existing scheduled geometry rules.

## 6. State, Providers, Controllers, Or DI

Retain existing `@riverpod` ComposerImages family and onboarding AsyncNotifier. Use plain `Provider` only for existing media/picker/error-reporting dependencies; no global editor provider holding `BuildContext` or active private photos is needed.

```text
post/project widget --editor callback + ActiveAccountLease--> ComposerImages
  -> composerImageMediaServiceProvider
  -> editor callback --widget navigation--> BasicImageEditorPage/package
  -> existing draft/submission state

Edit Profile / OnboardingPage --avatar editor callback--> existing picker/flow
  -> profileImageMediaServiceProvider
  -> callback (local route only)
  -> existing managed blob client
  -> existing profile save/completion
```

- Add non-persisted edit-operation bookkeeping to ComposerImages, including an immutable observable set of editing IDs on `ComposerImagesState` (default empty). Tokens remain private and are invalidated on removal/replacement/disposal. Follow existing `ref.onDispose` cleanup.
- `hasImageEditsInFlight` participates in `isSubmitReady`, `canSaveDraftMedia`, and actual save/publish/schedule command guards. Update direct command paths and project-specific controls rather than relying solely on a covered button. Keep previously accepted `ImageReady` bytes mounted during processing.
- Single editing session per composer at a time; do not allow concurrent modal editors. Duplicate Done/export callbacks are suppressed in both page and operation owner.
- Capture `ActiveAccountLease` from current session registry at initiation and compare it after each await. Widget/flow guards must work if no further user action occurs after an account switch. Close/invalidate an editor route on ownership loss; late futures become inert.
- Edit Profile currently reads the persistent profile picker; add a generation-aware guard scoped to its initiating dialog/lease. Onboarding already has `_isCurrent()` and retains a picker subscription across selection; extend those existing checks through the editor interval.
- No new broad AsyncNotifier is needed for the editor. Page-local exporting/error state plus the owning operation's transient state are sufficient. Do not persist edit tokens or bytes in route URLs, router extras intended for serialization, analytics, or provider diagnostic strings.
- If state/notice DTOs change, regenerate only relevant outputs using the existing build_runner workflow from `app/`; inspect generated diff and preserve unrelated files. No route codegen is needed for a local modal page.

## 7. UI, Widgets, Routes, Or User-Facing Surfaces

Use the existing `Navigator`/`MaterialPageRoute` pattern used by photo details, on the initiating navigator with matching ProviderScope. The new local editor has no deep link or shell destination. If route settings are used, give them a static safe name with no bytes/path/identity in arguments. Do not change drawer or shell navigation.

In the shared attachment section, add optional `onEditImage(imageId)` supplied only by post/project callers. For each ready local summary thumbnail, render inline `Stack(preview, positioned Edit control)` with a minimum accessible touch target and unique image key. The existing summary InkWell opens details; ensure tapping Edit consumes that interaction and opens only the editor. Keep a separate usable details target and focus restoration. Current thumbnails are 64 logical pixels; increase only if needed to prevent control overlap, retaining existing spacing/theme. Do not store a widget in a variable; inline it at use sites.

Add the same action over eligible details previews so the auto-opened details screen after image selection does not hide editing access. Retain the current behaviour where photo selection opens photo details; this is distinct from automatically opening the crop editor, which remains optional. Hide Edit for reference-only/pending/unavailable states and outside opted-in surfaces. During a current edit retain previews and guard conflicting actions; restore focus on return where the initiating item still exists.

Avatar entry: Edit Profile's existing source menu and onboarding's source callback pass the launcher. The user selects a photo, then immediately gets the avatar-mode editor. Continue to use CraftSky's existing circular avatar widget after acceptance. No remote-avatar editing shortcut or banner edit control is added.

## 8. Error, Loading, Empty, And Edge States

| State / Case | Planned Handling | Requirement IDs | AC IDs | Test IDs |
|---|---|---|---|---|
| No local bytes / incomplete attachment | No Edit button; keep existing preview/recovery controls | RULE-001 | AC-008 | AT-008, UT-001, REG-002 |
| Editor open / export / preparation pending | Keep accepted preview; show bounded progress for processing; guard submissions and duplicate actions | FR-003/004, RULE-003 | AC-003/004/010 | AT-003/004/009, UT-003/005 |
| Cancel / Back / picker cancelled | Return null; no upload/new issue; release only current busy token; preserve accepted image/form | FR-004/006 | AC-004/006 | AT-004/006, UT-003, REG-003 |
| Invalid/corrupt/oversized/extreme crop result | Existing validation rejects before replacement/upload; localized recoverable feedback; previous accepted state intact | FR-004, RULE-002 | AC-004/009 | UT-004, UT-004 |
| Package/export/preparation unexpected error | Preserve cause/stack; owning flow reports once via supported diagnostic helper; show generic localized recovery | FR-004, NFR-003 | AC-004/013 | AT-004, UT-006 |
| Avatar upload failure | Preserve prior accepted avatar and unrelated form changes; allow reselect/retry through existing UX | FR-004/006 | AC-004/006 | AT-004/006, AT-006, REG-003 |
| Removed/replaced/disposed/account-changed operation | Invalidate token/guard; no commit/upload after invalidation; release route/resources when safe | RULE-003 | AC-010 | AT-009, UT-005, AT-006 |
| PNG transparency / EXIF / HEIC | Normalize via existing platform preparation, preserve current alpha policy, do not rotate twice | RULE-002 | AC-009 | AT-005, UT-004, MAN-001 |

Follow `docs/development/logging-error-reporting.md`: select private context at the consuming owner, preserve original typed causes/stacks, no private payload/paths/URLs or credential material, expected cancellation/rejection classification, positive fields plus protected canaries in serialized sink tests. The shared page must not capture an error that the flow also reports. Diagnostic reporting remains consent-controlled and independent of business outcomes.

## 9. Test Implementation Plan

Automate CraftSky-owned unit/provider and widget behaviour only, as requested. Use fake editor outcomes for orchestration, existing media/snapshot test fixtures and a small real-package widget smoke check for our visible controls. Do not add package pixel/geometry tests, native-codec checks, cross-layer disk journeys, device/browser harnesses, drivers or an export/performance matrix. Existing integration tests remain untouched.

| Order | Test ID | Target | Setup / Fixture | Initial Expected Failure |
|---|---|---|---|---|
| 1 | AT-001, UT-001 | Existing attachment widget suite | Ready local images and callback spy; unavailable/reference states | No per-image Edit action/callback |
| 2 | AT-002/005 | Shared editor widget suite | Actual package UI, small image, composer/avatar modes | No restricted wrapper, circular preview or callback wiring |
| 3 | UT-002/003/005, AT-003/004/008/009 | Composer provider/widget suites | Fake editor output, completers, account lease generations | No atomic accepted edit, cancellation/fencing/readiness |
| 4 | AT-005/006/009, REG-003 | Profile picker/onboarding/Edit Profile suites | Fake editor/upload results and prior avatar | No pre-upload editor or previous-avatar restoration |
| 5 | AT-007, UT-007, REG-002 | Existing snapshot/media adapter/provider suites | Repeated edited payloads with stored origins; standard/project | Old origin/hash/bytes may survive mapping or dirty checks |
| 6 | UT-004/006/008, REG-004 | Existing validation/privacy/business suites | Existing boundary fixtures, diagnostic canaries, recording clients | New validation/reporting/local boundary paths unverified |
| 7 | AT-010, REG-001 | Existing UI/localization and editor widgets | Semantics, locale, focus, reduced motion, text scale | Missing accessible labels or control regressions |

Use existing ProviderContainer.test, fakes, completers and reporters. For edited media, assert current bytes/hash/dimensions/MIME/order/alt at provider/adapter boundaries. Test clearing old stored origin so the existing repository receives fresh prepared media. Do not relabel a full file-backed workflow as a unit test; rely on existing repository tests for storage mechanics.

Brief manual checks (MAN-001–MAN-003) use the finished app: crop/rotate and cancel/accept a photo; compose an avatar and inspect its circle; save/reopen a post and project draft; check relevant controls/focus/labels. Use available iOS/Android/web targets and record what was checked. No new fixture application, browser matrix, codec export comparison, memory profiling project or physical-device test infrastructure is required. Package platform support is trusted. Normal local/test-environment publishing may be checked if already available and authorized; no production write is required.

## 10. Sequencing And Guardrails

- First TDD step: AT-001 in the existing shared attachment widget suite. Run it unchanged to demonstrate the missing on-image Edit behaviour, then add only the minimum UI callback boundary. Pair with hidden-state UT-001.
- Next, resolve the package candidate and add the restricted-editor widget smoke checks. Inspect byte callback/pop ownership, mask-only configuration, bounds and shortcuts while wiring the wrapper. Trust the package implementation; no independent export/conformance suite.
- Implement composer transaction/fencing, then avatar orchestration, then snapshot/submission mapping coverage and remaining diagnostics/accessibility. Every slice follows red-green-refactor; do not prewrite the whole implementation from this plan.
- Bound resources: source validation remains 100 MB/48 MP/current side limits; normalized editor input is at most existing 4000-by-4000 prepared output. Keep one session/export active; retain encoded accepted bytes and only the necessary decoded image/export, release image handles/controllers/subscriptions on close, avoid decoded originals or package history beyond the session. PNG intermediate may be larger than final upload, so bound byte length and dimensions before preparing it. Retain sensible resource bounds in application code; independent memory profiling and worker/shader verification are not part of this test scope.
- Avoid repeated normalization on opening an already prepared composer image. Avatar initial preparation normalizes once; output has no old orientation tag and final preparation validates the edited result. No new broad source-format support or altered limits.
- Check all imperative save/draft/schedule/submit guards, form dirty checks, and profile completion guards. A disabled visible control alone is insufficient.
- No editor callbacks or raw request/result bytes in generated serialization, logs or persistent route state. Keep permitted reporting context private and bounded.
- Use relevant local codegen only when DTO/localization changes require it; inspect diff for unrelated generated deletions. `flutter gen-l10n` and `dart run build_runner build` are run from `app/` at implementation time; scope generated outputs to actual changes. Do not create desktop target folders for host testing.
- Out of scope: advanced tools, remote-image acquisition/editing, banner/business editor entry, durable originals/history, backend/schema/permission changes, live production actions, deployment or pushing.

## 11. Risks And Open Questions

| ID | Type | Description | Impact | Resolution |
|---|---|---|---|---|
| CPQ-001 | Non-blocking validation | Cached upstream sources span versions; exact package resolution and standalone export configuration not executed | API/dependency integration may differ | Start with pinned 14.21.1 manifest candidate; inspect resolved source and verify AT-002/005 widget wiring. If incompatible with Must scope, report concrete conflict; no silent package/SDK substitution |
| CPQ-002 | Non-blocking validation | Native/web resource use for 48 MP input and PNG export | Closure/stall or extra memory copies | Existing serialized source preparation + bounded editor input/export, single active session, dispose resources; brief finished-UI checks where available; no new performance/export suite |
| CPQ-003 | Non-blocking validation | Real-package controls/shortcuts may expose tilt, ratio changes or inaccessible interactions despite hidden buttons | Basic/fixed-square/accessibility contract could fail | Test real gestures, shortcuts, focus/semantics and reduced motion early; use supported customization hooks; flag a Must conflict if unresolvable |
| CPQ-004 | Non-blocking environment | Some device/browser targets may not be available for manual checks | Some acceptance evidence cannot run locally | Use available targets for brief manual checks; no device/browser harness prerequisite or new integration gate |
| CPQ-005 | Non-blocking quality | Avatar pre-editor preparation and final encoding can introduce another resample/encode | Visible image degradation | Lossless intermediate export and no composer reprepare-on-open; briefly inspect accepted results in the finished UI; retain current source/output limits and do not promise raw-original editing |

No unresolved product or architecture decision blocks starting TDD. The maintainer accepts reliance on package behaviour and the specified unit/widget/manual coverage; no platform success is claimed by this plan.

## 12. Handoff To TDD Builder

- Coding plan: `docs/changes/2026-10-29-basic-image-editing/04-coding-plan.md`.
- TDD execution plan: `05-implementation-plan.md` in the same folder, owned by the implementation stage.
- Start with test: AT-001 per-image Edit in `app/test/feed/widgets/composer_image_attachment_section_test.dart`; preserve stable IDs in feature-qualified test descriptions.
- Focused command from repository root: `just app-test test/feed/widgets/composer_image_attachment_section_test.dart`.
- Subsequent focused commands: `just app-test test/shared/image/basic_image_editor_test.dart`; provider/profile/onboarding/draft/scheduling/privacy suites as their slices change; then `just app-analyze` and the full `just app-test` after focused slices pass. Run no new integration tests; report brief manual checks separately from unit/widget results.
- Notes: Carry confirmed A1–A4 and review approval forward. DR-001 is recorded here; DR-002 is reduced to resolved-API inspection and widget wiring; DR-003 harness work and DR-004 pixel/export oracle are superseded by the maintainer’s no-integration-tests decision. The requirements, test specification and review are aligned with the same reduced test scope.
- Approval boundary: Coding planning is complete. The maintainer approved the plan with reduced test scope; this revision does not itself start implementation. No commits are enabled for this workflow.
