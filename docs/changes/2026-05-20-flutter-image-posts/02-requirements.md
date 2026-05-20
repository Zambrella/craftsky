# Business Requirements: Flutter Image Posts

## 1. Summary
Add Flutter app support for top-level image posts using the existing AppView upload-then-create flow, including immediate image upload after selection, upload progress in the composer, required alt text, delete-before-post behavior, configurable media limits, optional image-only posts, feed-card image rendering, and a full-screen gallery with swipe and pinch-to-zoom.

## 2. Problem / Opportunity
The AppView backend now supports image upload and render-ready image post responses, but the Flutter app remains text-only. Users cannot attach images to posts, cannot see upload progress, and cannot view image posts in the feed or gallery surfaces. This blocks end-to-end image posting from the mobile client.

## 3. Goals
- G-001: Let a signed-in user attach images to a new top-level post using the AppView-mediated PDS upload flow.
- G-002: Prevent incomplete or invalid image posts from being submitted while still allowing valid text-only and image-only top-level posts.
- G-003: Render image posts in the feed using the accepted bounded-aspect carousel pattern.
- G-004: Provide a richer viewing experience through tap-to-gallery, swipe, hero transition, and pinch-to-zoom.
- G-005: Preserve existing text-only and reply behavior.
- G-006: Remove most EXIF and non-essential image metadata client-side before upload.

## 4. Non-Goals
- NG-001: Reply-image composition.
- NG-002: Video selection, upload, display, or playback.
- NG-003: Client-side image compression, resizing, or transcoding.
- NG-004: Orphaned blob cleanup after image deletion or abandoned composition.
- NG-005: Avatar or banner upload.
- NG-006: Backend/API contract redesign beyond consuming the existing image upload/create/read surfaces.

## 5. Users / Actors
| Actor | Description | Needs |
|---|---|---|
| Signed-in Craftsky user | User composing a new top-level post | Select images, see upload state, add alt text, and publish only when the post is valid |
| Replying user | User composing a reply | Keep the current text-only reply flow unchanged |
| Flutter app | Client application mediating selection, upload, and display | Validate locally, call existing AppView endpoints, and render image posts consistently |
| AppView | Existing authenticated HTTP backend | Receive blob uploads and post-create requests using the established contract |
| User's PDS | atproto Personal Data Server | Store uploaded blobs and post records via AppView |

## 6. Current Behavior
The Flutter app supports text-only post creation and text-only replies. The `Post` model and post API client do not carry image data. The composer has no image selection or upload state, and feed cards have no image rendering behavior. The backend image upload and image post response contract exists but is unused by the app.

## 7. Desired Behavior
When a signed-in user opens the top-level post composer, they can select supported images up to a configured image-count limit. The app validates selected files as much as practical, strips most EXIF and non-essential metadata before upload, uploads each selected image immediately through AppView, shows per-image preview and progress, requires non-empty alt text for each selected image, allows image reordering, and disables post submission until the draft is valid. A valid top-level post may contain text only, images only, or both. If the user deletes a selected image, it is removed from the draft even if it was already uploaded. Once posted, the new and fetched posts include images rendered in feed cards as an inline bounded-aspect carousel. Tapping the image area opens a full-screen gallery with swipe, hero transition, visible alt text at the bottom of the screen, and pinch-to-zoom; pinch-to-zoom is also available inline in the feed image area.

