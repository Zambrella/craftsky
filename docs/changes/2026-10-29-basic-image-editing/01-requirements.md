# Requirements: Basic Image Editing and Avatar Preview

## 1. Initial Request

Users need basic photo cropping and rotation before upload on iOS, Android, and web. Profile pictures should show how the selected composition looks as a circle. After package discovery, the maintainer selected `pro_image_editor`, partly for its potential to support additional editing tools later. For post/project images, an Edit button on the image in the composer must open the editor. This stage documents requirements only.

Source labels used below: **P1** = initial crop/rotate, avatar, and platform request; **P2** = confirmed package choice and composer Edit-button request; **D1** = preceding package research; **C1–C5** = current code findings below; **A1–A4** = explicitly recorded assumptions in section 20.

## 2. Current Codebase Findings

- **C1 — Composer UI:** `app/lib/feed/widgets/composer_image_attachment_section.dart` is shared by `post_composer_sheet.dart` and `app/lib/projects/widgets/project_composer_sheet.dart`. They use thumbnail summaries and a separate photo-details route containing alt text, removal, and ordering controls. The shared component also serves business forms; those must not acquire editing implicitly.
- **C2 — Image lifecycle:** `app/lib/feed/providers/composer_images_provider.dart` prepares selected/camera images locally. `composer_image_state.dart` distinguishes local `ImageReady` bytes from retained `PublishedImageReady` references and server-stored scheduled media. Existing published-image edits do not download old image bytes.
- **C3 — Avatars:** `app/lib/profile/pages/edit_profile_dialog.dart` and `app/lib/onboarding/providers/onboarding_flow_provider.dart` call `ProfileImagePicker.pickAndUpload`. `app/lib/profile/providers/profile_image_picker_provider.dart` currently picks, validates, prepares, previews, and uploads without an interactive editor. Business product/event forms also use this picker, so editor invocation needs an explicit use context.
- **C4 — Preparation:** `composer_image_media_service.dart`, `media_config.dart`, and `platform_image_compression.dart` own existing format, source-size, pixel, output-size, orientation, and metadata handling. Profile output is limited to 1 MB. `app/pubspec.yaml` includes `image_picker`, `image`, and existing compression support, but no interactive editor dependency.
- **C5 — Verification patterns:** Existing tests cover composer attachment UI/state, local drafts, scheduling, profile picker, Edit Profile, and onboarding. Root recipes are `just app-analyze` and `just app-test <test paths>`; the latter prepares host Rive libraries before Flutter tests. Brief manual checks of the finished UI cover platform-specific interaction; no new integration tests are planned. No build or tests run for this documentation stage.

## 3. Clarifying Questions And Decisions

### Q1: Which package and initial tools?

Answer: The maintainer selected `pro_image_editor` and requested basic crop/rotate only.
Decision / implication: Use that package; future tools remain a motivation, not current scope.

### Q2: How does a user edit a post/project photo?

Answer: An Edit button on the image in the composer opens the editor.
Decision / implication: Editing is optional and explicitly invoked per photo; selecting multiple photos does not launch an editor for each one.

### Q3: How should avatar editing be entered, and what existing media is editable?

Answer: The maintainer confirmed all A1–A4.
Decision / implication: These are confirmed scope decisions; no blocking questions remain. The maintainer subsequently approved the plan with new integration tests excluded.

## 4. Candidate Approaches

### Option A: `pro_image_editor` — selected

Summary: A Flutter editor restricted to crop/rotate for this release.
Pros: Shared UI across supported platforms; configurable tools; oval preview and independently configurable mask export; future extension potential.
Cons: Broader package/API than the initial feature needs.
Risks: Device memory use, export fidelity, dependency compatibility, and exact standalone-editor integration require verification.

### Option B: `image_cropper` — considered

Summary: Platform-specific mobile editors and Cropper.js on web, familiar from Stash Hub.
Pros: Established crop/rotate functionality and maintainer familiarity.
Cons: Different UI/configuration by platform; documented web settings do not expose mobile circular crop style.
Risks: Additional web work for avatar preview parity.