## 8. Requirements
| ID | Type | Priority | Requirement | Rationale | Source | Acceptance Criteria |
|---|---|---|---|---|---|---|
| BR-001 | Business | Must | Craftsky must let a signed-in user create a top-level post with images using the existing AppView-mediated upload-then-create flow. | This is the core user outcome requested for the Flutter app. | User prompt, discovery | AC-001, AC-002 |
| BR-002 | Business | Must | Craftsky must present image posts in the feed and gallery surfaces so users can view posted images after creation and on subsequent reads. | Posting without display would not complete the user journey. | User prompt, discovery, ADR-004 | AC-003, AC-004 |
| BR-003 | Business | Must | This change must preserve the current text-only reply flow and must not expand image composition to replies in this pass. | Keeps scope aligned with the confirmed discovery direction. | User answer, discovery | AC-005 |
| BR-004 | Business | Must | This change must remain limited to images and must not add video behavior. | Prevents scope creep beyond the requested slice. | User prompt, discovery | AC-006 |
| BR-005 | Business | Must | Craftsky must allow valid top-level posts that contain only text, only images, or both text and images. | Users should not be forced to include both media types. | User feedback | AC-026 |
| FR-001 | Functional | Must | The system shall let a signed-in user select images while composing a new top-level post, up to a configured maximum image count exposed through app-level configuration. | Keeps user-facing behavior aligned with backend/media limits while avoiding scattered hard-coded limits. | User feedback, discovery, backend workflow | AC-001, AC-007 |
| FR-002 | Functional | Must | The system shall validate selected images client-side where practical against the supported image contract: JPEG, PNG, or WebP, and a maximum size per image exposed through app-level configuration. | Reduces avoidable backend failures and follows the request to validate as much as possible in the app. | User prompt, user feedback, discovery | AC-007 |
| FR-003 | Functional | Must | The system shall upload each selected image to `POST /v1/blobs/images` as soon as the user adds it to the draft. | This is the confirmed upload timing and flow. | User prompt, discovery | AC-001, AC-008 |
| FR-004 | Functional | Must | Before upload, the system shall strip most EXIF and non-essential embedded metadata from selected images while preserving the visible image content and compatibility with the supported upload formats. | Reduces unnecessary metadata leakage while staying within image-only scope. | User feedback | AC-009 |
| FR-005 | Functional | Must | The composer shall display a local preview and a per-image upload status indicator for each selected image while upload is pending or in progress. | Users need immediate feedback during upload. | User prompt, discovery | AC-008 |
| FR-006 | Functional | Must | The composer shall surface upload failure on the affected image and shall block submission until that image is either successfully uploaded or removed from the draft. | Prevents invalid create-post payloads and resolves failure handling without requiring backend cleanup. | Discovery refinement | AC-010 |
| FR-007 | Functional | Must | The composer shall allow the user to remove any selected image from the draft before post submission, including images that have already uploaded successfully. | Matches the requested delete behavior. | User prompt, discovery | AC-011 |
| FR-008 | Functional | Must | The composer shall require non-empty alt text for every selected image before allowing post submission. | Aligns with the backend contract and accessibility requirement. | User answer, discovery, backend workflow | AC-012 |
| FR-008A | Functional | Must | The composer shall display an editable alt-text field or equivalent editing affordance for each selected image before post submission. | Users need a direct way to satisfy the required alt-text rule. | Discovery, user feedback | AC-012 |
| FR-009 | Functional | Must | The system shall capture and send image aspect ratio metadata for selected images when it can determine width and height locally. | Supports the accepted lexicon field and feed layout behavior. | Discovery, ADR-003, ADR-004 | AC-013 |
| FR-010 | Functional | Must | `POST /v1/posts` requests from the Flutter app shall include top-level `images[]` entries using the uploaded blob metadata, required alt text, and optional aspect ratio metadata. | Keeps the Flutter client aligned with the existing Craftsky post media contract. | Backend workflow, discovery | AC-002, AC-013 |
| FR-011 | Functional | Must | The top-level post submit action shall remain disabled unless the draft contains valid post content, no selected image is still uploading, no selected image is in unresolved failure state, and all selected images have non-empty alt text. | Enforces a complete, valid draft before create-post while permitting text-only or image-only posts. | User prompt, user feedback, discovery | AC-014, AC-026 |
| FR-012 | Functional | Must | The system shall allow users to add more images to a draft after an earlier selection, up to the 4-image maximum. | Prevents unnecessarily restrictive composition behavior within the confirmed cap. | Discovery refinement | AC-001, AC-007 |
| FR-013 | Functional | Must | The system shall preserve selected image order in the draft, in the create-post payload, and in rendered post displays. | Discovery excluded reordering and ADR-004 preserves response order. | Discovery, ADR-004 notes | AC-011, AC-015 |
| FR-013A | Functional | Must | The composer shall allow the user to manually reorder selected images before post submission. | Reordering was promoted from a non-goal to a requested requirement. | User feedback | AC-015A |
| FR-014 | Functional | Must | The Flutter post data model and read-path parsing shall support post image metadata from AppView responses, including `cid`, `mime`, `size`, `alt`, `aspectRatio`, `thumb`, and `fullsize` when present. | The app needs the existing backend response contract for rendering. | Backend workflow, discovery | AC-003, AC-016 |
| FR-015 | Functional | Must | Feed cards for posts with images shall render an inline bounded-aspect carousel that shows one large image at a time and preserves the declared image aspect ratio within feed-safe bounds. | Follows the accepted feed display ADR. | ADR-004, discovery | AC-017 |
| FR-016 | Functional | Must | Multi-image feed cards shall expose horizontal paging plus a compact image count and page dots. | Makes multiple images visible and navigable per ADR. | ADR-004 | AC-017 |
| FR-016A | Functional | Must | Carousel count and page-dot indicators shall maintain sufficient contrast against the image content, either directly or by using a semi-transparent backing surface or equivalent treatment. | Ensures multi-image affordances remain visible across varied photography. | User feedback | AC-017A |
| FR-017 | Functional | Must | Tapping the image area in a feed card shall open a full-screen gallery, while tapping non-image areas of the same card shall preserve normal post/thread navigation and action behavior. | Separates media interaction from existing card navigation. | User prompt, ADR-004, discovery | AC-018 |
| FR-018 | Functional | Must | The full-screen gallery shall support swiping between images in post order. | Completes the requested gallery browsing behavior. | User prompt, discovery | AC-019 |
| FR-019 | Functional | Must | The user experience shall provide a hero-style transition between feed-card image entry and the full-screen gallery when feasible within the chosen navigation surface. | Matches the requested transition behavior while allowing implementation flexibility. | User prompt, discovery | AC-018 |
| FR-019A | Functional | Must | The full-screen gallery shall display the current image's alt text at the bottom of the screen. | Makes descriptive text visible in addition to accessibility semantics. | User feedback | AC-019A |
| FR-020 | Functional | Must | The system shall support pinch-to-zoom for post images both inline in the feed image area and in the full-screen gallery. | Reflects the confirmed in-scope zoom behavior. | User answer, discovery | AC-020 |
| FR-021 | Functional | Must | Inline image gestures shall not prevent normal vertical feed scrolling, horizontal carousel paging, or non-image post interactions from functioning as intended. | Gesture conflict handling is a key product risk that must be explicit. | Discovery risk, ADR-004 | AC-020, AC-021 |
| FR-022 | Functional | Must | Reply composition surfaces shall remain text-only and shall not show image selection or upload UI in this pass. | Preserves the confirmed scope boundary. | User answer, discovery | AC-005 |
| FR-023 | Functional | Must | Existing text-only post creation, text-only post rendering, and text-only replies shall remain backward compatible. | Prevents regressions in current core flows. | Discovery | AC-022 |
| NFR-001 | Non-functional | Must | The image-posting flow shall continue using existing authenticated AppView HTTP conventions, including shared session-backed API access and existing error handling patterns. | Keeps the new flow consistent with the app's current architecture. | Codebase, discovery | AC-001, AC-002, AC-010 |
| NFR-002 | Non-functional | Must | The image-posting and image-viewing experience shall expose image alt text through accessibility semantics for rendered images and gallery entry points. | Accessibility is already part of the backend/media contract and ADR notes. | ADR-004 notes, discovery | AC-012, AC-023 |
| NFR-003 | Non-functional | Should | The feed and gallery image rendering should use the existing feed image cache manager so repeated viewing avoids unnecessary network fetches. | The repository already provides a dedicated feed image cache surface. | Codebase, ADR-004 notes | AC-024 |
| NFR-004 | Non-functional | Should | The composer should preserve a stable layout during upload and image preview display so the user can continue composing text without disruptive reflow. | Helps usability in a media-rich composer flow. | Discovery | AC-008, AC-014 |
| RULE-001 | Business rule | Must | Image composition is allowed only for new top-level posts in this pass. | Locks the approved scope boundary. | User answer, discovery | AC-005 |
| RULE-002 | Business rule | Must | A draft may contain at most the configured maximum number of selected images, and that configuration must default to and remain aligned with the current backend-supported limit. | Matches the backend/media contract while allowing the limit to be defined centrally. | User feedback, discovery, backend workflow | AC-007 |
| RULE-003 | Business rule | Must | Every selected image must have non-empty alt text before the post can be submitted. | Accessibility and backend contract requirement. | User answer, backend workflow | AC-012, AC-014 |
| RULE-004 | Business rule | Must | Removing an uploaded image from the draft removes it from the eventual post payload but does not require client-side blob cleanup. | Matches the confirmed orphaned-blob acceptance. | User prompt, discovery | AC-011 |
| RULE-005 | Business rule | Must | The app must not claim or imply that uploaded post images are private media under the current atproto/PDS model. | Prevents misleading privacy expectations. | Backend workflow, discovery | AC-025 |
| RULE-006 | Business rule | Should | The app should not enforce a client-side maximum alt-text length unless a concrete product or backend limit is later defined. | No limit has been decided yet, so the app should avoid inventing one at the requirements stage. | User feedback | AC-027 |

## 9. Acceptance Criteria
| ID | Requirement IDs | Acceptance Criterion |
|---|---|---|
| AC-001 | BR-001, FR-001, FR-003, FR-012, NFR-001 | Given a signed-in user composing a new top-level post, when they add one or more supported images in one or more selection actions up to the 4-image cap, then each selected image begins uploading through the existing authenticated AppView flow without requiring any PDS token from the user. |
| AC-002 | BR-001, FR-010, NFR-001 | Given a draft with valid text and successfully uploaded images, when the user submits the post, then the Flutter app sends a create-post request containing top-level `images[]` entries built from uploaded blob metadata, alt text, and any available aspect ratio metadata. |
| AC-003 | BR-002, FR-014 | Given a fetched or newly created post that includes image metadata in the AppView response, when the Flutter app parses the response, then the post model preserves the returned image metadata needed for rendering. |
| AC-004 | BR-002, FR-018 | Given a post with multiple images, when the user opens the full-screen gallery, then they can browse the post's images in response order. |
| AC-005 | BR-003, FR-022, RULE-001 | Given the composer is opened for a reply, when the user views the reply composition surface, then no image selection or upload UI is shown and reply behavior remains text-only. |
| AC-006 | BR-004 | Given this change is implemented, when users interact with post media, then no video selection, upload, or playback behavior is introduced. |
| AC-007 | FR-001, FR-002, FR-012, RULE-002 | Given a user selects supported images within the configured image-count cap and configured size limit, when the app validates them, then valid images are accepted into the draft; and given an unsupported type, oversize file, or a selection that would exceed the configured cap, when validated, then the invalid image or excess addition is rejected with user-visible feedback and is not uploaded. |
| AC-008 | FR-003, FR-005, NFR-004 | Given a selected image upload is pending or in progress, when the composer renders that image, then the user sees a local preview and a per-image upload state indicator without losing the ability to continue editing text. |
| AC-009 | FR-004 | Given an image selected for upload, when the app prepares it for upload, then the app removes most EXIF and non-essential embedded metadata while preserving the visible image content and supported upload format. |
| AC-010 | FR-006, NFR-001 | Given an image upload fails, when the composer updates, then the failed image shows an error state and the post cannot be submitted until the failure is resolved by successful upload or image removal. |
| AC-011 | FR-007, FR-013, RULE-004 | Given a draft containing one or more selected images, when the user removes an image, then that image disappears from the draft and will not appear in the create-post payload, while the remaining images preserve their relative order. |
| AC-012 | FR-008, FR-008A, NFR-002, RULE-003 | Given a draft with one or more selected images, when any image is missing alt text, then submission remains disabled; and when image content is rendered or exposed as a gallery entry point, then its alt text is available through accessibility semantics. |
| AC-013 | FR-009, FR-010 | Given the app can determine selected image width and height locally, when it creates the post request, then the corresponding image entry includes aspect ratio metadata; and when it cannot determine dimensions, then the request can still proceed without aspect ratio metadata. |
| AC-014 | FR-011, NFR-004, RULE-003 | Given a top-level post draft, when the draft has neither valid text nor any selected images, an image is still uploading, an image is in unresolved failure, or an image is missing alt text, then the submit action is disabled; and when the draft contains valid text only, valid images only, or valid text plus valid images, then the submit action becomes enabled. |
| AC-015 | FR-013 | Given a user selects images in a particular order, when the draft is posted and later rendered, then the images appear in that same order. |
| AC-015A | FR-013A | Given a draft with multiple selected images, when the user reorders them in the composer, then the draft preview order updates and the final post preserves the new order. |
| AC-016 | FR-014 | Given an AppView post response that includes `cid`, `mime`, `size`, `alt`, `aspectRatio`, `thumb`, and `fullsize`, when parsed by the Flutter app, then those fields are made available to rendering surfaces when present. |
| AC-017 | FR-015, FR-016 | Given a post with one or more images, when rendered in the feed, then the post card shows an inline bounded-aspect carousel with one large image at a time; and if there are multiple images, then horizontal paging, a compact count, and page dots are shown. |
| AC-017A | FR-016A | Given a multi-image post rendered in the feed, when count and page-dot indicators are shown over varied image content, then they remain legible because they use sufficient contrast or a contrasting backing treatment. |
| AC-018 | FR-017, FR-019 | Given an image post in the feed, when the user taps the image area, then a full-screen gallery opens with a hero-style transition where feasible; and when the user taps outside the image area on the same card, then normal post/thread navigation and actions still work. |
| AC-019 | FR-018 | Given a multi-image full-screen gallery, when the user swipes horizontally, then the gallery moves between images in post order. |
| AC-019A | FR-019A | Given a full-screen gallery image is visible, when the screen renders, then the image's alt text is displayed at the bottom of the screen. |
| AC-020 | FR-020, FR-021 | Given an image is shown inline or in the full-screen gallery, when the user performs pinch-to-zoom gestures, then the image responds to zoom without breaking the surrounding surface's intended browsing behavior. |
| AC-021 | FR-021 | Given a feed containing image posts, when the user vertically scrolls the feed, horizontally pages media, taps image media, or taps non-image card regions, then those interactions remain distinct and function as intended. |
| AC-022 | FR-023 | Given an existing text-only post or text-only reply flow that works today, when this change is released, then those flows remain available and continue working except for additive image support on new top-level posts. |
| AC-023 | NFR-002 | Given assistive technology is used on feed or gallery media, when focus reaches an image or gallery entry control, then alt text is available in semantics for that image. |
| AC-024 | NFR-003 | Given a user revisits previously viewed feed or gallery media, when the image is requested again, then the app uses the feed image cache manager for eligible post-image loads. |
| AC-025 | RULE-005 | Given the image posting flow is presented in UI or associated copy, when the feature describes uploaded media behavior, then it does not imply that uploaded post images are private under the current PDS model. |
| AC-026 | BR-005, FR-011 | Given a top-level post draft, when it contains only valid text, only valid images, or both valid text and valid images, then the draft can be submitted; and when it contains neither text nor images, then it cannot be submitted. |
| AC-027 | RULE-006 | Given the current requirements stage has no defined product or backend alt-text length limit, when the app validates alt text, then it does not invent a separate client-side maximum length requirement. |