Other candidates considered were `extended_image` (more custom UI/export work), `crop_your_image` (rotation requires extra work), `crop_image` (custom circle preview), Croppy (limited-support warning), and CropperX (older release). These are discovery context, not implementation alternatives to reopen without cause.

Discovery references: [pro_image_editor](https://pub.dev/packages/pro_image_editor), [crop/rotate configuration](https://pub.dev/documentation/pro_image_editor/latest/core_models_editor_configs_crop_rotate_editor_configs/CropRotateEditorConfigs-class.html), [image_cropper](https://pub.dev/packages/image_cropper). Package capabilities were reviewed in the preceding discovery; integration and performance are not yet proven in CraftSky.

## 5. Recommended Direction

Use `pro_image_editor` for a deliberately small crop/rotate experience. Composer edits replace one local attachment only after successful confirmation and preparation. Avatar selection opens a square crop with a circular preview before upload. Export an ordinary square avatar image with the circle used only for preview. Retain CraftSky's existing validation, preparation, draft, upload, and account ownership rules.

## 6. Problem / Opportunity

Users currently need another app to adjust photo framing or rotation and cannot confirm avatar composition against its circular display before upload.

## 7. Goals

- G-001: Let users adjust individual post/project images without leaving the composer workflow.
- G-002: Let users compose avatars against an accurate circular preview.
- G-003: Deliver both on iOS, Android, and web without losing existing attachment data or bypassing upload constraints.

## 8. Non-Goals

- NG-001: Filters, brightness/colour adjustments, drawing, text, stickers, AI tools, video editing, and other editor modules.
- NG-002: Editing images in feeds/viewers, downloading retained published or server-stored scheduled images for editing, or changing existing record-edit policy.
- NG-003: Profile banners, business product/event images, or other attachment surfaces outside post/project composers and avatar selection.
- NG-004: Persisting editor history/originals, reverting to an original across sessions, or adding backend image processing, endpoints, migrations, or Lexicon fields.
- NG-005: Changing photo count, accepted formats, upload limits, or camera/gallery permissions.

## 9. Users / Actors

| Actor | Description | Needs |
|---|---|---|
| Author | User composing a post, reply, quote, or project through the existing composers | Optional per-image editing with existing text/order preserved |
| Profile owner | User selecting an avatar during onboarding or Edit Profile | Circular composition preview before upload |

## 10. Current Behavior

Selected composer images are prepared and previewed with existing attachment controls but no crop/rotate tool. Avatar selection proceeds to preparation/upload without interactive composition. Local drafts can retain app-owned bytes; published and server-stored scheduled attachments may have references without editable local bytes.

## 11. Desired Behavior

Eligible composer thumbnails expose an Edit button directly over the image. It opens crop/rotate controls for that image. Done accepts the result; Cancel or Back discards the session. The accepted result updates its previews and follows existing save/publish flows. Newly selected avatars use the same basic editor with a fixed square crop and circular mask before preparation/upload. Neither opening nor cancelling the editor changes the saved record.

## 12. Requirements

| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Users shall be able to crop and rotate eligible composer photos and newly selected avatars within CraftSky. | Avoid a separate editing app | P1, P2 | AC-001, AC-002, AC-005 |
| FR-001 | Functional | Must | Each eligible post/project composer image shall expose an Edit button over its thumbnail that opens that image in the editor. Cover shared post, reply, quote, and project composer paths wherever image attachment is already supported. | Requested entry point | P2, C1 | AC-001 |
| FR-002 | Functional | Must | The editor shall use `pro_image_editor` and expose crop framing, pan/zoom for composition, 90-degree rotation, Done, and Cancel. Composer cropping shall allow a free aspect ratio. Other editing tools shall be hidden. | Basic, bounded editing | P1, P2, A4 | AC-002 |
| FR-003 | Functional | Must | Done shall replace only the selected attachment after valid edited output is prepared, updating all of its composer previews and dimensions while preserving its identity, order, alt text, other attachments, and other form content. | Keep author work intact | C1, C2 | AC-003 |
| FR-004 | Functional | Must | Cancel, Back, editor failure, or invalid output shall leave the last accepted image and form content intact. Failures shall provide a recoverable message; cancellation shall not be presented as an error. | Safe editing | C2, C3 | AC-004 |
| FR-005 | Functional | Must | Selecting a new avatar in onboarding or Edit Profile shall open the editor before upload, with a fixed 1:1 crop and live circular preview. The exported avatar shall remain square with no baked-in circle or artificial outside-circle fill. | Match circular display | P1, C3, A1, A2 | AC-005 |
| FR-006 | Functional | Must | After avatar confirmation, preview and upload shall use the accepted composition; existing profile save/onboarding completion semantics shall remain in effect. Cancelling shall retain the previous avatar selection and avoid uploading the cancelled selection. | Correct profile result | P1, C3 | AC-006 |
| FR-007 | Functional | Must | Accepted composer edits shall survive existing local draft save/reopen and be used by immediate publishing or subsequent scheduling. Reopening the editor shall start from the current accepted image. | Prevent preview-only edits | C2, A3 | AC-007 |
| RULE-001 | Business rule | Must | Composer editing shall require available prepared local image bytes. Pending, unavailable, retained published, and server-stored scheduled images without such bytes shall not expose Edit. Newly selected/local-draft images in those composers remain eligible once ready. | Bound first-version scope | C2, A3 | AC-008 |
| RULE-002 | Business rule | Must | Edited output shall pass existing preparation and upload rules, including size/format/pixel limits, orientation normalization, and metadata stripping, before replacing the accepted image or uploading. | Preserve media constraints | C4 | AC-009 |
| RULE-003 | Business rule | Must | An editor result shall apply only to its initiating image, composer/profile flow, and account. Removal, replacement, disposal, or account changes shall prevent stale results from being applied. Saving/submitting shall not race an unconfirmed or processing edit. | Prevent stale or cross-account changes | C2, C3 | AC-010 |
| NFR-001 | Non-functional | Must | Crop/rotate and circular preview shall work on iOS, Android, and web, with touch controls and web pointer controls. | Supported platforms | P1, D1 | AC-011 |
| NFR-002 | Non-functional | Must | New controls shall use localized labels, screen-reader names, usable touch targets, keyboard activation/focus on web, and existing reduced-motion preferences. | Accessible editing | Repository UI conventions | AC-012 |
| NFR-003 | Non-functional | Must | Editing shall occur locally without sending photos to a third-party editing service, saving over gallery originals, requesting additional media permissions, or logging image bytes/private media references. Unexpected failures shall follow the mandatory diagnostic guide. | Privacy and safe diagnostics | C4, repository rules | AC-013 |

## 13. Acceptance Criteria

| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001 | Given ready local images in each supported composer path, when the author selects the on-image Edit button, then the correct image opens; adding images alone does not open the editor. The button is available from the composer summary without first opening photo details. |
| AC-002 | BR-001, FR-002 | Given the editor, when the user adjusts crop/pan/zoom and rotates by 90-degree increments, then the preview reflects those operations; composer cropping supports free proportions, Done/Cancel are available, and advanced tools are absent. |
| AC-003 | FR-003 | Given multiple attachments with alt text and an edited crop/rotation, when Done succeeds, then every preview for the target reflects its resulting composition and dimensions, its identity/order/alt text remain, and other attachments/form fields are unchanged. |
| AC-004 | FR-004 | Given a previously accepted attachment and form content, when the user cancels, goes Back, encounters export/preparation failure, or produces rejected output, then the accepted state remains intact; failure offers a message and retry opportunity, while cancellation produces no failure message. |
| AC-005 | BR-001, FR-005 | Given a camera/gallery avatar selection in onboarding or Edit Profile, when the editor opens, then its crop is fixed square and the live circle matches the composition displayed by CraftSky's avatar. Done produces square pixels without applying the circular mask to export. |
| AC-006 | FR-006 | Given an avatar editing session, when cancelled, then the prior selection remains and no upload is initiated for that selection; when confirmed, then the accepted composition is previewed/uploaded and only the existing save/completion action persists the profile change. |
| AC-007 | FR-007 | Given an accepted composer edit, when its local draft is saved/reopened, its editor is reopened, or it proceeds through immediate publish or scheduling, then the latest accepted composition is retained/used, with correct dimensions and alt text. No original/history restoration is implied. |
| AC-008 | RULE-001 | Given ready local-draft/new attachments and attachments in pending/unavailable/reference-only states, then only ready local attachments expose Edit. Replacing a reference-only image with a new ready local selection makes the replacement eligible without downloading the old image. |
| AC-009 | RULE-002 | Given edited images at existing source/output boundaries and with orientation/metadata fixtures, when accepted, then existing upload limits remain enforced, orientation is correct, private metadata is stripped, and rejected output never replaces/uploads over the last accepted selection. |
| AC-010 | RULE-003 | Given an active editor or output processing, when its image/flow/account is removed, replaced, disposed, or changed, then late output does not modify another state. Save/publish/schedule cannot submit an intermediate edit, and duplicate Done actions cannot cause duplicate application/upload. |
| AC-011 | NFR-001 | On iOS and Android devices and web, users can complete crop/rotate and avatar circle composition using platform-appropriate input, and accepted output matches the preview. Large accepted phone photos are included in platform verification. |
| AC-012 | NFR-002 | Given supported locales, screen reader, keyboard navigation on web, and reduced-motion settings, then the new controls remain identifiable, operable, readable, and consistent with those settings. |
| AC-013 | NFR-003 | Given editing, cancellation, and injected failures, then no third-party editing requests, extra permission requests, or gallery-original writes occur; serialized diagnostics exclude protected-value canaries and preserve permitted typed causes/stacks through supported reporting entry points. |

## 14. Edge Cases

| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | Several images; edit the middle one after reordering | Apply by image identity; preserve order and descriptions | FR-003, RULE-003 |
| EC-002 | Repeated edits, tiny/portrait/landscape crops, or four quarter-turns | Start from accepted pixels; maintain valid crop bounds and correct geometry | FR-002, FR-007, RULE-002 |
| EC-003 | Corrupt/unsupported image or export exceeds limits | Reject safely, preserve accepted selection, allow recovery | FR-004, RULE-002 |
| EC-004 | Large photo, EXIF-rotated JPEG, or currently accepted HEIC input | Preserve existing format/platform support; avoid incorrect orientation and verify device memory behaviour | RULE-002, NFR-001 |
| EC-005 | Flow closes/account switches while editor/export is active | Ignore stale completion; no cross-flow application/upload | RULE-003 |
| EC-006 | Cancel avatar selection/edit after a previous successful choice | Keep previous choice and allow later save/completion | FR-004, FR-006 |
| EC-007 | Local draft media missing; retained published/scheduled image | No Edit button until a local replacement is ready; existing other controls remain usable | RULE-001 |
| EC-008 | Business form shares picker or attachment component | No new editor entry point introduced outside scope | FR-001, FR-005 |

## 15. Data / Persistence Impact

- New fields: None required in public records or backend storage; no durable editor history/originals.
- Changed fields: Existing local draft media bytes, dimensions, MIME type, content identity/hash, and stored-origin linkage must represent the accepted edit. Existing upload/publish payloads carry the resulting image.
- Migration required: None anticipated; verify local draft handling during design.
- Backwards compatibility: Existing drafts and reference-only media must remain usable; no published record contract changes.

## 16. UI / API / CLI Impact

- UI: On-image Edit buttons on eligible composer thumbnails; restricted crop/rotate screen; avatar circle preview in onboarding/Edit Profile.
- API / Lexicon: None expected; use current upload and record-write contracts.
- CLI / Background jobs: None identified.

## 17. Security / Privacy / Permissions

- Authentication/authorization: Retain account ownership and current upload authorization; fence asynchronous editor completion.
- Sensitive data: Photos stay local until the existing upload flow; preserve metadata stripping and bounded image validation.
- Permissions: Existing camera/gallery access only. Never modify the source gallery asset or delete PDS records/blobs as editing cleanup.
- Abuse cases: Enforce existing byte/pixel limits on editor inputs and outputs; invalid or oversized output must not bypass preparation.

## 18. Observability

- Events: User cancellation is an expected outcome, not an error issue. No new product analytics requested.
- Logs: Unexpected editor/export/preparation failures use existing supported error reporting under `docs/development/logging-error-reporting.md`, with one issue owner, original typed causes/stacks, and permitted bounded context. No image bytes, private paths/URLs, credentials, or payloads.
- Metrics / Alerts: None identified.

## 19. Risks

| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Editor/export copies of large decoded images exhaust device/browser memory | Closure, stalled UI, or lost session | Bound input/display/export resources; retain sensible resource bounds and briefly check the finished UI on available targets; no new integration/performance suite |
| RISK-002 | Export, preparation, or EXIF normalization changes framing/quality | Preview/output mismatch or degraded image | Keep existing preparation tests and focused wrapper/state tests; briefly inspect the finished result manually |
| RISK-003 | Local draft storage reuses stale original media after edit | Edit disappears on reopen/publish | Verify bytes, content hash, dimensions, stored origin, and exact save/reopen journey |
| RISK-004 | Shared picker/components enable editing in unintended forms | Scope expansion or broken business forms | Explicitly scope invocation and verify unaffected callers |
| RISK-005 | Package version/API or accessibility limitations | Integration work or incomplete platform UX | Select compatible version in coding design; verify restricted tool set, mask-only preview, focus, semantics, reduced motion, and platform export |

## 20. Assumptions

| ID | Assumption | Impact If Wrong |
|---|---|---|
| A1 | New avatar selection automatically opens the editor in both onboarding and Edit Profile; editing the current remote avatar without selecting a photo is excluded. | Entry points and avatar acquisition requirements need revision |
| A2 | The circle is a composition preview; stored avatar pixels remain square. Banners/business images are excluded. | Output format/mask or additional surface scope needs revision |
| A3 | First-version editing covers ready local attachments, including restored local drafts and newly selected replacements; downloading retained published/server-stored scheduled media is excluded. | Requires additional remote acquisition, caching, privacy, and persistence requirements |
| A4 | Basic rotation means 90-degree steps; fine straightening, flips, and composer aspect-ratio presets are not required. | Tool configuration and acceptance cases need expansion |

## 21. Open Questions

- Blocking: None.
- Non-blocking: None regarding A1–A4; all were confirmed by the maintainer. Package version, export integration, and memory strategy belong in coding design; they do not change the selected product direction.

## 22. Review Status

Status: Approved
Risk level: Medium
Review recommended: Completed; see `03-document-review.md`.
Reviewer: Maintainer scope approval and Codex document review
Date: 2026-10-29
Notes: Package choice and composer entry point are confirmed. A1–A4 are confirmed. The maintainer approved the coding plan with new integration tests excluded; use focused unit/widget coverage and brief manual UI checks. No implementation, dependency changes, tests, commits, or production actions are authorized by this artifact.

## 23. Handoff To Test Design

- Requirements file: `docs/changes/2026-10-29-basic-image-editing/01-requirements.md`.
- Next test specification: `02-acceptance-tests.md` in the same folder.
- Must-cover requirement IDs: BR-001; FR-001–FR-007; RULE-001–RULE-003; NFR-001–NFR-003. All link to criteria above.
- Suggested test levels: Unit/provider tests for accepted-result state, draft media mapping, submit inputs and stale-operation handling; widget tests for per-image entry points, limited tools, cancellation, avatar preview and accessibility; brief manual checks of crop/rotate, avatar appearance and save/reopen in the finished UI. No new integration tests, device/browser harnesses or package-conformance suite. Test diagnostics with positive fields and protected-value canaries where reporting changes.
- Blocking open questions: None. Confirmed assumptions and the maintainer’s unit/widget-only automated test scope carry forward.