## 10. Edge Cases
| ID | Case | Expected Behavior | Requirement IDs |
|---|---|---|---|
| EC-001 | User selects more images than the configured cap across multiple add actions | Only images within the configured cap remain allowed; excess images are rejected with feedback and are not uploaded. | FR-001, FR-012, RULE-002 |
| EC-002 | One image upload fails while others succeed | The failed image shows error state; the post remains blocked until the failed image is removed or successfully uploaded. | FR-006, FR-011 |
| EC-003 | User deletes an image after it finished uploading | The image is removed from the draft and create payload without any client cleanup call. | FR-007, RULE-004 |
| EC-004 | App cannot determine aspect ratio for a selected image | The image can still be posted if all other required data is valid; aspect ratio is omitted. | FR-009, FR-010 |
| EC-005 | AppView response omits optional image fields such as `aspectRatio`, `thumb`, or `fullsize` | The post still renders with available metadata and fails soft on missing optional fields. | FR-014, FR-015 |
| EC-006 | Single-image post is rendered in feed | The image area renders without count/dots meant only for multi-image browsing. | FR-015, FR-016 |
| EC-007 | User opens reply composer from an image post | Reply composer remains text-only even though the referenced post contains images. | FR-022, RULE-001 |
| EC-008 | User composes a text-only top-level post after this change | The post can still be submitted without any image metadata or image UI interaction. | FR-023 |

## 11. Data / Persistence Impact
- New fields:
  - Flutter post model gains image metadata fields compatible with the AppView response contract.
  - Flutter create-post request shape expands to support top-level `images[]` entries.
  - Client-side draft state gains selected-image upload, alt text, and preview metadata.
  - Client-side draft state gains image-order metadata for manual reordering.
- Changed fields:
  - `POST /v1/posts` requests from Flutter are no longer text-only for top-level posts.
  - Feed-card rendering logic expands to consume post image metadata.
- Migration required:
  - None identified for persisted app or backend storage in this stage.
- Backwards compatibility:
  - Text-only top-level posts remain valid.
  - Reply flows remain text-only.
  - Posts without images continue to render without media UI.

## 12. UI / API / CLI Impact
- UI:
  - Top-level composer gains image selection, local preview, upload progress/error state, alt text entry, deletion, reordering, and submit gating.
  - Feed cards gain inline image carousel rendering.
  - Full-screen image gallery is added with visible alt-text display.
- API:
  - Consume existing `POST /v1/blobs/images` upload endpoint.
  - Consume existing image-capable `POST /v1/posts` contract.
  - Consume existing image-capable post read/list response contract.
- CLI:
  - None identified.
- Background jobs:
  - None identified.

## 13. Security / Privacy / Permissions
- Authentication:
  - Uses the existing signed-in session-backed AppView API access model.
- Authorization:
  - Image upload and post create operate only for the authenticated user.
- Sensitive data:
  - PDS tokens remain server-side; the app continues using the Craftsky session token only.
  - Selected local images are user content and should not be logged.
- Abuse cases:
  - Repeated oversize or unsupported file selections.
  - Repeated upload failures or retries.
  - Misunderstanding uploaded post images as private media.
  - Gesture confusion causing unintended post navigation or media interaction.

## 14. Observability
- Events:
  - None required.
- Logs:
  - None newly required at the requirements stage beyond existing client error handling patterns.
- Metrics:
  - None required.
- Alerts:
  - None required.

## 15. Risks
| ID | Risk | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Inline pinch-to-zoom conflicts with horizontal paging and vertical feed scroll | Could make feed media feel broken or unpredictable | Make gesture separation an explicit requirement and cover it in widget/integration tests |
| RISK-002 | Client-side validation differs from backend validation | Could create inconsistent acceptance/rejection behavior | Keep app validation aligned with backend MIME and size constraints and treat backend as final authority |
| RISK-003 | Missing or partial image metadata in read responses | Could produce unstable layouts or broken image surfaces | Require soft-fail rendering for missing optional fields and stable fallback layout behavior |
| RISK-004 | Media-selection dependency and permissions increase platform complexity | Could delay implementation or create platform-specific bugs | Keep requirements implementation-neutral but explicit about permission-sensitive media selection behavior |
| RISK-005 | Users may assume deleted draft images were also deleted from the PDS | Could create incorrect expectations about media lifecycle | Make orphaned-blob behavior explicit in requirements and user-facing wording |
| RISK-006 | Client-side EXIF stripping may interact differently across platforms or packages | Could create inconsistent metadata-removal behavior | Keep the requirement outcome-focused and verify representative platform behavior in test design |

## 16. Assumptions
| ID | Assumption | Impact If Wrong |
|---|---|---|
| ASM-001 | `01-discovery-notes.md` is the source of truth in place of a missing `00-initial-prompt.md` file in this workflow folder. | If wrong, requirements may not fully reflect an omitted initial prompt artifact. |
| ASM-002 | The existing AppView upload/create/read image contract is stable enough for Flutter integration without additional backend changes. | If wrong, requirements or API expectations may need revision. |
| ASM-003 | The app can obtain enough local file metadata to perform at least partial type/size validation before upload on supported platforms. | If wrong, more validation may need to rely on backend responses than currently intended. |
| ASM-004 | Hero transition behavior may vary slightly by final navigation implementation as long as the user still experiences a gallery-opening visual transition. | If wrong, requirements may need stricter transition wording. |
| ASM-005 | No backend or product-level alt-text length limit is currently required for this change. | If wrong, validation requirements and test cases will need to be updated. |

## 17. Review Status
Status: Draft
Risk level: High
Review recommended: Required
Reviewer:
Date:
Notes: High risk is carried forward from discovery due to authenticated media upload, public media implications, cross-surface UX changes, accessibility obligations, and complex inline gesture behavior.

## 18. Handoff To Test Design
- Requirements file: `02-requirements.md`
- Must-cover requirement IDs:
  - BR-001 through BR-004
  - FR-001 through FR-023
  - NFR-001 through NFR-002
  - RULE-001 through RULE-005
- Suggested test levels:
  - API client tests for upload and create-post payload shape
  - Provider/state tests for upload lifecycle and submit gating
  - Widget tests for composer image UI, feed carousel, and gallery entry behavior
  - Gesture-focused widget/integration tests for paging, tap routing, and pinch zoom
  - Widget/integration tests for reordering and fullscreen alt-text rendering
  - Regression tests for text-only top-level posts and replies
  - Accessibility/semantics tests for alt text exposure
- Blocking open questions:
  - None identified for test design; implementation details like exact package choice and tuned height bounds can be resolved without changing the core requirements.
